package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/geoadmin/tool-golang-bgdi/lib/aws/codebuild"
	"github.com/geoadmin/tool-golang-bgdi/lib/log"
)

func init() {
	rootCmd.AddCommand(getCmd)
	codebuild.DefineGetBuildFlags(getCmd.Flags())
	getCmd.Flags().BoolP("full-test-report", "r", false, "Show full test report for faulty tests.")
}

var getCmd = &cobra.Command{
	Use:   "get BUILD_ARN",
	Short: "Get informations about specified build.",
	RunE: func(cmd *cobra.Command, builds []string) error {
		if len(builds) != 1 {
			return fmt.Errorf(`expecting exactly one "build" argument, got %d`, len(builds))
		}
		buildID, e := codebuild.ParseBuildID(builds[0])
		if e != nil {
			return fmt.Errorf(`invalid build argument: %s: %w`, builds[0], e)
		}
		getOpt, e := codebuild.ParseGetFlags(*cmd.Flags())
		if e != nil {
			return fmt.Errorf("failed to parse GetBuild flags: %w", e)
		}
		showTestReport, e := cmd.Flags().GetBool("full-test-report")
		if e != nil {
			return fmt.Errorf(`failed to parse flag value "full-test-report": %w`, e)
		}
		if showTestReport {
			getOpt.FetchReport = true
			getOpt.TestCases = []codebuild.TestStatus{
				codebuild.TestStatusFailed,
				codebuild.TestStatusError,
			}
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		client, err := codebuild.NewClient(ctx, *cmd.Flags())
		if err != nil {
			return err
		}
		log.Debug("Initialised Codebuild client: %v", client)
		log.Debug("Getting build %v with options %+v", buildID, getOpt)
		build, err := client.GetBuildWithOptions(ctx, buildID, getOpt)
		if err != nil {
			return err
		}
		log.Debug("Got build %v", build)
		if showTestReport {
			fmt.Print(build.String(false, false))
		} else {
			fmt.Printf("%s\n", build.ShortString(false))
		}
		return nil
	},
}
