# Center Module Documentation

## Overview

The Center module provides STDIO stream monitoring and vulnerability detection for running processes. It captures stdin, stdout, and stderr streams to detect sensitive data leakage such as passwords, API keys, and tokens.

## Features

- **Real-time Stream Monitoring**: Captures process I/O streams in real-time
- **PTY Support**: Automatic fallback to strace for pseudo-terminal processes
- **Vulnerability Detection**: Pattern-based detection of credentials and secrets
- **Interactive Terminal UI**: Live display of captured data and vulnerabilities
- **Activity Monitoring**: `--show-activity` flag to see all stream data
- **JSONL Logging**: Structured logging for analysis and integration

## Usage

```bash
# Monitor by process name
./strigoi probe center --target nginx

# Monitor by PID with activity display
./strigoi probe center --target 12345 --show-activity

# Enable strace for PTY processes (opt-in)
./strigoi probe center --target ssh --enable-strace

# Filter specific patterns
./strigoi probe center --target app --filter "password|token|key"

# Log-only mode (no terminal UI)
./strigoi probe center --target daemon --no-display
```

## Capture Methods

### 1. ProcFS Method (Default)
- Reads from `/proc/PID/fd/*` file descriptors
- Captures environment variables and command-line arguments
- Fast and low-overhead
- Limited by PTY isolation

### 2. Strace Method (Fallback)
- Uses system call tracing via strace
- Captures data from PTY processes
- Opt-in with `--enable-strace` flag
- Higher performance impact

## PTY Detection and Strace Fallback

The module automatically detects when a process is using a pseudo-terminal:

1. Checks if stdin/stdout/stderr point to `/dev/pts/*`
2. Monitors for static data (only environ, no stream data)
3. Tracks consecutive read failures

When PTY is detected and `--enable-strace` is enabled, the module automatically switches to strace capture.

## Known Limitations

### Strace Initial Output
**Important**: When using strace fallback, initial process output may be missed due to the time between process start and strace attachment.

**Example**:
```
Process starts → Outputs "API_KEY=secret" → Strace attaches → Captures subsequent output
                  ↑ This data is missed
```

This limitation is inherent to strace's attach-based model. For defensive monitoring scenarios requiring complete capture, consider:

1. **Kernel Auditing**: Use auditd to log process creation events
2. **Log Correlation**: Combine strace data with system logs
3. **Early Detection**: Monitor parent processes that spawn PTY children

## Performance Considerations

- **ProcFS Method**: Minimal overhead, suitable for production
- **Strace Method**: Higher CPU usage, use selectively
- **Buffer Sizes**: Default 64KB per stream, adjustable with `--buffer-size`
- **Poll Interval**: Default 10ms, adjustable with `--poll-interval`

## Security Notes

- Requires appropriate permissions to access `/proc/PID/*`
- Strace requires ptrace permissions (CAP_SYS_PTRACE)
- Captured data may contain sensitive information
- JSONL logs should be protected with appropriate file permissions

## Integration

The JSONL output format allows easy integration with:
- SIEM systems for real-time alerting
- Log analysis tools for pattern detection
- Security dashboards for visibility
- CI/CD pipelines for validation