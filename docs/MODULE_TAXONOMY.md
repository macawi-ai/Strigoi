# Strigoi Module Taxonomy
## Agent Protocol Security Testing Framework

Inspired by Metasploit but simplified for agent protocol testing and audit-firm usage.

---

## Core Module Categories

### 1. `discovery/` - Protocol & Agent Discovery
Replaces MSF's complex scanner hierarchy with focused discovery:
```
discovery/
├── protocols/          # Detect which protocols are in use
│   ├── agntcy.ts      # AGNTCY protocol detection
│   ├── mcp.ts         # Model Context Protocol finder
│   ├── openai.ts      # OpenAI Assistants API
│   └── auto.ts        # Auto-detect all protocols
├── agents/            # Find active agents
│   ├── enumerate.ts   # List all agents
│   ├── fingerprint.ts # Identify agent types/versions
│   └── topology.ts    # Map agent relationships
└── inventory/         # Build asset inventory
    ├── from_nessus.ts # Parse Nessus scans
    ├── from_rapid7.ts # Parse Nexpose/InsightVM
    └── builder.ts     # Unified inventory builder
```

### 2. `testing/` - Security Test Modules
Simplified from MSF's exploit/auxiliary split:
```
testing/
├── injections/        # Protocol-specific injection tests
│   ├── prompt/        # Prompt injection variants
│   ├── goal/          # Goal manipulation attacks
│   └── memory/        # Consciousness corruption
├── manipulations/     # Business logic attacks
│   ├── authority/     # Permission escalation
│   ├── identity/      # Agent impersonation
│   └── state/         # State manipulation
├── resilience/        # Stress & DoS testing
│   ├── variety_bomb.ts # Overwhelm with variety
│   ├── rate_limit.ts  # Rate limit testing
│   └── resource.ts    # Resource exhaustion
└── scenarios/         # Multi-step attack chains
    ├── financial/     # Financial system scenarios
    ├── enterprise/    # Enterprise scenarios
    └── custom/        # User-defined chains
```

### 3. `analysis/` - Intelligence & Reporting
Audit-firm focused analysis tools:
```
analysis/
├── parsers/           # Input data parsers
│   ├── nessus.ts     # Parse Nessus XML
│   ├── burp.ts       # Parse Burp Suite
│   └── logs.ts       # Parse agent logs
├── intelligence/      # Threat intelligence
│   ├── patterns.ts    # Attack pattern analysis
│   ├── trends.ts      # Trend identification
│   └── correlation.ts # Cross-protocol correlation
└── reports/           # Report generation
    ├── executive.ts   # C-suite summaries
    ├── technical.ts   # Technical details
    ├── compliance.ts  # Compliance mapping
    └── inventory.ts   # Asset inventory reports
```

### 4. `validation/` - Defensive Validation
Test defensive measures (leads to Domovoi sales):
```
validation/
├── firewalls/         # Test agent firewalls
│   ├── domovoi.ts    # Domovoi-specific tests
│   ├── generic.ts    # Generic firewall tests
│   └── bypass.ts     # Bypass attempts
├── monitoring/        # Detection validation
│   ├── logging.ts    # Log completeness
│   ├── alerts.ts     # Alert accuracy
│   └── siem.ts       # SIEM integration
└── governance/        # Policy validation
    ├── permissions.ts # Permission boundaries
    ├── rate_limits.ts # Rate limit enforcement
    └── audit_trail.ts # Audit trail integrity
```

### 5. `utilities/` - Supporting Tools
Helper modules for common tasks:
```
utilities/
├── encoders/          # Data encoding/obfuscation
├── generators/        # Payload generators
├── proxies/          # Protocol proxies
└── helpers/          # Common utilities
```

---

## Module Naming Conventions

### Clear, Action-Oriented Names
- ✅ `discovery/protocols/agntcy.ts`
- ❌ `auxiliary/scanner/agntcy/agntcy_version.rb`

### Self-Documenting Structure
- ✅ `testing/injections/prompt/context_overflow.ts`
- ❌ `exploits/multi/agent/prompt_injection_2024_001.rb`

### Audit-Friendly Categories
- ✅ `analysis/reports/executive.ts`
- ❌ `post/gather/enum_configs.rb`

---

## CLI Integration

### Interactive Shell (`strigoi>`)
```bash
strigoi> use discovery/protocols/auto
strigoi> set TARGET api.example.com
strigoi> run

[*] Discovering agent protocols on api.example.com...
[+] Found: MCP v1.0 on port 443
[+] Found: AGNTCY v1.0 on port 8443
[!] Potential vulnerability: No rate limiting detected

strigoi> use testing/injections/prompt/basic
strigoi> set PROTOCOL mcp
strigoi> check
[*] Target appears vulnerable to prompt injection
```

### Command Line (`strigoi`)
```bash
# Quick scan with auto-discovery
strigoi discover --target api.example.com --output inventory.json

# Run specific test
strigoi test prompt-injection --protocol mcp --target api.example.com

# Generate executive report
strigoi report executive --input scan-results.json --output board-report.pdf

# Parse Nessus and build inventory
strigoi parse nessus --file scan.nessus --enhance-with-agents
```

---

## Audit Firm Features

### 1. Inventory Building
```typescript
// Parse multiple sources into unified inventory
const inventory = await StrigoiInventory.build({
  sources: [
    { type: 'nessus', file: 'network-scan.nessus' },
    { type: 'agent-discovery', target: 'api.example.com' },
    { type: 'manual', file: 'assets.csv' }
  ],
  enhance: true  // Add agent-specific data
});
```

### 2. Compliance Mapping
```typescript
// Map findings to compliance frameworks
const compliance = await StrigoiCompliance.map({
  findings: scanResults,
  frameworks: ['SOC2', 'ISO27001', 'NIST'],
  includeRemediation: true
});
```

### 3. Executive Reporting
```typescript
// Generate board-ready reports
const report = await StrigoiReport.executive({
  findings: scanResults,
  inventory: assetInventory,
  riskMatrix: true,
  remediation: {
    highlight: 'domovoi',  // Subtle product placement
    costBenefit: true
  }
});
```

---

## Integration with ATLAS

Modules can be used in ATLAS training scenarios:
- Each module becomes a training exercise
- Safe sandbox execution
- Progressive difficulty levels
- Certification tied to module mastery

---

## Benefits Over MSF Approach

1. **Focused**: Agent protocols only, not general exploitation
2. **Modern**: TypeScript, async/await, no Ruby legacy
3. **Intuitive**: Clear categories, obvious naming
4. **Audit-Ready**: Built for compliance and reporting
5. **Educational**: Integrated with ATLAS training

This taxonomy makes Strigoi accessible to IT auditors while maintaining power for security professionals.