# Center Module - STDIO Stream Security Analysis

## Overview

The Center module is Strigoi's real-time STDIO stream analyzer, designed to detect credentials, API keys, tokens, and other sensitive data flowing through process streams. Operating at the center of the compass, it provides comprehensive monitoring of data flows through stdin, stdout, and stderr.

## Key Features

### 🔍 Real-Time Detection
- Live monitoring of process STDIO streams
- Immediate vulnerability alerts as data flows
- Continuous operation for long-duration analysis
- User-level operation (no root required)

### 🛡️ Comprehensive Coverage
- **Credential Types**: Passwords, API keys, tokens, OAuth secrets
- **Protocols**: JSON, SQL, plaintext, and more
- **Sources**: Command arguments, environment variables, stream data
- **Formats**: Connection strings, bearer tokens, private keys

### 📊 Analysis Capabilities
- Multi-protocol dissection with confidence scoring
- Pattern-based credential hunting
- Context-aware vulnerability classification
- Redacted evidence for safe reporting

## Architecture

### Core Components

1. **Capture Engine**
   - ProcFS-based stream reading
   - Fallback to strace for enhanced capture
   - Ring buffer management for efficiency
   - Process lifecycle tracking

2. **Protocol Dissectors**
   - JSON: Detects credentials in structured data
   - SQL: Identifies passwords in queries and connection strings
   - PlainText: Catches credentials in unstructured text

3. **Credential Hunter**
   - Regex-based pattern matching
   - Confidence scoring for accuracy
   - Smart redaction for safe display
   - False positive filtering

4. **Terminal Display**
   - Real-time vulnerability alerts
   - Color-coded severity levels
   - Running statistics
   - Interactive controls

## Usage

### Basic Commands

```bash
# Monitor a process by name
strigoi probe center --target nginx

# Monitor by PID
strigoi probe center --target 12345

# Monitor with custom output
strigoi probe center --target mysql --output vulns.jsonl

# Time-limited monitoring
strigoi probe center --target api-server --duration 1h

# Filter specific patterns
strigoi probe center --target app --filter "password|token"

# Log-only mode (no UI)
strigoi probe center --target service --no-display
```

### Command Options

| Flag | Description | Default |
|------|-------------|---------|
| `--target, -t` | Process name or PID to monitor | Required |
| `--duration, -d` | Maximum monitoring duration | 0 (unlimited) |
| `--output, -o` | Log file path (JSONL format) | stream-monitor.jsonl |
| `--no-display` | Disable terminal UI | false |
| `--filter, -f` | Regex filter for stream data | "" |
| `--buffer-size` | Buffer size per stream (KB) | 64 |
| `--poll-interval` | Stream polling interval (ms) | 10 |

## Vulnerability Detection

### Severity Levels

- **CRITICAL**: Database passwords, root credentials, private keys
- **HIGH**: API keys, OAuth tokens, bearer tokens
- **MEDIUM**: Weak auth methods, sensitive data exposure
- **LOW**: Information disclosure, timing patterns

### Detection Examples

#### JSON Credentials
```json
{
  "database": {
    "password": "SuperSecret123!"
  },
  "api_key": "sk-1234567890abcdef"
}
```

#### SQL Passwords
```sql
CREATE USER 'admin' IDENTIFIED BY 'P@ssw0rd';
mysql://user:password@host:3306/db
```

#### Environment Variables
```
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG
GITHUB_TOKEN=ghp_a1b2c3d4e5f6g7h8
```

## Output Format

### Terminal Display
```
═══════════════ Strigoi Center - Stream Monitor ═══════════════
Processes: 3 | Status: MONITORING | Duration: 00:15:23

▼ Live Vulnerabilities Detected
╭────────────────┬──────────┬────────────┬─────────────────────╮
│ Time           │ Severity │ Type       │ Evidence            │
├────────────────┼──────────┼────────────┼─────────────────────┤
│ 10:23:45.123  │ CRITICAL │ password   │ ****                │
│ 10:23:46.456  │ HIGH     │ api_key    │ sk-****...****      │
╰────────────────┴──────────┴────────────┴─────────────────────╯

╭─────────────────────────────────────────────────────────────╮
│ Stats: 1.2MB captured | 2 vulns | 543 events               │
│ Last activity: 2s ago                                       │
╰─────────────────────────────────────────────────────────────╯

[Press 'q' to quit, 'p' to pause, 'c' to clear]
```

### JSONL Log Format
```json
{"timestamp":"2024-01-20T10:23:45.123Z","type":"vulnerability","vuln":{"id":"VULN-123","severity":"critical","type":"credential_exposure","subtype":"database_password","evidence":"****","location":"stdout","confidence":0.95,"process":{"pid":12345,"name":"mysql","cmd":"mysql -u root -p"}}}
{"timestamp":"2024-01-20T10:23:50.000Z","type":"statistics","data":{"total_bytes":1234567,"total_events":543,"total_vulns":2,"processes_count":3}}
```

## Integration

### With v2 Output Pipeline
The Center module fully integrates with Strigoi's v2 output pipeline:
- Standardized result formatting
- Severity-based grouping
- Pretty printing with color coding
- JSON/YAML export support

### Serial Studio Export
Future support for exporting to Serial Studio format for advanced telemetry visualization.

## Security Considerations

### Safe by Design
- **User-level operation**: No elevated privileges required
- **Read-only mode**: Default operation doesn't modify streams
- **Credential redaction**: Sensitive data automatically masked
- **Process isolation**: Monitors only specified targets

### Best Practices
1. Run with minimal required privileges
2. Secure log files containing vulnerability data
3. Use filters to reduce noise and false positives
4. Regularly review and update detection patterns

## Performance

### Resource Usage
- **Memory**: ~50MB base + buffer allocations
- **CPU**: <5% for typical monitoring
- **Disk**: Log growth ~1MB/minute (varies by activity)

### Optimization Tips
- Adjust buffer size for high-volume streams
- Increase poll interval for lower CPU usage
- Use filters to reduce processing overhead
- Enable log rotation for long-duration monitoring

## Troubleshooting

### Common Issues

**Permission Denied**
```
Error: Cannot access /proc/12345/fd: Permission denied
```
Solution: Ensure you're monitoring processes owned by your user.

**Process Not Found**
```
Error: Process 'nginx' not found
```
Solution: Verify process name with `ps aux | grep nginx`.

**High CPU Usage**
```
Warning: CPU usage above 80%
```
Solution: Increase poll interval or reduce buffer size.

## Future Enhancements

### Phase 2 (Security Hardening)
- Input validation framework
- Data sanitization pipeline
- ACL system for process monitoring
- Encrypted configuration

### Phase 3 (Performance)
- eBPF integration for kernel-level capture
- Asynchronous processing pipeline
- Performance monitoring dashboard
- Memory pooling optimizations

### Phase 4 (Advanced Features)
- Plugin architecture for custom dissectors
- Graph-based sudo chain detection
- Serial Studio real-time export
- High availability mode

## Example: Demo Script

```bash
#!/bin/bash
# Start a vulnerable test app
python3 vulnerable-app.py &
APP_PID=$!

# Monitor with Center module
strigoi probe center --target $APP_PID --duration 30s

# Clean up
kill $APP_PID
```

## Contributing

The Center module welcomes contributions for:
- New credential detection patterns
- Additional protocol dissectors
- Performance optimizations
- Security enhancements

See the [implementation plan](./center-implementation-plan.md) for detailed architecture and roadmap.