package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/k8s-validate/cmd"
)

func writeLinterConfig(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0755))
	content := []byte("checks:\n  doNotAutoAddDefaults: true\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".kube-linter.yaml"), content, 0600))
}

func TestFindLinterConfig_SameFolder(t *testing.T) {
	tmp := t.TempDir()
	writeLinterConfig(t, filepath.Join(tmp, "overlays", "prod"))
	t.Chdir(tmp)

	got, err := cmd.FindLinterConfig(filepath.Join("overlays", "prod"))

	require.NoError(t, err)
	assert.Equal(t, filepath.Join("overlays", "prod", ".kube-linter.yaml"), got)
}

func TestFindLinterConfig_ParentFolder(t *testing.T) {
	tmp := t.TempDir()
	writeLinterConfig(t, filepath.Join(tmp, "overlays"))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "overlays", "prod"), 0755))
	t.Chdir(tmp)

	got, err := cmd.FindLinterConfig(filepath.Join("overlays", "prod"))

	require.NoError(t, err)
	assert.Equal(t, filepath.Join("overlays", ".kube-linter.yaml"), got)
}

func TestFindLinterConfig_WorkingDirectory(t *testing.T) {
	tmp := t.TempDir()
	writeLinterConfig(t, tmp)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "overlays", "prod"), 0755))
	t.Chdir(tmp)

	got, err := cmd.FindLinterConfig(filepath.Join("overlays", "prod"))

	require.NoError(t, err)
	assert.Equal(t, ".kube-linter.yaml", got)
}

func TestFindLinterConfig_NotFound(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "overlays", "prod"), 0755))
	t.Chdir(tmp)

	_, err := cmd.FindLinterConfig(filepath.Join("overlays", "prod"))

	assert.Error(t, err)
}

func TestFindLinterConfig_DoesNotEscapeWorkingDirectory(t *testing.T) {
	// Config placed in the parent of the working directory must not be found:
	// the search must stop at cwd (".").
	// Structure: tmp/parent/.kube-linter.yaml, cwd=tmp/parent/cwd, folder=overlays/prod
	tmp := t.TempDir()
	cwd := filepath.Join(tmp, "parent", "cwd")
	writeLinterConfig(t, filepath.Join(tmp, "parent"))
	require.NoError(t, os.MkdirAll(filepath.Join(cwd, "overlays", "prod"), 0755))
	t.Chdir(cwd)

	_, err := cmd.FindLinterConfig(filepath.Join("overlays", "prod"))

	assert.Error(t, err)
}
