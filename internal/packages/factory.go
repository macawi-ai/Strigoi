package packages

import (
	"fmt"
	"strings"

	"github.com/macawi-ai/strigoi/internal/core"
	mcp "github.com/macawi-ai/strigoi/internal/modules.bak/mcp"
)

// DynamicModuleFactory creates modules dynamically from package definitions
type DynamicModuleFactory struct {
	logger core.Logger
}

// NewDynamicModuleFactory creates a new module factory
func NewDynamicModuleFactory(logger core.Logger) *DynamicModuleFactory {
	return &DynamicModuleFactory{
		logger: logger,
	}
}

// CreateModule creates a module from a package definition
func (f *DynamicModuleFactory) CreateModule(def TestModuleDefinition, pkg *ProtocolPackage) (core.Module, error) {
	f.logger.Debug("Creating module %s from package %s", def.ModuleID, pkg.Header.ProtocolIdentity.Name)
	
	// For now, we'll map to our existing modules
	// In the future, this could dynamically generate modules
	switch def.ModuleID {
	case "mcp/discovery/tools_list":
		return f.createToolsListModule(def, pkg), nil
	case "mcp/discovery/prompts_list":
		return f.createPromptsListModule(def, pkg), nil
	case "mcp/discovery/resources_list":
		return f.createResourcesListModule(def, pkg), nil
	case "mcp/auth/bypass":
		return f.createAuthBypassModule(def, pkg), nil
	case "mcp/dos/rate_limit":
		return f.createRateLimitModule(def, pkg), nil
	default:
		// For unknown modules, create a generic dynamic module
		return f.createDynamicModule(def, pkg), nil
	}
}

// createToolsListModule creates a tools list module with package-specific configuration
func (f *DynamicModuleFactory) createToolsListModule(def TestModuleDefinition, pkg *ProtocolPackage) core.Module {
	module := mcp.NewToolsListModule()
	
	// Apply package-specific configurations
	// In a real implementation, we'd enhance the module with package data
	// For example, severity patterns from the package
	
	return module
}

// createPromptsListModule creates a prompts list module
func (f *DynamicModuleFactory) createPromptsListModule(def TestModuleDefinition, pkg *ProtocolPackage) core.Module {
	return mcp.NewPromptsListModule()
}

// createResourcesListModule creates a resources list module
func (f *DynamicModuleFactory) createResourcesListModule(def TestModuleDefinition, pkg *ProtocolPackage) core.Module {
	return mcp.NewResourcesListModule()
}

// createAuthBypassModule creates an auth bypass module
func (f *DynamicModuleFactory) createAuthBypassModule(def TestModuleDefinition, pkg *ProtocolPackage) core.Module {
	return mcp.NewAuthBypassModule()
}

// createRateLimitModule creates a rate limit module
func (f *DynamicModuleFactory) createRateLimitModule(def TestModuleDefinition, pkg *ProtocolPackage) core.Module {
	return mcp.NewRateLimitModule()
}

// createDynamicModule creates a generic dynamic module for unknown module types
func (f *DynamicModuleFactory) createDynamicModule(def TestModuleDefinition, pkg *ProtocolPackage) core.Module {
	return &DynamicModule{
		BaseModule: mcp.NewBaseModule(),
		definition: def,
		pkg:        pkg,
	}
}

// DynamicModule is a generic module created from package definitions
type DynamicModule struct {
	*mcp.BaseModule
	definition TestModuleDefinition
	pkg        *ProtocolPackage
}

// Name returns the module name
func (m *DynamicModule) Name() string {
	return m.definition.ModuleID
}

// Description returns the module description
func (m *DynamicModule) Description() string {
	return fmt.Sprintf("Dynamic module: %s (Risk: %s)", m.definition.ModuleID, m.definition.RiskLevel)
}

// Type returns the module type
func (m *DynamicModule) Type() core.ModuleType {
	switch m.definition.ModuleType {
	case "discovery":
		return core.NetworkScanning
	case "attack":
		return core.NetworkScanning
	case "stress":
		return core.NetworkScanning
	default:
		return core.NetworkScanning
	}
}

// Info returns module information
func (m *DynamicModule) Info() *core.ModuleInfo {
	return &core.ModuleInfo{
		Name:        m.Name(),
		Version:     m.pkg.Header.StrigoiMetadata.PackageVersion,
		Author:      "Dynamic Package Loader",
		Description: m.Description(),
		References: []string{
			fmt.Sprintf("Package: %s v%s", 
				m.pkg.Header.ProtocolIdentity.Name,
				m.pkg.Header.ProtocolIdentity.Version),
		},
		Targets: []string{
			fmt.Sprintf("%s Protocol Implementations", m.pkg.Header.ProtocolIdentity.Name),
		},
	}
}

// Check performs a vulnerability check
func (m *DynamicModule) Check() bool {
	// Dynamic modules are always testable
	return true
}

// Run executes the dynamic module
func (m *DynamicModule) Run() (*core.ModuleResult, error) {
	result := &core.ModuleResult{
		Success:  true,
		Findings: []core.SecurityFinding{},
		Metadata: make(map[string]interface{}),
	}
	
	// This is where we'd implement dynamic test execution based on test vectors
	// For now, we'll create a simple finding
	finding := core.SecurityFinding{
		ID:          fmt.Sprintf("dynamic-%s", strings.ReplaceAll(m.definition.ModuleID, "/", "-")),
		Title:       fmt.Sprintf("Dynamic Test: %s", m.definition.ModuleID),
		Description: "This is a dynamically loaded test module",
		Severity:    m.mapRiskToSeverity(m.definition.RiskLevel),
		Evidence: []core.Evidence{
			{
				Type: "package",
				Data: map[string]interface{}{
					"module_id":   m.definition.ModuleID,
					"module_type": m.definition.ModuleType,
					"risk_level":  m.definition.RiskLevel,
					"vectors":     len(m.definition.TestVectors),
				},
				Description: "Dynamic module execution",
			},
		},
	}
	
	result.Findings = append(result.Findings, finding)
	result.Summary = m.summarizeFindings(result.Findings)
	
	return result, nil
}

// mapRiskToSeverity converts risk level to severity
func (m *DynamicModule) mapRiskToSeverity(risk string) core.Severity {
	switch strings.ToLower(risk) {
	case "critical":
		return core.Critical
	case "high":
		return core.High
	case "medium":
		return core.Medium
	case "low":
		return core.Low
	default:
		return core.Info
	}
}

// summarizeFindings creates a finding summary
func (m *DynamicModule) summarizeFindings(findings []core.SecurityFinding) *core.FindingSummary {
	summary := &core.FindingSummary{
		Total:    len(findings),
		ByModule: make(map[string]int),
	}
	
	for _, finding := range findings {
		switch finding.Severity {
		case core.Critical:
			summary.Critical++
		case core.High:
			summary.High++
		case core.Medium:
			summary.Medium++
		case core.Low:
			summary.Low++
		case core.Info:
			summary.Info++
		}
	}
	
	summary.ByModule[m.Name()] = len(findings)
	
	return summary
}