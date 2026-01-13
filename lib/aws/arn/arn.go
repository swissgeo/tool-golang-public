package arn

import (
	"errors"
	"fmt"
	"strings"

	aws_arn "github.com/aws/aws-sdk-go-v2/aws/arn"
)

type ARN struct {
	arn aws_arn.ARN
	// consider resource "type/name:id"
	//	resourceType = "type"
	//	resourceID = "name:id"
	//	resourceSubType = "name"
	//	resourceSubID = "id"
	resourceType    string
	resourceID      string
	resourceSubType string
	resourceSubID   string
}

type Role = ARN
type Build = ARN
type BuildReport = ARN

var ErrParseArn = errors.New("unable to parse ARN")
var ErrWrongService = errors.New("unexpected service field value")
var ErrWrongResourceType = errors.New("unexpected resource type")
var ErrWrongResourceSubType = errors.New("unexpected resource subtype")

func (a ARN) Partition() string {
	return a.arn.Partition
}

func (a ARN) Service() string {
	return a.arn.Service
}

func (a ARN) Region() string {
	return a.arn.Region
}

func (a ARN) AccountID() string {
	return a.arn.AccountID
}

func (a ARN) Resource() string {
	return a.arn.Resource
}

func (a ARN) ResourceType() string {
	return a.resourceType
}

func (a ARN) ResourceID() string {
	return a.resourceID
}

func (a ARN) ResourceSubType() string {
	return a.resourceSubType
}

func (a ARN) ResourceSubID() string {
	return a.resourceSubID
}

func split2(s string, separator string) (string, string) {
	split := strings.SplitN(s, separator, 2) //nolint:mnd
	switch {
	case len(split) == 1:
		return "", split[0]
	case len(split) == 2: //nolint:mnd
		return split[0], split[1]
	default:
		panic(fmt.Sprintf(`expected one or two elements as result of SplitN("%s", "%s", 2) = %v`, s, separator, split))
	}
}

func parseResource(a *ARN) {
	// TODO: the type/id separator can also be a colon
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference-arns.html
	a.resourceType, a.resourceID = split2(a.arn.Resource, "/")

	subTypeSeparator := "/"
	if strings.Contains(a.resourceID, ":") {
		subTypeSeparator = ":"
	}
	a.resourceSubType, a.resourceSubID = split2(a.resourceID, subTypeSeparator)
}

func (a ARN) String() string {
	return a.arn.String()
}

func Parse(s string) (ARN, error) {
	return ParseType(s, "", "", "")
}

func ParseType(arn string, expectedService string, expectedType string, expectedSubType string) (ARN, error) {
	var err error
	a := ARN{}
	a.arn, err = aws_arn.Parse(arn)
	if err != nil {
		return a, errors.Join(ErrParseArn, err)
	}
	parseResource(&a)
	if expectedService != "" && expectedService != a.Service() {
		return a, fmt.Errorf(`expected "%s", got "%s", %w`, expectedService, a.Service(), ErrWrongService)
	}
	if expectedType != "" && expectedType != a.ResourceType() {
		return a, fmt.Errorf(`expected "%s", got "%s", %w`, expectedType, a.ResourceType(), ErrWrongResourceType)
	}
	if expectedSubType != "" && expectedSubType != a.ResourceSubType() {
		return a, fmt.Errorf(`expected "%s", got "%s", %w`, expectedSubType, a.ResourceSubType(), ErrWrongResourceSubType)
	}
	return a, nil
}

func ParseRole(s string) (Role, error) {
	return ParseType(s, "iam", "role", "")
}

func ParseBuild(s string) (Build, error) {
	return ParseType(s, "codebuild", "build", "")
}

func ParseBuildReport(s string) (BuildReport, error) {
	return ParseType(s, "codebuild", "report", "")
}

func (a ARN) Link() string {
	switch a.ResourceType() {
	case "build":
		return linkForBuild(a)
	case "report":
		return linkForBuildReport(a)
	default:
		return ""
	}
}

func linkForBuildReport(a BuildReport) string {
	return fmt.Sprintf("https://%s.console.aws.amazon.com/codesuite/codebuild/%s/testReports/reports/%s/%s?region=%s",
		a.Region(),
		a.AccountID(),
		a.ResourceSubType(),
		a.ResourceID(),
		a.Region(),
	)
}

func linkForBuild(a Build) string {
	return fmt.Sprintf("https://%s.console.aws.amazon.com/codesuite/codebuild/%s/projects/%s/build/%s?region=%s",
		a.Region(),
		a.AccountID(),
		a.ResourceSubType(),
		a.ResourceID(),
		a.Region(),
	)
}
