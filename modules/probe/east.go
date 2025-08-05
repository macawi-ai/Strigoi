package probe

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/pkg/modules"
)

func init() {
	// Register the module factory
	modules.RegisterBuiltin("probe/east", NewEastModule)
}

// EastModule probes data flows and integrations.
type EastModule struct {
	modules.BaseModule
	executor         *SecureExecutor
	compiledPatterns map[string]*regexp.Regexp
}

// NewEastModule creates a new East probe module.
func NewEastModule() modules.Module {
	return &EastModule{
		BaseModule: modules.BaseModule{
			ModuleName:        "probe/east",
			ModuleDescription: "Trace data flows, API integrations, and information leakage",
			ModuleType:        modules.ProbeModule,
			ModuleOptions: map[string]*modules.ModuleOption{
				"target": {
					Name:        "target",
					Description: "Target directory to analyze",
					Type:        "string",
					Required:    true,
					Default:     ".",
				},
				"max_file_size": {
					Name:        "max_file_size",
					Description: "Maximum file size to analyze (bytes)",
					Type:        "int",
					Required:    false,
					Default:     10485760, // 10MB
				},
				"extensions": {
					Name:        "extensions",
					Description: "File extensions to scan (comma-separated)",
					Type:        "string",
					Required:    false,
					Default:     ".js,.py,.go,.java,.rb,.php,.ts,.jsx,.tsx",
				},
				"exclude_dirs": {
					Name:        "exclude_dirs",
					Description: "Directories to exclude (comma-separated)",
					Type:        "string",
					Required:    false,
					Default:     "node_modules,vendor,.git,dist,build",
				},
				"confidence_threshold": {
					Name:        "confidence_threshold",
					Description: "Minimum confidence level (low, medium, high)",
					Type:        "string",
					Required:    false,
					Default:     "low",
				},
				"include_self": {
					Name:        "include_self",
					Description: "Include Strigoi's own files in scan",
					Type:        "bool",
					Required:    false,
					Default:     false,
				},
			},
		},
		executor:         NewSecureExecutor(),
		compiledPatterns: make(map[string]*regexp.Regexp),
	}
}

// Check verifies the module can run.
func (m *EastModule) Check() bool {
	// Verify target exists
	targetOpt, _ := m.GetOption("target")
	target := targetOpt.(string)

	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		return false
	}

	// Compile patterns
	for _, pattern := range SecretPatterns {
		compiled, err := regexp.Compile(pattern.Regex)
		if err == nil {
			m.compiledPatterns[pattern.Name] = compiled
		}
	}

	return true
}

// Info returns module metadata.
func (m *EastModule) Info() *modules.ModuleInfo {
	return &modules.ModuleInfo{
		Author:  "Strigoi Team",
		Version: "1.0.0",
		Tags:    []string{"data-flow", "secrets", "api", "leakage"},
		References: []string{
			"https://owasp.org/www-project-top-ten/",
			"https://cwe.mitre.org/data/definitions/200.html",
		},
	}
}

// Run executes the east probe.
func (m *EastModule) Run() (*modules.ModuleResult, error) {
	startTime := time.Now()

	// Get options
	targetOpt, _ := m.GetOption("target")
	target := targetOpt.(string)

	// Validate target path
	if err := m.executor.ValidatePath(target); err != nil {
		return nil, fmt.Errorf("invalid target path: %w", err)
	}

	// Initialize result
	result := &DataFlowResult{
		Findings:         []Finding{},
		DataFlows:        []DataFlow{},
		ExternalServices: []ExternalService{},
	}

	// Get scan parameters
	extensions := m.getExtensions()
	excludeDirs := m.getExcludeDirs()
	maxFileSize := m.getMaxFileSize()

	// Scan files
	err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			// Check if we should skip this directory
			dir := filepath.Base(path)
			for _, exclude := range excludeDirs {
				if dir == exclude {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Skip large files
		if info.Size() > maxFileSize {
			return nil
		}

		// Check extension
		ext := filepath.Ext(path)
		if !m.shouldScanFile(ext, extensions) {
			return nil
		}

		// Check if we should skip Strigoi's own files
		includeSelfOpt, _ := m.GetOption("include_self")
		includeSelf := includeSelfOpt.(bool)
		if !includeSelf && m.isStrigoiFile(path) {
			return nil
		}

		// Analyze file
		findings := m.analyzeFile(path)
		result.Findings = append(result.Findings, findings...)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	// Extract data flows and services from findings
	m.extractDataFlows(result)
	m.extractServices(result)

	// Calculate summary
	result.Summary = m.calculateDataFlowSummary(result)

	// Filter by confidence threshold
	result.Findings = m.filterByConfidence(result.Findings)

	return &modules.ModuleResult{
		Module:    m.Name(),
		Status:    "completed",
		StartTime: startTime,
		EndTime:   time.Now(),
		Data: map[string]interface{}{
			"result": result,
		},
	}, nil
}

// analyzeFile scans a single file for security issues.
func (m *EastModule) analyzeFile(path string) []Finding {
	findings := []Finding{}

	file, err := os.Open(path)
	if err != nil {
		return findings
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check for hardcoded secrets
		for patternName, regex := range m.compiledPatterns {
			if matches := regex.FindStringSubmatch(line); len(matches) > 0 {
				// Find the pattern details
				var pattern Pattern
				for _, p := range SecretPatterns {
					if p.Name == patternName {
						pattern = p
						break
					}
				}

				finding := Finding{
					Type:        "hardcoded_secret",
					Category:    pattern.Category,
					Location:    fmt.Sprintf("%s:%d", path, lineNum),
					Confidence:  pattern.Confidence,
					Severity:    m.determineSeverity("hardcoded_secret", pattern.Category),
					Evidence:    m.redactSecret(line, matches[0]),
					Remediation: "Use environment variables or a secret management system",
				}

				// Add references based on category
				finding.References = m.getReferences(pattern.Category)

				findings = append(findings, finding)
			}
		}

		// Check for API endpoints
		if endpoint := m.extractAPIEndpoint(line); endpoint != "" {
			findings = append(findings, Finding{
				Type:        "api_endpoint",
				Category:    "external_service",
				Location:    fmt.Sprintf("%s:%d", path, lineNum),
				Confidence:  "medium",
				Severity:    m.determineSeverity("api_endpoint", "external_service"),
				Evidence:    endpoint,
				Impact:      "External service dependency",
				Remediation: "Ensure proper authentication and encryption",
			})
		}

		// Check for verbose errors
		if m.isVerboseError(line) {
			findings = append(findings, Finding{
				Type:        "information_disclosure",
				Category:    "verbose_error",
				Location:    fmt.Sprintf("%s:%d", path, lineNum),
				Confidence:  "medium",
				Severity:    m.determineSeverity("information_disclosure", "verbose_error"),
				Evidence:    m.truncate(line, 100),
				Impact:      "May expose internal paths or stack traces",
				Remediation: "Implement proper error handling for production",
			})
		}

		// Check for debug endpoints
		if m.isDebugEndpoint(line) {
			findings = append(findings, Finding{
				Type:        "misconfiguration",
				Category:    "debug_endpoint",
				Location:    fmt.Sprintf("%s:%d", path, lineNum),
				Confidence:  "high",
				Severity:    m.determineSeverity("misconfiguration", "debug_endpoint"),
				Evidence:    m.truncate(line, 100),
				Impact:      "Debug endpoints should not be exposed in production",
				Remediation: "Remove or protect debug endpoints",
			})
		}
	}

	return findings
}

// redactSecret partially hides a secret value.
func (m *EastModule) redactSecret(line, secret string) string {
	if len(secret) <= 8 {
		return strings.Replace(line, secret, "[REDACTED]", 1)
	}

	// Show first 4 and last 4 characters
	redacted := secret[:4] + "[REDACTED]" + secret[len(secret)-4:]
	return strings.Replace(line, secret, redacted, 1)
}

// extractAPIEndpoint finds API URLs in code.
func (m *EastModule) extractAPIEndpoint(line string) string {
	// Look for common API patterns
	patterns := []string{
		`https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/api/`,
		`https?://api\.[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
		`https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/v[0-9]+/`,
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		if match := regex.FindString(line); match != "" {
			// Parse and validate URL
			if u, err := url.Parse(match); err == nil && u.Host != "" {
				return u.String()
			}
		}
	}

	return ""
}

// isVerboseError checks for verbose error patterns.
func (m *EastModule) isVerboseError(line string) bool {
	errorPatterns := []string{
		`stack\s*trace`,
		`traceback`,
		`at\s+\w+\s*\([^)]+:[0-9]+:[0-9]+\)`,
		`File\s+"[^"]+",\s+line\s+[0-9]+`,
		`\.printStackTrace\(\)`,
		`console\.error\(.*Error.*\)`,
	}

	for _, pattern := range errorPatterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}

// isDebugEndpoint checks for debug endpoint patterns.
func (m *EastModule) isDebugEndpoint(line string) bool {
	debugPatterns := []string{
		`/debug/`,
		`/test/`,
		`/_debug`,
		`/phpinfo`,
		`/server-status`,
		`/__debugging`,
		`route.*debug.*true`,
	}

	for _, pattern := range debugPatterns {
		if strings.Contains(strings.ToLower(line), pattern) {
			return true
		}
	}

	return false
}

// Helper methods

func (m *EastModule) getExtensions() []string {
	extOpt, _ := m.GetOption("extensions")
	return strings.Split(extOpt.(string), ",")
}

func (m *EastModule) getExcludeDirs() []string {
	excludeOpt, _ := m.GetOption("exclude_dirs")
	return strings.Split(excludeOpt.(string), ",")
}

func (m *EastModule) getMaxFileSize() int64 {
	sizeOpt, _ := m.GetOption("max_file_size")
	return int64(sizeOpt.(int))
}

func (m *EastModule) shouldScanFile(ext string, extensions []string) bool {
	for _, allowed := range extensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

func (m *EastModule) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (m *EastModule) getReferences(category string) []string {
	references := map[string][]string{
		"aws_key": {
			"https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html",
			"https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/02-Configuration_and_Deployment_Management_Testing/09-Test_for_Subdomain_Takeover",
		},
		"api_key": {
			"https://owasp.org/www-community/vulnerabilities/Use_of_hard-coded_password",
			"https://cwe.mitre.org/data/definitions/798.html",
		},
		"private_key": {
			"https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/09-Testing_for_Weak_Cryptography/04-Testing_for_Weak_Encryption",
		},
	}

	if refs, exists := references[category]; exists {
		return refs
	}

	return []string{
		"https://owasp.org/www-project-top-ten/2017/A3_2017-Sensitive_Data_Exposure",
	}
}

// extractDataFlows analyzes findings to identify data flows.
func (m *EastModule) extractDataFlows(result *DataFlowResult) {
	// Group findings by file to identify flows
	fileFindings := make(map[string][]Finding)
	for _, finding := range result.Findings {
		parts := strings.Split(finding.Location, ":")
		if len(parts) > 0 {
			fileName := parts[0]
			fileFindings[fileName] = append(fileFindings[fileName], finding)
		}
	}

	// Create basic data flows based on findings
	flowID := 0
	for _, findings := range fileFindings {
		var hasInput, hasOutput, hasSecret bool
		var sensitiveData []string

		for _, finding := range findings {
			switch finding.Type {
			case "hardcoded_secret":
				hasSecret = true
				sensitiveData = append(sensitiveData, finding.Category)
			case "api_endpoint":
				hasOutput = true
			case "information_disclosure":
				hasInput = true
			}
		}

		// If we have both input and output, create a flow
		if hasInput && hasOutput {
			flowID++
			flow := DataFlow{
				ID:            fmt.Sprintf("flow_%d", flowID),
				Source:        "user_input",
				Destination:   "external_api",
				SensitiveData: sensitiveData,
			}

			if hasSecret {
				flow.Protection = []string{"none"}
			} else {
				flow.Protection = []string{"unknown"}
			}

			result.DataFlows = append(result.DataFlows, flow)
		}
	}
}

// extractServices identifies external services from findings.
func (m *EastModule) extractServices(result *DataFlowResult) {
	services := make(map[string]*ExternalService)

	for _, finding := range result.Findings {
		if finding.Type == "api_endpoint" {
			// Parse the URL
			if u, err := url.Parse(finding.Evidence); err == nil && u.Host != "" {
				if _, exists := services[u.Host]; !exists {
					services[u.Host] = &ExternalService{
						Domain:         u.Host,
						Purpose:        "unknown",
						Authentication: "unknown",
						DataShared:     []string{"unknown"},
						Encrypted:      u.Scheme == "https",
					}
				}
			}
		}
	}

	// Convert map to slice
	for _, service := range services {
		result.ExternalServices = append(result.ExternalServices, *service)
	}
}

// calculateDataFlowSummary generates summary statistics.
func (m *EastModule) calculateDataFlowSummary(result *DataFlowResult) DataFlowSummary {
	summary := DataFlowSummary{
		ExternalServices: len(result.ExternalServices),
		DataFlows:        len(result.DataFlows),
	}

	// Count findings by type
	for _, finding := range result.Findings {
		switch finding.Type {
		case "hardcoded_secret":
			summary.PotentialSecrets++
		case "information_disclosure", "misconfiguration":
			summary.LeakPoints++
		}
	}

	return summary
}

// filterByConfidence filters findings based on confidence threshold.
func (m *EastModule) filterByConfidence(findings []Finding) []Finding {
	thresholdOpt, _ := m.GetOption("confidence_threshold")
	threshold := thresholdOpt.(string)

	// Define confidence levels
	levels := map[string]int{
		"low":    1,
		"medium": 2,
		"high":   3,
	}

	minLevel := levels[threshold]
	filtered := []Finding{}

	for _, finding := range findings {
		if levels[finding.Confidence] >= minLevel {
			filtered = append(filtered, finding)
		}
	}

	return filtered
}

// determineSeverity calculates severity based on finding type and category.
func (m *EastModule) determineSeverity(findingType, category string) string {
	// Critical severity patterns
	criticalPatterns := map[string]bool{
		"private_key":           true,
		"aws_secret":            true,
		"hardcoded_credentials": true,
	}

	// High severity patterns
	highPatterns := map[string]bool{
		"aws_key":      true,
		"api_key":      true,
		"github_token": true,
		"slack_token":  true,
		"auth_token":   true,
	}

	// Check for critical
	if criticalPatterns[category] {
		return "critical"
	}

	// Special cases
	if findingType == "hardcoded_secret" && category == "private_key" {
		return "critical"
	}

	if findingType == "misconfiguration" && category == "debug_endpoint" {
		return "high"
	}

	// Check for high
	if highPatterns[category] {
		return "high"
	}

	// Information disclosure is typically medium
	if findingType == "information_disclosure" {
		return "medium"
	}

	// Default to medium for most findings
	return "medium"
}

// isStrigoiFile checks if a file belongs to the Strigoi project.
func (m *EastModule) isStrigoiFile(path string) bool {
	// Check if path contains Strigoi-specific directories or files
	strigoiPatterns := []string{
		"strigoi",
		"modules/probe",
		"pkg/security",
		"pkg/session",
		"pkg/output",
		"cmd/strigoi",
	}

	lowerPath := strings.ToLower(path)
	for _, pattern := range strigoiPatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}
