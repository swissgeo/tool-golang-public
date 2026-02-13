package completions

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/spf13/cobra"

	"github.com/geoadmin/tool-golang-bgdi/e2e-tests/cmd/organization"
)

//-----------------------------------------------------------------------------

func CompleteTests(cmd *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	org, err := cmd.Flags().GetString("org")
	if err != nil {
		fmt.Printf("Error getting --org flag: %v\n", err)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	repoPath, err := getE2ERepo(org)
	if err != nil {
		fmt.Printf("Error finding git repo: %v\n", err)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	testNames, err := findTests(repoPath, organization.Organization(org))
	if err != nil {
		fmt.Printf("Error finding tests: %v\n", err)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return testNames, cobra.ShellCompDirectiveNoFileComp
}

//-----------------------------------------------------------------------------

func pathToPythonModule(path string) string {
	moduleName := strings.ReplaceAll(path, string(os.PathSeparator), ".")
	moduleName = strings.TrimSuffix(moduleName, ".py")
	return moduleName
}

//-----------------------------------------------------------------------------

func findTests(repoPath string, org organization.Organization) ([]string, error) {
	var testNames []string
	// Search only in the tests folder of the repo
	root := filepath.Join(repoPath, "tests")

	// Walk the repo to find test files using WalkDir
	err := filepath.WalkDir(
		root,
		func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}

			if d.IsDir() && (d.Name() == "__pycache__" || d.Name() == "lib" || d.Name() == "fixtures") {
				return filepath.SkipDir
			}

			if d.IsDir() || (strings.HasPrefix(d.Name(), "test_") && strings.HasSuffix(d.Name(), ".py")) {
				// remove the root prefix
				relPath := strings.TrimPrefix(strings.TrimPrefix(path, root), string(os.PathSeparator))
				if org == organization.GEOADMIN {
					testNames = append(testNames, pathToPythonModule(relPath))
				} else {
					testNames = append(testNames, relPath)
				}
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return testNames, nil
}

//-----------------------------------------------------------------------------

func getE2ERepo(org string) (string, error) {
	var repo *git.Repository
	var err error
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get the cache directory: %w", err)
	}

	repoPath := filepath.Join(cacheDir, fmt.Sprintf("e2e-tests-%s", org))
	repoURL := fmt.Sprintf("git@github.com:%s/infra-e2e-tests.git", org)
	mainBranch := "master"
	if org == string(organization.SWISSGEO) {
		// Swissgeo organization repo use main as main branch instead of master
		mainBranch = "main"
	}

	auth, err := ssh.NewSSHAgentAuth("git")
	if err != nil {
		return "", fmt.Errorf("failed to create SSH agent auth: %w", err)
	}

	_, err = os.Stat(repoPath)
	switch {
	case os.IsNotExist(err):
		// folder does not exist, clone the repo
		repo, err = git.PlainClone(repoPath, false, &git.CloneOptions{
			URL:           repoURL,
			Auth:          auth,
			ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", mainBranch)),
			SingleBranch:  true,
			Depth:         1,
			Progress:      nil,
		})
		if err != nil {
			return "", fmt.Errorf("failed to clone repo %s into %s: %w", repoURL, repoPath, err)
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

	// Update the repo
	err = repo.Fetch(&git.FetchOptions{
		Auth:       auth,
		Depth:      1,
		Force:      true,
		Prune:      true,
		RemoteName: "origin",
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return "", fmt.Errorf("failed to fetch repo: %w", err)
	}

	// Get origin/master|main reference
	ref, err := repo.Reference(plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", mainBranch)), true)
	if err != nil {
		return "", fmt.Errorf("failed to get origin/%s ref: %w", mainBranch, err)
	}

	// Get the worktree
	workTree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	// Update the worktree repo
	err = workTree.Reset(&git.ResetOptions{
		Mode:   git.HardReset,
		Commit: ref.Hash(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to reset repo to remote %s: %w", mainBranch, err)
	}

	return repoPath, nil
}

//-----------------------------------------------------------------------------
