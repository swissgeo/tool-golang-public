/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/geoadmin/tool-golang-bgdi/lib/fmtc"
	"github.com/spf13/cobra"
)

//-----------------------------------------------------------------------------

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an E2E tests run status",
	Long: `Get an E2E tests run status.

Note that if the tests run is on-going, the command waits until its is finished.`,
	Run: func(cmd *cobra.Command, _ []string) {
		initPrint(cmd)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop() // Ensure cleanup

		client := getClient(ctx, cmd)

		id, e := cmd.Flags().GetString("test-id")
		if e != nil {
			log.Fatal(e)
		}

		input := &codebuild.BatchGetBuildsInput{
			Ids: []string{id},
		}
		r, e := client.BatchGetBuilds(ctx, input)
		if e != nil {
			log.Fatalf("failed to get tests run %s: %s", id, e)
		}

		if len(r.Builds) == 0 {
			log.Fatalf("failed to get tests run %s: not found", id)
		}
		if !r.Builds[0].BuildComplete {
			cPrintln(fmtc.NoColor, "E2E Tests run found, run in progress waiting to complete...")
			r = waitForBuild(ctx, client, id)
		} else {
			cPrintf(fmtc.NoColor, "E2E Tests run found and completed at %s\n", r.Builds[0].EndTime.UTC().String())
		}
		e = printTestReport(ctx, client, r.Builds[0].ReportArns[0], *r.Builds[0].Id)
		if e != nil {
			log.Fatalf("failed to print test reports: %s", e)
		}
	},
}

//-----------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().StringP("test-id", "t", "", "Test ID")
	_ = getCmd.MarkFlagRequired("test-id")
}

//-----------------------------------------------------------------------------
