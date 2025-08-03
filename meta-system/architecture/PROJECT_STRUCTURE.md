# Strigoi Project Structure - VSM Architecture

## Overview

Strigoi follows the Viable System Model (VSM) for cybernetic organization, ensuring recursive autonomy and adaptive behavior at every level.

## Directory Structure

```
Strigoi/
├── S1-operations/          # System 1: Operational Units
│   ├── validation/         # License validation telemetry
│   ├── core/              # Core attack engine
│   └── modules/           # Attack modules (A2A, MCP, etc.)
│
├── S2-coordination/        # System 2: Anti-oscillation
│   ├── protocols/         # Protocol handlers
│   └── interfaces/        # API and CLI interfaces
│
├── S3-control/            # System 3: Resource Bargaining
│   ├── governors/         # Cybernetic governors
│   └── metrics/          # Performance metrics
│
├── S4-intelligence/       # System 4: Environmental Scanning
│   ├── learning/         # ML-based adaptation
│   └── analytics/        # Attack analytics
│
├── S5-identity/          # System 5: Policy and Identity
│   ├── licensing/        # License management
│   └── ethics/          # Ethics enforcement
│
├── meta-system/          # Governance of Governance
│   ├── architecture/     # System design docs
│   └── documentation/    # User documentation
│
├── tools/                # Supporting Tools
│   ├── scripts/         # Build and deploy scripts
│   └── configs/         # Configuration files
│
└── tests/               # Testing Infrastructure
    ├── unit/           # Unit tests
    └── integration/    # Integration tests
```

## System Descriptions

### S1 - Operations (What We Do)
The operational heart of Strigoi. Each module has local autonomy within ethical constraints:
- **validation/**: DNS-based telemetry for license compliance
- **core/**: Main attack engine with variety amplification
- **modules/**: Specific attack vectors (A2A, MCP, traditional)

### S2 - Coordination (Preventing Chaos)
Ensures different attack modules don't interfere with each other:
- **protocols/**: Standardized communication between modules
- **interfaces/**: Consistent API/CLI for user interaction

### S3 - Control (Resource Management)
Allocates resources and enforces constraints:
- **governors/**: Cybernetic control mechanisms
- **metrics/**: Real-time performance monitoring

### S3* - Audit (Compliance Channel)
Embedded within S3, ensures ethical compliance:
- Monitors all operations for authorization
- Enforces rate limits and scope boundaries
- Reports violations to S5

### S4 - Intelligence (Environmental Awareness)
Adapts to target defenses and learns from responses:
- **learning/**: Pattern recognition and adaptation
- **analytics/**: Attack effectiveness analysis

### S5 - Identity (Purpose and Ethics)
Maintains tool identity and ethical boundaries:
- **licensing/**: License validation and enforcement
- **ethics/**: Hard-coded ethical constraints

### Meta-System (Recursive Governance)
Governs the governance structure itself:
- **architecture/**: System design and evolution
- **documentation/**: Knowledge management

## Information Flows

```
User Input → S2 (Interface) → S3 (Control) → S1 (Operations)
     ↑                            ↓               ↓
     ←  S4 (Intelligence)  ←  Results      Telemetry → S5
```

## Cybernetic Principles

1. **Autonomy**: Each subsystem operates independently
2. **Recursion**: VSM pattern repeats at each level
3. **Variety**: Requisite variety for environmental matching
4. **Feedback**: Continuous learning and adaptation
5. **Ethics**: Hard constraints prevent misuse

## Development Guidelines

1. **Module Independence**: S1 modules should be self-contained
2. **Governor First**: Always implement S3 governors before S1 operations
3. **Ethics by Design**: S5 constraints are non-negotiable
4. **Recursive Structure**: Apply VSM at module level too
5. **Information Ecology**: Consider data flows, not just functions

## File Naming Conventions

- Rust modules: `snake_case.rs`
- Documentation: `UPPER_CASE.md`
- Scripts: `kebab-case.sh`
- Configs: `lower_case.toml`

## Build System

The project uses:
- **Rust**: Core implementation language
- **Cargo**: Build and dependency management
- **Podman**: Containerization
- **Ansible**: Deployment automation

## Testing Philosophy

- **Unit Tests**: Test individual variety amplifiers
- **Integration Tests**: Test governor interactions
- **Ethical Tests**: Verify constraint enforcement
- **Chaos Tests**: Ensure graceful degradation

---

*"The purpose of a system is what it does" - Stafford Beer*

This architecture ensures Strigoi remains a powerful yet ethical security validation tool.