package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Nurozen/mermgen/parser"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Global rate limiter for API calls
var (
	lastAPICall         time.Time
	minTimeBetweenCalls = 3 * time.Second // Minimum time between API calls to avoid rate limiting
	maxRetries          = 3
)

// GenerateDiagrams generates various Mermaid diagrams from the parsed project data
func GenerateDiagrams(projectData *parser.RawProjectData) (map[string]string, error) {
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
func generateClassDiagram(projectData *parser.RawProjectData) (string, error) {
	// Prepare data for the AI prompt
	fileInfo := make([]map[string]interface{}, 0)

	// Counter to limit amount of data we send to the API
	const maxFiles = 10
	const maxContentLength = 1000000
	fileCount := 0

	for path, fileData := range projectData.Files {
		// Only process Go files and limit number of files
		if !strings.HasSuffix(path, ".go") || fileCount >= maxFiles {
			continue
		}

		// Truncate content if too large
		content := fileData.Content
		if len(content) > maxContentLength {
			content = content[:maxContentLength] + "... [truncated]"
		}

		// For parse tree, only include a summary to reduce size
		parseTreeSummary := "Parse tree available (truncated for size)"
		if len(fileData.ParseTree) < 100 {
			parseTreeSummary = fileData.ParseTree
		}

		fileInfo = append(fileInfo, map[string]interface{}{
			"path":        path,
			"packageName": fileData.PackageName,
			"content":     content,
			"parseTree":   parseTreeSummary,
		})

		fileCount++
	}

	//limit fileinfo to maxcontentlength
	jsonFile, err := json.Marshal(fileInfo)
	if err != nil {
		return "", fmt.Errorf("error marshaling fileInfo: %w", err)
	}

	jsonFileString := string(jsonFile)

	//limit jsonfile to maxcontentlength
	if len(jsonFileString) > maxContentLength {
		jsonFileString = jsonFileString[:maxContentLength] + "... [truncated]"
	}

	// Create AI prompt with clear instructions
	prompt := map[string]interface{}{
		"task":        "Generate a Mermaid class diagram that shows the structure and relationships between types in the Go codebase",
		"fileInfo":    jsonFileString,
		"explanation": "Create a class diagram showing the main types, their fields, methods, and relationships. Group related types together and focus on important relationships.",
	}

	// Call AI to generate diagram
	return callAI(prompt, "class")
}

// generatePackageDiagram creates a Mermaid package diagram from project data
func generatePackageDiagram(projectData *parser.RawProjectData) (string, error) {
	// Prepare data for the AI prompt
	fileInfo := make([]map[string]interface{}, 0)

	// Counter to limit amount of data we send to the API
	const maxFiles = 10
	const maxContentLength = 100000
	fileCount := 0

	for path, fileData := range projectData.Files {
		// Only process Go files and limit number of files
		if !strings.HasSuffix(path, ".go") || fileCount >= maxFiles {
			continue
		}

		// For package diagrams, we only need imports section
		// Find imports section to reduce content size
		content := extractImportsSection(fileData.Content)
		if len(content) > maxContentLength {
			content = content[:maxContentLength] + "... [truncated]"
		}

		fileInfo = append(fileInfo, map[string]interface{}{
			"path":        path,
			"packageName": fileData.PackageName,
			"content":     content,
		})

		fileCount++
	}

	// Create AI prompt with clear instructions
	prompt := map[string]interface{}{
		"task":        "Generate a Mermaid package diagram showing the structure and dependencies between packages in the Go codebase",
		"fileInfo":    fileInfo,
		"explanation": "Create a package diagram showing how packages depend on each other. Group related packages together and show the main dependencies between them.",
	}

	// Call AI to generate diagram
	return callAI(prompt, "package")
}

// generateSequenceDiagram creates a sample sequence diagram
func generateSequenceDiagram(projectData *parser.RawProjectData) (string, error) {
	// Prepare data for the AI prompt
	fileInfo := make([]map[string]interface{}, 0)

	// Counter to limit amount of data we send to the API
	const maxFiles = 10
	const maxContentLength = 100000
	fileCount := 0

	for path, fileData := range projectData.Files {
		// Only process Go files and limit number of files
		if !strings.HasSuffix(path, ".go") || fileCount >= maxFiles {
			continue
		}

		// Truncate content if too large
		content := fileData.Content
		if len(content) > maxContentLength {
			content = content[:maxContentLength] + "... [truncated]"
		}

		fileInfo = append(fileInfo, map[string]interface{}{
			"path":        path,
			"packageName": fileData.PackageName,
			"content":     content,
		})

		fileCount++
	}

	// Create AI prompt with clear instructions
	prompt := map[string]interface{}{
		"task":        "Generate a Mermaid sequence diagram that shows the flow of execution between key functions",
		"fileInfo":    fileInfo,
		"explanation": "Create a sequence diagram showing how the main components interact with each other. Focus on the most important function calls between different packages and types.",
	}

	// Call AI to generate diagram
	return callAI(prompt, "sequence")
}

// extractImportsSection extracts just the package and imports section from Go code
func extractImportsSection(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inImportBlock := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Always include package declaration
		if strings.HasPrefix(trimmedLine, "package ") {
			result = append(result, line)
			continue
		}

		// Track import blocks
		if strings.HasPrefix(trimmedLine, "import (") {
			inImportBlock = true
			result = append(result, line)
			continue
		}

		if inImportBlock {
			result = append(result, line)
			if trimmedLine == ")" {
				inImportBlock = false
				break // Stop after import block
			}
		} else if strings.HasPrefix(trimmedLine, "import ") {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// callAI calls Google's Generative AI service to generate a Mermaid diagram
func callAI(prompt map[string]interface{}, diagramType string) (string, error) {
	// Get API key from environment
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		fmt.Println("GOOGLE_API_KEY not set, using fallback diagram")
		return createFallbackDiagram(diagramType), nil
	}

	// Convert prompt to JSON
	promptJSON, err := json.MarshalIndent(prompt, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling prompt: %v, using fallback diagram\n", err)
		return createFallbackDiagram(diagramType), nil
	}

	// Include detailed instructions based on diagram type
	var systemPrompt string
	switch diagramType {
	case "class":
		systemPrompt = "You are an expert in Go programming and Mermaid diagrams. " +
			"Your task is to analyze the Go code structure provided and generate a comprehensive Mermaid class diagram. " +
			"Focus on showing the relationships between types, their fields, and methods. " +
			"If the data looks incomplete, do your best to create a meaningful diagram with what's available, be as verbose as possible. " +
			"Only return the Mermaid diagram code, nothing else."
	case "package":
		systemPrompt = "You are an expert in Go programming and Mermaid diagrams. " +
			"Your task is to analyze the Go code structure provided and generate a Mermaid package diagram showing " +
			"dependencies between packages. If the data looks incomplete, create a in depth package dependency diagram " +
			"with the information available. Be as verbose as possible. Only return the Mermaid diagram code, nothing else."
	case "sequence":
		systemPrompt = "You are an expert in Go programming and Mermaid diagrams. " +
			"Your task is to create a Mermaid sequence diagram showing the flow of execution between key components " +
			"based on the Go structure provided. If the data looks incomplete, create a in depth sequence diagram " +
			"showing common interactions. Be as verbose as possible. Only return the Mermaid diagram code, nothing else."
	default:
		systemPrompt = "You are an expert in Go programming and Mermaid diagrams. " +
			"Your task is to analyze Go code structure and generate a Mermaid diagram that " +
			"accurately represents the code, be as verbose as possible. Only return the Mermaid diagram code, nothing else."
	}

	// Create a more detailed prompt that includes explicit instructions
	promptStr := fmt.Sprintf("Generate a Mermaid %s diagram based on the following Go code structure:\n\n%s\n\n"+
		"Even if you think the data is incomplete, create the best diagram possible with what's provided. Prioritize human readability. Always double check the output for mermaid syntax errors.",
		diagramType, string(promptJSON))

	// Create client with API key
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		fmt.Printf("Error creating client: %v, using fallback diagram\n", err)
		return createFallbackDiagram(diagramType), nil
	}
	defer client.Close()

	// Use Gemini model
	model := client.GenerativeModel("gemini-2.0-flash")

	// Set system instruction
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text(systemPrompt),
		},
	}

	// Configure temperature
	temperature := float32(0.2)
	model.Temperature = &temperature

	// Try API call with retries and rate limiting
	var response *genai.GenerateContentResponse
	var apiError error

	for retry := 0; retry < maxRetries; retry++ {
		// Check if we need to wait before making another API call (rate limiting)
		timeSinceLastCall := time.Since(lastAPICall)
		if timeSinceLastCall < minTimeBetweenCalls {
			waitTime := minTimeBetweenCalls - timeSinceLastCall
			fmt.Printf("Rate limiting: waiting %v before next API call\n", waitTime)
			time.Sleep(waitTime)
		}

		// Update last API call time
		lastAPICall = time.Now()

		// Create request and send
		response, err = model.GenerateContent(ctx, genai.Text(promptStr))
		if err != nil {
			apiError = fmt.Errorf("API error: %w", err)

			// For rate limiting errors, wait longer and retry
			if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "429") {
				waitTime := time.Duration(4<<retry) * time.Second
				fmt.Printf("Rate limit exceeded. Retrying in %v (retry %d/%d)...\n",
					waitTime, retry+1, maxRetries)
				time.Sleep(waitTime)
				continue
			}

			// For other errors, try again
			continue
		}

		// If we got here, request was successful
		apiError = nil
		break
	}

	// If we still have an error after all retries, use fallback
	if apiError != nil {
		fmt.Printf("All API retries failed: %v, using fallback diagram\n", apiError)
		return createFallbackDiagram(diagramType), nil
	}

	// Check if we got a valid response
	if response == nil || len(response.Candidates) == 0 ||
		len(response.Candidates[0].Content.Parts) == 0 {
		fmt.Println("Invalid or empty response from API, using fallback diagram")
		return createFallbackDiagram(diagramType), nil
	}

	// Extract text from response
	content, ok := response.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		fmt.Println("Could not parse response content, using fallback diagram")
		return createFallbackDiagram(diagramType), nil
	}

	// Extract Mermaid code from content
	mermaidCode := extractMermaidCode(string(content))
	if mermaidCode == "" {
		fmt.Println("No Mermaid code found in API response, using fallback diagram")
		return createFallbackDiagram(diagramType), nil
	}

	// Format the final output as a Markdown document with the Mermaid diagram
	markdown := fmt.Sprintf("# %s Diagram\n\n```mermaid\n%s\n```\n",
		strings.Title(diagramType),
		mermaidCode)

	return markdown, nil
}

// createFallbackDiagram generates a simple default diagram when the AI service fails
func createFallbackDiagram(diagramType string) string {
	var mermaidCode string

	switch diagramType {
	case "class":
		mermaidCode = `classDiagram
    class Parser {
        +ParseGoProject(path) RawProjectData
    }
    class RawProjectData {
        +Files map[string]*FileData
    }
    class FileData {
        +Content string
        +PackageName string
        +ParseTree string
    }
    class Generator {
        +GenerateDiagrams(data) map[string]string
    }
    RawProjectData o-- FileData
    Parser ..> RawProjectData
    Generator ..> RawProjectData`
	case "package":
		mermaidCode = `flowchart LR
    main[main] --> parser[parser]
    main --> generator[generator]
    main --> github[github]
    generator --> parser`
	case "sequence":
		mermaidCode = `sequenceDiagram
    participant Main
    participant Parser
    participant Generator
    
    Main->>Parser: ParseGoProject(path)
    Parser-->>Main: projectData
    Main->>Generator: GenerateDiagrams(projectData)
    Generator-->>Main: diagrams`
	default:
		mermaidCode = `graph TD
    A[Start] --> B[Process Data]
    B --> C[Generate Output]`
	}

	// Format as markdown
	markdown := fmt.Sprintf("# %s Diagram\n\n```mermaid\n%s\n```\n",
		strings.Title(diagramType),
		mermaidCode)

	return markdown
}

// Helper function to get keys from a map
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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
