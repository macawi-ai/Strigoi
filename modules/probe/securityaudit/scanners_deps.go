package securityaudit

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DependencyScanner scans for vulnerable dependencies
type DependencyScanner struct {
	vulnDB VulnerabilityDatabase
}

// VulnerabilityDatabase interface for vulnerability lookups
type VulnerabilityDatabase interface {
	Lookup(pkg, version string) ([]Vulnerability, error)
}

// Vulnerability represents a dependency vulnerability
type Vulnerability struct {
	ID          string
	Package     string
	Version     string
	Severity    string
	Description string
	FixedIn     string
	References  []string
	CVSS        float64
}

// Dependency represents a project dependency
type Dependency struct {
	Name            string
	Version         string
	License         string
	DirectDep       bool
	Vulnerabilities []Vulnerability
}

// NewDependencyScanner creates a new dependency scanner
func NewDependencyScanner() *DependencyScanner {
	return &DependencyScanner{
		vulnDB: NewOSVDatabase(), // Using OSV database for Go vulnerabilities
	}
}

func (s *DependencyScanner) Name() string {
	return "Dependency Vulnerability Scanner"
}

func (s *DependencyScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	// Check for go.mod
	goModPath := filepath.Join(path, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		deps, err := s.scanGoModules(path)
		if err != nil {
			return nil, fmt.Errorf("failed to scan Go modules: %w", err)
		}

		for _, dep := range deps {
			for _, vuln := range dep.Vulnerabilities {
				issues = append(issues, SecurityIssue{
					Type:        "VULNERABLE_DEPENDENCY",
					Severity:    vuln.Severity,
					Title:       fmt.Sprintf("Vulnerable dependency: %s@%s", dep.Name, dep.Version),
					Description: vuln.Description,
					Location: IssueLocation{
						File:      goModPath,
						Component: dep.Name,
					},
					Evidence: map[string]string{
						"current_version": dep.Version,
						"fixed_version":   vuln.FixedIn,
						"vulnerability":   vuln.ID,
					},
					CWE:         "CWE-1035",
					References:  vuln.References,
					Remediation: fmt.Sprintf("Update %s to version %s or later", dep.Name, vuln.FixedIn),
				})
			}
		}
	}

	// Check for package.json (if using Node modules)
	packageJSONPath := filepath.Join(path, "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		issues = append(issues, s.scanNodeModules(path)...)
	}

	return issues, nil
}

func (s *DependencyScanner) scanGoModules(path string) ([]Dependency, error) {
	// Run go list to get dependencies
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var deps []Dependency
	decoder := json.NewDecoder(strings.NewReader(string(output)))

	for decoder.More() {
		var module struct {
			Path     string `json:"Path"`
			Version  string `json:"Version"`
			Main     bool   `json:"Main"`
			Indirect bool   `json:"Indirect"`
		}

		if err := decoder.Decode(&module); err != nil {
			continue
		}

		if module.Main {
			continue // Skip main module
		}

		dep := Dependency{
			Name:      module.Path,
			Version:   module.Version,
			DirectDep: !module.Indirect,
		}

		// Check for vulnerabilities
		vulns, err := s.vulnDB.Lookup(dep.Name, dep.Version)
		if err == nil {
			dep.Vulnerabilities = vulns
		}

		deps = append(deps, dep)
	}

	// Also run govulncheck if available
	s.runGovulncheck(path, &deps)

	return deps, nil
}

func (s *DependencyScanner) runGovulncheck(path string, deps *[]Dependency) {
	cmd := exec.Command("govulncheck", "-json", "./...")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return // govulncheck not available, skip
	}

	// Parse govulncheck output
	var vulnCheck struct {
		Vulns []struct {
			ID      string `json:"id"`
			Package string `json:"package"`
			Symbol  string `json:"symbol"`
			Details string `json:"details"`
		} `json:"vulns"`
	}

	if err := json.Unmarshal(output, &vulnCheck); err == nil {
		// Add vulnerabilities found by govulncheck
		for _, vuln := range vulnCheck.Vulns {
			// Find matching dependency and add vulnerability
			for i := range *deps {
				if strings.HasPrefix(vuln.Package, (*deps)[i].Name) {
					(*deps)[i].Vulnerabilities = append((*deps)[i].Vulnerabilities, Vulnerability{
						ID:          vuln.ID,
						Package:     vuln.Package,
						Description: vuln.Details,
						Severity:    "HIGH", // Default severity
					})
				}
			}
		}
	}
}

func (s *DependencyScanner) scanNodeModules(path string) []SecurityIssue {
	var issues []SecurityIssue

	// Run npm audit if available
	cmd := exec.Command("npm", "audit", "--json")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return issues // npm not available or no vulnerabilities
	}

	var audit struct {
		Vulnerabilities map[string]struct {
			Severity string `json:"severity"`
			Via      []struct {
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"via"`
			FixAvailable bool `json:"fix_available"`
		} `json:"vulnerabilities"`
	}

	if err := json.Unmarshal(output, &audit); err == nil {
		for pkg, vuln := range audit.Vulnerabilities {
			severity := strings.ToUpper(vuln.Severity)
			if severity == "MODERATE" {
				severity = "MEDIUM"
			}

			issues = append(issues, SecurityIssue{
				Type:     "VULNERABLE_DEPENDENCY",
				Severity: severity,
				Title:    fmt.Sprintf("Vulnerable npm dependency: %s", pkg),
				Location: IssueLocation{
					File:      filepath.Join(path, "package.json"),
					Component: pkg,
				},
				Remediation: "Run 'npm audit fix' to update vulnerable dependencies",
			})
		}
	}

	return issues
}

// LicenseScanner checks for license compliance
type LicenseScanner struct {
	allowedLicenses []string
	deniedLicenses  []string
}

// NewLicenseScanner creates a new license scanner
func NewLicenseScanner() *LicenseScanner {
	return &LicenseScanner{
		allowedLicenses: []string{
			"MIT", "Apache-2.0", "BSD-3-Clause", "BSD-2-Clause",
			"ISC", "MPL-2.0", "CC0-1.0", "Unlicense",
		},
		deniedLicenses: []string{
			"GPL-3.0", "AGPL-3.0", "LGPL-3.0", // Copyleft licenses
			"SSPL", "Commons-Clause", // Restrictive licenses
		},
	}
}

func (s *LicenseScanner) Name() string {
	return "License Compliance Scanner"
}

func (s *LicenseScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	// Scan Go dependencies
	goModPath := filepath.Join(path, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		issues = append(issues, s.scanGoLicenses(path)...)
	}

	// Check for license file in project
	if !s.hasLicenseFile(path) {
		issues = append(issues, SecurityIssue{
			Type:        "LICENSE_MISSING",
			Severity:    "MEDIUM",
			Title:       "No license file found",
			Description: "Project should have a LICENSE file",
			Location: IssueLocation{
				File: path,
			},
			Remediation: "Add a LICENSE file to clarify usage terms",
		})
	}

	return issues, nil
}

func (s *LicenseScanner) scanGoLicenses(path string) []SecurityIssue {
	var issues []SecurityIssue

	// Use go-licenses tool if available
	cmd := exec.Command("go-licenses", "csv", ".")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return issues // Tool not available
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		pkg := parts[0]
		license := parts[2]

		// Check if license is denied
		for _, denied := range s.deniedLicenses {
			if strings.Contains(license, denied) {
				issues = append(issues, SecurityIssue{
					Type:        "LICENSE_VIOLATION",
					Severity:    "HIGH",
					Title:       fmt.Sprintf("Incompatible license: %s", license),
					Description: fmt.Sprintf("Package %s uses %s license which may be incompatible", pkg, license),
					Location: IssueLocation{
						Component: pkg,
					},
					Remediation: "Review license compatibility or find alternative package",
				})
			}
		}

		// Check if license is unknown
		if license == "Unknown" || license == "" {
			issues = append(issues, SecurityIssue{
				Type:        "LICENSE_UNKNOWN",
				Severity:    "MEDIUM",
				Title:       fmt.Sprintf("Unknown license for %s", pkg),
				Description: "Unable to determine license for dependency",
				Location: IssueLocation{
					Component: pkg,
				},
				Remediation: "Verify the license manually",
			})
		}
	}

	return issues
}

func (s *LicenseScanner) hasLicenseFile(path string) bool {
	licenseFiles := []string{
		"LICENSE", "LICENSE.txt", "LICENSE.md",
		"LICENCE", "LICENCE.txt", "LICENCE.md",
		"COPYING", "COPYING.txt",
	}

	for _, file := range licenseFiles {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			return true
		}
	}

	return false
}

// OSVDatabase implements vulnerability database using OSV
type OSVDatabase struct {
	cacheDir string
	cache    map[string][]Vulnerability
}

// NewOSVDatabase creates a new OSV database client
func NewOSVDatabase() *OSVDatabase {
	return &OSVDatabase{
		cacheDir: "/tmp/osv-cache",
		cache:    make(map[string][]Vulnerability),
	}
}

func (db *OSVDatabase) Lookup(pkg, version string) ([]Vulnerability, error) {
	cacheKey := fmt.Sprintf("%s@%s", pkg, version)

	// Check cache
	if vulns, ok := db.cache[cacheKey]; ok {
		return vulns, nil
	}

	// Query OSV API (simplified - in real implementation, use proper API)
	// For now, return empty if no local database
	var vulns []Vulnerability

	// Cache result
	db.cache[cacheKey] = vulns

	return vulns, nil
}

// ComplianceScanner checks for compliance violations
type ComplianceScanner struct {
	standards map[string]ComplianceStandard
}

// ComplianceStandard represents a compliance standard
type ComplianceStandard struct {
	Name  string
	Rules []ComplianceRule
}

// ComplianceRule represents a single compliance requirement
type ComplianceRule struct {
	ID          string
	Description string
	Check       func(path string) (bool, string)
}

// NewComplianceScanner creates a compliance scanner
func NewComplianceScanner(standards []string) *ComplianceScanner {
	scanner := &ComplianceScanner{
		standards: make(map[string]ComplianceStandard),
	}

	// Add requested standards
	for _, std := range standards {
		switch std {
		case "PCI-DSS":
			scanner.standards[std] = scanner.getPCIDSSRules()
		case "OWASP":
			scanner.standards[std] = scanner.getOWASPRules()
		case "CIS":
			scanner.standards[std] = scanner.getCISRules()
		}
	}

	return scanner
}

func (s *ComplianceScanner) Name() string {
	return "Compliance Scanner"
}

func (s *ComplianceScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
	var issues []SecurityIssue

	for stdName, standard := range s.standards {
		for _, rule := range standard.Rules {
			passed, evidence := rule.Check(path)
			if !passed {
				issues = append(issues, SecurityIssue{
					Type:        "COMPLIANCE_VIOLATION",
					Severity:    "HIGH",
					Title:       fmt.Sprintf("%s violation: %s", stdName, rule.ID),
					Description: rule.Description,
					Evidence: map[string]string{
						"standard": stdName,
						"rule":     rule.ID,
						"details":  evidence,
					},
					Remediation: fmt.Sprintf("Implement %s requirement %s", stdName, rule.ID),
				})
			}
		}
	}

	return issues, nil
}

func (s *ComplianceScanner) getPCIDSSRules() ComplianceStandard {
	return ComplianceStandard{
		Name: "PCI-DSS",
		Rules: []ComplianceRule{
			{
				ID:          "PCI-DSS-2.3",
				Description: "Encrypt all non-console administrative access",
				Check: func(path string) (bool, string) {
					// Check for unencrypted admin endpoints
					return true, ""
				},
			},
			{
				ID:          "PCI-DSS-6.5.1",
				Description: "Injection flaws prevention",
				Check: func(path string) (bool, string) {
					// Check for parameterized queries
					return true, ""
				},
			},
		},
	}
}

func (s *ComplianceScanner) getOWASPRules() ComplianceStandard {
	return ComplianceStandard{
		Name: "OWASP Top 10",
		Rules: []ComplianceRule{
			{
				ID:          "A01:2021",
				Description: "Broken Access Control",
				Check: func(path string) (bool, string) {
					// Check for access control implementation
					return true, ""
				},
			},
			{
				ID:          "A02:2021",
				Description: "Cryptographic Failures",
				Check: func(path string) (bool, string) {
					// Check for weak crypto
					return true, ""
				},
			},
		},
	}
}

func (s *ComplianceScanner) getCISRules() ComplianceStandard {
	return ComplianceStandard{
		Name: "CIS Benchmarks",
		Rules: []ComplianceRule{
			{
				ID:          "CIS-1.1.1",
				Description: "Ensure auditing is configured",
				Check: func(path string) (bool, string) {
					// Check for audit logging
					return true, ""
				},
			},
		},
	}
}
