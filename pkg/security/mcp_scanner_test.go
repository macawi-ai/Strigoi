package security

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewMCPScanner(t *testing.T) {
	executor := NewSecureExecutor()
	scanner := NewMCPScanner(executor)

	if scanner.executor != executor {
		t.Error("Expected executor to be set")
	}

	if scanner.logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if scanner.rules == nil {
		t.Error("Expected rules engine to be initialized")
	}

	if scanner.findings == nil {
		t.Error("Expected findings slice to be initialized")
	}
}

func TestParseClaudeCodeConfig(t *testing.T) {
	scanner := NewMCPScanner(NewSecureExecutor())

	configContent := `{
		"mcpServers": {
			"neo4j-consciousness": {
				"command": "node",
				"args": ["neo4j-mcp-server.js"],
				"env": {
					"NEO4J_URI": "bolt://localhost:7688"
				}
			},
			"duckdb-analytics": {
				"command": "python",
				"args": ["-m", "duckdb_mcp"]
			}
		}
	}`

	tool := scanner.parseClaudeCodeConfig("/test/mcp_servers.json", []byte(configContent))

	if tool == nil {
		t.Fatal("Expected tool to be parsed")
	}

	if tool.Name != "Claude Code MCP Configuration" {
		t.Errorf("Expected name 'Claude Code MCP Configuration', got %s", tool.Name)
	}

	if tool.Type != "client" {
		t.Errorf("Expected type 'client', got %s", tool.Type)
	}

	if len(tool.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(tool.Dependencies))
	}

	// Check dependencies (order doesn't matter, just verify both exist)
	deps := tool.Dependencies
	depNames := make(map[string]bool)
	for _, dep := range deps {
		depNames[dep.Name] = true
	}

	if !depNames["neo4j-consciousness"] {
		t.Error("Expected 'neo4j-consciousness' dependency not found")
	}

	if !depNames["duckdb-analytics"] {
		t.Error("Expected 'duckdb-analytics' dependency not found")
	}
}

func TestParseProcessLine(t *testing.T) {
	scanner := NewMCPScanner(NewSecureExecutor())

	testCases := []struct {
		name        string
		processLine string
		expectNil   bool
		expectedPID int
		expectedCmd string
	}{
		{
			name:        "valid neo4j process",
			processLine: "user      12345  0.1  2.3 1234567 89012 ?        Sl   10:30   0:01 /usr/bin/java -jar neo4j-mcp-server.jar",
			expectNil:   false,
			expectedPID: 12345,
			expectedCmd: "/usr/bin/java",
		},
		{
			name:        "invalid process line",
			processLine: "user 12345",
			expectNil:   true,
		},
		{
			name:        "non-numeric PID",
			processLine: "user abc 0.1 2.3 1234567 89012 ? Sl 10:30 0:01 /usr/bin/test",
			expectNil:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			process := scanner.parseProcessLine(tc.processLine)

			if tc.expectNil {
				if process != nil {
					t.Error("Expected nil process")
				}
				return
			}

			if process == nil {
				t.Fatal("Expected non-nil process")
			}

			if process.PID != tc.expectedPID {
				t.Errorf("Expected PID %d, got %d", tc.expectedPID, process.PID)
			}

			if process.Command != tc.expectedCmd {
				t.Errorf("Expected command %s, got %s", tc.expectedCmd, process.Command)
			}
		})
	}
}

func TestIsConfigFile(t *testing.T) {
	scanner := NewMCPScanner(NewSecureExecutor())

	testCases := []struct {
		fileName string
		expected bool
	}{
		{"mcp_servers.json", true},
		{"config.yaml", true},
		{"settings.yml", true},
		{"app.conf", true},
		{"service.config", true},
		{".env", true},
		{"readme.txt", false},
		{"binary", false},
		{"script.sh", false},
	}

	for _, tc := range testCases {
		t.Run(tc.fileName, func(t *testing.T) {
			result := scanner.isConfigFile(tc.fileName)
			if result != tc.expected {
				t.Errorf("Expected %v for %s, got %v", tc.expected, tc.fileName, result)
			}
		})
	}
}

func TestContainsMCPKeywords(t *testing.T) {
	scanner := NewMCPScanner(NewSecureExecutor())

	testCases := []struct {
		name     string
		config   map[string]interface{}
		expected bool
	}{
		{
			name:     "contains mcp",
			config:   map[string]interface{}{"mcp_server": "test"},
			expected: true,
		},
		{
			name:     "contains neo4j",
			config:   map[string]interface{}{"database": "neo4j"},
			expected: true,
		},
		{
			name:     "contains duckdb",
			config:   map[string]interface{}{"analytics": "duckdb"},
			expected: true,
		},
		{
			name:     "no keywords",
			config:   map[string]interface{}{"app": "test"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := scanner.containsMCPKeywords(tc.config)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestScanCredentials(t *testing.T) {
	scanner := NewMCPScanner(NewSecureExecutor())

	testContent := `{
		"api_key": "sk-1234567890abcdef1234567890abcdef",
		"password": "secretpassword123",
		"database_url": "neo4j://user:password@localhost:7687"
	}`

	findings := scanner.rules.ScanCredentials(testContent, "test.json")

	if len(findings) == 0 {
		t.Error("Expected to find credential exposures")
	}

	// Check that findings have required fields
	for _, finding := range findings {
		if finding.ID == "" {
			t.Error("Finding should have ID")
		}

		if finding.Severity == "" {
			t.Error("Finding should have severity")
		}

		if finding.Category != "credential_exposure" {
			t.Errorf("Expected category 'credential_exposure', got %s", finding.Category)
		}

		if finding.Evidence == "" {
			t.Error("Finding should have evidence")
		}

		// Evidence should be masked
		if strings.Contains(finding.Evidence, "secretpassword123") {
			t.Error("Evidence should be masked")
		}
	}
}

func TestScanFileIntegration(t *testing.T) {
	// Create temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "mcp_servers.json")

	testContent := `{
		"mcpServers": {
			"neo4j-test": {
				"command": "node",
				"args": ["server.js"],
				"env": {
					"API_KEY": "secret1234567890abcdef",
					"NEO4J_PASSWORD": "password123456789"
				}
			}
		}
	}`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewMCPScanner(NewSecureExecutor())
	scanner.scanFile(testFile)

	// Check that tool was discovered
	findings := scanner.getFindings()
	if len(findings) == 0 {
		t.Error("Expected to find MCP tool")
	}

	tool := findings[0]
	if tool.Name != "Claude Code MCP Configuration" {
		t.Errorf("Expected 'Claude Code MCP Configuration', got %s", tool.Name)
	}

	if len(tool.SecurityRisks) == 0 {
		t.Error("Expected to find security risks")
	}
}

func TestProcessToMCPTool(t *testing.T) {
	scanner := NewMCPScanner(NewSecureExecutor())

	process := ProcessInfo{
		PID:         12345,
		User:        "testuser",
		Command:     "/usr/bin/mcp-server",
		Args:        []string{"--port", "8080"},
		ProcessName: "mcp-server",
	}

	tool := scanner.processToMCPTool(process)

	if tool.Name != "mcp-server" {
		t.Errorf("Expected name 'mcp-server', got %s", tool.Name)
	}

	if tool.Type != "server" {
		t.Errorf("Expected type 'server', got %s", tool.Type)
	}

	if tool.Status != "running" {
		t.Errorf("Expected status 'running', got %s", tool.Status)
	}

	if tool.ProcessID != 12345 {
		t.Errorf("Expected PID 12345, got %d", tool.ProcessID)
	}
}

func TestParseNetworkLine(t *testing.T) {
	scanner := NewMCPScanner(NewSecureExecutor())

	testCases := []struct {
		name         string
		networkLine  string
		expectNil    bool
		expectedPort int
	}{
		{
			name:         "valid tcp connection",
			networkLine:  "tcp        0      0 127.0.0.1:7687          0.0.0.0:*               LISTEN      12345/neo4j",
			expectNil:    false,
			expectedPort: 7687,
		},
		{
			name:        "invalid line",
			networkLine: "invalid line",
			expectNil:   true,
		},
		{
			name:        "no port",
			networkLine: "tcp 0 0 localhost 0.0.0.0:* LISTEN",
			expectNil:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn := scanner.parseNetworkLine(tc.networkLine)

			if tc.expectNil {
				if conn != nil {
					t.Error("Expected nil connection")
				}
				return
			}

			if conn == nil {
				t.Fatal("Expected non-nil connection")
			}

			if conn.Port != tc.expectedPort {
				t.Errorf("Expected port %d, got %d", tc.expectedPort, conn.Port)
			}
		})
	}
}

func TestDiscoverMCPToolsTimeout(t *testing.T) {
	scanner := NewMCPScanner(NewSecureExecutor())

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This should complete quickly or timeout gracefully
	_, err := scanner.DiscoverMCPTools(ctx)

	// Should not panic or hang
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCorrelateFindings(t *testing.T) {
	scanner := NewMCPScanner(NewSecureExecutor())

	// Add a running process
	runningTool := MCPTool{
		ID:        "1",
		Name:      "neo4j-server",
		Type:      "server",
		Status:    "running",
		ProcessID: 12345,
	}

	// Add a configuration with matching name pattern
	configTool := MCPTool{
		ID:         "2",
		Name:       "neo4j-server configuration", // Name contains "neo4j-server" to match running tool
		Type:       "config",
		Status:     "configured",
		ConfigPath: "/etc/neo4j.conf",
		Configuration: map[string]interface{}{
			"host": "localhost",
		},
	}

	scanner.findings = []MCPTool{runningTool, configTool}
	scanner.correlateFindings()

	// Check if correlation worked
	findings := scanner.getFindings()
	found := false
	for _, tool := range findings {
		if tool.Status == "running" && tool.ConfigPath != "" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected correlation between running process and configuration")
	}
}

func TestAnalyzeSecurityRisks(t *testing.T) {
	scanner := NewMCPScanner(NewSecureExecutor())

	// Add a tool with config path
	tool := MCPTool{
		ID:         "1",
		Name:       "test-tool",
		ConfigPath: "/test/config.json",
	}

	// Add security findings for the same path
	finding := SecurityFinding{
		ID:       "risk1",
		FilePath: "/test/config.json",
		Severity: "high",
	}

	scanner.findings = []MCPTool{tool}
	scanner.risks = []SecurityFinding{finding}

	scanner.analyzeSecurityRisks()

	// Check if risks were attached to tool
	findings := scanner.getFindings()
	if len(findings) == 0 {
		t.Fatal("Expected tool finding")
	}

	if len(findings[0].SecurityRisks) == 0 {
		t.Error("Expected security risks to be attached to tool")
	}

	if findings[0].SecurityRisks[0].ID != "risk1" {
		t.Error("Expected risk to be attached correctly")
	}
}
