package version

import (
	"os/exec"
	"strings"
)

// Version holds the application version
var Version = "dev" // Fallback version

// getGitVersion attempts to retrieve the latest Git tag
func GetGitVersion() string {
	cmd := exec.Command("git", "describe", "--tags", "--always")
	output, err := cmd.Output()
	if err != nil {
		return Version // Return default if Git command fails
	}
	return strings.TrimSpace(string(output))
}
