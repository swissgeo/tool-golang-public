package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var FailFast = false
var Parallel = 0

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-validate",
	Short: "Validate all kubernetes manifests in subdirectories",
	Long:  `Run kustomization build in all subfolders containing a kustomization.yaml file`,
	Run: func(cmd *cobra.Command, args []string) {
		var workers int
		if Parallel == 0 {
			workers = runtime.NumCPU()
		} else {
			workers = Parallel
		}
		var valid = ValidateKustomize(workers, FailFast)
		if !valid {
			os.Exit(1)
		}
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

	// Add completion command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				if err := rootCmd.GenBashCompletion(os.Stdout); err != nil {
					cmd.PrintErrln("Error generating Bash completion:", err)
				}
			case "zsh":
				if err := rootCmd.GenZshCompletion(os.Stdout); err != nil {
					cmd.PrintErrln("Error generating Zsh completion:", err)
				}
			case "fish":
				if err := rootCmd.GenFishCompletion(os.Stdout, true); err != nil {
					cmd.PrintErrln("Error generating Fish completion:", err)
				}
			case "powershell":
				if err := rootCmd.GenPowerShellCompletionWithDesc(os.Stdout); err != nil {
					cmd.PrintErrln("Error generating PowerShell completion:", err)
				}
			default:
				cmd.PrintErrln("Unsupported shell type")
			}
		},
	})
}
