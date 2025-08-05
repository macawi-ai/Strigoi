package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/pkg/modules"
	"github.com/macawi-ai/strigoi/pkg/security"
)

func init() {
	// Register the module factory
	modules.RegisterBuiltin("probe/south", NewSouthModule)
}

// SouthModule probes dependencies and supply chain.
type SouthModule struct {
	modules.BaseModule
	executor   *security.SecureExecutor
	mcpScanner *security.MCPScanner
}

// NewSouthModule creates a new South probe module.
func NewSouthModule() modules.Module {
	return &SouthModule{
		BaseModule: modules.BaseModule{
			ModuleName:        "probe/south",
			ModuleDescription: "Analyze dependencies, libraries, and supply chain vulnerabilities",
			ModuleType:        modules.ProbeModule,
			ModuleOptions: map[string]*modules.ModuleOption{
				"target": {
					Name:        "target",
					Description: "Target directory to analyze",
					Type:        "string",
					Required:    true,
					Default:     ".",
				},
				"package_manager": {
					Name:        "package_manager",
					Description: "Package manager to use (auto-detect if empty)",
					Type:        "string",
					Required:    false,
					Default:     "auto",
				},
				"max_depth": {
					Name:        "max_depth",
					Description: "Maximum dependency tree depth",
					Type:        "int",
					Required:    false,
					Default:     10,
				},
				"skip_dev": {
					Name:        "skip_dev",
					Description: "Skip development dependencies",
					Type:        "bool",
					Required:    false,
					Default:     false,
				},
				"cache": {
					Name:        "cache",
					Description: "Use cached results if available",
					Type:        "bool",
					Required:    false,
					Default:     true,
				},
				"scan_mcp": {
					Name:        "scan_mcp",
					Description: "Enable MCP tools scanning",
					Type:        "bool",
					Required:    false,
					Default:     false,
				},
				"include_self": {
					Name:        "include_self",
					Description: "Include Strigoi's own files and processes in scan",
					Type:        "bool",
					Required:    false,
					Default:     false,
				},
			},
		},
		executor:   security.NewSecureExecutor(),
		mcpScanner: security.NewMCPScanner(security.NewSecureExecutor()),
	}
}

// Check verifies the module can run.
func (m *SouthModule) Check() bool {
	// Verify target exists
	targetOpt, _ := m.GetOption("target")
	target := targetOpt.(string)

	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		return false
	}

	return true
}

// Info returns module metadata.
func (m *SouthModule) Info() *modules.ModuleInfo {
	return &modules.ModuleInfo{
		Author:  "Strigoi Team",
		Version: "1.0.0",
		Tags:    []string{"dependencies", "supply-chain", "vulnerabilities", "licenses"},
		References: []string{
			"https://owasp.org/www-project-dependency-check/",
			"https://nvd.nist.gov/",
		},
	}
}

// Run executes the south probe.
func (m *SouthModule) Run() (*modules.ModuleResult, error) {
	startTime := time.Now()

	// Get options
	targetOpt, _ := m.GetOption("target")
	target := targetOpt.(string)

	// Validate target path
	if err := m.executor.ValidatePath(target); err != nil {
		return nil, fmt.Errorf("invalid target path: %w", err)
	}

	// Detect package manager
	pm, manifestFile := m.detectPackageManager(target)
	if pm == "" {
		return &modules.ModuleResult{
			Module:    m.Name(),
			Status:    "failed",
			StartTime: startTime,
			EndTime:   time.Now(),
			Error:     "No package manager detected",
		}, nil
	}

	// Initialize result
	result := &SupplyChainResult{
		PackageManager:  pm,
		ManifestFile:    manifestFile,
		Dependencies:    []Dependency{},
		Vulnerabilities: []Vulnerability{},
		Licenses:        make(map[string]int),
		MCPTools:        []security.MCPTool{},
	}

	// Analyze based on package manager
	var err error
	switch pm {
	case "npm":
		err = m.analyzeNPM(target, result)
	case "pip":
		err = m.analyzePip(target, result)
	case "go":
		err = m.analyzeGo(target, result)
	default:
		err = fmt.Errorf("unsupported package manager: %s", pm)
	}

	if err != nil {
		return &modules.ModuleResult{
			Module:    m.Name(),
			Status:    "failed",
			StartTime: startTime,
			EndTime:   time.Now(),
			Error:     err.Error(),
		}, nil
	}

	// Perform MCP scanning if enabled
	scanMCPOpt, _ := m.GetOption("scan_mcp")
	if scanMCP := scanMCPOpt.(bool); scanMCP {
		// Configure include-self option
		includeSelfOpt, _ := m.GetOption("include_self")
		includeSelf := includeSelfOpt.(bool)
		m.mcpScanner.SetIncludeSelf(includeSelf)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		mcpTools, mcpErr := m.mcpScanner.DiscoverMCPTools(ctx)
		if mcpErr != nil {
			// Log error but don't fail the module
			fmt.Printf("Warning: MCP scanning encountered errors: %v\n", mcpErr)
		} else {
			result.MCPTools = mcpTools
		}
	}

	// Calculate summary
	result.Summary = m.calculateSummary(result)

	return &modules.ModuleResult{
		Module:    m.Name(),
		Status:    "completed",
		StartTime: startTime,
		EndTime:   time.Now(),
		Data: map[string]interface{}{
			"result": result,
		},
	}, nil
}

// detectPackageManager identifies the package manager in use.
func (m *SouthModule) detectPackageManager(path string) (string, string) {
	// Check for manifest files
	manifests := map[string]string{
		"package.json":      "npm",
		"package-lock.json": "npm",
		"yarn.lock":         "yarn",
		"requirements.txt":  "pip",
		"Pipfile":           "pip",
		"go.mod":            "go",
		"Cargo.toml":        "cargo",
		"pom.xml":           "maven",
		"build.gradle":      "gradle",
	}

	for manifest, pm := range manifests {
		manifestPath := filepath.Join(path, manifest)
		if _, err := os.Stat(manifestPath); err == nil {
			return pm, manifest
		}
	}

	return "", ""
}

// analyzeNPM analyzes Node.js dependencies.
func (m *SouthModule) analyzeNPM(path string, result *SupplyChainResult) error {
	// Read package.json
	packageJSON := filepath.Join(path, "package.json")
	data, err := os.ReadFile(packageJSON)
	if err != nil {
		return fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("failed to parse package.json: %w", err)
	}

	// Parse dependencies
	skipDev, _ := m.GetOption("skip_dev")
	for name, version := range pkg.Dependencies {
		result.Dependencies = append(result.Dependencies, Dependency{
			Name:    name,
			Version: version,
			Type:    "direct",
			License: "unknown", // Would need to look this up
		})
	}

	if !skipDev.(bool) {
		for name, version := range pkg.DevDependencies {
			result.Dependencies = append(result.Dependencies, Dependency{
				Name:    name,
				Version: version,
				Type:    "dev",
				License: "unknown",
			})
		}
	}

	// Run npm audit if available
	if m.executor.CommandExists("npm") {
		vulns, _ := m.runNPMAudit(path)
		result.Vulnerabilities = append(result.Vulnerabilities, vulns...)
	}

	return nil
}

// analyzePip analyzes Python dependencies.
func (m *SouthModule) analyzePip(path string, result *SupplyChainResult) error {
	// Check for requirements.txt
	reqFile := filepath.Join(path, "requirements.txt")
	if _, err := os.Stat(reqFile); err != nil {
		return fmt.Errorf("requirements.txt not found")
	}

	data, err := os.ReadFile(reqFile)
	if err != nil {
		return fmt.Errorf("failed to read requirements.txt: %w", err)
	}

	// Parse requirements
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Simple parsing - would need more robust parsing in production
		parts := strings.Split(line, "==")
		name := parts[0]
		version := "latest"
		if len(parts) > 1 {
			version = parts[1]
		}

		result.Dependencies = append(result.Dependencies, Dependency{
			Name:    name,
			Version: version,
			Type:    "direct",
		})
	}

	// Run safety check if available
	if m.executor.CommandExists("safety") {
		vulns, _ := m.runSafetyCheck(path)
		result.Vulnerabilities = append(result.Vulnerabilities, vulns...)
	}

	return nil
}

// analyzeGo analyzes Go dependencies.
func (m *SouthModule) analyzeGo(path string, result *SupplyChainResult) error {
	// Read go.mod
	goMod := filepath.Join(path, "go.mod")
	data, err := os.ReadFile(goMod)
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	// Parse go.mod (simplified)
	lines := strings.Split(string(data), "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "require (" {
			inRequire = true
			continue
		}

		if line == ")" {
			inRequire = false
			continue
		}

		if inRequire && line != "" {
			// Parse dependency line
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				result.Dependencies = append(result.Dependencies, Dependency{
					Name:    parts[0],
					Version: parts[1],
					Type:    "direct",
				})
			}
		}
	}

	// Run govulncheck if available
	if m.executor.CommandExists("govulncheck") {
		vulns, _ := m.runGovulncheck(path)
		result.Vulnerabilities = append(result.Vulnerabilities, vulns...)
	}

	return nil
}

// calculateSummary generates summary statistics.
func (m *SouthModule) calculateSummary(result *SupplyChainResult) Summary {
	summary := Summary{
		TotalDependencies: len(result.Dependencies),
		Vulnerabilities: VulnSummary{
			Critical: 0,
			High:     0,
			Medium:   0,
			Low:      0,
		},
		Licenses: LicenseSummary{
			Permissive: 0,
			Copyleft:   0,
			Commercial: 0,
			Unknown:    0,
		},
	}

	// Count direct vs transitive
	for _, dep := range result.Dependencies {
		if dep.Type == "direct" {
			summary.DirectDependencies++
		} else if dep.Type == "transitive" {
			summary.TransitiveDependencies++
		}
	}

	// Count vulnerabilities by severity
	for _, vuln := range result.Vulnerabilities {
		switch strings.ToLower(vuln.Severity) {
		case "critical":
			summary.Vulnerabilities.Critical++
		case "high":
			summary.Vulnerabilities.High++
		case "medium":
			summary.Vulnerabilities.Medium++
		case "low":
			summary.Vulnerabilities.Low++
		}
	}

	// Count licenses
	for license, count := range result.Licenses {
		if isPermissiveLicense(license) {
			summary.Licenses.Permissive += count
		} else if isCopyleftLicense(license) {
			summary.Licenses.Copyleft += count
		} else if isCommercialLicense(license) {
			summary.Licenses.Commercial += count
		} else {
			summary.Licenses.Unknown += count
		}
	}

	return summary
}

// Helper functions for license classification.
func isPermissiveLicense(license string) bool {
	permissive := []string{"MIT", "Apache-2.0", "BSD-3-Clause", "BSD-2-Clause", "ISC"}
	license = strings.ToUpper(license)
	for _, p := range permissive {
		if strings.Contains(license, strings.ToUpper(p)) {
			return true
		}
	}
	return false
}

func isCopyleftLicense(license string) bool {
	copyleft := []string{"GPL", "LGPL", "AGPL", "MPL"}
	license = strings.ToUpper(license)
	for _, c := range copyleft {
		if strings.Contains(license, c) {
			return true
		}
	}
	return false
}

func isCommercialLicense(license string) bool {
	commercial := []string{"Commercial", "Proprietary", "EULA"}
	license = strings.ToUpper(license)
	for _, c := range commercial {
		if strings.Contains(license, strings.ToUpper(c)) {
			return true
		}
	}
	return false
}

// Stub implementations for tool runners.
func (m *SouthModule) runNPMAudit(_ string) ([]Vulnerability, error) {
	// TODO: Implement actual npm audit integration
	return []Vulnerability{}, nil
}

func (m *SouthModule) runSafetyCheck(_ string) ([]Vulnerability, error) {
	// TODO: Implement actual safety check integration
	return []Vulnerability{}, nil
}

func (m *SouthModule) runGovulncheck(_ string) ([]Vulnerability, error) {
	// TODO: Implement actual govulncheck integration
	return []Vulnerability{}, nil
}
