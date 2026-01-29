package config

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	aws_credentials "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/spf13/pflag"

	"github.com/geoadmin/tool-golang-bgdi/lib/aws/arn"
	"github.com/geoadmin/tool-golang-bgdi/lib/log"
	"github.com/geoadmin/tool-golang-bgdi/lib/str"
)

type awsConfigFlags struct {
	RetryMaxAttempts  int
	Region            string
	Profile           string
	Role              arn.Role
	SessionDuration   time.Duration
	SessionNamePrefix string
}

func init() {
	DefineFlags(pflag.CommandLine)
}

func DefineFlags(flags *pflag.FlagSet) {
	flags.Int("max-attempts", 3, "Maximum number of attempts when sending requests to AWS.") //nolint:mnd
	flags.String("region", "eu-central-1", "AWS region to send requests to.")
	flags.String("profile", "swisstopo-bgdi-builder", "AWS CLI profile to use.")
	flags.String("role", "",
		"ARN of an IAM role to use to communicate with AWS. E.g. arn:aws:iam::1234:role/SomeRoleName")
	flags.Duration("session-duration", 45*time.Minute, //nolint:mnd
		"Duration of the role session. Only useful when specifying a role.")
	flags.String("session-prefix", "BgdiAwsConfigAssumeRole",
		"Prefix for the role session name. Only useful when specifying a role.")
}

func parseFlags(flags pflag.FlagSet) (awsConfigFlags, error) {
	var configFlags awsConfigFlags

	region, err := flags.GetString("region")
	if err != nil {
		return awsConfigFlags{}, fmt.Errorf(`unable to determine "region" flag value,`+
			` did you forget to call DefineFlags? %w`,
			err)
	}
	configFlags.Region = region

	profile, err := flags.GetString("profile")
	if err != nil {
		return awsConfigFlags{}, fmt.Errorf(`unable to determine "profile" flag value,`+
			` did you forget to call DefineFlags? %w`, err)
	}
	configFlags.Profile = profile

	maxAttempts, err := flags.GetInt("max-attempts")
	if err != nil {
		return awsConfigFlags{}, fmt.Errorf(`invalid "max-attempts" flag value:  %w`, err)
	}
	configFlags.RetryMaxAttempts = maxAttempts

	roleStr, err := flags.GetString("role")
	if err != nil {
		return awsConfigFlags{}, fmt.Errorf(`unable to determine "role" flag value,`+
			` did you forget to call DefineFlags? %w`,
			err)
	}
	if roleStr != "" {
		role, e := arn.ParseRole(roleStr)
		if e != nil {
			return configFlags, fmt.Errorf(`unable to parse "role" as valid role ARN: %w`, e)
		}
		configFlags.Role = role
	}

	sessionDuration, err := flags.GetDuration("session-duration")
	if err != nil {
		return configFlags, fmt.Errorf(`invalid "session-duration" flag value: %w`, err)
	}
	configFlags.SessionDuration = sessionDuration

	sessionPrefix, err := flags.GetString("session-prefix")
	if err != nil {
		return awsConfigFlags{}, fmt.Errorf(`unable to determine "session-prefix" flag value,`+
			` did you forget to call DefineFlags? %w`,
			err)
	}
	configFlags.SessionNamePrefix = sessionPrefix

	return configFlags, nil
}

func getSessionName(role arn.Role, prefix string) string {
	return fmt.Sprintf("%s/%s", prefix, role.ResourceSubID())
}

func getCredentialsProvider(ctx context.Context, configFlags awsConfigFlags) (aws.CredentialsProvider, error) {
	if reflect.ValueOf(configFlags.Role).IsZero() {
		return nil, nil //nolint:nilnil // That first nil is a perfectly valid
		// value that will be processed correctly by aws_config.WithCredentialsProvider.
	}
	cfg, err := aws_config.LoadDefaultConfig(ctx, aws_config.WithRegion(configFlags.Region))
	if err != nil {
		return nil, err
	}
	log.Debug("Loaded AWS config for STS: %+v", cfg)
	stsClient := sts.NewFromConfig(cfg)

	sessionName := getSessionName(configFlags.Role, configFlags.SessionNamePrefix)
	sessionDurationInt32 := int32(configFlags.SessionDuration.Seconds())
	assumeRoleInput := sts.AssumeRoleInput{
		RoleArn:         str.Ptr(configFlags.Role.String()),
		RoleSessionName: &sessionName,
		DurationSeconds: &sessionDurationInt32,
	}
	log.Debug("Calling stsClient.AssumeRole(%+v)", assumeRoleInput)
	assumeRoleOutput, err := stsClient.AssumeRole(ctx, &assumeRoleInput)
	// NOT printing the output as it contains credentials
	if err != nil {
		return nil, err
	}
	if assumeRoleOutput == nil {
		return nil, errors.New("AssumeRole returned nil pointer")
	}
	if assumeRoleOutput.Credentials == nil {
		return nil, errors.New("AssumeRole Credentials is nil")
	}
	creds := assumeRoleOutput.Credentials
	if creds.AccessKeyId == nil {
		return nil, errors.New("AccessKeyId is nil")
	}
	if creds.SecretAccessKey == nil {
		return nil, errors.New("SecretAccessKey is nil")
	}
	if creds.SessionToken == nil {
		return nil, errors.New("SessionToken is nil")
	}
	return aws_credentials.NewStaticCredentialsProvider(
		*creds.AccessKeyId,
		*creds.SecretAccessKey,
		*creds.SessionToken,
	), nil
}

func getConfigLoaders(ctx context.Context, flags pflag.FlagSet) ([]aws_config.LoadOptionsFunc, error) {
	configFlags, err := parseFlags(flags)
	if err != nil {
		return nil, err
	}
	log.Debug("Parsed AWS config flags: %+v", configFlags)

	credentialsProvider, err := getCredentialsProvider(ctx, configFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials provider for role %s: %w", configFlags.Role, err)
	}

	return []aws_config.LoadOptionsFunc{
		aws_config.WithRetryMaxAttempts(configFlags.RetryMaxAttempts),
		aws_config.WithSharedConfigProfile(configFlags.Profile),
		aws_config.WithRegion(configFlags.Region),
		aws_config.WithCredentialsProvider(credentialsProvider),
	}, nil
}

func LoadConfig(ctx context.Context, flags pflag.FlagSet) (aws.Config, error) {
	loaders, err := getConfigLoaders(ctx, flags)
	if err != nil {
		return aws.Config{}, err
	}
	// aws_config defines the following types:
	//	type LoadOptionsFunc func(*LoadOptions) error
	//	func LoadDefaultConfig(ctx context.Context, optFns ...func(*LoadOptions) error) (cfg aws.Config, err error)
	// 	func WithWhatever(whatever) LoadOptionsFunc
	// One would expect to be able to pass the output of the WithWatever functions
	// to LoadDefaultConfig. One would be wrong as these are different types
	// despite representing the same thing. We need to convert them explicitly,
	// which is what we do here.
	// See also https://groups.google.com/g/golang-nuts/c/4UaVImFuS1M
	convertedLoaders := []func(*aws_config.LoadOptions) error{}
	for i := range loaders {
		convertedLoaders = append(convertedLoaders, loaders[i])
	}
	return aws_config.LoadDefaultConfig(ctx, convertedLoaders...)
}
