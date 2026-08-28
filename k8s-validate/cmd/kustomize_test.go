package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/k8s-validate/cmd"
)

func writeKustomizationYAML(t *testing.T, dir, kind string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0755))
	content := "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: " + kind + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(content), 0600))
}

func TestFindKustomizeFolders_Empty(t *testing.T) {
	t.Chdir(t.TempDir())

	folders, err := cmd.FindKustomizeFolders()

	require.NoError(t, err)
	assert.Empty(t, folders)
}

func TestFindKustomizeFolders_Single(t *testing.T) {
	tmp := t.TempDir()
	writeKustomizationYAML(t, filepath.Join(tmp, "overlays", "prod"), "Kustomization")
	t.Chdir(tmp)

	folders, err := cmd.FindKustomizeFolders()

	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join("overlays", "prod")}, folders)
}

func TestFindKustomizeFolders_SortedAndFiltered(t *testing.T) {
	tmp := t.TempDir()
	writeKustomizationYAML(t, filepath.Join(tmp, "z-last"), "Kustomization")
	writeKustomizationYAML(t, filepath.Join(tmp, "a-first"), "Kustomization")
	writeKustomizationYAML(t, filepath.Join(tmp, "middle"), "Component") // excluded: wrong kind
	t.Chdir(tmp)

	folders, err := cmd.FindKustomizeFolders()

	require.NoError(t, err)
	assert.Equal(t, []string{"a-first", "z-last"}, folders)
}

func TestFindKustomizeFolders_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "kustomization.yaml"), []byte("key: [unclosed"), 0600))
	t.Chdir(tmp)

	_, err := cmd.FindKustomizeFolders()

	assert.Error(t, err)
}
