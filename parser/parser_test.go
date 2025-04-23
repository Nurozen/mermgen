package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGoFile(t *testing.T) {
	// Create a temporary Go file for testing
	tmpDir, err := os.MkdirTemp("", "parser-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a sample Go file
	sampleCode := `package sample

import (
	"fmt"
	"strings"
)

// Person represents a human
type Person struct {
	Name string
	Age  int
}

// SayHello prints a greeting
func (p *Person) SayHello() {
	fmt.Printf("Hello, my name is %s\n", p.Name)
}

func main() {
	person := &Person{
		Name: "John",
		Age:  30,
	}
	person.SayHello()
}
`

	filePath := filepath.Join(tmpDir, "sample.go")
	if err := os.WriteFile(filePath, []byte(sampleCode), 0644); err != nil {
		t.Fatalf("Failed to write sample file: %v", err)
	}

	// Parse the file
	fileData, err := parseGoFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse Go file: %v", err)
	}

	// Verify the parsed results
	if fileData.PackageName != "sample" {
		t.Errorf("Expected package name 'sample', got '%v'", fileData.PackageName)
	}

	// Check content was correctly stored
	if fileData.Content != sampleCode {
		t.Errorf("File content does not match original")
	}

	// Verify parse tree was created
	if fileData.ParseTree == "" {
		t.Errorf("Parse tree is empty")
	}
}

func TestParseGoProject(t *testing.T) {
	// This is an integration test that requires Tree-sitter to be working correctly
	// For simplicity in this example, we'll skip it if SKIP_INTEGRATION_TESTS is set
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test")
	}

	// Create a temporary project structure
	tmpDir, err := os.MkdirTemp("", "project-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple project structure
	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	// Create a main.go file
	mainCode := `package main

import (
	"fmt"
	"example.com/myproject/pkg"
)

func main() {
	service := pkg.NewService("test")
	fmt.Println(service.Name())
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainCode), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Create a service.go file in the pkg directory
	serviceCode := `package pkg

// Service provides functionality
type Service struct {
	name string
}

// NewService creates a new service
func NewService(name string) *Service {
	return &Service{name: name}
}

// Name returns the service name
func (s *Service) Name() string {
	return s.name
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "service.go"), []byte(serviceCode), 0644); err != nil {
		t.Fatalf("Failed to write service.go: %v", err)
	}

	// Parse the project
	projectData, err := ParseGoProject(tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse project: %v", err)
	}

	// Verify the parsed project structure
	if len(projectData.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(projectData.Files))
	}

	// Check for the main.go file
	mainFilePath := filepath.Join(tmpDir, "main.go")
	mainFileData, exists := projectData.Files[mainFilePath]
	if !exists {
		t.Error("Main file not found")
	} else if mainFileData.PackageName != "main" {
		t.Errorf("Expected package 'main', got '%s'", mainFileData.PackageName)
	}

	// Check for the service.go file
	serviceFilePath := filepath.Join(pkgDir, "service.go")
	serviceFileData, exists := projectData.Files[serviceFilePath]
	if !exists {
		t.Error("Service file not found")
	} else if serviceFileData.PackageName != "pkg" {
		t.Errorf("Expected package 'pkg', got '%s'", serviceFileData.PackageName)
	}
}
