package github

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// CloneRepository clones a GitHub repository to a temporary directory and returns the path to the cloned repo
func CloneRepository(repoURL string) (string, error) {
	// Check if this is a specific file request
	if strings.HasPrefix(repoURL, "@") {
		return FetchSingleFile(strings.TrimPrefix(repoURL, "@"))
	}

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "mermgen-")
	fmt.Println("tempDir", tempDir)
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

// FetchSingleFile downloads a single file from GitHub
// Example URL: https://github.com/spf13/cobra/blob/main/cobra.go
func FetchSingleFile(fileURL string) (string, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "mermgen-file-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Parse GitHub URL to extract owner, repo, branch, and path
	re := regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)`)
	matches := re.FindStringSubmatch(fileURL)
	if matches == nil || len(matches) < 5 {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("invalid GitHub file URL format: %s", fileURL)
	}

	owner := matches[1]
	repo := matches[2]
	branch := matches[3]
	path := matches[4]

	// Generate the raw content URL
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, branch, path)

	// Download the file
	resp, err := http.Get(rawURL)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to download file, status code: %d", resp.StatusCode)
	}

	// Create the necessary directories
	filename := filepath.Base(path)
	filePath := filepath.Join(tempDir, filename)

	// Create and write to the file
	out, err := os.Create(filePath)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filePath, nil
}
