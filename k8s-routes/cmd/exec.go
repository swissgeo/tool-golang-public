package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// run runs an external command and returns its stdout. On failure, the error contains its stderr.
func run(ctx context.Context, name string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("%s not found in PATH", name)
		}
		return nil, fmt.Errorf(
			"%s %s failed: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()),
		)
	}
	return stdout.Bytes(), nil
}
