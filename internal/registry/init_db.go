package registry

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"
)

//go:embed schema_duckdb.sql
var schemaFS embed.FS

// InitializeDatabase creates and initializes the entity registry database
func InitializeDatabase(dbPath string) error {
	// Connect to DuckDB
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	
	// Read schema
	schemaBytes, err := schemaFS.ReadFile("schema_duckdb.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}
	
	// Split schema into individual statements
	// DuckDB requires executing statements one at a time
	statements := splitSQLStatements(string(schemaBytes))
	
	// Execute each statement
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		
		_, err := db.Exec(stmt)
		if err != nil {
			return fmt.Errorf("failed to execute statement %d: %w\nStatement: %s", i+1, err, stmt)
		}
	}
	
	// Insert initial data
	if err := insertInitialData(db); err != nil {
		return fmt.Errorf("failed to insert initial data: %w", err)
	}
	
	fmt.Printf("Database initialized successfully at: %s\n", dbPath)
	return nil
}

// splitSQLStatements splits SQL script into individual statements
func splitSQLStatements(sql string) []string {
	// Simple splitter - in production would need more sophisticated parsing
	statements := strings.Split(sql, ";")
	var result []string
	
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}
	
	return result
}

// insertInitialData adds foundational entities
func insertInitialData(db *sql.DB) error {
	// Insert root policy entity
	_, err := db.Exec(`
		INSERT INTO entities (
			entity_id, entity_type, base_id, version,
			name, description, status, severity,
			author, organization
		) VALUES (
			'POL-2025-00001-v1.0.0', 'POL', 'POL-2025-00001', 'v1.0.0',
			'Strigoi Ethical Framework',
			'Core ethical policy: We protect, never exploit. All discoveries become defensive tools.',
			'active', 'critical',
			'Strigoi Team', 'Macawi AI'
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to insert root policy: %w", err)
	}
	
	// Insert framework configuration
	_, err = db.Exec(`
		INSERT INTO entities (
			entity_id, entity_type, base_id, version,
			name, description, status,
			author, organization
		) VALUES (
			'CFG-2025-00001-v1.0.0', 'CFG', 'CFG-2025-00001', 'v1.0.0',
			'Strigoi Base Configuration',
			'Default configuration for Strigoi framework with entity registry support',
			'active',
			'Strigoi Team', 'Macawi AI'
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to insert base config: %w", err)
	}
	
	// Create initial changelog entries
	_, err = db.Exec(`
		INSERT INTO changelog (
			entity_id, change_version, change_author,
			change_type, change_summary
		) VALUES 
		('POL-2025-00001-v1.0.0', 'v1.0.0', 'System', 'created', 'Initial ethical framework policy'),
		('CFG-2025-00001-v1.0.0', 'v1.0.0', 'System', 'created', 'Initial framework configuration')
	`)
	if err != nil {
		return fmt.Errorf("failed to insert changelog: %w", err)
	}
	
	return nil
}