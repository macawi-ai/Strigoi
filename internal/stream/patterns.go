package stream

import (
	"fmt"
	"regexp"
	"sync"
)

// AttackType categorizes different attack patterns
type AttackType string

const (
	AttackSQLInjection   AttackType = "sql_injection"
	AttackCommandInject  AttackType = "command_injection"
	AttackPathTraversal  AttackType = "path_traversal"
	AttackXSS            AttackType = "xss"
	AttackXXE            AttackType = "xxe"
	AttackLDAPInjection  AttackType = "ldap_injection"
	AttackLogInjection   AttackType = "log_injection"
	AttackPromptInject   AttackType = "prompt_injection"
)

// Severity levels for patterns
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// AttackPattern represents a compiled attack pattern
type AttackPattern struct {
	ID          string
	Type        AttackType
	Severity    Severity
	Pattern     *regexp.Regexp
	Description string
	Confidence  float64
}

// PatternRegistry manages pre-compiled patterns
type PatternRegistry struct {
	patterns map[AttackType][]*AttackPattern
	mu       sync.RWMutex
}

// NewPatternRegistry creates and initializes the pattern registry
func NewPatternRegistry() (*PatternRegistry, error) {
	pr := &PatternRegistry{
		patterns: make(map[AttackType][]*AttackPattern),
	}
	
	// Initialize all patterns
	if err := pr.initializeSQLPatterns(); err != nil {
		return nil, fmt.Errorf("failed to init SQL patterns: %w", err)
	}
	if err := pr.initializeCommandPatterns(); err != nil {
		return nil, fmt.Errorf("failed to init command patterns: %w", err)
	}
	if err := pr.initializePathPatterns(); err != nil {
		return nil, fmt.Errorf("failed to init path patterns: %w", err)
	}
	if err := pr.initializeXSSPatterns(); err != nil {
		return nil, fmt.Errorf("failed to init XSS patterns: %w", err)
	}
	if err := pr.initializePromptPatterns(); err != nil {
		return nil, fmt.Errorf("failed to init prompt patterns: %w", err)
	}
	
	return pr, nil
}

// initializeSQLPatterns loads SQL injection patterns
func (pr *PatternRegistry) initializeSQLPatterns() error {
	sqlPatterns := []struct {
		id          string
		pattern     string
		description string
		severity    Severity
		confidence  float64
	}{
		{
			"sql-union-1",
			`(?i)\bUNION\s+(ALL\s+)?SELECT\b`,
			"SQL UNION SELECT injection",
			SeverityCritical,
			0.95,
		},
		{
			"sql-or-1",
			`(?i)'\s*OR\s*'?\d*'?\s*=\s*'?\d*'?`,
			"SQL OR condition injection",
			SeverityHigh,
			0.90,
		},
		{
			"sql-comment-1",
			`(?i)(--|#|/\*|\*/|@@|@)`,
			"SQL comment injection",
			SeverityMedium,
			0.70,
		},
		{
			"sql-keywords-1",
			`(?i)\b(DROP|DELETE|TRUNCATE|ALTER|CREATE|INSERT|UPDATE)\s+(TABLE|DATABASE|SCHEMA|INDEX|VIEW)\b`,
			"SQL DDL injection",
			SeverityCritical,
			0.98,
		},
		{
			"sql-time-1",
			`(?i)\b(SLEEP|BENCHMARK|WAITFOR\s+DELAY|PG_SLEEP)\s*\(`,
			"SQL time-based injection",
			SeverityHigh,
			0.85,
		},
		{
			"sql-meta-1",
			`(?i)\b(INFORMATION_SCHEMA|SYSOBJECTS|SYSCOLUMNS|SYSUSERS)\b`,
			"SQL metadata access",
			SeverityHigh,
			0.80,
		},
	}
	
	return pr.compilePatterns(AttackSQLInjection, sqlPatterns)
}

// initializeCommandPatterns loads command injection patterns
func (pr *PatternRegistry) initializeCommandPatterns() error {
	cmdPatterns := []struct {
		id          string
		pattern     string
		description string
		severity    Severity
		confidence  float64
	}{
		{
			"cmd-semicolon-1",
			`;\s*(cat|ls|pwd|whoami|id|uname|wget|curl|nc|ncat)\b`,
			"Command chaining with semicolon",
			SeverityCritical,
			0.95,
		},
		{
			"cmd-pipe-1",
			`\|\s*(cat|grep|awk|sed|cut|sort|uniq|head|tail)\b`,
			"Command piping",
			SeverityHigh,
			0.85,
		},
		{
			"cmd-backtick-1",
			"`[^`]+`",
			"Command substitution with backticks",
			SeverityHigh,
			0.80,
		},
		{
			"cmd-dollar-1",
			`\$\([^)]+\)`,
			"Command substitution with $()",
			SeverityHigh,
			0.80,
		},
		{
			"cmd-redirect-1",
			`(>|>>|<|<<)\s*[/\w]+`,
			"Command output redirection",
			SeverityMedium,
			0.70,
		},
		{
			"cmd-escape-1",
			`\\x[0-9a-fA-F]{2}|\\[0-7]{3}`,
			"Hex/octal escape sequences",
			SeverityMedium,
			0.75,
		},
	}
	
	return pr.compilePatterns(AttackCommandInject, cmdPatterns)
}

// initializePathPatterns loads path traversal patterns
func (pr *PatternRegistry) initializePathPatterns() error {
	pathPatterns := []struct {
		id          string
		pattern     string
		description string
		severity    Severity
		confidence  float64
	}{
		{
			"path-dotdot-1",
			`\.\.(/|\\)`,
			"Directory traversal with ../",
			SeverityHigh,
			0.90,
		},
		{
			"path-absolute-1",
			`^/etc/(passwd|shadow|hosts|sudoers)`,
			"Absolute path to sensitive files",
			SeverityCritical,
			0.95,
		},
		{
			"path-windows-1",
			`[cC]:\\\\(windows|winnt|boot\.ini|system\.ini)`,
			"Windows path traversal",
			SeverityHigh,
			0.85,
		},
		{
			"path-encoded-1",
			`%2e%2e(%2f|%5c)|%252e%252e(%252f|%255c)`,
			"URL encoded path traversal",
			SeverityHigh,
			0.85,
		},
		{
			"path-unicode-1",
			`\\u002e\\u002e\\u002f|\\u002e\\u002e\\u005c`,
			"Unicode encoded traversal",
			SeverityMedium,
			0.80,
		},
	}
	
	return pr.compilePatterns(AttackPathTraversal, pathPatterns)
}

// initializeXSSPatterns loads XSS patterns
func (pr *PatternRegistry) initializeXSSPatterns() error {
	xssPatterns := []struct {
		id          string
		pattern     string
		description string
		severity    Severity
		confidence  float64
	}{
		{
			"xss-script-1",
			`(?i)<script[^>]*>.*?</script>`,
			"Script tag injection",
			SeverityCritical,
			0.95,
		},
		{
			"xss-event-1",
			`(?i)\bon(load|error|click|mouse\w+|key\w+)\s*=`,
			"Event handler injection",
			SeverityHigh,
			0.90,
		},
		{
			"xss-javascript-1",
			`(?i)(javascript|vbscript|livescript)\s*:`,
			"JavaScript protocol injection",
			SeverityHigh,
			0.85,
		},
		{
			"xss-img-1",
			`(?i)<img[^>]+src[^>]+onerror\s*=`,
			"Image tag with error handler",
			SeverityHigh,
			0.85,
		},
	}
	
	return pr.compilePatterns(AttackXSS, xssPatterns)
}

// initializePromptPatterns loads AI prompt injection patterns
func (pr *PatternRegistry) initializePromptPatterns() error {
	promptPatterns := []struct {
		id          string
		pattern     string
		description string
		severity    Severity
		confidence  float64
	}{
		{
			"prompt-ignore-1",
			`(?i)(ignore|disregard|forget)\s+(previous|above|prior)\s+(instructions?|commands?|directives?)`,
			"Instruction override attempt",
			SeverityCritical,
			0.95,
		},
		{
			"prompt-system-1",
			`(?i)^\s*system\s*:\s*|\[system\]|<system>`,
			"System prompt injection",
			SeverityHigh,
			0.90,
		},
		{
			"prompt-roleplay-1",
			`(?i)(you are|act as|pretend to be|roleplay as)\s+.*(admin|root|system|developer)`,
			"Role assumption attempt",
			SeverityHigh,
			0.85,
		},
		{
			"prompt-reveal-1",
			`(?i)(show|reveal|display|print)\s+.*(prompt|instructions?|system\s+message)`,
			"Prompt extraction attempt",
			SeverityMedium,
			0.80,
		},
	}
	
	return pr.compilePatterns(AttackPromptInject, promptPatterns)
}

// compilePatterns compiles and registers patterns
func (pr *PatternRegistry) compilePatterns(attackType AttackType, patterns []struct {
	id          string
	pattern     string
	description string
	severity    Severity
	confidence  float64
}) error {
	compiled := make([]*AttackPattern, 0, len(patterns))
	
	for _, p := range patterns {
		re, err := regexp.Compile(p.pattern)
		if err != nil {
			return fmt.Errorf("failed to compile pattern %s: %w", p.id, err)
		}
		
		compiled = append(compiled, &AttackPattern{
			ID:          p.id,
			Type:        attackType,
			Severity:    p.severity,
			Pattern:     re,
			Description: p.description,
			Confidence:  p.confidence,
		})
	}
	
	pr.mu.Lock()
	pr.patterns[attackType] = compiled
	pr.mu.Unlock()
	
	return nil
}

// GetPatterns returns patterns for a specific attack type
func (pr *PatternRegistry) GetPatterns(attackType AttackType) []*AttackPattern {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.patterns[attackType]
}

// GetAllPatterns returns all registered patterns
func (pr *PatternRegistry) GetAllPatterns() []*AttackPattern {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	
	all := make([]*AttackPattern, 0)
	for _, patterns := range pr.patterns {
		all = append(all, patterns...)
	}
	return all
}

// MatchAll checks data against all patterns
func (pr *PatternRegistry) MatchAll(data []byte) []Finding {
	findings := make([]Finding, 0)
	
	for attackType, patterns := range pr.patterns {
		for _, pattern := range patterns {
			if pattern.Pattern.Match(data) {
				findings = append(findings, Finding{
					Type:       string(attackType),
					Severity:   string(pattern.Severity),
					Confidence: pattern.Confidence,
					Details: map[string]interface{}{
						"pattern_id":  pattern.ID,
						"description": pattern.Description,
					},
				})
			}
		}
	}
	
	return findings
}