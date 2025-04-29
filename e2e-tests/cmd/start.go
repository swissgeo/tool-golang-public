package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/geoadmin/tool-golang-bgdi/e2e-tests/cmd/completions"
	"github.com/geoadmin/tool-golang-bgdi/lib/fmtc"
	"github.com/geoadmin/tool-golang-bgdi/lib/str"
	"github.com/spf13/cobra"
)

//-----------------------------------------------------------------------------

type StartCmdFlags struct {
	Staging      string
	Tests        []string
	Revision     string
	DoDataTest   bool
	ShowProgress bool
	Interval     int
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
		printStart(flags.Staging, flags.Tests)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop() // Ensure cleanup

		client, e := getClient(ctx, cmd)
		if e != nil {
			return e
		}

		rs, e := startBuild(ctx, client, flags)
		if e != nil {
			return e
		}

		// Wait for the build to finish
		re, e := waitForBuild(ctx, client, *rs.Build.Id, flags.ShowProgress, flags.Interval)
		if e != nil {
			return e
		}

		return printTestResult(ctx, client, re, false)
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
	startCmd.Flags().String("revision", "master", "Revision of the tests to run. Default is master")
	startCmd.Flags().Bool("data-tests", false, "Do also data integration tests (tests take much longer !)")
	startCmd.Flags().StringArrayP("tests", "t", []string{}, "Test to run. Default is all tests")

	// Completions functions
	_ = startCmd.RegisterFlagCompletionFunc(
		"staging",
		func(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective,
		) {
			return []cobra.Completion{"dev", "int", "prod"}, cobra.ShellCompDirectiveDefault
		})
	_ = startCmd.RegisterFlagCompletionFunc("tests", completions.CompleteTests)
}

//-----------------------------------------------------------------------------

func printStart(staging string, tests []string) {
	cPrintf(fmtc.NoColor, "Starting E2E tests on %s staging", staging)
	if len(tests) > 0 {
		cPrintf(fmtc.NoColor, " with tests: %s\n", strings.Join(tests, ", "))
	} else {
		cPrintf(fmtc.NoColor, " with all tests\n")
	}
}

// -----------------------------------------------------------------------------
// Get start command flags
func getCmdStartFlags(cmd *cobra.Command) (StartCmdFlags, error) {
	var flags StartCmdFlags
	flags.Staging = cmd.Flag("staging").Value.String()
	flags.Revision = cmd.Flag("revision").Value.String()
	doDataTest, err := cmd.Flags().GetBool("data-tests")
	if err != nil {
		return StartCmdFlags{}, err
	}
	flags.DoDataTest = doDataTest

	tests, err := cmd.Flags().GetStringArray("tests")
	if err != nil {
		return StartCmdFlags{}, err
	}

	// Append the "tests." prefix to all tests
	tests = func() []string {
		out := make([]string, len(tests))
		for i, t := range tests {
			out[i] = "tests." + t
		}
		return out
	}()
	flags.Tests = tests

	np, err := cmd.Flags().GetBool("no-progress")
	if err != nil {
		return StartCmdFlags{}, err
	}
	showProgress := !np
	flags.ShowProgress = showProgress

	interval, err := cmd.Flags().GetInt("interval")
	if err != nil {
		return StartCmdFlags{}, err
	}
	flags.Interval = interval

	return flags, nil
}

//-----------------------------------------------------------------------------

func startBuild(
	ctx context.Context,
	client *codebuild.Client,
	flags StartCmdFlags,
) (*codebuild.StartBuildOutput, error) {
	d := "0"
	if flags.DoDataTest {
		d = "1"
	}
	input := &codebuild.StartBuildInput{
		ProjectName:   str.Ptr(projectName(flags.Staging)),
		SourceVersion: &flags.Revision,
		EnvironmentVariablesOverride: []types.EnvironmentVariable{
			{
				Name:  str.Ptr("IS_PULL_REQUEST"),
				Value: str.Ptr("0"),
				Type:  types.EnvironmentVariableTypePlaintext,
			},
			{
				Name:  str.Ptr("DO_DATA_TEST"),
				Value: &d,
				Type:  types.EnvironmentVariableTypePlaintext,
			},
			{
				Name:  str.Ptr("TEST_NAMES"),
				Value: str.Ptr(strings.Join(flags.Tests, ",")),
				Type:  types.EnvironmentVariableTypePlaintext,
			},
		},
	}

	result, err := client.StartBuild(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to start build: %w", err)
	}
	cPrintf(fmtc.NoColor, "E2E tests started with ID: %s\n", *result.Build.Id)
	cPrintln(fmtc.Yellow, buildLogLink(*result.Build.Id))
	return result, nil
}
