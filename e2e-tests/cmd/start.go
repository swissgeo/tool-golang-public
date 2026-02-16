package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	codebuild_types "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/geoadmin/tool-golang-bgdi/e2e-tests/cmd/completions"
	"github.com/geoadmin/tool-golang-bgdi/lib/fmtc"
	"github.com/geoadmin/tool-golang-bgdi/lib/str"
	"github.com/spf13/cobra"

	"github.com/geoadmin/tool-golang-bgdi/e2e-tests/cmd/organization"
	"github.com/geoadmin/tool-golang-bgdi/lib/aws/codebuild"
)

//-----------------------------------------------------------------------------

type StartCmdFlags struct {
	Common     CommonCmdFlags
	Staging    string
	Tests      []string
	Markers    []string
	Revision   string
	DoDataTest bool
}

func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

//-----------------------------------------------------------------------------

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start E2E tests and wait for the result",
	Long:  `Start E2E tests on Codebuild and wait for the result.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		e := initPrint(cmd)
		if e != nil {
			return e
		}
		flags, e := getCmdStartFlags(cmd)
		if e != nil {
			return e
		}
		printStart(flags.Staging, flags)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop() // Ensure cleanup

		startOpt := codebuild.StartOptions{
			SourceVersion: flags.Revision,
			Timeout:       30 * time.Minute, //nolint:mnd
			Environment: []codebuild_types.EnvironmentVariable{
				{
					Name:  str.Ptr("IS_PULL_REQUEST"),
					Value: str.Ptr("0"),
					Type:  codebuild_types.EnvironmentVariableTypePlaintext,
				},
				{
					Name:  str.Ptr("DO_DATA_TEST"),
					Value: str.Ptr(boolToStr(flags.DoDataTest)),
					Type:  codebuild_types.EnvironmentVariableTypePlaintext,
				},
				{
					Name:  str.Ptr("TEST_NAMES"),
					Value: str.Ptr(strings.Join(flags.Tests, ",")),
					Type:  codebuild_types.EnvironmentVariableTypePlaintext,
				},
				{
					Name:  str.Ptr("TEST_MARKERS"),
					Value: str.Ptr(strings.Join(flags.Markers, ",")),
					Type:  codebuild_types.EnvironmentVariableTypePlaintext,
				},
			},
		}
		if flags.DoDataTest {
			startOpt.Timeout = 60 * time.Minute //nolint:mnd
		}

		client, e := codebuild.NewClient(ctx, *cmd.Flags())
		if e != nil {
			return e
		}

		build, e := client.StartBuildWithOptions(ctx, projectName(flags.Staging), startOpt)
		if e != nil {
			return e
		}
		fmt.Printf("  ID: %v\n  %s\n", build.ID, fmtc.Colorise(fmtc.Yellow, build.Link()))

		getOpt := codebuild.GetOptions{
			WaitForCompletion: true,
			WaitSleepInterval: time.Duration(flags.Common.Interval) * time.Second,
			ProgressOutput:    os.Stdout,
			FetchReport:       true,
			TestCases:         []codebuild.TestStatus{codebuild.TestStatusFailed, codebuild.TestStatusError},
		}
		if !flags.Common.ShowProgress {
			getOpt.ProgressOutput = nil
		}
		// Wait for the build to finish
		re, e := client.GetBuildWithOptions(ctx, build.ID, getOpt)
		if e != nil {
			return e
		}

		e = printTestResult(re, flags.Common.Detailed)
		if e != nil {
			return e
		}
		if !re.Succeeded() {
			return ErrTestFailed
		}
		return nil
	},
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		// Avoid doing file/folder completion after the command
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
}

//-----------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.
	startCmd.Flags().StringP("staging", "s", "dev", "Staging environment to use. Default is dev")
	startCmd.Flags().String("revision", "", "Revision of the tests to run. Default is master or main depending on the org")
	startCmd.Flags().Bool("data-tests", false, "Do also data integration tests (tests take much longer !)")
	startCmd.Flags().StringArrayP("tests", "t", []string{}, "Test to run. Default is all tests")
	startCmd.Flags().StringArrayP("markers", "m", []string{}, "Test to run selected by markers. Default is all tests")

	// Completions functions
	_ = startCmd.RegisterFlagCompletionFunc(
		"staging",
		cobra.FixedCompletions(
			[]cobra.Completion{
				cobra.Completion("dev"),
				cobra.Completion("int"),
				cobra.Completion("prod"),
			},
			cobra.ShellCompDirectiveNoFileComp,
		),
	)
	_ = startCmd.RegisterFlagCompletionFunc("revision", cobra.NoFileCompletions)
	_ = startCmd.RegisterFlagCompletionFunc("tests", completions.CompleteTests)
	_ = startCmd.RegisterFlagCompletionFunc(
		"markers",
		cobra.FixedCompletions(
			[]cobra.Completion{
				cobra.Completion("api"),
				cobra.Completion("auth"),
				cobra.Completion("data"),
				cobra.Completion("frontend"),
				cobra.Completion("mutating"),
				cobra.Completion("slow"),
			},
			cobra.ShellCompDirectiveNoFileComp,
		),
	)
	_ = startCmd.RegisterFlagCompletionFunc(
		"org",
		cobra.FixedCompletions(
			[]cobra.Completion{
				cobra.Completion(organization.GEOADMIN),
				cobra.Completion(organization.SWISSGEO),
			},
			cobra.ShellCompDirectiveNoFileComp,
		),
	)
}

//-----------------------------------------------------------------------------

func printStart(staging string, flags StartCmdFlags) {
	if len(flags.Markers) > 0 && flags.Common.Organization != organization.SWISSGEO {
		fmtc.Printf(fmtc.Yellow, "WARNING: flag --markers has no effect with --org %s\n", flags.Common.Organization)
	}
	fmt.Printf("Starting E2E tests on %s staging:\n", staging)
	if len(flags.Tests) > 0 {
		fmt.Printf("  tests: %s\n", strings.Join(flags.Tests, ", "))
	} else {
		fmt.Printf("  tests: all\n")
	}
	if len(flags.Markers) > 0 {
		fmt.Printf("  markers: %s\n", strings.Join(flags.Markers, ", "))
	}
}

// -----------------------------------------------------------------------------
// Get start command flags
func getCmdStartFlags(cmd *cobra.Command) (StartCmdFlags, error) {
	var flags StartCmdFlags
	var err error

	flags.Common, err = getCmdCommonFlags(cmd)
	if err != nil {
		return StartCmdFlags{}, err
	}

	flags.Staging = cmd.Flag("staging").Value.String()
	flags.Revision = cmd.Flag("revision").Value.String()
	if flags.Revision == "" {
		switch flags.Common.Organization {
		case organization.GEOADMIN:
			flags.Revision = "master"
		case organization.SWISSGEO:
			flags.Revision = "main"
		}
	}
	doDataTest, err := cmd.Flags().GetBool("data-tests")
	if err != nil {
		return StartCmdFlags{}, err
	}
	flags.DoDataTest = doDataTest

	tests, err := cmd.Flags().GetStringArray("tests")
	if err != nil {
		return StartCmdFlags{}, err
	}

	// Append the "tests" prefix to all tests
	tests = func() []string {
		out := make([]string, len(tests))
		for i, t := range tests {
			if flags.Common.Organization == organization.GEOADMIN {
				// Geoadmin uses python module path dot notation
				out[i] = "tests." + t
			} else {
				// swissgeo uses file path notation
				out[i] = filepath.Join("tests", t)
			}
		}
		return out
	}()
	flags.Tests = tests

	markers, err := cmd.Flags().GetStringArray("markers")
	if err != nil {
		return StartCmdFlags{}, err
	}
	flags.Markers = markers

	return flags, nil
}
