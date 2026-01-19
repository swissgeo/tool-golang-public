package codebuild_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/geoadmin/tool-golang-bgdi/lib/aws/arn"
	"github.com/geoadmin/tool-golang-bgdi/lib/aws/codebuild"
)

func TestBuildLink(t *testing.T) {
	var b codebuild.Build
	var e error
	b.Arn, e = arn.ParseBuild("arn:aws:codebuild:venus-west-13:007:build/secret-project:dead-decaf")
	require.NoError(t, e)
	require.Equal(t,
		"https://venus-west-13.console.aws.amazon.com/"+
			"codesuite/codebuild/007/projects/secret-project/"+
			"build/secret-project:dead-decaf?region=venus-west-13",
		b.Link())
}

func TestBuildReportLink(t *testing.T) {
	var r codebuild.Report
	var e error
	r.Arn, e = arn.ParseBuildReport("arn:aws:codebuild:jupiter-north-2:42:report/some-project-reports:1234")
	require.NoError(t, e)
	require.Equal(t,
		"https://jupiter-north-2.console.aws.amazon.com/"+
			"codesuite/codebuild/42/testReports/reports/"+
			"some-project-reports/some-project-reports:1234"+
			"?region=jupiter-north-2",
		r.Link())
}
