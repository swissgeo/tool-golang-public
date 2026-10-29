package cmd_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeBin prepends to PATH a directory containing an executable shell script with the given name
// and body, so that it is used instead of the real binary.
func fakeBin(t *testing.T, name, body string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake binaries are shell scripts")
	}
	bin := t.TempDir()
	path := filepath.Join(bin, name)
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0700)) //nolint:gosec // must be executable
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}
