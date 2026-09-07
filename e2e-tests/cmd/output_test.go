package cmd //nolint:testpackage // Verify internal output helpers without invoking AWS.

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/e2e-tests/cmd/organization"
	"github.com/swissgeo/tool-golang-public/lib/aws/codebuild"
)

//nolint:reassign // Serial output tests temporarily redirect stdout and restore it during cleanup.
func captureStdout(t *testing.T, run func()) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	original := os.Stdout
	os.Stdout = file
	t.Cleanup(func() {
		os.Stdout = original
		require.NoError(t, file.Close())
	})
	run()
	os.Stdout = original
	data, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	return string(data)
}

func TestPrintStart(t *testing.T) {
	for _, tt := range []struct {
		name    string
		flags   StartCmdFlags
		want    string
		warning bool
	}{
		{name: "all tests", want: "Starting E2E tests on int staging:\n  tests: all\n"},
		{
			name: "selected tests and markers",
			flags: StartCmdFlags{CommonCmdFlags: CommonCmdFlags{Organization: organization.SWISSGEO},
				Tests: []string{"tests/api", "tests/test_login.py"}, Markers: []string{testAPIMarker, "slow"}},
			want: "Starting E2E tests on int staging:\n  tests: tests/api, tests/test_login.py\n  markers: api, slow\n",
		},
		{
			name: "unsupported markers",
			flags: StartCmdFlags{
				CommonCmdFlags: CommonCmdFlags{Organization: organization.GEOADMIN}, Markers: []string{testAPIMarker},
			},
			want:    "Starting E2E tests on int staging:\n  tests: all\n  markers: api\n",
			warning: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() { printStart("int", tt.flags) })
			if tt.warning {
				assert.Contains(t, output, "WARNING: flag --markers has no effect with --org geoadmin")
				assert.Contains(t, output, tt.want)
			} else {
				assert.Equal(t, tt.want, output)
			}
		})
	}
}

func TestInitPrint(t *testing.T) {
	original := noColor
	t.Cleanup(func() { noColor = original })
	command := newFlagCommand()
	for _, value := range []string{"true", "false"} {
		require.NoError(t, command.Flags().Set("no-color", value))
		require.NoError(t, initPrint(command))
		assert.Equal(t, value == "true", noColor)
	}
	require.ErrorContains(t, initPrint(&cobra.Command{}), "no-color")
}

func TestPrintTestResultMissingReport(t *testing.T) {
	for _, detailed := range []bool{false, true} {
		require.ErrorContains(t, printTestResult(codebuild.Build{}, detailed), "no test report found for build")
	}
}
