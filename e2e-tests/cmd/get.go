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
	"time"

	"github.com/spf13/cobra"

	"github.com/geoadmin/tool-golang-bgdi/lib/aws/codebuild"
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

		client, e := codebuild.NewClient(ctx, *cmd.Flags())
		if e != nil {
			return e
		}
		getOpt := codebuild.GetOptions{
			WaitForCompletion: true,
			WaitSleepInterval: time.Duration(flags.Interval) * time.Second,
			ProgressOutput:    os.Stdout,
			FetchReport:       true,
			TestCases:         []codebuild.TestStatus{codebuild.TestStatusFailed, codebuild.TestStatusError},
		}
		if !flags.ShowProgress {
			getOpt.ProgressOutput = nil
		}
		build, e := codebuild.ParseBuildID(flags.TestID)
		if e != nil {
			return fmt.Errorf("invalid test ID: %w", e)
		}
		r, e := client.GetBuildWithOptions(ctx, build, getOpt)
		if e != nil {
			return fmt.Errorf("failed to get tests run %s: %w", flags.TestID, e)
		}
		printTestResult(r, flags.Detailed)
		if !r.Succeeded() {
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
