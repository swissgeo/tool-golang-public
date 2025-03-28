package completions

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

//-----------------------------------------------------------------------------

func CompleteTests(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	repoPath, err := findGitRepo()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	testNames, err := findTests(repoPath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return testNames, cobra.ShellCompDirectiveNoFileComp
}

//-----------------------------------------------------------------------------

func findTests(repoPath string) ([]string, error) {
	var testNames []string
	repoPath = fmt.Sprintf("%s/tests", repoPath)

	// Walk the repo to find test files using WalkDir
	err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}

		if strings.HasSuffix(path, "__pycache__") {
			return filepath.SkipDir
		}

		// Match test files
		matched, _ := regexp.MatchString(`test_.*\.py$|__init__\.py$`, d.Name())
		if matched {
			// Convert file path to Python module notation
			relPath := strings.TrimPrefix(path, repoPath+"/")
			moduleName := strings.ReplaceAll(relPath, "/", ".")
			moduleName = strings.TrimSuffix(moduleName, ".py")
			moduleName = strings.TrimSuffix(moduleName, ".__init__") // Remove trailing .__init__
			if moduleName != "__init__" {
				testNames = append(testNames, moduleName)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return testNames, nil
}

//-----------------------------------------------------------------------------

// Find the Git repository for the E2E tests on the local home folder
func findGitRepo() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	homeDir := usr.HomeDir
	gitRepo := "infra-e2e-tests"
	var gitRepoPath string

	// Walk through the home directory to find a matching .git/config file
	err = filepath.WalkDir(homeDir, func(path string, entry os.DirEntry, e error) error {
		if e != nil {
			return e
		}

		// skip dot directory (hidden)
		if strings.HasPrefix(entry.Name(), ".") && entry.IsDir() {
			return filepath.SkipDir
		}

		// skip well known directory
		if entry.IsDir() && slices.Contains([]string{
			"node_modules",
			"Downloads",
			"Pictures",
			"Music",
			"snap",
			"Videos",
			"Templates",
			"aws",
			"go",
		}, entry.Name()) {
			return filepath.SkipDir
		}

		if entry.Name() == "infra-e2e-tests" {
			gitRepoPath = path
			return filepath.SkipAll
		}

		return nil // continue walking
	})
	if err != nil {
		return "", fmt.Errorf("error while trying to find git repo: %w", err)
	}

	if gitRepoPath == "" {
		return "", fmt.Errorf("git repo %s not found", gitRepo)
	}
	return gitRepoPath, nil
}

//-----------------------------------------------------------------------------
