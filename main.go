package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Nurozen/mermgen/generator"
	"github.com/Nurozen/mermgen/github"
	"github.com/Nurozen/mermgen/parser"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env.local"); err != nil {
		log.Fatal("Error loading .env.local file")
	}

	// Define command line arguments
	repoURL := flag.String("repo", "", "GitHub repository URL (e.g., github.com/user/repo)")
	outputDir := flag.String("output", "diagrams", "Output directory for generated diagrams")
	flag.Parse()

	if *repoURL == "" {
		fmt.Println("Please provide a GitHub repository URL with -repo")
		flag.Usage()
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	err := os.MkdirAll(*outputDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Clone the repository
	fmt.Printf("Cloning repository: %s\n", *repoURL)
	repoPath, err := github.CloneRepository(*repoURL)
	if err != nil {
		log.Fatalf("Failed to clone repository: %v", err)
	}
	//defer os.RemoveAll(repoPath) // Clean up the cloned repo after we're done

	// Parse the Go code with tree-sitter
	fmt.Println("Parsing Go code...")
	parsedData, err := parser.ParseGoProject(repoPath)
	if err != nil {
		log.Fatalf("Failed to parse Go code: %v", err)
	}

	// Generate Mermaid diagrams
	fmt.Println("Generating Mermaid diagrams...")
	diagrams, err := generator.GenerateDiagrams(parsedData)
	if err != nil {
		log.Fatalf("Failed to generate diagrams: %v", err)
	}

	// Save diagrams to output directory
	for name, content := range diagrams {
		filePath := filepath.Join(*outputDir, name+".md")
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			log.Printf("Error writing diagram %s: %v", name, err)
		}
	}

	fmt.Printf("Generated %d diagrams in %s\n", len(diagrams), *outputDir)
}
