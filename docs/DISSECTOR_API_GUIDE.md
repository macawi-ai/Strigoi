# Strigoi Dissector API Guide

## Overview

Dissectors are the core components of Strigoi's protocol analysis engine. They identify, parse, and analyze different protocol data streams to detect security vulnerabilities. This guide provides comprehensive documentation for creating custom dissectors.

## Table of Contents

1. [Dissector Interface](#dissector-interface)
2. [Implementation Requirements](#implementation-requirements)
3. [Creating a Custom Dissector](#creating-a-custom-dissector)
4. [Best Practices](#best-practices)
5. [Testing Guidelines](#testing-guidelines)
6. [Performance Considerations](#performance-considerations)
7. [Security Guidelines](#security-guidelines)
8. [Example: Redis Dissector](#example-redis-dissector)

## Dissector Interface

Every dissector must implement the following interface:

```go
type Dissector interface {
    // Identify checks if the data matches this dissector's protocol.
    // Returns: (matches bool, confidence float64)
    // confidence: 0.0 to 1.0, where 1.0 is absolute certainty
    Identify(data []byte) (bool, float64)
    
    // Dissect parses the data into a structured frame.
    // Returns: (*Frame, error)
    Dissect(data []byte) (*Frame, error)
    
    // FindVulnerabilities analyzes a frame for security issues.
    // Returns: []StreamVulnerability
    FindVulnerabilities(frame *Frame) []StreamVulnerability
    
    // GetSessionID extracts the session identifier from a frame.
    // Returns: (sessionID string, error)
    GetSessionID(frame *Frame) (string, error)
}
```

### Core Types

```go
// Frame represents parsed protocol data
type Frame struct {
    Protocol string                 `json:"protocol"`
    Fields   map[string]interface{} `json:"fields"`
    Raw      []byte                 `json:"-"`
}

// StreamVulnerability represents a detected security issue
type StreamVulnerability struct {
    Type       string    // e.g., "credential", "injection", "configuration"
    Subtype    string    // e.g., "password_exposure", "sql_injection"
    Severity   string    // "critical", "high", "medium", "low"
    Confidence float64   // 0.0 to 1.0
    Evidence   string    // What was found (redacted)
    Context    string    // Where it was found
    Timestamp  time.Time
}
```

## Implementation Requirements

### 1. Protocol Identification

The `Identify` method must:
- Be fast and efficient (called on every data chunk)
- Return accurate confidence scores
- Handle partial data gracefully
- Not modify the input data

```go
func (d *MyDissector) Identify(data []byte) (bool, float64) {
    if len(data) < 4 {
        return false, 0.0
    }
    
    // Check magic bytes, headers, or patterns
    if bytes.HasPrefix(data, []byte("MYPROTO")) {
        return true, 0.95
    }
    
    // Check for protocol-specific patterns
    if d.patternRegex.Match(data) {
        return true, 0.7
    }
    
    return false, 0.0
}
```

### 2. Data Dissection

The `Dissect` method must:
- Parse protocol-specific structures
- Extract relevant fields
- Handle malformed data without panicking
- Preserve raw data for forensics

```go
func (d *MyDissector) Dissect(data []byte) (*Frame, error) {
    frame := &Frame{
        Protocol: "MyProtocol",
        Fields:   make(map[string]interface{}),
        Raw:      data,
    }
    
    // Parse protocol headers
    header, err := d.parseHeader(data)
    if err != nil {
        return nil, fmt.Errorf("header parse failed: %w", err)
    }
    
    frame.Fields["type"] = header.Type
    frame.Fields["version"] = header.Version
    frame.Fields["payload"] = header.Payload
    
    return frame, nil
}
```

### 3. Vulnerability Detection

The `FindVulnerabilities` method must:
- Check for protocol-specific security issues
- Use confidence scoring appropriately
- Redact sensitive data in evidence
- Provide actionable context

```go
func (d *MyDissector) FindVulnerabilities(frame *Frame) []StreamVulnerability {
    var vulns []StreamVulnerability
    
    // Check for exposed credentials
    if creds := d.findCredentials(frame); len(creds) > 0 {
        for _, cred := range creds {
            vulns = append(vulns, StreamVulnerability{
                Type:       "credential",
                Subtype:    cred.Type,
                Severity:   "critical",
                Confidence: cred.Confidence,
                Evidence:   redact(cred.Value),
                Context:    cred.Location,
                Timestamp:  time.Now(),
            })
        }
    }
    
    // Check for injection vulnerabilities
    if injections := d.findInjections(frame); len(injections) > 0 {
        vulns = append(vulns, injections...)
    }
    
    return vulns
}
```

### 4. Session Identification

The `GetSessionID` method must:
- Extract consistent session identifiers
- Return protocol-prefixed IDs to avoid collisions
- Handle missing session data gracefully

```go
func (d *MyDissector) GetSessionID(frame *Frame) (string, error) {
    // Check protocol-specific session fields
    if sid, ok := frame.Fields["session_id"].(string); ok && sid != "" {
        return fmt.Sprintf("myproto_session_%s", sid), nil
    }
    
    // Check for connection-based sessions
    if connID, ok := frame.Fields["connection_id"].(string); ok {
        return fmt.Sprintf("myproto_conn_%s", connID), nil
    }
    
    // Generate hash-based ID from identifying fields
    if canHash := d.canGenerateSessionHash(frame); canHash {
        hash := d.generateSessionHash(frame)
        return fmt.Sprintf("myproto_hash_%x", hash), nil
    }
    
    return "", fmt.Errorf("no session identifier found")
}
```

## Creating a Custom Dissector

### Step 1: Define Your Dissector Structure

```go
package probe

import (
    "bytes"
    "fmt"
    "regexp"
    "time"
)

type MyProtocolDissector struct {
    // Pre-compiled patterns for efficiency
    headerPattern     *regexp.Regexp
    credentialPattern *regexp.Regexp
    
    // Protocol-specific constants
    magicBytes []byte
    minSize    int
    
    // Caching for performance
    cache map[string]interface{}
}

func NewMyProtocolDissector() *MyProtocolDissector {
    return &MyProtocolDissector{
        headerPattern:     regexp.MustCompile(`^MYPROTO/(\d+\.\d+)`),
        credentialPattern: regexp.MustCompile(`(?i)(password|token|key)\s*[:=]\s*([^\s,]+)`),
        magicBytes:        []byte("MYPROTO"),
        minSize:          8,
        cache:            make(map[string]interface{}),
    }
}
```

### Step 2: Implement Required Methods

See the interface implementation examples above.

### Step 3: Add Helper Methods

```go
// parseHeader extracts protocol header information
func (d *MyProtocolDissector) parseHeader(data []byte) (*MyProtoHeader, error) {
    if len(data) < d.minSize {
        return nil, fmt.Errorf("insufficient data for header")
    }
    
    // Protocol-specific parsing logic
    header := &MyProtoHeader{}
    // ... parsing implementation ...
    
    return header, nil
}

// findCredentials searches for exposed credentials
func (d *MyProtocolDissector) findCredentials(frame *Frame) []Credential {
    var creds []Credential
    
    // Search in various frame fields
    for fieldName, fieldValue := range frame.Fields {
        if str, ok := fieldValue.(string); ok {
            matches := d.credentialPattern.FindAllStringSubmatch(str, -1)
            for _, match := range matches {
                creds = append(creds, Credential{
                    Type:       match[1],
                    Value:      match[2],
                    Location:   fieldName,
                    Confidence: 0.9,
                })
            }
        }
    }
    
    return creds
}
```

### Step 4: Register Your Dissector

Add your dissector to the Center module's dissector list:

```go
// In center.go
func (m *CenterModule) initDissectors() {
    m.dissectors = []Dissector{
        NewHTTPDissector(),
        NewGRPCDissectorV2(),
        NewWebSocketDissector(),
        NewMyProtocolDissector(), // Add your dissector
        // ... other dissectors
    }
}
```

## Best Practices

### 1. Error Handling

- Never panic on malformed input
- Return meaningful error messages
- Use error wrapping for context
- Log errors at appropriate levels

```go
if err != nil {
    return nil, fmt.Errorf("failed to parse %s header: %w", d.Protocol, err)
}
```

### 2. Performance Optimization

- Pre-compile regular expressions
- Use byte operations over string operations
- Implement caching for repeated patterns
- Avoid unnecessary allocations

```go
// Good: Pre-compiled pattern
var headerPattern = regexp.MustCompile(`^PROTO/\d+`)

// Bad: Compiling in hot path
func identify(data []byte) bool {
    pattern := regexp.MustCompile(`^PROTO/\d+`) // Don't do this!
    return pattern.Match(data)
}
```

### 3. Security Considerations

- Validate all input data
- Set maximum sizes for buffers
- Prevent ReDoS attacks with regex timeouts
- Sanitize output in vulnerability evidence

```go
// Prevent ReDoS
func (d *MyDissector) safeRegexMatch(data []byte) bool {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    result := make(chan bool, 1)
    go func() {
        result <- d.complexPattern.Match(data)
    }()
    
    select {
    case matched := <-result:
        return matched
    case <-ctx.Done():
        return false // Timeout - pattern too complex
    }
}
```

### 4. Protocol Versioning

Support multiple protocol versions:

```go
type ProtocolVersion int

const (
    ProtoV1 ProtocolVersion = iota
    ProtoV2
    ProtoV3
)

func (d *MyDissector) detectVersion(data []byte) ProtocolVersion {
    // Version detection logic
}

func (d *MyDissector) Dissect(data []byte) (*Frame, error) {
    version := d.detectVersion(data)
    
    switch version {
    case ProtoV1:
        return d.dissectV1(data)
    case ProtoV2:
        return d.dissectV2(data)
    default:
        return nil, fmt.Errorf("unsupported protocol version")
    }
}
```

## Testing Guidelines

### 1. Unit Tests

Test each method independently:

```go
func TestMyDissector_Identify(t *testing.T) {
    d := NewMyProtocolDissector()
    
    tests := []struct {
        name       string
        data       []byte
        wantMatch  bool
        wantConf   float64
    }{
        {
            name:      "valid protocol header",
            data:      []byte("MYPROTO/1.0\r\n"),
            wantMatch: true,
            wantConf:  0.95,
        },
        {
            name:      "partial header",
            data:      []byte("MYPRO"),
            wantMatch: false,
            wantConf:  0.0,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            match, conf := d.Identify(tt.data)
            if match != tt.wantMatch {
                t.Errorf("Identify() match = %v, want %v", match, tt.wantMatch)
            }
            if conf != tt.wantConf {
                t.Errorf("Identify() confidence = %v, want %v", conf, tt.wantConf)
            }
        })
    }
}
```

### 2. Fuzzing Tests

Test with random/malformed input:

```go
func FuzzMyDissector(f *testing.F) {
    d := NewMyProtocolDissector()
    
    // Add seed corpus
    f.Add([]byte("MYPROTO/1.0\r\nHello"))
    f.Add([]byte("INVALID DATA"))
    
    f.Fuzz(func(t *testing.T, data []byte) {
        // Should not panic
        _, _ = d.Identify(data)
        
        if match, _ := d.Identify(data); match {
            _, err := d.Dissect(data)
            if err == nil {
                // Valid data, check further
                frame, _ := d.Dissect(data)
                _ = d.FindVulnerabilities(frame)
                _, _ = d.GetSessionID(frame)
            }
        }
    })
}
```

### 3. Benchmark Tests

Measure performance:

```go
func BenchmarkMyDissector_Identify(b *testing.B) {
    d := NewMyProtocolDissector()
    data := []byte("MYPROTO/1.0\r\nSample data...")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        d.Identify(data)
    }
}
```

## Performance Considerations

### 1. Identification Performance

- Identify should complete in < 1ms for typical data
- Use quick checks (magic bytes) before expensive operations
- Return early on negative matches

### 2. Memory Management

- Reuse buffers where possible
- Limit frame field sizes
- Clear caches periodically

```go
// Buffer pool for efficiency
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

func (d *MyDissector) processData(data []byte) {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)
    
    // Use buf for processing
}
```

### 3. Concurrency

Dissectors must be thread-safe:

```go
type MyDissector struct {
    // Immutable after creation
    patterns []*regexp.Regexp
    
    // Protected by mutex
    mu    sync.RWMutex
    cache map[string]interface{}
}

func (d *MyDissector) getCached(key string) (interface{}, bool) {
    d.mu.RLock()
    defer d.mu.RUnlock()
    val, ok := d.cache[key]
    return val, ok
}
```

## Security Guidelines

### 1. Input Validation

- Validate all input sizes
- Check for integer overflows
- Sanitize regex patterns

```go
const maxFrameSize = 10 * 1024 * 1024 // 10MB

func (d *MyDissector) Dissect(data []byte) (*Frame, error) {
    if len(data) > maxFrameSize {
        return nil, fmt.Errorf("frame too large: %d bytes", len(data))
    }
    
    // Process safely
}
```

### 2. Resource Limits

- Set timeouts for operations
- Limit regex complexity
- Cap memory usage

### 3. Data Redaction

Always redact sensitive data:

```go
func redact(value string) string {
    if len(value) <= 8 {
        return "***"
    }
    return value[:3] + "***" + value[len(value)-3:]
}
```

## Example: Redis Dissector

Here's a complete example of a Redis protocol dissector:

```go
package probe

import (
    "bufio"
    "bytes"
    "fmt"
    "regexp"
    "strconv"
    "strings"
    "time"
)

type RedisDissector struct {
    authPattern *regexp.Regexp
    respParser  *RESPParser
}

func NewRedisDissector() *RedisDissector {
    return &RedisDissector{
        authPattern: regexp.MustCompile(`(?i)^AUTH\s+(.+)$`),
        respParser:  NewRESPParser(),
    }
}

func (d *RedisDissector) Identify(data []byte) (bool, float64) {
    if len(data) < 3 {
        return false, 0.0
    }
    
    // Check for RESP protocol markers
    firstByte := data[0]
    switch firstByte {
    case '+', '-', ':', '$', '*':
        // Looks like RESP protocol
        if d.isValidRESP(data) {
            return true, 0.9
        }
        return true, 0.5
    }
    
    // Check for inline commands
    if d.looksLikeRedisCommand(data) {
        return true, 0.7
    }
    
    return false, 0.0
}

func (d *RedisDissector) Dissect(data []byte) (*Frame, error) {
    frame := &Frame{
        Protocol: "Redis",
        Fields:   make(map[string]interface{}),
        Raw:      data,
    }
    
    // Parse RESP format
    resp, err := d.respParser.Parse(data)
    if err != nil {
        // Try inline command format
        if cmd := d.parseInlineCommand(data); cmd != nil {
            frame.Fields["type"] = "inline_command"
            frame.Fields["command"] = cmd.Name
            frame.Fields["args"] = cmd.Args
            return frame, nil
        }
        return nil, err
    }
    
    frame.Fields["type"] = resp.Type
    frame.Fields["data"] = resp.Data
    
    // Extract command if array
    if resp.Type == "array" && len(resp.Array) > 0 {
        frame.Fields["command"] = strings.ToUpper(resp.Array[0])
        if len(resp.Array) > 1 {
            frame.Fields["args"] = resp.Array[1:]
        }
    }
    
    return frame, nil
}

func (d *RedisDissector) FindVulnerabilities(frame *Frame) []StreamVulnerability {
    var vulns []StreamVulnerability
    
    // Check for AUTH command with password
    if cmd, ok := frame.Fields["command"].(string); ok && cmd == "AUTH" {
        if args, ok := frame.Fields["args"].([]string); ok && len(args) > 0 {
            vulns = append(vulns, StreamVulnerability{
                Type:       "credential",
                Subtype:    "redis_password",
                Severity:   "critical",
                Confidence: 1.0,
                Evidence:   fmt.Sprintf("AUTH %s", redact(args[0])),
                Context:    "Redis authentication command",
                Timestamp:  time.Now(),
            })
        }
    }
    
    // Check for dangerous commands
    dangerousCommands := map[string]string{
        "FLUSHDB":   "Database wipe command",
        "FLUSHALL":  "All databases wipe command",
        "CONFIG":    "Configuration access",
        "SCRIPT":    "Lua script execution",
        "EVAL":      "Lua code execution",
        "MODULE":    "Module management",
    }
    
    if cmd, ok := frame.Fields["command"].(string); ok {
        if desc, isDangerous := dangerousCommands[cmd]; isDangerous {
            vulns = append(vulns, StreamVulnerability{
                Type:       "configuration",
                Subtype:    "dangerous_command",
                Severity:   "high",
                Confidence: 0.9,
                Evidence:   cmd,
                Context:    desc,
                Timestamp:  time.Now(),
            })
        }
    }
    
    return vulns
}

func (d *RedisDissector) GetSessionID(frame *Frame) (string, error) {
    // Redis doesn't have built-in sessions, use connection info
    if connID, ok := frame.Fields["connection_id"].(string); ok {
        return fmt.Sprintf("redis_conn_%s", connID), nil
    }
    
    // Use client info if available
    if clientID, ok := frame.Fields["client_id"].(string); ok {
        return fmt.Sprintf("redis_client_%s", clientID), nil
    }
    
    return "", fmt.Errorf("no Redis session identifier found")
}

// Helper methods

func (d *RedisDissector) isValidRESP(data []byte) bool {
    scanner := bufio.NewScanner(bytes.NewReader(data))
    if !scanner.Scan() {
        return false
    }
    
    line := scanner.Text()
    if len(line) < 1 {
        return false
    }
    
    switch line[0] {
    case '+', '-', ':':
        return true
    case '$':
        // Bulk string, check length
        if len(line) > 1 {
            _, err := strconv.Atoi(line[1:])
            return err == nil
        }
    case '*':
        // Array, check count
        if len(line) > 1 {
            _, err := strconv.Atoi(line[1:])
            return err == nil
        }
    }
    
    return false
}

func (d *RedisDissector) looksLikeRedisCommand(data []byte) bool {
    // Common Redis commands
    commands := []string{
        "GET", "SET", "DEL", "EXISTS", "EXPIRE", "TTL",
        "INCR", "DECR", "LPUSH", "RPUSH", "LPOP", "RPOP",
        "SADD", "SREM", "SMEMBERS", "HGET", "HSET", "HDEL",
        "ZADD", "ZREM", "ZRANGE", "PUBLISH", "SUBSCRIBE",
        "AUTH", "PING", "INFO", "CONFIG", "CLIENT", "MONITOR",
    }
    
    upperData := strings.ToUpper(string(data))
    for _, cmd := range commands {
        if strings.HasPrefix(upperData, cmd+" ") || upperData == cmd {
            return true
        }
    }
    
    return false
}
```

## Integration Checklist

Before submitting your dissector:

- [ ] Implements all interface methods
- [ ] Has comprehensive unit tests (>80% coverage)
- [ ] Includes benchmark tests
- [ ] Handles malformed input gracefully
- [ ] Thread-safe implementation
- [ ] Documentation with examples
- [ ] No security vulnerabilities (regex DoS, buffer overflows)
- [ ] Performance meets requirements (<1ms identification)
- [ ] Proper error handling and logging
- [ ] Follows Go best practices and style guide

## Support

For questions or assistance:
- Review existing dissectors in `/modules/probe/dissector_*.go`
- Check test files for examples
- Open an issue on GitHub
- Contact the maintainers

---

*Last updated: Phase 3 Implementation*
*Version: 1.0.0*