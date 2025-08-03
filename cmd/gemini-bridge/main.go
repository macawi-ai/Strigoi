package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// A2ARequest represents a request from Claude to Gemini
type A2ARequest struct {
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Operation string                 `json:"operation"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

// A2AResponse represents Gemini's response
type A2AResponse struct {
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Status    string                 `json:"status"`
	Result    interface{}            `json:"result"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// GeminiBridge manages communication with Gemini
type GeminiBridge struct {
	contextDir   string
	geminiCmd    string
	maxContext   int
	contextCache map[string]string
}

// NewGeminiBridge creates a new bridge instance
func NewGeminiBridge() *GeminiBridge {
	homeDir, _ := os.UserHomeDir()
	contextDir := filepath.Join(homeDir, ".strigoi", "gemini-context")
	os.MkdirAll(contextDir, 0755)
	
	return &GeminiBridge{
		contextDir:   contextDir,
		geminiCmd:    "gemini", // Assumes gemini-cli is in PATH
		maxContext:   1000000,  // 1M tokens
		contextCache: make(map[string]string),
	}
}

// QueryGemini sends a query with context to Gemini
func (gb *GeminiBridge) QueryGemini(ctx context.Context, prompt, contextKey string) (*A2AResponse, error) {
	// Prepare context file
	contextFile := filepath.Join(gb.contextDir, fmt.Sprintf("%s_context.txt", contextKey))
	
	// Build context from cache and files
	fullContext := gb.buildContext(contextKey)
	
	// Write context to temp file
	if err := os.WriteFile(contextFile, []byte(fullContext), 0644); err != nil {
		return nil, fmt.Errorf("failed to write context: %w", err)
	}
	
	// Prepare Gemini command
	cmd := exec.CommandContext(ctx, gb.geminiCmd,
		"--context-file", contextFile,
		"--prompt", prompt,
		"--format", "json",
	)
	
	// Execute and capture output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gemini execution failed: %w", err)
	}
	
	// Parse response
	response := &A2AResponse{
		Type:      "a2a_response",
		From:      "gemini",
		To:        "claude",
		Status:    "success",
		Result:    string(output),
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"context_size": len(fullContext),
			"context_key":  contextKey,
		},
	}
	
	return response, nil
}

// StoreContext saves context for later use
func (gb *GeminiBridge) StoreContext(key string, data string) error {
	contextFile := filepath.Join(gb.contextDir, fmt.Sprintf("%s.context", key))
	return os.WriteFile(contextFile, []byte(data), 0644)
}

// AnalyzeCodebase performs deep analysis using Gemini's large context
func (gb *GeminiBridge) AnalyzeCodebase(projectPath, query string) (*A2AResponse, error) {
	// Collect all Go files
	var codebaseContent strings.Builder
	
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Include Go files and documentation
		if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".md") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			
			codebaseContent.WriteString(fmt.Sprintf("\n\n=== File: %s ===\n", path))
			codebaseContent.Write(content)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk codebase: %w", err)
	}
	
	// Store as context
	gb.StoreContext("codebase_analysis", codebaseContent.String())
	
	// Query Gemini with the analysis request
	prompt := fmt.Sprintf(`
Analyze the provided codebase with the following query:
%s

Please provide:
1. Direct answer to the query
2. Supporting evidence from the code
3. Potential concerns or improvements
4. Architectural insights

Format your response as structured JSON.
`, query)
	
	return gb.QueryGemini(context.Background(), prompt, "codebase_analysis")
}

// buildContext assembles context from multiple sources
func (gb *GeminiBridge) buildContext(key string) string {
	var context strings.Builder
	
	// Add persistent context
	persistentFile := filepath.Join(gb.contextDir, "persistent.context")
	if persistent, err := os.ReadFile(persistentFile); err == nil {
		context.Write(persistent)
		context.WriteString("\n\n")
	}
	
	// Add key-specific context
	keyFile := filepath.Join(gb.contextDir, fmt.Sprintf("%s.context", key))
	if keyContent, err := os.ReadFile(keyFile); err == nil {
		context.Write(keyContent)
		context.WriteString("\n\n")
	}
	
	// Add cybernetic principles
	context.WriteString(`
=== CYBERNETIC ECOLOGY PRINCIPLES ===
- Think in systems and relationships, not just individual functions
- Design for information flow between components
- Create feedback loops that improve system behavior
- Build recursive patterns that work at multiple scales
- Every significant system should include self-regulating mechanisms

`)
	
	return context.String()
}

// Interactive mode for testing
func (gb *GeminiBridge) InteractiveMode() {
	scanner := bufio.NewScanner(os.Stdin)
	
	fmt.Println("🤖 Gemini A2A Bridge Interactive Mode")
	fmt.Println("Commands: query, analyze, store, exit")
	fmt.Println()
	
	for {
		fmt.Print("gemini> ")
		if !scanner.Scan() {
			break
		}
		
		input := scanner.Text()
		parts := strings.Fields(input)
		
		if len(parts) == 0 {
			continue
		}
		
		switch parts[0] {
		case "query":
			if len(parts) < 2 {
				fmt.Println("Usage: query <prompt>")
				continue
			}
			prompt := strings.Join(parts[1:], " ")
			resp, err := gb.QueryGemini(context.Background(), prompt, "interactive")
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Gemini says: %s\n", resp.Result)
			}
			
		case "analyze":
			if len(parts) < 3 {
				fmt.Println("Usage: analyze <path> <query>")
				continue
			}
			path := parts[1]
			query := strings.Join(parts[2:], " ")
			resp, err := gb.AnalyzeCodebase(path, query)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Analysis result: %s\n", resp.Result)
			}
			
		case "store":
			if len(parts) < 3 {
				fmt.Println("Usage: store <key> <data>")
				continue
			}
			key := parts[1]
			data := strings.Join(parts[2:], " ")
			if err := gb.StoreContext(key, data); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Stored context for key: %s\n", key)
			}
			
		case "exit":
			fmt.Println("Goodbye!")
			return
			
		default:
			fmt.Printf("Unknown command: %s\n", parts[0])
		}
	}
}

func main() {
	bridge := NewGeminiBridge()
	
	// Check if Gemini CLI is available
	if _, err := exec.LookPath("gemini"); err != nil {
		log.Println("Warning: gemini CLI not found in PATH")
		log.Println("This is a prototype for A2A communication")
	}
	
	// Start interactive mode
	bridge.InteractiveMode()
}