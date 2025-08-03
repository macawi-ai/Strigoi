package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/macawi-ai/strigoi/internal/registry"
)

func main() {
	var (
		dbPath     = flag.String("db", "", "Path to registry database (default: ./data/registry/strigoi.duckdb)")
		initDB     = flag.Bool("init", false, "Initialize new database")
		listCmd    = flag.Bool("list", false, "List all entities")
		searchTerm = flag.String("search", "", "Search entities")
		entityType = flag.String("type", "", "Filter by entity type")
	)
	
	flag.Parse()
	
	// Set default database path
	if *dbPath == "" {
		*dbPath = filepath.Join(".", "data", "registry", "strigoi.duckdb")
	}
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		log.Fatal("Failed to create directory:", err)
	}
	
	// Initialize database if requested
	if *initDB {
		if err := registry.InitializeDatabase(*dbPath); err != nil {
			log.Fatal("Failed to initialize database:", err)
		}
		fmt.Println("✓ Database initialized successfully")
		return
	}
	
	// Connect to registry
	reg, err := registry.NewRegistry(*dbPath)
	if err != nil {
		log.Fatal("Failed to connect to registry:", err)
	}
	defer reg.Close()
	
	ctx := context.Background()
	
	// Handle commands
	if *listCmd {
		listEntities(ctx, reg, *entityType)
	} else if *searchTerm != "" {
		searchEntities(ctx, reg, *searchTerm)
	} else {
		showStats(ctx, reg)
	}
}

func listEntities(ctx context.Context, reg *registry.Registry, filterType string) {
	// TODO: Implement list functionality
	fmt.Println("Entity listing:")
	fmt.Println("  [MOD-2025-10001-v1.0.0] Sudo Cache Detection")
	fmt.Println("  [MOD-2025-20001-v1.0.0] Sudo MCP Exploitation Scanner")
	fmt.Println("  [VUL-2025-00001-v1.0.0] Rogue MCP Sudo Tailgating")
	// This will be implemented to query the actual database
}

func searchEntities(ctx context.Context, reg *registry.Registry, term string) {
	fmt.Printf("Searching for: %s\n", term)
	
	entities, err := reg.SearchEntities(ctx, term, nil)
	if err != nil {
		log.Printf("Search error: %v", err)
		return
	}
	
	fmt.Printf("\nFound %d entities:\n", len(entities))
	for _, e := range entities {
		fmt.Printf("  [%s] %s - %s\n", e.EntityID, e.Name, e.Description)
	}
}

func showStats(ctx context.Context, reg *registry.Registry) {
	fmt.Println("Strigoi Entity Registry Statistics")
	fmt.Println("==================================")
	
	// TODO: Query actual stats from database
	fmt.Println("Total Entities: 3")
	fmt.Println("  Modules:       2")
	fmt.Println("  Vulnerabilities: 1")
	fmt.Println("  Policies:      1")
	fmt.Println("  Configurations: 1")
	
	fmt.Println("\nUse -list to show all entities")
	fmt.Println("Use -search <term> to search")
}