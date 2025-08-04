# Center Module Quick Start Guide

## Overview
The Center module provides real-time STDIO stream monitoring and vulnerability detection for Strigoi. It operates at the center of the compass, analyzing data flows through processes.

## Key Features
- **Real-time monitoring**: Live vulnerability detection as data flows
- **Multi-protocol support**: JSON, SQL, plaintext, and more
- **User-level operation**: No root required (with optional elevated mode)
- **Continuous operation**: Designed for long-duration monitoring
- **Export capabilities**: Serial Studio, Wireshark, structured logs

## Quick Start

### Basic Monitoring
```bash
# Monitor a process by name
strigoi probe center monitor nginx

# Monitor by PID
strigoi probe center monitor --pid 12345

# Monitor with logging
strigoi probe center monitor nginx --output vulns.jsonl
```

### What You'll See
```
═══════════════ Strigoi Center - Stream Monitor ═══════════════
Target: nginx (PID: 12345) | Mode: User-Level | Duration: 00:15:23

▼ Live Vulnerabilities Detected
╭────────────────┬──────────┬────────────┬─────────────────────╮
│ Time           │ Severity │ Type       │ Details             │
├────────────────┼──────────┼────────────┼─────────────────────┤
│ 10:23:45.123  │ CRITICAL │ Credential │ DB pass in argv     │
│ 10:23:46.456  │ HIGH     │ API Key    │ OpenAI key in JSON  │
╰────────────────┴──────────┴────────────┴─────────────────────╯

[Press 'q' to quit, 'p' to pause, 'f' to filter]
```

## Common Use Cases

### 1. Database Credential Detection
```bash
# Monitor database connections
strigoi probe center monitor mysql

# Detects:
- Passwords in connection strings
- SQL queries with embedded credentials
- Authentication packets
```

### 2. API Key Hunting
```bash
# Monitor API client
strigoi probe center monitor --name "python3" --filter "api_key|token"

# Detects:
- API keys in JSON payloads
- Bearer tokens in headers
- OAuth credentials
```

### 3. Long-Duration Baseline
```bash
# 24-hour monitoring with hourly log rotation
strigoi probe center monitor api-server --duration 24h --rotate 1h

# Useful for:
- Detecting periodic credential leaks
- Finding timing-based vulnerabilities
- Building normal behavior baseline
```

### 4. Export for Analysis
```bash
# Export to Serial Studio format
strigoi probe center monitor embedded-device --export serial-studio

# Live stream to another tool
strigoi probe center monitor nginx --redirect /dev/pts/2
```

## Vulnerability Types Detected

### Critical Severity
- Database passwords in plaintext
- Root/admin credentials
- Private keys (SSH, TLS)
- Cloud provider credentials (AWS, GCP)

### High Severity
- API keys and tokens
- OAuth secrets
- Internal service passwords
- Hardcoded credentials

### Medium Severity
- Weak authentication methods
- Sensitive data exposure
- Protocol violations
- Unusual access patterns

## Output Format

### Live Display
- Real-time vulnerability alerts
- Color-coded severity levels
- Redacted sensitive data
- Running statistics

### Log Format (JSONL)
```json
{"time":"2024-01-20T10:23:45.123Z","type":"vulnerability","severity":"critical","vuln":{"type":"credential_exposure","subtype":"database_password","evidence":"[REDACTED]","location":"process_args","confidence":0.95}}
{"time":"2024-01-20T10:23:46.456Z","type":"statistics","captured_bytes":1234567,"total_vulns":5,"protocols_seen":["json","sql"]}
```

## Advanced Options

### Filtering
```bash
# Filter by vulnerability type
strigoi probe center monitor nginx --vulns-only

# Custom regex filter
strigoi probe center monitor app --filter "password|secret|key"
```

### Performance Tuning
```bash
# Adjust buffer size for high-volume streams
strigoi probe center monitor proxy --buffer-size 1MB

# Set capture interval
strigoi probe center monitor app --poll-interval 5ms
```

### Security Options
```bash
# Run with elevated privileges (if available)
strigoi probe center monitor app --privileged

# Specify ACL rules
strigoi probe center monitor --acl-config custom.yaml
```

## Troubleshooting

### Permission Denied
```
Error: Cannot access /proc/12345/fd: Permission denied
```
Solution: Ensure you're monitoring processes owned by your user or use sudo.

### Process Not Found
```
Error: Process 'nginx' not found
```
Solution: Verify process name with `ps aux | grep nginx`

### High CPU Usage
```
Warning: CPU usage above 80%
```
Solution: Increase poll interval or reduce buffer size

## Integration Examples

### With Serial Studio
```bash
# Start Serial Studio
serial-studio &

# Stream to Serial Studio
strigoi probe center monitor device --export serial-studio --live
```

### With Analysis Pipeline
```bash
# Capture and analyze
strigoi probe center monitor app --output capture.jsonl

# Post-process
cat capture.jsonl | jq '.vuln | select(.severity=="critical")'
```

## Best Practices

1. **Start Simple**: Begin with basic monitoring before adding filters
2. **Use Logging**: Always log to file for forensic analysis
3. **Monitor Resources**: Watch CPU/memory usage for long runs
4. **Redact Sensitive Data**: Enable redaction in production
5. **Regular Reviews**: Analyze logs for patterns

## Limitations (Phase 1)

- Basic pattern matching only
- No deep protocol inspection
- Limited to user-accessible processes
- No plugin support yet
- Cannot monitor kernel threads

## Getting Help

```bash
# Built-in help
strigoi probe center --help

# Show examples
strigoi probe center monitor --examples

# Version info
strigoi --version
```

## Next Steps

1. Try monitoring a known vulnerable application
2. Set up continuous monitoring for your services
3. Export data for deeper analysis
4. Contribute patterns for new vulnerability types

---

For detailed implementation plans, see:
- [Implementation Plan](./center-implementation-plan.md)
- [Phase 1 Tasks](./center-phase1-tasks.md)
- [Architecture Design](../architecture/center-module.md)