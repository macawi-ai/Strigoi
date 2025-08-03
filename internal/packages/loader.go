package packages

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/macawi-ai/strigoi/internal/core"
	"gopkg.in/yaml.v3"
)

// PackageType represents the type of protocol package
type PackageType string

const (
	OfficialPackage  PackageType = "official"
	CommunityPackage PackageType = "community"
	UpdatePackage    PackageType = "updates"
)

// ProtocolPackage represents a parsed APMS protocol package
type ProtocolPackage struct {
	Header struct {
		ProtocolIdentity struct {
			Name    string `yaml:"name"`
			Version string `yaml:"version"`
			UUID    string `yaml:"uuid"`
			Family  string `yaml:"family"`
		} `yaml:"protocol_identity"`
		
		StrigoiMetadata struct {
			PackageType    string    `yaml:"package_type"`
			PackageVersion string    `yaml:"package_version"`
			LastUpdated    time.Time `yaml:"last_updated"`
			Compatibility  string    `yaml:"compatibility"`
		} `yaml:"strigoi_metadata"`
		
		SecurityAssessment struct {
			TestCoverage      float64   `yaml:"test_coverage"`
			VulnerabilityCount int      `yaml:"vulnerability_count"`
			CriticalFindings   int      `yaml:"critical_findings"`
			LastAssessment     time.Time `yaml:"last_assessment"`
		} `yaml:"security_assessment"`
	} `yaml:"header"`
	
	Payload struct {
		TestModules []TestModuleDefinition `yaml:"test_modules"`
		ProtocolIntelligence struct {
			KnownImplementations   []Implementation      `yaml:"known_implementations"`
			CommonVulnerabilities  []Vulnerability       `yaml:"common_vulnerabilities"`
			AttackChains          []AttackChain         `yaml:"attack_chains"`
		} `yaml:"protocol_intelligence"`
		UpdateConfiguration struct {
			UpdateSource         string   `yaml:"update_source"`
			UpdateFrequency      string   `yaml:"update_frequency"`
			SignatureVerification bool    `yaml:"signature_verification"`
			RollbackSupported    bool     `yaml:"rollback_supported"`
			ScheduledUpdates     []Update `yaml:"scheduled_updates"`
		} `yaml:"update_configuration"`
	} `yaml:"payload"`
	
	Distribution struct {
		Channels     []string `yaml:"channels"`
		Dependencies []string `yaml:"dependencies"`
		Verification struct {
			Checksum  string `yaml:"checksum"`
			Signature string `yaml:"signature"`
		} `yaml:"verification"`
	} `yaml:"distribution"`
}

// TestModuleDefinition defines a test module in the package
type TestModuleDefinition struct {
	ModuleID     string       `yaml:"module_id"`
	ModuleType   string       `yaml:"module_type"`
	RiskLevel    string       `yaml:"risk_level"`
	Status       string       `yaml:"status,omitempty"`
	TestVectors  []TestVector `yaml:"test_vectors"`
}

// TestVector represents a specific test within a module
type TestVector struct {
	Vector           string                 `yaml:"vector"`
	Description      string                 `yaml:"description,omitempty"`
	SeverityChecks   []SeverityCheck       `yaml:"severity_checks,omitempty"`
	InjectionPatterns []string             `yaml:"injection_patterns,omitempty"`
	SensitivePatterns []SensitivePattern   `yaml:"sensitive_patterns,omitempty"`
	Parameters       map[string]interface{} `yaml:"parameters,omitempty"`
}

// SeverityCheck defines pattern-based severity classification
type SeverityCheck struct {
	Pattern  string `yaml:"pattern"`
	Severity string `yaml:"severity"`
}

// SensitivePattern defines sensitive data patterns
type SensitivePattern struct {
	Category string   `yaml:"category"`
	Patterns []string `yaml:"patterns"`
	Severity string   `yaml:"severity"`
}

// Implementation represents a known protocol implementation
type Implementation struct {
	Name          string   `yaml:"name"`
	Vendor        string   `yaml:"vendor"`
	SpecificTests []string `yaml:"specific_tests"`
}

// Vulnerability represents a known vulnerability
type Vulnerability struct {
	CVE              string   `yaml:"cve"`
	Description      string   `yaml:"description"`
	AffectedVersions string   `yaml:"affected_versions"`
	Severity         string   `yaml:"severity"`
}

// AttackChain represents a multi-step attack sequence
type AttackChain struct {
	ChainID    string   `yaml:"chain_id"`
	Name       string   `yaml:"name"`
	Steps      []string `yaml:"steps"`
	Complexity string   `yaml:"complexity"`
	Impact     string   `yaml:"impact"`
}

// Update represents a scheduled update
type Update struct {
	Date    time.Time `yaml:"date"`
	Changes []string  `yaml:"changes"`
}

// PackageLoader handles loading and managing protocol packages
type PackageLoader struct {
	baseDir       string
	packages      map[string]*ProtocolPackage
	moduleFactory ModuleFactory
	updateClient  *http.Client
	mu            sync.RWMutex
	logger        core.Logger
}

// ModuleFactory creates modules from package definitions
type ModuleFactory interface {
	CreateModule(def TestModuleDefinition, pkg *ProtocolPackage) (core.Module, error)
}

// NewPackageLoader creates a new package loader
func NewPackageLoader(baseDir string, factory ModuleFactory, logger core.Logger) *PackageLoader {
	return &PackageLoader{
		baseDir:      baseDir,
		packages:     make(map[string]*ProtocolPackage),
		moduleFactory: factory,
		updateClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:       logger,
	}
}

// LoadPackages loads all packages from the filesystem
func (pl *PackageLoader) LoadPackages() error {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	
	// Load packages from each directory
	for _, pkgType := range []PackageType{OfficialPackage, CommunityPackage, UpdatePackage} {
		dir := filepath.Join(pl.baseDir, string(pkgType))
		if err := pl.loadPackagesFromDir(dir, pkgType); err != nil {
			pl.logger.Error("Failed to load %s packages: %v", pkgType, err)
		}
	}
	
	pl.logger.Info("Loaded %d protocol packages", len(pl.packages))
	return nil
}

// loadPackagesFromDir loads packages from a specific directory
func (pl *PackageLoader) loadPackagesFromDir(dir string, pkgType PackageType) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist yet
		}
		return fmt.Errorf("failed to read directory: %w", err)
	}
	
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".apms.yaml") {
			continue
		}
		
		pkgPath := filepath.Join(dir, entry.Name())
		pkg, err := pl.loadPackage(pkgPath)
		if err != nil {
			pl.logger.Error("Failed to load package %s: %v", entry.Name(), err)
			continue
		}
		
		// Store by protocol UUID
		pl.packages[pkg.Header.ProtocolIdentity.UUID] = pkg
		pl.logger.Info("Loaded package: %s v%s", 
			pkg.Header.ProtocolIdentity.Name,
			pkg.Header.ProtocolIdentity.Version)
	}
	
	return nil
}

// loadPackage loads a single package from file
func (pl *PackageLoader) loadPackage(path string) (*ProtocolPackage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read package file: %w", err)
	}
	
	var pkg ProtocolPackage
	if err := yaml.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package YAML: %w", err)
	}
	
	// TODO: Verify package signature
	// if pkg.Distribution.Verification.Signature != "" {
	//     if err := pl.verifySignature(&pkg, data); err != nil {
	//         return nil, fmt.Errorf("signature verification failed: %w", err)
	//     }
	// }
	
	return &pkg, nil
}

// GetPackage retrieves a loaded package by UUID
func (pl *PackageLoader) GetPackage(uuid string) (*ProtocolPackage, bool) {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	
	pkg, exists := pl.packages[uuid]
	return pkg, exists
}

// GetPackageByName retrieves a package by protocol name
func (pl *PackageLoader) GetPackageByName(name string) (*ProtocolPackage, bool) {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	
	for _, pkg := range pl.packages {
		if pkg.Header.ProtocolIdentity.Name == name {
			return pkg, true
		}
	}
	
	return nil, false
}

// ListPackages returns all loaded packages
func (pl *PackageLoader) ListPackages() []*ProtocolPackage {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	
	packages := make([]*ProtocolPackage, 0, len(pl.packages))
	for _, pkg := range pl.packages {
		packages = append(packages, pkg)
	}
	
	return packages
}

// CheckForUpdates checks for package updates
func (pl *PackageLoader) CheckForUpdates(ctx context.Context) error {
	pl.mu.RLock()
	packages := make([]*ProtocolPackage, 0, len(pl.packages))
	for _, pkg := range pl.packages {
		packages = append(packages, pkg)
	}
	pl.mu.RUnlock()
	
	for _, pkg := range packages {
		if pkg.Payload.UpdateConfiguration.UpdateSource == "" {
			continue
		}
		
		pl.logger.Info("Checking updates for %s from %s",
			pkg.Header.ProtocolIdentity.Name,
			pkg.Payload.UpdateConfiguration.UpdateSource)
		
		if err := pl.checkPackageUpdate(ctx, pkg); err != nil {
			pl.logger.Error("Failed to check updates for %s: %v",
				pkg.Header.ProtocolIdentity.Name, err)
		}
	}
	
	return nil
}

// checkPackageUpdate checks for updates for a specific package
func (pl *PackageLoader) checkPackageUpdate(ctx context.Context, pkg *ProtocolPackage) error {
	updateURL := fmt.Sprintf("%s/latest.apms.yaml", pkg.Payload.UpdateConfiguration.UpdateSource)
	
	req, err := http.NewRequestWithContext(ctx, "GET", updateURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add current version header
	req.Header.Set("X-Current-Version", pkg.Header.StrigoiMetadata.PackageVersion)
	
	resp, err := pl.updateClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch update: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotModified {
		pl.logger.Info("Package %s is up to date", pkg.Header.ProtocolIdentity.Name)
		return nil
	}
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update server returned status %d", resp.StatusCode)
	}
	
	// Download update
	updateData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read update data: %w", err)
	}
	
	// Parse update
	var updatedPkg ProtocolPackage
	if err := yaml.Unmarshal(updateData, &updatedPkg); err != nil {
		return fmt.Errorf("failed to parse update: %w", err)
	}
	
	// Check if update is newer
	if updatedPkg.Header.StrigoiMetadata.LastUpdated.After(pkg.Header.StrigoiMetadata.LastUpdated) {
		pl.logger.Info("New version available for %s: %s -> %s",
			pkg.Header.ProtocolIdentity.Name,
			pkg.Header.StrigoiMetadata.PackageVersion,
			updatedPkg.Header.StrigoiMetadata.PackageVersion)
		
		// Save update to updates directory
		updatePath := filepath.Join(pl.baseDir, string(UpdatePackage), 
			fmt.Sprintf("%s-%s.apms.yaml", 
				updatedPkg.Header.ProtocolIdentity.Name,
				updatedPkg.Header.StrigoiMetadata.PackageVersion))
		
		if err := os.WriteFile(updatePath, updateData, 0644); err != nil {
			return fmt.Errorf("failed to save update: %w", err)
		}
		
		// Reload to pick up the update
		if newPkg, err := pl.loadPackage(updatePath); err == nil {
			pl.mu.Lock()
			pl.packages[newPkg.Header.ProtocolIdentity.UUID] = newPkg
			pl.mu.Unlock()
			pl.logger.Info("Successfully updated %s", pkg.Header.ProtocolIdentity.Name)
		}
	}
	
	return nil
}

// GenerateModules creates Strigoi modules from loaded packages
func (pl *PackageLoader) GenerateModules() ([]core.Module, error) {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	
	var modules []core.Module
	
	for _, pkg := range pl.packages {
		for _, moduleDef := range pkg.Payload.TestModules {
			// Skip planned modules
			if moduleDef.Status == "planned" {
				continue
			}
			
			module, err := pl.moduleFactory.CreateModule(moduleDef, pkg)
			if err != nil {
				pl.logger.Error("Failed to create module %s: %v", moduleDef.ModuleID, err)
				continue
			}
			
			modules = append(modules, module)
		}
	}
	
	return modules, nil
}