package cmd_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/k8s-routes/cmd"
)

func mkdirs(t *testing.T, root string, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(root, filepath.FromSlash(d)), 0755))
	}
}

func TestFindOverlays(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, "services/b/overlays/dev", "services/a/overlays/dev", "services/c/overlays/int")
	// a file matching the glob is ignored
	mkdirs(t, root, "services/d/overlays")
	require.NoError(t, os.WriteFile(filepath.Join(root, "services/d/overlays/dev"), nil, 0600))

	overlays, err := cmd.FindOverlays(root, "services/*/overlays/{env}", "dev")

	require.NoError(t, err)
	assert.Equal(t, []cmd.Overlay{
		{Dir: filepath.Join(root, "services/a/overlays/dev"), Namespace: "a"},
		{Dir: filepath.Join(root, "services/b/overlays/dev"), Namespace: "b"},
	}, overlays)
}

func TestFindOverlays_NamespaceSegment(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, "apps/svc-api/dev", "static/dev")
	tests := []struct {
		name string
		glob string
		want []cmd.Overlay
	}{
		{
			"whole segment", "apps/svc-*/{env}",
			[]cmd.Overlay{{Dir: filepath.Join(root, "apps/svc-api/dev"), Namespace: "svc-api"}},
		},
		{"no wildcard", "static/{env}", []cmd.Overlay{{Dir: filepath.Join(root, "static/dev")}}},
		{
			"leading dot", "./apps/svc-*/{env}",
			[]cmd.Overlay{{Dir: filepath.Join(root, "apps/svc-api/dev"), Namespace: "svc-api"}},
		},
		{
			"double slash", "apps//svc-*/{env}",
			[]cmd.Overlay{{Dir: filepath.Join(root, "apps/svc-api/dev"), Namespace: "svc-api"}},
		},
		{
			"parent dir", "static/../apps/svc-*/{env}",
			[]cmd.Overlay{{Dir: filepath.Join(root, "apps/svc-api/dev"), Namespace: "svc-api"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlays, err := cmd.FindOverlays(root, tt.glob, "dev")

			require.NoError(t, err)
			assert.Equal(t, tt.want, overlays)
		})
	}
}

func TestFindOverlays_NoMatch(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, "services/a/overlays/dev")

	_, err := cmd.FindOverlays(root, "services/*/overlays/{env}", "prod")

	require.Error(t, err)
	assert.Contains(t, err.Error(), `no kustomize overlays found for env "prod"`)
}

func TestFindOverlays_InvalidGlob(t *testing.T) {
	_, err := cmd.FindOverlays(t.TempDir(), "services/[/{env}", "dev")

	assert.Error(t, err)
}

const fakeManifests = `apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: web
spec:
  routes:
    - match: Host(` + "`web.example.com`" + `)
---
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: shared
  namespace: explicit
spec:
  routes:
    - match: PathPrefix(` + "`/`" + `)
`

func TestRenderOverlays(t *testing.T) {
	overlays := []cmd.Overlay{
		{Dir: "dir-a", Namespace: "a"},
		{Dir: "dir-b", Namespace: "b"},
		{Dir: "dir-c", Namespace: "c"},
	}
	var calls atomic.Int32
	build := func(_ context.Context, dir string) ([]byte, error) {
		calls.Add(1)
		if dir == "dir-b" {
			return []byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: no-routes\n"), nil
		}
		return []byte(fakeManifests), nil
	}

	var warn bytes.Buffer
	irs, err := cmd.RenderOverlays(context.Background(), overlays, 2, build, &warn)

	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
	var keys []string
	for _, ir := range irs {
		keys = append(keys, ir.Key())
	}
	// in the order of the overlays, the explicit namespace is kept
	assert.Equal(t, []string{"a/web", "explicit/shared", "c/web", "explicit/shared"}, keys)
	// the explicit namespace makes both overlays render the same IngressRoute
	assert.Equal(t, "Warning: IngressRoute explicit/shared is rendered by several overlays: dir-a, dir-c\n", warn.String())
}

func TestRenderOverlays_NoDuplicate(t *testing.T) {
	build := func(_ context.Context, _ string) ([]byte, error) {
		return []byte(fakeManifests), nil
	}

	var warn bytes.Buffer
	_, err := cmd.RenderOverlays(context.Background(), []cmd.Overlay{{Dir: "dir-a", Namespace: "a"}}, 1, build, &warn)

	require.NoError(t, err)
	assert.Empty(t, warn.String())
}

func TestRenderOverlays_BuildError(t *testing.T) {
	overlays := []cmd.Overlay{{Dir: "ok"}, {Dir: "broken"}}
	build := func(_ context.Context, dir string) ([]byte, error) {
		if dir == "broken" {
			return nil, errors.New("kustomize build broken failed")
		}
		return []byte(fakeManifests), nil
	}

	_, err := cmd.RenderOverlays(context.Background(), overlays, 1, build, io.Discard)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken")
}

func TestRenderOverlays_ParseError(t *testing.T) {
	build := func(_ context.Context, _ string) ([]byte, error) {
		return []byte("kind: [unclosed"), nil
	}

	_, err := cmd.RenderOverlays(context.Background(), []cmd.Overlay{{Dir: "invalid"}}, 1, build, io.Discard)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse the output of invalid")
}

// fakeKustomize puts a fake kustomize in PATH that prints its arguments, or fails if the
// directory to build is named "broken".
func fakeKustomize(t *testing.T) {
	t.Helper()
	fakeBin(t, "kustomize", `for last in "$@"; do :; done
if [ "$last" = broken ]; then echo "Error: boom" >&2; exit 1; fi
echo "# $*"
`)
}

func TestKustomizeBuild(t *testing.T) {
	fakeKustomize(t)
	tests := []struct {
		name       string
		enableHelm bool
		want       string
	}{
		{"default", false, "# build some/dir\n"},
		{"enable helm", true, "# build --enable-helm some/dir\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := cmd.KustomizeBuild(tt.enableHelm)(context.Background(), "some/dir")

			require.NoError(t, err)
			assert.Equal(t, tt.want, string(out))
		})
	}
}

func TestKustomizeBuild_Error(t *testing.T) {
	fakeKustomize(t)

	_, err := cmd.KustomizeBuild(false)(context.Background(), "broken")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "kustomize build broken failed")
	assert.True(t, strings.HasSuffix(err.Error(), "Error: boom"), "stderr is part of the error: %q", err)
}

func TestKustomizeBuild_NotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := cmd.KustomizeBuild(false)(context.Background(), "some/dir")

	require.EqualError(t, err, "kustomize not found in PATH")
}
