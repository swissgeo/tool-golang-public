package completions

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/spf13/cobra"
)

//-----------------------------------------------------------------------------

func CompleteTests(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	repoPath, err := getE2ERepo()
	if err != nil {
		fmt.Printf("Error finding git repo: %v\n", err)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	testNames, err := findTests(repoPath)
	if err != nil {
		fmt.Printf("Error finding tests: %v\n", err)
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

func getE2ERepo() (string, error) {
	var repo *git.Repository
	var err error
	repoPath := os.Getenv("HOME") + "/.e2e-tests"

	auth, err := ssh.NewSSHAgentAuth("git")
	if err != nil {
		return "", fmt.Errorf("failed to create SSH agent auth: %w", err)
	}

	_, err = os.Stat(repoPath)
	switch {
	case os.IsNotExist(err):
		// folder does not exist, clone the repo
		repo, err = git.PlainClone(repoPath, false, &git.CloneOptions{
			URL:           "git@github.com:geoadmin/infra-e2e-tests.git",
			Auth:          auth,
			ReferenceName: "refs/heads/master",
			SingleBranch:  true,
			Depth:         1,
			Progress:      nil,
		})
		if err != nil {
			return "", fmt.Errorf("failed to clone repo: %w", err)
		}

	case err != nil:
		return "", fmt.Errorf("failed to stat %s: %w", repo, err)

	default:
		// open the existing repo
		repo, err = git.PlainOpen(repoPath)
		if err != nil {
			return "", fmt.Errorf("failed to open existing repo: %w", err)
		}
	}

	// Get the worktree
	workTree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	// Update the repo
	err = workTree.Pull(&git.PullOptions{
		Auth:         auth,
		Progress:     nil,
		SingleBranch: true,
		Depth:        1,
		Force:        true,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return "", fmt.Errorf("failed to pull repo: %w", err)
	}

	return repoPath, nil
}

//-----------------------------------------------------------------------------
