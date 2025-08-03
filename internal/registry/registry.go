package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

// EntityType represents the type of entity in the registry
type EntityType string

const (
	EntityTypeMOD EntityType = "MOD" // Modules
	EntityTypeVUL EntityType = "VUL" // Vulnerabilities
	EntityTypeATK EntityType = "ATK" // Attack Patterns
	EntityTypeSIG EntityType = "SIG" // Detection Signatures
	EntityTypeDEM EntityType = "DEM" // Demonstrators/PoCs
	EntityTypeCFG EntityType = "CFG" // Configurations
	EntityTypePOL EntityType = "POL" // Policies
	EntityTypeRPT EntityType = "RPT" // Reports
	EntityTypeSES EntityType = "SES" // Sessions
	EntityTypeRUN EntityType = "RUN" // Test Runs
	EntityTypeMAN EntityType = "MAN" // Manifolds
	EntityTypePRO EntityType = "PRO" // Protocols
	EntityTypeSCN EntityType = "SCN" // Scenarios
	EntityTypeTOP EntityType = "TOP" // Topologies
	EntityTypeEVD EntityType = "EVD" // Evidence
	EntityTypeBLU EntityType = "BLU" // Blueprints
	EntityTypeTEL EntityType = "TEL" // Telemetry
	EntityTypeVAL EntityType = "VAL" // Validation Cycles
	EntityTypeNBK EntityType = "NBK" // Notebooks
	EntityTypePKG EntityType = "PKG" // Packages
)

// EntityStatus represents the lifecycle status of an entity
type EntityStatus string

const (
	StatusDraft      EntityStatus = "draft"
	StatusActive     EntityStatus = "active"
	StatusTesting    EntityStatus = "testing"
	StatusDeprecated EntityStatus = "deprecated"
	StatusArchived   EntityStatus = "archived"
	StatusRevoked    EntityStatus = "revoked"
)

// SeverityLevel represents risk/severity classifications
type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "critical"
	SeverityHigh     SeverityLevel = "high"
	SeverityMedium   SeverityLevel = "medium"
	SeverityLow      SeverityLevel = "low"
	SeverityInfo     SeverityLevel = "info"
)

// Entity represents a tracked entity in the registry
type Entity struct {
	// Identity
	EntityID    string     `json:"entity_id"`    // e.g., MOD-2025-10001-v1.0.0
	EntityType  EntityType `json:"entity_type"`
	BaseID      string     `json:"base_id"`      // e.g., MOD-2025-10001
	Version     string     `json:"version"`      // e.g., v1.0.0
	
	// Basic metadata
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Status      EntityStatus   `json:"status"`
	Severity    SeverityLevel  `json:"severity,omitempty"`
	
	// Timestamps
	DiscoveryDate       *time.Time `json:"discovery_date,omitempty"`
	AnalysisDate        *time.Time `json:"analysis_date,omitempty"`
	ImplementationDate  *time.Time `json:"implementation_date,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	ArchivedAt          *time.Time `json:"archived_at,omitempty"`
	
	// Attribution
	Author       string `json:"author"`
	Organization string `json:"organization"`
	
	// Categorization
	Category string   `json:"category,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	
	// Technical details
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
}

// VersionHistory tracks changes between versions
type VersionHistory struct {
	ID                int       `json:"id"`
	EntityID          string    `json:"entity_id"`
	VersionFrom       string    `json:"version_from"`
	VersionTo         string    `json:"version_to"`
	ChangeType        string    `json:"change_type"`
	ChangeDescription string    `json:"change_description"`
	ChangedBy         string    `json:"changed_by"`
	ChangedAt         time.Time `json:"changed_at"`
	RollbackInfo      map[string]interface{} `json:"rollback_info,omitempty"`
}

// EntityRelationship represents connections between entities
type EntityRelationship struct {
	ID                   int                    `json:"id"`
	SourceEntityID       string                 `json:"source_entity_id"`
	TargetEntityID       string                 `json:"target_entity_id"`
	RelationshipType     string                 `json:"relationship_type"`
	RelationshipMetadata map[string]interface{} `json:"relationship_metadata,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
}

// ChangelogEntry tracks granular changes
type ChangelogEntry struct {
	ID            int                    `json:"id"`
	EntityID      string                 `json:"entity_id"`
	ChangeVersion string                 `json:"change_version"`
	ChangeDate    time.Time              `json:"change_date"`
	ChangeAuthor  string                 `json:"change_author"`
	ChangeType    string                 `json:"change_type"`
	ChangeSummary string                 `json:"change_summary"`
	ChangeDetails map[string]interface{} `json:"change_details,omitempty"`
	CommitHash    string                 `json:"commit_hash,omitempty"`
}

// Registry manages the entity database
type Registry struct {
	db     *sql.DB
	dbPath string
}

// NewRegistry creates a new registry instance
func NewRegistry(dbPath string) (*Registry, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	registry := &Registry{
		db:     db,
		dbPath: dbPath,
	}
	
	// Initialize schema if needed
	if err := registry.initializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	
	return registry, nil
}

// Close closes the database connection
func (r *Registry) Close() error {
	return r.db.Close()
}

// initializeSchema creates tables if they don't exist
func (r *Registry) initializeSchema() error {
	// Read schema from embedded file or execute simplified version
	// For now, we'll check if the main table exists
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'entities'
		)
	`).Scan(&exists)
	
	if err != nil || !exists {
		// Execute schema creation
		// In production, this would read from schema.sql
		return r.createSchema()
	}
	
	return nil
}

// RegisterEntity creates a new entity with automatic ID generation
func (r *Registry) RegisterEntity(ctx context.Context, entity *Entity) (*Entity, error) {
	// Generate ID if not provided
	if entity.BaseID == "" {
		baseID, err := r.generateEntityID(ctx, entity.EntityType)
		if err != nil {
			return nil, err
		}
		entity.BaseID = baseID
	}
	
	// Set version if not provided
	if entity.Version == "" {
		entity.Version = "v1.0.0"
	}
	
	// Construct full entity ID
	entity.EntityID = fmt.Sprintf("%s-%s", entity.BaseID, entity.Version)
	
	// Set timestamps
	now := time.Now()
	entity.CreatedAt = now
	entity.UpdatedAt = now
	
	// Insert entity
	metadataJSON, _ := json.Marshal(entity.Metadata)
	configJSON, _ := json.Marshal(entity.Configuration)
	tagsArray := "{" + strings.Join(entity.Tags, ",") + "}"
	
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO entities (
			entity_id, entity_type, base_id, version,
			name, description, status, severity,
			discovery_date, analysis_date, implementation_date,
			created_at, updated_at,
			author, organization,
			category, tags,
			metadata, configuration
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11,
			$12, $13,
			$14, $15,
			$16, $17,
			$18, $19
		)`,
		entity.EntityID, entity.EntityType, entity.BaseID, entity.Version,
		entity.Name, entity.Description, entity.Status, entity.Severity,
		entity.DiscoveryDate, entity.AnalysisDate, entity.ImplementationDate,
		entity.CreatedAt, entity.UpdatedAt,
		entity.Author, entity.Organization,
		entity.Category, tagsArray,
		string(metadataJSON), string(configJSON),
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to register entity: %w", err)
	}
	
	// Log the registration
	r.logChange(ctx, entity.EntityID, entity.Version, "created", 
		fmt.Sprintf("Registered new entity: %s", entity.Name), entity.Author)
	
	return entity, nil
}

// GetEntity retrieves an entity by ID
func (r *Registry) GetEntity(ctx context.Context, entityID string) (*Entity, error) {
	var entity Entity
	var metadataJSON, configJSON interface{} // DuckDB returns JSON as interface{}
	var tagsArray sql.NullString
	var description, category, author, organization sql.NullString
	var severity sql.NullString
	
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			entity_id, entity_type, base_id, version,
			name, description, status, severity,
			discovery_date, analysis_date, implementation_date,
			created_at, updated_at, archived_at,
			author, organization,
			category, tags,
			metadata, configuration
		FROM entities
		WHERE entity_id = $1
	`, entityID).Scan(
		&entity.EntityID, &entity.EntityType, &entity.BaseID, &entity.Version,
		&entity.Name, &description, &entity.Status, &severity,
		&entity.DiscoveryDate, &entity.AnalysisDate, &entity.ImplementationDate,
		&entity.CreatedAt, &entity.UpdatedAt, &entity.ArchivedAt,
		&author, &organization,
		&category, &tagsArray,
		&metadataJSON, &configJSON,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}
	
	// Handle nullable strings
	if description.Valid {
		entity.Description = description.String
	}
	if category.Valid {
		entity.Category = category.String
	}
	if author.Valid {
		entity.Author = author.String
	}
	if organization.Valid {
		entity.Organization = organization.String
	}
	if severity.Valid {
		entity.Severity = SeverityLevel(severity.String)
	}
	
	// Handle JSON fields - DuckDB returns them as map[string]interface{}
	if metadataJSON != nil {
		switch v := metadataJSON.(type) {
		case map[string]interface{}:
			entity.Metadata = v
		case string:
			if v != "" {
				json.Unmarshal([]byte(v), &entity.Metadata)
			}
		}
	}
	
	if configJSON != nil {
		switch v := configJSON.(type) {
		case map[string]interface{}:
			entity.Configuration = v
		case string:
			if v != "" {
				json.Unmarshal([]byte(v), &entity.Configuration)
			}
		}
	}
	
	// Parse tags array
	if tagsArray.Valid && tagsArray.String != "" && tagsArray.String != "{}" {
		tagStr := strings.Trim(tagsArray.String, "{}")
		if tagStr != "" {
			entity.Tags = strings.Split(tagStr, ",")
		}
	}
	
	return &entity, nil
}

// UpdateEntity updates an existing entity and tracks version history
func (r *Registry) UpdateEntity(ctx context.Context, entity *Entity, changeType, changeDescription, changedBy string) error {
	// Get current version
	current, err := r.GetEntity(ctx, entity.EntityID)
	if err != nil {
		return err
	}
	
	// Update timestamp
	entity.UpdatedAt = time.Now()
	
	// Update entity
	metadataJSON, _ := json.Marshal(entity.Metadata)
	configJSON, _ := json.Marshal(entity.Configuration)
	tagsArray := "{" + strings.Join(entity.Tags, ",") + "}"
	
	_, err = r.db.ExecContext(ctx, `
		UPDATE entities SET
			name = $2, description = $3, status = $4, severity = $5,
			discovery_date = $6, analysis_date = $7, implementation_date = $8,
			updated_at = $9, archived_at = $10,
			author = $11, organization = $12,
			category = $13, tags = $14,
			metadata = $15, configuration = $16
		WHERE entity_id = $1`,
		entity.EntityID,
		entity.Name, entity.Description, entity.Status, entity.Severity,
		entity.DiscoveryDate, entity.AnalysisDate, entity.ImplementationDate,
		entity.UpdatedAt, entity.ArchivedAt,
		entity.Author, entity.Organization,
		entity.Category, tagsArray,
		string(metadataJSON), string(configJSON),
	)
	
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}
	
	// Record version history if version changed
	if current.Version != entity.Version {
		_, err = r.db.ExecContext(ctx, `
			INSERT INTO version_history (
				entity_id, version_from, version_to,
				change_type, change_description, changed_by
			) VALUES ($1, $2, $3, $4, $5, $6)`,
			entity.EntityID, current.Version, entity.Version,
			changeType, changeDescription, changedBy,
		)
		if err != nil {
			return fmt.Errorf("failed to record version history: %w", err)
		}
	}
	
	// Log the change
	r.logChange(ctx, entity.EntityID, entity.Version, changeType, changeDescription, changedBy)
	
	return nil
}

// generateEntityID generates the next ID for an entity type
func (r *Registry) generateEntityID(ctx context.Context, entityType EntityType) (string, error) {
	year := time.Now().Year()
	
	// Query for the highest existing number
	var maxNum sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT MAX(CAST(SUBSTRING(base_id, LENGTH(base_id) - 4, 5) AS INTEGER))
		FROM entities
		WHERE entity_type = $1
		AND base_id LIKE $2`,
		entityType,
		fmt.Sprintf("%s-%d-%%", entityType, year),
	).Scan(&maxNum)
	
	if err != nil {
		return "", err
	}
	
	// Determine starting number based on type
	var startNum int64
	switch entityType {
	case EntityTypeMOD:
		startNum = 10000
	default:
		startNum = 1
	}
	
	nextNum := startNum
	if maxNum.Valid && maxNum.Int64 >= startNum {
		nextNum = maxNum.Int64 + 1
	}
	
	return fmt.Sprintf("%s-%d-%05d", entityType, year, nextNum), nil
}

// logChange records a change in the changelog
func (r *Registry) logChange(ctx context.Context, entityID, version, changeType, summary, author string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO changelog (
			entity_id, change_version, change_author,
			change_type, change_summary
		) VALUES ($1, $2, $3, $4, $5)`,
		entityID, version, author, changeType, summary,
	)
	return err
}

// AddVulnerabilityAttributes adds vulnerability-specific attributes
func (r *Registry) AddVulnerabilityAttributes(ctx context.Context, entityID string, cvssScore float64, complexity string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vulnerability_attributes (
			entity_id, cvss_score, exploitation_complexity
		) VALUES ($1, $2, $3)
		ON CONFLICT (entity_id) DO UPDATE SET
			cvss_score = $2,
			exploitation_complexity = $3`,
		entityID, cvssScore, complexity,
	)
	return err
}

// === Entity Relationship Methods ===

// AddRelationship creates a new relationship between entities
func (r *Registry) AddRelationship(ctx context.Context, sourceID, targetID, relationshipType string, confidence float64, sourceOfTruth string, metadata map[string]interface{}) (*EntityRelationship, error) {
	metadataJSON, _ := json.Marshal(metadata)
	
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO entity_relationships (
			source_entity_id, target_entity_id, relationship_type,
			relationship_metadata
		) VALUES ($1, $2, $3, $4)`,
		sourceID, targetID, relationshipType, string(metadataJSON),
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to add relationship: %w", err)
	}
	
	// Return the created relationship
	return &EntityRelationship{
		SourceEntityID:       sourceID,
		TargetEntityID:       targetID,
		RelationshipType:     relationshipType,
		RelationshipMetadata: metadata,
		CreatedAt:            time.Now(),
	}, nil
}

// GetRelationships retrieves all relationships for an entity
func (r *Registry) GetRelationships(ctx context.Context, entityID string) ([]*EntityRelationship, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			id, source_entity_id, target_entity_id, relationship_type,
			relationship_metadata, created_at
		FROM entity_relationships
		WHERE source_entity_id = $1 OR target_entity_id = $1
		ORDER BY created_at DESC`,
		entityID,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships: %w", err)
	}
	defer rows.Close()
	
	var relationships []*EntityRelationship
	for rows.Next() {
		var rel EntityRelationship
		var metadataJSON interface{} // DuckDB returns JSON as interface{}
		
		err := rows.Scan(
			&rel.ID, &rel.SourceEntityID, &rel.TargetEntityID,
			&rel.RelationshipType, &metadataJSON, &rel.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		// Parse metadata - DuckDB returns JSON as map[string]interface{}
		if metadataJSON != nil {
			switch v := metadataJSON.(type) {
			case map[string]interface{}:
				rel.RelationshipMetadata = v
			case string:
				if v != "" {
					json.Unmarshal([]byte(v), &rel.RelationshipMetadata)
				}
			}
		}
		
		relationships = append(relationships, &rel)
	}
	
	return relationships, nil
}

// GetRelationshipsByType retrieves relationships of a specific type
func (r *Registry) GetRelationshipsByType(ctx context.Context, relationshipType string) ([]*EntityRelationship, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			id, source_entity_id, target_entity_id, relationship_type,
			relationship_metadata, created_at
		FROM entity_relationships
		WHERE relationship_type = $1
		ORDER BY created_at DESC`,
		relationshipType,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships by type: %w", err)
	}
	defer rows.Close()
	
	var relationships []*EntityRelationship
	for rows.Next() {
		var rel EntityRelationship
		var metadataJSON interface{} // DuckDB returns JSON as interface{}
		
		err := rows.Scan(
			&rel.ID, &rel.SourceEntityID, &rel.TargetEntityID,
			&rel.RelationshipType, &metadataJSON, &rel.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		// Parse metadata - DuckDB returns JSON as map[string]interface{}
		if metadataJSON != nil {
			switch v := metadataJSON.(type) {
			case map[string]interface{}:
				rel.RelationshipMetadata = v
			case string:
				if v != "" {
					json.Unmarshal([]byte(v), &rel.RelationshipMetadata)
				}
			}
		}
		
		relationships = append(relationships, &rel)
	}
	
	return relationships, nil
}

// FindVulnerabilitiesDetectedByModule finds all vulnerabilities detected by a specific module
func (r *Registry) FindVulnerabilitiesDetectedByModule(ctx context.Context, moduleID string) ([]*Entity, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			e.entity_id, e.entity_type, e.base_id, e.version,
			e.name, e.description, e.status, e.severity,
			e.created_at, e.updated_at,
			r.relationship_metadata
		FROM entities e
		JOIN entity_relationships r ON e.entity_id = r.target_entity_id
		WHERE r.source_entity_id = $1
		  AND r.relationship_type = 'DETECTS'
		  AND e.entity_type = 'VUL'
		ORDER BY e.created_at DESC`,
		moduleID,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to find vulnerabilities detected by module: %w", err)
	}
	defer rows.Close()
	
	var entities []*Entity
	for rows.Next() {
		var e Entity
		var description, severity sql.NullString
		var metadataJSON interface{} // DuckDB returns JSON as interface{}
		
		err := rows.Scan(
			&e.EntityID, &e.EntityType, &e.BaseID, &e.Version,
			&e.Name, &description, &e.Status, &severity,
			&e.CreatedAt, &e.UpdatedAt, &metadataJSON,
		)
		if err != nil {
			return nil, err
		}
		
		// Handle nullable fields
		if description.Valid {
			e.Description = description.String
		}
		if severity.Valid {
			e.Severity = SeverityLevel(severity.String)
		}
		
		entities = append(entities, &e)
	}
	
	return entities, nil
}

// FindModulesDetectingVulnerability finds all modules that detect a specific vulnerability
func (r *Registry) FindModulesDetectingVulnerability(ctx context.Context, vulnerabilityID string) ([]*Entity, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			e.entity_id, e.entity_type, e.base_id, e.version,
			e.name, e.description, e.status, e.severity,
			e.created_at, e.updated_at,
			r.relationship_metadata
		FROM entities e
		JOIN entity_relationships r ON e.entity_id = r.source_entity_id
		WHERE r.target_entity_id = $1
		  AND r.relationship_type = 'DETECTS'
		  AND e.entity_type = 'MOD'
		ORDER BY e.created_at DESC`,
		vulnerabilityID,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to find modules detecting vulnerability: %w", err)
	}
	defer rows.Close()
	
	var entities []*Entity
	for rows.Next() {
		var e Entity
		var description, severity sql.NullString
		var metadataJSON interface{} // DuckDB returns JSON as interface{}
		
		err := rows.Scan(
			&e.EntityID, &e.EntityType, &e.BaseID, &e.Version,
			&e.Name, &description, &e.Status, &severity,
			&e.CreatedAt, &e.UpdatedAt, &metadataJSON,
		)
		if err != nil {
			return nil, err
		}
		
		// Handle nullable fields
		if description.Valid {
			e.Description = description.String
		}
		if severity.Valid {
			e.Severity = SeverityLevel(severity.String)
		}
		
		entities = append(entities, &e)
	}
	
	return entities, nil
}

// FindAttackChains discovers attack → vulnerability → detection module chains
func (r *Registry) FindAttackChains(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH attack_vulns AS (
			SELECT 
				r1.source_entity_id as attack_id,
				r1.target_entity_id as vuln_id,
				r1.relationship_metadata as exploit_metadata
			FROM entity_relationships r1
			WHERE r1.relationship_type = 'EXPLOITS'
		),
		vuln_detectors AS (
			SELECT 
				r2.source_entity_id as module_id,
				r2.target_entity_id as vuln_id,
				r2.relationship_metadata as detect_metadata
			FROM entity_relationships r2
			WHERE r2.relationship_type = 'DETECTS'
		)
		SELECT 
			av.attack_id,
			av.vuln_id,
			vd.module_id,
			av.exploit_metadata,
			vd.detect_metadata
		FROM attack_vulns av
		JOIN vuln_detectors vd ON av.vuln_id = vd.vuln_id
		ORDER BY av.attack_id, av.vuln_id`,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to find attack chains: %w", err)
	}
	defer rows.Close()
	
	var chains []map[string]interface{}
	for rows.Next() {
		var attackID, vulnID, moduleID string
		var exploitMeta, detectMeta interface{} // DuckDB returns JSON as interface{}
		
		err := rows.Scan(&attackID, &vulnID, &moduleID, &exploitMeta, &detectMeta)
		if err != nil {
			return nil, err
		}
		
		chain := map[string]interface{}{
			"attack_id":  attackID,
			"vuln_id":    vulnID,
			"module_id":  moduleID,
		}
		
		// Parse metadata if present - DuckDB returns JSON as map[string]interface{}
		if exploitMeta != nil {
			switch v := exploitMeta.(type) {
			case map[string]interface{}:
				chain["exploit_metadata"] = v
			case string:
				if v != "" {
					var meta map[string]interface{}
					json.Unmarshal([]byte(v), &meta)
					chain["exploit_metadata"] = meta
				}
			}
		}
		
		if detectMeta != nil {
			switch v := detectMeta.(type) {
			case map[string]interface{}:
				chain["detect_metadata"] = v
			case string:
				if v != "" {
					var meta map[string]interface{}
					json.Unmarshal([]byte(v), &meta)
					chain["detect_metadata"] = meta
				}
			}
		}
		
		chains = append(chains, chain)
	}
	
	return chains, nil
}

// DeleteRelationship removes a relationship between entities
func (r *Registry) DeleteRelationship(ctx context.Context, sourceID, targetID, relationshipType string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM entity_relationships
		WHERE source_entity_id = $1 
		  AND target_entity_id = $2 
		  AND relationship_type = $3`,
		sourceID, targetID, relationshipType,
	)
	
	if err != nil {
		return fmt.Errorf("failed to delete relationship: %w", err)
	}
	
	return nil
}

// SearchEntities performs full-text search
func (r *Registry) SearchEntities(ctx context.Context, query string, filters map[string]interface{}) ([]*Entity, error) {
	// Build search query
	baseQuery := `
		SELECT 
			entity_id, entity_type, base_id, version,
			name, description, status, severity,
			created_at, updated_at
		FROM entities
		WHERE search_vector @@ plainto_tsquery('english', $1)
	`
	
	// Add filters
	args := []interface{}{query}
	argNum := 2
	
	for key, value := range filters {
		baseQuery += fmt.Sprintf(" AND %s = $%d", key, argNum)
		args = append(args, value)
		argNum++
	}
	
	baseQuery += " ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC"
	
	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var entities []*Entity
	for rows.Next() {
		var e Entity
		err := rows.Scan(
			&e.EntityID, &e.EntityType, &e.BaseID, &e.Version,
			&e.Name, &e.Description, &e.Status, &e.Severity,
			&e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		entities = append(entities, &e)
	}
	
	return entities, nil
}

// createSchema creates the database schema
func (r *Registry) createSchema() error {
	// Create sequences first
	sequences := []string{
		"CREATE SEQUENCE IF NOT EXISTS seq_version_history START 1",
		"CREATE SEQUENCE IF NOT EXISTS seq_relationships START 1",
		"CREATE SEQUENCE IF NOT EXISTS seq_changelog START 1",
		"CREATE SEQUENCE IF NOT EXISTS seq_audit START 1",
	}
	
	for _, seq := range sequences {
		if _, err := r.db.Exec(seq); err != nil {
			return fmt.Errorf("failed to create sequence: %w", err)
		}
	}
	
	// Simplified schema for DuckDB compatibility
	schema := `
	CREATE TABLE IF NOT EXISTS entities (
		entity_id VARCHAR PRIMARY KEY,
		entity_type VARCHAR NOT NULL,
		base_id VARCHAR NOT NULL,
		version VARCHAR NOT NULL,
		name VARCHAR NOT NULL,
		description TEXT,
		status VARCHAR DEFAULT 'draft',
		severity VARCHAR,
		discovery_date TIMESTAMP,
		analysis_date TIMESTAMP,
		implementation_date TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		archived_at TIMESTAMP,
		author VARCHAR,
		organization VARCHAR DEFAULT 'Strigoi Team',
		category VARCHAR,
		tags VARCHAR,
		metadata JSON,
		configuration JSON,
		search_vector VARCHAR
	);
	
	CREATE TABLE IF NOT EXISTS version_history (
		id INTEGER PRIMARY KEY DEFAULT nextval('seq_version_history'),
		entity_id VARCHAR,
		version_from VARCHAR,
		version_to VARCHAR NOT NULL,
		change_type VARCHAR NOT NULL,
		change_description TEXT,
		changed_by VARCHAR,
		changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		rollback_info JSON
	);
	
	CREATE TABLE IF NOT EXISTS changelog (
		id INTEGER PRIMARY KEY DEFAULT nextval('seq_changelog'),
		entity_id VARCHAR,
		change_version VARCHAR NOT NULL,
		change_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		change_author VARCHAR,
		change_type VARCHAR,
		change_summary TEXT,
		change_details JSON,
		commit_hash VARCHAR
	);
	
	CREATE TABLE IF NOT EXISTS entity_relationships (
		id INTEGER PRIMARY KEY DEFAULT nextval('seq_relationships'),
		source_entity_id VARCHAR,
		target_entity_id VARCHAR,
		relationship_type VARCHAR NOT NULL,
		relationship_metadata JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS vulnerability_attributes (
		entity_id VARCHAR PRIMARY KEY,
		cve_id VARCHAR,
		cvss_score DECIMAL(3,1),
		cvss_vector VARCHAR,
		affected_systems VARCHAR,
		exploitation_complexity VARCHAR,
		remediation_available BOOLEAN DEFAULT FALSE,
		public_disclosure_date TIMESTAMP
	);
	`
	
	_, err := r.db.Exec(schema)
	return err
}