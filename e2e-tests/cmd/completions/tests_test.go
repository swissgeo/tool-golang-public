package completions //nolint:testpackage // Test filesystem helpers without cloning remote repositories.

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/e2e-tests/cmd/organization"
)

func TestShouldFetch(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		timestamp            []byte
		wantFetch, wantError bool
	}{
		{name: "missing timestamp", timestamp: nil, wantFetch: true, wantError: false},
		{
			name: "recent fetch", timestamp: []byte(time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)),
			wantFetch: false, wantError: false,
		},
		{
			name: "expired fetch", timestamp: []byte(time.Now().Add(-25 * time.Hour).UTC().Format(time.RFC3339)),
			wantFetch: true, wantError: false,
		},
		{name: "invalid timestamp", timestamp: []byte("invalid"), wantFetch: true, wantError: true},
		{name: "empty timestamp", timestamp: []byte{}, wantFetch: true, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0755))
			if tt.timestamp != nil {
				require.NoError(t, os.WriteFile(filepath.Join(repo, lastFetchFile), tt.timestamp, 0600))
			}
			fetch, err := shouldFetch(repo)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantFetch, fetch)
		})
	}
}

func TestFetchTimestampDirReadError(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, lastFetchFile), 0755))
	fetch, err := shouldFetch(repo)
	require.Error(t, err)
	assert.False(t, fetch)
}

func TestSetLastFetchTime(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0755))
	require.NoError(t, setLastFetchTime(repo))
	data, err := os.ReadFile(filepath.Join(repo, lastFetchFile))
	require.NoError(t, err)
	timestamp, err := time.Parse(time.RFC3339, string(data))
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), timestamp, 5*time.Second)
	fetch, err := shouldFetch(repo)
	require.NoError(t, err)
	assert.False(t, fetch)
	require.Error(t, setLastFetchTime(t.TempDir()))
}

func TestPathToPythonModule(t *testing.T) {
	for _, tt := range []struct{ path, want string }{
		{"", ""},
		{"test_api.py", "test_api"},
		{filepath.Join("api", "test_search.py"), "api.test_search"},
		{filepath.Join("api", "nested"), "api.nested"},
		{filepath.Join("api", "nested", "module.py"), "api.nested.module"},
		{"test_api.py.bak", "test_api.py.bak"},
	} {
		assert.Equal(t, tt.want, pathToPythonModule(tt.path))
	}
}

func TestFindTests(t *testing.T) {
	repo := t.TempDir()
	for _, name := range []string{
		"tests/test_root.py", "tests/api/test_search.py", "tests/api/helper.py",
		"tests/api/test_search.py.bak", "tests/api/__init__.py",
		"tests/__pycache__/test_hidden.py", "tests/lib/test_hidden.py", "tests/fixtures/test_hidden.py",
		"tests/api/fixtures/test_hidden.py", "outside/test_ignored.py",
	} {
		path := filepath.Join(repo, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, nil, 0600))
	}
	require.NoError(t, os.Mkdir(filepath.Join(repo, "tests", "empty"), 0755))
	for _, tt := range []struct {
		org  organization.Organization
		want []string
	}{
		{organization.GEOADMIN, []string{"api", "api.test_search", "empty", "test_root"}},
		{organization.SWISSGEO, []string{"api", filepath.Join("api", "test_search.py"), "empty", "test_root.py"}},
	} {
		t.Run(string(tt.org), func(t *testing.T) {
			names, err := findTests(repo, tt.org)
			require.NoError(t, err)
			assert.Equal(t, tt.want, names)
		})
	}
}

func TestFindTestsMissingDirectory(t *testing.T) {
	names, err := findTests(t.TempDir(), organization.GEOADMIN)
	require.Error(t, err)
	assert.Nil(t, names)
}

func TestCompleteTestsMissingOrg(t *testing.T) {
	names, directive := CompleteTests(&cobra.Command{}, nil, "")
	assert.Nil(t, names)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}
