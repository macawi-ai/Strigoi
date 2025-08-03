package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/internal/core"
)

// PromptsListModule checks for prompt injection vulnerabilities
type PromptsListModule struct {
	*BaseModule
}

// NewPromptsListModule creates a new prompts list module
func NewPromptsListModule() *PromptsListModule {
	return &PromptsListModule{
		BaseModule: NewBaseModule(),
	}
}

// Name returns the module name
func (m *PromptsListModule) Name() string {
	return "mcp/discovery/prompts_list"
}

// Description returns the module description
func (m *PromptsListModule) Description() string {
	return "Enumerate MCP prompts and test for injection vulnerabilities"
}

// Type returns the module type
func (m *PromptsListModule) Type() core.ModuleType {
	return core.NetworkScanning
}

// Info returns detailed module information
func (m *PromptsListModule) Info() *core.ModuleInfo {
	return &core.ModuleInfo{
		Name:        m.Name(),
		Version:     "1.0.0",
		Author:      "Strigoi Team",
		Description: m.Description(),
		References: []string{
			"https://spec.modelcontextprotocol.io/specification/basic/prompts/",
			"https://owasp.org/www-community/attacks/Prompt_Injection",
		},
		Targets: []string{
			"MCP Servers with prompt endpoints",
			"AI/LLM integration points",
		},
	}
}

// Check performs a vulnerability check
func (m *PromptsListModule) Check() bool {
	ctx := context.Background()
	resp, err := m.SendMCPRequest(ctx, "prompts/list", nil)
	if err != nil {
		return false
	}
	return resp.Error == nil
}

// Run executes the module
func (m *PromptsListModule) Run() (*core.ModuleResult, error) {
	result := &core.ModuleResult{
		Success:  true,
		Findings: []core.SecurityFinding{},
		Metadata: make(map[string]interface{}),
	}

	startTime := time.Now()
	ctx := context.Background()

	// Send prompts/list request
	resp, err := m.SendMCPRequest(ctx, "prompts/list", nil)
	if err != nil {
		result.Success = false
		finding := core.SecurityFinding{
			ID:          "mcp-prompts-connection-failed",
			Title:       "MCP Prompts Connection Failed",
			Description: fmt.Sprintf("Failed to connect to MCP prompts endpoint: %v", err),
			Severity:    core.Info,
			Evidence: []core.Evidence{
				{
					Type:        "network",
					Data:        err.Error(),
					Description: "Connection error details",
				},
			},
		}
		result.Findings = append(result.Findings, finding)
		result.Duration = time.Since(startTime)
		result.Summary = m.summarizeFindings(result.Findings)
		return result, nil
	}

	// Check for error response
	if resp.Error != nil {
		// prompts/list not implemented is info, not a vulnerability
		severity := core.Info
		if resp.Error.Code == -32601 { // Method not found
			finding := core.SecurityFinding{
				ID:          "mcp-prompts-not-implemented",
				Title:       "MCP Prompts Not Implemented",
				Description: "Server does not implement prompts/list endpoint",
				Severity:    severity,
			}
			result.Findings = append(result.Findings, finding)
		}
	} else {
		// Parse prompts response
		var promptsResponse struct {
			Prompts []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Arguments   []struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					Required    bool   `json:"required"`
				} `json:"arguments"`
			} `json:"prompts"`
		}

		if err := json.Unmarshal(resp.Result, &promptsResponse); err != nil {
			result.Success = false
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("failed to parse prompts response: %w", err)
		}

		// Analyze each prompt for injection vulnerabilities
		vulnerablePrompts := []string{}
		exposedPrompts := len(promptsResponse.Prompts)

		for _, prompt := range promptsResponse.Prompts {
			vulnDetails := m.analyzePromptVulnerabilities(prompt.Name, prompt.Description, prompt.Arguments)
			
			if len(vulnDetails) > 0 {
				vulnerablePrompts = append(vulnerablePrompts, prompt.Name)
				
				severity := m.getPromptSeverity(vulnDetails)
				
				finding := core.SecurityFinding{
					ID:          fmt.Sprintf("prompt-injection-risk-%s", prompt.Name),
					Title:       fmt.Sprintf("Potential Prompt Injection: %s", prompt.Name),
					Description: fmt.Sprintf("The prompt '%s' may be vulnerable to injection attacks", prompt.Name),
					Severity:    severity,
					Evidence: []core.Evidence{
						{
							Type: "response",
							Data: map[string]interface{}{
								"name":            prompt.Name,
								"description":     prompt.Description,
								"arguments":       prompt.Arguments,
								"vulnerabilities": vulnDetails,
							},
							Description: "Prompt definition and vulnerability analysis",
						},
					},
					Remediation: &core.Remediation{
						Description: "Implement prompt injection defenses",
						Steps: []string{
							"Validate and sanitize all user inputs",
							"Use structured prompt templates with clear boundaries",
							"Implement prompt injection detection filters",
							"Consider using prompt sandboxing techniques",
							"Monitor for unusual prompt patterns",
						},
					},
				}
				result.Findings = append(result.Findings, finding)
			}
		}

		// General finding about exposed prompts
		if exposedPrompts > 0 {
			severity := core.Low
			if len(vulnerablePrompts) > 0 {
				severity = core.Medium
			}
			if len(vulnerablePrompts) > 2 {
				severity = core.High
			}

			finding := core.SecurityFinding{
				ID:          "mcp-prompts-exposed",
				Title:       "MCP Prompts Endpoint Exposed",
				Description: fmt.Sprintf("Found %d exposed prompts, %d potentially vulnerable to injection", exposedPrompts, len(vulnerablePrompts)),
				Severity:    severity,
				Evidence: []core.Evidence{
					{
						Type: "response",
						Data: map[string]interface{}{
							"total_prompts":      exposedPrompts,
							"vulnerable_prompts": vulnerablePrompts,
							"all_prompts":        m.extractPromptNames(promptsResponse.Prompts),
						},
						Description: "Summary of exposed prompts",
					},
				},
			}
			result.Findings = append(result.Findings, finding)
		}

		// Store metadata
		result.Metadata["exposed_prompts"] = exposedPrompts
		result.Metadata["vulnerable_prompts"] = vulnerablePrompts
	}

	result.Duration = time.Since(startTime)
	result.Summary = m.summarizeFindings(result.Findings)
	
	return result, nil
}

// analyzePromptVulnerabilities checks for injection vulnerabilities
func (m *PromptsListModule) analyzePromptVulnerabilities(name, description string, args []struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}) []string {
	vulnerabilities := []string{}

	// Check for dangerous patterns in prompt name/description
	combined := strings.ToLower(name + " " + description)
	
	// Direct command execution patterns
	if strings.Contains(combined, "execute") || strings.Contains(combined, "system") || 
	   strings.Contains(combined, "command") || strings.Contains(combined, "shell") {
		vulnerabilities = append(vulnerabilities, "May allow command execution via prompts")
	}

	// Code execution patterns
	if strings.Contains(combined, "code") || strings.Contains(combined, "eval") || 
	   strings.Contains(combined, "script") || strings.Contains(combined, "function") {
		vulnerabilities = append(vulnerabilities, "May allow code execution through prompts")
	}

	// File system access patterns
	if strings.Contains(combined, "file") || strings.Contains(combined, "path") || 
	   strings.Contains(combined, "directory") || strings.Contains(combined, "read") {
		vulnerabilities = append(vulnerabilities, "May expose file system through prompts")
	}

	// Data extraction patterns
	if strings.Contains(combined, "database") || strings.Contains(combined, "query") || 
	   strings.Contains(combined, "extract") || strings.Contains(combined, "dump") {
		vulnerabilities = append(vulnerabilities, "May allow data extraction via prompts")
	}

	// Check for unvalidated user input arguments
	hasUserInput := false
	for _, arg := range args {
		argLower := strings.ToLower(arg.Name + " " + arg.Description)
		if strings.Contains(argLower, "user") || strings.Contains(argLower, "input") ||
		   strings.Contains(argLower, "text") || strings.Contains(argLower, "content") ||
		   strings.Contains(argLower, "message") || strings.Contains(argLower, "prompt") {
			hasUserInput = true
			break
		}
	}

	if hasUserInput {
		vulnerabilities = append(vulnerabilities, "Accepts user-controlled input without clear validation")
	}

	// Check for template/format string patterns
	if strings.Contains(combined, "template") || strings.Contains(combined, "format") ||
	   strings.Contains(combined, "replace") || strings.Contains(combined, "substitute") {
		vulnerabilities = append(vulnerabilities, "May be vulnerable to template injection")
	}

	return vulnerabilities
}

// getPromptSeverity determines severity based on vulnerabilities
func (m *PromptsListModule) getPromptSeverity(vulnerabilities []string) core.Severity {
	// Critical if command/code execution is possible
	for _, vuln := range vulnerabilities {
		if strings.Contains(vuln, "command execution") || strings.Contains(vuln, "code execution") {
			return core.Critical
		}
	}
	
	// High if file system or data access
	for _, vuln := range vulnerabilities {
		if strings.Contains(vuln, "file system") || strings.Contains(vuln, "data extraction") {
			return core.High
		}
	}
	
	// Medium for other injection risks
	if len(vulnerabilities) > 0 {
		return core.Medium
	}
	
	return core.Low
}

// extractPromptNames gets just the prompt names for summary
func (m *PromptsListModule) extractPromptNames(prompts []struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Arguments   []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Required    bool   `json:"required"`
	} `json:"arguments"`
}) []string {
	names := make([]string, len(prompts))
	for i, prompt := range prompts {
		names[i] = prompt.Name
	}
	return names
}

// summarizeFindings creates a finding summary
func (m *PromptsListModule) summarizeFindings(findings []core.SecurityFinding) *core.FindingSummary {
	summary := &core.FindingSummary{
		Total:    len(findings),
		ByModule: make(map[string]int),
	}

	for _, finding := range findings {
		switch finding.Severity {
		case core.Critical:
			summary.Critical++
		case core.High:
			summary.High++
		case core.Medium:
			summary.Medium++
		case core.Low:
			summary.Low++
		case core.Info:
			summary.Info++
		}
	}

	summary.ByModule[m.Name()] = len(findings)
	
	return summary
}