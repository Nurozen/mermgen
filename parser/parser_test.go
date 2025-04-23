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
	result, err := parseGoFile(filePath)
	if err != nil {
		t.Fatalf("Failed to parse Go file: %v", err)
	}

	// Verify the parsed results
	packageName, ok := result["package"].(string)
	if !ok || packageName != "sample" {
		t.Errorf("Expected package name 'sample', got '%v'", result["package"])
	}

	imports, ok := result["imports"].([]string)
	if !ok {
		t.Errorf("Failed to extract imports")
	} else {
		if len(imports) != 2 {
			t.Errorf("Expected 2 imports, got %d", len(imports))
		}
	}

	types, ok := result["types"].([]map[string]interface{})
	if !ok {
		t.Errorf("Failed to extract types")
	} else {
		if len(types) != 1 {
			t.Errorf("Expected 1 type, got %d", len(types))
		} else {
			typeInfo := types[0]
			if typeName, ok := typeInfo["name"].(string); !ok || typeName != "Person" {
				t.Errorf("Expected type name 'Person', got '%v'", typeInfo["name"])
			}
			
			if kind, ok := typeInfo["kind"].(string); !ok || kind != "struct" {
				t.Errorf("Expected kind 'struct', got '%v'", typeInfo["kind"])
			}
		}
	}

	functions, ok := result["functions"].([]map[string]interface{})
	if !ok {
		t.Errorf("Failed to extract functions")
	} else {
		// Should find the main function
		foundMain := false
		for _, fn := range functions {
			if name, ok := fn["name"].(string); ok && name == "main" {
				foundMain = true
				break
			}
		}
		
		if !foundMain {
			t.Errorf("Failed to find 'main' function")
		}
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
	if len(projectData.Packages) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(projectData.Packages))
	}

	// Check for the main package
	mainPkg, exists := projectData.Packages["main"]
	if !exists {
		t.Error("Main package not found")
	} else if len(mainPkg.Functions) == 0 {
		t.Error("No functions found in main package")
	}

	// Check for the pkg package
	pkgPkg, exists := projectData.Packages["pkg"]
	if !exists {
		t.Error("pkg package not found")
	} else {
		// Check for the Service type
		_, exists := pkgPkg.Types["Service"]
		if !exists {
			t.Error("Service type not found in pkg package")
		}

		// Check for the NewService function
		_, exists = pkgPkg.Functions["NewService"]
		if !exists {
			t.Error("NewService function not found in pkg package")
		}
	}
} 