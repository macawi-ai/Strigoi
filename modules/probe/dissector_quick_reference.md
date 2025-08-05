# Strigoi Dissector Quick Reference

## Interface Methods

```go
type Dissector interface {
    Identify(data []byte) (canHandle bool, confidence float64)
    Dissect(data []byte) (*Frame, error)
    FindVulnerabilities(frame *Frame) []*Vulnerability
    GetProtocolName() string
    GetDefaultPort() uint16
}
```

## Common Patterns

### Protocol Identification

```go
// Magic bytes check
if bytes.HasPrefix(data, []byte{0x47, 0x45, 0x54, 0x20}) { // "GET "
    return true, 0.90
}

// Pattern matching
if regexp.MustCompile(`^[A-Z]+ /.*? HTTP/\d\.\d`).Match(data) {
    return true, 0.85
}

// Length validation
if len(data) < MinPacketSize {
    return false, 0.0
}
```

### Error Handling

```go
// Input validation
if len(data) < HeaderSize {
    return nil, fmt.Errorf("insufficient data: need %d, got %d", HeaderSize, len(data))
}

// Boundary checks
if offset+length > len(data) {
    return nil, errors.New("field extends beyond data boundaries")
}

// Protocol violations
if version > MaxSupportedVersion {
    return nil, fmt.Errorf("unsupported version: %d", version)
}
```

### Field Extraction

```go
// String field
frame.Fields["method"] = &Field{
    Name:   "method",
    Value:  string(data[0:3]),
    Type:   FieldTypeString,
    Offset: 0,
    Length: 3,
}

// Integer field
frame.Fields["length"] = &Field{
    Name:   "length",
    Value:  binary.BigEndian.Uint32(data[4:8]),
    Type:   FieldTypeInt,
    Offset: 4,
    Length: 4,
}

// Sensitive field
frame.Fields["auth_token"] = &Field{
    Name:        "auth_token",
    Value:       token,
    Type:        FieldTypeString,
    IsSensitive: true,
}
```

## Vulnerability Types

### Common Vulnerabilities

```go
// Injection
&Vulnerability{
    Type:        "SQL_INJECTION",
    Severity:    SeverityCritical,
    Description: "SQL injection in query parameter",
    OWASP:       "A03:2021 – Injection",
}

// Authentication
&Vulnerability{
    Type:        "WEAK_AUTHENTICATION",
    Severity:    SeverityHigh,
    Description: "Basic authentication over unencrypted connection",
    OWASP:       "A07:2021 – Identification and Authentication Failures",
}

// Exposure
&Vulnerability{
    Type:        "SENSITIVE_DATA_EXPOSURE",
    Severity:    SeverityHigh,
    Description: "API key transmitted in clear text",
    OWASP:       "A02:2021 – Cryptographic Failures",
}
```

### Severity Levels

- `SeverityCritical`: Immediate exploitation possible
- `SeverityHigh`: Significant security impact
- `SeverityMedium`: Moderate risk, defense in depth
- `SeverityLow`: Minor issue, best practice violation
- `SeverityInfo`: Informational finding

## Performance Tips

### Memory Management

```go
// Use sync.Pool for buffers
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

// String conversion without allocation
func bytesToString(b []byte) string {
    return *(*string)(unsafe.Pointer(&b))
}
```

### Efficient Parsing

```go
// Use bytes operations instead of string conversion
if bytes.Equal(data[0:4], []byte("POST")) {
    // Process POST request
}

// Minimize regex compilation
var (
    headerRegex = regexp.MustCompile(`^([^:]+):\s*(.+)$`)
)
```

## Testing Checklist

- [ ] Unit tests for all methods
- [ ] Edge cases (empty data, oversized data)
- [ ] Malformed protocol data
- [ ] Fuzz testing
- [ ] Benchmark tests
- [ ] Concurrent access safety
- [ ] Memory leak checks

## Registration

```go
func init() {
    RegisterDissector("MyProtocol", NewMyProtocolDissector())
}
```

## Debugging

```go
// Enable debug logging
if debugMode {
    log.Printf("[%s] Parsing frame: %d bytes", d.GetProtocolName(), len(data))
}

// Hex dump for binary protocols
if debugMode {
    fmt.Printf("Raw data:\n%s", hex.Dump(data[:min(256, len(data))]))
}
```