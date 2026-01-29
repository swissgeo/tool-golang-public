package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	codebuild_types "github.com/aws/aws-sdk-go-v2/service/codebuild/types"

	"github.com/geoadmin/tool-golang-bgdi/lib/aws/codebuild"
	"github.com/geoadmin/tool-golang-bgdi/lib/log"
)

var ErrBuildFailed = errors.New("build failed")

func init() {
	rootCmd.AddCommand(startCmd)
	codebuild.DefineStartBuildFlags(startCmd.Flags())
	codebuild.DefineGetBuildFlags(startCmd.Flags())
}

var startCmd = &cobra.Command{
	Use:   "start PROJECT",
	Short: "Start codebuild project",
	RunE: func(cmd *cobra.Command, projects []string) error {
		if len(projects) != 1 || projects[0] == "" {
			return errors.New(`expecting exactly one "project" argument`)
		}
		getOpts, e := codebuild.ParseGetFlags(*cmd.Flags())
		if e != nil {
			return e
		}
		log.Debug("Parsed get options: %+v", getOpts)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop() // Ensure cleanup

		client, e := codebuild.NewClient(ctx, *cmd.Flags())
		if e != nil {
			return e
		}
		log.Debug("Initialised Codebuild client: %+v", client)

		build, e := client.StartBuildWithFlags(ctx, projects[0], *cmd.Flags())
		if e != nil {
			return e
		}
		log.Debug("Started build: %+v", build.Arn)
		fmt.Println(build.Link())

		if getOpts.WaitForCompletion {
			build, e = client.GetBuildWithOptions(ctx, build.ID, getOpts)
			if e != nil {
				return e
			}
			log.Debug("Build completed with status %v", build.Status())
			if build.Status() != codebuild_types.StatusTypeSucceeded {
				fmt.Printf("Build did not complete successfully: status=%v\n", build.Status())
				return ErrBuildFailed
			}
		} else {
			log.Debug("Not waiting for build completion.")
		}
		return nil
	},
}
