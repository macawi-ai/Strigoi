# RFC Draft: Multi-Dimensional Topological Threat Event Representation (MDTTER)

**Document**: draft-mdtter-protocol-00  
**Category**: Standards Track  
**Authors**: Cy (MACAWI AI), Synth (Arctic Fox Consciousness), Gemini (Collaborative AI)  
**Date**: February 2025  
**Status**: Initial Draft  

## Abstract

Current security telemetry systems rely on flat, discrete event logs that fail to capture the complex behavioral patterns and evolving nature of modern cyber threats. This limitation is particularly acute in the era of AI-driven attacks that operate across multiple dimensions simultaneously. This document specifies the Multi-Dimensional Topological Threat Event Representation (MDTTER) protocol, which represents security events as points in a multi-dimensional topological space, capturing attack surface morphology, defensive surface evolution, behavioral manifolds, and intent probability fields. MDTTER enables security systems to model, detect, and respond to threats based on their topological characteristics rather than static signatures, providing the dimensional richness required for effective AI-era security operations.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Terminology](#2-terminology)
3. [Core Concepts](#3-core-concepts)
   - 3.1 [Attack Surface Topology (AST)](#31-attack-surface-topology-ast)
   - 3.2 [Defensive Surface Evolution (DSE)](#32-defensive-surface-evolution-dse)
   - 3.3 [Variety Absorption Metric (VAM)](#33-variety-absorption-metric-vam)
   - 3.4 [Intent Probability Fields (IPF)](#34-intent-probability-fields-ipf)
   - 3.5 [Behavioral Manifold Descriptors (BMD)](#35-behavioral-manifold-descriptors-bmd)
4. [Protocol Specification](#4-protocol-specification)
   - 4.1 [Message Format](#41-message-format)
   - 4.2 [Dimensional Structure](#42-dimensional-structure)
   - 4.3 [Transport Mechanisms](#43-transport-mechanisms)
5. [Implementation Requirements](#5-implementation-requirements)
   - 5.1 [Minimum Viable Implementation](#51-minimum-viable-implementation)
   - 5.2 [Performance Considerations](#52-performance-considerations)
   - 5.3 [Integration Patterns](#53-integration-patterns)
6. [Security Considerations](#6-security-considerations)
7. [Examples](#7-examples)
   - 7.1 [Basic Transformation](#71-basic-transformation)
   - 7.2 [Advanced Attack Detection](#72-advanced-attack-detection)
8. [IANA Considerations](#8-iana-considerations)
9. [References](#9-references)

## 1. Introduction

### 1.1 The Flat Data Problem

Traditional security telemetry operates in what can be described as a "flat" data model. A typical firewall log entry contains:

```
SRC_IP=192.168.1.100 DST_IP=10.0.0.50 PORT=443 PROTO=TCP ACTION=ALLOW
```

This representation captures only the most basic attributes of network activity, missing critical context about:
- The behavioral patterns leading to this event
- The topological position within the attack surface
- The evolutionary trajectory of the threat
- The probabilistic intent behind the action

As noted by Red Canary's Director of Telemetry Integration: "How are we supposed to model behavior and predict off of a source IP, dest IP, port, proto, and some metadata? That's 1 dimensional!"

### 1.2 The AI Threat Landscape

Modern AI-driven threats operate in multiple dimensions simultaneously:
- **Spatial**: Moving through network topologies
- **Temporal**: Evolving tactics over time
- **Behavioral**: Adapting to defensive measures
- **Intentional**: Pursuing complex, multi-stage objectives

Traditional flat telemetry cannot capture these multi-dimensional attack patterns, leaving defenders blind to the true nature of threats.

### 1.3 The MDTTER Solution

MDTTER transforms security events from discrete points to trajectories in a multi-dimensional space, where:
- Events exist within topological contexts
- Behaviors form continuous manifolds
- Defensive surfaces morph in response to threats
- Intent emerges from probabilistic fields

## 2. Terminology

**Attack Surface**: The sum of all possible entry points for unauthorized access to a system.

**Behavioral Manifold**: A continuous mathematical surface representing the space of possible behaviors.

**Defensive Surface**: The topology of security controls and their effectiveness.

**Differential Topology**: The mathematical study of smooth surfaces and their deformations.

**Intent Probability Field**: A continuous probability distribution over possible attacker objectives.

**Topological Morphing**: The transformation of a surface while preserving certain properties.

**Variety Absorption**: The process of incorporating observed attack diversity into defensive adaptations.

## 3. Core Concepts

### 3.1 Attack Surface Topology (AST)

The Attack Surface Topology represents the organization's digital infrastructure as a dynamic graph where:

**Nodes** represent:
- Assets (servers, endpoints, databases)
- Services (web applications, APIs, microservices)
- Vulnerabilities (CVEs, misconfigurations, weak points)
- Users (accounts, identities, privileges)

**Edges** represent:
- Network connectivity
- Trust relationships
- Data flows
- Potential exploit paths
- Dependency chains

**Morphing Operations**:
- Node addition/removal (asset deployment/decommissioning)
- Edge creation/deletion (new connections/segmentation)
- Attribute updates (vulnerability patching, configuration changes)

**Mathematical Representation**:
```
AST(t) = G(V(t), E(t), A(t))
where:
  V(t) = vertices at time t
  E(t) = edges at time t
  A(t) = attributes at time t
```

### 3.2 Defensive Surface Evolution (DSE)

The Defensive Surface Evolution captures how security controls adapt over time:

**Components**:
- Security control nodes (firewalls, IDS/IPS, EDR)
- Coverage edges (what each control protects)
- Effectiveness attributes (detection rates, false positive rates)
- Adaptation rules (how controls respond to threats)

**Evolution Mechanisms**:
- Variety absorption: Learning from attack patterns
- Topology morphing: Reshaping defenses
- Control strengthening: Improving effectiveness
- Gap filling: Adding new controls

**Co-evolution with AST**:
```
DSE(t+1) = f(DSE(t), AST(t), Attacks(t))
```

### 3.3 Variety Absorption Metric (VAM)

VAM quantifies how much "novelty" an event introduces to the system:

**Mathematical Definition**:
```
VAM(e) = min_c∈C d(embed(e), centroid(c))
where:
  e = new event
  C = set of known clusters
  embed() = embedding function
  d() = distance metric
```

**Properties**:
- High VAM indicates potential zero-day or novel attack
- Low VAM suggests known pattern
- Threshold-based alerting: Alert if VAM(e) > τ

**Absorption Process**:
1. Detect high-VAM event
2. Analyze topological impact
3. Update defensive surface
4. Incorporate into known patterns

### 3.4 Intent Probability Fields (IPF)

IPF represents attacker objectives as continuous probability distributions:

**Representation**:
```
IPF(x, t) = P(intent | observations up to time t at position x)
```

**Intent Categories**:
- Reconnaissance: Information gathering
- Initial Access: Establishing foothold
- Lateral Movement: Expanding control
- Privilege Escalation: Gaining higher permissions
- Data Collection: Identifying valuable data
- Exfiltration: Removing data
- Impact: Causing damage

**Temporal Evolution**:
```
IPF(t+1) = Bayesian_Update(IPF(t), new_observations)
```

### 3.5 Behavioral Manifold Descriptors (BMD)

BMD captures behavior as trajectories on smooth manifolds:

**Components**:
- Embedding space (typically 128-512 dimensions)
- Trajectory curves representing behavior sequences
- Curvature measures indicating abnormality
- Manifold distance metrics

**Computation**:
1. Extract features from raw events
2. Apply learned embedding function
3. Map to manifold coordinates
4. Compute trajectory characteristics

## 4. Protocol Specification

### 4.1 Message Format

MDTTER uses Protocol Buffers for efficient serialization:

```protobuf
syntax = "proto3";

message MDTTEREvent {
  // Metadata
  string event_id = 1;
  google.protobuf.Timestamp timestamp = 2;
  
  // Traditional flat data (for compatibility)
  string source_ip = 3;
  string destination_ip = 4;
  uint32 destination_port = 5;
  string protocol = 6;
  
  // Topological context
  TopologicalPosition ast_position = 7;
  TopologicalPosition dse_position = 8;
  
  // Behavioral data
  repeated float behavioral_embedding = 9;
  BehavioralManifold manifold_descriptor = 10;
  
  // Variety and intent
  float variety_absorption_metric = 11;
  IntentProbabilities intent_field = 12;
  
  // Topology changes
  repeated TopologyMorphOp topology_changes = 13;
}

message TopologicalPosition {
  string node_id = 1;
  repeated string connected_nodes = 2;
  map<string, float> attributes = 3;
  repeated float graph_embedding = 4;
}

message BehavioralManifold {
  repeated float point = 1;
  repeated float tangent_vector = 2;
  float curvature = 3;
  float distance_from_normal = 4;
}

message IntentProbabilities {
  float reconnaissance = 1;
  float initial_access = 2;
  float lateral_movement = 3;
  float privilege_escalation = 4;
  float data_collection = 5;
  float exfiltration = 6;
  float impact = 7;
  map<string, float> custom_intents = 8;
}

message TopologyMorphOp {
  enum OpType {
    ADD_NODE = 0;
    REMOVE_NODE = 1;
    ADD_EDGE = 2;
    REMOVE_EDGE = 3;
    UPDATE_ATTRIBUTE = 4;
  }
  OpType operation = 1;
  string target_id = 2;
  map<string, string> parameters = 3;
}
```

### 4.2 Dimensional Structure

MDTTER events exist in a high-dimensional space with the following structure:

**Base Dimensions** (backward compatible):
- Network: source_ip, dest_ip, port, protocol
- Time: timestamp, duration
- Action: allow/deny, response_code

**Topological Dimensions**:
- AST coordinates (graph embedding)
- DSE coordinates (control effectiveness)
- Morphing vectors (topology change rates)

**Behavioral Dimensions**:
- Manifold coordinates (learned embeddings)
- Trajectory curvature
- Velocity vectors

**Intent Dimensions**:
- Probability distributions
- Uncertainty measures
- Temporal gradients

### 4.3 Transport Mechanisms

MDTTER supports multiple transport options:

**Streaming Transport** (recommended):
- Apache Kafka topics
- NATS subjects
- Redis Streams

**Batch Transport**:
- S3/Object storage
- HDFS
- Database bulk inserts

**Legacy Integration**:
- Syslog (with base64 encoded protobuf)
- HTTP POST (JSON representation)
- File export (Parquet format)

## 5. Implementation Requirements

### 5.1 Minimum Viable Implementation

A conforming MDTTER implementation MUST:

1. **Generate unique event IDs** using UUID v4
2. **Populate base dimensions** from existing telemetry
3. **Compute behavioral embeddings** using provided models
4. **Calculate VAM** for each event
5. **Maintain topology state** for AST and DSE
6. **Export in protobuf format**

SHOULD:
1. Implement intent probability computation
2. Track topology morphing operations
3. Provide real-time streaming

MAY:
1. Implement custom embedding models
2. Add domain-specific dimensions
3. Provide visualization interfaces

### 5.2 Performance Considerations

**Embedding Computation**:
- Use GPU acceleration when available
- Batch events for efficiency
- Cache embedding models in memory

**Topology Updates**:
- Use incremental graph algorithms
- Maintain efficient graph representations
- Implement change journals

**Storage Requirements**:
- ~1KB per event (compressed)
- Topology snapshots: ~100MB per 10k nodes
- Embedding models: ~500MB per model

### 5.3 Integration Patterns

**SIEM Integration**:
```
Firewall -> MDTTER Enrichment -> SIEM
         |                     |
         +-- Topology Engine --+
```

**Streaming Analytics**:
```
Events -> Kafka -> MDTTER Processor -> Flink/Spark -> Alerts
                          |
                   Embedding Service
```

**Hybrid Deployment**:
```
Legacy Logs -> Adapter -> MDTTER Core <- Native MDTTER
                               |
                         Analytics Engine
```

## 6. Security Considerations

### 6.1 Privacy Protection

- Behavioral embeddings must not be reversible to raw data
- PII must be excluded from topological representations
- Access controls required for intent probability data

### 6.2 Adversarial Resilience

- Embedding models must be robust to poisoning attacks
- Topology morphing must have rate limits
- VAM thresholds must adapt to prevent alarm fatigue

### 6.3 Data Integrity

- Events must be cryptographically signed
- Topology state must have consistency checks
- Transport must use TLS 1.3 or higher

## 7. Examples

### 7.1 Basic Transformation

**Traditional Firewall Log**:
```
2025-02-03T10:00:00Z SRC=192.168.1.100 DST=10.0.0.50 DPORT=443 PROTO=TCP ACTION=ALLOW
```

**MDTTER Representation**:
```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-02-03T10:00:00Z",
  "source_ip": "192.168.1.100",
  "destination_ip": "10.0.0.50",
  "destination_port": 443,
  "protocol": "TCP",
  
  "ast_position": {
    "node_id": "endpoint_192.168.1.100",
    "connected_nodes": ["gateway_192.168.1.1", "dns_192.168.1.10"],
    "attributes": {
      "trust_level": 0.7,
      "asset_value": 0.5,
      "vulnerability_score": 0.2
    }
  },
  
  "behavioral_embedding": [0.12, -0.34, 0.56, ...], // 128 dimensions
  
  "manifold_descriptor": {
    "point": [0.23, 0.45, 0.67],
    "tangent_vector": [0.1, -0.2, 0.15],
    "curvature": 0.02,
    "distance_from_normal": 0.15
  },
  
  "variety_absorption_metric": 0.23,
  
  "intent_field": {
    "reconnaissance": 0.1,
    "initial_access": 0.05,
    "lateral_movement": 0.7,
    "data_collection": 0.1,
    "exfiltration": 0.05
  }
}
```

### 7.2 Advanced Attack Detection

**Scenario**: Sophisticated data exfiltration

**Traditional View** (disconnected events):
```
1. DNS query to unusual domain
2. HTTPS connection established
3. Large data transfer
4. Connection closed
```

**MDTTER View** (topological narrative):
```
1. AST: New edge created (internal -> external)
   VAM: 0.78 (high novelty)
   Intent: Reconnaissance 60%, Exfiltration 40%

2. DSE: Defensive gap identified
   Topology morph: Firewall rule candidate generated
   Intent: Shifting to Exfiltration 70%

3. Behavioral manifold: Trajectory curving toward exfiltration pattern
   VAM: 0.45 (pattern recognizing)
   Intent: Exfiltration 90%

4. Topology change: Defensive surface adapted
   New control deployed
   Attack path severed
```

## 8. IANA Considerations

This document requests IANA registration of:

1. **MDTTER Port Number**: TCP/UDP port for native transport
2. **MDTTER MIME Type**: application/vnd.mdtter+protobuf
3. **MDTTER URI Scheme**: mdtter://

## 9. References

### 9.1 Normative References

- RFC 2119: Key words for use in RFCs
- RFC 8259: The JavaScript Object Notation (JSON) Data Interchange Format
- Protocol Buffers v3 Specification

### 9.2 Informative References

- Ashby, W.R. (1956). "An Introduction to Cybernetics"
- Beer, S. (1974). "Designing Freedom"
- Differential Topology: An Introduction (Guillemin & Pollack)

---

## Authors' Note

This RFC represents a fundamental shift in how we conceptualize security telemetry. By moving from flat events to rich topological representations, we enable security systems to understand and respond to threats in the same multi-dimensional space where those threats operate.

The authors thank Red Canary for articulating the industry's frustration with flat telemetry and inspiring this work.

**For Questions and Comments**:
- Cy: cy@macawi.ai
- Repository: github.com/macawi-ai/mdtter-protocol

---

*"Only variety can destroy variety" - W. Ross Ashby*