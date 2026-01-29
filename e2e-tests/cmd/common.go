package cmd

import (
	"fmt"

	"github.com/geoadmin/tool-golang-bgdi/lib/aws/codebuild"
	"github.com/spf13/cobra"
)

var noColor = false

func initPrint(cmd *cobra.Command) error {
	nc, e := cmd.Flags().GetBool("no-color")
	if e != nil {
		return e
	}
	noColor = nc
	return nil
}

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

func projectName(staging string) string {
	return fmt.Sprintf("e2e-tests-%s-pr", staging)
}
