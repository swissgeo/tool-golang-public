package codebuild

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	aws_codebuild "github.com/aws/aws-sdk-go-v2/service/codebuild"
	codebuild_types "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/geoadmin/tool-golang-bgdi/lib/aws/arn"
	"github.com/geoadmin/tool-golang-bgdi/lib/aws/config"
	"github.com/geoadmin/tool-golang-bgdi/lib/fmtc"
	"github.com/geoadmin/tool-golang-bgdi/lib/log"
	"github.com/geoadmin/tool-golang-bgdi/lib/str"
	"github.com/spf13/pflag"
)

type Client struct {
	client aws_codebuild.Client
}

type BuildID string

func ParseBuildID(s string) (BuildID, error) {
	a, err := arn.ParseBuild(s)
	if err == nil {
		return BuildID(a.ResourceID()), nil
	}
	re := regexp.MustCompile("^[a-zA-Z0-9-]+:[0-9a-f-]+$")
	if !re.MatchString(s) {
		return "", fmt.Errorf(`"%s" does not look like a valid build ID:`+
			` it should be a full ARN or something like project-name:abdc-1234-4242`, s)
	}
	return BuildID(s), nil
}

type Build struct {
	ID     BuildID
	Arn    arn.Build
	build  codebuild_types.Build
	report Report
}

type Report struct {
	Arn        arn.BuildReport
	TestCases  map[TestStatus][]codebuild_types.TestCase
	Duration   time.Duration
	TestsCount int
}

func newBuild(b codebuild_types.Build) (Build, error) {
	if b.Arn == nil {
		return Build{}, fmt.Errorf("invalid build: nil ARN pointer! %v", b)
	}
	a, err := arn.ParseBuild(*b.Arn)
	if err != nil {
		return Build{}, fmt.Errorf("invalid build ARN for %v: %w", b, err)
	}
	buildID, err := ParseBuildID(a.String())
	if err != nil {
		return Build{}, fmt.Errorf("unable to determine BuildID for %v: %w", b, err)
	}
	return Build{
		Arn:   a,
		ID:    buildID,
		build: b,
	}, nil
}

func (b Build) ShortString(colorise bool) string {
	statusTime := time.Now()
	if b.build.EndTime != nil {
		statusTime = *b.build.EndTime
	}
	statusStr := fmt.Sprintf("%v", b.build.BuildStatus)
	if colorise {
		statusColor := map[codebuild_types.StatusType]fmtc.Color{
			codebuild_types.StatusTypeSucceeded:  fmtc.Green,
			codebuild_types.StatusTypeFailed:     fmtc.Red,
			codebuild_types.StatusTypeFault:      fmtc.Red,
			codebuild_types.StatusTypeTimedOut:   fmtc.Red,
			codebuild_types.StatusTypeStopped:    fmtc.Red,
			codebuild_types.StatusTypeInProgress: fmtc.NoColor,
		}
		color := statusColor[b.build.BuildStatus]
		statusStr = fmtc.Colorise(color, statusStr)
	}
	return fmt.Sprintf("Status: %s at %v", statusStr, statusTime)
}

func (b Build) Status() codebuild_types.StatusType {
	return b.build.BuildStatus
}

func (b Build) Succeeded() bool {
	return b.Status() == codebuild_types.StatusTypeSucceeded
}

func (b Build) String(colorise bool, detailed bool) string {
	s := fmt.Sprintf("Build %s\n%s\n%s\n", b.Arn.String(), b.Link(), b.ShortString(colorise))
	if b.report.FaultsCount() > 0 {
		color := fmtc.NoColor
		if colorise {
			color = fmtc.Red
		}
		s += fmt.Sprintf("\n%s", fmtc.Colorise(color, b.report.String(detailed)))
	}
	return s
}

func testCaseToString(t codebuild_types.TestCase, detailed bool) string {
	prefix := "[invalid prefix]"
	name := "[invalid name]"
	duration := "[invalid duration]"
	if t.Prefix != nil {
		prefix = *t.Prefix
	}
	if t.Name != nil {
		name = *t.Name
	}
	if t.DurationInNanoSeconds != nil {
		duration = time.Duration(*t.DurationInNanoSeconds).String()
	}
	s := fmt.Sprintf("- %s.%s (%s)\n", prefix, name, duration)
	if detailed {
		message := "[invalid message]"
		if t.Message != nil {
			message = *t.Message
		}
		s += fmt.Sprintf("\t%s\n\n", strings.ReplaceAll(message, "\n", "\n\t"))
	}
	return s
}

func (r Report) String(detailed bool) string {
	s := fmt.Sprintf("Build report %s\n", r.Arn.String())
	for status, tcs := range r.TestCases {
		if len(tcs) > 0 {
			s += fmt.Sprintf("%d tests in state %s\n", len(tcs), status)
		}
		for _, t := range tcs {
			s += testCaseToString(t, detailed)
		}
	}
	faultyPct := "NaN%"
	if r.TestsCount != 0 {
		faultyPct = fmt.Sprintf("%d%%", r.FaultsCount()*100/r.TestsCount)
	}
	s += fmt.Sprintf("\nTests failures/errors %s (%d/%d)\n", faultyPct, r.FaultsCount(), r.TestsCount)
	s += fmt.Sprintf("%s\n", r.Link())
	return s
}

func (r Report) FaultsCount() int {
	return len(r.TestCases[TestStatusError]) + len(r.TestCases[TestStatusFailed])
}

func newReport(r codebuild_types.Report) (Report, error) {
	if r.Arn == nil {
		return Report{}, fmt.Errorf("invalid report: nil ARN pointer! %v", r)
	}
	a, err := arn.ParseBuildReport(*r.Arn)
	if err != nil {
		return Report{}, fmt.Errorf("invalid report ARN for %v: %w", r, err)
	}
	if r.TestSummary == nil {
		return Report{}, fmt.Errorf("nil TestSummary pointer for %v", r)
	}
	if r.TestSummary.DurationInNanoSeconds == nil {
		return Report{}, fmt.Errorf("nil duration pointer for %v", r.TestSummary)
	}
	if r.TestSummary.Total == nil {
		return Report{}, fmt.Errorf("nil total pointer for %v", r.TestSummary)
	}
	return Report{
		Arn:        a,
		Duration:   time.Duration(*r.TestSummary.DurationInNanoSeconds) * time.Nanosecond,
		TestsCount: int(*r.TestSummary.Total),
	}, nil
}

func DefineNewClientFlags(flags *pflag.FlagSet) {
	config.DefineFlags(flags)
}

func NewClient(ctx context.Context, flags pflag.FlagSet) (Client, error) {
	config, err := config.LoadConfig(ctx, flags)
	if err != nil {
		return Client{}, fmt.Errorf("failed to load AWS config: %w", err)
	}
	log.Debug("Loaded AWS config: %+v", config)
	client := aws_codebuild.NewFromConfig(config)
	if client == nil {
		return Client{}, errors.New("codebuild client is nil")
	}
	return Client{
		client: *client,
	}, nil
}

type GetOptions struct {
	WaitForCompletion bool
	WaitSleepInterval time.Duration
	ProgressOutput    *os.File
	FetchReport       bool
	TestCases         []TestStatus
}

func DefineGetBuildFlags(flags *pflag.FlagSet) {
	flags.Duration("wait-interval", 3*time.Second, "How long to wait between checks.") //nolint:mnd
	flags.BoolP("wait", "w", true, "Whether to wait for completion.")
	flags.BoolP("quiet-wait", "q", false, "Disable wait feedback indicator.")
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
		log.Debug("BatchGetBuilds(%+v)", getBuildsInput)
		result, e := c.client.BatchGetBuilds(ctx, &getBuildsInput)
		log.Debug("BatchGetBuilds result: %+v, %v", result, e)
		if e != nil {
			return Build{}, fmt.Errorf("failed to get build status: %w", e)
		}
		e = isBatchGetBuildsOutputSane(result)
		if e != nil {
			return Build{}, e
		}
		if opt.WaitForCompletion && !result.Builds[0].BuildComplete {
			log.Debug("waiting %s before attempting to fetch build info", opt.WaitSleepInterval)
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
			c.fetchReport(ctx, &b, opt.TestCases)
		}
		return b, nil
	}
}

func (c Client) fetchReport(ctx context.Context, b *Build, testCases []TestStatus) {
	if len(b.build.ReportArns) == 0 {
		log.Warn("not report for build: %#v", b.build)
		return
	}
	if len(b.build.ReportArns) > 1 {
		log.Warn("multiple reports found for build, only considering the first one: %#v", b.build)
	}
	a, err := arn.ParseBuildReport(b.build.ReportArns[0])
	if err != nil {
		log.Warn("failed to parse report ARN: %v: %v", b.build.ReportArns[0], err)
		return
	}
	rs, err := c.client.BatchGetReports(ctx, &aws_codebuild.BatchGetReportsInput{
		ReportArns: []string{a.String()},
	})
	if err != nil {
		log.Warn("failed to fetch report for %v: %v", a, err)
		return
	}
	if len(rs.Reports) == 0 {
		log.Warn("received no report for %v", a)
		return
	}
	if len(rs.Reports) > 1 {
		log.Warn("received multiple reports, only considering the first one for %v: %v", a, rs.Reports)
	}
	b.report, err = newReport(rs.Reports[0])
	if err != nil {
		log.Warn("failed to attach report: %v", err)
		return
	}
	if len(testCases) > 0 {
		b.report.TestCases = make(map[TestStatus][]codebuild_types.TestCase)
	}
	for _, status := range testCases {
		err = c.fetchTestCases(ctx, status, &b.report)
		if err != nil {
			log.Warn("failed to fetch test cases: %v", err)
		}
	}
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

type StartOptions struct {
	SourceVersion string
	Environment   []codebuild_types.EnvironmentVariable
	Timeout       time.Duration
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

	log.Debug("Calling StartBuild with argument %+v", input)
	result, err := c.client.StartBuild(ctx, input)
	log.Debug("StarBuild result: %v, %v", result, err)
	if err != nil {
		return Build{}, fmt.Errorf("failed to start build for %s: %w", project, err)
	}
	if result.Build == nil {
		return Build{}, fmt.Errorf("build started but nil Build pointer! %v", result)
	}
	return newBuild(*result.Build)
}
