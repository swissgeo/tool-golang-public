package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	local_aws "github.com/geoadmin/tool-golang-bgdi/lib/aws"
	"github.com/geoadmin/tool-golang-bgdi/lib/completioncache"
	"github.com/spf13/cobra"
)

func profileCompletion(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	profiles, err := local_aws.GetLocalBgdiAdminProfiles()
	if err != nil {
		return []cobra.Completion{}, cobra.ShellCompDirectiveError
	}
	return profiles, cobra.ShellCompDirectiveDefault
}

func nameCompletionCached() cobra.CompletionFunc {
	cache, _ := completioncache.NewCache("aws-ssm", time.Hour*24) //nolint:mnd
	return func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		profile, _ := cmd.Flags().GetString("profile")
		c, d, err := cache.Read(fmt.Sprintf("get-name-%s", profile))
		if err == nil {
			return c, d
		}
		c, d, noCache := nameCompletion(cmd, args, toComplete)
		if !noCache {
			_ = cache.Write(fmt.Sprintf("get-name-%s", profile), c, d)
		}
		return c, d
	}
}

func nameCompletion(cmd *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective, bool) {
	profile, _ := cmd.Flags().GetString("profile")
	if len(profile) == 0 {
		return cobra.AppendActiveHelp(
			[]cobra.Completion{},
			"ERROR: no profile provided, please enter first --profile option.",
		), cobra.ShellCompDirectiveNoFileComp, true
	}
	cmdArgs := []string{
		"--no-cli-pager",
		"ssm", "describe-parameters",
		"--profile", profile,
		"--query", "Parameters[*].Name",
		"--output", "text",
	}
	awsCmd := exec.Command("aws", cmdArgs...)
	res, err := awsCmd.Output()
	if err != nil {
		return cobra.AppendActiveHelp(nil, fmt.Sprintf("ERROR: %s", err.Error())), cobra.ShellCompDirectiveNoFileComp, true
	}
	return strings.Fields(string(res)), cobra.ShellCompDirectiveDefault, false
}
