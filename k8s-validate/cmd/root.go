package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/swissgeo/tool-golang-public/lib/version"
)

var FailFast = false
var Parallel = 0
var Lint = false
var NoColor = false

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
		folders, err := FindKustomizeFolders()
		if err != nil {
			return fmt.Errorf("error: failed to find kustomize folders: %w", err)
		}
		if len(folders) == 0 {
			fmt.Println("No kustomize folder found")
			return nil
		}
		var errs []string
		if !ValidateKustomize(folders, workers, FailFast) {
			errs = append(errs, "Error: kustomize validation failed")
		}
		if Lint && !LintKustomize(folders, workers, FailFast) {
			errs = append(errs, "Error: linter validation failed")
		}
		if len(errs) > 0 {
			return fmt.Errorf("%s", strings.Join(errs, "\n"))
		}
		return nil
	},
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
		fmt.Fprintln(os.Stderr, colorize(colorRed, err.Error()))
		os.Exit(1)
	}
}

func init() {
	// Suppress cobra's default error and usage printing so Execute() can
	// print a single, consistently formatted and colourised error line.
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().BoolVar(&FailFast, "fail-fast", false, "Fail on first error.")
	rootCmd.PersistentFlags().BoolVar(&Lint, "lint", false, "Also run kube-linter on all kustomize folders.")
	rootCmd.PersistentFlags().BoolVar(&NoColor, "no-color", false, "Disable colorized output.")
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
