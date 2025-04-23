package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Nurozen/mermgen/parser"
)

// GenerateDiagrams generates various Mermaid diagrams from the parsed project data
func GenerateDiagrams(projectData *parser.ProjectData) (map[string]string, error) {
	diagrams := make(map[string]string)

	// Generate different types of diagrams
	classDiagram, err := generateClassDiagram(projectData)
	if err != nil {
		return nil, fmt.Errorf("error generating class diagram: %w", err)
	}
	diagrams["class-diagram"] = classDiagram

	packageDiagram, err := generatePackageDiagram(projectData)
	if err != nil {
		return nil, fmt.Errorf("error generating package diagram: %w", err)
	}
	diagrams["package-diagram"] = packageDiagram

	sequenceDiagram, err := generateSequenceDiagram(projectData)
	if err != nil {
		return nil, fmt.Errorf("error generating sequence diagram: %w", err)
	}
	diagrams["sequence-diagram"] = sequenceDiagram

	return diagrams, nil
}

// generateClassDiagram creates a Mermaid class diagram from project data
func generateClassDiagram(projectData *parser.ProjectData) (string, error) {
	// Prepare data for the AI prompt
	var typeDefinitions []map[string]interface{}
	
	for pkgName, pkg := range projectData.Packages {
		for typeName, typeData := range pkg.Types {
			typeInfo := map[string]interface{}{
				"package": pkgName,
				"name":    typeName,
				"kind":    typeData.Kind,
				"fields":  typeData.Fields,
			}
			
			methods := make([]string, 0, len(typeData.Methods))
			for methodName := range typeData.Methods {
				methods = append(methods, methodName)
			}
			typeInfo["methods"] = methods
			
			typeDefinitions = append(typeDefinitions, typeInfo)
		}
	}
	
	// Build relationships
	var relationships []map[string]string
	for _, relations := range projectData.Relations {
		for i := 0; i < len(relations); i += 2 {
			if i+1 < len(relations) {
				relationships = append(relationships, map[string]string{
					"from": relations[i],
					"to":   relations[i+1],
					"type": "->",
				})
			}
		}
	}
	
	// Create AI prompt
	prompt := map[string]interface{}{
		"task":          "Generate a Mermaid class diagram",
		"types":         typeDefinitions,
		"relationships": relationships,
	}
	
	// Call AI to generate diagram
	return callAI(prompt, "class")
}

// generatePackageDiagram creates a Mermaid package diagram from project data
func generatePackageDiagram(projectData *parser.ProjectData) (string, error) {
	// Prepare data for the AI prompt
	var packages []map[string]interface{}
	
	for pkgName, pkg := range projectData.Packages {
		pkgInfo := map[string]interface{}{
			"name":      pkgName,
			"typeCount": len(pkg.Types),
			"funcCount": len(pkg.Functions),
		}
		packages = append(packages, pkgInfo)
	}
	
	// Build dependencies
	var dependencies []map[string]string
	for pkgName, imports := range projectData.Imports {
		for _, importedPkg := range imports {
			// Clean up import path to get just the package name
			parts := strings.Split(importedPkg, "/")
			importedPkgName := parts[len(parts)-1]
			
			// Check if it's a project package (not a standard library or external package)
			if _, exists := projectData.Packages[importedPkgName]; exists {
				dependencies = append(dependencies, map[string]string{
					"from": pkgName,
					"to":   importedPkgName,
				})
			}
		}
	}
	
	// Create AI prompt
	prompt := map[string]interface{}{
		"task":         "Generate a Mermaid package diagram",
		"packages":     packages,
		"dependencies": dependencies,
	}
	
	// Call AI to generate diagram
	return callAI(prompt, "package")
}

// generateSequenceDiagram creates a sample sequence diagram
func generateSequenceDiagram(projectData *parser.ProjectData) (string, error) {
	// For a sequence diagram, we'd need more information about function calls and flows
	// This is a simplified version that just creates a sample diagram
	
	// Find a "main" function or any entry point
	var entryPoints []string
	for pkgName, pkg := range projectData.Packages {
		for funcName := range pkg.Functions {
			if funcName == "main" {
				entryPoints = append(entryPoints, fmt.Sprintf("%s.%s", pkgName, funcName))
			}
		}
	}
	
	// Create AI prompt
	prompt := map[string]interface{}{
		"task":        "Generate a Mermaid sequence diagram",
		"entryPoints": entryPoints,
		"packages":    projectData.Packages,
	}
	
	// Call AI to generate diagram
	return callAI(prompt, "sequence")
}

// callAI calls an AI service to generate a Mermaid diagram
func callAI(prompt map[string]interface{}, diagramType string) (string, error) {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Convert prompt to JSON
	promptJSON, err := json.Marshal(prompt)
	if err != nil {
		return "", fmt.Errorf("error marshaling prompt: %w", err)
	}

	// Create request to OpenAI API
	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]interface{}{
			{
				"role": "system",
				"content": "You are an expert in Go programming and Mermaid diagrams. " +
					"Your task is to analyze Go code structure and generate a Mermaid diagram that " +
					"accurately represents the code. Only return the Mermaid diagram code, nothing else.",
			},
			{
				"role":    "user",
				"content": string(promptJSON),
			},
		},
		"temperature": 0.2,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	// Send request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestJSON))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	// Extract diagram from response
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("invalid response format")
	}

	message, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid message format")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid content format")
	}

	// Extract Mermaid code from content
	mermaidCode := extractMermaidCode(content)
	if mermaidCode == "" {
		return "", fmt.Errorf("no Mermaid code found in response")
	}

	// Format the final output as a Markdown document with the Mermaid diagram
	markdown := fmt.Sprintf("# %s Diagram\n\n```mermaid\n%s\n```\n", 
		strings.Title(diagramType), 
		mermaidCode)

	return markdown, nil
}

// extractMermaidCode extracts the Mermaid code from the AI response
func extractMermaidCode(content string) string {
	// Look for ```mermaid ... ``` blocks
	mermaidStart := strings.Index(content, "```mermaid")
	if mermaidStart == -1 {
		// Try without the mermaid tag
		mermaidStart = strings.Index(content, "```")
	}

	if mermaidStart == -1 {
		// If no code blocks, return the content as is (might be just the diagram code)
		return content
	}

	// Find the end of the code block
	contentAfterStart := content[mermaidStart+3:]
	mermaidEnd := strings.Index(contentAfterStart, "```")
	
	if mermaidEnd == -1 {
		// If no end marker, return everything after the start marker
		return contentAfterStart
	}

	// Extract the code between the markers
	if mermaidStart+3+mermaidEnd <= len(content) {
		// Skip the "```mermaid" part
		codeStart := mermaidStart + 3
		if strings.HasPrefix(contentAfterStart, "mermaid") {
			codeStart += len("mermaid")
		}
		
		// Remove any newline right after the start marker
		if codeStart < len(content) && content[codeStart] == '\n' {
			codeStart++
		}
		
		return strings.TrimSpace(content[codeStart : mermaidStart+3+mermaidEnd])
	}

	return ""
} 