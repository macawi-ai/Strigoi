-- Strigoi Entity Registry Schema
-- Version: 1.0.0
-- Purpose: Track all entities with comprehensive versioning and relationships

-- Core entity types enumeration
CREATE TYPE entity_type AS ENUM (
    'MOD',  -- Modules
    'VUL',  -- Vulnerabilities  
    'ATK',  -- Attack Patterns
    'SIG',  -- Detection Signatures
    'DEM',  -- Demonstrators/PoCs
    'CFG',  -- Configurations
    'POL',  -- Policies
    'RPT',  -- Reports
    'SES',  -- Sessions
    'RUN',  -- Test Runs
    'MAN',  -- Manifolds
    'PRO',  -- Protocols
    'SCN',  -- Scenarios
    'TOP',  -- Topologies
    'EVD',  -- Evidence
    'BLU',  -- Blueprints
    'TEL',  -- Telemetry
    'VAL',  -- Validation Cycles
    'NBK',  -- Notebooks
    'PKG'   -- Packages
);

-- Entity status enumeration
CREATE TYPE entity_status AS ENUM (
    'draft',
    'active', 
    'testing',
    'deprecated',
    'archived',
    'revoked'
);

-- Risk/severity levels
CREATE TYPE severity_level AS ENUM (
    'critical',
    'high',
    'medium',
    'low',
    'info'
);

-- Main entities table - tracks all entities with versioning
CREATE TABLE entities (
    -- Identity
    entity_id VARCHAR PRIMARY KEY,  -- e.g., MOD-2025-10001-v1.0.0
    entity_type entity_type NOT NULL,
    base_id VARCHAR NOT NULL,       -- e.g., MOD-2025-10001 (without version)
    version VARCHAR NOT NULL,       -- e.g., v1.0.0 or v001
    
    -- Basic metadata
    name VARCHAR NOT NULL,
    description TEXT,
    status entity_status DEFAULT 'draft',
    severity severity_level,
    
    -- Timestamps
    discovery_date TIMESTAMP,       -- When first discovered/conceived
    analysis_date TIMESTAMP,        -- When analyzed/researched
    implementation_date TIMESTAMP,  -- When implemented
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    archived_at TIMESTAMP,
    
    -- Attribution
    author VARCHAR,
    organization VARCHAR DEFAULT 'Strigoi Team',
    
    -- Categorization
    category VARCHAR,               -- Sub-category within type
    tags VARCHAR[],                 -- Array of tags
    
    -- Technical details (JSON for flexibility)
    metadata JSON,                 -- Type-specific metadata
    configuration JSON,            -- Configuration data
    
    -- Search optimization
    search_vector VARCHAR,         -- Full-text search content
    
    -- Constraints
    UNIQUE(base_id, version),
    CHECK (entity_id = base_id || '-' || version)
);

-- Version history tracking
CREATE TABLE version_history (
    id SERIAL PRIMARY KEY,
    entity_id VARCHAR REFERENCES entities(entity_id),
    version_from VARCHAR,
    version_to VARCHAR NOT NULL,
    change_type VARCHAR NOT NULL,   -- major, minor, patch, security
    change_description TEXT,
    changed_by VARCHAR,
    changed_at TIMESTAMP DEFAULT NOW(),
    rollback_info JSON
);

-- Entity relationships (graph structure)
CREATE TABLE entity_relationships (
    id SERIAL PRIMARY KEY,
    source_entity_id VARCHAR REFERENCES entities(entity_id),
    target_entity_id VARCHAR REFERENCES entities(entity_id),
    relationship_type VARCHAR NOT NULL,  -- uses, implements, detects, exploits, etc.
    relationship_metadata JSON,
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Prevent duplicate relationships
    UNIQUE(source_entity_id, target_entity_id, relationship_type)
);

-- Changelog for granular tracking
CREATE TABLE changelog (
    id SERIAL PRIMARY KEY,
    entity_id VARCHAR REFERENCES entities(entity_id),
    change_version VARCHAR NOT NULL,
    change_date TIMESTAMP DEFAULT NOW(),
    change_author VARCHAR,
    change_type VARCHAR,            -- added, modified, removed, security_fix
    change_summary TEXT,
    change_details JSON,           -- Detailed changes
    commit_hash VARCHAR             -- Git commit reference if applicable
);

-- Module-specific attributes
CREATE TABLE module_attributes (
    entity_id VARCHAR PRIMARY KEY REFERENCES entities(entity_id),
    module_type VARCHAR,            -- attack, scanner, discovery, etc.
    risk_level severity_level,
    requirements VARCHAR[],         -- System requirements
    options JSON,                  -- Module options schema
    performance_metrics JSON       -- Execution metrics
);

-- Vulnerability-specific attributes
CREATE TABLE vulnerability_attributes (
    entity_id VARCHAR PRIMARY KEY REFERENCES entities(entity_id),
    cve_id VARCHAR,
    cvss_score DECIMAL(3,1),
    cvss_vector VARCHAR,
    affected_systems VARCHAR[],
    exploitation_complexity VARCHAR,
    remediation_available BOOLEAN DEFAULT FALSE,
    public_disclosure_date TIMESTAMP
);

-- Detection signature attributes
CREATE TABLE signature_attributes (
    entity_id VARCHAR PRIMARY KEY REFERENCES entities(entity_id),
    signature_type VARCHAR,         -- yara, behavioral, network, log
    signature_content TEXT,
    false_positive_rate DECIMAL(5,2),
    detection_confidence VARCHAR,
    performance_impact VARCHAR
);

-- Test run results
CREATE TABLE test_runs (
    entity_id VARCHAR PRIMARY KEY REFERENCES entities(entity_id),
    run_date TIMESTAMP NOT NULL,
    duration_seconds INTEGER,
    success BOOLEAN,
    findings_count INTEGER,
    environment JSON,
    results JSON,
    artifacts_path VARCHAR
);

-- Indexes for performance
CREATE INDEX idx_entities_type ON entities(entity_type);
CREATE INDEX idx_entities_status ON entities(status);
CREATE INDEX idx_entities_base_id ON entities(base_id);
CREATE INDEX idx_entities_created ON entities(created_at DESC);
CREATE INDEX idx_entities_search ON entities USING gin(search_vector);
CREATE INDEX idx_entities_tags ON entities USING gin(tags);
CREATE INDEX idx_relationships_source ON entity_relationships(source_entity_id);
CREATE INDEX idx_relationships_target ON entity_relationships(target_entity_id);
CREATE INDEX idx_changelog_entity ON changelog(entity_id, change_date DESC);

-- Full-text search trigger
CREATE OR REPLACE FUNCTION update_search_vector() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.category, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(array_to_string(NEW.tags, ' '), '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_search_vector_trigger 
    BEFORE INSERT OR UPDATE ON entities
    FOR EACH ROW EXECUTE FUNCTION update_search_vector();

-- Audit trail
CREATE TABLE audit_log (
    id SERIAL PRIMARY KEY,
    entity_id VARCHAR,
    action VARCHAR NOT NULL,        -- create, update, delete, view
    user_id VARCHAR,
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMP DEFAULT NOW(),
    details JSON
);

-- Views for common queries

-- Latest version of each entity
CREATE VIEW latest_entities AS
SELECT DISTINCT ON (base_id) *
FROM entities
ORDER BY base_id, version DESC;

-- Active modules view
CREATE VIEW active_modules AS
SELECT e.*, ma.*
FROM entities e
JOIN module_attributes ma ON e.entity_id = ma.entity_id
WHERE e.entity_type = 'MOD' AND e.status = 'active';

-- Vulnerability dashboard
CREATE VIEW vulnerability_dashboard AS
SELECT e.*, va.*
FROM entities e
JOIN vulnerability_attributes va ON e.entity_id = va.entity_id
WHERE e.entity_type = 'VUL' AND e.status IN ('active', 'testing');

-- Entity statistics
CREATE VIEW entity_statistics AS
SELECT 
    entity_type,
    COUNT(*) as total_count,
    COUNT(DISTINCT base_id) as unique_entities,
    COUNT(CASE WHEN status = 'active' THEN 1 END) as active_count,
    COUNT(CASE WHEN created_at > NOW() - INTERVAL '30 days' THEN 1 END) as recent_count
FROM entities
GROUP BY entity_type;

-- Functions for entity management

-- Generate next ID for entity type
CREATE OR REPLACE FUNCTION generate_entity_id(
    p_entity_type entity_type,
    p_year INTEGER DEFAULT EXTRACT(YEAR FROM NOW())
) RETURNS VARCHAR AS $$
DECLARE
    v_prefix VARCHAR;
    v_next_num INTEGER;
    v_category_start INTEGER;
BEGIN
    -- Get prefix
    v_prefix := p_entity_type::text;
    
    -- Determine category start based on type (for modules)
    IF p_entity_type = 'MOD' THEN
        v_category_start := 10000; -- Default to attack modules
    ELSE
        v_category_start := 1;
    END IF;
    
    -- Get next number
    SELECT COALESCE(MAX(CAST(SUBSTRING(base_id FROM '\d{5}$') AS INTEGER)), v_category_start - 1) + 1
    INTO v_next_num
    FROM entities
    WHERE entity_type = p_entity_type
    AND base_id LIKE v_prefix || '-' || p_year || '-%';
    
    RETURN v_prefix || '-' || p_year || '-' || LPAD(v_next_num::text, 5, '0');
END;
$$ LANGUAGE plpgsql;

-- Version comparison function
CREATE OR REPLACE FUNCTION compare_versions(v1 VARCHAR, v2 VARCHAR) 
RETURNS INTEGER AS $$
-- Returns: -1 if v1 < v2, 0 if equal, 1 if v1 > v2
BEGIN
    -- Simple implementation - can be enhanced for semantic versioning
    IF v1 < v2 THEN RETURN -1;
    ELSIF v1 > v2 THEN RETURN 1;
    ELSE RETURN 0;
    END IF;
END;
$$ LANGUAGE plpgsql;