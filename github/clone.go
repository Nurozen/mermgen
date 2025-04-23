package github

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CloneRepository clones a GitHub repository to a temporary directory and returns the path to the cloned repo
func CloneRepository(repoURL string) (string, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "mermgen-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Ensure the URL is in the correct format
	if !strings.HasPrefix(repoURL, "https://") && !strings.HasPrefix(repoURL, "git@") {
		repoURL = "https://" + repoURL
	}

	// Clone the repository
	cmd := exec.Command("git", "clone", "--depth=1", repoURL, tempDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tempDir) // Clean up the temp directory on error
		return "", fmt.Errorf("git clone failed: %w\nOutput: %s", err, output)
	}

	return tempDir, nil
} 