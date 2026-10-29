package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
)

// CurrentContext returns the current kubectl context.
func CurrentContext(ctx context.Context) (string, error) {
	out, err := run(ctx, "kubectl", "config", "current-context")
	if err != nil {
		return "", err
	}
	kubeContext := strings.TrimSpace(string(out))
	if kubeContext == "" {
		return "", errors.New("no current kubectl context, use --context")
	}
	return kubeContext, nil
}

// LiveIngressRoutes returns the IngressRoutes of all namespaces deployed in the cluster of the given
// kubectl context.
func LiveIngressRoutes(ctx context.Context, kubeContext string) ([]IngressRoute, error) {
	out, err := run(ctx, "kubectl", "--context", kubeContext,
		"get", "ingressroutes.traefik.io", "--all-namespaces", "-o", "yaml")
	if err != nil {
		return nil, err
	}
	irs, err := ParseList(bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the IngressRoutes of context %s: %w", kubeContext, err)
	}
	return irs, nil
}
