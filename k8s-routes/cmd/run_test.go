package cmd_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/k8s-routes/cmd"
)

// execute runs the k8s-routes command with the given arguments and returns its stdout and stderr.
func execute(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	t.Cleanup(cmd.ResetFlags)
	var stdout, stderr bytes.Buffer
	cmd.RootCmd.SetOut(&stdout)
	cmd.RootCmd.SetErr(&stderr)
	cmd.RootCmd.SetArgs(args)
	err := cmd.RootCmd.ExecuteContext(context.Background())
	return stdout.String(), stderr.String(), err
}

// localOverlays creates a kustomize project in a temporary directory with the given manifests per
// overlay directory and puts a fake kustomize in PATH that prints them.
func localOverlays(t *testing.T, manifests map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for dir, manifest := range manifests {
		mkdirs(t, root, dir)
		require.NoError(t, os.WriteFile(filepath.Join(root, dir, "manifest.yaml"), []byte(manifest), 0600))
	}
	fakeBin(t, "kustomize", `for last in "$@"; do :; done
cat "$last/manifest.yaml"
`)
	return root
}

const localShop = `kind: IngressRoute
metadata:
  name: shop
spec:
  routes:
    - match: Host(` + "`www.example.net`" + `) || Host(` + "`example.net`" + `)
      services: [{name: shop}, {name: shop-canary}]
    - match: PathPrefix(` + "`/status`" + `)
      services: [{name: status}]
`

// Changes the second route of service-api/api of liveYAML and adds a new IngressRoute
const localAPI = `kind: IngressRoute
metadata:
  name: api
spec:
  entryPoints: [websecure]
  routes:
    - match: Host(` + "`api.example.com`" + `) && PathPrefix(` + "`/v1`" + `)
      priority: 100
      services: [{name: api, port: 80}]
      middlewares: [{name: cors}, {name: auth, namespace: traefik}]
    - match: Host(` + "`api.example.com`" + `) && PathPrefix(` + "`/v2`" + `)
      services: [{name: api-v2, port: 80}]
---
kind: IngressRoute
metadata:
  name: beta
spec:
  routes:
    - match: Host(` + "`beta.example.com`" + `)
      services: [{name: beta}]
`

func TestLocalCommand(t *testing.T) {
	root := localOverlays(t, map[string]string{
		"services/shop/overlays/dev": localShop,
		"services/shop/overlays/int": "kind: Service\n",
	})

	stdout, _, err := execute(t, "local", "dev", "--root", root)

	require.NoError(t, err)
	assertGolden(t, "local", []byte(stdout))
}

func TestLocalCommand_HostFilter(t *testing.T) {
	root := localOverlays(t, map[string]string{"services/shop/overlays/dev": localShop})

	stdout, _, err := execute(t, "local", "dev", "--root", root, "--host", "www")

	require.NoError(t, err)
	assert.Contains(t, stdout, "PathPrefix(`/status`)", "any host routes are kept")
	assert.Contains(t, stdout, "Host(`www.example.net`)")
	// the example.net group is filtered out, so the shop route is only listed once
	assert.Equal(t, 1, strings.Count(stdout, "shop,shop-canary"))
}

func TestLocalCommand_NoOverlay(t *testing.T) {
	root := localOverlays(t, map[string]string{"services/shop/overlays/dev": localShop})

	_, _, err := execute(t, "local", "prod", "--root", root)

	assert.ErrorContains(t, err, `no kustomize overlays found for env "prod"`)
}

func TestLiveCommand(t *testing.T) {
	fakeKubectl(t, "swissgeo/int-blue")

	stdout, stderr, err := execute(t, "live")

	require.NoError(t, err)
	assert.Equal(t, "Context: swissgeo/int-blue\n", stderr)
	assertGolden(t, "live", []byte(stdout))
}

func TestLiveCommand_WithLocal(t *testing.T) {
	fakeKubectl(t, "")
	root := localOverlays(t, map[string]string{"services/service-api/overlays/int": localAPI})

	stdout, stderr, err := execute(t, "live", "--context", "swissgeo/int-blue", "--local", "--root", root)

	require.NoError(t, err)
	assert.Equal(t, "Context: swissgeo/int-blue (with local changes of \"int\")\n", stderr)
	assertGolden(t, "live_local", []byte(stdout))
}

func TestLiveCommand_WithLocalUnknownEnv(t *testing.T) {
	fakeKubectl(t, "minikube")

	_, _, err := execute(t, "live", "--local")

	assert.ErrorContains(t, err, `cannot determine the env (dev|int|prod) from the kube context "minikube"`)
}
