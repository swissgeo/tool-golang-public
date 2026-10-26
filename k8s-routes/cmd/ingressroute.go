package cmd

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	"gopkg.in/yaml.v3"
)

const ingressRouteKind = "IngressRoute"

// IngressRoute is the subset of a Traefik IngressRoute needed to list its routes.
type IngressRoute struct {
	Kind     string           `yaml:"kind"`
	Metadata Metadata         `yaml:"metadata"`
	Spec     IngressRouteSpec `yaml:"spec"`

	// Marker is not parsed, it is set when merging local IngressRoutes into the live ones.
	Marker Marker `yaml:"-"`

	// rawSpec is the full generic spec, used to detect any difference between two IngressRoutes,
	// including fields not modelled above.
	rawSpec any
}

// Marker flags a local IngressRoute merged into the live ones.
type Marker string

const (
	NoMarker Marker = ""
	// MarkerNew flags a local IngressRoute that does not exist live yet.
	MarkerNew Marker = "(+)"
	// MarkerChanged flags a local IngressRoute that differs from the live one.
	MarkerChanged Marker = "(*)"
)

type Metadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type IngressRouteSpec struct {
	Routes []Route `yaml:"routes"`
}

type Route struct {
	Match string `yaml:"match"`
	// Priority 0 means unset, Traefik then uses the length of the match rule.
	Priority    int   `yaml:"priority"`
	Services    []Ref `yaml:"services"`
	Middlewares []Ref `yaml:"middlewares"`
}

type Ref struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// Key returns the unique identifier <namespace>/<name> of the IngressRoute.
func (ir IngressRoute) Key() string {
	return ir.Metadata.Namespace + "/" + ir.Metadata.Name
}

// SameSpec reports whether both IngressRoutes have a semantically equal spec.
func (ir IngressRoute) SameSpec(other IngressRoute) bool {
	return reflect.DeepEqual(ir.rawSpec, other.rawSpec)
}

// decodeIngressRoute decodes a single object. The bool is false if it is not an IngressRoute
// (including empty YAML documents).
func decodeIngressRoute(node *yaml.Node) (IngressRoute, bool, error) {
	var ir IngressRoute
	if err := node.Decode(&ir); err != nil {
		return ir, false, err
	}
	if ir.Kind != ingressRouteKind {
		return ir, false, nil
	}
	var raw struct {
		Spec any `yaml:"spec"`
	}
	if err := node.Decode(&raw); err != nil {
		return ir, false, err
	}
	ir.rawSpec = raw.Spec
	return ir, true, nil
}

// ParseList parses a list of objects as returned by `kubectl get -o yaml` and returns its
// IngressRoutes.
func ParseList(r io.Reader) ([]IngressRoute, error) {
	var list struct {
		Items []yaml.Node `yaml:"items"`
	}
	if err := yaml.NewDecoder(r).Decode(&list); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed to parse object list: %w", err)
	}
	var routes []IngressRoute
	for i := range list.Items {
		ir, ok, err := decodeIngressRoute(&list.Items[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse item %d: %w", i, err)
		}
		if ok {
			routes = append(routes, ir)
		}
	}
	return routes, nil
}

// ParseManifests parses a multi-document YAML stream as returned by `kustomize build` and returns
// its IngressRoutes.
func ParseManifests(r io.Reader) ([]IngressRoute, error) {
	dec := yaml.NewDecoder(r)
	var routes []IngressRoute
	// Decode each yaml document separated by --- (default behaviour of yaml decoder)
	for i := 0; ; i++ {
		var doc yaml.Node
		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML document %d: %w", i, err)
		}
		ir, ok, err := decodeIngressRoute(&doc)
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML document %d: %w", i, err)
		}
		if ok {
			routes = append(routes, ir)
		}
	}
	return routes, nil
}
