# Strigoi Stream Output Architecture

## Overview

The stream output system in Strigoi is designed to be as flexible as Wireshark/tcpdump for network traffic, but for STDIO streams. This allows security teams to integrate Strigoi into their existing workflows and tools.

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│   Process   │────▶│ Stream Actor │────▶│ Output Writer   │
│ (Claude/MCP)│     │  (Monitor)   │     │                 │
└─────────────┘     └──────────────┘     └─────────────────┘
                            │                      │
                            ▼                      ▼
                    ┌──────────────┐      ┌───────────────┐
                    │   Pattern    │      │ Destinations: │
                    │  Detection   │      │ - File        │
                    └──────────────┘      │ - TCP Socket  │
                            │             │ - Unix Socket │
                            ▼             │ - Named Pipe  │
                    ┌──────────────┐      │ - Integration │
                    │   Security   │      └───────────────┘
                    │    Alerts    │
                    └──────────────┘
```

## Output Formats

### 1. **JSONL (JSON Lines)**
Default format for easy parsing:
```json
{"type":"event","timestamp":"2025-08-01T16:30:00Z","data":{"pid":1234,"direction":"inbound","size":48,"data":"..."}}
{"type":"alert","timestamp":"2025-08-01T16:30:01Z","data":{"severity":"high","pattern":"SENSITIVE_FILE_ACCESS"}}
```

### 2. **CEF (Common Event Format)**
For SIEM integration:
```
CEF:0|Macawi|Strigoi|1.0|StreamEvent|STDIO Activity|3|pid=1234 direction=inbound size=48
CEF:0|Macawi|Strigoi|1.0|SecurityAlert|Sensitive File Access|8|pid=1234 pattern=SENSITIVE_FILE cat=/etc/passwd
```

### 3. **PCAP (Future)**
Network-style capture format for Wireshark compatibility

## Use Cases

### 1. **Real-time Monitoring**
```bash
# Console output with color coding
stream/tap --auto-discover --output stdout

# Stream to analysis server
stream/tap --auto-discover --output tcp:soc.internal:9999
```

### 2. **Forensics & Compliance**
```bash
# Capture to timestamped file
stream/tap --pid $PID --duration 10m \
  --output file:/forensics/case123/capture_$(date +%Y%m%d_%H%M%S).jsonl

# Archive with compression
stream/tap --auto-discover \
  --output pipe:compress | gzip > /archive/mcp_audit.jsonl.gz
```

### 3. **Integration with Security Tools**
```bash
# Send to Elasticsearch
stream/tap --auto-discover \
  --output tcp:elastic.local:9200 \
  --format json

# Send to Splunk HEC
stream/tap --auto-discover \
  --output tcp:splunk.local:8088 \
  --format json

# Local syslog
stream/tap --auto-discover \
  --output integration:syslog
```

### 4. **Development & Debugging**
```bash
# Watch specific patterns
stream/tap --auto-discover --output pipe:grep | \
  grep -E "(password|secret|key)" | \
  jq .

# Real-time protocol analysis
stream/tap --auto-discover --output stdout | \
  jq -r 'select(.data.data | contains("jsonrpc"))' | \
  ./jsonrpc-analyzer
```

## Implementation Details

### Output Writer Interface
```go
type OutputWriter interface {
    WriteEvent(event *StreamEvent) error
    WriteAlert(alert *SecurityAlert) error
    Close() error
}
```

### Supported Destinations

1. **File Output**
   - Automatic rotation at size threshold
   - Timestamped filenames
   - Append mode for continuous capture

2. **TCP Output**
   - Streaming to remote collectors
   - Reconnection on failure
   - Buffering for reliability

3. **Unix Socket**
   - High-performance local IPC
   - Integration with local services
   - Lower overhead than TCP

4. **Named Pipe**
   - Classic Unix philosophy
   - Chain with existing tools
   - Real-time processing pipelines

5. **Integration Output**
   - Direct to Prometheus metrics
   - Syslog with CEF formatting
   - Custom integrations via plugins

## Security Considerations

1. **Output Sanitization**
   - Sensitive data can be redacted
   - Pattern-based filtering
   - Compliance with data policies

2. **Access Control**
   - File permissions on outputs
   - TLS for network outputs
   - Authentication for integrations

3. **Performance**
   - Async writing to prevent blocking
   - Buffering for burst handling
   - Graceful degradation

## Future Enhancements

1. **Multiple Simultaneous Outputs**
   ```bash
   stream/tap --auto-discover \
     --output stdout \
     --output file:/var/log/strigoi.jsonl \
     --output tcp:siem:514
   ```

2. **BPF-style Filtering**
   ```bash
   stream/tap --filter "pid==1234 && size>1000 && data contains 'password'"
   ```

3. **Protocol Decoders**
   ```bash
   stream/tap --decode json-rpc --decode base64
   ```

4. **Replay Capability**
   ```bash
   stream/replay capture.jsonl --speed 2x --output tcp:analyzer:9999
   ```

## Example: Complete Security Pipeline

```bash
#!/bin/bash
# Security monitoring pipeline for MCP servers

# 1. Capture all MCP traffic
./strigoi << EOF
stream/tap --auto-discover \
  --duration 24h \
  --output file:/var/log/mcp/raw_$(date +%Y%m%d).jsonl \
  --output pipe:realtime
EOF &

# 2. Real-time alerting
tail -f /tmp/strigoi-realtime.pipe | \
  jq 'select(.type=="alert" and .data.severity=="high")' | \
  ./send-to-pagerduty.sh

# 3. Metrics collection
tail -f /tmp/strigoi-realtime.pipe | \
  jq 'select(.type=="event")' | \
  ./prometheus-exporter.py

# 4. Daily analysis
0 1 * * * /usr/local/bin/analyze-mcp-logs.sh /var/log/mcp/raw_$(date -d yesterday +%Y%m%d).jsonl
```

This architecture provides the flexibility needed for:
- Real-time security monitoring
- Forensic investigation
- Compliance auditing
- Development debugging
- Integration with existing security infrastructure