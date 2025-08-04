package probe

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// JSONDissector analyzes JSON data for vulnerabilities.
type JSONDissector struct {
	credHunter *CredentialHunter
}

// NewJSONDissector creates a new JSON dissector.
func NewJSONDissector() *JSONDissector {
	return &JSONDissector{
		credHunter: NewCredentialHunter(),
	}
}

// Identify checks if data appears to be JSON.
func (d *JSONDissector) Identify(data []byte) (bool, float64) {
	// Quick checks for JSON
	str := strings.TrimSpace(string(data))
	if len(str) == 0 {
		return false, 0
	}

	// Check for JSON object or array markers
	if (strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}")) ||
		(strings.HasPrefix(str, "[") && strings.HasSuffix(str, "]")) {
		// Try to parse
		var v interface{}
		if err := json.Unmarshal([]byte(str), &v); err == nil {
			return true, 0.95
		}
		return true, 0.5 // Looks like JSON but doesn't parse
	}

	// Check for JSON-like patterns
	if strings.Contains(str, "\"") && strings.Contains(str, ":") {
		return true, 0.3
	}

	return false, 0
}

// Dissect parses JSON data.
func (d *JSONDissector) Dissect(data []byte) (*Frame, error) {
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	frame := &Frame{
		Protocol: "json",
		Fields:   make(map[string]interface{}),
		Raw:      data,
	}

	// Flatten JSON structure for analysis
	d.flattenJSON("", parsed, frame.Fields)

	return frame, nil
}

// FindVulnerabilities searches for security issues in JSON.
func (d *JSONDissector) FindVulnerabilities(frame *Frame) []StreamVulnerability {
	vulns := []StreamVulnerability{}

	// Check for credentials in field names and values
	for key, value := range frame.Fields {
		keyLower := strings.ToLower(key)
		valueStr := fmt.Sprintf("%v", value)

		// Check field names that suggest credentials
		if d.isSensitiveField(keyLower) {
			vuln := StreamVulnerability{
				ID:         fmt.Sprintf("JSON-%d", len(vulns)),
				Type:       "credential_exposure",
				Subtype:    d.classifyField(keyLower),
				Evidence:   d.redactValue(keyLower, valueStr),
				Context:    fmt.Sprintf("Field: %s", key),
				Confidence: 0.85,
			}

			// Adjust severity based on content
			if strings.Contains(keyLower, "password") || strings.Contains(keyLower, "secret") {
				vuln.Severity = "critical"
			} else if strings.Contains(keyLower, "key") || strings.Contains(keyLower, "token") {
				vuln.Severity = "high"
			} else {
				vuln.Severity = "medium"
			}

			vulns = append(vulns, vuln)
		}

		// Also check for patterns in values
		creds := d.credHunter.Hunt([]byte(valueStr))
		for _, cred := range creds {
			vuln := StreamVulnerability{
				ID:         fmt.Sprintf("JSON-%d", len(vulns)),
				Type:       "credential_pattern",
				Subtype:    cred.Type,
				Evidence:   cred.Redacted,
				Context:    fmt.Sprintf("Found in field: %s", key),
				Confidence: cred.Confidence,
				Severity:   cred.Severity,
			}
			vulns = append(vulns, vuln)
		}
	}

	return vulns
}

// flattenJSON recursively flattens JSON structure.
func (d *JSONDissector) flattenJSON(prefix string, data interface{}, result map[string]interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			newKey := key
			if prefix != "" {
				newKey = prefix + "." + key
			}
			d.flattenJSON(newKey, value, result)
		}
	case []interface{}:
		for i, value := range v {
			newKey := fmt.Sprintf("%s[%d]", prefix, i)
			d.flattenJSON(newKey, value, result)
		}
	default:
		result[prefix] = v
	}
}

// isSensitiveField checks if a field name suggests sensitive data.
func (d *JSONDissector) isSensitiveField(fieldName string) bool {
	sensitivePatterns := []string{
		"password", "passwd", "pwd",
		"secret", "token", "key",
		"auth", "credential", "cred",
		"api", "private", "cert",
		"bearer", "oauth", "jwt",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(fieldName, pattern) {
			return true
		}
	}
	return false
}

// classifyField determines the type of sensitive field.
func (d *JSONDissector) classifyField(fieldName string) string {
	if strings.Contains(fieldName, "password") || strings.Contains(fieldName, "passwd") {
		return "password"
	}
	if strings.Contains(fieldName, "api") && strings.Contains(fieldName, "key") {
		return "api_key"
	}
	if strings.Contains(fieldName, "token") {
		return "token"
	}
	if strings.Contains(fieldName, "secret") {
		return "secret"
	}
	return "credential"
}

// redactValue redacts sensitive values based on type.
func (d *JSONDissector) redactValue(fieldName, value string) string {
	if len(value) <= 4 {
		return "****"
	}
	if strings.Contains(fieldName, "token") || strings.Contains(fieldName, "key") {
		return value[:4] + "****"
	}
	return "****"
}

// SQLDissector analyzes SQL queries for vulnerabilities.
type SQLDissector struct {
	credHunter *CredentialHunter
	sqlPattern *regexp.Regexp
}

// NewSQLDissector creates a new SQL dissector.
func NewSQLDissector() *SQLDissector {
	return &SQLDissector{
		credHunter: NewCredentialHunter(),
		sqlPattern: regexp.MustCompile(`(?i)\b(SELECT|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP|GRANT|IDENTIFIED\s+BY)\b`),
	}
}

// Identify checks if data appears to be SQL.
func (d *SQLDissector) Identify(data []byte) (bool, float64) {
	str := string(data)

	// Check for SQL keywords
	if d.sqlPattern.MatchString(str) {
		confidence := 0.5

		// Higher confidence for certain patterns
		if regexp.MustCompile(`(?i)SELECT\s+.*\s+FROM\s+`).MatchString(str) {
			confidence = 0.9
		} else if regexp.MustCompile(`(?i)INSERT\s+INTO\s+`).MatchString(str) {
			confidence = 0.9
		} else if regexp.MustCompile(`(?i)UPDATE\s+.*\s+SET\s+`).MatchString(str) {
			confidence = 0.9
		}

		return true, confidence
	}

	return false, 0
}

// Dissect parses SQL queries.
func (d *SQLDissector) Dissect(data []byte) (*Frame, error) {
	frame := &Frame{
		Protocol: "sql",
		Fields:   make(map[string]interface{}),
		Raw:      data,
	}

	query := string(data)
	frame.Fields["query"] = query

	// Extract query type
	queryUpper := strings.ToUpper(strings.TrimSpace(query))
	for _, qtype := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "GRANT"} {
		if strings.HasPrefix(queryUpper, qtype) {
			frame.Fields["query_type"] = qtype
			break
		}
	}

	// Extract table names
	tables := d.extractTableNames(query)
	if len(tables) > 0 {
		frame.Fields["tables"] = tables
	}

	return frame, nil
}

// FindVulnerabilities searches for security issues in SQL.
func (d *SQLDissector) FindVulnerabilities(frame *Frame) []StreamVulnerability {
	vulns := []StreamVulnerability{}
	query := frame.Fields["query"].(string)

	// Check for passwords in SQL
	passwordPatterns := []struct {
		pattern *regexp.Regexp
		subtype string
		context string
	}{
		{
			regexp.MustCompile(`(?i)IDENTIFIED\s+BY\s+['"]([^'"]+)['"]`),
			"database_password",
			"CREATE USER statement",
		},
		{
			regexp.MustCompile(`(?i)PASSWORD\s*=\s*['"]([^'"]+)['"]`),
			"database_password",
			"Password in query",
		},
		{
			regexp.MustCompile(`(?i)PASSWORD\s*\(\s*['"]([^'"]+)['"]\s*\)`),
			"database_password",
			"PASSWORD() function",
		},
	}

	for _, p := range passwordPatterns {
		if matches := p.pattern.FindStringSubmatch(query); matches != nil {
			password := matches[1]
			vuln := StreamVulnerability{
				ID:         fmt.Sprintf("SQL-%d", len(vulns)),
				Type:       "credential_exposure",
				Subtype:    p.subtype,
				Evidence:   "****",
				Context:    p.context,
				Confidence: 0.95,
				Severity:   "critical",
			}
			vulns = append(vulns, vuln)

			// Log the actual password for internal use (not displayed)
			_ = password
		}
	}

	// Check for connection strings
	connStrPattern := regexp.MustCompile(`(?i)(mysql|postgres|postgresql|mssql|oracle):\/\/([^:]+):([^@]+)@`)
	if matches := connStrPattern.FindStringSubmatch(query); matches != nil {
		vuln := StreamVulnerability{
			ID:         fmt.Sprintf("SQL-%d", len(vulns)),
			Type:       "credential_exposure",
			Subtype:    "connection_string",
			Evidence:   matches[1] + "://****:****@...",
			Context:    "Database connection string in query",
			Confidence: 0.95,
			Severity:   "critical",
		}
		vulns = append(vulns, vuln)
	}

	// Check for potential SQL injection vulnerabilities
	if d.detectSQLInjection(query) {
		vuln := StreamVulnerability{
			ID:         fmt.Sprintf("SQL-%d", len(vulns)),
			Type:       "sql_injection",
			Subtype:    "potential",
			Evidence:   "[Query contains dynamic values]",
			Context:    "Possible SQL injection vector",
			Confidence: 0.7,
			Severity:   "high",
		}
		vulns = append(vulns, vuln)
	}

	// Run general credential hunter on query
	creds := d.credHunter.Hunt([]byte(query))
	for _, cred := range creds {
		vuln := StreamVulnerability{
			ID:         fmt.Sprintf("SQL-%d", len(vulns)),
			Type:       "credential_pattern",
			Subtype:    cred.Type,
			Evidence:   cred.Redacted,
			Context:    "Found in SQL query",
			Confidence: cred.Confidence,
			Severity:   cred.Severity,
		}
		vulns = append(vulns, vuln)
	}

	return vulns
}

// extractTableNames extracts table names from SQL query.
func (d *SQLDissector) extractTableNames(query string) []string {
	tables := []string{}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
		regexp.MustCompile(`(?i)INTO\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
		regexp.MustCompile(`(?i)UPDATE\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
		regexp.MustCompile(`(?i)TABLE\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(query, -1)
		for _, match := range matches {
			if len(match) > 1 {
				tables = append(tables, match[1])
			}
		}
	}

	return tables
}

// detectSQLInjection checks for potential SQL injection patterns.
func (d *SQLDissector) detectSQLInjection(query string) bool {
	injectionPatterns := []string{
		"' OR '",
		"\" OR \"",
		"1=1",
		"1' OR '1'='1",
		"admin'--",
		"';--",
		"\";--",
		"UNION SELECT",
		"UNION ALL SELECT",
	}

	queryLower := strings.ToLower(query)
	for _, pattern := range injectionPatterns {
		if strings.Contains(queryLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// PlainTextDissector analyzes plain text for credentials.
type PlainTextDissector struct {
	credHunter *CredentialHunter
}

// NewPlainTextDissector creates a new plain text dissector.
func NewPlainTextDissector() *PlainTextDissector {
	return &PlainTextDissector{
		credHunter: NewCredentialHunter(),
	}
}

// Identify always returns true as fallback.
func (d *PlainTextDissector) Identify(_ []byte) (bool, float64) {
	// Plain text is the fallback dissector
	return true, 0.1
}

// Dissect treats data as plain text.
func (d *PlainTextDissector) Dissect(data []byte) (*Frame, error) {
	frame := &Frame{
		Protocol: "plaintext",
		Fields:   make(map[string]interface{}),
		Raw:      data,
	}

	frame.Fields["text"] = string(data)
	frame.Fields["length"] = len(data)

	return frame, nil
}

// FindVulnerabilities searches for credentials in plain text.
func (d *PlainTextDissector) FindVulnerabilities(frame *Frame) []StreamVulnerability {
	vulns := []StreamVulnerability{}
	text := frame.Fields["text"].(string)

	// Run credential hunter
	creds := d.credHunter.Hunt([]byte(text))
	for i, cred := range creds {
		vuln := StreamVulnerability{
			ID:         fmt.Sprintf("TEXT-%d", i),
			Type:       "credential_pattern",
			Subtype:    cred.Type,
			Evidence:   cred.Redacted,
			Context:    "Found in plain text stream",
			Confidence: cred.Confidence,
			Severity:   cred.Severity,
		}
		vulns = append(vulns, vuln)
	}

	return vulns
}
