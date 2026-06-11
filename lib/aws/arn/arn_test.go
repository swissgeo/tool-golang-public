package arn_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/geoadmin/tool-golang-bgdi/lib/aws/arn"
)

func TestParse(t *testing.T) {
	s := "arn:aws2:codebuild:mars-south-42:1337:report/report-group:report-id"
	a, e := arn.Parse(s)
	require.NoError(t, e)
	require.Equal(t, s, a.String())
	require.Equal(t, "aws2", a.Partition())
	require.Equal(t, "codebuild", a.Service())
	require.Equal(t, "mars-south-42", a.Region())
	require.Equal(t, "1337", a.AccountID())
	require.Equal(t, "report/report-group:report-id", a.Resource())
	require.Equal(t, "report", a.ResourceType())
	require.Equal(t, "report-group:report-id", a.ResourceID())
	require.Equal(t, "report-group", a.ResourceSubType())
	require.Equal(t, "report-id", a.ResourceSubID())

	_, e = arn.Parse("not-an-arn")
	require.ErrorIs(t, e, arn.ErrParseArn)
}

func TestParseType(t *testing.T) {
	const wrongValue = "wrong"
	defaultArn := "arn:aws2:codebuild:mars-south-42:1337:report/report-group:report-id"
	testCases := map[string]struct {
		arn     string
		svc     string
		rType   string
		subType string
		err     error
	}{
		"no-type": {},
		"service": {
			svc: "codebuild",
		},
		"resource-type": {
			rType: "report",
		},
		"resource-sub-type": {
			subType: "report-group",
		},
		"invalid-arn": {
			arn: wrongValue,
			err: arn.ErrParseArn,
		},
		"wrong-service": {
			svc: wrongValue,
			err: arn.ErrWrongService,
		},
		"wrong-type": {
			rType: wrongValue,
			err:   arn.ErrWrongResourceType,
		},
		"wrong-sub-type": {
			subType: wrongValue,
			err:     arn.ErrWrongResourceSubType,
		},
		"slash-separated-subtype": {
			arn:     "arn:aws:iam::123:role/aws-service-role/foo/bar",
			subType: "aws-service-role",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.arn == "" {
				tc.arn = defaultArn
			}
			a, e := arn.ParseType(tc.arn, tc.svc, tc.rType, tc.subType)
			if tc.err == nil {
				require.NoError(t, e)
			} else {
				require.ErrorIs(t, e, tc.err)
				return
			}
			require.Equal(t, tc.arn, a.String())
		})
	}
}

func TestParseRole(t *testing.T) {
	var a arn.Role
	a, e := arn.ParseRole("arn:aws:iam::12345:role/some/role/name/with/path")
	require.NoError(t, e)
	require.Equal(t, "iam", a.Service())
	require.Equal(t, "role/some/role/name/with/path", a.Resource())
	require.Equal(t, "some/role/name/with/path", a.ResourceID())
	require.Equal(t, "role", a.ResourceType())

	_, e = arn.ParseRole("not-an-arn")
	require.ErrorIs(t, e, arn.ErrParseArn)

	_, e = arn.ParseRole("arn:aws:s3:::some-bucket")
	require.ErrorIs(t, e, arn.ErrWrongService)

	_, e = arn.ParseRole("arn:aws:iam::12345:")
	require.ErrorIs(t, e, arn.ErrWrongResourceType)

	_, e = arn.ParseRole("arn:aws:iam::12345:user/some/user/name/with/path")
	require.ErrorIs(t, e, arn.ErrWrongResourceType)
}

func TestParseBuild(t *testing.T) {
	var a arn.Build
	a, e := arn.ParseBuild("arn:aws:codebuild:moon-south-1:12345:build/project-name:abcd-1234")
	require.NoError(t, e)
	require.Equal(t, "codebuild", a.Service())
	require.Equal(t, "build/project-name:abcd-1234", a.Resource())
	require.Equal(t, "project-name:abcd-1234", a.ResourceID())
	require.Equal(t, "build", a.ResourceType())

	_, e = arn.ParseBuild("not-an-arn")
	require.ErrorIs(t, e, arn.ErrParseArn)

	_, e = arn.ParseBuild("arn:aws:s3:::some-bucket")
	require.ErrorIs(t, e, arn.ErrWrongService)

	_, e = arn.ParseBuild("arn:aws:codebuild:moon-south-1:12345:")
	require.ErrorIs(t, e, arn.ErrWrongResourceType)

	_, e = arn.ParseBuild("arn:aws:codebuild:moon-south-1:12345:report/group/id")
	require.ErrorIs(t, e, arn.ErrWrongResourceType)
}

func TestParseBuildReport(t *testing.T) {
	var a arn.BuildReport
	a, e := arn.ParseBuildReport(
		"arn:aws:codebuild:jupiter-north-2:42:report/some-project-reports/some-project-reports:1234")
	require.NoError(t, e)
	require.Equal(t, "codebuild", a.Service())
	require.Equal(t, "report/some-project-reports/some-project-reports:1234", a.Resource())
	require.Equal(t, "some-project-reports/some-project-reports:1234", a.ResourceID())
	require.Equal(t, "report", a.ResourceType())
}
