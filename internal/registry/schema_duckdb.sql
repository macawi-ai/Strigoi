-- Strigoi Entity Registry Schema for DuckDB
-- Version: 1.0.0
-- Purpose: Track all entities with comprehensive versioning and relationships

-- Main entities table - tracks all entities with versioning
CREATE TABLE IF NOT EXISTS entities (
    -- Identity
    entity_id VARCHAR PRIMARY KEY,  -- e.g., MOD-2025-10001-v1.0.0
    entity_type VARCHAR NOT NULL,   -- MOD, VUL, ATK, etc.
    base_id VARCHAR NOT NULL,       -- e.g., MOD-2025-10001 (without version)
    version VARCHAR NOT NULL,       -- e.g., v1.0.0 or v001
    
    -- Basic metadata
    name VARCHAR NOT NULL,
    description TEXT,
    status VARCHAR DEFAULT 'draft', -- draft, active, testing, deprecated, archived, revoked
    severity VARCHAR,               -- critical, high, medium, low, info
    
    -- Timestamps
    discovery_date TIMESTAMP,       -- When first discovered/conceived
    analysis_date TIMESTAMP,        -- When analyzed/researched
    implementation_date TIMESTAMP,  -- When implemented
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    archived_at TIMESTAMP,
    
    -- Attribution
    author VARCHAR,
    organization VARCHAR DEFAULT 'Strigoi Team',
    
    -- Categorization
    category VARCHAR,               -- Sub-category within type
    tags VARCHAR,                   -- Comma-separated tags (DuckDB doesn't support arrays well)
    
    -- Technical details (JSON for flexibility)
    metadata JSON,                  -- Type-specific metadata
    configuration JSON,             -- Configuration data
    
    -- Search optimization
    search_vector VARCHAR,          -- Full-text search content
    
    -- Constraints
    UNIQUE(base_id, version)
);

-- Version history tracking
CREATE SEQUENCE IF NOT EXISTS seq_version_history START 1;
CREATE TABLE IF NOT EXISTS version_history (
    id INTEGER PRIMARY KEY DEFAULT nextval('seq_version_history'),
    entity_id VARCHAR,
    version_from VARCHAR,
    version_to VARCHAR NOT NULL,
    change_type VARCHAR NOT NULL,   -- major, minor, patch, security
    change_description TEXT,
    changed_by VARCHAR,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    rollback_info JSON,
    FOREIGN KEY (entity_id) REFERENCES entities(entity_id)
);

-- Entity relationships (graph structure)
CREATE SEQUENCE IF NOT EXISTS seq_relationships START 1;
CREATE TABLE IF NOT EXISTS entity_relationships (
    id INTEGER PRIMARY KEY DEFAULT nextval('seq_relationships'),
    source_entity_id VARCHAR,
    target_entity_id VARCHAR,
    relationship_type VARCHAR NOT NULL,  -- uses, implements, detects, exploits, etc.
    relationship_metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (source_entity_id) REFERENCES entities(entity_id),
    FOREIGN KEY (target_entity_id) REFERENCES entities(entity_id),
    -- Prevent duplicate relationships
    UNIQUE(source_entity_id, target_entity_id, relationship_type)
);

-- Changelog for granular tracking
CREATE SEQUENCE IF NOT EXISTS seq_changelog START 1;
CREATE TABLE IF NOT EXISTS changelog (
    id INTEGER PRIMARY KEY DEFAULT nextval('seq_changelog'),
    entity_id VARCHAR,
    change_version VARCHAR NOT NULL,
    change_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    change_author VARCHAR,
    change_type VARCHAR,            -- added, modified, removed, security_fix
    change_summary TEXT,
    change_details JSON,            -- Detailed changes
    commit_hash VARCHAR,            -- Git commit reference if applicable
    FOREIGN KEY (entity_id) REFERENCES entities(entity_id)
);

-- Module-specific attributes
CREATE TABLE IF NOT EXISTS module_attributes (
    entity_id VARCHAR PRIMARY KEY,
    module_type VARCHAR,            -- attack, scanner, discovery, etc.
    risk_level VARCHAR,             -- critical, high, medium, low, info
    requirements VARCHAR,           -- Comma-separated system requirements
    options JSON,                   -- Module options schema
    performance_metrics JSON,       -- Execution metrics
    FOREIGN KEY (entity_id) REFERENCES entities(entity_id)
);

-- Vulnerability-specific attributes
CREATE TABLE IF NOT EXISTS vulnerability_attributes (
    entity_id VARCHAR PRIMARY KEY,
    cve_id VARCHAR,
    cvss_score DECIMAL(3,1),
    cvss_vector VARCHAR,
    affected_systems VARCHAR,       -- Comma-separated list
    exploitation_complexity VARCHAR,
    remediation_available BOOLEAN DEFAULT FALSE,
    public_disclosure_date TIMESTAMP,
    FOREIGN KEY (entity_id) REFERENCES entities(entity_id)
);

-- Detection signature attributes
CREATE TABLE IF NOT EXISTS signature_attributes (
    entity_id VARCHAR PRIMARY KEY,
    signature_type VARCHAR,         -- yara, behavioral, network, log
    signature_content TEXT,
    false_positive_rate DECIMAL(5,2),
    detection_confidence VARCHAR,
    performance_impact VARCHAR,
    FOREIGN KEY (entity_id) REFERENCES entities(entity_id)
);

-- Test run results
CREATE TABLE IF NOT EXISTS test_runs (
    entity_id VARCHAR PRIMARY KEY,
    run_date TIMESTAMP NOT NULL,
    duration_seconds INTEGER,
    success BOOLEAN,
    findings_count INTEGER,
    environment JSON,
    results JSON,
    artifacts_path VARCHAR,
    FOREIGN KEY (entity_id) REFERENCES entities(entity_id)
);

-- Audit trail
CREATE SEQUENCE IF NOT EXISTS seq_audit START 1;
CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY DEFAULT nextval('seq_audit'),
    entity_id VARCHAR,
    action VARCHAR NOT NULL,        -- create, update, delete, view
    user_id VARCHAR,
    ip_address VARCHAR,
    user_agent TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    details JSON
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(entity_type);
CREATE INDEX IF NOT EXISTS idx_entities_status ON entities(status);
CREATE INDEX IF NOT EXISTS idx_entities_base_id ON entities(base_id);
CREATE INDEX IF NOT EXISTS idx_entities_created ON entities(created_at);
CREATE INDEX IF NOT EXISTS idx_relationships_source ON entity_relationships(source_entity_id);
CREATE INDEX IF NOT EXISTS idx_relationships_target ON entity_relationships(target_entity_id);
CREATE INDEX IF NOT EXISTS idx_changelog_entity ON changelog(entity_id, change_date);