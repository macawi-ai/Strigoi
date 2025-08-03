# Strigoi Quick Start Guide
## Get Testing in 5 Minutes

---

## Prerequisites

- Node.js 20+ installed
- Written authorization to test target systems
- Basic understanding of security testing

---

## Installation

```bash
# Clone the repository (authorized users only)
git clone https://github.com/macawi-ai/strigoi.git
cd strigoi

# Install dependencies
npm install

# Build the project
npm run build
```

---

## First Test - Local Demo

### 1. Start a Test Target

```bash
# In a separate terminal, start our demo MCP server
cd topologies/apps/fedrate-monitor
python3 fedrate_monitor.py
```

### 2. Launch Strigoi

```bash
# In the main directory
npm run dev

# Or for production build
./dist/cli/strigoi.js
```

### 3. Discover Protocols

```bash
strigoi> discover protocols localhost:3000

[*] Probing http://localhost:3000 for MCP...
[+] Found MCP at http://localhost:3000!

✅ Discovered Protocols:
┌──────────┬─────────┬───────────────┬────────────┬────────────────┐
│ Protocol │ Version │ Endpoint      │ Risk Level │ Authentication │
├──────────┼─────────┼───────────────┼────────────┼────────────────┤
│ MCP      │ 1.0     │ localhost:3000│ 🔴 High    │ None          │
└──────────┴─────────┴───────────────┴────────────┴────────────────┘

⚠️  Warning: Found endpoints without authentication!
```

### 4. Run Security Assessment

```bash
strigoi> test security

[*] Running security assessment on localhost:3000...

⚠️  Security Assessment Results:
Target: http://localhost:3000
Severity: HIGH

Findings:
  • No authentication required
    Recommendation: Implement proper authentication mechanism
  • Not using HTTPS
    Recommendation: Enable TLS encryption
```

---

## Testing Real Systems

### Step 1: Get Authorization

Before testing any system:
1. Obtain written permission
2. Define scope clearly
3. Agree on testing windows

### Step 2: Discovery

```bash
strigoi> discover protocols target.example.com

# Strigoi will probe common ports and paths for:
# - MCP endpoints
# - AGNTCY servers
# - Other agent protocols
```

### Step 3: Assessment

```bash
strigoi> test all --target target.example.com

# Runs comprehensive security checks:
# - Authentication verification
# - Encryption validation
# - Rate limit testing
# - Configuration analysis
```

### Step 4: Reporting

```bash
strigoi> report generate --format pdf

# Creates professional report with:
# - Executive summary
# - Technical findings
# - Risk ratings
# - Remediation recommendations
```

---

## Common Commands

| Command | Description |
|---------|-------------|
| `discover protocols <target>` | Find agent protocols |
| `test security` | Run security assessment |
| `test all` | Run all available tests |
| `help` | Show available commands |
| `exit` | Exit Strigoi |

---

## Testing Containerlab Networks

```bash
# Deploy a test network
npm run lab:deploy topologies/first-liberty-bank.clab.yml

# Discover all services in the network
strigoi> discover all 172.20.20.0/24

# Test specific vulnerable services
strigoi> test security fedrate-monitor.first-liberty-bank.io
```

---

## Best Practices

1. **Always Test in Stages**
   - Start with discovery
   - Run passive tests first
   - Active tests only when safe

2. **Document Everything**
   - Use lab notebooks
   - Capture evidence
   - Note timestamps

3. **Minimize Impact**
   - Test during agreed windows
   - Avoid resource exhaustion
   - Stop if systems degrade

4. **Report Professionally**
   - Focus on business impact
   - Provide clear remediation
   - Include positive findings

---

## Troubleshooting

### Connection Errors
- Verify target is reachable
- Check firewall rules
- Ensure correct protocol/port

### No Protocols Found
- Target may use non-standard ports
- Try specific port with `:port`
- Check if service is running

### Permission Denied
- Verify your authorization
- Check scope limitations
- Contact system owner

---

## Need Help?

- Documentation: `/docs` directory
- Lab examples: `/topologies` directory
- Email: support@macawi.ai

---

**Remember: Always test ethically and with authorization!**