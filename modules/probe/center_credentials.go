package probe

import (
	"regexp"
	"strings"
	"time"
)

// CredentialHunter detects credentials in stream data.
type CredentialHunter struct {
	patterns map[string]*CredentialPattern
}

// CredentialPattern defines a pattern for detecting credentials.
type CredentialPattern struct {
	Type       string
	Regex      *regexp.Regexp
	Confidence float64
	Severity   string
	Redactor   func(string) string
}

// NewCredentialHunter creates a new credential detector.
func NewCredentialHunter() *CredentialHunter {
	h := &CredentialHunter{
		patterns: make(map[string]*CredentialPattern),
	}

	// Initialize patterns
	h.initializePatterns()
	return h
}

// initializePatterns sets up detection patterns.
func (h *CredentialHunter) initializePatterns() {
	// Database passwords
	h.addPattern("mysql_password", &CredentialPattern{
		Type:       "database_password",
		Regex:      regexp.MustCompile(`mysql://[^:]+:([^@]+)@`),
		Confidence: 0.95,
		Severity:   "critical",
		Redactor:   h.redactFull,
	})

	h.addPattern("postgres_password", &CredentialPattern{
		Type:       "database_password",
		Regex:      regexp.MustCompile(`postgres://[^:]+:([^@]+)@`),
		Confidence: 0.95,
		Severity:   "critical",
		Redactor:   h.redactFull,
	})

	h.addPattern("sql_password", &CredentialPattern{
		Type:       "database_password",
		Regex:      regexp.MustCompile(`(?i)(?:password|passwd|pwd)\s*[=:]\s*['"]?([^'"\s]+)['"]?`),
		Confidence: 0.85,
		Severity:   "high",
		Redactor:   h.redactFull,
	})

	// API Keys
	h.addPattern("openai_key", &CredentialPattern{
		Type:       "api_key",
		Regex:      regexp.MustCompile(`sk-[a-zA-Z0-9]{48}`),
		Confidence: 0.99,
		Severity:   "critical",
		Redactor:   h.redactAPIKey,
	})

	h.addPattern("github_token", &CredentialPattern{
		Type:       "api_key",
		Regex:      regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
		Confidence: 0.99,
		Severity:   "critical",
		Redactor:   h.redactAPIKey,
	})

	h.addPattern("aws_key", &CredentialPattern{
		Type:       "api_key",
		Regex:      regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		Confidence: 0.95,
		Severity:   "critical",
		Redactor:   h.redactAPIKey,
	})

	h.addPattern("aws_secret", &CredentialPattern{
		Type:       "api_secret",
		Regex:      regexp.MustCompile(`(?i)aws_secret_access_key\s*[=:]\s*['"]?([a-zA-Z0-9/+=]{40})['"]?`),
		Confidence: 0.95,
		Severity:   "critical",
		Redactor:   h.redactFull,
	})

	// OAuth/JWT
	h.addPattern("bearer_token", &CredentialPattern{
		Type:       "bearer_token",
		Regex:      regexp.MustCompile(`(?i)bearer\s+([a-zA-Z0-9\-._~+/]+=*)`),
		Confidence: 0.85,
		Severity:   "high",
		Redactor:   h.redactToken,
	})

	h.addPattern("jwt_token", &CredentialPattern{
		Type:       "jwt_token",
		Regex:      regexp.MustCompile(`eyJ[a-zA-Z0-9\-_]+\.eyJ[a-zA-Z0-9\-_]+\.[a-zA-Z0-9\-_]+`),
		Confidence: 0.95,
		Severity:   "high",
		Redactor:   h.redactJWT,
	})

	// Generic patterns
	h.addPattern("generic_key", &CredentialPattern{
		Type:       "api_key",
		Regex:      regexp.MustCompile(`(?i)(?:api[_-]?key|apikey)\s*[=:]\s*['"]?([a-zA-Z0-9\-_]{20,})['"]?`),
		Confidence: 0.75,
		Severity:   "high",
		Redactor:   h.redactAPIKey,
	})

	h.addPattern("generic_token", &CredentialPattern{
		Type:       "token",
		Regex:      regexp.MustCompile(`(?i)(?:auth[_-]?token|token)\s*[=:]\s*['"]?([a-zA-Z0-9\-_]{20,})['"]?`),
		Confidence: 0.75,
		Severity:   "high",
		Redactor:   h.redactToken,
	})

	h.addPattern("generic_secret", &CredentialPattern{
		Type:       "secret",
		Regex:      regexp.MustCompile(`(?i)(?:secret|client[_-]?secret)\s*[=:]\s*['"]?([a-zA-Z0-9\-_]{16,})['"]?`),
		Confidence: 0.80,
		Severity:   "high",
		Redactor:   h.redactFull,
	})

	// Private keys
	h.addPattern("private_key_begin", &CredentialPattern{
		Type:       "private_key",
		Regex:      regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA )?PRIVATE KEY-----`),
		Confidence: 0.99,
		Severity:   "critical",
		Redactor:   h.redactPrivateKey,
	})

	// Credit cards (PCI compliance)
	h.addPattern("credit_card", &CredentialPattern{
		Type:       "credit_card",
		Regex:      regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12})\b`),
		Confidence: 0.90,
		Severity:   "critical",
		Redactor:   h.redactCreditCard,
	})

	// SSH keys
	h.addPattern("ssh_private_key", &CredentialPattern{
		Type:       "ssh_key",
		Regex:      regexp.MustCompile(`ssh-(?:rsa|ed25519|ecdsa|dss)\s+[A-Za-z0-9+/=]+`),
		Confidence: 0.95,
		Severity:   "critical",
		Redactor:   h.redactSSHKey,
	})
}

// addPattern registers a credential pattern.
func (h *CredentialHunter) addPattern(name string, pattern *CredentialPattern) {
	h.patterns[name] = pattern
}

// Hunt searches for credentials in data.
func (h *CredentialHunter) Hunt(data []byte) []Credential {
	credentials := []Credential{}
	dataStr := string(data)

	for name, pattern := range h.patterns {
		matches := pattern.Regex.FindAllStringSubmatch(dataStr, -1)
		for _, match := range matches {
			value := match[0]
			if len(match) > 1 {
				value = match[1] // Use captured group if available
			}

			cred := Credential{
				Type:       pattern.Type,
				Value:      value,
				Redacted:   pattern.Redactor(value),
				Confidence: pattern.Confidence,
				Severity:   pattern.Severity,
				Timestamp:  time.Now(),
			}

			// Skip if it looks like a placeholder
			if h.isPlaceholder(value) {
				cred.Confidence *= 0.5
			}

			// Skip low confidence matches
			if cred.Confidence >= 0.5 {
				credentials = append(credentials, cred)
			}

			// Log pattern name for debugging
			_ = name
		}
	}

	return h.deduplicateCredentials(credentials)
}

// isPlaceholder checks if a value looks like a placeholder.
func (h *CredentialHunter) isPlaceholder(value string) bool {
	placeholders := []string{
		"xxxxxx",
		"******",
		"changeme",
		"password",
		"secret",
		"token",
		"your-",
		"<your",
		"${",
		"example",
		"test",
		"demo",
		"sample",
	}

	lowerValue := strings.ToLower(value)
	for _, ph := range placeholders {
		if strings.Contains(lowerValue, ph) {
			return true
		}
	}

	// Check for all same character
	if len(value) > 4 {
		firstChar := value[0]
		allSame := true
		for _, ch := range value {
			if byte(ch) != firstChar {
				allSame = false
				break
			}
		}
		if allSame {
			return true
		}
	}

	return false
}

// deduplicateCredentials removes duplicate findings.
func (h *CredentialHunter) deduplicateCredentials(creds []Credential) []Credential {
	seen := make(map[string]bool)
	unique := []Credential{}

	for _, cred := range creds {
		key := cred.Type + ":" + cred.Value
		if !seen[key] {
			seen[key] = true
			unique = append(unique, cred)
		}
	}

	return unique
}

// Redaction functions

func (h *CredentialHunter) redactFull(_ string) string {
	return "****"
}

func (h *CredentialHunter) redactAPIKey(value string) string {
	if len(value) <= 8 {
		return h.redactFull(value)
	}
	return value[:4] + "****" + value[len(value)-4:]
}

func (h *CredentialHunter) redactToken(value string) string {
	if len(value) <= 10 {
		return h.redactFull(value)
	}
	return value[:6] + "......" + value[len(value)-4:]
}

func (h *CredentialHunter) redactJWT(value string) string {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return h.redactFull(value)
	}
	// Show header but redact payload and signature
	return parts[0] + ".****.**"
}

func (h *CredentialHunter) redactPrivateKey(_ string) string {
	return "-----BEGIN PRIVATE KEY----- [REDACTED]"
}

func (h *CredentialHunter) redactCreditCard(value string) string {
	if len(value) < 8 {
		return h.redactFull(value)
	}
	// Show last 4 digits only
	return "****-****-****-" + value[len(value)-4:]
}

func (h *CredentialHunter) redactSSHKey(value string) string {
	parts := strings.Fields(value)
	if len(parts) < 2 {
		return h.redactFull(value)
	}
	// Show key type but redact key material
	return parts[0] + " [REDACTED]"
}
