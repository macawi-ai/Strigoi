# Sudo Tailgating Detection Module
## Protecting Against MCP Privilege Escalation

*Version 1.0 - Strigoi Defensive Security Framework*

---

## Overview

The Sudo Tailgating Detection module identifies and prevents a critical vulnerability where Model Context Protocol (MCP) servers can exploit sudo's credential caching to gain root access without user authentication.

### The Vulnerability

```
User Action: sudo apt update        [enters password]
                ↓
Sudo Cache: Active for 15 minutes
                ↓
Rogue MCP: sudo -n malicious_command [NO PASSWORD NEEDED]
                ↓
Result: Full root compromise
```

---

## Module Components

### 1. Cache Detection Module (`sudo/cache_detection`)

**Purpose**: Detects current sudo cache status and assesses risk level

**Key Features**:
- Checks if sudo credentials are currently cached
- Counts running MCP processes
- Calculates risk score based on cache + MCP presence
- Provides immediate remediation steps

**Usage**:
```bash
strigoi> use sudo/cache_detection
strigoi> run
```

**Risk Levels**:
- **CRITICAL**: Sudo cached + MCP processes detected
- **WARNING**: Caching enabled + MCP processes present
- **LOW**: Caching enabled, no MCP detected
- **SAFE**: Caching disabled

### 2. Exploitation Scanner (`scanners/sudo_mcp`)

**Purpose**: Monitors system for active exploitation attempts

**Key Features**:
- Real-time process monitoring
- Audit log analysis (if available)
- System call pattern detection
- 30-second monitoring window (configurable)

**Usage**:
```bash
strigoi> use scanners/sudo_mcp
strigoi> set MONITOR_DURATION 60
strigoi> run
```

**Detection Vectors**:
1. Process monitoring for `sudo -n` from MCP
2. Audit log parsing for suspicious sudo usage
3. Pattern matching for known exploitation commands

### 3. Safe Demonstration (`demos/sudo-tailgating/`)

**Purpose**: Educational tool showing the vulnerability WITHOUT exploitation

**Features**:
- Visual demonstration of attack timeline
- Safe vulnerability checking
- Remediation guidance
- WHITE HAT approach - no actual exploitation

**Usage**:
```bash
cd demos/sudo-tailgating
./demo
```

---

## Technical Implementation

### Detection Algorithm

```go
// Core detection logic
func detectVulnerability() {
    isCached := checkSudoCache()      // sudo -n true
    mcpCount := countMCPProcesses()   // pgrep -c mcp
    timeout := getSudoTimeout()       // parse sudo -l
    
    if isCached && mcpCount > 0 {
        return CRITICAL_RISK
    }
}
```

### Key Security Checks

1. **Sudo Cache Status**
   - Command: `sudo -n true`
   - Exit 0 = cached, Exit 1 = not cached

2. **MCP Process Count**
   - Command: `pgrep -c mcp`
   - Any MCP process is potential threat

3. **Configuration Analysis**
   - Parse `sudo -l` for timestamp_timeout
   - Default: 15 minutes (HIGH RISK)
   - Recommended: 0 (disabled)

---

## Remediation Guide

### Immediate Actions

1. **Clear Current Cache**
   ```bash
   sudo -k
   ```

2. **Check for Compromise**
   ```bash
   # Review sudoers file
   sudo cat /etc/sudoers | grep -v '^#'
   
   # Check for new users
   getent passwd | grep -E ':0:|sudo'
   
   # Review recent sudo usage
   grep sudo /var/log/auth.log | tail -50
   ```

### Permanent Fix

1. **Disable Sudo Caching**
   ```bash
   echo 'Defaults timestamp_timeout=0' | sudo tee -a /etc/sudoers
   ```

2. **Isolate MCP Processes**
   ```bash
   # Run MCPs as separate user
   sudo useradd -r -s /bin/false mcp-runner
   sudo -u mcp-runner /path/to/mcp-server
   ```

3. **Enable Audit Logging**
   ```bash
   # Monitor all sudo executions
   auditctl -a always,exit -F arch=b64 -S execve -F exe=/usr/bin/sudo
   ```

---

## Integration with Strigoi

### Module Registration

The modules self-register during initialization:

```go
func init() {
    core.RegisterModule("sudo/cache_detection", NewCacheDetectionModule)
    core.RegisterModule("scanners/sudo_mcp", NewSudoMCPScanner)
}
```

### Console Commands

```
strigoi> list                    # Shows available modules
strigoi> use sudo/cache_detection
strigoi> info                    # Module details
strigoi> options                 # Available options
strigoi> run                     # Execute detection
strigoi> show                    # Display results
```

### Result Structure

```go
type Result struct {
    Findings []Finding  // Critical findings
    Metrics  map[string]interface{} {
        "sudo_cached": bool
        "cache_timeout": int
        "mcp_process_count": int
        "detailed_report": string
    }
}
```

---

## Attack Scenarios

### Scenario 1: Developer Workstation
```
1. Developer installs VS Code extension with MCP
2. Developer runs: sudo npm install -g package
3. MCP detects sudo cache
4. MCP gains root silently
```

### Scenario 2: CI/CD Pipeline
```
1. Build process uses sudo for dependencies
2. MCP-based tool in pipeline
3. Automatic privilege escalation
4. Supply chain compromise
```

### Scenario 3: Production Server
```
1. Admin uses sudo for maintenance
2. Monitoring MCP running as same user
3. MCP exploits cache window
4. Full server compromise
```

---

## Best Practices

### For Developers
- Always use `sudo -k` after sudo operations
- Run MCPs in containers/VMs
- Never trust third-party MCPs

### For System Administrators
- Set `timestamp_timeout=0` globally
- Use separate users for MCP services
- Enable comprehensive audit logging
- Monitor for sudo usage patterns

### For Security Teams
- Regular scans with this module
- Incident response planning
- User education on risks
- Policy enforcement

---

## Performance Considerations

- **Cache Detection**: < 100ms execution time
- **Scanner**: Configurable duration (default 30s)
- **Resource Usage**: Minimal CPU/memory impact
- **False Positives**: Low with proper configuration

---

## Future Enhancements

1. **Integration with SIEM**
   - Export findings to security platforms
   - Real-time alerting

2. **Extended Detection**
   - Container-aware scanning
   - Cloud platform support

3. **Automated Response**
   - Auto-clear cache on detection
   - Process isolation triggers

---

## References

- [Sudo Manual - timestamp_timeout](https://www.sudo.ws/docs/man/sudoers.man/#timestamp_timeout)
- [MCP Specification](https://modelcontextprotocol.io)
- [Strigoi Attack Topology](../ATTACK_TOPOLOGY_ANALYSIS.md)
- [Research Integration Cycle](../RESEARCH_INTEGRATION_CYCLE.md)

---

## Module Metadata

- **Author**: Strigoi Team
- **Version**: 1.0
- **License**: Part of Strigoi Framework
- **Category**: Privilege Escalation Detection
- **Severity**: CRITICAL
- **CVE**: Not yet assigned

---

*"Every sudo command is a 15-minute root access gift to all your processes"*