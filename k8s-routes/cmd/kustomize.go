package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sync/errgroup"
)

const envPlaceholder = "{env}"

// Overlay is a kustomize overlay directory to render.
type Overlay struct {
	Dir string
	// Namespace given to the IngressRoutes without namespace, like the ArgoCD application would do.
	Namespace string
}

// BuildFunc renders a kustomize directory and returns its manifests as a multi-document YAML stream.
type BuildFunc func(ctx context.Context, dir string) ([]byte, error)

// FindOverlays returns the directories matching the glob relative to root, where {env} is replaced
// by env. The namespace of each overlay is the path segment containing the first wildcard of the
// glob (e.g. the service name for services/*/overlays/{env}), or empty if the glob has none.
func FindOverlays(root, glob, env string) ([]Overlay, error) {
	pattern := filepath.FromSlash(strings.ReplaceAll(glob, envPlaceholder, env))
	matches, globErr := filepath.Glob(filepath.Join(root, pattern))
	if globErr != nil {
		return nil, fmt.Errorf("invalid glob %q: %w", glob, globErr)
	}
	nsIndex := slices.IndexFunc(strings.Split(pattern, string(filepath.Separator)), hasWildcard)

	var overlays []Overlay
	for _, m := range matches {
		overlay, ok, err := newOverlay(root, m, nsIndex)
		if err != nil {
			return nil, err
		}
		if ok {
			overlays = append(overlays, overlay)
		}
	}
	if len(overlays) == 0 {
		return nil, fmt.Errorf("no kustomize overlays found for env %q matching %q", env, filepath.Join(root, pattern))
	}
	return overlays, nil
}

// newOverlay returns the overlay of the glob match dir. The bool is false if dir is not a directory.
func newOverlay(root, dir string, nsIndex int) (Overlay, bool, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return Overlay{}, false, err
	}
	if nsIndex < 0 {
		return Overlay{Dir: dir}, true, nil
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return Overlay{}, false, err
	}
	return Overlay{Dir: dir, Namespace: strings.Split(rel, string(filepath.Separator))[nsIndex]}, true, nil
}

func hasWildcard(segment string) bool {
	return strings.ContainsAny(segment, "*?[")
}

// KustomizeBuild returns a BuildFunc running `kustomize build`.
func KustomizeBuild(enableHelm bool) BuildFunc {
	return func(ctx context.Context, dir string) ([]byte, error) {
		args := []string{"build"}
		if enableHelm {
			args = append(args, "--enable-helm")
		}
		return run(ctx, "kustomize", append(args, dir)...)
	}
}

// RenderOverlays renders the overlays in parallel with at most workers concurrent builds and returns
// their IngressRoutes, in the order of the overlays. IngressRoutes without namespace get the
// namespace of their overlay. It stops at the first failure.
func RenderOverlays(ctx context.Context, overlays []Overlay, workers int, build BuildFunc) ([]IngressRoute, error) {
	results := make([][]IngressRoute, len(overlays))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)
	for i, overlay := range overlays {
		g.Go(func() error {
			out, err := build(ctx, overlay.Dir)
			if err != nil {
				return err
			}
			irs, err := ParseManifests(bytes.NewReader(out))
			if err != nil {
				return fmt.Errorf("failed to parse the output of %s: %w", overlay.Dir, err)
			}
			for j := range irs {
				if irs[j].Metadata.Namespace == "" {
					irs[j].Metadata.Namespace = overlay.Namespace
				}
			}
			results[i] = irs
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	var irs []IngressRoute
	for _, r := range results {
		irs = append(irs, r...)
	}
	return irs, nil
}
