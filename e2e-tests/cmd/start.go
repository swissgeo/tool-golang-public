package cmd

import (
	"context"
	"log"
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

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start E2E tests and wait for the result",
	Long:  `Start E2E tests on Codebuild and wait for the result.`,
	Run: func(cmd *cobra.Command, _ []string) {
		initPrint(cmd)
		staging, tests, revision, doDataTest := getFlags(cmd)
		printStart(staging, tests)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop() // Ensure cleanup

		client := getClient(ctx, cmd)

		rs := startBuild(ctx, client, staging, tests, revision, doDataTest)

		// Wait for the build to finish
		re := waitForBuild(ctx, client, *rs.Build.Id)

		printTestResult(ctx, client, re)
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
	_ = startCmd.RegisterFlagCompletionFunc("tests", completions.FindTests)
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
func getFlags(cmd *cobra.Command) (string, []string, string, bool) {
	staging := cmd.Flag("staging").Value.String()
	revision := cmd.Flag("revision").Value.String()
	doDataTest, err := cmd.Flags().GetBool("data-tests")
	if err != nil {
		log.Fatal(err)
	}
	tests, err := cmd.Flags().GetStringArray("tests")
	if err != nil {
		log.Fatal(err)
	}
	return staging, tests, revision, doDataTest
}

//-----------------------------------------------------------------------------

func startBuild(
	ctx context.Context,
	client *codebuild.Client,
	staging string,
	tests []string,
	revision string,
	doDataTest bool,
) *codebuild.StartBuildOutput {
	d := "0"
	if doDataTest {
		d = "1"
	}
	input := &codebuild.StartBuildInput{
		ProjectName:   str.Ptr(projectName(staging)),
		SourceVersion: &revision,
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
				Value: str.Ptr(strings.Join(tests, ",")),
				Type:  types.EnvironmentVariableTypePlaintext,
			},
		},
	}

	result, err := client.StartBuild(ctx, input)
	if err != nil {
		log.Fatalf("failed to start build: %v", err)
	}
	cPrintf(fmtc.NoColor, "E2E tests started with ID: %s\n", *result.Build.Id)
	cPrintln(fmtc.Yellow, buildLogLink(*result.Build.Id))
	return result
}
