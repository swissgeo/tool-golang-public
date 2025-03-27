package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
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

func initPrint(cmd *cobra.Command) {
	noColor, e := cmd.Flags().GetBool("no-color")
	if e != nil {
		log.Fatal(e)
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
}

//-----------------------------------------------------------------------------

// Returns a CodeBuild client
func getClient(ctx context.Context, cmd *cobra.Command) *codebuild.Client {
	noProfile, e := cmd.Flags().GetBool("no-profile")
	if e != nil {
		log.Fatal(e)
	}
	role := cmd.Flag("role").Value.String()
	var cfg aws.Config

	switch {
	case role != "":
		var cred *sts.AssumeRoleOutput
		cfg, e = config.LoadDefaultConfig(ctx, config.WithRegion("eu-central-1"))
		if e != nil {
			log.Fatalf("failed to load configuration: %v", e)
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
			log.Fatalf("failed to assume role %s: %v", role, e)
		}

		cfg, e = config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				*cred.Credentials.AccessKeyId,
				*cred.Credentials.SecretAccessKey,
				*cred.Credentials.SessionToken,
			),
		))
		if e != nil {
			log.Fatalf("failed to load configuration with role credentials: %v", e)
		}
	case noProfile:
		cfg, e = config.LoadDefaultConfig(ctx, config.WithRegion("eu-central-1"))
		if e != nil {
			log.Fatalf("failed to load configuration: %v", e)
		}
	default:
		cfg, e = config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile("swisstopo-bgdi-builder"))
		if e != nil {
			log.Fatalf("failed to load configuration: %v", e)
		}
	}

	client := codebuild.NewFromConfig(cfg)

	return client
}

//-----------------------------------------------------------------------------

func waitForBuild(ctx context.Context, client *codebuild.Client, buildID string) *codebuild.BatchGetBuildsOutput {
	var result *codebuild.BatchGetBuildsOutput
	var e error

	input := &codebuild.BatchGetBuildsInput{
		Ids: []string{buildID},
	}
	for {
		time.Sleep(5 * time.Second) //nolint:mnd
		result, e = client.BatchGetBuilds(ctx, input)
		if e != nil {
			log.Fatalf("failed to get build status: %v", e)
		}
		if len(result.Builds) == 0 {
			log.Fatalf("no build found with id: %s", buildID)
		}
		if result.Builds[0].BuildComplete {
			fmt.Printf("E2E tests finished with status: %s\n", result.Builds[0].BuildStatus)
			break
		}
	}

	return result
}

//-----------------------------------------------------------------------------

func printTestResult(ctx context.Context, client *codebuild.Client, result *codebuild.BatchGetBuildsOutput) {
	if result.Builds[0].BuildStatus == types.StatusTypeSucceeded {
		cPrintln(fmtc.Green, "E2E tests succeeded")
		os.Exit(0)
	}
	// If the build failed, print the reports
	cPrintln(fmtc.Red, "E2E tests failed !")
	for _, report := range result.Builds[0].ReportArns {
		e := printTestReport(ctx, client, report, *result.Builds[0].Id)
		if e != nil {
			log.Fatal(e)
		}
	}
}

//-----------------------------------------------------------------------------

func printTestReport(ctx context.Context, client *codebuild.Client, reportArn string, buildID string) error {
	var e error

	// First get the errors
	e = printTestReportByStatus(ctx, client, reportArn, "ERROR")
	if e != nil {
		return e
	}

	// Then gets the failure
	e = printTestReportByStatus(ctx, client, reportArn, "FAILED")
	if e != nil {
		return e
	}

	cPrintln(fmtc.Red, "\nTest report link:")
	cPrintln(fmtc.Red, "-----------------")
	cPrintln(fmtc.Red, buildReportLink(projectNameFromBuildID(buildID), reportArn))

	return nil
}

//-----------------------------------------------------------------------------

func printTestReportByStatus(ctx context.Context, client *codebuild.Client, reportArn string, status string) error {
	filter := &types.TestCaseFilter{
		Status: &status,
	}
	input := &codebuild.DescribeTestCasesInput{
		ReportArn: &reportArn,
		Filter:    filter,
	}
	r, e := client.DescribeTestCases(ctx, input)
	if e != nil {
		return fmt.Errorf("failed to describe test case %s for reportARN=%s: %w", status, reportArn, e)
	}
	printTestCases(r.TestCases, status)

	return nil
}

//-----------------------------------------------------------------------------

func printTestCases(tests []types.TestCase, statusType string) {
	if len(tests) > 0 {
		cPrintf(fmtc.Red, "Tests with %s:\n", statusType)
		cPrintf(fmtc.Red, "-----------%s-\n", strings.Repeat("-", len(statusType)))
		for _, t := range tests {
			const nanoSeconds int64 = 1_000_000_000
			cPrintf(fmtc.Red, "- %s.%s (%d s)", *t.Prefix, *t.Name, *t.DurationInNanoSeconds/nanoSeconds)
			cPrintf(fmtc.Red, "    %s\n", strings.ReplaceAll(*t.Message, "\n", "\n    "))
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
