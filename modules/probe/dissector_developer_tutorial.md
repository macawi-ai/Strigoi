# Strigoi Dissector Developer Tutorial

## Introduction

This tutorial will walk you through creating a custom dissector for the Strigoi platform. We'll build a Redis protocol dissector from scratch, demonstrating best practices and common patterns.

## Prerequisites

- Go 1.18 or later
- Basic understanding of network protocols
- Familiarity with the Strigoi architecture

## Tutorial: Building a Redis Dissector

### Step 1: Understanding the Redis Protocol

Redis uses a text-based protocol called RESP (REdis Serialization Protocol). Key characteristics:

- Commands are sent as arrays of bulk strings
- Responses use type prefixes: `+` (simple string), `-` (error), `:` (integer), `$` (bulk string), `*` (array)
- Line endings are CRLF (`\r\n`)

Example command:
```
*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n
```

### Step 2: Create the Dissector Structure

Create a new file `redis_dissector.go`:

```go
package probe

import (
    "bytes"
    "errors"
    "fmt"
    "strconv"
    "strings"
    "time"
)

// RedisDissector analyzes Redis protocol traffic
type RedisDissector struct {
    // Configuration
    maxBulkSize      int
    detectPasswords  bool
    detectDangerous  bool
    
    // Statistics
    commandStats     map[string]int
}

// NewRedisDissector creates a new Redis protocol dissector
func NewRedisDissector() *RedisDissector {
    return &RedisDissector{
        maxBulkSize:     1024 * 1024, // 1MB max bulk string
        detectPasswords: true,
        detectDangerous: true,
        commandStats:    make(map[string]int),
    }
}
```

### Step 3: Implement Protocol Identification

```go
func (d *RedisDissector) Identify(data []byte) (bool, float64) {
    if len(data) < 3 {
        return false, 0.0
    }
    
    // Check for RESP protocol markers
    firstByte := data[0]
    
    // RESP types: +, -, :, $, *
    switch firstByte {
    case '+', '-', ':', '$', '*':
        // Check for CRLF
        if bytes.Contains(data[:min(100, len(data))], []byte("\r\n")) {
            return true, 0.85
        }
        return true, 0.60
        
    default:
        // Check for inline commands (older protocol)
        // Commands like: PING\r\n or GET key\r\n
        if d.isInlineCommand(data) {
            return true, 0.70
        }
    }
    
    return false, 0.0
}

func (d *RedisDissector) isInlineCommand(data []byte) bool {
    // Common Redis commands
    commands := []string{
        "PING", "PONG", "GET", "SET", "DEL", "EXISTS",
        "INCR", "DECR", "LPUSH", "RPUSH", "LPOP", "RPOP",
        "SADD", "SREM", "SMEMBERS", "HSET", "HGET",
    }
    
    upperData := strings.ToUpper(string(data[:min(20, len(data))]))
    for _, cmd := range commands {
        if strings.HasPrefix(upperData, cmd) {
            return true
        }
    }
    
    return false
}
```

### Step 4: Implement Frame Dissection

```go
func (d *RedisDissector) Dissect(data []byte) (*Frame, error) {
    frame := &Frame{
        Protocol:  "Redis",
        Timestamp: time.Now(),
        Fields:    make(map[string]*Field),
        Metadata:  make(map[string]interface{}),
    }
    
    // Parse RESP message
    msg, err := d.parseRESP(data)
    if err != nil {
        // Try inline protocol
        if cmd, args, err := d.parseInline(data); err == nil {
            frame.Fields["command"] = &Field{
                Name:  "command",
                Value: cmd,
                Type:  FieldTypeString,
            }
            
            for i, arg := range args {
                frame.Fields[fmt.Sprintf("arg%d", i)] = &Field{
                    Name:  fmt.Sprintf("arg%d", i),
                    Value: arg,
                    Type:  FieldTypeString,
                }
            }
            
            frame.Payload = data
            return frame, nil
        }
        
        return nil, fmt.Errorf("failed to parse Redis protocol: %w", err)
    }
    
    // Process parsed message
    switch msg.Type {
    case "array":
        // Command request
        if len(msg.Array) > 0 {
            cmd := strings.ToUpper(msg.Array[0].String)
            frame.Fields["command"] = &Field{
                Name:  "command",
                Value: cmd,
                Type:  FieldTypeString,
            }
            
            // Record command statistics
            d.commandStats[cmd]++
            
            // Extract arguments
            for i := 1; i < len(msg.Array); i++ {
                fieldName := fmt.Sprintf("arg%d", i-1)
                frame.Fields[fieldName] = &Field{
                    Name:  fieldName,
                    Value: msg.Array[i].String,
                    Type:  FieldTypeString,
                    IsSensitive: d.isSensitiveArg(cmd, i-1, msg.Array[i].String),
                }
            }
        }
        
    case "simple_string", "bulk_string":
        frame.Fields["response"] = &Field{
            Name:  "response",
            Value: msg.String,
            Type:  FieldTypeString,
        }
        
    case "error":
        frame.Fields["error"] = &Field{
            Name:  "error",
            Value: msg.String,
            Type:  FieldTypeString,
        }
        
    case "integer":
        frame.Fields["integer"] = &Field{
            Name:  "integer",
            Value: msg.Integer,
            Type:  FieldTypeInt,
        }
    }
    
    frame.Payload = data
    frame.Metadata["resp_type"] = msg.Type
    
    return frame, nil
}

// RESPMessage represents a parsed RESP message
type RESPMessage struct {
    Type    string
    String  string
    Integer int64
    Array   []*RESPMessage
    Error   string
}

func (d *RedisDissector) parseRESP(data []byte) (*RESPMessage, error) {
    if len(data) == 0 {
        return nil, errors.New("empty data")
    }
    
    reader := bytes.NewReader(data)
    return d.parseRESPMessage(reader)
}

func (d *RedisDissector) parseRESPMessage(reader *bytes.Reader) (*RESPMessage, error) {
    typeByte, err := reader.ReadByte()
    if err != nil {
        return nil, err
    }
    
    switch typeByte {
    case '+':
        // Simple string
        line, err := d.readLine(reader)
        if err != nil {
            return nil, err
        }
        return &RESPMessage{Type: "simple_string", String: line}, nil
        
    case '-':
        // Error
        line, err := d.readLine(reader)
        if err != nil {
            return nil, err
        }
        return &RESPMessage{Type: "error", Error: line}, nil
        
    case ':':
        // Integer
        line, err := d.readLine(reader)
        if err != nil {
            return nil, err
        }
        num, err := strconv.ParseInt(line, 10, 64)
        if err != nil {
            return nil, err
        }
        return &RESPMessage{Type: "integer", Integer: num}, nil
        
    case '$':
        // Bulk string
        line, err := d.readLine(reader)
        if err != nil {
            return nil, err
        }
        length, err := strconv.Atoi(line)
        if err != nil {
            return nil, err
        }
        
        if length == -1 {
            // Null bulk string
            return &RESPMessage{Type: "null"}, nil
        }
        
        if length > d.maxBulkSize {
            return nil, fmt.Errorf("bulk string too large: %d bytes", length)
        }
        
        // Read bulk string data
        data := make([]byte, length)
        _, err = reader.Read(data)
        if err != nil {
            return nil, err
        }
        
        // Skip CRLF
        reader.ReadByte()
        reader.ReadByte()
        
        return &RESPMessage{Type: "bulk_string", String: string(data)}, nil
        
    case '*':
        // Array
        line, err := d.readLine(reader)
        if err != nil {
            return nil, err
        }
        count, err := strconv.Atoi(line)
        if err != nil {
            return nil, err
        }
        
        if count == -1 {
            // Null array
            return &RESPMessage{Type: "null"}, nil
        }
        
        array := make([]*RESPMessage, count)
        for i := 0; i < count; i++ {
            elem, err := d.parseRESPMessage(reader)
            if err != nil {
                return nil, err
            }
            array[i] = elem
        }
        
        return &RESPMessage{Type: "array", Array: array}, nil
        
    default:
        return nil, fmt.Errorf("unknown RESP type: %c", typeByte)
    }
}

func (d *RedisDissector) readLine(reader *bytes.Reader) (string, error) {
    var line []byte
    for {
        b, err := reader.ReadByte()
        if err != nil {
            return "", err
        }
        if b == '\r' {
            next, err := reader.ReadByte()
            if err != nil {
                return "", err
            }
            if next == '\n' {
                return string(line), nil
            }
            line = append(line, b, next)
        } else {
            line = append(line, b)
        }
    }
}
```

### Step 5: Implement Vulnerability Detection

```go
func (d *RedisDissector) FindVulnerabilities(frame *Frame) []*Vulnerability {
    var vulns []*Vulnerability
    
    // Check command-specific vulnerabilities
    if cmdField, ok := frame.Fields["command"]; ok {
        command := cmdField.Value.(string)
        
        // Check for dangerous commands
        if d.detectDangerous && d.isDangerousCommand(command) {
            vulns = append(vulns, &Vulnerability{
                Type:        "DANGEROUS_COMMAND",
                Severity:    SeverityHigh,
                Description: fmt.Sprintf("Dangerous Redis command detected: %s", command),
                Evidence: map[string]string{
                    "command": command,
                },
                Location: VulnerabilityLocation{
                    Field: "command",
                },
                Remediation: "Restrict access to dangerous commands using ACL or rename-command",
                OWASP:       "A01:2021 – Broken Access Control",
            })
        }
        
        // Check for authentication issues
        if command == "AUTH" {
            if arg0, ok := frame.Fields["arg0"]; ok {
                password := arg0.Value.(string)
                if vuln := d.checkWeakPassword(password); vuln != nil {
                    vulns = append(vulns, vuln)
                }
            }
        }
        
        // Check for potential injection in EVAL/SCRIPT commands
        if command == "EVAL" || command == "SCRIPT" {
            if arg0, ok := frame.Fields["arg0"]; ok {
                script := arg0.Value.(string)
                if vuln := d.checkLuaInjection(script); vuln != nil {
                    vulns = append(vulns, vuln)
                }
            }
        }
    }
    
    // Check for sensitive data exposure
    for name, field := range frame.Fields {
        if strings.HasPrefix(name, "arg") && field.IsSensitive {
            value := fmt.Sprintf("%v", field.Value)
            if d.containsSensitivePattern(value) {
                vulns = append(vulns, &Vulnerability{
                    Type:        "SENSITIVE_DATA_EXPOSURE",
                    Severity:    SeverityHigh,
                    Description: "Sensitive data detected in Redis command",
                    Evidence: map[string]string{
                        "field": name,
                        "pattern": "contains sensitive pattern",
                    },
                    Location: VulnerabilityLocation{
                        Field: name,
                    },
                    Remediation: "Encrypt sensitive data before storing in Redis",
                    OWASP:       "A02:2021 – Cryptographic Failures",
                })
            }
        }
    }
    
    // Check for unencrypted connection
    if !d.isEncrypted(frame) {
        vulns = append(vulns, &Vulnerability{
            Type:        "UNENCRYPTED_CONNECTION",
            Severity:    SeverityMedium,
            Description: "Redis connection is not encrypted",
            Remediation: "Enable TLS for Redis connections",
            OWASP:       "A02:2021 – Cryptographic Failures",
        })
    }
    
    return vulns
}

func (d *RedisDissector) isDangerousCommand(cmd string) bool {
    dangerous := []string{
        "FLUSHDB", "FLUSHALL", "KEYS", "CONFIG", "SHUTDOWN",
        "DEBUG", "MONITOR", "SAVE", "BGSAVE", "BGREWRITEAOF",
        "CLIENT", "SLOWLOG", "SCRIPT",
    }
    
    for _, dangerous := range dangerous {
        if cmd == dangerous {
            return true
        }
    }
    return false
}

func (d *RedisDissector) checkWeakPassword(password string) *Vulnerability {
    // Check password strength
    if len(password) < 8 {
        return &Vulnerability{
            Type:        "WEAK_PASSWORD",
            Severity:    SeverityCritical,
            Description: "Weak Redis password detected",
            Evidence: map[string]string{
                "length": fmt.Sprintf("%d", len(password)),
            },
            Remediation: "Use a strong password with at least 16 characters",
            OWASP:       "A07:2021 – Identification and Authentication Failures",
        }
    }
    
    // Check for common passwords
    commonPasswords := []string{
        "password", "123456", "redis", "admin", "root",
        "test", "guest", "default", "changeme",
    }
    
    lowerPassword := strings.ToLower(password)
    for _, common := range commonPasswords {
        if lowerPassword == common {
            return &Vulnerability{
                Type:        "COMMON_PASSWORD",
                Severity:    SeverityCritical,
                Description: "Common password detected",
                Evidence: map[string]string{
                    "type": "common password",
                },
                Remediation: "Use a unique, strong password",
                OWASP:       "A07:2021 – Identification and Authentication Failures",
            }
        }
    }
    
    return nil
}

func (d *RedisDissector) checkLuaInjection(script string) *Vulnerability {
    // Check for potential Lua injection patterns
    dangerous := []string{
        "os.execute", "io.popen", "loadfile", "dofile",
        "require", "rawget", "rawset", "getfenv", "setfenv",
    }
    
    scriptLower := strings.ToLower(script)
    for _, pattern := range dangerous {
        if strings.Contains(scriptLower, pattern) {
            return &Vulnerability{
                Type:        "LUA_INJECTION",
                Severity:    SeverityCritical,
                Description: "Potential Lua injection in Redis EVAL/SCRIPT",
                Evidence: map[string]string{
                    "pattern": pattern,
                },
                Remediation: "Validate and sanitize Lua scripts, use EVALSHA with pre-loaded scripts",
                OWASP:       "A03:2021 – Injection",
            }
        }
    }
    
    return nil
}

func (d *RedisDissector) isSensitiveArg(cmd string, argIndex int, value string) bool {
    // AUTH command password
    if cmd == "AUTH" && argIndex == 0 {
        return true
    }
    
    // SET/HSET with potential sensitive keys
    if (cmd == "SET" || cmd == "HSET") && argIndex == 0 {
        sensitiveKeys := []string{
            "password", "passwd", "pwd", "secret", "token",
            "api_key", "apikey", "private_key", "credit_card",
            "ssn", "pin",
        }
        
        keyLower := strings.ToLower(value)
        for _, sensitive := range sensitiveKeys {
            if strings.Contains(keyLower, sensitive) {
                return true
            }
        }
    }
    
    return false
}

func (d *RedisDissector) containsSensitivePattern(value string) bool {
    // Credit card pattern
    if matched, _ := regexp.MatchString(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`, value); matched {
        return true
    }
    
    // SSN pattern
    if matched, _ := regexp.MatchString(`\b\d{3}-\d{2}-\d{4}\b`, value); matched {
        return true
    }
    
    // API key patterns
    if strings.HasPrefix(value, "sk_") || strings.HasPrefix(value, "pk_") {
        return true
    }
    
    return false
}

func (d *RedisDissector) isEncrypted(frame *Frame) bool {
    // Check if connection uses TLS
    // This would need integration with connection metadata
    if tls, ok := frame.Metadata["tls"].(bool); ok {
        return tls
    }
    return false
}
```

### Step 6: Implement Helper Methods

```go
func (d *RedisDissector) GetProtocolName() string {
    return "Redis"
}

func (d *RedisDissector) GetDefaultPort() uint16 {
    return 6379
}

func (d *RedisDissector) parseInline(data []byte) (string, []string, error) {
    // Parse inline commands like: GET key\r\n
    line := bytes.TrimSuffix(data, []byte("\r\n"))
    parts := bytes.Split(line, []byte(" "))
    
    if len(parts) == 0 {
        return "", nil, errors.New("empty command")
    }
    
    cmd := string(bytes.ToUpper(parts[0]))
    args := make([]string, len(parts)-1)
    for i := 1; i < len(parts); i++ {
        args[i-1] = string(parts[i])
    }
    
    return cmd, args, nil
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

### Step 7: Write Comprehensive Tests

Create `redis_dissector_test.go`:

```go
package probe

import (
    "testing"
)

func TestRedisDissector_Identify(t *testing.T) {
    d := NewRedisDissector()
    
    tests := []struct {
        name       string
        data       []byte
        wantHandle bool
        minConf    float64
    }{
        {
            name:       "RESP array command",
            data:       []byte("*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"),
            wantHandle: true,
            minConf:    0.80,
        },
        {
            name:       "RESP simple string",
            data:       []byte("+OK\r\n"),
            wantHandle: true,
            minConf:    0.80,
        },
        {
            name:       "RESP error",
            data:       []byte("-ERR unknown command\r\n"),
            wantHandle: true,
            minConf:    0.80,
        },
        {
            name:       "Inline command",
            data:       []byte("PING\r\n"),
            wantHandle: true,
            minConf:    0.60,
        },
        {
            name:       "Not Redis",
            data:       []byte("GET / HTTP/1.1\r\n"),
            wantHandle: false,
            minConf:    0.0,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            gotHandle, gotConf := d.Identify(tt.data)
            if gotHandle != tt.wantHandle {
                t.Errorf("Identify() gotHandle = %v, want %v", gotHandle, tt.wantHandle)
            }
            if gotHandle && gotConf < tt.minConf {
                t.Errorf("Identify() confidence = %v, want >= %v", gotConf, tt.minConf)
            }
        })
    }
}

func TestRedisDissector_Dissect(t *testing.T) {
    d := NewRedisDissector()
    
    tests := []struct {
        name         string
        data         []byte
        wantCommand  string
        wantArgs     []string
        wantResponse string
        wantError    bool
    }{
        {
            name:        "SET command",
            data:        []byte("*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"),
            wantCommand: "SET",
            wantArgs:    []string{"key", "value"},
        },
        {
            name:         "Simple string response",
            data:         []byte("+OK\r\n"),
            wantResponse: "OK",
        },
        {
            name:        "GET command inline",
            data:        []byte("GET mykey\r\n"),
            wantCommand: "GET",
            wantArgs:    []string{"mykey"},
        },
        {
            name:      "Malformed data",
            data:      []byte("*2\r\n$3\r\nGET"),
            wantError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            frame, err := d.Dissect(tt.data)
            
            if tt.wantError {
                if err == nil {
                    t.Errorf("Dissect() expected error but got none")
                }
                return
            }
            
            if err != nil {
                t.Errorf("Dissect() unexpected error: %v", err)
                return
            }
            
            if tt.wantCommand != "" {
                cmd, ok := frame.Fields["command"]
                if !ok {
                    t.Errorf("Missing command field")
                } else if cmd.Value != tt.wantCommand {
                    t.Errorf("Command = %v, want %v", cmd.Value, tt.wantCommand)
                }
                
                // Check arguments
                for i, wantArg := range tt.wantArgs {
                    argField, ok := frame.Fields[fmt.Sprintf("arg%d", i)]
                    if !ok {
                        t.Errorf("Missing arg%d", i)
                    } else if argField.Value != wantArg {
                        t.Errorf("arg%d = %v, want %v", i, argField.Value, wantArg)
                    }
                }
            }
            
            if tt.wantResponse != "" {
                resp, ok := frame.Fields["response"]
                if !ok {
                    t.Errorf("Missing response field")
                } else if resp.Value != tt.wantResponse {
                    t.Errorf("Response = %v, want %v", resp.Value, tt.wantResponse)
                }
            }
        })
    }
}

func TestRedisDissector_FindVulnerabilities(t *testing.T) {
    d := NewRedisDissector()
    
    tests := []struct {
        name      string
        data      []byte
        wantVulns []string
    }{
        {
            name:      "FLUSHALL command",
            data:      []byte("*1\r\n$8\r\nFLUSHALL\r\n"),
            wantVulns: []string{"DANGEROUS_COMMAND", "UNENCRYPTED_CONNECTION"},
        },
        {
            name:      "Weak AUTH",
            data:      []byte("*2\r\n$4\r\nAUTH\r\n$6\r\n123456\r\n"),
            wantVulns: []string{"WEAK_PASSWORD", "COMMON_PASSWORD", "UNENCRYPTED_CONNECTION"},
        },
        {
            name:      "EVAL with dangerous Lua",
            data:      []byte("*3\r\n$4\r\nEVAL\r\n$29\r\nos.execute('rm -rf /')\r\n$1\r\n0\r\n"),
            wantVulns: []string{"LUA_INJECTION", "UNENCRYPTED_CONNECTION"},
        },
        {
            name:      "SET with password key",
            data:      []byte("*3\r\n$3\r\nSET\r\n$8\r\npassword\r\n$10\r\nsecret123!\r\n"),
            wantVulns: []string{"UNENCRYPTED_CONNECTION"},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            frame, err := d.Dissect(tt.data)
            if err != nil {
                t.Fatalf("Failed to dissect: %v", err)
            }
            
            vulns := d.FindVulnerabilities(frame)
            
            if len(vulns) != len(tt.wantVulns) {
                t.Errorf("Got %d vulnerabilities, want %d", len(vulns), len(tt.wantVulns))
            }
            
            // Check vulnerability types
            foundTypes := make(map[string]bool)
            for _, vuln := range vulns {
                foundTypes[vuln.Type] = true
            }
            
            for _, wantType := range tt.wantVulns {
                if !foundTypes[wantType] {
                    t.Errorf("Missing expected vulnerability type: %s", wantType)
                }
            }
        })
    }
}

func BenchmarkRedisDissector_Dissect(b *testing.B) {
    d := NewRedisDissector()
    
    // Typical SET command
    data := []byte("*3\r\n$3\r\nSET\r\n$8\r\nmykey123\r\n$27\r\nThis is a test value 12345\r\n")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        frame, err := d.Dissect(data)
        if err != nil {
            b.Fatal(err)
        }
        _ = frame
    }
}

func FuzzRedisDissector(f *testing.F) {
    d := NewRedisDissector()
    
    // Add seed corpus
    seeds := [][]byte{
        []byte("*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"),
        []byte("+OK\r\n"),
        []byte("-ERR unknown\r\n"),
        []byte("PING\r\n"),
        []byte("*2\r\n$4\r\nAUTH\r\n$8\r\npassword\r\n"),
    }
    
    for _, seed := range seeds {
        f.Add(seed)
    }
    
    f.Fuzz(func(t *testing.T, data []byte) {
        // Should not panic
        canHandle, _ := d.Identify(data)
        if canHandle {
            frame, err := d.Dissect(data)
            if err == nil && frame != nil {
                // Should not panic
                _ = d.FindVulnerabilities(frame)
            }
        }
    })
}
```

### Step 8: Integration and Registration

Create `redis_dissector_integration.go`:

```go
package probe

func init() {
    // Register the Redis dissector
    RegisterDissector("Redis", NewRedisDissector())
}

// Example usage in session analysis
func AnalyzeRedisSession(session *Session) *SessionAnalysis {
    analysis := &SessionAnalysis{
        SessionID: session.ID,
        Protocol:  "Redis",
        StartTime: session.StartTime,
    }
    
    dissector := NewRedisDissector()
    
    for _, frame := range session.Frames {
        // Find vulnerabilities
        vulns := dissector.FindVulnerabilities(frame)
        analysis.Vulnerabilities = append(analysis.Vulnerabilities, vulns...)
        
        // Extract commands for analysis
        if cmd, ok := frame.Fields["command"]; ok {
            analysis.Commands = append(analysis.Commands, cmd.Value.(string))
        }
    }
    
    // Analyze command patterns
    analysis.Patterns = analyzeCommandPatterns(analysis.Commands)
    
    return analysis
}

type SessionAnalysis struct {
    SessionID       string
    Protocol        string
    StartTime       time.Time
    Commands        []string
    Vulnerabilities []*Vulnerability
    Patterns        []string
}

func analyzeCommandPatterns(commands []string) []string {
    var patterns []string
    
    // Check for suspicious patterns
    getCount := 0
    setCount := 0
    dangerousCount := 0
    
    for _, cmd := range commands {
        switch cmd {
        case "GET", "MGET", "HGET":
            getCount++
        case "SET", "MSET", "HSET":
            setCount++
        case "FLUSHDB", "FLUSHALL", "CONFIG", "SCRIPT":
            dangerousCount++
        }
    }
    
    if float64(dangerousCount)/float64(len(commands)) > 0.1 {
        patterns = append(patterns, "HIGH_DANGEROUS_COMMAND_RATIO")
    }
    
    if getCount > 100 && setCount == 0 {
        patterns = append(patterns, "READ_ONLY_PATTERN")
    }
    
    if setCount > 100 && getCount == 0 {
        patterns = append(patterns, "WRITE_ONLY_PATTERN")
    }
    
    return patterns
}
```

## Best Practices Summary

1. **Protocol Identification**
   - Use quick heuristics first
   - Return confidence scores
   - Handle edge cases gracefully

2. **Parsing Robustness**
   - Validate all input
   - Handle malformed data
   - Set reasonable limits

3. **Vulnerability Detection**
   - Check for protocol-specific issues
   - Use OWASP references
   - Provide actionable remediation

4. **Performance**
   - Minimize allocations
   - Use buffered I/O
   - Cache compiled regexes

5. **Testing**
   - Unit test all methods
   - Fuzz test for robustness
   - Benchmark critical paths

## Next Steps

1. **Extend the Dissector**
   - Add support for Redis Cluster protocol
   - Implement Redis Streams parsing
   - Add Pub/Sub message handling

2. **Enhance Detection**
   - Add anomaly detection
   - Implement rate limiting checks
   - Detect data exfiltration patterns

3. **Integration**
   - Connect with session management
   - Add metrics collection
   - Implement alerting

## Conclusion

You've now created a complete Redis protocol dissector for Strigoi. This dissector can:
- Identify Redis traffic with confidence scoring
- Parse both RESP and inline protocol formats
- Detect various security vulnerabilities
- Handle malformed data gracefully

Use this pattern as a template for creating dissectors for other protocols!