package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// RawProjectData represents the parsed structure of a Go project
// with raw parse tree information instead of manually extracted data
type RawProjectData struct {
	Files map[string]*FileData // filepath -> parsed file data
}

// FileData represents a parsed Go file with its raw content and tree
type FileData struct {
	Content     string
	PackageName string
	ParseTree   string // Serialized parse tree
}

// ParseGoProject parses a Go project directory and returns raw data
func ParseGoProject(projectPath string) (*RawProjectData, error) {
	projectData := &RawProjectData{
		Files: make(map[string]*FileData),
	}

	fmt.Println("projectPath", projectPath)

	// Walk through the project directory
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		// Parse the Go file with tree-sitter
		fileData, err := parseGoFile(path)
		if err != nil {
			return fmt.Errorf("error parsing file %s: %w", path, err)
		}

		// Store the raw file data
		projectData.Files[path] = fileData
		fmt.Printf("Added file: %s\n", path)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking project directory: %w", err)
	}
	//project data to string
	projectDataString, err := json.Marshal(projectData)
	if err != nil {
		return nil, fmt.Errorf("error marshalling project data: %w", err)
	}
	fmt.Println("Parsed project data", string(projectDataString))
	return projectData, nil
}

// parseGoFile parses a single Go file using Tree-sitter
func parseGoFile(filePath string) (*FileData, error) {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Initialize Tree-sitter parser
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	// Create a context and parse the file
	ctx := context.Background()
	tree, err := parser.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil, fmt.Errorf("error parsing with tree-sitter: %w", err)
	}
	fmt.Printf("tree: %v\n", tree)
	defer tree.Close()

	// Extract package name for basic information
	packageName := ""
	packageNode := findFirstNodeOfType(tree.RootNode(), "package_clause")
	if packageNode != nil {
		identifierNode := findFirstNodeOfType(packageNode, "identifier")
		if identifierNode != nil {
			packageName = string(content[identifierNode.StartByte():identifierNode.EndByte()])
		}
	}

	// Create file data with raw parse tree
	fileData := &FileData{
		Content:     string(content),
		PackageName: packageName,
		ParseTree:   tree.RootNode().String(),
	}

	return fileData, nil
}

// Helper functions to navigate the syntax tree

func findFirstNodeOfType(node *sitter.Node, nodeType string) *sitter.Node {
	if node.Type() == nodeType {
		return node
	}

	childCount := int(node.ChildCount())
	for i := 0; i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if result := findFirstNodeOfType(child, nodeType); result != nil {
			return result
		}
	}

	return nil
}

func findFirstChildOfType(node *sitter.Node, nodeType string) *sitter.Node {
	childCount := int(node.ChildCount())
	for i := 0; i < childCount; i++ {
		child := node.Child(i)
		if child != nil && child.Type() == nodeType {
			return child
		}
	}
	return nil
}

func findAllNodesOfType(node *sitter.Node, nodeType string) []*sitter.Node {
	var results []*sitter.Node

	if node.Type() == nodeType {
		results = append(results, node)
	}

	childCount := int(node.ChildCount())
	for i := 0; i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		results = append(results, findAllNodesOfType(child, nodeType)...)
	}

	return results
}
