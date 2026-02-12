package codebuild

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	codebuild_types "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/geoadmin/tool-golang-bgdi/lib/aws/arn"
	"github.com/geoadmin/tool-golang-bgdi/lib/fmtc"
)

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
	ID      BuildID
	Arn     arn.Build
	build   codebuild_types.Build
	reports []Report
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

func (b Build) ReportsCount() int {
	return len(b.reports)
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
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Build %s\n%s\n%s\n", b.Arn.String(), b.Link(), b.ShortString(colorise)))
	for _, r := range b.reports {
		if r.FaultsCount() > 0 {
			color := fmtc.NoColor
			if colorise {
				color = fmtc.Red
			}
			s.WriteString(fmt.Sprintf("\n%s", fmtc.Colorise(color, r.String(detailed))))
		}
	}
	return s.String()
}

func (b Build) Link() string {
	return fmt.Sprintf("https://%s.console.aws.amazon.com/codesuite/codebuild/%s/projects/%s/build/%s?region=%s",
		b.Arn.Region(),
		b.Arn.AccountID(),
		b.Arn.ResourceSubType(),
		b.Arn.ResourceID(),
		b.Arn.Region(),
	)
}
