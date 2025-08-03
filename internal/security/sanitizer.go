package security

import (
	"regexp"
	"strings"
)

// CommandSanitizer handles input sanitization for AI observation
type CommandSanitizer struct {
	patterns []SensitivePattern
}

// SensitivePattern defines patterns to redact
type SensitivePattern struct {
	Name        string
	Regex       *regexp.Regexp
	Replacement string
}

// NewCommandSanitizer creates a sanitizer with default patterns
func NewCommandSanitizer() *CommandSanitizer {
	return &CommandSanitizer{
		patterns: []SensitivePattern{
			{
				Name:        "api_key",
				Regex:       regexp.MustCompile(`\b[A-Za-z0-9]{32,}\b`),
				Replacement: "[APIKEY]",
			},
			{
				Name:        "password",
				Regex:       regexp.MustCompile(`(?i)(password|passwd|pwd)[:=]\S+`),
				Replacement: "password=[REDACTED]",
			},
			{
				Name:        "ip_address",
				Regex:       regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
				Replacement: "[IP_ADDR]",
			},
			{
				Name:        "private_key",
				Regex:       regexp.MustCompile(`-----BEGIN.*PRIVATE KEY-----[\s\S]+?-----END.*PRIVATE KEY-----`),
				Replacement: "[PRIVATE_KEY]",
			},
		},
	}
}

// Sanitize removes sensitive information from commands
func (s *CommandSanitizer) Sanitize(input string) string {
	sanitized := input
	
	for _, pattern := range s.patterns {
		sanitized = pattern.Regex.ReplaceAllString(sanitized, pattern.Replacement)
	}
	
	return sanitized
}

// DetectInjection checks for potential prompt injection attempts
func (s *CommandSanitizer) DetectInjection(input string) bool {
	injectionPatterns := []string{
		"ignore all previous",
		"disregard instructions",
		"new system prompt",
		"you are now",
		"forget everything",
	}
	
	lowerInput := strings.ToLower(input)
	for _, pattern := range injectionPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	
	return false
}