package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/internal/core"
)

// ToolsListModule checks for exposed MCP tools
type ToolsListModule struct {
	*BaseModule
}

// NewToolsListModule creates a new tools list module
func NewToolsListModule() *ToolsListModule {
	return &ToolsListModule{
		BaseModule: NewBaseModule(),
	}
}

// Name returns the module name
func (m *ToolsListModule) Name() string {
	return "mcp/discovery/tools_list"
}

// Description returns the module description
func (m *ToolsListModule) Description() string {
	return "Enumerate exposed MCP tools and check for dangerous capabilities"
}

// Type returns the module type
func (m *ToolsListModule) Type() core.ModuleType {
	return core.NetworkScanning
}

// Info returns detailed module information
func (m *ToolsListModule) Info() *core.ModuleInfo {
	return &core.ModuleInfo{
		Name:        m.Name(),
		Version:     "1.0.0",
		Author:      "Strigoi Team",
		Description: m.Description(),
		References: []string{
			"https://github.com/modelcontextprotocol/specification",
			"https://spec.modelcontextprotocol.io/specification/basic/tools/",
		},
		Targets: []string{
			"MCP Servers",
			"Model Context Protocol endpoints",
		},
	}
}

// Check performs a vulnerability check
func (m *ToolsListModule) Check() bool {
	// Quick check if tools/list is exposed
	ctx := context.Background()
	resp, err := m.SendMCPRequest(ctx, "tools/list", nil)
	if err != nil {
		return false
	}
	return resp.Error == nil
}

// Run executes the module
func (m *ToolsListModule) Run() (*core.ModuleResult, error) {
	result := &core.ModuleResult{
		Success:  true,
		Findings: []core.SecurityFinding{},
		Metadata: make(map[string]interface{}),
	}

	startTime := time.Now()
	ctx := context.Background()

	// Send tools/list request
	resp, err := m.SendMCPRequest(ctx, "tools/list", nil)
	if err != nil {
		result.Success = false
		finding := core.SecurityFinding{
			ID:          "mcp-connection-failed",
			Title:       "MCP Connection Failed",
			Description: fmt.Sprintf("Failed to connect to MCP server: %v", err),
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
		finding := core.SecurityFinding{
			ID:          "mcp-tools-list-error",
			Title:       "MCP Tools List Error",
			Description: fmt.Sprintf("Server returned error: %s", resp.Error.Message),
			Severity:    core.Info,
			Evidence: []core.Evidence{
				{
					Type: "response",
					Data: resp.Error,
					Description: "Error response from server",
				},
			},
		}
		result.Findings = append(result.Findings, finding)
	} else {
		// Parse tools response
		var toolsResponse struct {
			Tools []struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				InputSchema json.RawMessage `json:"inputSchema"`
			} `json:"tools"`
		}

		if err := json.Unmarshal(resp.Result, &toolsResponse); err != nil {
			result.Success = false
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("failed to parse tools response: %w", err)
		}

		// Analyze each tool for security issues
		dangerousTools := []string{}
		exposedTools := len(toolsResponse.Tools)

		for _, tool := range toolsResponse.Tools {
			// Check for dangerous tool patterns
			if m.isDangerousTool(tool.Name, tool.Description) {
				dangerousTools = append(dangerousTools, tool.Name)
				
				finding := core.SecurityFinding{
					ID:          fmt.Sprintf("dangerous-tool-%s", tool.Name),
					Title:       fmt.Sprintf("Dangerous Tool Exposed: %s", tool.Name),
					Description: fmt.Sprintf("The tool '%s' may allow dangerous operations: %s", tool.Name, tool.Description),
					Severity:    m.getToolSeverity(tool.Name),
					Evidence: []core.Evidence{
						{
							Type: "response",
							Data: map[string]interface{}{
								"name":        tool.Name,
								"description": tool.Description,
								"schema":      string(tool.InputSchema),
							},
							Description: "Tool definition from server",
						},
					},
					Remediation: &core.Remediation{
						Description: "Review if this tool should be exposed publicly. Consider implementing authentication and access controls.",
						Steps: []string{
							"Implement authentication for MCP endpoints",
							"Use capability-based access control",
							"Audit tool permissions and scope",
							"Consider removing or restricting dangerous tools",
						},
					},
				}
				result.Findings = append(result.Findings, finding)
			}
		}

		// General finding about exposed tools
		if exposedTools > 0 {
			severity := core.Low
			if len(dangerousTools) > 0 {
				severity = core.Medium
			}
			if len(dangerousTools) > 3 {
				severity = core.High
			}

			finding := core.SecurityFinding{
				ID:          "mcp-tools-exposed",
				Title:       "MCP Tools Endpoint Exposed",
				Description: fmt.Sprintf("Found %d exposed tools, %d potentially dangerous", exposedTools, len(dangerousTools)),
				Severity:    severity,
				Evidence: []core.Evidence{
					{
						Type: "response",
						Data: map[string]interface{}{
							"total_tools":     exposedTools,
							"dangerous_tools": dangerousTools,
							"all_tools":       m.extractToolNames(toolsResponse.Tools),
						},
						Description: "Summary of exposed tools",
					},
				},
			}
			result.Findings = append(result.Findings, finding)
		}

		// Store metadata
		result.Metadata["exposed_tools"] = exposedTools
		result.Metadata["dangerous_tools"] = dangerousTools
	}

	result.Duration = time.Since(startTime)
	result.Summary = m.summarizeFindings(result.Findings)
	
	return result, nil
}

// isDangerousTool checks if a tool name/description indicates dangerous capabilities
func (m *ToolsListModule) isDangerousTool(name, description string) bool {
	dangerousPatterns := []string{
		"exec", "execute", "shell", "cmd", "command",
		"write", "delete", "remove", "modify",
		"system", "process", "spawn",
		"eval", "compile", "interpret",
		"file", "path", "directory",
		"network", "request", "fetch",
		"database", "query", "sql",
		"credential", "password", "secret",
	}

	combined := strings.ToLower(name + " " + description)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(combined, pattern) {
			return true
		}
	}

	return false
}

// getToolSeverity determines severity based on tool capabilities
func (m *ToolsListModule) getToolSeverity(toolName string) core.Severity {
	toolLower := strings.ToLower(toolName)
	
	// Critical severity patterns
	if strings.Contains(toolLower, "exec") || strings.Contains(toolLower, "shell") {
		return core.Critical
	}
	
	// High severity patterns
	if strings.Contains(toolLower, "write") || strings.Contains(toolLower, "delete") {
		return core.High
	}
	
	// Medium severity patterns
	if strings.Contains(toolLower, "read") || strings.Contains(toolLower, "fetch") {
		return core.Medium
	}
	
	return core.Low
}

// extractToolNames gets just the tool names for summary
func (m *ToolsListModule) extractToolNames(tools []struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	return names
}

// summarizeFindings creates a finding summary
func (m *ToolsListModule) summarizeFindings(findings []core.SecurityFinding) *core.FindingSummary {
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