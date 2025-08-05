package security_audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// AuditFramework performs security audits on the Strigoi platform
type AuditFramework struct {
	config    AuditConfig
	scanners  []SecurityScanner
	reporters []AuditReporter
	results   *AuditResults
	startTime time.Time
}

// AuditConfig configures the security audit
type AuditConfig struct {
	// Paths to scan
	SourcePaths  []string
	ExcludePaths []string

	// Scan types
	EnableCodeScan    bool
	EnableDepsScan    bool
	EnableConfigScan  bool
	EnableRuntimeScan bool
	EnableNetworkScan bool

	// Risk thresholds
	MaxCriticalIssues int
	MaxHighIssues     int

	// Output
	OutputFormat string // json, html, markdown
	OutputPath   string
}

// SecurityScanner interface for different security scanners
type SecurityScanner interface {
	Name() string
	Scan(path string, config AuditConfig) ([]SecurityIssue, error)
}

// AuditReporter interface for generating reports
type AuditReporter interface {
	GenerateReport(results *AuditResults, output io.Writer) error
}

// SecurityIssue represents a security finding
type SecurityIssue struct {
	ID            string
	Type          string
	Severity      string // CRITICAL, HIGH, MEDIUM, LOW, INFO
	Title         string
	Description   string
	Location      IssueLocation
	Evidence      map[string]string
	Remediation   string
	References    []string
	CWE           string
	OWASP         string
	Verified      bool
	FalsePositive bool
}

// IssueLocation describes where the issue was found
type IssueLocation struct {
	File      string
	Line      int
	Column    int
	Function  string
	Package   string
	Component string
}

// AuditResults contains all audit findings
type AuditResults struct {
	StartTime time.Time
	EndTime   time.Time
	Platform  PlatformInfo
	Summary   AuditSummary
	Issues    []SecurityIssue
	Metrics   AuditMetrics
}

// PlatformInfo contains information about the audited platform
type PlatformInfo struct {
	Name       string
	Version    string
	GitCommit  string
	BuildTime  string
	GoVersion  string
	Platform   string
	Components []ComponentInfo
}

// ComponentInfo describes a platform component
type ComponentInfo struct {
	Name     string
	Version  string
	Path     string
	Type     string // module, service, library
	Critical bool
}

// AuditSummary provides high-level statistics
type AuditSummary struct {
	TotalIssues       int
	CriticalIssues    int
	HighIssues        int
	MediumIssues      int
	LowIssues         int
	InfoIssues        int
	FilesScanned      int
	LinesScanned      int
	ComponentsScanned int
	TimeElapsed       time.Duration
}

// AuditMetrics provides detailed metrics
type AuditMetrics struct {
	CodeCoverage    float64
	SecurityScore   float64
	ComplianceScore float64
	RiskScore       float64
	TechnicalDebt   time.Duration
}

// NewAuditFramework creates a new audit framework
func NewAuditFramework(config AuditConfig) *AuditFramework {
	framework := &AuditFramework{
		config:    config,
		scanners:  []SecurityScanner{},
		reporters: []AuditReporter{},
		results: &AuditResults{
			Issues: []SecurityIssue{},
		},
	}

	// Add default scanners
	if config.EnableCodeScan {
		framework.scanners = append(framework.scanners,
			NewCodeScanner(),
			NewInjectionScanner(),
			NewCryptoScanner(),
			NewAuthScanner(),
		)
	}

	if config.EnableDepsScan {
		framework.scanners = append(framework.scanners,
			NewDependencyScanner(),
			NewLicenseScanner(),
		)
	}

	if config.EnableConfigScan {
		framework.scanners = append(framework.scanners,
			NewConfigScanner(),
			NewSecretsScanner(),
		)
	}

	if config.EnableRuntimeScan {
		framework.scanners = append(framework.scanners,
			NewMemoryScanner(),
			NewRaceScanner(),
		)
	}

	if config.EnableNetworkScan {
		framework.scanners = append(framework.scanners,
			NewNetworkScanner(),
			NewTLSScanner(),
		)
	}

	// Add reporters
	switch config.OutputFormat {
	case "json":
		framework.reporters = append(framework.reporters, NewJSONReporter())
	case "html":
		framework.reporters = append(framework.reporters, NewHTMLReporter())
	case "markdown":
		framework.reporters = append(framework.reporters, NewMarkdownReporter())
	default:
		framework.reporters = append(framework.reporters, NewMarkdownReporter())
	}

	return framework
}

// AddScanner adds a custom scanner to the framework
func (f *AuditFramework) AddScanner(scanner SecurityScanner) {
	f.scanners = append(f.scanners, scanner)
}

// AddReporter adds a custom reporter to the framework
func (f *AuditFramework) AddReporter(reporter AuditReporter) {
	f.reporters = append(f.reporters, reporter)
}

// RunAudit performs the security audit
func (f *AuditFramework) RunAudit() (*AuditResults, error) {
	f.startTime = time.Now()
	f.results.StartTime = f.startTime

	// Collect platform information
	if err := f.collectPlatformInfo(); err != nil {
		return nil, fmt.Errorf("failed to collect platform info: %w", err)
	}

	// Run all scanners
	for _, scanner := range f.scanners {
		fmt.Printf("Running %s scanner...\n", scanner.Name())

		for _, path := range f.config.SourcePaths {
			if f.shouldExcludePath(path) {
				continue
			}

			issues, err := scanner.Scan(path, f.config)
			if err != nil {
				fmt.Printf("Warning: %s scanner failed for %s: %v\n",
					scanner.Name(), path, err)
				continue
			}

			f.results.Issues = append(f.results.Issues, issues...)
		}
	}

	// Calculate summary and metrics
	f.calculateSummary()
	f.calculateMetrics()

	// Check thresholds
	if err := f.checkThresholds(); err != nil {
		return f.results, err
	}

	f.results.EndTime = time.Now()

	// Generate reports
	if err := f.generateReports(); err != nil {
		return f.results, fmt.Errorf("failed to generate reports: %w", err)
	}

	return f.results, nil
}

// collectPlatformInfo gathers information about the platform
func (f *AuditFramework) collectPlatformInfo() error {
	f.results.Platform = PlatformInfo{
		Name:      "Strigoi Security Platform",
		Version:   "1.0.0", // TODO: Read from version file
		Platform:  fmt.Sprintf("%s/%s", os.Getenv("GOOS"), os.Getenv("GOARCH")),
		GoVersion: os.Getenv("GOVERSION"),
	}

	// Identify components
	components := []ComponentInfo{
		{Name: "Core Engine", Type: "module", Path: ".", Critical: true},
		{Name: "Circular Buffer", Type: "module", Path: "./circular_buffer", Critical: true},
		{Name: "Protocol Dissectors", Type: "module", Path: "./dissectors", Critical: true},
		{Name: "SIEM Integration", Type: "module", Path: "./siem", Critical: false},
		{Name: "Session Manager", Type: "module", Path: "./session", Critical: true},
	}

	f.results.Platform.Components = components
	return nil
}

// shouldExcludePath checks if a path should be excluded
func (f *AuditFramework) shouldExcludePath(path string) bool {
	for _, exclude := range f.config.ExcludePaths {
		matched, _ := filepath.Match(exclude, path)
		if matched {
			return true
		}
	}
	return false
}

// calculateSummary calculates audit summary statistics
func (f *AuditFramework) calculateSummary() {
	summary := &f.results.Summary

	for _, issue := range f.results.Issues {
		if issue.FalsePositive {
			continue
		}

		summary.TotalIssues++

		switch issue.Severity {
		case "CRITICAL":
			summary.CriticalIssues++
		case "HIGH":
			summary.HighIssues++
		case "MEDIUM":
			summary.MediumIssues++
		case "LOW":
			summary.LowIssues++
		case "INFO":
			summary.InfoIssues++
		}
	}

	summary.TimeElapsed = time.Since(f.startTime)
	summary.ComponentsScanned = len(f.results.Platform.Components)
}

// calculateMetrics calculates security metrics
func (f *AuditFramework) calculateMetrics() {
	metrics := &f.results.Metrics

	// Calculate security score (100 - penalties)
	score := 100.0
	score -= float64(f.results.Summary.CriticalIssues) * 20.0
	score -= float64(f.results.Summary.HighIssues) * 10.0
	score -= float64(f.results.Summary.MediumIssues) * 5.0
	score -= float64(f.results.Summary.LowIssues) * 2.0

	if score < 0 {
		score = 0
	}

	metrics.SecurityScore = score

	// Calculate risk score (inverse of security score)
	metrics.RiskScore = 100.0 - score

	// Estimate technical debt
	hours := f.results.Summary.CriticalIssues * 8
	hours += f.results.Summary.HighIssues * 4
	hours += f.results.Summary.MediumIssues * 2
	hours += f.results.Summary.LowIssues * 1

	metrics.TechnicalDebt = time.Duration(hours) * time.Hour
}

// checkThresholds verifies issues don't exceed configured thresholds
func (f *AuditFramework) checkThresholds() error {
	if f.config.MaxCriticalIssues > 0 &&
		f.results.Summary.CriticalIssues > f.config.MaxCriticalIssues {
		return fmt.Errorf("critical issues (%d) exceed threshold (%d)",
			f.results.Summary.CriticalIssues, f.config.MaxCriticalIssues)
	}

	if f.config.MaxHighIssues > 0 &&
		f.results.Summary.HighIssues > f.config.MaxHighIssues {
		return fmt.Errorf("high severity issues (%d) exceed threshold (%d)",
			f.results.Summary.HighIssues, f.config.MaxHighIssues)
	}

	return nil
}

// generateReports creates audit reports
func (f *AuditFramework) generateReports() error {
	for _, reporter := range f.reporters {
		var output io.Writer

		if f.config.OutputPath != "" {
			file, err := os.Create(f.config.OutputPath)
			if err != nil {
				return err
			}
			defer file.Close()
			output = file
		} else {
			output = os.Stdout
		}

		if err := reporter.GenerateReport(f.results, output); err != nil {
			return err
		}
	}

	return nil
}

// HashFile computes SHA256 hash of a file
func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ExtractStrings extracts string literals from code
func ExtractStrings(content string) []string {
	var strings []string

	// Match string literals
	re := regexp.MustCompile(`"([^"\\]|\\.)*"|` + "`" + `([^` + "`" + `])*` + "`")
	matches := re.FindAllString(content, -1)

	for _, match := range matches {
		// Remove quotes
		str := match[1 : len(match)-1]
		if len(str) > 0 {
			strings = append(strings, str)
		}
	}

	return strings
}

// IsTestFile checks if a file is a test file
func IsTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go") ||
		strings.Contains(path, "/test/") ||
		strings.Contains(path, "/tests/")
}

// GetFunctionName extracts function name from a line of code
func GetFunctionName(line string) string {
	// Match function declarations
	re := regexp.MustCompile(`func\s+(\(.*?\)\s+)?(\w+)\s*\(`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 2 {
		return matches[2]
	}
	return ""
}
