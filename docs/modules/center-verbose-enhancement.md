# Center Module Verbose Mode Enhancement

## Problem
When monitoring processes, analysts can't tell if:
- The module is actually capturing data
- The process is idle
- The capture method is working
- No vulnerabilities just means no data flow

## Proposed Solution

### 1. Add --verbose flag to show:
- Process attachment status
- Stream activity indicators
- Bytes captured in real-time
- Data snippets (safely truncated)
- Capture method being used

### 2. Add --show-activity flag to display:
- Non-vulnerable data flow (first 50 chars)
- Stream source (stdin/stdout/stderr)
- Timestamp of each capture
- Process state changes

### 3. Enhanced Terminal Display in Verbose Mode:
```
═══════════════ Strigoi Center - Stream Monitor ═══════════════
Target: claude (PID: 1751) | Mode: User-Level | Duration: 00:02:15
Capture Method: procfs | Status: ACTIVE

▼ Stream Activity
╭────────────────┬────────┬───────────┬──────────────────────────╮
│ Time           │ Stream │ Bytes     │ Preview                  │
├────────────────┼────────┼───────────┼──────────────────────────┤
│ 10:23:45.123  │ stdout │ 1,234     │ {"id":"msg_123","type".. │
│ 10:23:45.456  │ stdin  │ 567       │ {"messages":[{"role":".. │
│ 10:23:46.789  │ stdout │ 8,901     │ I'll help you with th... │
╰────────────────┴────────┴───────────┴──────────────────────────╯

▼ Vulnerabilities Detected
[No vulnerabilities detected yet]

▼ Capture Statistics
Total Captured: 10.7 KB | Events: 3 | Active Streams: 2/3
Last Activity: 2s ago | Capture Errors: 0

[Press 'q' to quit, 'v' for verbose, 'f' to filter]
```

### 4. Debug Output (--debug flag):
```
[DEBUG] Attempting to attach to PID 1751 (claude)
[DEBUG] Opening /proc/1751/fd/0 (stdin): Permission denied
[DEBUG] Opening /proc/1751/fd/1 (stdout): Success (pipe:[123456])
[DEBUG] Opening /proc/1751/fd/2 (stderr): Success (pipe:[123457])
[DEBUG] Fallback to strace method...
[DEBUG] Read 1234 bytes from stdout
[DEBUG] No credentials detected in chunk
[DEBUG] Buffer position: 1234/65536
```

### 5. Implementation Changes Needed:

#### In center.go:
- Add "verbose" and "show-activity" options
- Log activity even without vulnerabilities
- Track capture method success/failure

#### In center_display.go:
- Add activity table
- Show non-sensitive data previews
- Display capture statistics
- Show capture method and errors

#### In center_capture.go:
- Return capture method used
- Log permission errors
- Track failed attempts

### 6. Command Examples:
```bash
# Show all stream activity
./strigoi probe center --target claude --verbose

# Show activity without vulnerabilities
./strigoi probe center --target nginx --show-activity

# Debug capture issues
./strigoi probe center --target mysql --debug

# Combine with filters
./strigoi probe center --target claude --verbose --filter "api|key|token"
```

This would help analysts:
1. Confirm the module is working
2. See data flow patterns
3. Debug permission issues
4. Understand capture limitations
5. Tune their monitoring approach