package security_audit

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ConfigScanner checks for configuration security issues
type ConfigScanner struct {
	configPatterns map[string]*regexp.Regexp
}

// NewConfigScanner creates a new configuration scanner
func NewConfigScanner() *ConfigScanner {
	return &ConfigScanner{
		configPatterns: map[string]*regexp.Regexp{
			"weak_perms":     regexp.MustCompile(`[67][67][67]`), // World writable
			"debug_enabled":  regexp.MustCompile(`(?i)(debug|verbose).*true|enabled`),
			"default_creds":  regexp.MustCompile(`(?i)(admin|root|default).*password`),
			"exposed_metric": regexp.MustCompile(`(?i)metrics.*0\.0\.0\.0`),
		},
	}
}

func (s *ConfigScanner) Name() string {
	return "Configuration Security Scanner"
}

func (s *ConfigScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	// Common config file patterns
	configFiles := []string{
		"*.yaml", "*.yml", "*.json", "*.toml", "*.conf",
		"*.env", ".env.*", "config/*", "configs/*",
	}

	for _, pattern := range configFiles {
		matches, err := filepath.Glob(filepath.Join(path, pattern))
		if err != nil {
			continue
		}

		for _, file := range matches {
			fileIssues, err := s.scanConfigFile(file)
			if err != nil {
				continue
			}
			issues = append(issues, fileIssues...)
		}
	}

	// Check file permissions
	issues = append(issues, s.checkFilePermissions(path)...)

	// Check for insecure defaults
	issues = append(issues, s.checkInsecureDefaults(path)...)

	return issues, nil
}

func (s *ConfigScanner) scanConfigFile(filename string) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return issues, err
	}

	// Check file permissions first
	info, err := os.Stat(filename)
	if err == nil {
		mode := info.Mode()
		if mode.Perm()&0066 != 0 { // World readable/writable
			issues = append(issues, SecurityIssue{
				Type:        "INSECURE_FILE_PERMISSIONS",
				Severity:    "HIGH",
				Title:       "Configuration file has insecure permissions",
				Description: fmt.Sprintf("File %s is world readable/writable", filename),
				Location: IssueLocation{
					File: filename,
				},
				CWE:         "CWE-732",
				Remediation: "Set file permissions to 600 or 640",
			})
		}
	}

	// Parse based on file type
	ext := filepath.Ext(filename)
	switch ext {
	case ".json":
		issues = append(issues, s.scanJSONConfig(content, filename)...)
	case ".yaml", ".yml":
		issues = append(issues, s.scanYAMLConfig(content, filename)...)
	case ".env":
		issues = append(issues, s.scanEnvFile(content, filename)...)
	default:
		issues = append(issues, s.scanGenericConfig(string(content), filename)...)
	}

	return issues, nil
}

func (s *ConfigScanner) scanJSONConfig(content []byte, filename string) []SecurityIssue {
	var issues []SecurityIssue
	var config map[string]interface{}

	if err := json.Unmarshal(content, &config); err != nil {
		return issues
	}

	// Check for security-relevant settings
	issues = append(issues, s.checkConfigValues(config, filename)...)

	return issues
}

func (s *ConfigScanner) scanYAMLConfig(content []byte, filename string) []SecurityIssue {
	var issues []SecurityIssue

	// For now, just scan as generic config
	// TODO: Add proper YAML parsing
	issues = append(issues, s.scanGenericConfig(string(content), filename)...)

	return issues
}

func (s *ConfigScanner) scanEnvFile(content []byte, filename string) []SecurityIssue {
	var issues []SecurityIssue
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Check for sensitive values
		if strings.Contains(strings.ToLower(key), "password") ||
			strings.Contains(strings.ToLower(key), "secret") ||
			strings.Contains(strings.ToLower(key), "key") {
			if len(value) < 12 {
				issues = append(issues, SecurityIssue{
					Type:        "WEAK_SECRET",
					Severity:    "HIGH",
					Title:       fmt.Sprintf("Weak %s in environment", key),
					Description: "Secret value appears to be weak or default",
					Location: IssueLocation{
						File: filename,
						Line: i + 1,
					},
					CWE:         "CWE-521",
					Remediation: "Use strong, randomly generated secrets",
				})
			}
		}

		// Check for debug mode
		if strings.Contains(strings.ToLower(key), "debug") &&
			(value == "true" || value == "1" || value == "enabled") {
			issues = append(issues, SecurityIssue{
				Type:        "DEBUG_ENABLED",
				Severity:    "MEDIUM",
				Title:       "Debug mode enabled in production",
				Description: "Debug mode can expose sensitive information",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-489",
				Remediation: "Disable debug mode in production",
			})
		}
	}

	return issues
}

func (s *ConfigScanner) scanGenericConfig(content, filename string) []SecurityIssue {
	var issues []SecurityIssue
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// Check for debug enabled
		if s.configPatterns["debug_enabled"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "DEBUG_ENABLED",
				Severity:    "MEDIUM",
				Title:       "Debug mode enabled",
				Description: "Debug settings found in configuration",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				Evidence: map[string]string{
					"line": strings.TrimSpace(line),
				},
				CWE:         "CWE-489",
				Remediation: "Disable debug mode in production",
			})
		}

		// Check for default credentials
		if s.configPatterns["default_creds"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "DEFAULT_CREDENTIALS",
				Severity:    "CRITICAL",
				Title:       "Default credentials detected",
				Description: "Configuration contains default or weak credentials",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-798",
				OWASP:       "A07:2021",
				Remediation: "Change default credentials immediately",
			})
		}
	}

	return issues
}

func (s *ConfigScanner) checkConfigValues(config map[string]interface{}, filename string) []SecurityIssue {
	var issues []SecurityIssue

	// Recursively check configuration values
	s.walkConfig(config, filename, "", &issues)

	return issues
}

func (s *ConfigScanner) walkConfig(config map[string]interface{}, filename, path string, issues *[]SecurityIssue) {
	for key, value := range config {
		currentPath := key
		if path != "" {
			currentPath = path + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			s.walkConfig(v, filename, currentPath, issues)

		case string:
			// Check for insecure values
			keyLower := strings.ToLower(key)

			// SSL/TLS verification disabled
			if (keyLower == "verify_ssl" || keyLower == "ssl_verify") &&
				(v == "false" || v == "0") {
				*issues = append(*issues, SecurityIssue{
					Type:        "SSL_VERIFICATION_DISABLED",
					Severity:    "HIGH",
					Title:       "SSL verification disabled",
					Description: fmt.Sprintf("SSL verification disabled at %s", currentPath),
					Location: IssueLocation{
						File: filename,
					},
					CWE:         "CWE-295",
					Remediation: "Enable SSL verification",
				})
			}

			// Weak encryption
			if strings.Contains(keyLower, "algorithm") &&
				(v == "DES" || v == "MD5" || v == "SHA1") {
				*issues = append(*issues, SecurityIssue{
					Type:        "WEAK_ENCRYPTION",
					Severity:    "HIGH",
					Title:       fmt.Sprintf("Weak encryption algorithm: %s", v),
					Description: fmt.Sprintf("Weak algorithm configured at %s", currentPath),
					Location: IssueLocation{
						File: filename,
					},
					CWE:         "CWE-326",
					Remediation: "Use strong encryption algorithms",
				})
			}

		case bool:
			// Debug mode
			if strings.Contains(strings.ToLower(key), "debug") && v {
				*issues = append(*issues, SecurityIssue{
					Type:        "DEBUG_ENABLED",
					Severity:    "MEDIUM",
					Title:       "Debug mode enabled",
					Description: fmt.Sprintf("Debug enabled at %s", currentPath),
					Location: IssueLocation{
						File: filename,
					},
					CWE:         "CWE-489",
					Remediation: "Disable debug in production",
				})
			}
		}
	}
}

func (s *ConfigScanner) checkFilePermissions(path string) []SecurityIssue {
	var issues []SecurityIssue

	// Check permissions on sensitive directories
	sensitiveeDirs := []string{
		".ssh", ".gnupg", "certs", "certificates", "keys", "secrets",
	}

	for _, dir := range sensitiveeDirs {
		dirPath := filepath.Join(path, dir)
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			mode := info.Mode()
			if mode.Perm()&0077 != 0 { // Others have access
				issues = append(issues, SecurityIssue{
					Type:        "INSECURE_DIRECTORY_PERMISSIONS",
					Severity:    "HIGH",
					Title:       fmt.Sprintf("Sensitive directory has insecure permissions: %s", dir),
					Description: "Directory contains sensitive data but is accessible to others",
					Location: IssueLocation{
						File: dirPath,
					},
					CWE:         "CWE-732",
					Remediation: "Set directory permissions to 700",
				})
			}
		}
	}

	return issues
}

func (s *ConfigScanner) checkInsecureDefaults(path string) []SecurityIssue {
	var issues []SecurityIssue

	// Check for common insecure default configurations
	defaultFiles := map[string]string{
		"docker-compose.yml": "services.*ports.*0\\.0\\.0\\.0",
		"prometheus.yml":     "0\\.0\\.0\\.0:9090",
		"grafana.ini":        "admin_password.*admin",
	}

	for file, pattern := range defaultFiles {
		filePath := filepath.Join(path, file)
		if content, err := ioutil.ReadFile(filePath); err == nil {
			if matched, _ := regexp.Match(pattern, content); matched {
				issues = append(issues, SecurityIssue{
					Type:        "INSECURE_DEFAULT_CONFIG",
					Severity:    "HIGH",
					Title:       fmt.Sprintf("Insecure default configuration in %s", file),
					Description: "Default configuration exposes service or uses weak credentials",
					Location: IssueLocation{
						File: filePath,
					},
					Remediation: "Update configuration with secure values",
				})
			}
		}
	}

	return issues
}

// SecretsScanner searches for exposed secrets
type SecretsScanner struct {
	patterns map[string]*regexp.Regexp
	entropy  *EntropyAnalyzer
}

// NewSecretsScanner creates a new secrets scanner
func NewSecretsScanner() *SecretsScanner {
	return &SecretsScanner{
		patterns: map[string]*regexp.Regexp{
			// API Keys
			"aws_key":      regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			"aws_secret":   regexp.MustCompile(`[0-9a-zA-Z/+=]{40}`),
			"github_token": regexp.MustCompile(`ghp_[0-9a-zA-Z]{36}`),
			"slack_token":  regexp.MustCompile(`xox[baprs]-[0-9a-zA-Z]{10,48}`),
			"google_api":   regexp.MustCompile(`AIza[0-9A-Za-z\\-_]{35}`),

			// Private Keys
			"rsa_private": regexp.MustCompile(`-----BEGIN RSA PRIVATE KEY-----`),
			"ssh_private": regexp.MustCompile(`-----BEGIN OPENSSH PRIVATE KEY-----`),
			"pgp_private": regexp.MustCompile(`-----BEGIN PGP PRIVATE KEY BLOCK-----`),

			// Generic patterns
			"generic_secret": regexp.MustCompile(`(?i)(api_key|apikey|secret|password|passwd|pwd|token|auth)[\s]*[:=][\s]*["|']([^"|']{8,})["|']`),
			"base64_creds":   regexp.MustCompile(`[A-Za-z0-9+/]{40,}={0,2}`),
			"hex_secret":     regexp.MustCompile(`[0-9a-fA-F]{32,}`),
		},
		entropy: NewEntropyAnalyzer(),
	}
}

func (s *SecretsScanner) Name() string {
	return "Secrets Detection Scanner"
}

func (s *SecretsScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	// Define files to skip
	skipPatterns := []string{
		"*.test", "*.md", "*.lock", "vendor/*", "node_modules/*",
	}

	err := filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories and binary files
		if info.IsDir() || info.Size() > 10*1024*1024 { // Skip files > 10MB
			return nil
		}

		// Check skip patterns
		for _, pattern := range skipPatterns {
			if matched, _ := filepath.Match(pattern, file); matched {
				return nil
			}
		}

		// Scan file for secrets
		fileIssues, err := s.scanFile(file)
		if err != nil {
			return nil
		}

		issues = append(issues, fileIssues...)
		return nil
	})

	return issues, err
}

func (s *SecretsScanner) scanFile(filename string) ([]SecurityIssue, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Skip binary files
	if !isTextFile(content) {
		return nil, nil
	}

	var issues []SecurityIssue
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		// Skip comments in common languages
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") ||
			strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "/*") {
			continue
		}

		// Check each pattern
		for secretType, pattern := range s.patterns {
			if matches := pattern.FindAllString(line, -1); len(matches) > 0 {
				for _, match := range matches {
					// Skip if it's a placeholder or example
					if isPlaceholder(match) {
						continue
					}

					// Check entropy for generic patterns
					if secretType == "generic_secret" || secretType == "hex_secret" {
						if s.entropy.Calculate(match) < 4.0 {
							continue // Low entropy, likely not a real secret
						}
					}

					severity := "CRITICAL"
					if secretType == "generic_secret" {
						severity = "HIGH"
					}

					issues = append(issues, SecurityIssue{
						Type:        "EXPOSED_SECRET",
						Severity:    severity,
						Title:       fmt.Sprintf("Exposed %s", formatSecretType(secretType)),
						Description: fmt.Sprintf("Found potential %s in source code", secretType),
						Location: IssueLocation{
							File: filename,
							Line: i + 1,
						},
						Evidence: map[string]string{
							"type":    secretType,
							"pattern": maskSecret(match),
						},
						CWE:         "CWE-798",
						OWASP:       "A07:2021",
						Remediation: "Remove secret from code and use environment variables or secret management",
					})
				}
			}
		}

		// High entropy check for potential secrets
		tokens := strings.Fields(line)
		for _, token := range tokens {
			if len(token) > 20 && s.entropy.Calculate(token) > 5.0 {
				// High entropy string that might be a secret
				if !isCommonHighEntropy(token) {
					issues = append(issues, SecurityIssue{
						Type:        "HIGH_ENTROPY_STRING",
						Severity:    "MEDIUM",
						Title:       "High entropy string detected",
						Description: "String with high randomness detected, could be a secret",
						Location: IssueLocation{
							File: filename,
							Line: i + 1,
						},
						Evidence: map[string]string{
							"entropy": fmt.Sprintf("%.2f", s.entropy.Calculate(token)),
							"preview": maskSecret(token),
						},
						Remediation: "Review if this is a secret and move to secure storage",
					})
				}
			}
		}
	}

	return issues, nil
}

// EntropyAnalyzer calculates Shannon entropy
type EntropyAnalyzer struct{}

func NewEntropyAnalyzer() *EntropyAnalyzer {
	return &EntropyAnalyzer{}
}

func (e *EntropyAnalyzer) Calculate(s string) float64 {
	if len(s) == 0 {
		return 0.0
	}

	// Count character frequencies
	freq := make(map[rune]int)
	for _, c := range s {
		freq[c]++
	}

	// Calculate entropy
	length := float64(len(s))
	entropy := 0.0

	for _, count := range freq {
		p := float64(count) / length
		if p > 0 {
			entropy -= p * (p * 2) // Simplified log2
		}
	}

	return entropy
}

// Helper functions

func isTextFile(content []byte) bool {
	// Simple check for binary content
	for _, b := range content[:min(512, len(content))] {
		if b == 0 {
			return false
		}
	}
	return true
}

func isPlaceholder(s string) bool {
	placeholders := []string{
		"YOUR_", "XXXX", "your-", "example", "test", "demo",
		"<", ">", "${", "{{", "REPLACE", "CHANGE_ME",
	}

	lower := strings.ToLower(s)
	for _, p := range placeholders {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}

	return false
}

func isCommonHighEntropy(s string) bool {
	// Common high entropy strings that aren't secrets
	patterns := []string{
		".min.js", ".min.css", "node_modules",
		"vendor/", "dist/", "build/",
	}

	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}

	// Check if it's a hash (common in go.sum, package-lock.json)
	if regexp.MustCompile(`^[a-f0-9]{32,}$`).MatchString(s) {
		return true
	}

	return false
}

func formatSecretType(t string) string {
	replacements := map[string]string{
		"aws_key":        "AWS Access Key",
		"aws_secret":     "AWS Secret Key",
		"github_token":   "GitHub Token",
		"slack_token":    "Slack Token",
		"google_api":     "Google API Key",
		"rsa_private":    "RSA Private Key",
		"ssh_private":    "SSH Private Key",
		"pgp_private":    "PGP Private Key",
		"generic_secret": "Secret/Password",
		"base64_creds":   "Base64 Encoded Credentials",
		"hex_secret":     "Hex Encoded Secret",
	}

	if formatted, ok := replacements[t]; ok {
		return formatted
	}
	return t
}

func maskSecret(s string) string {
	if len(s) <= 8 {
		return "[REDACTED]"
	}

	// Show first 4 and last 4 characters
	return s[:4] + "..." + s[len(s)-4:]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
