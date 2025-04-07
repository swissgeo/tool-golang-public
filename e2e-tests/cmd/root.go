package cmd

import (
	"fmt"
	"os"

	"github.com/geoadmin/tool-golang-bgdi/lib/version"
	"github.com/spf13/cobra"
)

//-----------------------------------------------------------------------------

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "e2e-tests",
	Short: "BGDI CLI tool to control E2E tests",
	Long: `This tool use the AWS SDK to control Codebuild to start E2E tests on Codebuild
and get the final reports`,
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

//-----------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().Bool("no-color", false, "Do not use color in output")
	rootCmd.PersistentFlags().Bool("no-profile", false, "Do not use AWS profile swisstopo-bgdi-builder for credentials")
	rootCmd.PersistentFlags().String("role", "", "Role to assume for AWS permissions")
	rootCmd.PersistentFlags().Bool("no-progress", false, "For long running command don't display progress indicator")
	rootCmd.PersistentFlags().Int("interval", 1, "Interval in seconds to check the E2E tests status")
}

//-----------------------------------------------------------------------------
