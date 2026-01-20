package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/geoadmin/tool-golang-bgdi/lib/aws/codebuild"
	"github.com/geoadmin/tool-golang-bgdi/lib/fmtc"
	"github.com/geoadmin/tool-golang-bgdi/lib/version"
	"github.com/spf13/cobra"
)

const ErrBuildFailedCode = 2

//-----------------------------------------------------------------------------

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "codebuild",
	Short:         "BGDI CLI tool to control Codebuild projects",
	Long:          `This tool use the AWS SDK to control BGDI Codebuild projects.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

//-----------------------------------------------------------------------------

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(version.GetGitVersion())
	},
}

//-----------------------------------------------------------------------------

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		if errors.Is(err, ErrBuildFailed) {
			os.Exit(ErrBuildFailedCode)
		}
		fmt.Fprintln(os.Stderr, string(fmtc.Red)+err.Error()+string(fmtc.Reset))
		os.Exit(1)
	}
}

//-----------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(versionCmd)
	codebuild.DefineNewClientFlags(rootCmd.Flags())
}

//-----------------------------------------------------------------------------
