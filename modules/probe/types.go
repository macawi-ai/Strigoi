package probe

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/pkg/security"
)

// Dependency represents a package dependency
type Dependency struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	Type       string   `json:"type"` // direct, transitive, dev
	Path       []string `json:"path,omitempty"`
	License    string   `json:"license,omitempty"`
	Repository string   `json:"repository,omitempty"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	CVE            string   `json:"cve,omitempty"`
	CWE            string   `json:"cwe,omitempty"`
	Package        string   `json:"package"`
	Version        string   `json:"version"`
	Severity       string   `json:"severity"`
	CVSSScore      float64  `json:"cvss_score,omitempty"`
	Description    string   `json:"description"`
	Remediation    string   `json:"remediation"`
	DependencyPath []string `json:"dependency_path,omitempty"`
	Confidence     string   `json:"confidence"`
	References     []string `json:"references,omitempty"`
}

// SupplyChainResult contains dependency analysis results
type SupplyChainResult struct {
	PackageManager  string             `json:"package_manager"`
	ManifestFile    string             `json:"manifest_file"`
	Summary         Summary            `json:"summary"`
	Dependencies    []Dependency       `json:"dependencies"`
	Vulnerabilities []Vulnerability    `json:"vulnerabilities"`
	Licenses        map[string]int     `json:"licenses"`
	DependencyGraph *DependencyGraph   `json:"dependency_graph,omitempty"`
	MCPTools        []security.MCPTool `json:"mcp_tools,omitempty"`
}

// Summary provides high-level statistics
type Summary struct {
	TotalDependencies      int            `json:"total_dependencies"`
	DirectDependencies     int            `json:"direct_dependencies"`
	TransitiveDependencies int            `json:"transitive_dependencies"`
	Vulnerabilities        VulnSummary    `json:"vulnerabilities"`
	Licenses               LicenseSummary `json:"licenses"`
	UpdatesAvailable       int            `json:"updates_available,omitempty"`
}

// VulnSummary categorizes vulnerabilities by severity
type VulnSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

// LicenseSummary categorizes licenses by type
type LicenseSummary struct {
	Permissive int `json:"permissive"`
	Copyleft   int `json:"copyleft"`
	Commercial int `json:"commercial"`
	Unknown    int `json:"unknown"`
}

// DependencyGraph represents the dependency tree
type DependencyGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode represents a dependency in the graph
type GraphNode struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

// GraphEdge represents a dependency relationship
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// DataFlowResult contains data flow analysis results
type DataFlowResult struct {
	Summary          DataFlowSummary   `json:"summary"`
	Findings         []Finding         `json:"findings"`
	DataFlows        []DataFlow        `json:"data_flows"`
	ExternalServices []ExternalService `json:"external_services"`
}

// DataFlowSummary provides high-level statistics
type DataFlowSummary struct {
	ExternalServices int `json:"external_services"`
	PotentialSecrets int `json:"potential_secrets"`
	DataFlows        int `json:"data_flows"`
	LeakPoints       int `json:"leak_points"`
}

// Finding represents a security finding
type Finding struct {
	Type        string   `json:"type"`
	Category    string   `json:"category"`
	Location    string   `json:"location"`
	Confidence  string   `json:"confidence"`
	Severity    string   `json:"severity"` // NEW: critical/high/medium/low
	Evidence    string   `json:"evidence"`
	Impact      string   `json:"impact,omitempty"`
	Remediation string   `json:"remediation"`
	DataFlow    []string `json:"data_flow,omitempty"`
	References  []string `json:"references,omitempty"`
}

// DataFlow represents how data moves through the system
type DataFlow struct {
	ID              string   `json:"id"`
	Source          string   `json:"source"`
	Transformations []string `json:"transformations"`
	Destination     string   `json:"destination"`
	SensitiveData   []string `json:"sensitive_data,omitempty"`
	Protection      []string `json:"protection,omitempty"`
}

// ExternalService represents an external API or service
type ExternalService struct {
	Domain         string   `json:"domain"`
	Purpose        string   `json:"purpose"`
	Authentication string   `json:"authentication"`
	DataShared     []string `json:"data_shared"`
	Encrypted      bool     `json:"encrypted"`
}

// SecureExecutor provides safe command execution
type SecureExecutor struct {
	allowedPaths []string
	timeout      time.Duration
	maxOutput    int64
}

// NewSecureExecutor creates a secure executor with defaults
func NewSecureExecutor() *SecureExecutor {
	// Get working directory
	wd, _ := os.Getwd()

	return &SecureExecutor{
		allowedPaths: []string{
			wd,
			"/tmp",
			os.TempDir(),
		},
		timeout:   30 * time.Second,
		maxOutput: 10 * 1024 * 1024, // 10MB
	}
}

// ValidatePath ensures a path is within allowed directories
func (s *SecureExecutor) ValidatePath(path string) error {
	// Clean and resolve the path
	cleaned := filepath.Clean(path)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if it's within allowed paths
	isAllowed := false
	for _, base := range s.allowedPaths {
		baseAbs, _ := filepath.Abs(base)
		if strings.HasPrefix(abs, baseAbs) {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("path '%s' is outside allowed directories", abs)
	}

	// Ensure it exists and is accessible
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("path does not exist or is not accessible: %w", err)
	}

	// Additional checks
	if strings.Contains(abs, "..") {
		return fmt.Errorf("path contains suspicious patterns")
	}

	// Check if it's a symlink pointing outside allowed paths
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return fmt.Errorf("cannot resolve symlink: %w", err)
		}

		// Recursively validate the target
		return s.ValidatePath(target)
	}

	return nil
}

// CommandExists checks if a command is available
func (s *SecureExecutor) CommandExists(cmd string) bool {
	// Only check for known safe commands
	allowedCommands := []string{
		"npm", "yarn", "pip", "pip3", "pipenv",
		"go", "cargo", "maven", "gradle",
		"safety", "npm-audit", "govulncheck",
		"trivy", "snyk",
	}

	isAllowed := false
	for _, allowedCmd := range allowedCommands {
		if cmd == allowedCmd {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return false
	}

	_, err := exec.LookPath(cmd)
	return err == nil
}

// AddAllowedPath adds a path to the whitelist
func (s *SecureExecutor) AddAllowedPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Verify it exists
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}

	s.allowedPaths = append(s.allowedPaths, abs)
	return nil
}

// Pattern represents a detection pattern
type Pattern struct {
	Name       string
	Regex      string
	Confidence string
	Category   string
}

// Common patterns for secret detection
var SecretPatterns = []Pattern{
	{
		Name:       "AWS Access Key",
		Regex:      `AKIA[0-9A-Z]{16}`,
		Confidence: "high",
		Category:   "aws_key",
	},
	{
		Name:       "AWS Secret Key",
		Regex:      `[0-9a-zA-Z/+=]{40}`,
		Confidence: "medium",
		Category:   "aws_secret",
	},
	{
		Name:       "API Key Generic",
		Regex:      `[aA][pP][iI]_?[kK][eE][yY]\s*[:=]\s*['"]\w{16,}['"]`,
		Confidence: "medium",
		Category:   "api_key",
	},
	{
		Name:       "Private Key",
		Regex:      `-----BEGIN (RSA |EC )?PRIVATE KEY-----`,
		Confidence: "high",
		Category:   "private_key",
	},
	{
		Name:       "GitHub Token",
		Regex:      `gh[ps]_[0-9a-zA-Z]{36}`,
		Confidence: "high",
		Category:   "github_token",
	},
	{
		Name:       "Slack Token",
		Regex:      `xox[baprs]-[0-9a-zA-Z-]+`,
		Confidence: "high",
		Category:   "slack_token",
	},
	{
		Name:       "Generic Secret",
		Regex:      `[sS][eE][cC][rR][eE][tT]\s*[:=]\s*['"]\w{8,}['"]`,
		Confidence: "low",
		Category:   "generic_secret",
	},
}
