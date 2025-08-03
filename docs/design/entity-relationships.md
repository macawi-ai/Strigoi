# Entity Relationship Design for Strigoi

*Based on Gemini A2A consultation*

## Overview

This document describes the flexible entity relationship system for Strigoi's entity registry, designed to handle many-to-many relationships with rich metadata and version history.

## Core Design Principle

Instead of creating separate link tables for each relationship type, we use a **single, centralized association table** that can flexibly represent any relationship between any entities.

## Schema Design

### Entity Relationships Table

```sql
CREATE TABLE entity_relationships (
    -- Relationship Identity
    relationship_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- The "From" and "To" sides of the relationship
    source_id VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,

    -- The Verb: Defines the nature of the relationship
    relationship_type VARCHAR NOT NULL, -- e.g., 'DETECTS', 'EXPLOITS', 'MITIGATES', 'CONTAINS'

    -- Relationship Metadata (the "Adverbs")
    confidence DOUBLE DEFAULT 1.0,
    discovery_date TIMESTAMPTZ DEFAULT now(),
    source_of_truth VARCHAR, -- e.g., 'NVD', 'InternalScan', 'ThreatIntelFeed'
    status VARCHAR DEFAULT 'ACTIVE', -- e.g., 'ACTIVE', 'DEPRECATED', 'PENDING_REVIEW'

    -- Built-in Version History
    valid_since TIMESTAMPTZ DEFAULT now(),
    superseded_at TIMESTAMPTZ, -- NULL means this is the current, active version

    -- Flexible key-value metadata for anything else
    metadata JSON
);

-- Create indexes for fast lookups
CREATE INDEX idx_relationships_source ON entity_relationships(source_id);
CREATE INDEX idx_relationships_target ON entity_relationships(target_id);
CREATE INDEX idx_relationships_type ON entity_relationships(relationship_type);
CREATE INDEX idx_relationships_active ON entity_relationships(superseded_at);
```

## Relationship Types

### Primary Relationships

1. **DETECTS**: MOD → VUL (Module detects vulnerability)
2. **EXPLOITS**: ATK → VUL (Attack pattern exploits vulnerability)
3. **MITIGATES**: SIG → ATK (Signature mitigates attack)
4. **IMPLEMENTS**: MOD → ATK (Module implements attack for testing)
5. **CONFIGURES**: CFG → MOD (Configuration applies to module)
6. **ENFORCES**: POL → VUL (Policy enforces against vulnerability)
7. **DOCUMENTS**: RPT → VUL/ATK/MOD (Report documents entity)
8. **USES**: RUN → MOD (Run session uses module)
9. **DISCOVERED**: RUN → VUL (Run session discovered vulnerability)
10. **REQUIRES**: MOD → MOD (Module requires another module)

### Metadata Examples

```json
{
  "false_positive_rate": 0.02,
  "detection_accuracy": 0.98,
  "environmental_factors": ["network_segmented", "ids_active"],
  "severity_adjustment": "+2",
  "notes": "Confidence reduced due to environmental complexity"
}
```

## Usage Patterns

### 1. Creating a Relationship

```sql
INSERT INTO entity_relationships (
    source_id, 
    target_id, 
    relationship_type, 
    confidence, 
    source_of_truth,
    metadata
) VALUES (
    'MOD-2025-10001',           -- Config Scanner Module
    'VUL-2025-00001',           -- Rogue MCP Sudo Tailgating
    'DETECTS',
    0.95,
    'Internal Testing',
    '{"detection_method": "behavioral_analysis", "test_cycles": 5}'
);
```

### 2. Updating a Relationship (Versioning)

```sql
BEGIN;

-- Mark old relationship as superseded
UPDATE entity_relationships
SET superseded_at = NOW()
WHERE source_id = 'MOD-2025-10001'
  AND target_id = 'VUL-2025-00001'
  AND relationship_type = 'DETECTS'
  AND superseded_at IS NULL;

-- Insert new version
INSERT INTO entity_relationships (
    source_id, target_id, relationship_type, 
    confidence, source_of_truth
) VALUES (
    'MOD-2025-10001', 'VUL-2025-00001', 'DETECTS',
    0.80, 'Updated Testing Results'
);

COMMIT;
```

### 3. Query Patterns

#### Find all vulnerabilities detected by a module
```sql
SELECT 
    v.*,
    r.confidence,
    r.discovery_date,
    r.metadata
FROM vulnerabilities v
JOIN entity_relationships r ON v.entity_id = r.target_id
WHERE r.source_id = 'MOD-2025-10001'
  AND r.relationship_type = 'DETECTS'
  AND r.superseded_at IS NULL;
```

#### Find attack chains
```sql
-- Find ATK → VUL → MOD chains
WITH attack_vulns AS (
    SELECT 
        r1.source_id as attack_id,
        r1.target_id as vuln_id,
        r1.confidence as exploit_confidence
    FROM entity_relationships r1
    WHERE r1.relationship_type = 'EXPLOITS'
      AND r1.superseded_at IS NULL
),
vuln_detectors AS (
    SELECT 
        r2.source_id as module_id,
        r2.target_id as vuln_id,
        r2.confidence as detect_confidence
    FROM entity_relationships r2
    WHERE r2.relationship_type = 'DETECTS'
      AND r2.superseded_at IS NULL
)
SELECT 
    av.attack_id,
    av.vuln_id,
    vd.module_id,
    av.exploit_confidence * vd.detect_confidence as chain_confidence
FROM attack_vulns av
JOIN vuln_detectors vd ON av.vuln_id = vd.vuln_id
ORDER BY chain_confidence DESC;
```

## Benefits

1. **Flexibility**: New relationship types require no schema changes
2. **History**: Complete audit trail with point-in-time queries
3. **Performance**: DuckDB excels at analytical queries on this structure
4. **Metadata**: Rich context for each relationship
5. **Simplicity**: Single table to understand and maintain

## Integration with Entity Registry

This relationship system integrates seamlessly with the existing entity registry:
- Entity IDs (MOD-YYYY-#####) serve as foreign keys
- Relationships are themselves versioned like entities
- The registry can track relationship changes as events

## Next Steps

1. Implement the schema in DuckDB
2. Create Go structures and methods for relationship management
3. Add relationship queries to the Registry interface
4. Build visualization tools for relationship graphs
5. Implement relationship-based risk scoring