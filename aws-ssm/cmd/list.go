package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	local_aws "github.com/geoadmin/tool-golang-bgdi/lib/aws"
	"github.com/geoadmin/tool-golang-bgdi/lib/fmtc"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type listCmdOptions struct {
	profile        string
	withDecryption bool
	shared         bool
	search         string
	verbose        bool
}

func newListCmd() *cobra.Command {
	options := listCmdOptions{}

	listCmd := &cobra.Command{
		Use:   "list [--profile PROFILE] [-s|--search <search-string>] [-d|--with-decryption]",
		Short: "List available SSM parameters",
		Long:  `List SSM parameters from all AWS accounts.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			options.verbose, err = cmd.Flags().GetBool("verbose")
			if err != nil {
				return err
			}

			return listSSMParameter(options)
		},
	}
	listCmd.Flags().StringVarP(&options.profile, "profile", "p", "", "Profile/Account name")
	listCmd.Flags().BoolVarP(&options.withDecryption, "with-decryption", "d", false, "Dump the credential values")
	listCmd.Flags().BoolVarP(&options.shared, "shared", "a", false, "Show the shared parameters")
	listCmd.Flags().StringVarP(&options.search, "search", "s", "", "Search for SEARCH in the parameter names")
	_ = listCmd.RegisterFlagCompletionFunc("profile", profileCompletion)
	return listCmd
}

func listSSMParameter(opts listCmdOptions) error {
	// Get profiles:
	profiles, err := local_aws.GetLocalBgdiAdminProfiles()
	if err != nil {
		return err
	}
	if opts.profile != "" {
		profiles = []string{opts.profile}
	}

	results := []result{}
	errG := new(errgroup.Group)
	for _, p := range profiles {
		errG.Go(func() error {
			res, e := listSSMParametersForProfile(p, opts.search, opts.shared, opts.verbose)
			if e != nil {
				return e
			}
			results = append(results, res...)
			return nil
		})
	}
	if err = errG.Wait(); err != nil {
		return err
	}
	if opts.withDecryption {
		for i, r := range results {
			errG.Go(func() error {
				res, e := getSSMParameter(getCmdOptions{
					profile:   r.profile,
					name:      r.key,
					valueOnly: true,
					verbose:   opts.verbose,
				})
				if e != nil {
					return e
				}
				results[i].value = strings.TrimSpace(res)
				return nil
			})
		}
		if err = errG.Wait(); err != nil {
			return err
		}
	}
	printResults(results)
	return nil
}

func listSSMParametersForProfile(profile, search string, shared, verbose bool) ([]result, error) {
	cmdArgs := []string{"ssm", "describe-parameters", "--query", "Parameters[*].Name", "--output", "text"}
	if search != "" {
		cmdArgs = append(cmdArgs, "--parameter-filters", "Key=Name,Option=Contains,Values="+search)
	}
	if shared {
		cmdArgs = append(cmdArgs, "--shared")
	}
	cmdArgs = append(cmdArgs, "--profile", profile)

	if verbose {
		fmt.Println("aws", strings.Join(cmdArgs, " "))
	}

	awsCmd := exec.Command("aws", cmdArgs...)
	awsCmd.Stderr = os.Stderr
	res, err := awsCmd.Output()
	if err != nil {
		return nil, err
	}
	var results []result
	for _, r := range strings.Fields(string(res)) {
		results = append(results, result{key: r, profile: profile, color: getProfileColor(profile)})
	}
	return results, nil
}

type result struct {
	key     string
	profile string
	value   string
	color   fmtc.Color
}

func printResults(res []result) {
	sort.Slice(res, func(i, j int) bool {
		return res[i].key < res[j].key
	})
	for _, r := range res {
		fmt.Printf("%s %s (%s)\n", r.key, r.value, fmtc.Sprintf(r.color, "%s", r.profile))
	}
}

var profileColors = make(map[string]fmtc.Color)

// getProfileColor returns a color for the given profile. Caches the profile-color relations
// to always return the same color for the same profile. If there are more profiles than colors,
// black will be used for any further profiles.
func getProfileColor(p string) fmtc.Color {
	var availableColors = []fmtc.Color{
		"\033[31m", // Red
		"\033[32m", // Green
		"\033[33m", // Yellow
		"\033[34m", // Blue
		"\033[35m", // Magenta
		"\033[36m", // Cyan
		"\033[91m", // Bright Red
		"\033[92m", // Bright Green
		"\033[93m", // Bright Yellow
		"\033[94m", // Bright Blue
		"\033[95m", // Bright Magenta
		"\033[96m", // Bright Cyan
		"\033[30m", // Black
	}
	if c, found := profileColors[p]; found {
		return c
	}
	profileColors[p] = availableColors[min(len(profileColors), len(availableColors)-1)]
	return profileColors[p]
}

func init() {
	rootCmd.AddCommand(newListCmd())
}
