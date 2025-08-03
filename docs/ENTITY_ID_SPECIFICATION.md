# Strigoi Entity ID Specification
## Comprehensive Identity and Version Control System

*Version: 1.0.0*  
*Status: Active*

---

## Overview

The Strigoi Entity ID system provides granular version control and tracking for all components in the security framework. Every trackable entity receives a unique identifier with semantic versioning support.

---

## ID Format

### Standard Format
```
[PREFIX]-[YEAR]-[NUMBER]-[VERSION]
```

Example: `MOD-2025-10001-v1.2.3`

### Components

1. **PREFIX** (3 letters): Entity type identifier
2. **YEAR** (4 digits): Year of creation
3. **NUMBER** (5 digits): Sequential number within type/year
4. **VERSION**: Semantic version (vMAJOR.MINOR.PATCH)

---

## Entity Types and Prefixes

| Prefix | Entity Type | Number Range | Description |
|--------|-------------|--------------|-------------|
| MOD | Modules | 10000-99999 | Security modules and tools |
| VUL | Vulnerabilities | 00001-99999 | Discovered vulnerabilities |
| ATK | Attack Patterns | 00001-99999 | Attack techniques and chains |
| SIG | Signatures | 00001-99999 | Detection signatures |
| DEM | Demonstrators | 00001-99999 | PoCs and demonstrations |
| CFG | Configurations | 00001-99999 | System configurations |
| POL | Policies | 00001-99999 | Security policies |
| RPT | Reports | 00001-99999 | Assessment reports |
| SES | Sessions | 00001-99999 | Test/scan sessions |
| RUN | Test Runs | TIMESTAMP | Individual executions |
| MAN | Manifolds | 00001-99999 | Test manifolds |
| PRO | Protocols | NAME-VERSION | Protocol definitions |
| SCN | Scenarios | 00001-99999 | Attack scenarios |
| TOP | Topologies | 00001-99999 | Network topologies |
| EVD | Evidence | 00001-99999 | Captured evidence |
| BLU | Blueprints | 00001-99999 | Attack blueprints |
| TEL | Telemetry | 00001-99999 | Telemetry data |
| VAL | Validations | 00001-99999 | Validation cycles |
| NBK | Notebooks | 00001-99999 | Lab notebooks |
| PKG | Packages | 00001-99999 | Deployable packages |

### Module Sub-ranges (MOD)

| Range | Category | Description |
|-------|----------|-------------|
| 10000-19999 | Attack | Offensive security modules |
| 20000-29999 | Scanner | Vulnerability scanners |
| 30000-39999 | Discovery | Discovery tools |
| 40000-49999 | Exploit | Exploitation modules |
| 50000-59999 | Payload | Payload generators |
| 60000-69999 | Post | Post-exploitation |
| 70000-79999 | Auxiliary | Helper modules |
| 90000-99999 | Misc | Uncategorized |

---

## Version Control

### Semantic Versioning

Format: `vMAJOR.MINOR.PATCH`

- **MAJOR**: Incompatible changes
- **MINOR**: New functionality (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Special Versions

- `v0.0.0`: Initial draft
- `v0.x.x`: Pre-release/testing
- `v1.0.0`: First stable release
- `-alpha`, `-beta`, `-rc`: Pre-release tags

### Version Lifecycle

1. **Draft** (`v0.0.x`): Under development
2. **Testing** (`v0.x.x`): In testing phase
3. **Active** (`v1.x.x`): Production ready
4. **Deprecated** (`vX.X.X-deprecated`): Scheduled for removal
5. **Archived** (`vX.X.X-archived`): Historical reference only

---

## Metadata Requirements

### Mandatory Fields

1. **entity_id**: Full ID with version
2. **name**: Human-readable name
3. **description**: Clear description
4. **status**: Current lifecycle status
5. **created_at**: Creation timestamp
6. **author**: Creator identification

### Optional Fields

1. **severity**: Risk level (for applicable types)
2. **tags**: Searchable keywords
3. **category**: Sub-classification
4. **metadata**: Type-specific data
5. **relationships**: Links to other entities

---

## Relationship Types

Entities can have the following relationships:

- **uses**: Module uses Signature
- **implements**: Module implements Attack Pattern
- **detects**: Signature detects Vulnerability
- **exploits**: Attack exploits Vulnerability
- **produces**: Test Run produces Evidence
- **contains**: Report contains Vulnerabilities
- **depends_on**: Module depends on Configuration
- **supersedes**: Newer version supersedes older

---

## Registry Operations

### Creation
```sql
-- Generate new entity ID
SELECT generate_entity_id('MOD', 2025); -- Returns: MOD-2025-10001

-- Register entity
INSERT INTO entities (entity_id, ...) VALUES ('MOD-2025-10001-v1.0.0', ...);
```

### Versioning
```sql
-- Create new version
INSERT INTO entities VALUES ('MOD-2025-10001-v1.1.0', ...);

-- Record version history
INSERT INTO version_history (entity_id, version_from, version_to, ...)
VALUES ('MOD-2025-10001', 'v1.0.0', 'v1.1.0', ...);
```

### Search
```sql
-- Find by ID
SELECT * FROM entities WHERE entity_id = 'MOD-2025-10001-v1.0.0';

-- Find latest version
SELECT * FROM latest_entities WHERE base_id = 'MOD-2025-10001';

-- Full-text search
SELECT * FROM entities WHERE search_vector @@ 'sudo tailgating';
```

---

## Best Practices

### ID Assignment

1. **Never reuse IDs** - Even if entity is deleted
2. **Sequential within type/year** - Maintains order
3. **Version everything** - Track all changes
4. **Meaningful prefixes** - Clear entity type

### Version Management

1. **Semantic versioning** - Clear change indication
2. **Changelog entries** - Document all changes
3. **Backward compatibility** - Note breaking changes
4. **Deprecation notices** - Gradual phase-out

### Metadata Quality

1. **Comprehensive descriptions** - Clear purpose
2. **Accurate timestamps** - Track lifecycle
3. **Proper categorization** - Enable discovery
4. **Rich relationships** - Show connections

---

## Examples

### Module Registration
```
MOD-2025-10001-v1.0.0: Sudo Cache Detection
MOD-2025-10001-v1.0.1: Bug fix for false positives
MOD-2025-10001-v1.1.0: Added MCP process counting
MOD-2025-10001-v2.0.0: Rewrite with new detection engine
```

### Vulnerability Tracking
```
VUL-2025-00042-v1.0.0: MCP Sudo Tailgating (discovered)
VUL-2025-00042-v1.1.0: Added CVSS scoring
VUL-2025-00042-v1.2.0: Updated affected systems
VUL-2025-00042-v1.3.0: Added remediation steps
```

### Attack Pattern Evolution
```
ATK-2025-00156-v1.0.0: Basic sudo exploitation
ATK-2025-00156-v2.0.0: Combined with MCP vectors
ATK-2025-00156-v2.1.0: Added container escape
```

---

## Integration Points

### With Strigoi Framework
- Module registration uses MOD- IDs
- Reports reference entity IDs
- Sessions track used modules

### With DuckDB Registry
- All entities stored in registry
- Full-text search enabled
- Relationship graph queryable

### With Git
- Commit hash tracking
- Version tag alignment
- Change attribution

---

## Future Enhancements

1. **Blockchain anchoring** - Immutable audit trail
2. **Federation support** - Cross-organization IDs
3. **AI-suggested IDs** - Smart categorization
4. **Dependency graphs** - Visual relationships
5. **Impact analysis** - Change propagation

---

*"Every entity tells a story through its version history"*