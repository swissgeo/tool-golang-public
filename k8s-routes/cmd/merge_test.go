package cmd_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/k8s-routes/cmd"
)

func TestDetectEnv(t *testing.T) {
	tests := []struct {
		kubeContext string
		want        string
	}{
		{"swissgeo/int-blue", "int"},
		{"swissgeo/prod-green", "prod"},
		{"dev", "dev"},
		{"my_dev_cluster", "dev"},
		{"arn:aws:eks:eu-central-1:123456789012:cluster/swissgeo-prod", "prod"},
	}
	for _, tt := range tests {
		t.Run(tt.kubeContext, func(t *testing.T) {
			env, err := cmd.DetectEnv(tt.kubeContext)

			require.NoError(t, err)
			assert.Equal(t, tt.want, env)
		})
	}
}

func TestDetectEnv_Error(t *testing.T) {
	for _, kubeContext := range []string{"", "minikube", "swissgeo/production", "interface", "devops/staging"} {
		t.Run(kubeContext, func(t *testing.T) {
			_, err := cmd.DetectEnv(kubeContext)

			assert.ErrorContains(t, err, "cannot determine the env")
		})
	}
}

// parseIngressRoute parses a single IngressRoute with the given namespace, name and routes match.
func parseIngressRoute(t *testing.T, namespace, name, match string) cmd.IngressRoute {
	t.Helper()
	manifest := `kind: IngressRoute
metadata:
  name: ` + name + `
  namespace: ` + namespace + `
spec:
  routes:
    - match: '` + match + `'
`
	irs, err := cmd.ParseManifests(strings.NewReader(manifest))
	require.NoError(t, err)
	require.Len(t, irs, 1)
	return irs[0]
}

func TestMerge(t *testing.T) {
	live := []cmd.IngressRoute{
		parseIngressRoute(t, "web", "unchanged", "Host(`web.example.com`)"),
		parseIngressRoute(t, "api", "changed", "Host(`api.example.com`)"),
		parseIngressRoute(t, "infra", "live-only", "PathPrefix(`/metrics`)"),
	}
	local := []cmd.IngressRoute{
		parseIngressRoute(t, "web", "unchanged", "Host(`web.example.com`)"),
		parseIngressRoute(t, "api", "changed", "Host(`api.example.com`) && PathPrefix(`/v2`)"),
		parseIngressRoute(t, "api", "new", "Host(`new.example.com`)"),
	}

	merged := cmd.Merge(live, local)

	type result struct {
		key    string
		marker cmd.Marker
		match  string
	}
	var got []result
	for _, ir := range merged {
		got = append(got, result{ir.Key(), ir.Marker, ir.Spec.Routes[0].Match})
	}
	assert.Equal(t, []result{
		{"api/changed", cmd.MarkerChanged, "Host(`api.example.com`) && PathPrefix(`/v2`)"},
		{"api/new", cmd.MarkerNew, "Host(`new.example.com`)"},
		{"infra/live-only", cmd.NoMarker, "PathPrefix(`/metrics`)"},
		{"web/unchanged", cmd.NoMarker, "Host(`web.example.com`)"},
	}, got)
}

func TestMerge_SameNameInOtherNamespace(t *testing.T) {
	live := []cmd.IngressRoute{parseIngressRoute(t, "a", "route", "Host(`a.example.com`)")}
	local := []cmd.IngressRoute{parseIngressRoute(t, "b", "route", "Host(`a.example.com`)")}

	merged := cmd.Merge(live, local)

	require.Len(t, merged, 2)
	assert.Equal(t, "a/route", merged[0].Key())
	assert.Equal(t, cmd.NoMarker, merged[0].Marker)
	assert.Equal(t, "b/route", merged[1].Key())
	assert.Equal(t, cmd.MarkerNew, merged[1].Marker)
}

func TestMerge_NoLocal(t *testing.T) {
	live := []cmd.IngressRoute{
		parseIngressRoute(t, "z", "last", "Host(`z.example.com`)"),
		parseIngressRoute(t, "a", "first", "Host(`a.example.com`)"),
	}

	merged := cmd.Merge(live, nil)

	require.Len(t, merged, 2)
	assert.Equal(t, "a/first", merged[0].Key())
	assert.Equal(t, "z/last", merged[1].Key())
	assert.Equal(t, cmd.NoMarker, merged[0].Marker)
}
