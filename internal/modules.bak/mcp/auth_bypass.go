package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/internal/core"
)

// AuthBypassModule tests MCP authentication mechanisms
type AuthBypassModule struct {
	*BaseModule
}

// NewAuthBypassModule creates a new auth bypass module
func NewAuthBypassModule() *AuthBypassModule {
	return &AuthBypassModule{
		BaseModule: NewBaseModule(),
	}
}

// Name returns the module name
func (m *AuthBypassModule) Name() string {
	return "mcp/auth/bypass"
}

// Description returns the module description
func (m *AuthBypassModule) Description() string {
	return "Test MCP authentication mechanisms for bypass vulnerabilities"
}

// Type returns the module type
func (m *AuthBypassModule) Type() core.ModuleType {
	return core.NetworkScanning
}

// Info returns detailed module information
func (m *AuthBypassModule) Info() *core.ModuleInfo {
	return &core.ModuleInfo{
		Name:        m.Name(),
		Version:     "1.0.0",
		Author:      "Strigoi Team",
		Description: m.Description(),
		References: []string{
			"https://spec.modelcontextprotocol.io/specification/architecture/#security-considerations",
			"https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/04-Authentication_Testing/",
		},
		Targets: []string{
			"MCP Servers with authentication",
			"OAuth/token-based auth endpoints",
		},
	}
}

// Check performs a vulnerability check
func (m *AuthBypassModule) Check() bool {
	// Try multiple auth-related endpoints
	ctx := context.Background()
	
	// Check if server requires auth
	resp, err := m.SendMCPRequest(ctx, "tools/list", nil)
	if err != nil {
		return false
	}
	
	// If we get an auth error, there's something to test
	if resp.Error != nil && (resp.Error.Code == -32603 || strings.Contains(strings.ToLower(resp.Error.Message), "auth")) {
		return true
	}
	
	// Also vulnerable if no auth at all
	return resp.Error == nil
}

// Run executes the module
func (m *AuthBypassModule) Run() (*core.ModuleResult, error) {
	result := &core.ModuleResult{
		Success:  true,
		Findings: []core.SecurityFinding{},
		Metadata: make(map[string]interface{}),
	}

	startTime := time.Now()
	ctx := context.Background()

	// Test various authentication bypass techniques
	authTests := []struct {
		name        string
		description string
		test        func(context.Context) (*core.SecurityFinding, error)
	}{
		{
			name:        "no-auth-required",
			description: "Check if authentication is required",
			test:        m.testNoAuth,
		},
		{
			name:        "empty-token",
			description: "Test with empty authentication token",
			test:        m.testEmptyToken,
		},
		{
			name:        "default-credentials",
			description: "Test common default credentials",
			test:        m.testDefaultCredentials,
		},
		{
			name:        "jwt-none-algorithm",
			description: "Test JWT 'none' algorithm bypass",
			test:        m.testJWTNoneAlgorithm,
		},
		{
			name:        "auth-header-injection",
			description: "Test authentication header injection",
			test:        m.testAuthHeaderInjection,
		},
	}

	vulnerableTests := 0
	
	for _, test := range authTests {
		finding, err := test.test(ctx)
		if err != nil {
			// Test error, log but continue
			result.Metadata[test.name+"_error"] = err.Error()
			continue
		}
		
		if finding != nil {
			result.Findings = append(result.Findings, *finding)
			if finding.Severity == core.Critical || finding.Severity == core.High {
				vulnerableTests++
			}
		}
	}

	// Summary finding
	if vulnerableTests > 0 {
		severity := core.High
		if vulnerableTests > 2 {
			severity = core.Critical
		}
		
		finding := core.SecurityFinding{
			ID:          "mcp-auth-vulnerable",
			Title:       "MCP Authentication Vulnerabilities Found",
			Description: fmt.Sprintf("Found %d authentication bypass vulnerabilities", vulnerableTests),
			Severity:    severity,
			Remediation: &core.Remediation{
				Description: "Implement secure authentication mechanisms",
				Steps: []string{
					"Enforce authentication on all MCP endpoints",
					"Use secure token generation (avoid predictable tokens)",
					"Implement proper JWT validation (check algorithm)",
					"Use rate limiting to prevent brute force",
					"Log and monitor authentication failures",
				},
			},
		}
		result.Findings = append(result.Findings, finding)
	}

	result.Duration = time.Since(startTime)
	result.Summary = m.summarizeFindings(result.Findings)
	
	return result, nil
}

// testNoAuth checks if endpoints work without authentication
func (m *AuthBypassModule) testNoAuth(ctx context.Context) (*core.SecurityFinding, error) {
	// Test critical endpoints without auth
	endpoints := []string{"tools/list", "prompts/list", "resources/list"}
	exposedEndpoints := []string{}
	
	for _, endpoint := range endpoints {
		resp, err := m.SendMCPRequest(ctx, endpoint, nil)
		if err == nil && resp.Error == nil {
			exposedEndpoints = append(exposedEndpoints, endpoint)
		}
	}
	
	if len(exposedEndpoints) > 0 {
		return &core.SecurityFinding{
			ID:          "no-auth-required",
			Title:       "MCP Endpoints Accessible Without Authentication",
			Description: fmt.Sprintf("Found %d endpoints accessible without authentication", len(exposedEndpoints)),
			Severity:    core.High,
			Evidence: []core.Evidence{
				{
					Type: "response",
					Data: map[string]interface{}{
						"exposed_endpoints": exposedEndpoints,
					},
					Description: "Endpoints that responded without authentication",
				},
			},
		}, nil
	}
	
	return nil, nil
}

// testEmptyToken tests with empty auth tokens
func (m *AuthBypassModule) testEmptyToken(ctx context.Context) (*core.SecurityFinding, error) {
	// We'll need to modify the base HTTP request
	// For now, this is a placeholder - would need to extend BaseModule
	// to support custom headers
	
	return nil, nil
}

// testDefaultCredentials tests common default credentials
func (m *AuthBypassModule) testDefaultCredentials(ctx context.Context) (*core.SecurityFinding, error) {
	// Would need auth endpoint to test
	// Common default credentials to test:
	// admin/admin, admin/password, admin/123456
	// user/user, test/test, demo/demo, mcp/mcp
	// Placeholder for now
	
	return nil, nil
}

// testJWTNoneAlgorithm tests JWT none algorithm vulnerability
func (m *AuthBypassModule) testJWTNoneAlgorithm(ctx context.Context) (*core.SecurityFinding, error) {
	// Create JWT with 'none' algorithm
	// This would require JWT manipulation capabilities
	
	return nil, nil
}

// testAuthHeaderInjection tests auth header injection
func (m *AuthBypassModule) testAuthHeaderInjection(ctx context.Context) (*core.SecurityFinding, error) {
	// Test various injection payloads in auth headers
	injectionPayloads := []string{
		"' OR '1'='1",
		"admin' --",
		"*/true/*",
		"bearer undefined",
		"bearer null",
	}
	
	// Would need to test these against auth endpoint
	_ = injectionPayloads
	
	return nil, nil
}

// summarizeFindings creates a finding summary
func (m *AuthBypassModule) summarizeFindings(findings []core.SecurityFinding) *core.FindingSummary {
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