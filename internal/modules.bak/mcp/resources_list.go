package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/internal/core"
)

// ResourcesListModule checks for exposed MCP resources
type ResourcesListModule struct {
	*BaseModule
}

// NewResourcesListModule creates a new resources list module
func NewResourcesListModule() *ResourcesListModule {
	return &ResourcesListModule{
		BaseModule: NewBaseModule(),
	}
}

// Name returns the module name
func (m *ResourcesListModule) Name() string {
	return "mcp/discovery/resources_list"
}

// Description returns the module description
func (m *ResourcesListModule) Description() string {
	return "Enumerate exposed MCP resources and identify sensitive data exposure"
}

// Type returns the module type
func (m *ResourcesListModule) Type() core.ModuleType {
	return core.NetworkScanning
}

// Info returns detailed module information
func (m *ResourcesListModule) Info() *core.ModuleInfo {
	return &core.ModuleInfo{
		Name:        m.Name(),
		Version:     "1.0.0",
		Author:      "Strigoi Team",
		Description: m.Description(),
		References: []string{
			"https://spec.modelcontextprotocol.io/specification/basic/resources/",
			"https://owasp.org/www-project-api-security/",
		},
		Targets: []string{
			"MCP Servers with resource endpoints",
			"Data exposure vulnerabilities",
		},
	}
}

// Check performs a vulnerability check
func (m *ResourcesListModule) Check() bool {
	ctx := context.Background()
	resp, err := m.SendMCPRequest(ctx, "resources/list", nil)
	if err != nil {
		return false
	}
	return resp.Error == nil
}

// Run executes the module
func (m *ResourcesListModule) Run() (*core.ModuleResult, error) {
	result := &core.ModuleResult{
		Success:  true,
		Findings: []core.SecurityFinding{},
		Metadata: make(map[string]interface{}),
	}

	startTime := time.Now()
	ctx := context.Background()

	// Send resources/list request
	resp, err := m.SendMCPRequest(ctx, "resources/list", nil)
	if err != nil {
		result.Success = false
		finding := core.SecurityFinding{
			ID:          "mcp-resources-connection-failed",
			Title:       "MCP Resources Connection Failed",
			Description: fmt.Sprintf("Failed to connect to MCP resources endpoint: %v", err),
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
		if resp.Error.Code == -32601 { // Method not found
			finding := core.SecurityFinding{
				ID:          "mcp-resources-not-implemented",
				Title:       "MCP Resources Not Implemented",
				Description: "Server does not implement resources/list endpoint",
				Severity:    core.Info,
			}
			result.Findings = append(result.Findings, finding)
		}
	} else {
		// Parse resources response
		var resourcesResponse struct {
			Resources []struct {
				URI         string `json:"uri"`
				Name        string `json:"name"`
				Description string `json:"description"`
				MimeType    string `json:"mimeType"`
			} `json:"resources"`
		}

		if err := json.Unmarshal(resp.Result, &resourcesResponse); err != nil {
			result.Success = false
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("failed to parse resources response: %w", err)
		}

		// Analyze each resource for security issues
		sensitiveResources := []string{}
		exposedResources := len(resourcesResponse.Resources)
		criticalFindings := 0

		for _, resource := range resourcesResponse.Resources {
			severity, issues := m.analyzeResourceSecurity(resource.URI, resource.Name, resource.Description, resource.MimeType)
			
			if len(issues) > 0 {
				if severity == core.Critical || severity == core.High {
					sensitiveResources = append(sensitiveResources, resource.URI)
				}
				
				if severity == core.Critical {
					criticalFindings++
				}
				
				finding := core.SecurityFinding{
					ID:          fmt.Sprintf("sensitive-resource-%s", m.sanitizeID(resource.URI)),
					Title:       fmt.Sprintf("Sensitive Resource Exposed: %s", resource.Name),
					Description: fmt.Sprintf("Resource '%s' may expose sensitive data", resource.URI),
					Severity:    severity,
					Evidence: []core.Evidence{
						{
							Type: "response",
							Data: map[string]interface{}{
								"uri":         resource.URI,
								"name":        resource.Name,
								"description": resource.Description,
								"mimeType":    resource.MimeType,
								"issues":      issues,
							},
							Description: "Resource details and security analysis",
						},
					},
					Remediation: &core.Remediation{
						Description: "Implement proper access controls for sensitive resources",
						Steps: []string{
							"Implement authentication before exposing resources",
							"Use principle of least privilege for resource access",
							"Encrypt sensitive data at rest and in transit",
							"Audit resource access patterns",
							"Consider data classification policies",
						},
					},
				}
				result.Findings = append(result.Findings, finding)
			}
		}

		// General finding about exposed resources
		if exposedResources > 0 {
			severity := core.Low
			if len(sensitiveResources) > 0 {
				severity = core.Medium
			}
			if criticalFindings > 0 {
				severity = core.High
			}

			finding := core.SecurityFinding{
				ID:          "mcp-resources-exposed",
				Title:       "MCP Resources Endpoint Exposed",
				Description: fmt.Sprintf("Found %d exposed resources, %d potentially sensitive", exposedResources, len(sensitiveResources)),
				Severity:    severity,
				Evidence: []core.Evidence{
					{
						Type: "response",
						Data: map[string]interface{}{
							"total_resources":     exposedResources,
							"sensitive_resources": sensitiveResources,
							"critical_findings":   criticalFindings,
						},
						Description: "Summary of exposed resources",
					},
				},
			}
			result.Findings = append(result.Findings, finding)
		}

		// Store metadata
		result.Metadata["exposed_resources"] = exposedResources
		result.Metadata["sensitive_resources"] = sensitiveResources
	}

	result.Duration = time.Since(startTime)
	result.Summary = m.summarizeFindings(result.Findings)
	
	return result, nil
}

// analyzeResourceSecurity checks for security issues in resources
func (m *ResourcesListModule) analyzeResourceSecurity(uri, name, description, mimeType string) (core.Severity, []string) {
	issues := []string{}
	severity := core.Low

	// Check URI patterns for sensitive paths
	uriLower := strings.ToLower(uri)
	
	// Critical: Credentials, keys, secrets
	criticalPatterns := []string{
		"password", "passwd", "credential", "secret", "key", "token",
		"private", "pem", "cert", "certificate", ".env", "config",
	}
	for _, pattern := range criticalPatterns {
		if strings.Contains(uriLower, pattern) {
			issues = append(issues, fmt.Sprintf("URI contains sensitive pattern: %s", pattern))
			severity = core.Critical
		}
	}

	// High: Database, internal paths, backups
	highPatterns := []string{
		"database", "db", "sql", "backup", "dump", "export",
		"internal", "admin", "root", "system",
	}
	for _, pattern := range highPatterns {
		if strings.Contains(uriLower, pattern) && severity < core.High {
			issues = append(issues, fmt.Sprintf("URI suggests sensitive data: %s", pattern))
			severity = core.High
		}
	}

	// Check file extensions
	if strings.HasSuffix(uriLower, ".sql") || strings.HasSuffix(uriLower, ".db") {
		issues = append(issues, "Database file exposed")
		severity = core.Critical
	}
	if strings.HasSuffix(uriLower, ".log") {
		issues = append(issues, "Log file exposed (may contain sensitive data)")
		if severity < core.High {
			severity = core.High
		}
	}
	if strings.HasSuffix(uriLower, ".bak") || strings.HasSuffix(uriLower, ".backup") {
		issues = append(issues, "Backup file exposed")
		if severity < core.High {
			severity = core.High
		}
	}

	// Check description for sensitive indicators
	descLower := strings.ToLower(description)
	if strings.Contains(descLower, "private") || strings.Contains(descLower, "internal") ||
	   strings.Contains(descLower, "confidential") || strings.Contains(descLower, "secret") {
		issues = append(issues, "Description indicates sensitive data")
		if severity < core.Medium {
			severity = core.Medium
		}
	}

	// Check MIME types
	if mimeType == "application/x-sqlite3" || mimeType == "application/sql" {
		issues = append(issues, "Database MIME type detected")
		severity = core.Critical
	}

	// Check for PII patterns
	piiPatterns := []string{
		"user", "customer", "client", "personal", "private",
		"email", "phone", "address", "ssn", "identity",
	}
	combinedText := strings.ToLower(uri + " " + name + " " + description)
	for _, pattern := range piiPatterns {
		if strings.Contains(combinedText, pattern) && len(issues) == 0 {
			issues = append(issues, "May contain personally identifiable information")
			if severity < core.Medium {
				severity = core.Medium
			}
			break
		}
	}

	return severity, issues
}

// sanitizeID creates a safe ID from URI
func (m *ResourcesListModule) sanitizeID(uri string) string {
	// Replace non-alphanumeric with hyphens
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, uri)
	
	// Limit length
	if len(safe) > 50 {
		safe = safe[:50]
	}
	
	return safe
}

// summarizeFindings creates a finding summary
func (m *ResourcesListModule) summarizeFindings(findings []core.SecurityFinding) *core.FindingSummary {
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