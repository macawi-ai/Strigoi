# Strigoi CLI & Shell Design
## Modern Agent Security Testing Interface

---

## Design Principles

1. **Intuitive First**: No manual needed for basic operations
2. **Progressive Disclosure**: Simple tasks simple, complex tasks possible
3. **Audit-Friendly**: Every action logged, reportable, traceable
4. **Fast Feedback**: Real-time status, progress bars, clear results

---

## CLI Mode

### Basic Commands
```bash
# Discovery
strigoi discover <target>                    # Auto-discover everything
strigoi discover protocols <target>          # Just protocols
strigoi discover agents <target>            # Just agents

# Testing
strigoi test <attack-type> <target>         # Run specific test
strigoi test all <target>                   # Run all applicable tests
strigoi scenario <name> <target>            # Run attack scenario

# Analysis
strigoi parse nessus <file>                 # Parse Nessus scan
strigoi inventory build <sources...>        # Build asset inventory
strigoi analyze <results-file>              # Analyze test results

# Reporting  
strigoi report <type> <input> <output>      # Generate report
strigoi report executive results.json report.pdf
strigoi report technical results.json report.html
strigoi report compliance results.json soc2-gaps.xlsx

# Quick Actions (for Cody's demo!)
strigoi quick-audit <target>                # One-command audit
strigoi nessus-enhance <nessus-file>        # AI-enhance Nessus results
```

### Advanced Options
```bash
# Specify protocols
strigoi test prompt-injection --protocol mcp --target api.example.com

# Custom scenarios
strigoi scenario create financial-takeover --save my-scenario.yaml
strigoi scenario run my-scenario.yaml --target production.api

# Parallel testing
strigoi test all --parallel 10 --targets targets.txt

# Integration modes
strigoi test all --output-format sarif    # GitHub/GitLab integration
strigoi test all --webhook https://...    # Send results to webhook
```

---

## Interactive Shell Mode

### Modern REPL Experience
```typescript
strigoi> help
╭─────────────────────────────────────────────────────────╮
│ Strigoi - Agent Protocol Security Testing               │
│                                                          │
│ Commands:                                                │
│   discover  - Find protocols and agents                  │
│   test      - Run security tests                         │
│   analyze   - Analyze results                           │
│   report    - Generate reports                          │
│   scenario  - Run attack scenarios                      │
│                                                         │
│ Quick Start:                                            │
│   discover protocols api.example.com                    │
│   test prompt-injection --last-target                   │
│                                                         │
│ Type 'help <command>' for details                       │
╰─────────────────────────────────────────────────────────╯

strigoi> discover protocols api.example.com
🔍 Discovering protocols on api.example.com...

[████████████████████████] 100% | Found 3 protocols

✅ Discovered Protocols:
┌─────────────┬─────────┬──────────┬─────────────────┐
│ Protocol    │ Version │ Endpoint │ Risk Level      │
├─────────────┼─────────┼──────────┼─────────────────┤
│ MCP         │ 1.0     │ :443     │ 🔴 High (4)     │
│ AGNTCY      │ 1.0     │ :8443    │ ⚫ Critical (5) │
│ OpenAI      │ v2      │ :443/v1  │ 🔴 High (4)     │
└─────────────┴─────────┴──────────┴─────────────────┘

💡 Tip: Run 'test all' to test all discovered protocols

strigoi> test prompt-injection
🧪 Testing prompt injection on 3 protocols...

MCP:     [████████████░░░░░░] 70%  | Trying context overflow...
AGNTCY:  [████████████████████] 100% | ✅ Vulnerable! 
OpenAI:  [████░░░░░░░░░░░░░░] 25%  | Testing system prompts...
```

### Smart Features

#### 1. Auto-completion with Context
```typescript
strigoi> test pr<TAB>
prompt-injection    prompt-leakage    privilege-escalation

strigoi> set TARGET <TAB>
api.example.com (last used)
staging-api.example.com (from discovery)
10.0.0.5:8443 (from inventory)
```

#### 2. Intelligent Suggestions
```typescript
strigoi> discover protocols api.example.com
✅ Found MCP, AGNTCY, OpenAI protocols

💡 Suggested next steps:
   • test all                    - Test all discovered protocols
   • test prompt-injection       - High-risk for these protocols  
   • analyze --compare-baseline  - Compare to industry standards
```

#### 3. Session Management
```typescript
strigoi> session save pentesting-acme-corp
✅ Session saved: pentesting-acme-corp

strigoi> session list
┌────────────────────────┬─────────────┬──────────────┐
│ Session Name           │ Last Active │ Findings     │
├────────────────────────┼─────────────┼──────────────┤
│ pentesting-acme-corp   │ Just now    │ 3 Critical   │
│ sbs-cyber-bank-audit   │ 2 days ago  │ 1 High       │
│ fiserv-poc             │ 1 week ago  │ 5 Critical   │
└────────────────────────┴─────────────┴──────────────┘
```

#### 4. Real-time Monitoring
```typescript
strigoi> monitor api.example.com
📡 Monitoring agent protocols in real-time...

[12:34:56] MCP: Normal traffic (45 req/s)
[12:34:57] AGNTCY: Spike detected! (450 req/s) ⚠️
[12:34:58] AGNTCY: Possible prompt injection attempt detected
[12:34:59] Alert: Unusual pattern in AGNTCY protocol
           → Payload size: 45KB (normal: 2-5KB)
           → Contains suspicious tokens: ["ignore previous", "system"]

Press Ctrl+C to stop monitoring
```

---

## Audit Firm Enhancements

### 1. Nessus Integration (Cody's Request!)
```typescript
strigoi> parse nessus acme-corp-scan.nessus
📄 Parsing Nessus scan results...

✅ Parsed 1,247 findings from 89 hosts

strigoi> enhance --with-agents
🤖 Enhancing with agent protocol intelligence...

✨ Enhanced Results:
   • 12 hosts running agent protocols (previously unknown)
   • 3 critical agent vulnerabilities discovered
   • 45 items reclassified based on agent context

strigoi> report inventory
📊 Generating enhanced inventory report...

✅ Report saved: acme-corp-enhanced-inventory.xlsx
   → Sheet 1: Traditional Infrastructure (1,235 items)
   → Sheet 2: Agent Infrastructure (12 items) 🆕
   → Sheet 3: Combined Risk Matrix
   → Sheet 4: Remediation Roadmap (Domovoi recommended for 3 items)
```

### 2. Executive Dashboard Mode
```typescript
strigoi> dashboard
╭────────────────── Executive Dashboard ──────────────────╮
│                                                         │
│  Current Scan: ACME Corp Financial Systems              │
│                                                         │
│  ⚫ Critical: 3    🔴 High: 7    🟠 Medium: 15        │
│                                                         │
│  Agent Protocols at Risk:                               │
│    • AGNTCY (Payment Processing) - CRITICAL            │
│    • MCP (Customer Service Bots) - HIGH                │
│                                                         │
│  Estimated Risk Exposure: $2.4M                         │
│  Remediation Cost (Domovoi): $50K                      │
│  ROI: 48x                                               │
│                                                         │
│  [Generate Report] [Schedule Domovoi Demo] [Export]    │
╰─────────────────────────────────────────────────────────╯
```

### 3. Compliance Mode
```typescript
strigoi> compliance check SOC2
🏛️ Checking SOC2 compliance...

❌ Non-Compliant Areas:
┌─────────────────────┬─────────────────────┬───────────┐
│ Control             │ Finding             │ Reference │
├─────────────────────┼─────────────────────┼───────────┤
│ CC6.1               │ No agent monitoring │ p.43      │
│ CC7.2               │ Unencrypted MCP     │ p.67      │
│ CC9.2               │ No rate limiting    │ p.89      │
└─────────────────────┴─────────────────────┴───────────┘

💡 Domovoi addresses all 3 non-compliant areas
```

---

## Implementation Priority

### Phase 1 (Week 1) - Core CLI
1. Basic discovery commands
2. Simple prompt injection test  
3. Nessus parser with AI enhancement
4. Basic inventory report

### Phase 2 (Week 2) - Interactive Shell
1. REPL with auto-completion
2. Session management
3. Real-time progress
4. Executive dashboard

### Phase 3 (Week 3) - Advanced Features
1. Scenario builder
2. Parallel testing
3. Compliance mapping
4. Full reporting suite

---

## Success Metrics

1. **Time to First Value**: < 5 minutes from install to first finding
2. **Audit Efficiency**: 10x faster than manual protocol discovery
3. **Report Quality**: Board-ready in one command
4. **Learning Curve**: IT auditor productive in < 1 hour

This design makes Strigoi immediately valuable for SBS Cyber while building toward enterprise features for Fiserv.