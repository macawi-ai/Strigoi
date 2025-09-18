package securityaudit

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// MemoryScanner checks for memory safety issues
type MemoryScanner struct {
	patterns map[string]*regexp.Regexp
}

// NewMemoryScanner creates a new memory scanner
func NewMemoryScanner() *MemoryScanner {
	return &MemoryScanner{
		patterns: map[string]*regexp.Regexp{
			"buffer_overflow": regexp.MustCompile(`\[\d+\].*\[[^\]]+\]`), // Array access with dynamic index
			"null_deref":      regexp.MustCompile(`\*\w+\s*=|\w+\.\w+`),  // Potential nil dereference
			"use_after_free":  regexp.MustCompile(`defer.*Close.*\n.*Read|Write`),
			"memory_leak":     regexp.MustCompile(`make\s*\(.*,\s*\d{6,}`), // Large allocations
		},
	}
}

func (s *MemoryScanner) Name() string {
	return "Memory Safety Scanner"
}

func (s *MemoryScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	// Static analysis for memory issues
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

		issues = append(issues, s.scanMemoryIssues(string(content), file)...)
		return nil
	})

	if err != nil {
		return issues, err
	}

	// Dynamic analysis if tests exist
	if config.EnableRuntimeScan {
		issues = append(issues, s.runDynamicAnalysis(path)...)
	}

	return issues, nil
}

func (s *MemoryScanner) scanMemoryIssues(content, filename string) []SecurityIssue {
	var issues []SecurityIssue
	lines := strings.Split(content, "\n")

	// Track defer statements
	var deferredCalls []string

	for i, line := range lines {
		// Check for defer statements
		if strings.Contains(line, "defer") {
			deferredCalls = append(deferredCalls, line)
		}

		// Check for potential buffer overflows
		if s.patterns["buffer_overflow"].MatchString(line) &&
			!strings.Contains(line, "len(") &&
			!strings.Contains(line, "cap(") {
			issues = append(issues, SecurityIssue{
				Type:        "BUFFER_OVERFLOW",
				Severity:    "HIGH",
				Title:       "Potential buffer overflow",
				Description: "Array access without bounds checking",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-119",
				Remediation: "Add bounds checking before array access",
			})
		}

		// Check for large memory allocations
		if s.patterns["memory_leak"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "MEMORY_LEAK",
				Severity:    "MEDIUM",
				Title:       "Large memory allocation detected",
				Description: "Large allocation may lead to memory exhaustion",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-400",
				Remediation: "Consider using streaming or chunked processing",
			})
		}

		// Check for nil pointer dereferences
		if strings.Contains(line, "if") && strings.Contains(line, "!= nil") {
			// Good - checking for nil (positive security pattern)
			// nolint:revive // intentional empty block for positive pattern recognition
		} else if s.patterns["null_deref"].MatchString(line) &&
			!strings.Contains(line, ":=") &&
			!strings.Contains(line, "=") {
			// Potential nil dereference without check
			varName := extractVarName(line)
			if varName != "" && !isCheckedForNil(lines, i, varName) {
				issues = append(issues, SecurityIssue{
					Type:        "NULL_DEREFERENCE",
					Severity:    "HIGH",
					Title:       "Potential nil pointer dereference",
					Description: fmt.Sprintf("Variable '%s' may be nil", varName),
					Location: IssueLocation{
						File: filename,
						Line: i + 1,
					},
					CWE:         "CWE-476",
					Remediation: "Add nil check before dereferencing",
				})
			}
		}
	}

	return issues
}

func (s *MemoryScanner) runDynamicAnalysis(path string) []SecurityIssue {
	var issues []SecurityIssue

	// Run tests with race detector
	cmd := exec.Command("go", "test", "-race", "./...")
	cmd.Dir = path
	output, err := cmd.CombinedOutput()

	if err != nil && strings.Contains(string(output), "DATA RACE") {
		// Parse race detector output
		races := parseRaceOutput(string(output))
		for _, race := range races {
			issues = append(issues, SecurityIssue{
				Type:        "RACE_CONDITION",
				Severity:    "HIGH",
				Title:       "Data race detected",
				Description: race.Description,
				Location: IssueLocation{
					File:     race.File,
					Line:     race.Line,
					Function: race.Function,
				},
				CWE: "CWE-362",
				Evidence: map[string]string{
					"stack_trace": race.StackTrace,
				},
				Remediation: "Use synchronization primitives (mutex, channels)",
			})
		}
	}

	// Check for memory leaks using pprof
	issues = append(issues, s.checkMemoryLeaks(path)...)

	return issues
}

func (s *MemoryScanner) checkMemoryLeaks(path string) []SecurityIssue {
	var issues []SecurityIssue

	// Create a test that profiles memory
	testFile := filepath.Join(path, "memory_audit_test.go")
	testContent := `package main

import (
	"runtime"
	"testing"
	"time"
)

func TestMemoryAudit(t *testing.T) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// Run some operations
	time.Sleep(100 * time.Millisecond)
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	leaked := m2.HeapAlloc - m1.HeapAlloc
	if leaked > 10*1024*1024 { // 10MB threshold
		t.Logf("MEMORY_LEAK: %d bytes", leaked)
	}
}`

	if err := ioutil.WriteFile(testFile, []byte(testContent), 0600); err == nil {
		defer os.Remove(testFile)

		cmd := exec.Command("go", "test", "-v", testFile)
		cmd.Dir = path
		output, _ := cmd.CombinedOutput()

		if strings.Contains(string(output), "MEMORY_LEAK") {
			issues = append(issues, SecurityIssue{
				Type:        "MEMORY_LEAK",
				Severity:    "MEDIUM",
				Title:       "Memory leak detected during testing",
				Description: "Significant memory allocation not freed",
				CWE:         "CWE-401",
				Remediation: "Profile memory usage and ensure proper cleanup",
			})
		}
	}

	return issues
}

// RaceScanner specifically checks for race conditions
type RaceScanner struct {
	concurrencyPatterns []*regexp.Regexp
}

// NewRaceScanner creates a new race condition scanner
func NewRaceScanner() *RaceScanner {
	return &RaceScanner{
		concurrencyPatterns: []*regexp.Regexp{
			regexp.MustCompile(`go\s+func`),                // Goroutines
			regexp.MustCompile(`\bmap\[.*\].*{`),           // Shared maps
			regexp.MustCompile(`\+\+|\-\-`),                // Increment/decrement
			regexp.MustCompile(`\w+\s*=\s*\w+\s*\+\s*\d+`), // Non-atomic updates
		},
	}
}

func (s *RaceScanner) Name() string {
	return "Race Condition Scanner"
}

func (s *RaceScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	// Find all Go files
	var goFiles []string
	err := filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if strings.HasSuffix(file, ".go") && !IsTestFile(file) {
			goFiles = append(goFiles, file)
		}
		return nil
	})

	if err != nil {
		return issues, err
	}

	// Analyze each file
	for _, file := range goFiles {
		fileIssues, err := s.analyzeFile(file)
		if err != nil {
			continue
		}
		issues = append(issues, fileIssues...)
	}

	// Run race detector on tests
	if config.EnableRuntimeScan {
		cmd := exec.Command("go", "test", "-race", "-short", "./...")
		cmd.Dir = path
		output, err := cmd.CombinedOutput()

		if err != nil && bytes.Contains(output, []byte("WARNING: DATA RACE")) {
			issues = append(issues, SecurityIssue{
				Type:        "RACE_CONDITION",
				Severity:    "CRITICAL",
				Title:       "Data race detected by race detector",
				Description: "The Go race detector found actual data races",
				Evidence: map[string]string{
					"output": string(output),
				},
				CWE:         "CWE-362",
				Remediation: "Fix all data races reported by 'go test -race'",
			})
		}
	}

	return issues, nil
}

func (s *RaceScanner) analyzeFile(filename string) ([]SecurityIssue, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var issues []SecurityIssue
	lines := strings.Split(string(content), "\n")

	// Track shared variables
	sharedVars := make(map[string]bool)
	inGoroutine := false

	for i, line := range lines {
		// Check if we're in a goroutine
		if strings.Contains(line, "go func") {
			inGoroutine = true
		} else if inGoroutine && strings.Contains(line, "}()") {
			inGoroutine = false
		}

		// Look for shared map access without synchronization
		if strings.Contains(line, "map[") && !strings.Contains(line, "sync.Map") {
			if inGoroutine || hasGoroutineNearby(lines, i) {
				issues = append(issues, SecurityIssue{
					Type:        "UNSYNC_MAP_ACCESS",
					Severity:    "HIGH",
					Title:       "Concurrent map access without synchronization",
					Description: "Maps are not safe for concurrent access",
					Location: IssueLocation{
						File: filename,
						Line: i + 1,
					},
					CWE:         "CWE-362",
					Remediation: "Use sync.Map or protect with mutex",
				})
			}
		}

		// Check for non-atomic counter updates
		if s.concurrencyPatterns[2].MatchString(line) || s.concurrencyPatterns[3].MatchString(line) {
			varName := extractVarName(line)
			if varName != "" {
				if inGoroutine || sharedVars[varName] {
					issues = append(issues, SecurityIssue{
						Type:        "NON_ATOMIC_UPDATE",
						Severity:    "MEDIUM",
						Title:       "Non-atomic variable update",
						Description: fmt.Sprintf("Variable '%s' updated without atomic operations", varName),
						Location: IssueLocation{
							File: filename,
							Line: i + 1,
						},
						CWE:         "CWE-366",
						Remediation: "Use atomic operations from sync/atomic package",
					})
				}
				sharedVars[varName] = true
			}
		}
	}

	return issues, nil
}

// NetworkScanner checks for network security issues
type NetworkScanner struct {
	insecurePatterns map[string]*regexp.Regexp
}

// NewNetworkScanner creates a new network scanner
func NewNetworkScanner() *NetworkScanner {
	return &NetworkScanner{
		insecurePatterns: map[string]*regexp.Regexp{
			"http_server":     regexp.MustCompile(`http\.ListenAndServe\(`),
			"no_timeout":      regexp.MustCompile(`net\.Dial\(`),
			"bind_all":        regexp.MustCompile(`::\d+|0\.0\.0\.0:\d+`),
			"high_port":       regexp.MustCompile(`:\d{5}`),
			"insecure_cookie": regexp.MustCompile(`Secure:\s*false`),
		},
	}
}

func (s *NetworkScanner) Name() string {
	return "Network Security Scanner"
}

func (s *NetworkScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
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

		issues = append(issues, s.scanNetworkIssues(string(content), file)...)
		return nil
	})

	// Check for exposed ports
	if config.EnableNetworkScan {
		issues = append(issues, s.checkExposedPorts()...)
	}

	return issues, err
}

func (s *NetworkScanner) scanNetworkIssues(content, filename string) []SecurityIssue {
	var issues []SecurityIssue
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// Check for HTTP server without TLS
		if s.insecurePatterns["http_server"].MatchString(line) &&
			!strings.Contains(content, "ListenAndServeTLS") {
			issues = append(issues, SecurityIssue{
				Type:        "UNENCRYPTED_COMMUNICATION",
				Severity:    "HIGH",
				Title:       "HTTP server without TLS",
				Description: "Server accepts unencrypted connections",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-319",
				OWASP:       "A02:2021",
				Remediation: "Use ListenAndServeTLS with valid certificates",
			})
		}

		// Check for missing timeouts
		if s.insecurePatterns["no_timeout"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "MISSING_TIMEOUT",
				Severity:    "MEDIUM",
				Title:       "Network operation without timeout",
				Description: "Missing timeout can lead to resource exhaustion",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-400",
				Remediation: "Use DialTimeout or set deadlines on connections",
			})
		}

		// Check for binding to all interfaces
		if s.insecurePatterns["bind_all"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "BIND_ALL_INTERFACES",
				Severity:    "MEDIUM",
				Title:       "Service binds to all network interfaces",
				Description: "Binding to 0.0.0.0 or :: exposes service to all networks",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				Evidence: map[string]string{
					"code": strings.TrimSpace(line),
				},
				Remediation: "Bind to specific interfaces or use firewall rules",
			})
		}

		// Check for insecure cookies
		if s.insecurePatterns["insecure_cookie"].MatchString(line) {
			issues = append(issues, SecurityIssue{
				Type:        "INSECURE_COOKIE",
				Severity:    "HIGH",
				Title:       "Cookie without Secure flag",
				Description: "Cookies may be transmitted over unencrypted connections",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-614",
				OWASP:       "A05:2021",
				Remediation: "Set Secure: true for all cookies",
			})
		}
	}

	return issues
}

func (s *NetworkScanner) checkExposedPorts() []SecurityIssue {
	var issues []SecurityIssue

	// Use netstat or ss to check listening ports
	cmd := exec.Command("ss", "-tlnp")
	output, err := cmd.Output()
	if err != nil {
		// Try netstat as fallback
		cmd = exec.Command("netstat", "-tlnp")
		output, err = cmd.Output()
		if err != nil {
			return issues
		}
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Look for high ports or unusual services
		if strings.Contains(line, "LISTEN") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				addr := parts[3]
				if strings.Contains(addr, "0.0.0.0:") || strings.Contains(addr, ":::") {
					port := extractPort(addr)
					if port > 1024 && port < 10000 {
						issues = append(issues, SecurityIssue{
							Type:        "EXPOSED_PORT",
							Severity:    "LOW",
							Title:       fmt.Sprintf("Service exposed on port %d", port),
							Description: "Non-standard port exposed to all interfaces",
							Evidence: map[string]string{
								"address": addr,
							},
							Remediation: "Review if this port needs to be publicly accessible",
						})
					}
				}
			}
		}
	}

	return issues
}

// TLSScanner checks TLS/SSL configuration
type TLSScanner struct{}

// NewTLSScanner creates a new TLS scanner
func NewTLSScanner() *TLSScanner {
	return &TLSScanner{}
}

func (s *TLSScanner) Name() string {
	return "TLS Configuration Scanner"
}

func (s *TLSScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
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

		issues = append(issues, s.scanTLSConfig(string(content), file)...)
		return nil
	})

	return issues, err
}

func (s *TLSScanner) scanTLSConfig(content, filename string) []SecurityIssue {
	var issues []SecurityIssue
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// Check for InsecureSkipVerify
		if strings.Contains(line, "InsecureSkipVerify") && strings.Contains(line, "true") {
			issues = append(issues, SecurityIssue{
				Type:        "TLS_VERIFICATION_DISABLED",
				Severity:    "CRITICAL",
				Title:       "TLS certificate verification disabled",
				Description: "InsecureSkipVerify bypasses certificate validation",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-295",
				OWASP:       "A07:2021",
				Remediation: "Enable certificate verification and use proper CA certificates",
			})
		}

		// Check for weak TLS versions
		if strings.Contains(line, "tls.VersionSSL30") ||
			strings.Contains(line, "tls.VersionTLS10") ||
			strings.Contains(line, "tls.VersionTLS11") {
			issues = append(issues, SecurityIssue{
				Type:        "WEAK_TLS_VERSION",
				Severity:    "HIGH",
				Title:       "Weak TLS version allowed",
				Description: "Old TLS versions have known vulnerabilities",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-326",
				Remediation: "Use TLS 1.2 or higher (tls.VersionTLS12)",
			})
		}

		// Check for weak cipher suites
		if strings.Contains(line, "CipherSuites") &&
			(strings.Contains(line, "TLS_RSA_") || strings.Contains(line, "TLS_ECDHE_RSA_WITH_3DES")) {
			issues = append(issues, SecurityIssue{
				Type:        "WEAK_CIPHER_SUITE",
				Severity:    "MEDIUM",
				Title:       "Weak cipher suite configured",
				Description: "Weak or deprecated cipher suites in use",
				Location: IssueLocation{
					File: filename,
					Line: i + 1,
				},
				CWE:         "CWE-326",
				Remediation: "Use modern cipher suites with forward secrecy",
			})
		}
	}

	return issues
}

// Helper functions

func extractVarName(line string) string {
	// Simple variable name extraction
	parts := strings.Fields(line)
	for _, part := range parts {
		if strings.Contains(part, "++") || strings.Contains(part, "--") {
			return strings.TrimSuffix(strings.TrimSuffix(part, "++"), "--")
		}
		if strings.Contains(line, "=") {
			if idx := strings.Index(line, "="); idx > 0 {
				return strings.TrimSpace(line[:idx])
			}
		}
	}
	return ""
}

func isCheckedForNil(lines []string, currentLine int, varName string) bool {
	// Look backwards for nil check
	for i := currentLine - 1; i >= 0 && i > currentLine-10; i-- {
		if strings.Contains(lines[i], varName) &&
			strings.Contains(lines[i], "!= nil") {
			return true
		}
	}
	return false
}

func hasGoroutineNearby(lines []string, currentLine int) bool {
	// Check if there's a goroutine within 5 lines
	for i := currentLine - 5; i <= currentLine+5 && i < len(lines); i++ {
		if i >= 0 && strings.Contains(lines[i], "go ") {
			return true
		}
	}
	return false
}

func extractPort(addr string) int {
	parts := strings.Split(addr, ":")
	if len(parts) >= 2 {
		port, _ := strconv.Atoi(parts[len(parts)-1])
		return port
	}
	return 0
}

// RaceInfo holds information about a detected race
type RaceInfo struct {
	Description string
	File        string
	Line        int
	Function    string
	StackTrace  string
}

func parseRaceOutput(output string) []RaceInfo {
	var races []RaceInfo

	// Simple parsing - in production, use more sophisticated parsing
	sections := strings.Split(output, "WARNING: DATA RACE")
	for _, section := range sections[1:] {
		race := RaceInfo{
			Description: "Data race detected",
			StackTrace:  section,
		}

		// Extract file and line
		lines := strings.Split(section, "\n")
		for _, line := range lines {
			if strings.Contains(line, ".go:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					race.File = parts[0]
					lineNum, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
					race.Line = lineNum
					break
				}
			}
		}

		races = append(races, race)
	}

	return races
}
