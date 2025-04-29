package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/geoadmin/tool-golang-bgdi/lib/fmtc"
	"github.com/geoadmin/tool-golang-bgdi/lib/str"
	"github.com/spf13/cobra"
)

//----------------------------------------------------------------------------

var cPrintln func(c fmtc.Color, v ...any) = fmtc.Println
var cPrintf func(c fmtc.Color, format string, v ...any) = fmtc.Printf

//-----------------------------------------------------------------------------

func initPrint(cmd *cobra.Command) error {
	noColor, e := cmd.Flags().GetBool("no-color")
	if e != nil {
		return e
	}

	if noColor {
		cPrintln = func(_ fmtc.Color, v ...any) {
			log.Println(v...)
		}
		cPrintf = func(_ fmtc.Color, f string, v ...any) {
			log.Printf(f, v...)
		}
	} else {
		log.SetFlags(0)
	}
	return nil
}

//-----------------------------------------------------------------------------

// Returns a CodeBuild client
func getClient(ctx context.Context, cmd *cobra.Command) (*codebuild.Client, error) {
	noProfile, e := cmd.Flags().GetBool("no-profile")
	if e != nil {
		return nil, e
	}
	role := cmd.Flag("role").Value.String()
	var cfg aws.Config

	switch {
	case role != "":
		var cred *sts.AssumeRoleOutput
		cfg, e = config.LoadDefaultConfig(ctx, config.WithRegion("eu-central-1"))
		if e != nil {
			return nil, fmt.Errorf("failed to load configuration: %w", e)
		}
		splittedRole := strings.Split(role, "/")
		roleName := splittedRole[len(splittedRole)-1]
		stsClient := sts.NewFromConfig(cfg)
		cred, e = stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         &role,
			RoleSessionName: str.Ptr(fmt.Sprintf("ToolE2ETestsAssumeRole%s", roleName)),
			DurationSeconds: aws.Int32(45 * 60), //nolint:mnd
		})
		if e != nil {
			return nil, fmt.Errorf("failed to assume role %s: %w", role, e)
		}

		cfg, e = config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				*cred.Credentials.AccessKeyId,
				*cred.Credentials.SecretAccessKey,
				*cred.Credentials.SessionToken,
			),
		))
		if e != nil {
			return nil, fmt.Errorf("failed to load configuration with role credentials: %w", e)
		}
	case noProfile:
		cfg, e = config.LoadDefaultConfig(ctx, config.WithRegion("eu-central-1"))
		if e != nil {
			return nil, fmt.Errorf("failed to load configuration: %w", e)
		}
	default:
		cfg, e = config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile("swisstopo-bgdi-builder"))
		if e != nil {
			return nil, fmt.Errorf("failed to load configuration: %w", e)
		}
	}

	client := codebuild.NewFromConfig(cfg)

	return client, nil
}

//-----------------------------------------------------------------------------

func waitForBuild(
	ctx context.Context,
	client *codebuild.Client,
	buildID string,
	showProgress bool,
	interval int,
) (*codebuild.BatchGetBuildsOutput, error) {
	var result *codebuild.BatchGetBuildsOutput
	var e error

	input := &codebuild.BatchGetBuildsInput{
		Ids: []string{buildID},
	}
	c := 0
	for {
		if showProgress {
			cPrintf(fmtc.NoColor, "Waiting for result: %ds\r", c)
			c += interval
		}
		time.Sleep(time.Duration(interval) * time.Second)
		result, e = client.BatchGetBuilds(ctx, input)
		if e != nil {
			return nil, fmt.Errorf("failed to get build status: %w", e)
		}
		if len(result.Builds) == 0 {
			return nil, fmt.Errorf("no build found with id: %s", buildID)
		}
		if result.Builds[0].BuildComplete {
			fmt.Printf("E2E tests finished with status: %s\n", result.Builds[0].BuildStatus)
			break
		}
	}

	return result, nil
}

//-----------------------------------------------------------------------------

func printTestResult(
	ctx context.Context,
	client *codebuild.Client,
	result *codebuild.BatchGetBuildsOutput,
	detailed bool,
) error {
	if result.Builds[0].BuildStatus == types.StatusTypeSucceeded {
		cPrintln(fmtc.Green, "E2E tests succeeded")
		return nil
	}
	// If the build failed, print the reports
	cPrintln(fmtc.Red, "E2E tests failed !")
	for _, report := range result.Builds[0].ReportArns {
		e := printTestReport(ctx, client, report, *result.Builds[0].Id, detailed)
		if e != nil {
			return e
		}
	}
	// For E2E tests error we use exit code 2 to differentiate between e2e-tests command failure
	return ErrTestFailed
}

//-----------------------------------------------------------------------------

func printTestReport(
	ctx context.Context,
	client *codebuild.Client,
	reportArn string,
	buildID string,
	detailed bool,
) error {
	// Get the number of tests
	input := &codebuild.BatchGetReportsInput{
		ReportArns: []string{reportArn},
	}
	r, e := client.BatchGetReports(ctx, input)
	if e != nil {
		return fmt.Errorf("failed to describe test case for reportARN=%s: %w", reportArn, e)
	}
	nbTests := int(*r.Reports[0].TestSummary.Total)

	// First get the errors
	nbErr, e := printTestReportByStatus(ctx, client, reportArn, "ERROR", detailed)
	if e != nil {
		return e
	}

	// Then gets the failure
	nbFails, e := printTestReportByStatus(ctx, client, reportArn, "FAILED", detailed)
	if e != nil {
		return e
	}

	cPrintf(fmtc.Red, "\nTests failures/errors %d%% (%d/%d)\n", (nbErr+nbFails)*100/nbTests, nbErr+nbFails, nbTests)

	cPrintln(fmtc.Red, "Test report link:")
	cPrintln(fmtc.Red, "-----------------")
	cPrintln(fmtc.Red, buildReportLink(projectNameFromBuildID(buildID), reportArn))

	return nil
}

//-----------------------------------------------------------------------------

func printTestReportByStatus(
	ctx context.Context,
	client *codebuild.Client,
	reportArn string,
	status string,
	detailed bool,
) (int, error) {
	filter := &types.TestCaseFilter{
		Status: &status,
	}
	input := &codebuild.DescribeTestCasesInput{
		ReportArn: &reportArn,
		Filter:    filter,
	}
	r, e := client.DescribeTestCases(ctx, input)
	if e != nil {
		return 0, fmt.Errorf("failed to describe test case %s for reportARN=%s: %w", status, reportArn, e)
	}
	printTestCases(r.TestCases, status, detailed)

	return len(r.TestCases), nil
}

//-----------------------------------------------------------------------------

func printTestCases(tests []types.TestCase, statusType string, detailed bool) {
	if len(tests) > 0 {
		cPrintf(fmtc.Red, "\n%-3d tests %s:\n", len(tests), statusType)
		cPrintf(fmtc.Red, "----------%s-\n", strings.Repeat("-", len(statusType)))
		for _, t := range tests {
			const nanoSeconds int64 = 1_000_000_000
			cPrintf(fmtc.Red, "- %s.%s (%d s)\n", *t.Prefix, *t.Name, *t.DurationInNanoSeconds/nanoSeconds)
			if detailed {
				cPrintf(fmtc.Red, "    %s\n\n", strings.ReplaceAll(*t.Message, "\n", "\n    "))
			}
		}
	}
}

//-----------------------------------------------------------------------------

func projectName(staging string) string {
	return fmt.Sprintf("e2e-tests-%s-pr", staging)
}

//-----------------------------------------------------------------------------

func buildReportLink(project string, reportArn string) string {
	if len(reportArn) == 0 {
		return ""
	}

	splitArn := strings.Split(reportArn, ":")
	reportIDParts := strings.Split(strings.Join(splitArn[len(splitArn)-2:], "%3A"), "/")
	if len(reportIDParts) < 2 { //nolint:mnd
		return ""
	}
	reportID := reportIDParts[1]
	//nolint:lll
	return fmt.Sprintf("https://eu-central-1.console.aws.amazon.com/codesuite/codebuild/974517877189/testReports/reports/%s-reports/%s?region=eu-central-1", project, reportID)
}

//-----------------------------------------------------------------------------

func buildLogLink(buildID string) string {
	splitBuildID := strings.Split(buildID, ":")
	project := splitBuildID[0]
	encodedBuildID := strings.Join(splitBuildID, "%3A")
	//nolint:lll
	return fmt.Sprintf("https://eu-central-1.console.aws.amazon.com/codesuite/codebuild/974517877189/projects/%s/build/%s?region=eu-central-1", project, encodedBuildID)
}

//-----------------------------------------------------------------------------

func projectNameFromBuildID(buildID string) string {
	return strings.Split(buildID, ":")[0]
}

//-----------------------------------------------------------------------------
