package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// FindLinterConfig searches for a kube-linter config file starting at folder
// and walking up to the working directory. It tries .kube-linter.yaml then
// .kube-linter.yml at each level, matching the order kube-linter itself uses
// when searching for its config file.
func FindLinterConfig(folder string) (string, error) {
	current := folder
	for {
		for _, name := range []string{".kube-linter.yaml", ".kube-linter.yml"} {
			configPath := filepath.Join(current, name)
			if _, err := os.Stat(configPath); err == nil {
				return configPath, nil
			}
		}
		if current == "." {
			break
		}
		current = filepath.Dir(current)
	}
	return "", fmt.Errorf("no .kube-linter.yaml or .kube-linter.yml config file found for %s", folder)
}

func lintFolder(folder, config string) bool {
	ctx := context.Background()
	kustomizationFile := filepath.Join(folder, "kustomization.yaml")
	args := []string{"lint", kustomizationFile, "--config", config, "--fail-if-no-objects-found"}
	if !NoColor {
		args = append(args, "--with-color")
	}
	cmd := exec.CommandContext(ctx, "kube-linter", args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	ok := err == nil
	var details []byte
	if !ok {
		details = append(stdoutBuf.Bytes(), stderrBuf.Bytes()...)
	}
	printResult("kube-linter lint", folder, ok, details)
	return ok
}

func LintKustomize(folders []string, workers int, failFast bool) bool {
	// Resolve all configs up front so a missing config fails immediately.
	configs := make(map[string]string, len(folders))
	for _, folder := range folders {
		var config string
		var cfgErr error
		config, cfgErr = FindLinterConfig(folder)
		if cfgErr != nil {
			fmt.Fprintln(os.Stderr, cfgErr)
			return false
		}
		configs[folder] = config
	}

	var wg sync.WaitGroup
	taskChan := make(chan string, len(folders))

	valid := true
	for range workers {
		wg.Go(func() {
			for folder := range taskChan {
				if !lintFolder(folder, configs[folder]) {
					valid = false
					if failFast {
						os.Exit(1)
					}
				}
			}
		})
	}

	for _, folder := range folders {
		taskChan <- folder
	}
	close(taskChan)
	wg.Wait()

	return valid
}
