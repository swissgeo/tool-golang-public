package cmd

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
)

// The env must be a whole part of the kube context name, separated by / _ or -
// (e.g. swissgeo/int-blue), so that e.g. "production" or "interface" don't match.
var envRe = regexp.MustCompile(`(?:^|[/_-])(dev|int|prod)(?:[/_-]|$)`)

// DetectEnv returns the env (dev|int|prod) contained in the kube context name.
func DetectEnv(kubeContext string) (string, error) {
	m := envRe.FindStringSubmatch(kubeContext)
	if m == nil {
		return "", fmt.Errorf("cannot determine the env (dev|int|prod) from the kube context %q", kubeContext)
	}
	return m[1], nil
}

// Merge returns the live IngressRoutes with the local ones replacing those with the same
// <namespace>/<name>, sorted by key. The local IngressRoutes are marked with MarkerNew if they
// don't exist live and MarkerChanged if their spec differs from the live one. Live IngressRoutes
// without local counterpart (e.g. removed or renamed locally) are kept.
func Merge(live, local []IngressRoute) []IngressRoute {
	byKey := make(map[string]IngressRoute, len(live)+len(local))
	for _, ir := range live {
		byKey[ir.Key()] = ir
	}
	for _, ir := range local {
		liveIR, exists := byKey[ir.Key()]
		switch {
		case !exists:
			ir.Marker = MarkerNew
		case !ir.SameSpec(liveIR):
			ir.Marker = MarkerChanged
		default:
			ir.Marker = NoMarker
		}
		byKey[ir.Key()] = ir
	}
	merged := make([]IngressRoute, 0, len(byKey))
	for _, key := range slices.Sorted(maps.Keys(byKey)) {
		merged = append(merged, byKey[key])
	}
	return merged
}
