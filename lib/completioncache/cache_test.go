package completioncache_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/geoadmin/tool-golang-bgdi/lib/completioncache"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	cmdName := "test-completion-cache"

	cache, err := completioncache.NewCache(cmdName, time.Hour)
	require.NoError(t, err)

	// Test missing key is not found
	completions, dir, err := cache.Read("subCmd-missing")
	assert.Nil(t, completions)
	assert.Equal(t, cobra.ShellCompDirectiveError, dir)
	require.ErrorIs(t, err, completioncache.ErrNotFound)

	err = cache.Write("subCmd-FlagA", []cobra.Completion{"val-a", "val-b"}, cobra.ShellCompDirectiveDefault)
	require.NoError(t, err)

	completions, dir, err = cache.Read("subCmd-FlagA")
	require.NoError(t, err)
	assert.Equal(t, []cobra.Completion{"val-a", "val-b"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveDefault, dir)

	// Check with new Cache that previous value is still found.
	cache, err = completioncache.NewCache(cmdName, time.Hour)
	require.NoError(t, err)
	completions, dir, err = cache.Read("subCmd-FlagA")
	require.NoError(t, err)
	assert.Equal(t, []cobra.Completion{"val-a", "val-b"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveDefault, dir)

	// clean up cache directory
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	filePath, err := filepath.Abs(home + "/" + cmdName + "/")
	require.NoError(t, err)
	err = os.RemoveAll(filePath)
	require.NoError(t, err)
}
