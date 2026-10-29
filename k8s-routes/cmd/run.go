package cmd

import (
	"context"
	"fmt"
	"io"
	"runtime"

	"golang.org/x/sync/errgroup"
)

// runLocal lists the routes rendered from the local kustomize overlays of env.
func runLocal(ctx context.Context, out, errOut io.Writer, env string) error {
	irs, err := renderLocal(ctx, errOut, env)
	if err != nil {
		return err
	}
	return printRoutes(out, irs, func(ir IngressRoute) string { return ir.Metadata.Name })
}

// runLive lists the routes deployed in the cluster, with the local ones merged in if WithLocal.
func runLive(ctx context.Context, out, errOut io.Writer) error {
	kubeContext := KubeContext
	if kubeContext == "" {
		var err error
		if kubeContext, err = CurrentContext(ctx); err != nil {
			return err
		}
	}
	var env string
	if WithLocal {
		var err error
		if env, err = DetectEnv(kubeContext); err != nil {
			return err
		}
		fmt.Fprintf(errOut, "Context: %s (with local changes of %q)\n", kubeContext, env)
	} else {
		fmt.Fprintf(errOut, "Context: %s\n", kubeContext)
	}

	// Fetch the live IngressRoutes while rendering the local ones
	var live, local []IngressRoute
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		live, err = LiveIngressRoutes(gctx, kubeContext)
		return err
	})
	if WithLocal {
		g.Go(func() error {
			var err error
			local, err = renderLocal(gctx, errOut, env)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return printRoutes(out, Merge(live, local), IngressRoute.Key)
}

func renderLocal(ctx context.Context, errOut io.Writer, env string) ([]IngressRoute, error) {
	overlays, err := FindOverlays(Root, Glob, env)
	if err != nil {
		return nil, err
	}
	return RenderOverlays(ctx, overlays, runtime.NumCPU(), KustomizeBuild(EnableHelm), errOut)
}

func printRoutes(out io.Writer, irs []IngressRoute, name func(IngressRoute) string) error {
	rows := FilterHost(BuildRows(irs, name), Host)
	return RenderTable(out, rows, UseColor(out))
}
