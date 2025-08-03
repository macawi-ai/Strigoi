package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/macawi-ai/strigoi/internal/core"
	"github.com/macawi-ai/strigoi/internal/registry"
)

// ModuleMigration represents a module to migrate
type ModuleMigration struct {
	OldPath     string
	ModuleType  core.ModuleType
	Category    string
	Name        string
	Description string
	Author      string
	Severity    registry.SeverityLevel
}

var modulesToMigrate = []ModuleMigration{
	// Attack Modules
	{
		OldPath:     "mcp/config/credential_storage",
		ModuleType:  core.ModuleTypeAttack,
		Category:    "credential",
		Name:        "Config Credential Scanner",
		Description: "Scans for plaintext credentials in MCP configuration files",
		Author:      "Strigoi Team",
		Severity:    registry.SeverityCritical,
	},
	{
		OldPath:     "mcp/privilege/yama_bypass_detection",
		ModuleType:  core.ModuleTypeAttack,
		Category:    "privilege",
		Name:        "YAMA Bypass Detection",
		Description: "Detects YAMA ptrace_scope configuration and bypass risks",
		Author:      "Strigoi Team",
		Severity:    registry.SeverityHigh,
	},
	{
		OldPath:     "mcp/session/header_hijack",
		ModuleType:  core.ModuleTypeAttack,
		Category:    "session",
		Name:        "Session Header Hijacking Scanner",
		Description: "Detects vulnerabilities in MCP session management via headers",
		Author:      "Strigoi Team",
		Severity:    registry.SeverityHigh,
	},
	{
		OldPath:     "mcp/validation/command_injection",
		ModuleType:  core.ModuleTypeAttack,
		Category:    "injection",
		Name:        "Command Injection Scanner",
		Description: "Detects command injection vulnerabilities in MCP tool execution",
		Author:      "Strigoi Team",
		Severity:    registry.SeverityCritical,
	},
	{
		OldPath:     "sudo/cache_detection",
		ModuleType:  core.ModuleTypeAttack,
		Category:    "credential",
		Name:        "Sudo Cache Detection",
		Description: "Detects sudo credential caching vulnerabilities that can be exploited by rogue MCPs",
		Author:      "Strigoi Team",
		Severity:    registry.SeverityCritical,
	},
	{
		OldPath:     "mcp/stdio/mitm_intercept",
		ModuleType:  core.ModuleTypeAttack,
		Category:    "transport",
		Name:        "STDIO MitM Detection",
		Description: "Detects vulnerabilities to STDIO interception and manipulation",
		Author:      "Strigoi Team",
		Severity:    registry.SeverityHigh,
	},
	// Scanner Modules
	{
		OldPath:     "scanners/sudo_mcp",
		ModuleType:  core.ModuleTypeScanner,
		Category:    "exploitation",
		Name:        "Sudo MCP Exploitation Scanner",
		Description: "Monitors for MCP processes attempting to exploit sudo cache",
		Author:      "Strigoi Team",
		Severity:    registry.SeverityCritical,
	},
}

func main() {
	var (
		dbPath  = flag.String("db", "", "Path to registry database")
		dryRun  = flag.Bool("dry-run", false, "Show what would be migrated without doing it")
		verbose = flag.Bool("v", false, "Verbose output")
	)
	
	flag.Parse()
	
	// Set default database path
	if *dbPath == "" {
		*dbPath = filepath.Join(".", "data", "registry", "strigoi.duckdb")
	}
	
	// Connect to registry
	reg, err := registry.NewRegistry(*dbPath)
	if err != nil {
		log.Fatal("Failed to connect to registry:", err)
	}
	defer reg.Close()
	
	ctx := context.Background()
	
	fmt.Println("🔄 Strigoi Module Migration")
	fmt.Println("===========================")
	fmt.Printf("Database: %s\n", *dbPath)
	fmt.Printf("Modules to migrate: %d\n", len(modulesToMigrate))
	
	if *dryRun {
		fmt.Println("\n⚠️  DRY RUN MODE - No changes will be made")
	}
	
	fmt.Println()
	
	// Migrate each module
	successCount := 0
	for i, mod := range modulesToMigrate {
		fmt.Printf("[%d/%d] Migrating: %s\n", i+1, len(modulesToMigrate), mod.OldPath)
		
		if *verbose {
			fmt.Printf("  Type: %s\n", mod.ModuleType)
			fmt.Printf("  Name: %s\n", mod.Name)
			fmt.Printf("  Category: %s\n", mod.Category)
		}
		
		if !*dryRun {
			entity := &registry.Entity{
				EntityType:  registry.EntityTypeMOD,
				Name:        mod.Name,
				Description: mod.Description,
				Status:      registry.StatusActive,
				Severity:    mod.Severity,
				Author:      mod.Author,
				Organization: "Macawi AI",
				Category:    mod.Category,
				Tags:        extractTags(mod),
				DiscoveryDate: &time.Time{},
				ImplementationDate: &time.Time{},
				Metadata: map[string]interface{}{
					"old_path":     mod.OldPath,
					"module_type":  string(mod.ModuleType),
					"migrated_from": "v0.3.0-alpha",
				},
			}
			
			// Set discovery date based on module
			discoveryDate := time.Date(2025, 7, 24, 0, 0, 0, 0, time.UTC)
			entity.DiscoveryDate = &discoveryDate
			
			// Implementation date is today
			implDate := time.Now()
			entity.ImplementationDate = &implDate
			
			// Register the entity
			registered, err := reg.RegisterEntity(ctx, entity)
			if err != nil {
				fmt.Printf("  ❌ Error: %v\n", err)
				continue
			}
			
			fmt.Printf("  ✅ Registered as: %s\n", registered.EntityID)
			successCount++
			
			// Add relationships if applicable
			if mod.OldPath == "sudo/cache_detection" {
				// This module relates to the sudo tailgating vulnerability
				// We'll add this relationship later when we migrate vulnerabilities
			}
		} else {
			fmt.Printf("  🔍 Would register as MOD entity\n")
			successCount++
		}
	}
	
	fmt.Printf("\n✨ Migration complete: %d/%d successful\n", successCount, len(modulesToMigrate))
	
	if !*dryRun {
		// Register the migration itself as an event
		migrationEntity := &registry.Entity{
			EntityType:  registry.EntityTypeRUN,
			BaseID:      fmt.Sprintf("RUN-%s", time.Now().Format("2006-0102-150405")),
			Version:     "v1.0.0",
			Name:        "Module Migration Run",
			Description: fmt.Sprintf("Migrated %d modules to entity registry", successCount),
			Status:      registry.StatusActive,
			Author:      "Migration Tool",
			Organization: "Macawi AI",
			Metadata: map[string]interface{}{
				"modules_migrated": successCount,
				"total_modules":    len(modulesToMigrate),
				"timestamp":        time.Now().Unix(),
			},
		}
		
		if _, err := reg.RegisterEntity(ctx, migrationEntity); err != nil {
			log.Printf("Failed to register migration run: %v", err)
		}
	}
}

// extractTags generates tags from module information
func extractTags(mod ModuleMigration) []string {
	tags := []string{
		string(mod.ModuleType),
		mod.Category,
	}
	
	// Add specific tags based on path
	if contains(mod.OldPath, "mcp") {
		tags = append(tags, "mcp")
	}
	if contains(mod.OldPath, "sudo") {
		tags = append(tags, "sudo", "privilege-escalation")
	}
	if contains(mod.OldPath, "injection") {
		tags = append(tags, "injection")
	}
	if contains(mod.OldPath, "credential") {
		tags = append(tags, "credential", "secrets")
	}
	if contains(mod.Name, "Scanner") {
		tags = append(tags, "scanner")
	}
	
	return tags
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || 
		   len(s) >= len(substr) && s[:len(substr)] == substr ||
		   len(s) > len(substr) && s[1:len(s)-1] == substr
}