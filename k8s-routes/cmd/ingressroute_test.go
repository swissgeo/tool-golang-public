package cmd_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/k8s-routes/cmd"
)

// As returned by `kubectl get -o yaml`
const liveYAML = `apiVersion: v1
kind: List
items:
- apiVersion: traefik.io/v1alpha1
  kind: IngressRoute
  metadata:
    name: api
    namespace: service-api
    resourceVersion: "42"
  spec:
    entryPoints:
    - websecure
    routes:
    - match: Host(` + "`api.example.com`" + `) && PathPrefix(` + "`/v1`" + `)
      priority: 100
      services:
      - name: api
        port: 80
      middlewares:
      - name: cors
      - name: auth
        namespace: traefik
    - match: Host(` + "`api.example.com`" + `)
      services:
      - name: api
        port: 80
- apiVersion: v1
  kind: Service
  metadata:
    name: api
    namespace: service-api
`

// As returned by `kustomize build`
const localYAML = `---
apiVersion: v1
kind: Service
metadata:
  name: api
---
# empty document below
---
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: api
spec:
  entryPoints:
    - websecure
  routes:
    - match: Host(` + "`api.example.com`" + `) && PathPrefix(` + "`/v1`" + `)
      priority: 100
      services:
        - name: api
          port: 80
      middlewares:
        - name: cors
        - name: auth
          namespace: traefik
    - match: Host(` + "`api.example.com`" + `)
      services:
        - name: api
          port: 80
`

func TestParseList(t *testing.T) {
	routes, err := cmd.ParseList(strings.NewReader(liveYAML))

	require.NoError(t, err)
	require.Len(t, routes, 1)
	ir := routes[0]
	assert.Equal(t, "service-api/api", ir.Key())
	require.Len(t, ir.Spec.Routes, 2)
	assert.Equal(t, cmd.Route{
		Match:       "Host(`api.example.com`) && PathPrefix(`/v1`)",
		Priority:    100,
		Services:    []cmd.Ref{{Name: "api"}},
		Middlewares: []cmd.Ref{{Name: "cors"}, {Name: "auth", Namespace: "traefik"}},
	}, ir.Spec.Routes[0])
	assert.Equal(t, 0, ir.Spec.Routes[1].Priority)
	assert.Empty(t, ir.Spec.Routes[1].Middlewares)
}

func TestParseList_Empty(t *testing.T) {
	for name, input := range map[string]string{
		"no items":  "apiVersion: v1\nkind: List\nitems: []\n",
		"no output": "",
	} {
		t.Run(name, func(t *testing.T) {
			routes, err := cmd.ParseList(strings.NewReader(input))

			require.NoError(t, err)
			assert.Empty(t, routes)
		})
	}
}

func TestParseList_Invalid(t *testing.T) {
	_, err := cmd.ParseList(strings.NewReader("items: [unclosed"))

	assert.Error(t, err)
}

func TestParseManifests(t *testing.T) {
	routes, err := cmd.ParseManifests(strings.NewReader(localYAML))

	require.NoError(t, err)
	require.Len(t, routes, 1)
	ir := routes[0]
	assert.Equal(t, "api", ir.Metadata.Name)
	assert.Empty(t, ir.Metadata.Namespace)
	require.Len(t, ir.Spec.Routes, 2)
	assert.Equal(t, 100, ir.Spec.Routes[0].Priority)
	assert.Equal(t, []cmd.Ref{{Name: "cors"}, {Name: "auth", Namespace: "traefik"}}, ir.Spec.Routes[0].Middlewares)
}

func TestParseManifests_Empty(t *testing.T) {
	routes, err := cmd.ParseManifests(strings.NewReader(""))

	require.NoError(t, err)
	assert.Empty(t, routes)
}

func TestParseManifests_Invalid(t *testing.T) {
	_, err := cmd.ParseManifests(strings.NewReader("kind: [unclosed"))

	assert.Error(t, err)
}

func TestSameSpec_LiveAndLocal(t *testing.T) {
	live, err := cmd.ParseList(strings.NewReader(liveYAML))
	require.NoError(t, err)
	local, err := cmd.ParseManifests(strings.NewReader(localYAML))
	require.NoError(t, err)

	// metadata (namespace, resourceVersion) and indentation differ but the spec is the same
	assert.True(t, local[0].SameSpec(live[0]))
}

func TestSameSpec_UnmodelledFieldDiffers(t *testing.T) {
	live, err := cmd.ParseList(strings.NewReader(liveYAML))
	require.NoError(t, err)
	local, err := cmd.ParseManifests(strings.NewReader(strings.Replace(localYAML, "websecure", "web", 1)))
	require.NoError(t, err)

	assert.False(t, local[0].SameSpec(live[0]))
}
