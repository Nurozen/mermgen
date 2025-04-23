package parser

import (
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// ProjectData represents the parsed structure of a Go project
type ProjectData struct {
	Packages  map[string]*PackageData
	Imports   map[string][]string // package -> imported packages
	Relations map[string][]string // relationships between types
}

// PackageData represents a parsed Go package
type PackageData struct {
	Name      string
	Types     map[string]*TypeData
	Functions map[string]*FunctionData
}

// TypeData represents a Go type (struct, interface, etc.)
type TypeData struct {
	Name       string
	Kind       string // "struct", "interface", etc.
	Fields     map[string]string
	Methods    map[string]*FunctionData
	Implements []string
}

// FunctionData represents a Go function
type FunctionData struct {
	Name       string
	Receiver   string
	Parameters []ParameterData
	Returns    []string
}

// ParameterData represents a function parameter
type ParameterData struct {
	Name string
	Type string
}

// ParseGoProject parses a Go project directory and returns structured data
func ParseGoProject(projectPath string) (*ProjectData, error) {
	projectData := &ProjectData{
		Packages:  make(map[string]*PackageData),
		Imports:   make(map[string][]string),
		Relations: make(map[string][]string),
	}

	// Walk through the project directory
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		// Parse the Go file
		fileData, err := parseGoFile(path)
		if err != nil {
			return fmt.Errorf("error parsing file %s: %w", path, err)
		}

		// Merge the file data into the project data
		mergeFileData(projectData, fileData, path)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking project directory: %w", err)
	}

	return projectData, nil
}

// parseGoFile parses a single Go file using Tree-sitter
func parseGoFile(filePath string) (map[string]interface{}, error) {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Initialize Tree-sitter parser
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	// Parse the file
	tree, err := parser.ParseCtx(nil, nil, content)
	if err != nil {
		return nil, fmt.Errorf("error parsing with tree-sitter: %w", err)
	}
	defer tree.Close()

	// Extract information from the syntax tree
	result := extractInformation(tree.RootNode(), content)

	return result, nil
}

// extractInformation extracts relevant information from the syntax tree
func extractInformation(node *sitter.Node, content []byte) map[string]interface{} {
	result := make(map[string]interface{})
	
	// This is a simplified version - in a real implementation, you'd recursively
	// traverse the syntax tree to extract packages, imports, type definitions, etc.
	
	// Get package name
	packageNode := findFirstNodeOfType(node, "package_clause")
	if packageNode != nil {
		identifierNode := findFirstNodeOfType(packageNode, "identifier")
		if identifierNode != nil {
			result["package"] = string(content[identifierNode.StartByte():identifierNode.EndByte()])
		}
	}
	
	// Get imports
	imports := findAllNodesOfType(node, "import_spec")
	importList := make([]string, 0, len(imports))
	for _, importNode := range imports {
		importPathNode := findFirstNodeOfType(importNode, "interpreted_string_literal")
		if importPathNode != nil {
			// Remove the quotes from the import path
			importPath := string(content[importPathNode.StartByte()+1:importPathNode.EndByte()-1])
			importList = append(importList, importPath)
		}
	}
	result["imports"] = importList
	
	// Extract type declarations (structs, interfaces)
	typeDecls := findAllNodesOfType(node, "type_declaration")
	types := make([]map[string]interface{}, 0, len(typeDecls))
	
	for _, typeDecl := range typeDecls {
		typeSpec := findFirstNodeOfType(typeDecl, "type_spec")
		if typeSpec == nil {
			continue
		}
		
		typeNameNode := findFirstNodeOfType(typeSpec, "identifier")
		if typeNameNode == nil {
			continue
		}
		
		typeName := string(content[typeNameNode.StartByte():typeNameNode.EndByte()])
		
		typeInfo := map[string]interface{}{
			"name": typeName,
		}
		
		// Check if it's a struct or interface
		structTypeNode := findFirstNodeOfType(typeSpec, "struct_type")
		if structTypeNode != nil {
			typeInfo["kind"] = "struct"
			// Extract fields - simplified for this example
		}
		
		interfaceTypeNode := findFirstNodeOfType(typeSpec, "interface_type")
		if interfaceTypeNode != nil {
			typeInfo["kind"] = "interface"
			// Extract methods - simplified for this example
		}
		
		types = append(types, typeInfo)
	}
	result["types"] = types
	
	// Extract function declarations
	funcDecls := findAllNodesOfType(node, "function_declaration")
	functions := make([]map[string]interface{}, 0, len(funcDecls))
	
	for _, funcDecl := range funcDecls {
		funcNameNode := findFirstChildOfType(funcDecl, "identifier")
		if funcNameNode == nil {
			continue
		}
		
		funcName := string(content[funcNameNode.StartByte():funcNameNode.EndByte()])
		
		functionInfo := map[string]interface{}{
			"name": funcName,
		}
		
		// Check if it's a method
		receiverNode := findFirstNodeOfType(funcDecl, "parameter_list")
		if receiverNode != nil && receiverNode.PrevSibling() != nil && receiverNode.PrevSibling().Type() == "identifier" {
			// It's a method, extract receiver type - simplified
		}
		
		functions = append(functions, functionInfo)
	}
	result["functions"] = functions
	
	return result
}

// Helper functions to navigate the syntax tree

func findFirstNodeOfType(node *sitter.Node, nodeType string) *sitter.Node {
	if node.Type() == nodeType {
		return node
	}
	
	childCount := node.ChildCount()
	for i := uint32(0); i < childCount; i++ {
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
	childCount := node.ChildCount()
	for i := uint32(0); i < childCount; i++ {
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
	
	childCount := node.ChildCount()
	for i := uint32(0); i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		
		results = append(results, findAllNodesOfType(child, nodeType)...)
	}
	
	return results
}

// mergeFileData merges the data from a single file into the project data
func mergeFileData(projectData *ProjectData, fileData map[string]interface{}, filePath string) {
	// Extract package name
	packageName, ok := fileData["package"].(string)
	if !ok {
		return
	}
	
	// Create package if it doesn't exist
	if _, exists := projectData.Packages[packageName]; !exists {
		projectData.Packages[packageName] = &PackageData{
			Name:      packageName,
			Types:     make(map[string]*TypeData),
			Functions: make(map[string]*FunctionData),
		}
	}
	
	// Add imports
	if imports, ok := fileData["imports"].([]string); ok {
		projectData.Imports[packageName] = append(projectData.Imports[packageName], imports...)
	}
	
	// Add types
	if types, ok := fileData["types"].([]map[string]interface{}); ok {
		for _, typeInfo := range types {
			typeName, ok := typeInfo["name"].(string)
			if !ok {
				continue
			}
			
			kind, _ := typeInfo["kind"].(string)
			
			// Create type if it doesn't exist
			if _, exists := projectData.Packages[packageName].Types[typeName]; !exists {
				projectData.Packages[packageName].Types[typeName] = &TypeData{
					Name:    typeName,
					Kind:    kind,
					Fields:  make(map[string]string),
					Methods: make(map[string]*FunctionData),
				}
			}
			
			// More processing would be done here for fields, methods, etc.
		}
	}
	
	// Add functions
	if functions, ok := fileData["functions"].([]map[string]interface{}); ok {
		for _, funcInfo := range functions {
			funcName, ok := funcInfo["name"].(string)
			if !ok {
				continue
			}
			
			// Create function
			function := &FunctionData{
				Name:       funcName,
				Parameters: []ParameterData{},
				Returns:    []string{},
			}
			
			// Check if it's a method
			if receiver, ok := funcInfo["receiver"].(string); ok {
				function.Receiver = receiver
				
				// Add method to type if it exists
				if typeData, exists := projectData.Packages[packageName].Types[receiver]; exists {
					typeData.Methods[funcName] = function
				}
			} else {
				// It's a regular function
				projectData.Packages[packageName].Functions[funcName] = function
			}
		}
	}
} 