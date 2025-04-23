package github

import (
	"os"
	"strings"
	"testing"
)

func TestCloneRepository(t *testing.T) {
	// Skip this test if running in CI without valid Git credentials
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI environment")
	}

	// Test with a small, public repository
	repoURL := "github.com/golang/example"
	
	tempDir, err := CloneRepository(repoURL)
	if err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after test
	
	// Verify the repo was cloned by checking for common files
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read cloned directory: %v", err)
	}
	
	// Check if common Git repo files/directories exist
	hasGitDir := false
	hasReadme := false
	
	for _, file := range files {
		if file.IsDir() && file.Name() == ".git" {
			hasGitDir = true
		}
		if strings.HasPrefix(strings.ToLower(file.Name()), "readme") {
			hasReadme = true
		}
	}
	
	if !hasGitDir {
		t.Error("Cloned repository doesn't have a .git directory")
	}
	
	if !hasReadme {
		t.Error("Cloned repository doesn't have a README file")
	}
} 