package cmd

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/geoadmin/tool-golang-bgdi/lib/version"
	"github.com/spf13/cobra"
)

var FailFast = false
var Parallel = 0

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-validate",
	Short: "Validate all kubernetes manifests in subdirectories",
	Long:  `Run kustomization build in all subfolders containing a kustomization.yaml file`,
	RunE: func(_ *cobra.Command, _ []string) error {
		var workers int
		if Parallel == 0 {
			workers = runtime.NumCPU()
		} else {
			workers = Parallel
		}
		var valid = ValidateKustomize(workers, FailFast)
		if !valid {
			return errors.New("validation failed")
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(version.GetGitVersion())
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().BoolVar(&FailFast, "fail-fast", false, "Fail on first error.")
	rootCmd.PersistentFlags().IntVarP(
		&Parallel,
		"parallel",
		"j",
		0,
		`Run validation in parallel.
By default it is set to 0 which means that it use the number of available CPU
to determine how many parallel jobs are executed.`,
	)
}
