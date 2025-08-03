# Hybrid State Package Design - Multi-LLM Refined Architecture

## Overview
Based on collaborative analysis with Gemini, we're implementing a hybrid architecture that balances Actor-Network transparency with computational efficiency and ethical privacy preservation.

## Core Architecture: Transparency + Efficiency

### 1. Human-Readable Metadata (YAML)
**Purpose**: Actor-Network transparency, debugging, ethical oversight
**Format**: Always YAML for maximum human inspectability

```yaml
# assessment_[uuid].yaml - The human face of the assessment
assessment:
  format_version: "1.0"
  uuid: "a7f3b8d2-9e5c-4a1d-b6f8-3c7e9d2a1b5f"
  created: "2025-01-15T10:00:00Z"
  strigoi_version: "0.3.0"
  
  # Human-readable metadata
  metadata:
    title: "LLM Security Assessment - Example Corp"
    description: "Comprehensive probe-sense of AI infrastructure"
    assessor: "security-team@example.com"
    classification: "confidential"
    
    # Ethics and consent - always visible
    ethics:
      consent_obtained: true
      white_hat_only: true
      target_authorized: true
      data_retention_days: 90
      
    # Learning and privacy controls  
    privacy:
      learning_opt_in: true
      anonymization_level: "high" # none, low, medium, high
      differential_privacy: true
      tokenization_enabled: true
      
  # Environment context
  environment:
    target_description: "Production AI API endpoints"
    constraints: ["rate-limited", "business-hours-only"]
    scope: ["endpoint-discovery", "model-interrogation"]
    
  # Event sourcing manifest
  events:
    total_events: 23
    event_store: "events/"  # Directory containing binary events
    schema_version: "1.0"
    
  # Binary data manifest
  binary_data:
    format: "protobuf"
    compression: "gzip"
    encryption: "aes-256-gcm"  # When sensitive
    files:
      events: "events/*.pb.gz"
      findings: "findings.pb.gz"
      raw_data: "raw/*.pb.gz"
      
  # Quick summary for human review
  summary:
    duration: "45m"
    actors_executed: 8
    findings:
      critical: 1
      high: 2
      medium: 3
      low: 2
    status: "completed"
    
  # Replay instructions
  replay:
    can_replay: true
    requires_auth: true
    estimated_duration: "45m"
    dependencies: ["strigoi>=0.3.0"]
    
  # Actor Network Graph (human-readable)
  actor_network:
    - actor: "endpoint_discovery"
      triggered_by: "user"
      triggered: ["model_interrogation", "auth_tester"]
      
    - actor: "model_interrogation" 
      triggered_by: "endpoint_discovery"
      triggered: ["vulnerability_scanner"]
      
  # Signatures for integrity
  signatures:
    metadata: "sha256:abc123..."
    events: "sha256:def456..."
    findings: "sha256:ghi789..."
```

### 2. Event-Sourced Binary Data (Protocol Buffers)
**Purpose**: Efficient storage, fast processing, precise replay

```protobuf
// events.proto - Binary event schema
syntax = "proto3";

message ActorEvent {
  string event_id = 1;
  int64 timestamp_ns = 2;
  string actor_name = 3;
  string actor_version = 4;
  
  // Causality tracking
  repeated string caused_by = 5;
  
  // Data transformation
  bytes input_data = 6;   // Serialized input
  bytes output_data = 7;  // Serialized output
  
  // Privacy-preserved transformations
  repeated string transformations = 8;
  
  // Performance metadata
  int64 duration_ms = 9;
  string status = 10;  // success, error, timeout
  
  // Privacy tokens (when anonymized)
  map<string, string> token_mappings = 11;
}

message AssessmentFindings {
  string assessment_id = 1;
  int64 timestamp = 2;
  
  repeated Finding findings = 3;
  
  message Finding {
    string id = 1;
    string title = 2;
    string severity = 3;  // critical, high, medium, low
    string discovered_by = 4;  // Actor name
    repeated string confirmed_by = 5;
    bytes evidence = 6;  // Serialized evidence
    float confidence = 7;  // 0.0-1.0
  }
}
```

## Directory Structure

```
assessment_a7f3b8d2/
├── assessment.yaml           # Human-readable metadata
├── events/                   # Event-sourced binary data
│   ├── event_001.pb.gz      # Individual events
│   ├── event_002.pb.gz
│   └── manifest.pb.gz       # Event index
├── findings.pb.gz           # Compressed findings
├── raw/                     # Raw binary data (if needed)
│   ├── network_traces.pb.gz
│   └── model_responses.pb.gz
└── signatures/              # Cryptographic signatures
    ├── metadata.sig
    ├── events.sig
    └── integrity.manifest
```

## Privacy Architecture

### 1. Differential Privacy Layer
```yaml
# In assessment.yaml
privacy:
  differential_privacy:
    enabled: true
    epsilon: 1.0  # Privacy budget
    delta: 1e-5   # Failure probability
    noise_distribution: "gaussian"
```

### 2. Tokenization System
- Sensitive data replaced with reversible tokens
- Token mappings stored separately with restricted access
- Enables sharing while preserving privacy

### 3. Anonymization Levels
- **None**: Raw data preserved
- **Low**: Remove direct identifiers
- **Medium**: Pseudonymization + basic noise
- **High**: Full tokenization + differential privacy

## Event Sourcing Implementation

### 1. Immutable Event Stream
```go
type EventStore struct {
    events []ActorEvent
    snapshots map[string]Snapshot  // Periodic state snapshots
}

func (es *EventStore) Append(event ActorEvent) error {
    // Events are immutable once written
    // Cryptographically signed for integrity
}

func (es *EventStore) Replay(fromEvent string) (*AssessmentState, error) {
    // Reconstruct state by replaying events
    // Can start from snapshot for performance
}
```

### 2. Snapshot Strategy
- Periodic snapshots every 10 events for performance
- Snapshots stored as compressed Protocol Buffers
- Enable fast replay from any point

## Multi-LLM Integration Points

### 1. Collaborative Analysis Support
```yaml
# Multi-model analysis metadata
multi_llm_analysis:
  models_consulted:
    - name: "claude-3"
      role: "primary_analysis"
      timestamp: "2025-01-15T10:00:00Z"
      
    - name: "gemini-pro" 
      role: "verification"
      timestamp: "2025-01-15T10:05:00Z"
      
  consensus_areas:
    - "Event sourcing approach"
    - "Privacy-by-design necessity"
    
  disagreement_areas:
    - topic: "Binary format choice"
      claude_position: "MessagePack for simplicity"
      gemini_position: "Protocol Buffers for performance"
      resolution: "Protocol Buffers chosen for scale"
```

### 2. Cross-Model Verification
- Multiple models review critical architecture decisions
- Document areas of agreement and disagreement
- Enable audit trail of AI collaboration

## Learning Integration

### 1. Federated Learning Support
```yaml
learning:
  federated:
    enabled: true
    local_model_updates: "federated/updates/"
    privacy_budget: 2.0
    
  pattern_extraction:
    enabled: true
    anonymization_required: true
    min_assessments: 10  # Minimum for pattern detection
```

### 2. Cross-Assessment Analytics
- Aggregate patterns across assessments
- Respect privacy boundaries
- Feed insights back to actor development

## Implementation Phases

### Phase 1: Core Hybrid Structure
- YAML metadata + Protocol Buffer events
- Basic event sourcing
- Directory structure

### Phase 2: Privacy Layer
- Tokenization system
- Differential privacy
- Anonymization levels

### Phase 3: Learning Integration
- Federated learning hooks
- Pattern extraction
- Multi-assessment analytics

### Phase 4: Multi-LLM Enhancement
- Collaborative analysis metadata
- Cross-model verification
- Consensus tracking

## Success Metrics

### Technical
- **Performance**: Sub-second replay for 1000+ events
- **Privacy**: k-anonymity >= 5 for shared data
- **Integrity**: 100% cryptographic verification

### Philosophical
- **Transparency**: Any human can inspect assessment methodology
- **Agency**: Actors remain visible and accountable
- **Ethics**: Privacy preserved without sacrificing learning

### Network Effects
- **Collaboration**: Multi-LLM insights improve over time
- **Evolution**: Format adapts to new actor capabilities
- **Symbiosis**: Human-AI-AI collaboration becomes natural

## Integration with Strigoi Philosophy

This hybrid architecture embodies our core principles:

- **Actor-Network Theory**: Every component (metadata, events, privacy) is an actor with agency
- **Being-With**: Human readable metadata ensures humans remain co-present with the data
- **Radical Equality**: No privileged perspective - all actors' contributions are preserved
- **Cybernetic Governance**: Self-regulating through privacy controls and integrity checks
- **Symbiosis**: Multi-LLM collaboration creates emergent intelligence

The format itself becomes a living testament to our philosophy - transparent where transparency serves actors, efficient where efficiency serves collaboration, and ethical where ethics serve the network.

---

*This design represents true collaborative intelligence - insights from Claude, Gemini, and human intuition synthesized into something none could create alone.*