package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/macawi-ai/strigoi/internal/registry"
)

// VulnerabilityMigration represents a vulnerability to migrate
type VulnerabilityMigration struct {
	Name         string
	Description  string
	Severity     registry.SeverityLevel
	CVE          string
	CVSS         float64
	Discovery    time.Time
	Author       string
	Tags         []string
	Related      []string // Related module paths
}

var vulnerabilities = []VulnerabilityMigration{
	{
		Name:        "Rogue MCP Sudo Tailgating",
		Description: "MCP processes can exploit sudo credential caching to gain root access without authentication",
		Severity:    registry.SeverityCritical,
		CVE:         "",
		CVSS:        9.8,
		Discovery:   time.Date(2025, 7, 26, 0, 0, 0, 0, time.UTC),
		Author:      "Sleep",
		Tags:        []string{"mcp", "sudo", "privilege-escalation", "credential-cache"},
		Related:     []string{"sudo/cache_detection", "scanners/sudo_mcp"},
	},
	{
		Name:        "MCP Same-User Catastrophe",
		Description: "All MCP servers running as the same user can access each other's resources and credentials",
		Severity:    registry.SeverityCritical,
		CVE:         "",
		CVSS:        8.8,
		Discovery:   time.Date(2025, 7, 24, 0, 0, 0, 0, time.UTC),
		Author:      "Strigoi Team",
		Tags:        []string{"mcp", "process-isolation", "credential-access"},
	},
	{
		Name:        "YAMA Bypass via Parent-Child",
		Description: "YAMA ptrace_scope protection can be bypassed through parent-child process relationships",
		Severity:    registry.SeverityHigh,
		CVE:         "",
		CVSS:        7.5,
		Discovery:   time.Date(2025, 7, 24, 0, 0, 0, 0, time.UTC),
		Author:      "Strigoi Team",
		Tags:        []string{"yama", "ptrace", "process-injection", "linux"},
		Related:     []string{"mcp/privilege/yama_bypass_detection"},
	},
	{
		Name:        "MCP Credential Triangle",
		Description: "Credentials exposed through process arguments, environment variables, and config files",
		Severity:    registry.SeverityCritical,
		CVE:         "",
		CVSS:        8.1,
		Discovery:   time.Date(2025, 7, 25, 0, 0, 0, 0, time.UTC),
		Author:      "Strigoi Team",
		Tags:        []string{"mcp", "credential-exposure", "secrets-management"},
		Related:     []string{"mcp/config/credential_storage"},
	},
	{
		Name:        "Command Injection via Tool Execution",
		Description: "Insufficient validation in MCP tool execution allows command injection attacks",
		Severity:    registry.SeverityCritical,
		CVE:         "",
		CVSS:        9.1,
		Discovery:   time.Date(2025, 7, 25, 0, 0, 0, 0, time.UTC),
		Author:      "Strigoi Team",
		Tags:        []string{"mcp", "command-injection", "input-validation"},
		Related:     []string{"mcp/validation/command_injection"},
	},
	{
		Name:        "STDIO MITM Attack",
		Description: "STDIO transport can be intercepted and manipulated by processes with file descriptor access",
		Severity:    registry.SeverityHigh,
		CVE:         "",
		CVSS:        7.3,
		Discovery:   time.Date(2025, 7, 25, 0, 0, 0, 0, time.UTC),
		Author:      "Strigoi Team",
		Tags:        []string{"mcp", "mitm", "stdio", "transport-security"},
		Related:     []string{"mcp/stdio/mitm_intercept"},
	},
	{
		Name:        "Session Header Hijacking",
		Description: "MCP session management vulnerable to header manipulation and hijacking",
		Severity:    registry.SeverityHigh,
		CVE:         "",
		CVSS:        7.8,
		Discovery:   time.Date(2025, 7, 25, 0, 0, 0, 0, time.UTC),
		Author:      "Strigoi Team",
		Tags:        []string{"mcp", "session-hijacking", "authentication"},
		Related:     []string{"mcp/session/header_hijack"},
	},
}

func main() {
	var (
		dbPath = flag.String("db", "", "Path to registry database")
		dryRun = flag.Bool("dry-run", false, "Show what would be migrated without doing it")
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
	
	fmt.Println("🔍 Strigoi Vulnerability Migration")
	fmt.Println("==================================")
	fmt.Printf("Database: %s\n", *dbPath)
	fmt.Printf("Vulnerabilities to migrate: %d\n", len(vulnerabilities))
	
	if *dryRun {
		fmt.Println("\n⚠️  DRY RUN MODE - No changes will be made")
	}
	
	fmt.Println()
	
	// Migrate each vulnerability
	successCount := 0
	for i, vuln := range vulnerabilities {
		fmt.Printf("[%d/%d] Migrating: %s\n", i+1, len(vulnerabilities), vuln.Name)
		
		if !*dryRun {
			entity := &registry.Entity{
				EntityType:  registry.EntityTypeVUL,
				Name:        vuln.Name,
				Description: vuln.Description,
				Status:      registry.StatusActive,
				Severity:    vuln.Severity,
				Author:      vuln.Author,
				Organization: "Macawi AI",
				Tags:        vuln.Tags,
				DiscoveryDate: &vuln.Discovery,
				AnalysisDate: &vuln.Discovery,
				Metadata: map[string]interface{}{
					"cvss_score": vuln.CVSS,
					"affected_components": "MCP implementations",
					"related_modules": vuln.Related,
				},
			}
			
			// Register the entity
			registered, err := reg.RegisterEntity(ctx, entity)
			if err != nil {
				fmt.Printf("  ❌ Error: %v\n", err)
				continue
			}
			
			fmt.Printf("  ✅ Registered as: %s\n", registered.EntityID)
			successCount++
			
			// Add vulnerability-specific attributes
			err = reg.AddVulnerabilityAttributes(ctx, registered.EntityID, vuln.CVSS, "Low")
			if err != nil {
				log.Printf("Failed to add vulnerability attributes: %v", err)
			}
		} else {
			fmt.Printf("  🔍 Would register as VUL entity\n")
			successCount++
		}
	}
	
	fmt.Printf("\n✨ Migration complete: %d/%d successful\n", successCount, len(vulnerabilities))
}