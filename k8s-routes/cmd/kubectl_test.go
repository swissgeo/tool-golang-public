package cmd_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/k8s-routes/cmd"
)

// fakeKubectl puts a fake kubectl in PATH that only answers the expected arguments, with
// currentContext as current context and liveYAML as IngressRoutes of the context "swissgeo/int-blue".
func fakeKubectl(t *testing.T, currentContext string) {
	t.Helper()
	fakeBin(t, "kubectl", `case "$*" in
"config current-context") echo "`+currentContext+`" ;;
"--context swissgeo/int-blue get ingressroutes.traefik.io --all-namespaces -o yaml")
	cat <<'EOF'
`+liveYAML+`EOF
	;;
"--context invalid-output "*) echo "items: [unclosed" ;;
*) echo "error: unexpected arguments: $*" >&2; exit 1 ;;
esac
`)
}

func TestCurrentContext(t *testing.T) {
	fakeKubectl(t, "swissgeo/int-blue")

	kubeContext, err := cmd.CurrentContext(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "swissgeo/int-blue", kubeContext)
}

func TestCurrentContext_Empty(t *testing.T) {
	fakeKubectl(t, "")

	_, err := cmd.CurrentContext(context.Background())

	require.EqualError(t, err, "no current kubectl context, use --context")
}

func TestCurrentContext_NotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := cmd.CurrentContext(context.Background())

	require.EqualError(t, err, "kubectl not found in PATH")
}

func TestLiveIngressRoutes(t *testing.T) {
	fakeKubectl(t, "")

	irs, err := cmd.LiveIngressRoutes(context.Background(), "swissgeo/int-blue")

	require.NoError(t, err)
	require.Len(t, irs, 1)
	assert.Equal(t, "service-api/api", irs[0].Key())
}

func TestLiveIngressRoutes_Error(t *testing.T) {
	fakeKubectl(t, "")

	_, err := cmd.LiveIngressRoutes(context.Background(), "unknown")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "kubectl --context unknown get ingressroutes.traefik.io")
	assert.Contains(t, err.Error(), "error: unexpected arguments", "stderr is part of the error")
}

func TestLiveIngressRoutes_InvalidOutput(t *testing.T) {
	fakeKubectl(t, "")

	_, err := cmd.LiveIngressRoutes(context.Background(), "invalid-output")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse the IngressRoutes of context invalid-output")
}
