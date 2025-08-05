package security_audit

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CodeScanner performs static code analysis
type CodeScanner struct {
	patterns map[string]*regexp.Regexp
}

// NewCodeScanner creates a new code scanner
func NewCodeScanner() *CodeScanner {
	return &CodeScanner{
		patterns: map[string]*regexp.Regexp{
			"unsafe_ptr":      regexp.MustCompile(`unsafe\.Pointer`),
			"race_risk":       regexp.MustCompile(`go\s+func.*\{[^}]*\b(shared|global)\b`),
			"panic_unhandled": regexp.MustCompile(`panic\(`),
			"todos":           regexp.MustCompile(`(?i)(todo|fixme|hack|xxx)`),
			"hardcoded_vals":  regexp.MustCompile(`(password|key|secret|token)\s*=\s*"[^"]+"'`),
		},
	}
}

func (s *CodeScanner) Name() string {
	return "Code Security Scanner"
}

func (s *CodeScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	err := filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !strings.HasSuffix(file, ".go") || IsTestFile(file) {
			return nil
		}

		fileIssues, err := s.scanFile(file)
		if err != nil {
			return nil
		}

		issues = append(issues, fileIssues...)
		return nil
	})

	return issues, err
}

func (s *CodeScanner) scanFile(filename string) ([]SecurityIssue, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var issues []SecurityIssue

	// Parse Go AST
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Check imports
	issues = append(issues, s.checkImports(node, filename)...)

	// Scan for patterns
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check unsafe pointer usage
		if s.patterns["unsafe_ptr"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "UNSAFE_CODE",
				Severity:    "HIGH",
				Title:       "Unsafe pointer usage detected",
				Description: "Use of unsafe.Pointer can lead to memory corruption",
				Location: IssueLocation{
					File: filename,
					Line: lineNum,
				},
				CWE:         "CWE-119",
				Remediation: "Consider safer alternatives or thoroughly review the unsafe code",
			})
		}

		// Check for potential race conditions
		if s.patterns["race_risk"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "RACE_CONDITION",
				Severity:    "MEDIUM",
				Title:       "Potential race condition",
				Description: "Concurrent access to shared data without synchronization",
				Location: IssueLocation{
					File: filename,
					Line: lineNum,
				},
				CWE:         "CWE-362",
				Remediation: "Use mutexes or channels for synchronization",
			})
		}

		// Check unhandled panics
		if s.patterns["panic_unhandled"].MatchString(line) && !strings.Contains(line, "recover") {
			issues = append(issues, SecurityIssue{
				Type:        "ERROR_HANDLING",
				Severity:    "MEDIUM",
				Title:       "Unhandled panic",
				Description: "Panic without corresponding recover can crash the application",
				Location: IssueLocation{
					File: filename,
					Line: lineNum,
				},
				CWE:         "CWE-248",
				Remediation: "Add recover() in defer or handle errors gracefully",
			})
		}

		// Check for TODOs (lower priority)
		if s.patterns["todos"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "CODE_QUALITY",
				Severity:    "LOW",
				Title:       "Unfinished code detected",
				Description: fmt.Sprintf("TODO/FIXME comment found: %s", line),
				Location: IssueLocation{
					File: filename,
					Line: lineNum,
				},
				Remediation: "Complete the implementation or remove if obsolete",
			})
		}
	}

	return issues, nil
}

func (s *CodeScanner) checkImports(node *ast.File, filename string) []SecurityIssue {
	var issues []SecurityIssue

	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, `"`)

		// Check for deprecated or risky packages
		switch path {
		case "crypto/md5", "crypto/sha1":
			issues = append(issues, SecurityIssue{
				Type:        "WEAK_CRYPTO",
				Severity:    "HIGH",
				Title:       "Weak cryptographic algorithm",
				Description: fmt.Sprintf("Use of weak crypto: %s", path),
				Location: IssueLocation{
					File: filename,
				},
				CWE:         "CWE-327",
				OWASP:       "A02:2021",
				Remediation: "Use crypto/sha256 or stronger algorithms",
			})
		case "net/http/cgi":
			issues = append(issues, SecurityIssue{
				Type:        "DEPRECATED_PACKAGE",
				Severity:    "MEDIUM",
				Title:       "Use of deprecated package",
				Description: "CGI package is deprecated and has security issues",
				Location: IssueLocation{
					File: filename,
				},
				Remediation: "Use modern web frameworks instead",
			})
		}
	}

	return issues
}

// InjectionScanner checks for injection vulnerabilities
type InjectionScanner struct {
	sqlPatterns  []*regexp.Regexp
	cmdPatterns  []*regexp.Regexp
	pathPatterns []*regexp.Regexp
}

// NewInjectionScanner creates a new injection scanner
func NewInjectionScanner() *InjectionScanner {
	return &InjectionScanner{
		sqlPatterns: []*regexp.Regexp{
			regexp.MustCompile(`fmt\.Sprintf\s*\(\s*"[^"]*\b(SELECT|INSERT|UPDATE|DELETE|DROP)\b`),
			regexp.MustCompile(`\+\s*"[^"]*\b(SELECT|INSERT|UPDATE|DELETE|DROP)\b`),
			regexp.MustCompile(`Query\s*\(\s*"[^"]*\s*\+`),
		},
		cmdPatterns: []*regexp.Regexp{
			regexp.MustCompile(`exec\.Command\s*\([^,]+,\s*[^)]*\+`),
			regexp.MustCompile(`os\.Exec\s*\(`),
			regexp.MustCompile(`syscall\.Exec\s*\(`),
		},
		pathPatterns: []*regexp.Regexp{
			regexp.MustCompile(`filepath\.Join\s*\([^)]*\.\./`),
			regexp.MustCompile(`os\.Open\s*\([^)]*\+`),
			regexp.MustCompile(`ioutil\.ReadFile\s*\([^)]*\+`),
		},
	}
}

func (s *InjectionScanner) Name() string {
	return "Injection Vulnerability Scanner"
}

func (s *InjectionScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	err := filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !strings.HasSuffix(file, ".go") || IsTestFile(file) {
			return nil
		}

		content, err := ioutil.ReadFile(file)
		if err != nil {
			return nil
		}

		issues = append(issues, s.scanContent(string(content), file)...)
		return nil
	})

	return issues, err
}

func (s *InjectionScanner) scanContent(content, filename string) []SecurityIssue {
	var issues []SecurityIssue
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// SQL Injection
		for _, pattern := range s.sqlPatterns {
			if pattern.MatchString(line) {
				issues = append(issues, SecurityIssue{
					Type:        "SQL_INJECTION",
					Severity:    "CRITICAL",
					Title:       "Potential SQL injection vulnerability",
					Description: "Dynamic SQL query construction detected",
					Location: IssueLocation{
						File: filename,
						Line: i + 1,
					},
					Evidence: map[string]string{
						"code": strings.TrimSpace(line),
					},
					CWE:         "CWE-89",
					OWASP:       "A03:2021",
					Remediation: "Use parameterized queries or prepared statements",
				})
			}
		}

		// Command Injection
		for _, pattern := range s.cmdPatterns {
			if pattern.MatchString(line) {
				issues = append(issues, SecurityIssue{
					Type:        "COMMAND_INJECTION",
					Severity:    "CRITICAL",
					Title:       "Potential command injection vulnerability",
					Description: "Dynamic command execution detected",
					Location: IssueLocation{
						File: filename,
						Line: i + 1,
					},
					Evidence: map[string]string{
						"code": strings.TrimSpace(line),
					},
					CWE:         "CWE-78",
					OWASP:       "A03:2021",
					Remediation: "Validate and sanitize all input, use allowlists",
				})
			}
		}

		// Path Traversal
		for _, pattern := range s.pathPatterns {
			if pattern.MatchString(line) {
				issues = append(issues, SecurityIssue{
					Type:        "PATH_TRAVERSAL",
					Severity:    "HIGH",
					Title:       "Potential path traversal vulnerability",
					Description: "Dynamic file path construction detected",
					Location: IssueLocation{
						File: filename,
						Line: i + 1,
					},
					Evidence: map[string]string{
						"code": strings.TrimSpace(line),
					},
					CWE:         "CWE-22",
					OWASP:       "A01:2021",
					Remediation: "Sanitize file paths and use filepath.Clean()",
				})
			}
		}
	}

	return issues
}

// CryptoScanner checks for cryptographic issues
type CryptoScanner struct {
	weakAlgos    map[string]string
	badPractices []*regexp.Regexp
}

// NewCryptoScanner creates a new crypto scanner
func NewCryptoScanner() *CryptoScanner {
	return &CryptoScanner{
		weakAlgos: map[string]string{
			"md5":  "MD5 is cryptographically broken",
			"sha1": "SHA1 is deprecated for security use",
			"des":  "DES has inadequate key length",
			"rc4":  "RC4 has known vulnerabilities",
		},
		badPractices: []*regexp.Regexp{
			regexp.MustCompile(`rand\.Read\s*\(`), // Should use crypto/rand
			regexp.MustCompile(`math/rand`),       // For crypto purposes
			regexp.MustCompile(`InsecureSkipVerify\s*:\s*true`),
			regexp.MustCompile(`tls\.Config\s*{[^}]*MinVersion\s*:\s*tls\.VersionSSL30`),
		},
	}
}

func (s *CryptoScanner) Name() string {
	return "Cryptography Scanner"
}

func (s *CryptoScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	err := filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !strings.HasSuffix(file, ".go") || IsTestFile(file) {
			return nil
		}

		content, err := ioutil.ReadFile(file)
		if err != nil {
			return nil
		}

		issues = append(issues, s.scanCrypto(string(content), file)...)
		return nil
	})

	return issues, err
}

func (s *CryptoScanner) scanCrypto(content, filename string) []SecurityIssue {
	var issues []SecurityIssue
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineLower := strings.ToLower(line)

		// Check weak algorithms
		for algo, reason := range s.weakAlgos {
			if strings.Contains(lineLower, algo) &&
				(strings.Contains(line, "New") || strings.Contains(line, "Sum")) {
				issues = append(issues, SecurityIssue{
					Type:        "WEAK_CRYPTO",
					Severity:    "HIGH",
					Title:       fmt.Sprintf("Use of weak algorithm: %s", strings.ToUpper(algo)),
					Description: reason,
					Location: IssueLocation{
						File: filename,
						Line: i + 1,
					},
					CWE:         "CWE-327",
					OWASP:       "A02:2021",
					Remediation: "Use SHA256 or stronger algorithms",
				})
			}
		}

		// Check bad practices
		for _, pattern := range s.badPractices {
			if pattern.MatchString(line) {
				severity := "MEDIUM"
				title := "Insecure cryptographic practice"

				if strings.Contains(line, "InsecureSkipVerify") {
					severity = "HIGH"
					title = "TLS certificate verification disabled"
				}

				issues = append(issues, SecurityIssue{
					Type:        "CRYPTO_MISUSE",
					Severity:    severity,
					Title:       title,
					Description: "Insecure cryptographic configuration detected",
					Location: IssueLocation{
						File: filename,
						Line: i + 1,
					},
					Evidence: map[string]string{
						"code": strings.TrimSpace(line),
					},
					CWE:         "CWE-295",
					Remediation: "Enable proper certificate verification",
				})
			}
		}
	}

	return issues
}

// AuthScanner checks for authentication and authorization issues
type AuthScanner struct {
	patterns map[string]*regexp.Regexp
}

// NewAuthScanner creates a new authentication scanner
func NewAuthScanner() *AuthScanner {
	return &AuthScanner{
		patterns: map[string]*regexp.Regexp{
			"basic_auth": regexp.MustCompile(`Basic\s+[A-Za-z0-9+/=]+`),
			"jwt_none":   regexp.MustCompile(`alg.*:.*none`),
			"no_auth":    regexp.MustCompile(`// TODO.*auth|FIXME.*auth|XXX.*auth`),
			"hardcoded":  regexp.MustCompile(`(password|secret|key)\s*[:=]\s*["'][^"']+["']`),
		},
	}
}

func (s *AuthScanner) Name() string {
	return "Authentication Scanner"
}

func (s *AuthScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	err := filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !strings.HasSuffix(file, ".go") || IsTestFile(file) {
			return nil
		}

		content, err := ioutil.ReadFile(file)
		if err != nil {
			return nil
		}

		issues = append(issues, s.scanAuth(string(content), file)...)
		return nil
	})

	return issues, err
}

func (s *AuthScanner) scanAuth(content, filename string) []SecurityIssue {
	var issues []SecurityIssue
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// Check for hardcoded credentials
		if s.patterns["hardcoded"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "HARDCODED_CREDENTIALS",
				Severity:    "CRITICAL",
				Title:       "Hardcoded credentials detected",
				Description: "Credentials should never be hardcoded in source",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-798",
				OWASP:       "A07:2021",
				Remediation: "Use environment variables or secure vaults",
			})
		}

		// Check for missing auth
		if s.patterns["no_auth"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "MISSING_AUTH",
				Severity:    "HIGH",
				Title:       "Missing authentication implementation",
				Description: "TODO comment indicates missing auth",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-306",
				Remediation: "Implement proper authentication",
			})
		}

		// Check JWT none algorithm
		if s.patterns["jwt_none"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "WEAK_AUTH",
				Severity:    "CRITICAL",
				Title:       "JWT 'none' algorithm detected",
				Description: "JWT with 'none' algorithm provides no security",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-347",
				Remediation: "Use proper JWT signing algorithms (RS256, HS256)",
			})
		}
	}

	return issues
}
