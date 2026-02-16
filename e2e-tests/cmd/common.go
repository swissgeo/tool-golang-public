package cmd

import (
	"fmt"

	"github.com/geoadmin/tool-golang-bgdi/e2e-tests/cmd/organization"
	"github.com/geoadmin/tool-golang-bgdi/lib/aws/codebuild"
	"github.com/spf13/cobra"
)

//-----------------------------------------------------------------------------

type CommonCmdFlags struct {
	Color        bool
	ShowProgress bool
	Interval     int
	Organization organization.Organization
	Detailed     bool
}

//-----------------------------------------------------------------------------

var noColor = false

//-----------------------------------------------------------------------------

func initPrint(cmd *cobra.Command) error {
	nc, e := cmd.Flags().GetBool("no-color")
	if e != nil {
		return e
	}
	noColor = nc
	return nil
}

//-----------------------------------------------------------------------------

func printTestResult(
	build codebuild.Build,
	detailed bool,
) error {
	reportsCount := build.ReportsCount()
	if reportsCount == 0 {
		return fmt.Errorf("no test report found for build: %v", build)
	}
	if reportsCount > 1 {
		fmt.Printf("%d test reports found, expected exactly 1", reportsCount)
	}
	fmt.Print(build.String(!noColor, detailed))
	return nil
}

//-----------------------------------------------------------------------------

func projectName(staging string) string {
	return fmt.Sprintf("e2e-tests-%s-pr", staging)
}

//-----------------------------------------------------------------------------

func getCmdCommonFlags(cmd *cobra.Command) (CommonCmdFlags, error) {
	var flags CommonCmdFlags

	flags.Organization = organization.Organization(cmd.Flag("org").Value.String())

	np, e := cmd.Flags().GetBool("no-progress")
	if e != nil {
		return CommonCmdFlags{}, e
	}
	showProgress := !np
	flags.ShowProgress = showProgress

	interval, e := cmd.Flags().GetInt("interval")
	if e != nil {
		return CommonCmdFlags{}, e
	}
	flags.Interval = interval

	detailed, e := cmd.Flags().GetBool("detailed")
	if e != nil {
		return CommonCmdFlags{}, e
	}
	flags.Detailed = detailed

	// Set the default AWS config profile when no --role and --profile options are given
	// the default profile is set based on --org option
	role, e := cmd.Flags().GetString("role")
	if e != nil {
		return CommonCmdFlags{}, e
	}
	profile, e := cmd.Flags().GetString("profile")
	if e != nil {
		return CommonCmdFlags{}, e
	}
	if role == "" && profile == "" {
		// Based on organization we need to set different AWS profile when no role is used
		switch flags.Organization {
		case organization.GEOADMIN:
			e = cmd.Flags().Set("profile", "swisstopo-bgdi-builder")
			if e != nil {
				return CommonCmdFlags{}, e
			}
		case organization.SWISSGEO:
			e = cmd.Flags().Set("profile", "swisstopo-swissgeo-builder")
			if e != nil {
				return CommonCmdFlags{}, e
			}
		}
	}

	return flags, nil
}
