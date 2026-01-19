package codebuild

import "fmt"

func (b Build) Link() string {
	return fmt.Sprintf("https://%s.console.aws.amazon.com/codesuite/codebuild/%s/projects/%s/build/%s?region=%s",
		b.Arn.Region(),
		b.Arn.AccountID(),
		b.Arn.ResourceSubType(),
		b.Arn.ResourceID(),
		b.Arn.Region(),
	)
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
