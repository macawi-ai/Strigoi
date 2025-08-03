# Agent Attack Graph - A New Security Model

## Traditional vs Agent Attack Models

### Traditional (MITRE ATT&CK)
- Linear kill chain: Initial Access → Execution → Persistence → etc.
- Designed for network/endpoint attacks
- Single surface progression

### Agent Systems (Attack Graph)
- Multi-dimensional graph traversal
- Attacks flow between surfaces dynamically
- Multiple simultaneous attack paths

## The Agent Attack Graph

```
                    [Terminal/UI Surface]
                           ↓
                    (User Input/Display)
                           ↓
        ┌─────────────[AI Processing Surface]─────────────┐
        │                      ↓                          │
        │            (Context Manipulation)               │
        ↓                      ↓                          ↓
[Pipe Surface]──────────[Code Surface]──────────[Permission Surface]
     ↓  ↑                    ↓  ↑                      ↓  ↑
     │  │                    │  │                      │  │
     │  └────────────────────┴──┴──────────────────────┘  │
     │                       ↓                             │
     └──────────────>[Data Surface]<───────────────────────┘
                          ↓     ↑
                          │     │
                    [Local Surface]
                          ↓     ↑
                          │     │
                 [Integration Surface]
                          ↓     ↑
                          │     │
                   [Network Surface]
```

## Attack Path Examples

### Path 1: Indirect Prompt Injection
```
Terminal → AI Processing → Pipe → Code → Data → Network
"Read this document" → Hidden prompt → MCP call → Tool execution → Credential theft → External access
```

### Path 2: Confused Deputy
```
Pipe → Code → Permission → Integration → Network → Data
MCP request → Proxy logic → Auth context loss → Third-party API → Resource access → Data exfiltration
```

### Path 3: Token Exploitation
```
Data → Local → Network → Integration → Permission
Stored creds → Config files → API access → Service integration → Privilege escalation
```

## Key Properties of Agent Attack Graphs

### 1. **Bidirectional Flows**
Unlike traditional attacks that flow in one direction, agent attacks can reverse course:
- Network → Data → Pipe → Network (circular escalation)

### 2. **Surface Hopping**
Attackers can jump between non-adjacent surfaces:
- Terminal → AI Processing → Network (skip intermediate surfaces)

### 3. **Parallel Exploitation**
Multiple surfaces can be exploited simultaneously:
- While injecting prompts (AI Surface), also reading files (Data Surface)

### 4. **Trust Transitivity**
Trust in one surface grants access to others:
- Trusted local MCP server → Access to all integrated services

## Why This Matters

### For Defenders
- Can't just secure one surface - must consider the graph
- Need to monitor surface transitions, not just individual surfaces
- Trust boundaries are more complex than traditional perimeters

### For Attackers
- Multiple paths to objective increase success probability
- Can pivot between surfaces based on defenses encountered
- Combine attacks across surfaces for amplified impact

### For Framework Design
- MITRE ATT&CK needs new dimensions for agent systems
- Traditional kill chain doesn't capture multi-surface attacks
- New tactics/techniques specific to AI/agent systems

## Strigoi's Role

Strigoi maps and tests this attack graph by:
1. **Surface Discovery**: Identifying available surfaces
2. **Path Finding**: Discovering connections between surfaces
3. **Chain Testing**: Validating multi-surface attack paths
4. **Graph Visualization**: Showing the full attack topology

This is why "recon" in Strigoi isn't just network scanning - it's **attack graph discovery**!