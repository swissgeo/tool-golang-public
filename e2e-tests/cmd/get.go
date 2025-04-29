/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/geoadmin/tool-golang-bgdi/lib/fmtc"
	"github.com/spf13/cobra"
)

//-----------------------------------------------------------------------------

type GetCmdFlags struct {
	TestID       string
	Detailed     bool
	ShowProgress bool
	Interval     int
}

//-----------------------------------------------------------------------------

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an E2E tests run status",
	Long: `Get an E2E tests run status.

Note that if the tests run is on-going, the command waits until its is finished.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		e := initPrint(cmd)
		if e != nil {
			return e
		}

		flags, e := getCmdGetFlags(cmd)
		if e != nil {
			return e
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop() // Ensure cleanup

		client, e := getClient(ctx, cmd)
		if e != nil {
			return e
		}

		input := &codebuild.BatchGetBuildsInput{
			Ids: []string{flags.TestID},
		}
		r, e := client.BatchGetBuilds(ctx, input)
		if e != nil {
			return fmt.Errorf("failed to get tests run %s: %w", flags.TestID, e)
		}

		if len(r.Builds) == 0 {
			return fmt.Errorf("failed to get tests run %s: not found", flags.TestID)
		}
		if !r.Builds[0].BuildComplete {
			cPrintln(fmtc.NoColor, "E2E Tests run found, run in progress waiting to complete...")
			r, e = waitForBuild(ctx, client, flags.TestID, flags.ShowProgress, flags.Interval)
			if e != nil {
				return e
			}
		} else {
			cPrintf(fmtc.NoColor, "E2E Tests run found and completed at %s\n", r.Builds[0].EndTime.UTC().String())
		}

		return printTestResult(ctx, client, r, flags.Detailed)
	},
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		// Avoid doing file/folder completion after the command
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
}

//-----------------------------------------------------------------------------

func getCmdGetFlags(cmd *cobra.Command) (GetCmdFlags, error) {
	var flags GetCmdFlags
	id, e := cmd.Flags().GetString("test-id")
	if e != nil {
		return GetCmdFlags{}, e
	}
	flags.TestID = id

	np, e := cmd.Flags().GetBool("no-progress")
	if e != nil {
		return GetCmdFlags{}, e
	}
	flags.ShowProgress = !np

	interval, e := cmd.Flags().GetInt("interval")
	if e != nil {
		return GetCmdFlags{}, e
	}
	flags.Interval = interval

	detailed, e := cmd.Flags().GetBool("detailed")
	if e != nil {
		return GetCmdFlags{}, e
	}
	flags.Detailed = detailed

	return flags, nil
}

//-----------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().StringP("test-id", "t", "", "Test ID")
	getCmd.Flags().BoolP("detailed", "d", false, "Show detailed test result")
	_ = getCmd.MarkFlagRequired("test-id")
}

//-----------------------------------------------------------------------------
