package codebuild

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	aws_codebuild "github.com/aws/aws-sdk-go-v2/service/codebuild"
	codebuild_types "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/geoadmin/tool-golang-bgdi/lib/aws/arn"
	"github.com/geoadmin/tool-golang-bgdi/lib/aws/config"
	"github.com/geoadmin/tool-golang-bgdi/lib/log"
	"github.com/geoadmin/tool-golang-bgdi/lib/str"
	"github.com/spf13/pflag"
)

var ErrFetchReportFailed = errors.New("failure while attempting to fetch build report")

func DefineNewClientFlags(flags *pflag.FlagSet) {
	config.DefineFlags(flags)
}

func DefineGetBuildFlags(flags *pflag.FlagSet) {
	flags.Duration("wait-interval", 3*time.Second, "How long to wait between checks.") //nolint:mnd
	flags.BoolP("wait", "w", true, "Whether to wait for completion.")
	flags.BoolP("quiet-wait", "q", false, "Disable wait feedback indicator.")
}

func DefineStartBuildFlags(flags *pflag.FlagSet) {
	flags.StringP("source-version", "s", "",
		"Version of the build input to be built."+
			" Empty string means the project's default is used.")
	flags.DurationP("timeout", "t", 0,
		"How long should Codebuild wait before timing out the build."+
			" Zero means the project's default is used.")
	flags.StringArrayP("environment", "e", []string{},
		`Environment variables to override for this build.`+
			` In the form of "VAR=value". Can be repeated multiple times.`)
}

type GetOptions struct {
	WaitForCompletion bool
	WaitSleepInterval time.Duration
	ProgressOutput    *os.File
	FetchReport       bool
	TestCases         []TestStatus
}

type StartOptions struct {
	SourceVersion string
	Environment   []codebuild_types.EnvironmentVariable
	Timeout       time.Duration
}

func ParseGetFlags(flags pflag.FlagSet) (GetOptions, error) {
	wait, err := flags.GetBool("wait")
	if err != nil {
		return GetOptions{}, fmt.Errorf(`invalid "wait" flag value: %w`, err)
	}
	progressOutput := os.Stderr
	quiet, err := flags.GetBool("quiet-wait")
	if err != nil {
		return GetOptions{}, fmt.Errorf(`invalid "quiet-wait" flag value: %w`, err)
	}
	if quiet {
		progressOutput = nil
	}
	interval, err := flags.GetDuration("wait-interval")
	if err != nil {
		return GetOptions{}, fmt.Errorf(`invalid "wait-interval" flag value: %w`, err)
	}
	if interval <= 0 {
		return GetOptions{}, fmt.Errorf(`invalid "wait-interval" flag value: must be greater than zero: %v`, interval)
	}
	return GetOptions{
		WaitForCompletion: wait,
		WaitSleepInterval: interval,
		ProgressOutput:    progressOutput,
	}, nil
}

func ParseStartFlags(flags pflag.FlagSet) (StartOptions, error) {
	sourceVersion, err := flags.GetString("source-version")
	if err != nil {
		return StartOptions{}, fmt.Errorf(`unable to determine "source-version" flag value: %w`, err)
	}

	timeout, err := flags.GetDuration("timeout")
	if err != nil {
		return StartOptions{}, fmt.Errorf(`invalid "timeout" flag value: %w`, err)
	}

	environmentStrings, err := flags.GetStringArray("environment")
	if err != nil {
		return StartOptions{}, fmt.Errorf(`invalid "environment" flag value: %w`, err)
	}
	var environment []codebuild_types.EnvironmentVariable
	for _, e := range environmentStrings {
		nameValue := strings.SplitN(e, "=", 2) //nolint:mnd
		if len(nameValue) != 2 {               //nolint:mnd
			return StartOptions{}, fmt.Errorf(`invalid "environment" flag value: %s`, e)
		}
		environment = append(environment, codebuild_types.EnvironmentVariable{
			Name:  &nameValue[0],
			Value: &nameValue[1],
			Type:  codebuild_types.EnvironmentVariableTypePlaintext,
		})
	}

	return StartOptions{
		SourceVersion: sourceVersion,
		Environment:   environment,
		Timeout:       timeout,
	}, nil
}

type Client struct {
	client aws_codebuild.Client
}

func NewClient(ctx context.Context, flags pflag.FlagSet) (Client, error) {
	config, err := config.LoadConfig(ctx, flags)
	if err != nil {
		return Client{}, fmt.Errorf("failed to load AWS config: %w", err)
	}
	log.Debugf("Loaded AWS config: %+v", config)
	client := aws_codebuild.NewFromConfig(config)
	if client == nil {
		return Client{}, errors.New("codebuild client is nil")
	}
	return Client{
		client: *client,
	}, nil
}

func (c Client) GetBuildWithFlags(ctx context.Context, build BuildID, flags pflag.FlagSet) (Build, error) {
	opt, err := ParseGetFlags(flags)
	if err != nil {
		return Build{}, fmt.Errorf("failed to parse get flags: %w", err)
	}
	return c.GetBuildWithOptions(ctx, build, opt)
}

func isBatchGetBuildsOutputSane(b *aws_codebuild.BatchGetBuildsOutput) error {
	if b == nil {
		return errors.New("nil pointer for BatchGetBuildsOutput")
	}
	if len(b.Builds) == 0 {
		return fmt.Errorf("no build found: %+v", b)
	}
	if len(b.Builds) > 1 {
		return fmt.Errorf("more than one build found: %+v", b)
	}
	return nil
}

func (c Client) GetBuildWithOptions(ctx context.Context, buildID BuildID, opt GetOptions) (Build, error) {
	if len(opt.TestCases) > 0 && !opt.FetchReport {
		return Build{}, fmt.Errorf("invalid options combination: fetching test cases without report for %v: %v", buildID, opt)
	}
	getBuildsInput := aws_codebuild.BatchGetBuildsInput{
		Ids: []string{string(buildID)},
	}
	start := time.Now()
	for {
		log.Debugf("BatchGetBuilds(%+v)", getBuildsInput)
		result, e := c.client.BatchGetBuilds(ctx, &getBuildsInput)
		log.Debugf("BatchGetBuilds result: %+v, %v", result, e)
		if e != nil {
			return Build{}, fmt.Errorf("failed to get build status: %w", e)
		}
		e = isBatchGetBuildsOutputSane(result)
		if e != nil {
			return Build{}, e
		}
		if opt.WaitForCompletion && !result.Builds[0].BuildComplete {
			log.Debugf("waiting %s before attempting to fetch build info", opt.WaitSleepInterval)
			time.Sleep(opt.WaitSleepInterval)
			if opt.ProgressOutput != nil {
				fmt.Fprintf(opt.ProgressOutput, "Waiting for build for %v...\r", time.Since(start).Truncate(time.Second))
			}
			continue
		}
		b, e := newBuild(result.Builds[0])
		if e != nil {
			return Build{}, e
		}
		if opt.FetchReport {
			e = c.fetchReport(ctx, &b, opt.TestCases)
			if e != nil {
				e = errors.Join(e, ErrFetchReportFailed)
			}
			return b, e
		}
		return b, nil
	}
}

func (c Client) fetchReport(ctx context.Context, b *Build, testCases []TestStatus) error {
	if len(b.build.ReportArns) == 0 {
		return nil
	}
	a, err := arn.ParseBuildReport(b.build.ReportArns[0])
	if err != nil {
		return fmt.Errorf("failed to parse report ARN: %v: %w", b.build.ReportArns[0], err)
	}
	rs, err := c.client.BatchGetReports(ctx, &aws_codebuild.BatchGetReportsInput{
		ReportArns: []string{a.String()},
	})
	if err != nil {
		return err
	}
	if len(rs.Reports) == 0 {
		return fmt.Errorf("received no report for %v", a)
	}
	for i := range rs.Reports {
		r, e := newReport(rs.Reports[i])
		if e != nil {
			return fmt.Errorf("failed to attach report: %w", e)
		}
		if len(testCases) > 0 {
			r.TestCases = make(map[TestStatus][]codebuild_types.TestCase)
		}
		for _, status := range testCases {
			e = c.fetchTestCases(ctx, status, &r)
			if e != nil {
				return fmt.Errorf("failed to fetch test cases: %w", e)
			}
		}
		b.reports = append(b.reports, r)
	}
	return nil
}

func (c Client) fetchTestCases(ctx context.Context, status TestStatus, r *Report) error {
	res, err := c.client.DescribeTestCases(ctx, &aws_codebuild.DescribeTestCasesInput{
		ReportArn: str.Ptr(r.Arn.String()),
		Filter: &codebuild_types.TestCaseFilter{
			Status: str.Ptr(string(status)),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to fetch %s test cases for %s: %w", status, r.Arn.String(), err)
	}
	r.TestCases[status] = res.TestCases
	return nil
}

type TestStatus string

const (
	TestStatusSucceeded = "SUCCEEDED"
	TestStatusFailed    = "FAILED"
	TestStatusError     = "ERROR"
	TestStatusSkipped   = "SKIPPED"
	TestStatusUnknown   = "UNKNOWN"
)

func (c Client) StartBuildWithFlags(ctx context.Context, project string, flags pflag.FlagSet) (Build, error) {
	opt, err := ParseStartFlags(flags)
	if err != nil {
		return Build{}, fmt.Errorf("failed to parse start flags: %w", err)
	}
	return c.StartBuildWithOptions(ctx, project, opt)
}

func (c Client) StartBuildWithOptions(ctx context.Context, project string, opt StartOptions) (Build, error) {
	input := &aws_codebuild.StartBuildInput{
		ProjectName: &project,
	}
	if opt.SourceVersion != "" {
		input.SourceVersion = &opt.SourceVersion
	}
	if len(opt.Environment) != 0 {
		input.EnvironmentVariablesOverride = opt.Environment
	}
	timeout := int32(opt.Timeout.Minutes())
	if timeout != 0 {
		input.TimeoutInMinutesOverride = &timeout
	}

	log.Debugf("Calling StartBuild with argument %+v", input)
	result, err := c.client.StartBuild(ctx, input)
	log.Debugf("StarBuild result: %v, %v", result, err)
	if err != nil {
		return Build{}, fmt.Errorf("failed to start build for %s: %w", project, err)
	}
	if result.Build == nil {
		return Build{}, fmt.Errorf("build started but nil Build pointer! %v", result)
	}
	return newBuild(*result.Build)
}
