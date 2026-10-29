package cmd_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swissgeo/tool-golang-public/k8s-routes/cmd"
)

func TestHosts(t *testing.T) {
	tests := []struct {
		name  string
		match string
		want  []string
	}{
		{"single host", "Host(`single.example.com`)", []string{"single.example.com"}},
		{"with path", "Host(`path.example.com`) && PathPrefix(`/api`)", []string{"path.example.com"}},
		{"host regexp", "HostRegexp(`^.+\\.example\\.com$`)", []string{"^.+\\.example\\.com$"}},
		{
			"regexp with parentheses",
			"HostRegexp(`^(www|api)\\.example\\.com$`) && Path(`/`)",
			[]string{"^(www|api)\\.example\\.com$"},
		},
		{
			"several values (v2)",
			"Host(`v2-b.example.com`, `v2-a.example.com`)",
			[]string{"v2-a.example.com", "v2-b.example.com"},
		},
		{
			"several matchers",
			"(Host(`or-b.example.com`) || Host(`or-a.example.com`)) && PathPrefix(`/`)",
			[]string{"or-a.example.com", "or-b.example.com"},
		},
		{"duplicates", "Host(`dup.example.com`) || Host(`dup.example.com`)", []string{"dup.example.com"}},
		{"no host", "PathPrefix(`/api`)", []string{cmd.AnyHost}},
		{"empty", "", []string{cmd.AnyHost}},
		{"case mix", "Host(`upper.example.com`) || Host(`Upper.example.com`)", []string{"upper.example.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cmd.Hosts(tt.match))
		})
	}
}

func TestEffectivePriority(t *testing.T) {
	prio, explicit := cmd.Route{Match: "Host(`a`)", Priority: 42}.EffectivePriority()
	assert.Equal(t, 42, prio)
	assert.True(t, explicit)

	prio, explicit = cmd.Route{Match: "Host(`a`)"}.EffectivePriority()
	assert.Equal(t, len("Host(`a`)"), prio)
	assert.False(t, explicit)
}

func ingressRoute(name string, routes ...cmd.Route) cmd.IngressRoute {
	return cmd.IngressRoute{
		Kind:     "IngressRoute",
		Metadata: cmd.Metadata{Name: name, Namespace: "ns"},
		Spec:     cmd.IngressRouteSpec{Routes: routes},
	}
}

func TestBuildRows(t *testing.T) {
	frontendMatch := "Host(`www.example.org`) || Host(`example.org`)"
	backendMatch := "Host(`example.org`) && PathPrefix(`/backend`)"
	healthMatch := "PathPrefix(`/health`)"
	irs := []cmd.IngressRoute{
		ingressRoute("frontend",
			cmd.Route{
				Match:       frontendMatch,
				Services:    []cmd.Ref{{Name: "frontend"}, {Name: "frontend-canary"}},
				Middlewares: []cmd.Ref{{Name: "strip-prefix", Namespace: "ingress"}},
			},
		),
		ingressRoute("backend",
			cmd.Route{Match: backendMatch, Priority: 1000},
			cmd.Route{Match: healthMatch},
		),
	}

	rows := cmd.BuildRows(irs, cmd.IngressRoute.Key)

	frontend := cmd.Row{
		Name: "ns/frontend", Priority: len(frontendMatch), Match: frontendMatch,
		Services: []string{"frontend", "frontend-canary"}, Middlewares: []string{"strip-prefix"},
	}
	frontendApex, frontendWWW := frontend, frontend
	frontendApex.Host = "example.org"
	frontendWWW.Host = "www.example.org"
	assert.Equal(t, []cmd.Row{
		{
			Host: cmd.AnyHost, Name: "ns/backend", Priority: len(healthMatch),
			Match: healthMatch, Services: []string{}, Middlewares: []string{},
		},
		{
			Host: "example.org", Name: "ns/backend", Priority: 1000, ExplicitPriority: true,
			Match: backendMatch, Services: []string{}, Middlewares: []string{},
		},
		frontendApex,
		frontendWWW,
	}, rows)
}

func TestBuildRows_SortWithinHost(t *testing.T) {
	irs := []cmd.IngressRoute{
		ingressRoute("c", cmd.Route{Match: "Host(`h`) && Path(`/b`)", Priority: 10}),
		ingressRoute("b", cmd.Route{Match: "Host(`h`) && Path(`/a`)", Priority: 10}),
		ingressRoute("a", cmd.Route{Match: "Host(`h`)", Priority: 5}),
		ingressRoute("d", cmd.Route{Match: "Host(`h`) && Path(`/c`)", Priority: 20}),
	}

	rows := cmd.BuildRows(irs, func(ir cmd.IngressRoute) string { return ir.Metadata.Name })

	var names []string
	for _, r := range rows {
		names = append(names, r.Name)
	}
	// priority desc, then match asc
	assert.Equal(t, []string{"d", "b", "c", "a"}, names)
}

func TestBuildRows_Empty(t *testing.T) {
	assert.Empty(t, cmd.BuildRows(nil, cmd.IngressRoute.Key))
}

func TestFilterHost(t *testing.T) {
	anyHost := cmd.Row{Host: cmd.AnyHost}
	api := cmd.Row{Host: "api.example.com"}
	www := cmd.Row{Host: "www.example.com"}
	other := cmd.Row{Host: "www.other.org"}
	rows := []cmd.Row{anyHost, api, www, other}
	tests := []struct {
		name   string
		substr string
		want   []cmd.Row
	}{
		{"empty returns all", "", rows},
		{"substring keeps any host", "example", []cmd.Row{anyHost, api, www}},
		{"case-insensitive", "WWW.Example", []cmd.Row{anyHost, www}},
		{"no match keeps any host", "nope", []cmd.Row{anyHost}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cmd.FilterHost(rows, tt.substr))
		})
	}
	// the input is not modified
	assert.Equal(t, []cmd.Row{anyHost, api, www, other}, rows)
}
