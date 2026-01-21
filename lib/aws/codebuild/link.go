package codebuild

import "fmt"

func (r Report) Link() string {
	return fmt.Sprintf("https://%s.console.aws.amazon.com/codesuite/codebuild/%s/testReports/reports/%s/%s?region=%s",
		r.Arn.Region(),
		r.Arn.AccountID(),
		r.Arn.ResourceSubType(),
		r.Arn.ResourceID(),
		r.Arn.Region(),
	)
}
