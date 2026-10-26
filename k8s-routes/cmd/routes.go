package cmd

import (
	"cmp"
	"regexp"
	"slices"
	"strings"
)

// AnyHost is the host of the routes without Host matcher, they compete with the routes of all hosts.
const AnyHost = "(any host)"

var (
	// Host(`a`) or HostRegexp(`a`), also with several values (Traefik v2): Host(`a`, `b`). The values
	// are matched as backtick quoted strings, so that parentheses in a regexp don't end the matcher.
	hostMatcherRe = regexp.MustCompile("Host(?:Regexp)?\\(((?:\\s*`[^`]*`\\s*,?)+)\\s*\\)")
	backtickRe    = regexp.MustCompile("`([^`]*)`")
)

// Row is a route of an IngressRoute for a single host.
type Row struct {
	Host string
	// Name of the IngressRoute as displayed
	Name     string
	Marker   Marker
	Priority int
	// ExplicitPriority is false when the priority is computed by Traefik from the match rule.
	ExplicitPriority bool
	Match            string
	Services         []string
	Middlewares      []string
}

// Hosts returns the sorted unique hosts of the Host and HostRegexp matchers of a match rule, or
// AnyHost if there are none.
func Hosts(match string) []string {
	var hosts []string
	for _, m := range hostMatcherRe.FindAllStringSubmatch(match, -1) {
		for _, v := range backtickRe.FindAllStringSubmatch(m[1], -1) {
			hosts = append(hosts, v[1])
		}
	}
	if len(hosts) == 0 {
		return []string{AnyHost}
	}
	slices.Sort(hosts)
	return slices.Compact(hosts)
}

// EffectivePriority returns the priority used by Traefik: the explicit one if set, otherwise the
// length of the match rule. The bool reports whether the priority is explicit.
func (r Route) EffectivePriority() (int, bool) {
	if r.Priority != 0 {
		return r.Priority, true
	}
	return len(r.Match), false
}

// BuildRows returns one row per route and host of the given IngressRoutes, sorted by host, then by
// priority (highest first), then by match. name returns the displayed name of an IngressRoute.
func BuildRows(irs []IngressRoute, name func(IngressRoute) string) []Row {
	var rows []Row
	for _, ir := range irs {
		for _, r := range ir.Spec.Routes {
			prio, explicit := r.EffectivePriority()
			for _, host := range Hosts(r.Match) {
				rows = append(rows, Row{
					Host:             host,
					Name:             name(ir),
					Marker:           ir.Marker,
					Priority:         prio,
					ExplicitPriority: explicit,
					Match:            r.Match,
					Services:         refNames(r.Services),
					Middlewares:      refNames(r.Middlewares),
				})
			}
		}
	}
	slices.SortFunc(rows, func(a, b Row) int {
		return cmp.Or(
			cmp.Compare(a.Host, b.Host),
			cmp.Compare(b.Priority, a.Priority),
			cmp.Compare(a.Match, b.Match),
			cmp.Compare(a.Name, b.Name),
		)
	})
	return rows
}

// FilterHost returns the rows whose host contains substr (case-insensitive), plus the AnyHost rows
// as they compete with all hosts. An empty substr returns all rows.
func FilterHost(rows []Row, substr string) []Row {
	if substr == "" {
		return rows
	}
	substr = strings.ToLower(substr)
	return slices.DeleteFunc(slices.Clone(rows), func(r Row) bool {
		return r.Host != AnyHost && !strings.Contains(strings.ToLower(r.Host), substr)
	})
}

func refNames(refs []Ref) []string {
	names := make([]string, 0, len(refs))
	for _, r := range refs {
		names = append(names, r.Name)
	}
	return names
}
