package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	
	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	var (
		dbPath = flag.String("db", "", "Path to registry database")
		query  = flag.String("query", "all", "Query type: all, modules, vulns, stats")
	)
	
	flag.Parse()
	
	// Set default database path
	if *dbPath == "" {
		*dbPath = filepath.Join(".", "data", "registry", "strigoi.duckdb")
	}
	
	// Connect to database
	db, err := sql.Open("duckdb", *dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()
	
	ctx := context.Background()
	
	switch *query {
	case "all":
		showAllEntities(ctx, db)
	case "modules":
		showModules(ctx, db)
	case "vulns":
		showVulnerabilities(ctx, db)
	case "stats":
		showStats(ctx, db)
	default:
		log.Fatal("Unknown query type:", *query)
	}
}

func showAllEntities(ctx context.Context, db *sql.DB) {
	fmt.Println("\n📊 STRIGOI ENTITY REGISTRY")
	fmt.Println("==========================")
	
	rows, err := db.QueryContext(ctx, `
		SELECT 
			entity_id,
			entity_type,
			name,
			status,
			severity,
			author,
			created_at
		FROM entities
		ORDER BY entity_type, base_id
	`)
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows.Close()
	
	fmt.Printf("\n%-20s %-4s %-40s %-10s %-10s %-15s\n", 
		"ID", "Type", "Name", "Status", "Severity", "Author")
	fmt.Println(strings.Repeat("-", 105))
	
	for rows.Next() {
		var (
			id, entityType, name, status string
			severity, author sql.NullString
			createdAt sql.NullTime
		)
		
		err := rows.Scan(&id, &entityType, &name, &status, &severity, &author, &createdAt)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		
		severityStr := ""
		if severity.Valid {
			severityStr = severity.String
		}
		
		authorStr := "Unknown"
		if author.Valid {
			authorStr = author.String
		}
		
		fmt.Printf("%-20s %-4s %-40s %-10s %-10s %-15s\n",
			id, entityType, truncate(name, 40), status, severityStr, authorStr)
	}
}

func showModules(ctx context.Context, db *sql.DB) {
	fmt.Println("\n🔧 MODULES")
	fmt.Println("==========")
	
	rows, err := db.QueryContext(ctx, `
		SELECT 
			entity_id,
			name,
			description,
			status,
			author
		FROM entities
		WHERE entity_type = 'MOD'
		ORDER BY base_id
	`)
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var id, name, status string
		var description, author sql.NullString
		
		err := rows.Scan(&id, &name, &description, &author, &status)
		if err != nil {
			continue
		}
		
		fmt.Printf("\n[%s] %s\n", id, name)
		if description.Valid {
			fmt.Printf("  Description: %s\n", description.String)
		}
		fmt.Printf("  Status: %s | Author: %s\n", status, author.String)
	}
}

func showVulnerabilities(ctx context.Context, db *sql.DB) {
	fmt.Println("\n🐛 VULNERABILITIES")
	fmt.Println("==================")
	
	rows, err := db.QueryContext(ctx, `
		SELECT 
			e.entity_id,
			e.name,
			e.severity,
			va.cvss_score,
			e.discovery_date
		FROM entities e
		LEFT JOIN vulnerability_attributes va ON e.entity_id = va.entity_id
		WHERE e.entity_type = 'VUL'
		ORDER BY va.cvss_score DESC NULLS LAST
	`)
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var id, name string
		var severity sql.NullString
		var cvss sql.NullFloat64
		var discovery sql.NullTime
		
		err := rows.Scan(&id, &name, &severity, &cvss, &discovery)
		if err != nil {
			continue
		}
		
		fmt.Printf("\n[%s] %s\n", id, name)
		
		if severity.Valid {
			fmt.Printf("  Severity: %s", severity.String)
		}
		if cvss.Valid {
			fmt.Printf(" | CVSS: %.1f", cvss.Float64)
		}
		if discovery.Valid {
			fmt.Printf(" | Discovered: %s", discovery.Time.Format("2006-01-02"))
		}
		fmt.Println()
	}
}

func showStats(ctx context.Context, db *sql.DB) {
	fmt.Println("\n📈 REGISTRY STATISTICS")
	fmt.Println("======================")
	
	// Count by type
	rows, err := db.QueryContext(ctx, `
		SELECT entity_type, COUNT(*) as count
		FROM entities
		GROUP BY entity_type
		ORDER BY count DESC
	`)
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows.Close()
	
	fmt.Println("\nEntities by Type:")
	for rows.Next() {
		var entityType string
		var count int
		rows.Scan(&entityType, &count)
		fmt.Printf("  %-4s: %d\n", entityType, count)
	}
	
	// Count by status
	rows2, err := db.QueryContext(ctx, `
		SELECT status, COUNT(*) as count
		FROM entities
		GROUP BY status
		ORDER BY count DESC
	`)
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows2.Close()
	
	fmt.Println("\nEntities by Status:")
	for rows2.Next() {
		var status string
		var count int
		rows2.Scan(&status, &count)
		fmt.Printf("  %-12s: %d\n", status, count)
	}
	
	// Severity distribution for vulnerabilities
	rows3, err := db.QueryContext(ctx, `
		SELECT severity, COUNT(*) as count
		FROM entities
		WHERE entity_type = 'VUL' AND severity IS NOT NULL
		GROUP BY severity
		ORDER BY 
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				WHEN 'low' THEN 4
				WHEN 'info' THEN 5
			END
	`)
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows3.Close()
	
	fmt.Println("\nVulnerability Severity:")
	for rows3.Next() {
		var severity string
		var count int
		rows3.Scan(&severity, &count)
		fmt.Printf("  %-8s: %d\n", severity, count)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}