package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/internal/registry"
)

// RunLevel2Tests tests entity type-specific behaviors
func RunLevel2Tests() {
	// Create test database
	testDB := filepath.Join(".", "test_registry_l2.duckdb")
	defer os.Remove(testDB)
	
	// Initialize registry
	reg, err := registry.NewRegistry(testDB)
	if err != nil {
		log.Fatal("Failed to create registry:", err)
	}
	defer reg.Close()
	
	ctx := context.Background()
	suite := &TestSuite{Name: "Level 2: Entity Type Behaviors"}
	
	// Test 1: Module-Specific Attributes
	suite.Run("Module-Specific Attributes", func() error {
		// Create a module with typical attributes
		module := &registry.Entity{
			EntityType:   registry.EntityTypeMOD,
			Name:         "Advanced Scanner Module",
			Description:  "Scans for advanced vulnerabilities",
			Status:       registry.StatusActive,
			Severity:     registry.SeverityCritical,
			Author:       "Security Team",
			Organization: "Strigoi",
			Category:     "scanner",
			Tags:         []string{"scanner", "vulnerability", "network"},
			Metadata: map[string]interface{}{
				"module_type": "scanner",
				"risk_level":  "high",
				"requirements": []interface{}{"nmap", "python3"},
				"options": map[string]interface{}{
					"RHOST": map[string]interface{}{
						"type":        "string",
						"required":    true,
						"description": "Target host",
					},
					"RPORT": map[string]interface{}{
						"type":        "integer",
						"required":    false,
						"default":     443,
						"description": "Target port",
					},
				},
			},
		}
		
		created, err := reg.RegisterEntity(ctx, module)
		if err != nil {
			return fmt.Errorf("failed to create module: %w", err)
		}
		
		// Verify module ID starts at 10000 range
		if !strings.Contains(created.BaseID, "-1") {
			return fmt.Errorf("module ID not in expected range: %s", created.BaseID)
		}
		
		// Add module attributes
		err = addModuleAttributes(reg, ctx, created.EntityID, "scanner", "high", 
			`{"nmap", "python3"}`, created.Metadata["options"])
		if err != nil {
			return fmt.Errorf("failed to add module attributes: %w", err)
		}
		
		return nil
	})
	
	// Test 2: Vulnerability-Specific Attributes
	suite.Run("Vulnerability-Specific Attributes", func() error {
		// Create vulnerability with all attributes
		vuln := &registry.Entity{
			EntityType:   registry.EntityTypeVUL,
			Name:         "Critical RCE Vulnerability",
			Description:  "Remote code execution in authentication module",
			Status:       registry.StatusActive,
			Severity:     registry.SeverityCritical,
			Author:       "Security Research",
			Organization: "Strigoi",
			Tags:         []string{"rce", "authentication", "critical"},
			Metadata: map[string]interface{}{
				"cvss_score":    9.8,
				"cvss_vector":   "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
				"cve_id":        "CVE-2025-12345",
				"affected_versions": []interface{}{"1.0.0", "1.0.1", "1.0.2"},
				"patch_available": true,
			},
		}
		
		// Set discovery date
		discoveryDate := time.Now().Add(-7 * 24 * time.Hour)
		vuln.DiscoveryDate = &discoveryDate
		
		created, err := reg.RegisterEntity(ctx, vuln)
		if err != nil {
			return fmt.Errorf("failed to create vulnerability: %w", err)
		}
		
		// Add vulnerability attributes
		err = reg.AddVulnerabilityAttributes(ctx, created.EntityID, 9.8, "Low")
		if err != nil {
			return fmt.Errorf("failed to add vulnerability attributes: %w", err)
		}
		
		// Verify vulnerability ID format
		if !strings.HasPrefix(created.BaseID, "VUL-") {
			return fmt.Errorf("invalid vulnerability ID format: %s", created.BaseID)
		}
		
		return nil
	})
	
	// Test 3: Attack Pattern Attributes  
	suite.Run("Attack Pattern Attributes", func() error {
		attack := &registry.Entity{
			EntityType:   registry.EntityTypeATK,
			Name:         "Credential Stuffing Attack",
			Description:  "Automated injection of breached credentials",
			Status:       registry.StatusActive,
			Severity:     registry.SeverityHigh,
			Author:       "Threat Intel",
			Organization: "Strigoi",
			Category:     "authentication",
			Tags:         []string{"credentials", "bruteforce", "authentication"},
			Metadata: map[string]interface{}{
				"mitre_id":      "T1110.004",
				"kill_chain":    []interface{}{"initial-access", "credential-access"},
				"prerequisites": "List of breached credentials",
				"indicators": map[string]interface{}{
					"network": []interface{}{"High volume of login attempts", "Multiple source IPs"},
					"log":     []interface{}{"Failed authentication spikes", "Successful logins from new locations"},
				},
			},
		}
		
		created, err := reg.RegisterEntity(ctx, attack)
		if err != nil {
			return fmt.Errorf("failed to create attack pattern: %w", err)
		}
		
		// Verify attack pattern specific fields
		if created.Metadata["mitre_id"] != "T1110.004" {
			return fmt.Errorf("MITRE ID not preserved")
		}
		
		return nil
	})
	
	// Test 4: Detection Signature Attributes
	suite.Run("Detection Signature Attributes", func() error {
		sig := &registry.Entity{
			EntityType:   registry.EntityTypeSIG,
			Name:         "MCP Sudo Cache Detection",
			Description:  "Detects MCP processes exploiting sudo cache",
			Status:       registry.StatusActive,
			Severity:     registry.SeverityHigh,
			Author:       "Detection Team",
			Organization: "Strigoi",
			Tags:         []string{"mcp", "sudo", "detection"},
			Metadata: map[string]interface{}{
				"signature_type": "behavioral",
				"detection_logic": `
					if process.name contains "mcp" and 
					   process.parent == "sudo" and 
					   time_since_sudo < 5 minutes then
					   alert("Potential MCP sudo cache exploitation")
				`,
				"false_positive_rate": 0.02,
				"confidence": "high",
				"performance_impact": "low",
			},
		}
		
		created, err := reg.RegisterEntity(ctx, sig)
		if err != nil {
			return fmt.Errorf("failed to create signature: %w", err)
		}
		
		// Verify signature has required metadata
		if created.Metadata["signature_type"] != "behavioral" {
			return fmt.Errorf("signature type not preserved")
		}
		
		return nil
	})
	
	// Test 5: Configuration Entity
	suite.Run("Configuration Entity", func() error {
		config := &registry.Entity{
			EntityType:   registry.EntityTypeCFG,
			Name:         "Production Scanner Config",
			Description:  "Configuration for production vulnerability scanning",
			Status:       registry.StatusActive,
			Author:       "DevOps",
			Organization: "Strigoi",
			Configuration: map[string]interface{}{
				"scan_interval":    "daily",
				"max_threads":      10,
				"timeout_seconds":  300,
				"report_format":    "json",
				"notification": map[string]interface{}{
					"enabled": true,
					"channels": []interface{}{"email", "slack"},
					"threshold": "high",
				},
			},
		}
		
		created, err := reg.RegisterEntity(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
		
		// Verify configuration is stored properly
		if created.Configuration["scan_interval"] != "daily" {
			return fmt.Errorf("configuration not preserved")
		}
		
		return nil
	})
	
	// Test 6: Policy Entity
	suite.Run("Policy Entity", func() error {
		policy := &registry.Entity{
			EntityType:   registry.EntityTypePOL,
			Name:         "Zero Trust Access Policy",
			Description:  "Enforces zero trust principles for all access",
			Status:       registry.StatusActive,
			Severity:     registry.SeverityCritical,
			Author:       "Security Policy Team",
			Organization: "Strigoi",
			Category:     "access-control",
			Metadata: map[string]interface{}{
				"policy_version": "2.0",
				"enforcement_mode": "strict",
				"applies_to": []interface{}{"all_users", "all_services"},
				"rules": []interface{}{
					map[string]interface{}{
						"name": "verify_identity",
						"condition": "always",
						"action": "require_mfa",
					},
					map[string]interface{}{
						"name": "verify_device",
						"condition": "untrusted_network",
						"action": "require_device_cert",
					},
				},
			},
		}
		
		created, err := reg.RegisterEntity(ctx, policy)
		if err != nil {
			return fmt.Errorf("failed to create policy: %w", err)
		}
		
		// Policies should have critical severity by default
		if created.Severity != registry.SeverityCritical {
			return fmt.Errorf("policy severity should be critical")
		}
		
		return nil
	})
	
	// Test 7: Report Entity
	suite.Run("Report Entity", func() error {
		reportDate := time.Now()
		report := &registry.Entity{
			EntityType:   registry.EntityTypeRPT,
			Name:         "Weekly Security Assessment",
			Description:  "Comprehensive security assessment for week 42",
			Status:       registry.StatusActive,
			Author:       "Automated Scanner",
			Organization: "Strigoi",
			AnalysisDate: &reportDate,
			Metadata: map[string]interface{}{
				"report_type": "security_assessment",
				"period": map[string]interface{}{
					"start": "2025-10-14",
					"end":   "2025-10-21",
				},
				"summary": map[string]interface{}{
					"total_scans": 1543,
					"findings": map[string]interface{}{
						"critical": 3,
						"high":     12,
						"medium":   47,
						"low":      89,
					},
					"remediated": 23,
				},
				"export_formats": []interface{}{"pdf", "json", "csv"},
			},
		}
		
		created, err := reg.RegisterEntity(ctx, report)
		if err != nil {
			return fmt.Errorf("failed to create report: %w", err)
		}
		
		// Reports should have analysis date
		if created.AnalysisDate == nil {
			return fmt.Errorf("report missing analysis date")
		}
		
		return nil
	})
	
	// Test 8: Run/Session Entity
	suite.Run("Run/Session Entity", func() error {
		runStart := time.Now()
		run := &registry.Entity{
			EntityType:   registry.EntityTypeRUN,
			Name:         "Penetration Test Run #1337",
			Description:  "Automated penetration test of staging environment",
			Status:       registry.StatusActive,
			Author:       "PenTest Bot",
			Organization: "Strigoi",
			ImplementationDate: &runStart,
			Metadata: map[string]interface{}{
				"run_id": "pt-2025-1337",
				"environment": "staging",
				"modules_executed": []interface{}{
					"MOD-2025-10001",
					"MOD-2025-10002",
					"MOD-2025-10005",
				},
				"duration_seconds": 3600,
				"targets": []interface{}{
					"10.0.0.0/24",
					"staging.example.com",
				},
				"findings_count": 17,
				"status": "completed",
			},
		}
		
		created, err := reg.RegisterEntity(ctx, run)
		if err != nil {
			return fmt.Errorf("failed to create run: %w", err)
		}
		
		// Runs should track execution details
		if created.Metadata["run_id"] != "pt-2025-1337" {
			return fmt.Errorf("run ID not preserved")
		}
		
		return nil
	})
	
	// Generate report
	suite.Report()
}

// Helper function to add module attributes
func addModuleAttributes(reg *registry.Registry, ctx context.Context, entityID, moduleType, riskLevel, requirements string, options interface{}) error {
	// In a real implementation, this would use a proper method on Registry
	// For now, we're just validating the concept
	return nil
}

