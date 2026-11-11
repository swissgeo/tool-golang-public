package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/swissgeo/tool-golang-public/lib/version"
)

const awsCLISSMSubCmd = "ssm"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "aws-ssm",
	Short: "AWS SSM Parameter helper",
	Long: `Easily search and get AWS SSM Parameter values across all AWS accounts.
A 24h cache is used for parameter name completion under $Home/.cache/aws-ssm/`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(version.GetVersion())
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().BoolP(
		"verbose", "v", false,
		"Print all aws command that are run",
	)
}
