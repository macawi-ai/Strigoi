package security

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SecurityRuleEngine manages and executes security rules for MCP scanning.
type SecurityRuleEngine struct {
	rules           []SecurityRule
	compiledRules   map[string]*CompiledRule
	credPatterns    map[string]*regexp.Regexp
	processPatterns []*regexp.Regexp
}

// SecurityRule defines a security check for MCP components.
type SecurityRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`               // "credential_exposure", "insecure_config", "network_exposure"
	Severity    string   `json:"severity"`               // "critical", "high", "medium", "low"
	Pattern     string   `json:"pattern"`                // Regex pattern to match
	FileTypes   []string `json:"file_types"`             // File extensions to check
	PathPattern string   `json:"path_pattern,omitempty"` // Optional path pattern
	Description string   `json:"description"`
	Remediation string   `json:"remediation"`
	References  []string `json:"references,omitempty"`
}

// CompiledRule contains a security rule with pre-compiled regex.
type CompiledRule struct {
	Rule      SecurityRule
	Pattern   *regexp.Regexp
	PathRegex *regexp.Regexp
}

// SecurityFinding represents a discovered security issue.
type SecurityFinding struct {
	ID          string    `json:"id"`
	RuleID      string    `json:"rule_id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Evidence    string    `json:"evidence"`
	FilePath    string    `json:"file_path"`
	LineNumber  int       `json:"line_number,omitempty"`
	Remediation string    `json:"remediation"`
	References  []string  `json:"references,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewSecurityRuleEngine creates a new rule engine with default MCP security rules.
func NewSecurityRuleEngine() *SecurityRuleEngine {
	engine := &SecurityRuleEngine{
		rules:         []SecurityRule{},
		compiledRules: make(map[string]*CompiledRule),
		credPatterns:  make(map[string]*regexp.Regexp),
	}

	// Load default rules
	engine.loadDefaultRules()
	return engine
}

// loadDefaultRules loads the default security rules for MCP scanning.
func (sre *SecurityRuleEngine) loadDefaultRules() {
	defaultRules := []SecurityRule{
		{
			ID:          "MCP-CRED-001",
			Name:        "Hardcoded API Key",
			Category:    "credential_exposure",
			Severity:    "high",
			Pattern:     `(?i)(api[_-]?key|apikey)\s*[:=]\s*['""]?([a-zA-Z0-9_-]{20,})['""]?`,
			FileTypes:   []string{".json", ".yaml", ".yml", ".env", ".config"},
			Description: "Hardcoded API key found in configuration file",
			Remediation: "Move API keys to secure environment variables or secret management system",
			References:  []string{"CWE-798", "OWASP-ASVS-2.10"},
		},
		{
			ID:          "MCP-CRED-002",
			Name:        "Hardcoded Password",
			Category:    "credential_exposure",
			Severity:    "critical",
			Pattern:     `(?i)(password|passwd|pwd)\s*[:=]\s*['""]?([^'"\s]{8,})['""]?`,
			FileTypes:   []string{".json", ".yaml", ".yml", ".env", ".config"},
			Description: "Hardcoded password found in configuration file",
			Remediation: "Use secure password storage mechanisms like environment variables or key vaults",
			References:  []string{"CWE-259", "OWASP-ASVS-2.1"},
		},
		{
			ID:          "MCP-CRED-003",
			Name:        "Bearer Token Exposure",
			Category:    "credential_exposure",
			Severity:    "high",
			Pattern:     `(?i)(bearer\s+|token\s*[:=]\s*)['""]?([a-zA-Z0-9_.-]{20,})['""]?`,
			FileTypes:   []string{".json", ".yaml", ".yml", ".env", ".config"},
			Description: "Bearer token found in configuration file",
			Remediation: "Store tokens securely using environment variables or secret management",
			References:  []string{"CWE-522", "RFC-6750"},
		},
		{
			ID:          "MCP-CONFIG-001",
			Name:        "Insecure HTTP Connection",
			Category:    "insecure_config",
			Severity:    "medium",
			Pattern:     `(?i)(url|endpoint|host)\s*[:=]\s*['""]?http://[^'"\s]+['""]?`,
			FileTypes:   []string{".json", ".yaml", ".yml", ".config"},
			Description: "Insecure HTTP connection configured instead of HTTPS",
			Remediation: "Use HTTPS for all network connections to ensure data encryption",
			References:  []string{"CWE-319", "OWASP-ASVS-9.1"},
		},
		{
			ID:          "MCP-CONFIG-002",
			Name:        "Neo4j Authentication Disabled",
			Category:    "insecure_config",
			Severity:    "high",
			Pattern:     `(?i)(auth\.enabled|dbms\.security\.auth_enabled)\s*[:=]\s*['""]?false['""]?`,
			FileTypes:   []string{".conf", ".properties", ".yaml", ".yml"},
			Description: "Neo4j authentication is disabled, allowing unrestricted access",
			Remediation: "Enable Neo4j authentication and use strong credentials",
			References:  []string{"CWE-306", "Neo4j-Security-Guide"},
		},
		{
			ID:          "MCP-CONFIG-003",
			Name:        "Default Neo4j Password",
			Category:    "insecure_config",
			Severity:    "critical",
			Pattern:     `(?i)(password|dbms\.security\.auth_password)\s*[:=]\s*['""]?(neo4j|password|admin)['""]?`,
			FileTypes:   []string{".conf", ".properties", ".yaml", ".yml"},
			Description: "Default or weak Neo4j password detected",
			Remediation: "Change default passwords to strong, unique credentials",
			References:  []string{"CWE-521", "Neo4j-Security-Guide"},
		},
		{
			ID:          "MCP-NETWORK-001",
			Name:        "Service Binding to All Interfaces",
			Category:    "network_exposure",
			Severity:    "medium",
			Pattern:     `(?i)(bind|listen|host)\s*[:=]\s*['""]?(0\.0\.0\.0|::)['""]?`,
			FileTypes:   []string{".json", ".yaml", ".yml", ".conf", ".config"},
			Description: "Service configured to bind to all network interfaces",
			Remediation: "Bind services to localhost or specific interfaces only",
			References:  []string{"CWE-668", "OWASP-ASVS-9.2"},
		},
		{
			ID:          "MCP-NETWORK-002",
			Name:        "Privileged Port Usage",
			Category:    "network_exposure",
			Severity:    "low",
			Pattern:     `(?i)(port)\s*[:=]\s*['""]?([1-9]|[1-9][0-9]|[1-9][0-9][0-9]|10[0-1][0-9]|102[0-3])['""]?`,
			FileTypes:   []string{".json", ".yaml", ".yml", ".conf", ".config"},
			Description: "Service configured to use privileged port (< 1024)",
			Remediation: "Use non-privileged ports (>= 1024) for better security",
			References:  []string{"Linux-Port-Security"},
		},
		{
			ID:          "MCP-SECRET-001",
			Name:        "Generic Secret Pattern",
			Category:    "credential_exposure",
			Severity:    "medium",
			Pattern:     `(?i)(secret|key|token)\s*[:=]\s*['""]?([a-zA-Z0-9_-]{16,})['""]?`,
			FileTypes:   []string{".json", ".yaml", ".yml", ".env", ".config"},
			Description: "Potential secret or key found in configuration",
			Remediation: "Use secure secret management systems",
			References:  []string{"CWE-798"},
		},
	}

	for _, rule := range defaultRules {
		_ = sre.AddRule(rule)
	}

	// Compile credential patterns
	sre.compileCredentialPatterns()
	sre.compileProcessPatterns()
}

// compileCredentialPatterns pre-compiles common credential detection patterns.
func (sre *SecurityRuleEngine) compileCredentialPatterns() {
	patterns := map[string]string{
		// JSON-style patterns
		"api_key":    `(?i)"(api[_-]?key|apikey)"\s*:\s*"([^"]{16,})"`,
		"password":   `(?i)"(password|passwd|pwd)"\s*:\s*"([^"]{8,})"`,
		"token":      `(?i)"(token|bearer[_-]?token)"\s*:\s*"([^"]{16,})"`,
		"secret":     `(?i)"(secret|secret[_-]?key)"\s*:\s*"([^"]{16,})"`,
		"aws_key":    `(?i)"(aws[_-]?access[_-]?key[_-]?id)"\s*:\s*"([A-Z0-9]{20})"`,
		"aws_secret": `(?i)"(aws[_-]?secret[_-]?access[_-]?key)"\s*:\s*"([a-zA-Z0-9+/]{40})"`,
		// Config file patterns (key=value)
		"config_api": `(?i)(api[_-]?key|apikey)\s*[:=]\s*([a-zA-Z0-9_-]{16,})`,
		"config_pwd": `(?i)(password|passwd)\s*[:=]\s*([a-zA-Z0-9_.-]{8,})`,
		// JWT tokens (standalone)
		"jwt": `eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`,
	}

	for name, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			sre.credPatterns[name] = compiled
		}
	}
}

// compileProcessPatterns pre-compiles process detection patterns.
func (sre *SecurityRuleEngine) compileProcessPatterns() {
	patterns := []string{
		`(?i)mcp[_-]?server`,
		`(?i)claude[_-]?mcp`,
		`(?i)neo4j.*mcp`,
		`(?i)duckdb.*mcp`,
		`(?i)chroma.*mcp`,
		`(?i).*mcp[_-]?bridge`,
	}

	for _, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			sre.processPatterns = append(sre.processPatterns, compiled)
		}
	}
}

// AddRule adds a new security rule to the engine.
func (sre *SecurityRuleEngine) AddRule(rule SecurityRule) error {
	// Validate rule
	if rule.ID == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}
	if rule.Pattern == "" {
		return fmt.Errorf("rule pattern cannot be empty")
	}

	// Compile regex pattern
	pattern, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern in rule %s: %w", rule.ID, err)
	}

	compiledRule := &CompiledRule{
		Rule:    rule,
		Pattern: pattern,
	}

	// Compile path pattern if present
	if rule.PathPattern != "" {
		pathRegex, err := regexp.Compile(rule.PathPattern)
		if err != nil {
			return fmt.Errorf("invalid path pattern in rule %s: %w", rule.ID, err)
		}
		compiledRule.PathRegex = pathRegex
	}

	sre.rules = append(sre.rules, rule)
	sre.compiledRules[rule.ID] = compiledRule
	return nil
}

// ScanContent scans content against all applicable security rules.
func (sre *SecurityRuleEngine) ScanContent(content, filePath string) []SecurityFinding {
	var findings []SecurityFinding

	fileExt := getFileExtension(filePath)

	for _, compiledRule := range sre.compiledRules {
		rule := compiledRule.Rule

		// Check if rule applies to this file type
		if len(rule.FileTypes) > 0 && !contains(rule.FileTypes, fileExt) {
			continue
		}

		// Check path pattern if specified
		if compiledRule.PathRegex != nil && !compiledRule.PathRegex.MatchString(filePath) {
			continue
		}

		// Scan content for matches
		matches := compiledRule.Pattern.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			finding := SecurityFinding{
				ID:          uuid.New().String(),
				RuleID:      rule.ID,
				Name:        rule.Name,
				Category:    rule.Category,
				Severity:    rule.Severity,
				Description: rule.Description,
				Evidence:    extractEvidence(content, match[0]),
				FilePath:    filePath,
				Remediation: rule.Remediation,
				References:  rule.References,
				Timestamp:   time.Now(),
			}

			// Add line number information
			finding.LineNumber = getLineNumber(content, match[0])
			findings = append(findings, finding)
		}
	}

	// Also call ScanCredentials for comprehensive scanning
	credFindings := sre.ScanCredentials(content, filePath)
	findings = append(findings, credFindings...)

	return findings
}

// ScanCredentials scans content specifically for credential patterns.
func (sre *SecurityRuleEngine) ScanCredentials(content, filePath string) []SecurityFinding {
	var findings []SecurityFinding

	for credType, pattern := range sre.credPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			finding := SecurityFinding{
				ID:          uuid.New().String(),
				RuleID:      fmt.Sprintf("CRED-%s", strings.ToUpper(credType)),
				Name:        fmt.Sprintf("Credential Exposure - %s", credType),
				Category:    "credential_exposure",
				Severity:    determineSeverity(credType),
				Description: fmt.Sprintf("Potential %s found in configuration", credType),
				Evidence:    maskSensitiveData(match[0]),
				FilePath:    filePath,
				LineNumber:  getLineNumber(content, match[0]),
				Remediation: "Move credentials to secure environment variables or secret management system",
				Timestamp:   time.Now(),
			}
			findings = append(findings, finding)
		}
	}

	return findings
}

// MatchProcess checks if a process matches MCP patterns.
func (sre *SecurityRuleEngine) MatchProcess(processLine string) bool {
	for _, pattern := range sre.processPatterns {
		if pattern.MatchString(processLine) {
			return true
		}
	}
	return false
}

// LoadRulesFromFile loads security rules from a JSON file.
func (sre *SecurityRuleEngine) LoadRulesFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read rules file: %w", err)
	}

	var rules []SecurityRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("failed to parse rules file: %w", err)
	}

	for _, rule := range rules {
		if err := sre.AddRule(rule); err != nil {
			return fmt.Errorf("failed to add rule %s: %w", rule.ID, err)
		}
	}

	return nil
}

// Helper functions

func getFileExtension(filePath string) string {
	parts := strings.Split(filePath, ".")
	if len(parts) > 1 {
		return "." + parts[len(parts)-1]
	}
	return ""
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func extractEvidence(content, match string) string {
	// Extract surrounding context for evidence
	index := strings.Index(content, match)
	if index == -1 {
		return match
	}

	start := index - 50
	if start < 0 {
		start = 0
	}

	end := index + len(match) + 50
	if end > len(content) {
		end = len(content)
	}

	evidence := content[start:end]
	// Mask sensitive parts
	return maskSensitiveData(evidence)
}

func maskSensitiveData(data string) string {
	// Simple masking - replace middle characters with *
	if len(data) <= 8 {
		return strings.Repeat("*", len(data))
	}

	start := data[:3]
	end := data[len(data)-3:]
	middle := strings.Repeat("*", len(data)-6)

	return start + middle + end
}

func getLineNumber(content, match string) int {
	index := strings.Index(content, match)
	if index == -1 {
		return 0
	}

	lines := strings.Split(content[:index], "\n")
	return len(lines)
}

func determineSeverity(credType string) string {
	criticalTypes := []string{"password", "aws_secret"}
	highTypes := []string{"api_key", "token", "jwt"}

	for _, critical := range criticalTypes {
		if credType == critical {
			return "critical"
		}
	}

	for _, high := range highTypes {
		if credType == high {
			return "high"
		}
	}

	return "medium"
}
