package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

type getCmdOptions struct {
	profile   string
	name      string
	valueOnly bool
	verbose   bool
}

func newGetCmd() *cobra.Command {
	options := getCmdOptions{}

	getCmd := &cobra.Command{
		Use:   "get --profile PROFILE --name NAME [-o|--value-only]",
		Short: "Get the SSM parameter NAME from the account PROFILE",
		Long:  `Get the SSM parameter NAME from the account PROFILE`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			options.verbose, err = cmd.Flags().GetBool("verbose")
			if err != nil {
				return err
			}

			return runGetCmd(options)
		},
	}
	getCmd.Flags().StringVarP(&options.profile, "profile", "p", "", "MANDATORY: Profile/Account name")
	getCmd.Flags().StringVarP(&options.name, "name", "n", "", "MANDATORY: Name of the ssm parameter")
	getCmd.Flags().BoolVarP(&options.valueOnly, "value-only", "o", false, "If you only want to see the value")
	_ = getCmd.MarkFlagRequired("profile")
	_ = getCmd.MarkFlagRequired("name")
	return getCmd
}

func runGetCmd(opts getCmdOptions) error {
	res, err := getSSMParameter(opts)
	if err != nil {
		return err
	}
	fmt.Println(res)
	return nil
}

func getSSMParameter(opts getCmdOptions) (string, error) {
	cmdArgs := []string{"ssm", "get-parameter", "--profile", opts.profile, "--name", opts.name, "--with-decryption"}
	if opts.valueOnly {
		cmdArgs = append(cmdArgs, "--query", "Parameter.Value", "--output", "text")
	}

	if opts.verbose {
		fmt.Println("aws", strings.Join(cmdArgs, " "))
	}

	awsCmd := exec.Command("aws", cmdArgs...)
	awsCmd.Stderr = os.Stderr
	res, err := awsCmd.Output()
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func init() {
	rootCmd.AddCommand(newGetCmd())
}
