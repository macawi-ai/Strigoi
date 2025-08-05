package siem

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// DataMaskingEngine handles sensitive data masking for SIEM integration
type DataMaskingEngine struct {
	// Patterns for sensitive data detection
	creditCardPattern *regexp.Regexp
	ssnPattern        *regexp.Regexp
	emailPattern      *regexp.Regexp
	ipPattern         *regexp.Regexp
	apiKeyPatterns    []*regexp.Regexp

	// Configuration
	config MaskingConfig
}

// MaskingConfig holds masking configuration
type MaskingConfig struct {
	MaskCreditCards bool
	MaskSSN         bool
	MaskEmails      bool
	MaskAPIKeys     bool
	MaskIPs         bool
	PreserveDomain  bool   // For emails, preserve domain
	HashSensitive   bool   // Use hash instead of static mask
	Salt            string // Salt for hashing
}

// NewDataMaskingEngine creates a new masking engine with default config
func NewDataMaskingEngine() *DataMaskingEngine {
	return &DataMaskingEngine{
		creditCardPattern: regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
		ssnPattern:        regexp.MustCompile(`\b\d{3}-?\d{2}-?\d{4}\b`),
		emailPattern:      regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
		ipPattern:         regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
		apiKeyPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\b(sk|pk|api|key|token)[-_][A-Za-z0-9_-]+\b`),
			regexp.MustCompile(`\bBearer\s+[A-Za-z0-9\-._~+/]+=*\b`),
			regexp.MustCompile(`\b[A-Za-z0-9]{32,}\b`), // Generic long strings
		},
		config: MaskingConfig{
			MaskCreditCards: true,
			MaskSSN:         true,
			MaskEmails:      true,
			MaskAPIKeys:     true,
			MaskIPs:         false, // IPs often needed for security analysis
			PreserveDomain:  true,
			HashSensitive:   false,
			Salt:            "strigoi-default-salt",
		},
	}
}

// MaskString masks sensitive data in a string
func (m *DataMaskingEngine) MaskString(input string) string {
	result := input

	// Mask credit cards
	if m.config.MaskCreditCards {
		result = m.maskCreditCards(result)
	}

	// Mask SSNs
	if m.config.MaskSSN {
		result = m.maskSSNs(result)
	}

	// Mask emails
	if m.config.MaskEmails {
		result = m.maskEmails(result)
	}

	// Mask API keys
	if m.config.MaskAPIKeys {
		result = m.maskAPIKeys(result)
	}

	// Mask IPs
	if m.config.MaskIPs {
		result = m.maskIPs(result)
	}

	return result
}

// MaskCreditCard masks a credit card number
func (m *DataMaskingEngine) MaskCreditCard(cc string) string {
	// Remove spaces and dashes
	cleaned := strings.ReplaceAll(strings.ReplaceAll(cc, " ", ""), "-", "")

	if len(cleaned) < 12 {
		return "[INVALID_CC]"
	}

	if m.config.HashSensitive {
		return m.hashValue(cleaned, "CC")
	}

	// Show first 4 and last 4 digits
	return fmt.Sprintf("%s****%s", cleaned[:4], cleaned[len(cleaned)-4:])
}

// MaskEmail masks an email address
func (m *DataMaskingEngine) MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "[INVALID_EMAIL]"
	}

	username := parts[0]
	domain := parts[1]

	if m.config.HashSensitive {
		return m.hashValue(email, "EMAIL")
	}

	// Mask username
	var maskedUsername string
	if len(username) <= 3 {
		maskedUsername = "***"
	} else {
		maskedUsername = username[:2] + "***"
	}

	if m.config.PreserveDomain {
		return maskedUsername + "@" + domain
	}

	return maskedUsername + "@***.***"
}

// MaskSSN masks a social security number
func (m *DataMaskingEngine) MaskSSN(ssn string) string {
	// Remove dashes
	cleaned := strings.ReplaceAll(ssn, "-", "")

	if len(cleaned) != 9 {
		return "[INVALID_SSN]"
	}

	if m.config.HashSensitive {
		return m.hashValue(cleaned, "SSN")
	}

	// Show last 4 digits only
	return "***-**-" + cleaned[5:]
}

// MaskAPIKey masks an API key or token
func (m *DataMaskingEngine) MaskAPIKey(key string) string {
	if len(key) < 8 {
		return "[INVALID_KEY]"
	}

	if m.config.HashSensitive {
		return m.hashValue(key, "APIKEY")
	}

	// Determine prefix (sk_, pk_, etc)
	prefix := ""
	if idx := strings.IndexAny(key, "_-"); idx > 0 && idx < 5 {
		prefix = key[:idx+1]
	}

	// Show prefix and last 4 characters
	if len(key) > 10 {
		return fmt.Sprintf("%s***%s", prefix, key[len(key)-4:])
	}

	return prefix + "***"
}

// MaskIP masks an IP address
func (m *DataMaskingEngine) MaskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return "[INVALID_IP]"
	}

	if m.config.HashSensitive {
		return m.hashValue(ip, "IP")
	}

	// Mask last octet
	return fmt.Sprintf("%s.%s.%s.xxx", parts[0], parts[1], parts[2])
}

// MaskGeneric provides generic masking for unknown sensitive data
func (m *DataMaskingEngine) MaskGeneric(value string) string {
	if m.config.HashSensitive {
		return m.hashValue(value, "DATA")
	}

	if len(value) <= 8 {
		return "[REDACTED]"
	}

	// Show first 2 and last 2 characters
	return fmt.Sprintf("%s***%s", value[:2], value[len(value)-2:])
}

// Internal masking methods

func (m *DataMaskingEngine) maskCreditCards(input string) string {
	return m.creditCardPattern.ReplaceAllStringFunc(input, func(match string) string {
		return m.MaskCreditCard(match)
	})
}

func (m *DataMaskingEngine) maskSSNs(input string) string {
	return m.ssnPattern.ReplaceAllStringFunc(input, func(match string) string {
		return m.MaskSSN(match)
	})
}

func (m *DataMaskingEngine) maskEmails(input string) string {
	return m.emailPattern.ReplaceAllStringFunc(input, func(match string) string {
		return m.MaskEmail(match)
	})
}

func (m *DataMaskingEngine) maskAPIKeys(input string) string {
	result := input
	for _, pattern := range m.apiKeyPatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Skip if it looks like a regular word
			if len(match) < 20 && !strings.Contains(match, "_") && !strings.Contains(match, "-") {
				return match
			}
			return m.MaskAPIKey(match)
		})
	}
	return result
}

func (m *DataMaskingEngine) maskIPs(input string) string {
	return m.ipPattern.ReplaceAllStringFunc(input, func(match string) string {
		// Validate it's a real IP (not version numbers, etc)
		parts := strings.Split(match, ".")
		if len(parts) != 4 {
			return match
		}

		for _, part := range parts {
			var num int
			if _, err := fmt.Sscanf(part, "%d", &num); err != nil || num > 255 {
				return match // Not a valid IP
			}
		}

		return m.MaskIP(match)
	})
}

// hashValue creates a consistent hash for sensitive values
func (m *DataMaskingEngine) hashValue(value, prefix string) string {
	h := sha256.New()
	h.Write([]byte(m.config.Salt))
	h.Write([]byte(value))
	hash := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("[%s-%s]", prefix, hash[:8])
}

// MaskingAnalyzer analyzes data to detect sensitive information
type MaskingAnalyzer struct {
	engine *DataMaskingEngine
	stats  MaskingStats
}

// MaskingStats tracks masking statistics
type MaskingStats struct {
	TotalFields      int
	MaskedFields     int
	CreditCards      int
	SSNs             int
	Emails           int
	APIKeys          int
	IPs              int
	MaskingDecisions []MaskingDecision
}

// MaskingDecision records a masking decision
type MaskingDecision struct {
	FieldName     string
	OriginalValue string
	MaskedValue   string
	DataType      string
	Reason        string
}

// NewMaskingAnalyzer creates a new masking analyzer
func NewMaskingAnalyzer() *MaskingAnalyzer {
	return &MaskingAnalyzer{
		engine: NewDataMaskingEngine(),
		stats:  MaskingStats{},
	}
}

// AnalyzeAndMask analyzes fields and applies masking
func (a *MaskingAnalyzer) AnalyzeAndMask(fields map[string]interface{}) (map[string]interface{}, MaskingStats) {
	result := make(map[string]interface{})
	a.stats = MaskingStats{
		MaskingDecisions: []MaskingDecision{},
	}

	for name, value := range fields {
		a.stats.TotalFields++

		stringValue := fmt.Sprintf("%v", value)
		maskedValue := stringValue
		dataType := ""
		masked := false

		// Check for credit card
		if a.engine.creditCardPattern.MatchString(stringValue) {
			maskedValue = a.engine.MaskCreditCard(stringValue)
			dataType = "credit_card"
			a.stats.CreditCards++
			masked = true
		} else if a.engine.ssnPattern.MatchString(stringValue) {
			maskedValue = a.engine.MaskSSN(stringValue)
			dataType = "ssn"
			a.stats.SSNs++
			masked = true
		} else if a.engine.emailPattern.MatchString(stringValue) {
			maskedValue = a.engine.MaskEmail(stringValue)
			dataType = "email"
			a.stats.Emails++
			masked = true
		} else if a.isSensitiveFieldName(name) {
			// Check field name for hints
			maskedValue = a.engine.MaskGeneric(stringValue)
			dataType = "sensitive_field"
			masked = true
		} else {
			// Check for API keys
			for _, pattern := range a.engine.apiKeyPatterns {
				if pattern.MatchString(stringValue) && len(stringValue) > 20 {
					maskedValue = a.engine.MaskAPIKey(stringValue)
					dataType = "api_key"
					a.stats.APIKeys++
					masked = true
					break
				}
			}
		}

		if masked {
			a.stats.MaskedFields++
			a.stats.MaskingDecisions = append(a.stats.MaskingDecisions, MaskingDecision{
				FieldName:     name,
				OriginalValue: stringValue,
				MaskedValue:   maskedValue,
				DataType:      dataType,
				Reason:        fmt.Sprintf("Detected %s pattern", dataType),
			})
			result[name] = maskedValue
		} else {
			result[name] = value
		}
	}

	return result, a.stats
}

// isSensitiveFieldName checks if field name indicates sensitive data
func (a *MaskingAnalyzer) isSensitiveFieldName(name string) bool {
	sensitiveNames := []string{
		"password", "passwd", "pwd", "secret",
		"token", "api_key", "apikey", "auth",
		"private_key", "priv_key", "credential",
		"credit_card", "cc_number", "cvv",
		"ssn", "social_security", "pin",
		"account_number", "routing_number",
	}

	nameLower := strings.ToLower(name)
	for _, sensitive := range sensitiveNames {
		if strings.Contains(nameLower, sensitive) {
			return true
		}
	}

	return false
}

// GenerateMaskingReport creates a report of masking activities
func (a *MaskingAnalyzer) GenerateMaskingReport() string {
	report := fmt.Sprintf(`Masking Report
==============
Total Fields: %d
Masked Fields: %d (%.1f%%)

By Type:
- Credit Cards: %d
- SSNs: %d
- Emails: %d
- API Keys: %d
- IPs: %d

Masking Decisions:
`,
		a.stats.TotalFields,
		a.stats.MaskedFields,
		float64(a.stats.MaskedFields)/float64(a.stats.TotalFields)*100,
		a.stats.CreditCards,
		a.stats.SSNs,
		a.stats.Emails,
		a.stats.APIKeys,
		a.stats.IPs,
	)

	for i, decision := range a.stats.MaskingDecisions {
		report += fmt.Sprintf("%d. Field: %s\n   Type: %s\n   Reason: %s\n   Original: %s\n   Masked: %s\n\n",
			i+1,
			decision.FieldName,
			decision.DataType,
			decision.Reason,
			decision.OriginalValue,
			decision.MaskedValue,
		)
	}

	return report
}
