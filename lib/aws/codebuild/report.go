package codebuild

import (
	"fmt"
	"strings"
	"time"

	codebuild_types "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/geoadmin/tool-golang-bgdi/lib/aws/arn"
)

type Report struct {
	Arn        arn.BuildReport
	TestCases  map[TestStatus][]codebuild_types.TestCase
	Duration   time.Duration
	TestsCount int
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
	var s strings.Builder
	fmt.Fprintf(&s, "Build report %s\n", r.Arn.String())
	for status, tcs := range r.TestCases {
		if len(tcs) > 0 {
			fmt.Fprintf(&s, "%d tests in state %s\n", len(tcs), status)
		}
		for _, t := range tcs {
			s.WriteString(testCaseToString(t, detailed))
		}
	}
	faultyPct := "NaN%"
	if r.TestsCount != 0 {
		faultyPct = fmt.Sprintf("%d%%", r.FaultsCount()*100/r.TestsCount)
	}
	fmt.Fprintf(&s, "\nTests failures/errors %s (%d/%d)\n", faultyPct, r.FaultsCount(), r.TestsCount)
	fmt.Fprintf(&s, "%s\n", r.Link())
	return s.String()
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

func (r Report) Link() string {
	return fmt.Sprintf("https://%s.console.aws.amazon.com/codesuite/codebuild/%s/testReports/reports/%s/%s?region=%s",
		r.Arn.Region(),
		r.Arn.AccountID(),
		r.Arn.ResourceSubType(),
		r.Arn.ResourceID(),
		r.Arn.Region(),
	)
}
