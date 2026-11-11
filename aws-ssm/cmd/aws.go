package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runAWSCmd runs the aws cli and returns its stdout and stderr. On failure the stderr is not
// printed so the caller can decide how to handle known errors (see awsCmdError). On success
// the stderr (e.g. warnings or credential notices) is forwarded to os.Stderr.
func runAWSCmd(ctx context.Context, cmdArgs []string, verbose bool) (string, string, error) {
	if verbose {
		fmt.Println("aws", strings.Join(cmdArgs, " "))
	}
	stdout, stderr, err := runAWSCmdQuiet(ctx, cmdArgs)
	if err == nil && stderr != "" {
		_, _ = os.Stderr.WriteString(stderr)
	}
	return stdout, stderr, err
}

// runAWSCmdQuiet runs the aws cli and returns its stdout and stderr without printing anything,
// e.g. for shell completion where any output would corrupt the shell prompt.
func runAWSCmdQuiet(ctx context.Context, cmdArgs []string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	awsCmd := exec.CommandContext(ctx, "aws", cmdArgs...)
	awsCmd.Stdout = &stdout
	awsCmd.Stderr = &stderr
	err := awsCmd.Run()
	return stdout.String(), stderr.String(), err
}

// awsCmdError wraps an aws cli error with a message and the aws cli stderr output.
func awsCmdError(msg string, err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return fmt.Errorf("%s: %w", msg, err)
	}
	return fmt.Errorf("%s: %w: %s", msg, err, stderr)
}
