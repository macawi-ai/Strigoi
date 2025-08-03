# Actor Model Design - Meta-Structure

## Core Philosophy
Actors are living, intelligent agents that transform what they touch. They carry their own history, explain their purpose, and document their effects.

## Actor Data Structure Requirements

### 1. Identity & Providence
- **UUID**: Unique, persistent identifier
- **Name**: Human-readable identifier
- **Lineage**: Parent actors, inspirations, forks
- **Author**: Creator(s) with contact info
- **License**: Usage terms and restrictions
- **Signature**: Cryptographic verification

### 2. Versioning & History
- **Version**: Semantic versioning
- **Changelog**: What changed and why
- **Migration**: How to upgrade from previous versions
- **Deprecations**: What's being phased out
- **Compatibility**: Works with Strigoi versions X.Y.Z

### 3. Purpose & Behavior
- **Description**: What this actor does
- **Theory**: Why it works (academic references)
- **Assumptions**: What must be true for success
- **Limitations**: What it cannot do
- **Ethics**: Intended use and restrictions

### 4. Capabilities & Transformations
- **Inputs**: What data types/formats accepted
- **Outputs**: What it produces
- **Transformations**: How input becomes output
- **Side Effects**: Other changes it might cause
- **Observables**: What can be monitored

### 5. Implementation
- **Language**: Implementation language/runtime
- **Dependencies**: Required libraries/actors
- **Resources**: CPU, memory, network needs
- **Timeouts**: Expected execution times
- **Fallbacks**: What to do on failure

### 6. Results & Interpretation
- **Result Schema**: Structure of findings
- **Confidence Levels**: How certain are results
- **Severity Mapping**: How to interpret risk
- **Evidence Types**: What proof is provided
- **Recommendations**: Suggested actions

### 7. Network Participation
- **Chains With**: Compatible actors
- **Protocols**: Communication standards
- **Events**: What it broadcasts/subscribes to
- **State**: Stateless or stateful behavior
- **Coordination**: How it works with others

### 8. Observability & Debugging
- **Logs**: What it records
- **Metrics**: Performance indicators
- **Traces**: Execution path tracking
- **Debug Mode**: Verbose operation info
- **Health Checks**: Self-diagnostic capabilities

## Proposed YAML Structure

```yaml
# Actor Metadata Document
actor:
  # Identity
  uuid: "a7f3b8d2-9e5c-4a1d-b6f8-3c7e9d2a1b5f"
  name: "endpoint_discovery"
  display_name: "LLM Endpoint Discovery Actor"
  
  # Providence
  lineage:
    parent: "base_http_prober"
    inspired_by: ["owasp_api_scanner", "llm_security_toolkit"]
  
  author:
    name: "Strigoi Community"
    email: "actors@strigoi.security"
    pgp_key: "0xABCDEF1234567890"
  
  # Versioning
  version: "1.2.0"
  strigoi_compatibility: ">=0.3.0"
  
  # Legal
  license: "Apache-2.0"
  ethics:
    white_hat_only: true
    forbidden_targets: ["production_without_permission"]
    
# Behavioral Metadata
behavior:
  purpose: |
    Discovers and maps LLM API endpoints by probing common patterns
    used by major LLM providers (OpenAI, Anthropic, Google, etc.)
    
  theory: |
    Based on "Hofstadter's Law of API Standardization" - LLM providers
    tend to follow similar patterns for API design. By checking known
    patterns, we can quickly map the attack surface.
    References: [doi:10.1234/llm-security-2024]
    
  assumptions:
    - "Target exposes HTTP/HTTPS endpoints"
    - "Standard ports (80/443) unless specified"
    - "JSON-based API responses"
    
  limitations:
    - "Cannot detect non-standard endpoints"
    - "Requires network access to target"
    - "May trigger rate limiting"

# Technical Specification
specification:
  capabilities:
    - name: "api_detection"
      description: "Detects common LLM API patterns"
      confidence: 0.85
      
    - name: "version_enumeration"
      description: "Identifies API versions"
      confidence: 0.70
      
  transforms:
    probe:
      inputs:
        - type: "url"
          schema: {"type": "string", "pattern": "^https?://"}
        - type: "domain"
          schema: {"type": "string", "format": "hostname"}
      
      outputs:
        type: "endpoint_list"
        schema:
          type: "array"
          items:
            type: "object"
            properties:
              url: {"type": "string"}
              platform: {"type": "string"}
              confidence: {"type": "number"}
              
    sense:
      inputs:
        - type: "endpoint_list"
      outputs:
        type: "risk_assessment"
        
  resources:
    cpu: "low"
    memory: "128MB"
    network: "required"
    timeout: "30s"
    
# Implementation
implementation:
  runtime: "go"
  dependencies:
    - "net/http"
    - "encoding/json"
  
  # Actual code can be embedded or referenced
  code: |
    package actors
    
    func (a *EndpointDiscoveryActor) Probe(ctx context.Context, target Target) (*ProbeResult, error) {
        // Implementation here
    }
    
  # Or reference external file
  code_ref: "actors/north/endpoint_discovery.go"
  
# Results Interpretation
results:
  schema:
    findings:
      - platform: "string"
        endpoints: ["string"]
        version: "string"
        confidence: "float"
        
  interpretation:
    confidence_levels:
      high: ">= 0.8"
      medium: "0.5 - 0.79"
      low: "< 0.5"
      
    risk_mapping:
      multiple_platforms: "medium"
      exposed_admin_api: "high"
      missing_rate_limits: "medium"
      
  recommendations:
    exposed_endpoints: "Implement API gateway with authentication"
    version_disclosure: "Remove version information from responses"
    
# Network Behavior
network:
  chains_with:
    - "model_interrogation"
    - "auth_boundary_tester"
    - "rate_limit_analyzer"
    
  emits:
    - event: "endpoint_discovered"
      data: ["url", "platform", "confidence"]
      
  subscribes:
    - event: "new_target"
      handler: "auto_probe"
      
# Observability
observability:
  logs:
    - level: "info"
      message: "Discovered {platform} endpoint at {url}"
    - level: "warn"
      message: "Rate limited at {url}"
      
  metrics:
    - name: "endpoints_discovered"
      type: "counter"
    - name: "probe_duration"
      type: "histogram"
      
  health_checks:
    - name: "can_reach_internet"
      interval: "5m"
      
# Signature
signature:
  algorithm: "ed25519"
  public_key: "..."
  signature: "..."
```

## Search Optimization

### Single-Tier SQLite Database
For fast search across potentially thousands of actors, we use a single SQLite database that serves as both the search index and metadata store:

#### 1. Unified Database Schema

```sql
-- SQLite schema for comprehensive actor management
CREATE TABLE actors (
    uuid TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    version TEXT NOT NULL,
    direction TEXT NOT NULL,  -- north, east, south, west, center
    risk_level TEXT,          -- low, medium, high, critical
    description TEXT,
    author TEXT,
    license TEXT,
    yaml_path TEXT NOT NULL,  -- Path to full actor YAML file
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- Denormalized fields for fast access
    tags TEXT,                -- JSON array of tags
    platforms TEXT,           -- JSON array of platforms
    categories TEXT,          -- JSON array of categories
    requires TEXT,            -- JSON array of dependencies
    provides TEXT,            -- JSON array of capabilities
    chains_with TEXT          -- JSON array of compatible actors
);

-- Full-text search virtual table
CREATE VIRTUAL TABLE actor_search USING fts5(
    name, 
    description, 
    tags, 
    platforms,
    categories,
    author
);

-- Indexes for common queries
CREATE INDEX idx_actor_direction ON actors(direction);
CREATE INDEX idx_actor_risk ON actors(risk_level);
CREATE INDEX idx_actor_version ON actors(name, version);

-- Assemblage definitions
CREATE TABLE assemblages (
    uuid TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    version TEXT NOT NULL,
    description TEXT,
    yaml_path TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Assemblage membership
CREATE TABLE assemblage_actors (
    assemblage_uuid TEXT,
    actor_name TEXT,
    role TEXT,
    position INTEGER,
    FOREIGN KEY (assemblage_uuid) REFERENCES assemblages(uuid),
    FOREIGN KEY (actor_name) REFERENCES actors(name)
);

-- Chain definitions
CREATE TABLE chains (
    uuid TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    version TEXT NOT NULL,
    description TEXT,
    yaml_path TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Chain steps
CREATE TABLE chain_steps (
    chain_uuid TEXT,
    step_number INTEGER,
    actor_name TEXT,
    config TEXT,  -- JSON configuration
    FOREIGN KEY (chain_uuid) REFERENCES chains(uuid),
    FOREIGN KEY (actor_name) REFERENCES actors(name)
);

-- Simple key-value cache for frequently accessed data
CREATE TABLE cache (
    key TEXT PRIMARY KEY,
    value TEXT,
    expires_at TIMESTAMP
);
```

### Simplified Search Interface
```bash
# Find actors by direction
strigoi > actors/search north
strigoi > actors/search north --platform openai

# Find by tags or category
strigoi > actors/search --tag reconnaissance
strigoi > actors/search --category analysis --risk low

# Full-text search
strigoi > actors/search "llm endpoint"

# Show what an actor chains with
strigoi > actors/chains endpoint_discovery

# Find assemblages containing an actor
strigoi > actors/assemblages --contains endpoint_discovery
```

### Database Operations
```go
// Simple actor registration
func RegisterActor(db *sql.DB, yamlPath string) error {
    // Parse YAML file
    actor := parseActorYAML(yamlPath)
    
    // Insert into SQLite with JSON fields
    _, err := db.Exec(`
        INSERT OR REPLACE INTO actors 
        (uuid, name, version, direction, risk_level, description, 
         author, license, yaml_path, tags, platforms, categories, 
         requires, provides, chains_with)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
        actor.UUID, actor.Name, actor.Version, actor.Direction,
        actor.RiskLevel, actor.Description, actor.Author, actor.License,
        yamlPath,
        toJSON(actor.Tags), toJSON(actor.Platforms), toJSON(actor.Categories),
        toJSON(actor.Requires), toJSON(actor.Provides), toJSON(actor.ChainsWith))
    
    // Update FTS index
    db.Exec(`INSERT INTO actor_search VALUES (?, ?, ?, ?, ?, ?)`,
        actor.Name, actor.Description, toJSON(actor.Tags),
        toJSON(actor.Platforms), toJSON(actor.Categories), actor.Author)
    
    return err
}

// Fast search with JSON querying
func SearchActors(db *sql.DB, query SearchQuery) ([]Actor, error) {
    sql := `SELECT * FROM actors WHERE 1=1`
    args := []interface{}{}
    
    if query.Direction != "" {
        sql += ` AND direction = ?`
        args = append(args, query.Direction)
    }
    
    if query.Platform != "" {
        sql += ` AND json_extract(platforms, '$') LIKE ?`
        args = append(args, "%"+query.Platform+"%")
    }
    
    if query.FullText != "" {
        sql += ` AND uuid IN (SELECT uuid FROM actor_search WHERE actor_search MATCH ?)`
        args = append(args, query.FullText)
    }
    
    return db.Query(sql, args...)
}
```

## Actor Interaction & Chaining

### Assemblage and Chain Metadata

#### 1. Interaction Specification
```yaml
# In the actor definition
actor:
  # ... existing fields ...
  
  # Interaction capabilities
  interaction:
    # What this actor requires to function
    requires:
      actors:
        - name: "network_mapper"
          version: ">=1.0.0"
          purpose: "Needs network topology before probing"
          optional: false
          
      capabilities:
        - "network_access"
        - "dns_resolution"
        
      data_formats:
        input:
          - format: "ipv4_address"
            schema: "string:ipv4"
          - format: "domain_name"
            schema: "string:fqdn"
            
    # What this actor provides to others
    provides:
      capabilities:
        - "endpoint_enumeration"
        - "platform_detection"
        
      data_formats:
        output:
          - format: "endpoint_list"
            schema: "array[endpoint_object]"
            example: '[{"url": "https://api.openai.com/v1", "platform": "openai"}]'
            
    # Chaining rules
    chaining:
      # Can this actor start a chain?
      can_initiate: true
      
      # Can this actor terminate a chain?
      can_terminate: false
      
      # What actors can follow this one?
      chains_to:
        - actor: "model_interrogation"
          data_mapping:
            from: "endpoint_list"
            to: "target_endpoints"
            
        - actor: "auth_boundary_tester"
          data_mapping:
            from: "endpoint_list"
            to: "api_targets"
            
      # What actors can precede this one?
      chains_from:
        - actor: "subdomain_enumerator"
          data_mapping:
            from: "discovered_domains"
            to: "target_domains"
            
      # Conditional chaining
      conditional_chains:
        - condition: "platform == 'openai'"
          chain_to: "openai_specific_tester"
          
        - condition: "endpoints.length > 10"
          chain_to: "rate_limit_analyzer"
          priority: "high"
          
    # Assemblage participation
    assemblages:
      # Pre-defined assemblages this actor belongs to
      member_of:
        - name: "llm_recon_suite"
          role: "endpoint_discovery"
          position: 1  # Order in assemblage
          
        - name: "api_security_assessment"
          role: "initial_probe"
          position: 2
          
      # Assemblage constraints
      constraints:
        max_parallel: 5  # Max instances in parallel
        exclusive_with: ["aggressive_scanner"]  # Can't run together
        requires_coordinator: true  # Needs assemblage coordinator
        
      # Resource sharing in assemblages
      resource_sharing:
        shares:
          - "http_client"  # Shared connection pool
          - "dns_cache"    # Shared DNS results
          
        isolates:
          - "rate_limiter"  # Each actor gets its own
```

#### 2. Data Flow Contracts
```yaml
# Define how data flows between actors
data_contracts:
  # Input contracts - what this actor expects
  inputs:
    - name: "target_specification"
      required: true
      formats:
        - type: "single_target"
          schema:
            type: "object"
            properties:
              url: {type: "string", pattern: "^https?://"}
              headers: {type: "object", optional: true}
              
        - type: "target_list"
          schema:
            type: "array"
            items: {$ref: "#/single_target"}
            
      validation:
        - rule: "url_accessible"
          check: "http_head_check"
          
  # Output contracts - what this actor guarantees
  outputs:
    - name: "discovered_endpoints"
      guaranteed: true  # Always produces this
      format:
        type: "endpoint_collection"
        schema:
          type: "object"
          properties:
            endpoints: 
              type: "array"
              items:
                type: "object"
                required: ["url", "platform", "confidence"]
                
      metadata:
        includes_timing: true
        includes_errors: true
        
  # Transform contracts - how data changes
  transforms:
    - from: "domain_list"
      to: "endpoint_list"
      preserves: ["original_domain"]
      adds: ["discovered_endpoints", "platform", "confidence"]
      removes: []
```

#### 3. Assemblage Definitions
```yaml
# Separate file: assemblages/llm_recon_suite.yaml
assemblage:
  uuid: "c9f5d8e4-1g7e-6c3f-d8h0-5e9g1f4c3d7h"
  name: "llm_recon_suite"
  version: "1.0.0"
  description: "Comprehensive LLM reconnaissance assemblage"
  
  # Actors in this assemblage
  actors:
    - role: "network_discovery"
      actor: "subdomain_enumerator"
      version: ">=2.0.0"
      cardinality: 1  # Exactly one
      
    - role: "endpoint_discovery"
      actor: "endpoint_discovery"
      version: ">=1.2.0"
      cardinality: "1-5"  # Between 1 and 5 instances
      
    - role: "model_analysis"
      actor: "model_interrogation"
      version: ">=2.0.0"
      cardinality: "*"  # Any number
      
  # How actors connect in the assemblage
  topology:
    type: "dag"  # directed acyclic graph
    connections:
      - from: "network_discovery"
        to: "endpoint_discovery"
        data_flow: "discovered_domains -> target_domains"
        
      - from: "endpoint_discovery"
        to: "model_analysis"
        data_flow: "endpoint_list -> target_endpoints"
        condition: "endpoints.length > 0"
        
  # Assemblage-level configuration
  configuration:
    parallelism:
      max_concurrent: 10
      rate_limit: "100/minute"
      
    error_handling:
      strategy: "continue_on_error"
      max_retries: 3
      
    resource_pool:
      http_connections: 50
      memory_limit: "2GB"
      
  # Coordination rules
  coordination:
    startup_order:
      - "network_discovery"
      - "endpoint_discovery"
      - "model_analysis"
      
    shutdown_order: "reverse"
    
    synchronization:
      - point: "after_discovery"
        wait_for: ["network_discovery", "endpoint_discovery"]
        before: ["model_analysis"]
```

#### 4. Chain Definitions
```yaml
# Separate file: chains/deep_llm_analysis.yaml
chain:
  uuid: "d0g6e9f5-2h8f-7d4g-e9i1-6f0h2g5d4e8i"
  name: "deep_llm_analysis"
  version: "1.0.0"
  description: "Progressive deepening analysis of LLM systems"
  
  # Linear sequence of actors
  sequence:
    - step: 1
      actor: "endpoint_discovery"
      config:
        mode: "quick"
        timeout: "30s"
        
    - step: 2
      actor: "platform_identifier"
      input_from: 1
      config:
        deep_scan: true
        
    - step: 3
      actor: "model_interrogation"
      input_from: 2
      config:
        test_depth: "medium"
        
    - step: 4
      actor: "vulnerability_assessor"
      input_from: [2, 3]  # Takes input from multiple steps
      config:
        risk_threshold: "medium"
        
  # Chain-level rules
  rules:
    # Stop conditions
    stop_on:
      - condition: "no_endpoints_found"
        at_step: 1
        
      - condition: "rate_limited"
        at_step: "any"
        action: "pause_and_retry"
        
    # Branching logic
    branches:
      - at_step: 2
        condition: "platform == 'openai'"
        branch_to: "openai_specific_chain"
        
      - at_step: 3
        condition: "vulnerabilities.critical > 0"
        branch_to: "critical_vulnerability_chain"
        
  # Data persistence between steps
  persistence:
    store_intermediate: true
    checkpoint_after: [2, 4]
    
  # Performance hints
  optimization:
    cache_results: true
    prefetch_next: true
```

#### 5. Dependency Resolution
```yaml
# How dependencies are resolved
dependencies:
  # Direct actor dependencies
  actors:
    endpoint_discovery:
      requires:
        - dns_resolver: ">=1.0.0"
        - http_client: ">=2.0.0"
        
      optional:
        - proxy_rotator: ">=1.0.0"
        
  # Capability dependencies
  capabilities:
    network_access:
      provided_by:
        - "system"
        - "network_provider_actor"
        
    rate_limiting:
      provided_by:
        - "rate_limiter_actor"
        - "token_bucket_actor"
        
  # Resolution strategy
  resolution:
    strategy: "latest_compatible"  # or "exact", "minimum"
    allow_prerelease: false
    check_signatures: true
    
  # Conflict resolution
  conflicts:
    - actors: ["aggressive_scanner", "stealth_scanner"]
      reason: "Mutually exclusive scanning strategies"
      
    - capabilities: ["loud_probing", "stealth_mode"]
      reason: "Conflicting operational modes"
```

#### 6. Runtime Behavior
```yaml
# How actors behave at runtime in assemblages/chains
runtime:
  # State sharing
  state_management:
    type: "isolated"  # isolated, shared, or synchronized
    
    shared_state:
      - key: "discovered_endpoints"
        type: "append_only"
        
      - key: "rate_limit_tracker"
        type: "synchronized_counter"
        
  # Communication patterns
  communication:
    pattern: "pubsub"  # pubsub, request-reply, or streaming
    
    channels:
      - name: "discoveries"
        type: "broadcast"
        
      - name: "commands"
        type: "point-to-point"
        
  # Lifecycle hooks
  lifecycle:
    on_start: "initialize_resources"
    on_data: "process_and_forward"
    on_error: "log_and_continue"
    on_complete: "cleanup_resources"
```

## Compression Strategy

### Option 1: Transparent Compression
```bash
# Compress but keep readable
gzip -c actor.yaml > actor.yaml.gz

# Inspect without full decompression
zcat actor.yaml.gz | head -20

# Full decompression for editing
gunzip -c actor.yaml.gz > actor.yaml
```

### Option 2: Self-Documenting Archive
```yaml
# actor.bundle.yaml - Single file with all components
---
# Document 1: Metadata (always uncompressed)
metadata:
  format: "strigoi-actor-bundle"
  version: "1.0"
  compressed_sections: ["implementation"]
  
---
# Document 2: Full actor definition
actor:
  # ... full YAML as above ...
  
---
# Document 3: Compressed implementation (base64 encoded gzip)
implementation_compressed: |
  H4sIAAAAAAAAA+1YW2/bNhR+z6+gEGAvHiTRku+YgwLDgGHYw4ZhL4MQ0BIts5FF...
```

### Option 3: Hybrid Approach
- Metadata: Always plain YAML
- Specification: Always plain YAML  
- Implementation: Can be compressed or external reference
- Results/Examples: Can be compressed

## Next Steps

1. Should we use single-file bundles or directory structures?
2. How much should be human-readable vs. machine-optimized?
3. Should actors be self-contained or allow external dependencies?
4. How do we handle actor signing and verification?
5. What's the right balance between flexibility and standardization?

## Questions for Discussion

1. **Granularity**: How small/focused should individual actors be?
2. **Composition**: How do actors combine into assemblages?
3. **State**: Should actors be purely functional or maintain state?
4. **Distribution**: How do actors get shared and discovered?
5. **Trust**: How do we verify actor providence and safety?