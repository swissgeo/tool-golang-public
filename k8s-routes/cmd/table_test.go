package cmd_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/k8s-routes/cmd"
)

var update = flag.Bool("update", false, "update the golden files in testdata/")

// assertGolden compares got with the content of testdata/<name>.golden, or writes it with -update.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		require.NoError(t, os.MkdirAll("testdata", 0755))
		require.NoError(t, os.WriteFile(path, got, 0600))
	}
	want, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}

var ansiRe = regexp.MustCompile("\033\\[[0-9;]*m")

func tableRows() []cmd.Row {
	shop := cmd.Row{
		Name: "ns/shop", Priority: 46,
		Match: "Host(`www.example.net`) || Host(`example.net`)", Services: []string{"shop", "shop-canary"},
	}
	shopApex, shopWWW := shop, shop
	shopApex.Host = "example.net"
	shopWWW.Host = "www.example.net"
	return []cmd.Row{
		{
			Host: cmd.AnyHost, Name: "ns/status", Priority: 21,
			Match: "PathPrefix(`/status`)", Services: []string{"status"},
		},
		{
			Host: "example.net", Name: "ns/api", Marker: cmd.MarkerChanged, Priority: 1000, ExplicitPriority: true,
			Match: "Host(`example.net`) && PathPrefix(`/api`)", Services: []string{"api"},
			Middlewares: []string{"rate-limit", "headers"},
		},
		shopApex,
		shopWWW,
		{
			Host: "www.example.net", Name: "ns/beta", Marker: cmd.MarkerNew, Priority: 5, ExplicitPriority: true,
			Match: "Host(`www.example.net`) && Path(`/beta`)", Services: []string{"beta"},
		},
	}
}

func TestRenderTable(t *testing.T) {
	var buf bytes.Buffer

	require.NoError(t, cmd.RenderTable(&buf, tableRows(), cmd.TableOptions{}))

	assertGolden(t, "table", buf.Bytes())
}

func TestRenderTable_SplitMatch(t *testing.T) {
	rows := append(tableRows(), cmd.Row{
		Host: "www.example.net", Name: "ns/v2", Priority: 3, ExplicitPriority: true,
		Match:    "Host(`www.example.net`) && (PathPrefix(`/v2`) || PathPrefix(`/v3`)) && Headers(`X-Api`, `a&&b`)",
		Services: []string{"v2"},
	})
	var buf bytes.Buffer

	require.NoError(t, cmd.RenderTable(&buf, rows, cmd.TableOptions{SplitMatch: true}))

	assertGolden(t, "table_split", buf.Bytes())
}

func TestSplitAnd(t *testing.T) {
	tests := []struct {
		match string
		want  []string
	}{
		{"PathPrefix(`/status`)", []string{"PathPrefix(`/status`)"}},
		{"Host(`a`) || Host(`b`)", []string{"Host(`a`) || Host(`b`)"}},
		{"Host(`a`) && PathPrefix(`/api`)", []string{"Host(`a`)", "PathPrefix(`/api`)"}},
		{"Host(`a`)&&Path(`/x`)&&Method(`GET`)", []string{"Host(`a`)", "Path(`/x`)", "Method(`GET`)"}},
		{"Host(`a`) && (Path(`/x`) && Method(`GET`) || Path(`/y`))",
			[]string{"Host(`a`)", "(Path(`/x`) && Method(`GET`) || Path(`/y`))"}},
		{"HostRegexp(`^a(&&)?$`) && Path(`/x`)", []string{"HostRegexp(`^a(&&)?$`)", "Path(`/x`)"}},
		{`Header("X", "a\" && b") && Path("/x")`, []string{`Header("X", "a\" && b")`, `Path("/x")`}},
	}
	for _, tt := range tests {
		t.Run(tt.match, func(t *testing.T) {
			assert.Equal(t, tt.want, cmd.SplitAnd(tt.match))
		})
	}
}

func TestRenderTable_Empty(t *testing.T) {
	var buf bytes.Buffer

	require.NoError(t, cmd.RenderTable(&buf, nil, cmd.TableOptions{SplitMatch: true}))

	assert.Equal(t, "MATCH  PRIORITY  SERVICES  MIDDLEWARES  NAME\n", buf.String())
}

func TestRenderTable_Color(t *testing.T) {
	var plain, colored bytes.Buffer
	require.NoError(t, cmd.RenderTable(&plain, tableRows(), cmd.TableOptions{SplitMatch: true}))

	require.NoError(t, cmd.RenderTable(&colored, tableRows(), cmd.TableOptions{Color: true, SplitMatch: true}))

	out := colored.String()
	assert.Contains(t, out, "\033[34m(21)\033[0m", "computed priority")
	assert.Contains(t, out, "ns/api \033[33m(*)\033[0m", "changed marker")
	assert.Contains(t, out, "ns/beta \033[32m(+)\033[0m", "new marker")
	assert.NotContains(t, out, "\033[34m1000", "explicit priority is not colored")
	// the colors don't change the alignment
	assert.Equal(t, plain.String(), ansiRe.ReplaceAllString(out, ""))
}
