package cmd //nolint:testpackage // Exercise unexported flag parsers without invoking AWS.

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/e2e-tests/cmd/organization"
)

const (
	testCustomProfile = "custom"
	testAPIMarker     = "api"
)

func newFlagCommand() *cobra.Command {
	command := &cobra.Command{}
	flags := command.Flags()
	flags.String("org", string(organization.GEOADMIN), "")
	flags.Bool("no-progress", false, "")
	flags.Bool("no-color", false, "")
	flags.Int("interval", 1, "")
	flags.Bool("detailed", false, "")
	flags.String("role", "", "")
	flags.String("profile", "", "")
	flags.String("staging", "dev", "")
	flags.String("revision", "", "")
	flags.Bool("data-tests", false, "")
	flags.StringArray("tests", nil, "")
	flags.StringArray("markers", nil, "")
	flags.String("test-id", "", "")
	return command
}

func TestCommonFlagsParse(t *testing.T) {
	for _, tt := range []struct {
		name, org, role, profile, wantProfile string
	}{
		{"geoadmin default", string(organization.GEOADMIN), "", "", "swisstopo-swissgeo-builder"},
		{"swissgeo default", string(organization.SWISSGEO), "", "", "swisstopo-swissgeo-builder"},
		{"explicit profile", string(organization.GEOADMIN), "", testCustomProfile, testCustomProfile},
		{"explicit role", string(organization.SWISSGEO), "assumed-role", "", ""},
		{"role and profile", string(organization.SWISSGEO), "assumed-role", testCustomProfile, testCustomProfile},
	} {
		t.Run(tt.name, func(t *testing.T) {
			command := newFlagCommand()
			require.NoError(t, command.ParseFlags([]string{
				"--org=" + tt.org, "--role=" + tt.role, "--profile=" + tt.profile,
				"--no-progress", "--interval=7", "--detailed",
			}))
			var flags CommonCmdFlags
			require.NoError(t, flags.parse(command))
			assert.Equal(t, organization.Organization(tt.org), flags.Organization)
			assert.False(t, flags.ShowProgress)
			assert.Equal(t, 7, flags.Interval)
			assert.True(t, flags.Detailed)
			profile, err := command.Flags().GetString("profile")
			require.NoError(t, err)
			assert.Equal(t, tt.wantProfile, profile)
		})
	}
}

func TestStartFlagsDefaults(t *testing.T) {
	for _, tt := range []struct {
		org, revision string
	}{
		{string(organization.GEOADMIN), "master"},
		{string(organization.SWISSGEO), "main"},
	} {
		t.Run(tt.org, func(t *testing.T) {
			command := newFlagCommand()
			require.NoError(t, command.Flags().Set("org", tt.org))
			var flags StartCmdFlags
			require.NoError(t, flags.parse(command))
			assert.Equal(t, "dev", flags.Staging)
			assert.Equal(t, tt.revision, flags.Revision)
			assert.Empty(t, flags.Tests)
			assert.Empty(t, flags.Markers)
			assert.False(t, flags.DoDataTest)
			assert.True(t, flags.ShowProgress)
			assert.Equal(t, 1, flags.Interval)
			assert.False(t, flags.Detailed)
		})
	}
}

func TestStartFlagsSelections(t *testing.T) {
	for _, tt := range []struct {
		org         string
		tests, want []string
	}{
		{
			string(organization.GEOADMIN),
			[]string{"api.test_search", "test_login"},
			[]string{"tests.api.test_search", "tests.test_login"},
		},
		{
			string(organization.SWISSGEO),
			[]string{"api/test_search.py", "test_login.py"},
			[]string{"tests/api/test_search.py", "tests/test_login.py"},
		},
	} {
		t.Run(tt.org, func(t *testing.T) {
			command := newFlagCommand()
			require.NoError(t, command.ParseFlags([]string{
				"--org=" + tt.org, "--staging=int", "--revision=feature/tests", "--data-tests",
				"--tests=" + tt.tests[0], "--tests=" + tt.tests[1], "--markers=api", "--markers=slow",
			}))
			var flags StartCmdFlags
			require.NoError(t, flags.parse(command))
			assert.Equal(t, "int", flags.Staging)
			assert.Equal(t, "feature/tests", flags.Revision)
			assert.True(t, flags.DoDataTest)
			assert.Equal(t, tt.want, flags.Tests)
			assert.Equal(t, []string{testAPIMarker, "slow"}, flags.Markers)
		})
	}
}

func TestGetFlagsParse(t *testing.T) {
	command := newFlagCommand()
	require.NoError(t, command.ParseFlags([]string{"--test-id=e2e-tests-dev-pr:abcd-1234", "--no-progress"}))
	var flags GetCmdFlags
	require.NoError(t, flags.parse(command))
	assert.Equal(t, "e2e-tests-dev-pr:abcd-1234", flags.TestID)
	assert.False(t, flags.ShowProgress)
}

func TestFlagParserErrors(t *testing.T) {
	for _, flagName := range []string{
		"no-progress", "interval", "detailed", "role", "profile", "data-tests", "tests", "markers", "test-id",
	} {
		t.Run(flagName, func(t *testing.T) {
			command := newFlagCommand()
			wantType := command.Flags().Lookup(flagName).Value.Type()
			// Give the flag an incompatible type to check error propagation.
			command.Flags().Lookup(flagName).Value = command.Flags().Lookup("interval").Value
			if flagName == "interval" {
				command.Flags().Lookup(flagName).Value = command.Flags().Lookup("org").Value
			}
			wantError := fmt.Sprintf("trying to get %s value of flag of type %s",
				wantType, command.Flags().Lookup(flagName).Value.Type())
			if flagName == "test-id" {
				var flags GetCmdFlags
				require.EqualError(t, flags.parse(command), wantError)
			} else {
				var flags StartCmdFlags
				require.EqualError(t, flags.parse(command), wantError)
			}
		})
	}
	var flags GetCmdFlags
	command := newFlagCommand()
	command.Flags().Lookup("profile").Value = command.Flags().Lookup("interval").Value
	require.EqualError(t, flags.parse(command), "trying to get string value of flag of type int")
}

func TestProjectNameAndBoolToStr(t *testing.T) {
	for _, staging := range []string{"dev", "int", "prod"} {
		assert.Equal(t, "e2e-tests-"+staging+"-pr", projectName(staging))
	}
	assert.Equal(t, "1", boolToStr(true))
	assert.Equal(t, "0", boolToStr(false))
}
