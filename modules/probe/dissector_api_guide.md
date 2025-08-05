# Strigoi Dissector API Documentation

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Core Interfaces](#core-interfaces)
- [Implementation Guide](#implementation-guide)
- [Best Practices](#best-practices)
- [Testing Guidelines](#testing-guidelines)
- [Examples](#examples)
- [API Reference](#api-reference)

## Overview

The Strigoi Dissector API provides a framework for creating protocol analyzers that can identify, parse, and detect vulnerabilities in network traffic. Dissectors are the core components that transform raw network data into structured frames and identify security issues.

### Key Concepts

- **Dissector**: A protocol analyzer that implements the `Dissector` interface
- **Frame**: A structured representation of protocol data
- **Field**: An individual data element within a frame
- **Vulnerability**: A detected security issue within the protocol data

## Architecture

```
┌─────────────────┐     ┌──────────────┐     ┌─────────────────┐
│ Capture Engine  │────▶│  Dissector   │────▶│     Frame       │
└─────────────────┘     └──────────────┘     └─────────────────┘
        │                       │                      │
        │                       ▼                      ▼
        │               ┌──────────────┐      ┌─────────────────┐
        └──────────────▶│ Vulnerability│      │  Session Mgr    │
                        │   Detector   │      └─────────────────┘
                        └──────────────┘
```

## Core Interfaces

### Dissector Interface

```go
type Dissector interface {
    // Identify determines if this dissector can handle the given data
    Identify(data []byte) (canHandle bool, confidence float64)
    
    // Dissect parses the data into a structured frame
    Dissect(data []byte) (*Frame, error)
    
    // FindVulnerabilities analyzes the frame for security issues
    FindVulnerabilities(frame *Frame) []*Vulnerability
    
    // GetProtocolName returns the name of the protocol
    GetProtocolName() string
    
    // GetDefaultPort returns the default port for this protocol
    GetDefaultPort() uint16
}
```

### Frame Structure

```go
type Frame struct {
    Protocol  string                 // Protocol name (e.g., "HTTP", "gRPC")
    Timestamp time.Time              // When the frame was captured
    Fields    map[string]*Field      // Protocol fields
    Payload   []byte                 // Raw payload data
    Metadata  map[string]interface{} // Additional metadata
}

type Field struct {
    Name        string      // Field name
    Value       interface{} // Field value
    Type        FieldType   // Data type
    Offset      int         // Offset in original data
    Length      int         // Length in bytes
    IsSensitive bool        // Contains sensitive data
    Tags        []string    // Additional tags
}

type FieldType int

const (
    FieldTypeString FieldType = iota
    FieldTypeInt
    FieldTypeFloat
    FieldTypeBool
    FieldTypeBytes
    FieldTypeTime
    FieldTypeJSON
    FieldTypeXML
)
```

### Vulnerability Structure

```go
type Vulnerability struct {
    Type        string              // Vulnerability type
    Severity    Severity            // Risk level
    Description string              // Human-readable description
    Evidence    map[string]string   // Supporting evidence
    Location    VulnerabilityLocation
    Remediation string              // Suggested fix
    CVE         string              // CVE identifier if applicable
    OWASP       string              // OWASP category
}

type VulnerabilityLocation struct {
    Field    string // Field containing the vulnerability
    Offset   int    // Byte offset
    Length   int    // Length of vulnerable data
    LineNum  int    // Line number if applicable
    ColNum   int    // Column number if applicable
}

type Severity int

const (
    SeverityInfo Severity = iota
    SeverityLow
    SeverityMedium
    SeverityHigh
    SeverityCritical
)
```

## Implementation Guide

### Step 1: Create Your Dissector Structure

```go
type MyProtocolDissector struct {
    // Add any state or configuration needed
    maxFrameSize int
    strictMode   bool
}

func NewMyProtocolDissector() *MyProtocolDissector {
    return &MyProtocolDissector{
        maxFrameSize: 65536,
        strictMode:   true,
    }
}
```

### Step 2: Implement Protocol Identification

```go
func (d *MyProtocolDissector) Identify(data []byte) (bool, float64) {
    if len(data) < 4 {
        return false, 0.0
    }
    
    // Check protocol magic bytes or patterns
    if bytes.HasPrefix(data, []byte("MYPROTO")) {
        return true, 0.95
    }
    
    // Check for protocol-specific patterns
    if d.hasProtocolSignature(data) {
        return true, 0.80
    }
    
    return false, 0.0
}
```

### Step 3: Implement Frame Dissection

```go
func (d *MyProtocolDissector) Dissect(data []byte) (*Frame, error) {
    frame := &Frame{
        Protocol:  d.GetProtocolName(),
        Timestamp: time.Now(),
        Fields:    make(map[string]*Field),
        Metadata:  make(map[string]interface{}),
    }
    
    // Parse header
    header, err := d.parseHeader(data)
    if err != nil {
        return nil, fmt.Errorf("failed to parse header: %w", err)
    }
    
    // Extract fields
    frame.Fields["version"] = &Field{
        Name:  "version",
        Value: header.Version,
        Type:  FieldTypeString,
        Offset: 0,
        Length: 2,
    }
    
    // Parse body
    body, err := d.parseBody(data[header.Size:])
    if err != nil {
        return nil, fmt.Errorf("failed to parse body: %w", err)
    }
    
    // Add body fields
    for name, field := range body.Fields {
        frame.Fields[name] = field
    }
    
    frame.Payload = data
    return frame, nil
}
```

### Step 4: Implement Vulnerability Detection

```go
func (d *MyProtocolDissector) FindVulnerabilities(frame *Frame) []*Vulnerability {
    var vulnerabilities []*Vulnerability
    
    // Check for weak authentication
    if auth, ok := frame.Fields["authorization"]; ok {
        if vuln := d.checkWeakAuth(auth); vuln != nil {
            vulnerabilities = append(vulnerabilities, vuln)
        }
    }
    
    // Check for sensitive data exposure
    for name, field := range frame.Fields {
        if d.isSensitiveField(name) && !field.IsSensitive {
            vulnerabilities = append(vulnerabilities, &Vulnerability{
                Type:        "SENSITIVE_DATA_EXPOSURE",
                Severity:    SeverityHigh,
                Description: fmt.Sprintf("Sensitive field '%s' transmitted without encryption", name),
                Evidence: map[string]string{
                    "field": name,
                    "value": fmt.Sprintf("%v", field.Value),
                },
                Location: VulnerabilityLocation{
                    Field:  name,
                    Offset: field.Offset,
                    Length: field.Length,
                },
                OWASP: "A3:2021 – Sensitive Data Exposure",
            })
        }
    }
    
    return vulnerabilities
}
```

### Step 5: Implement Utility Methods

```go
func (d *MyProtocolDissector) GetProtocolName() string {
    return "MyProtocol"
}

func (d *MyProtocolDissector) GetDefaultPort() uint16 {
    return 8080
}
```

## Best Practices

### 1. Efficient Protocol Identification

- Use quick checks first (magic bytes, fixed headers)
- Return confidence scores based on match quality
- Avoid expensive operations in `Identify()`

```go
func (d *MyDissector) Identify(data []byte) (bool, float64) {
    // Quick length check
    if len(data) < d.minPacketSize {
        return false, 0.0
    }
    
    // Check magic bytes first (fast)
    if !bytes.HasPrefix(data, d.magicBytes) {
        return false, 0.0
    }
    
    // More thorough checks for higher confidence
    if d.validateChecksum(data) {
        return true, 0.95
    }
    
    return true, 0.70
}
```

### 2. Robust Error Handling

- Always validate input data
- Use meaningful error messages
- Don't panic on malformed data

```go
func (d *MyDissector) parseHeader(data []byte) (*Header, error) {
    if len(data) < HeaderSize {
        return nil, fmt.Errorf("insufficient data for header: got %d bytes, need %d", 
            len(data), HeaderSize)
    }
    
    header := &Header{}
    
    // Validate version
    version := binary.BigEndian.Uint16(data[0:2])
    if version > MaxVersion {
        return nil, fmt.Errorf("unsupported protocol version: %d", version)
    }
    
    header.Version = version
    return header, nil
}
```

### 3. Memory Efficiency

- Reuse buffers when possible
- Avoid unnecessary allocations
- Use sync.Pool for temporary objects

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

func (d *MyDissector) parseData(data []byte) error {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)
    
    // Use buf for parsing...
    return nil
}
```

### 4. Comprehensive Vulnerability Detection

- Check for common vulnerability patterns
- Use CVE and OWASP references
- Provide actionable remediation advice

```go
func (d *MyDissector) checkInjection(field *Field) *Vulnerability {
    value := field.Value.(string)
    
    // SQL injection patterns
    sqlPatterns := []string{
        "' OR '1'='1",
        "'; DROP TABLE",
        "UNION SELECT",
    }
    
    for _, pattern := range sqlPatterns {
        if strings.Contains(value, pattern) {
            return &Vulnerability{
                Type:        "SQL_INJECTION",
                Severity:    SeverityCritical,
                Description: "Potential SQL injection detected",
                Evidence: map[string]string{
                    "pattern": pattern,
                    "field":   field.Name,
                },
                Remediation: "Use parameterized queries and input validation",
                OWASP:       "A03:2021 – Injection",
            }
        }
    }
    
    return nil
}
```

## Testing Guidelines

### Unit Tests

Every dissector should have comprehensive unit tests:

```go
func TestMyDissector_Identify(t *testing.T) {
    d := NewMyProtocolDissector()
    
    tests := []struct {
        name       string
        data       []byte
        wantHandle bool
        wantConf   float64
    }{
        {
            name:       "valid protocol",
            data:       []byte("MYPROTO\x01\x00..."),
            wantHandle: true,
            wantConf:   0.95,
        },
        {
            name:       "invalid magic",
            data:       []byte("INVALID\x01\x00..."),
            wantHandle: false,
            wantConf:   0.0,
        },
        {
            name:       "too short",
            data:       []byte("MY"),
            wantHandle: false,
            wantConf:   0.0,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            gotHandle, gotConf := d.Identify(tt.data)
            if gotHandle != tt.wantHandle {
                t.Errorf("Identify() gotHandle = %v, want %v", gotHandle, tt.wantHandle)
            }
            if math.Abs(gotConf-tt.wantConf) > 0.01 {
                t.Errorf("Identify() gotConf = %v, want %v", gotConf, tt.wantConf)
            }
        })
    }
}
```

### Fuzzing

Use fuzzing to test robustness:

```go
func FuzzMyDissector(f *testing.F) {
    d := NewMyProtocolDissector()
    
    // Add seed corpus
    f.Add([]byte("MYPROTO\x01\x00valid data"))
    f.Add([]byte("MYPROTO\xff\xffinvalid"))
    
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

### Benchmark Tests

Measure performance:

```go
func BenchmarkMyDissector_Dissect(b *testing.B) {
    d := NewMyProtocolDissector()
    data := generateTestData(1024)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        frame, err := d.Dissect(data)
        if err != nil {
            b.Fatal(err)
        }
        _ = frame
    }
}
```

## Examples

### HTTP Dissector (Simplified)

```go
type HTTPDissector struct {
    maxHeaderSize int
}

func (d *HTTPDissector) Identify(data []byte) (bool, float64) {
    if len(data) < 4 {
        return false, 0.0
    }
    
    // Check for HTTP methods
    methods := []string{"GET ", "POST", "PUT ", "HEAD", "HTTP"}
    for _, method := range methods {
        if bytes.HasPrefix(data, []byte(method)) {
            return true, 0.90
        }
    }
    
    return false, 0.0
}

func (d *HTTPDissector) Dissect(data []byte) (*Frame, error) {
    frame := &Frame{
        Protocol:  "HTTP",
        Timestamp: time.Now(),
        Fields:    make(map[string]*Field),
    }
    
    // Parse request/response line
    lines := bytes.Split(data, []byte("\r\n"))
    if len(lines) < 1 {
        return nil, errors.New("invalid HTTP data")
    }
    
    // Parse first line
    parts := bytes.Split(lines[0], []byte(" "))
    if len(parts) >= 3 {
        if bytes.HasPrefix(parts[0], []byte("HTTP")) {
            // Response
            frame.Fields["version"] = &Field{
                Name:  "version",
                Value: string(parts[0]),
                Type:  FieldTypeString,
            }
            frame.Fields["status_code"] = &Field{
                Name:  "status_code",
                Value: string(parts[1]),
                Type:  FieldTypeString,
            }
        } else {
            // Request
            frame.Fields["method"] = &Field{
                Name:  "method",
                Value: string(parts[0]),
                Type:  FieldTypeString,
            }
            frame.Fields["uri"] = &Field{
                Name:  "uri",
                Value: string(parts[1]),
                Type:  FieldTypeString,
            }
        }
    }
    
    // Parse headers
    for i := 1; i < len(lines); i++ {
        if len(lines[i]) == 0 {
            break // End of headers
        }
        
        colonIndex := bytes.IndexByte(lines[i], ':')
        if colonIndex > 0 {
            name := string(bytes.ToLower(lines[i][:colonIndex]))
            value := strings.TrimSpace(string(lines[i][colonIndex+1:]))
            
            frame.Fields[name] = &Field{
                Name:  name,
                Value: value,
                Type:  FieldTypeString,
            }
        }
    }
    
    frame.Payload = data
    return frame, nil
}

func (d *HTTPDissector) FindVulnerabilities(frame *Frame) []*Vulnerability {
    var vulns []*Vulnerability
    
    // Check for sensitive data in URI
    if uri, ok := frame.Fields["uri"]; ok {
        uriStr := uri.Value.(string)
        if strings.Contains(uriStr, "password=") || strings.Contains(uriStr, "api_key=") {
            vulns = append(vulns, &Vulnerability{
                Type:        "SENSITIVE_DATA_IN_URL",
                Severity:    SeverityHigh,
                Description: "Sensitive data transmitted in URL",
                Evidence: map[string]string{
                    "uri": uriStr,
                },
                Remediation: "Use POST requests with body for sensitive data",
                OWASP:       "A01:2021 – Broken Access Control",
            })
        }
    }
    
    // Check for missing security headers
    securityHeaders := []string{
        "strict-transport-security",
        "x-frame-options",
        "x-content-type-options",
        "content-security-policy",
    }
    
    for _, header := range securityHeaders {
        if _, ok := frame.Fields[header]; !ok {
            vulns = append(vulns, &Vulnerability{
                Type:        "MISSING_SECURITY_HEADER",
                Severity:    SeverityMedium,
                Description: fmt.Sprintf("Missing security header: %s", header),
                Evidence: map[string]string{
                    "header": header,
                },
                Remediation: fmt.Sprintf("Add %s header", header),
                OWASP:       "A05:2021 – Security Misconfiguration",
            })
        }
    }
    
    return vulns
}
```

## API Reference

### Constants

```go
// Field Types
const (
    FieldTypeString FieldType = iota
    FieldTypeInt
    FieldTypeFloat
    FieldTypeBool
    FieldTypeBytes
    FieldTypeTime
    FieldTypeJSON
    FieldTypeXML
)

// Severity Levels
const (
    SeverityInfo Severity = iota
    SeverityLow
    SeverityMedium
    SeverityHigh
    SeverityCritical
)

// Common Vulnerability Types
const (
    VulnTypeInjection           = "INJECTION"
    VulnTypeAuthFailure         = "AUTH_FAILURE"
    VulnTypeSensitiveData       = "SENSITIVE_DATA"
    VulnTypeMisconfiguration    = "MISCONFIGURATION"
    VulnTypeInsecureDesign      = "INSECURE_DESIGN"
    VulnTypeOutdatedComponent   = "OUTDATED_COMPONENT"
    VulnTypeIdentityFailure     = "IDENTITY_FAILURE"
    VulnTypeIntegrityFailure    = "INTEGRITY_FAILURE"
    VulnTypeLoggingFailure      = "LOGGING_FAILURE"
    VulnTypeSSRF                = "SSRF"
)
```

### Helper Functions

```go
// ExtractString safely extracts a string from data
func ExtractString(data []byte, offset, length int) (string, error)

// ParseDelimited parses delimited data (e.g., headers)
func ParseDelimited(data []byte, delimiter byte) map[string]string

// IsASCIIPrintable checks if data contains only printable ASCII
func IsASCIIPrintable(data []byte) bool

// DetectEncoding attempts to detect text encoding
func DetectEncoding(data []byte) string

// CalculateEntropy calculates Shannon entropy of data
func CalculateEntropy(data []byte) float64
```

### Registration

```go
// Register registers a dissector with the system
func RegisterDissector(name string, dissector Dissector) error

// GetDissector retrieves a registered dissector
func GetDissector(name string) (Dissector, error)

// ListDissectors returns all registered dissectors
func ListDissectors() []string
```

## Conclusion

The Strigoi Dissector API provides a powerful framework for protocol analysis and vulnerability detection. By following these guidelines and best practices, you can create robust dissectors that enhance the security monitoring capabilities of the Strigoi platform.

For more examples and the latest updates, visit the [Strigoi GitHub repository](https://github.com/macawi-ai/strigoi).