package stream

import (
    "regexp"
    "time"
)

// StreamEventType represents the type of stream event
type StreamEventType string

const (
    StreamEventRead    StreamEventType = "read"
    StreamEventWrite   StreamEventType = "write"
    StreamEventConnect StreamEventType = "connect"
    StreamEventClose   StreamEventType = "close"
    StreamEventError   StreamEventType = "error"
    StreamEventSummary StreamEventType = "summary"
)

// Direction represents the direction of data flow
type Direction string

const (
    DirectionInbound  Direction = "inbound"
    DirectionOutbound Direction = "outbound"
    DirectionUnknown  Direction = "unknown"
    DirectionNone     Direction = "none"
)

// StreamEvent represents a single stream event
type StreamEvent struct {
    Timestamp    time.Time
    Type         StreamEventType
    Direction    Direction
    PID          int
    ProcessName  string
    FD           int            // File descriptor
    Data         []byte
    Size         int
    Summary      string         // Human-readable summary
    Metadata     map[string]interface{}
    Severity     string         // For pattern matching
}

// SecurityAlert represents a detected security issue
type SecurityAlert struct {
    Timestamp   time.Time
    EventID     string  // Unique event ID
    Severity    string  // critical, high, medium, low
    Category    string  // injection, traversal, credential, etc.
    Pattern     string  // Pattern name that triggered
    Title       string
    Description string
    Details     string
    PID         int
    ProcessName string
    Evidence    string  // String evidence for easier JSON encoding
    Blocked     bool
    Mitigation  string
}

// SecurityPattern represents a pattern to detect in stream data
type SecurityPattern struct {
    Name        string
    Category    string
    Severity    string
    Pattern     *regexp.Regexp
    Description string
    Mitigation  string
}

// Matches checks if the pattern matches the given data
func (p *SecurityPattern) Matches(data []byte) bool {
    if p.Pattern == nil {
        return false
    }
    return p.Pattern.Match(data)
}

// DefaultSecurityPatterns returns default security patterns
func DefaultSecurityPatterns() []SecurityPattern {
    return []SecurityPattern{
        {
            Name:        "AWS_CREDENTIALS",
            Category:    "credential",
            Severity:    "critical",
            Pattern:     regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
            Description: "AWS access key detected in stream",
            Mitigation:  "Remove credentials from data stream",
        },
        {
            Name:        "PRIVATE_KEY",
            Category:    "credential",
            Severity:    "critical",
            Pattern:     regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
            Description: "Private key material detected",
            Mitigation:  "Use secure key storage instead of transmitting keys",
        },
        {
            Name:        "API_KEY",
            Category:    "credential",
            Severity:    "high",
            Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey|api[_-]?secret)["\s]*[:=]["\s]*([a-zA-Z0-9_\-]+)`),
            Description: "API key or secret detected",
            Mitigation:  "Use environment variables or secure vaults",
        },
        {
            Name:        "COMMAND_INJECTION",
            Category:    "injection",
            Severity:    "high",
            Pattern:     regexp.MustCompile(`(\||;|&|&&|\|\||` + "`" + `[^` + "`" + `]*` + "`" + `|\$\([^)]*\))`),
            Description: "Command injection attempt detected",
            Mitigation:  "Sanitize input and use parameterized commands",
        },
        {
            Name:        "PATH_TRAVERSAL",
            Category:    "traversal",
            Severity:    "high",
            Pattern:     regexp.MustCompile(`\.\.\/|\.\.\\`),
            Description: "Path traversal attempt detected",
            Mitigation:  "Validate and sanitize file paths",
        },
        {
            Name:        "SQL_INJECTION",
            Category:    "injection",
            Severity:    "high",
            Pattern:     regexp.MustCompile(`(?i)(union\s+select|drop\s+table|insert\s+into|delete\s+from|update\s+set|exec\s*\(|execute\s*\()`),
            Description: "SQL injection pattern detected",
            Mitigation:  "Use parameterized queries",
        },
        {
            Name:        "BASE64_LARGE",
            Category:    "suspicious",
            Severity:    "medium",
            Pattern:     regexp.MustCompile(`[A-Za-z0-9+/]{100,}={0,2}`),
            Description: "Large base64 encoded data detected",
            Mitigation:  "Verify data encoding necessity",
        },
        {
            Name:        "SUSPICIOUS_URL",
            Category:    "suspicious",
            Severity:    "medium",
            Pattern:     regexp.MustCompile(`https?://[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`),
            Description: "URL with IP address detected",
            Mitigation:  "Verify URL destinations",
        },
    }
}

// StreamMetadata contains metadata about a stream
type StreamMetadata struct {
    ProcessID      int
    ProcessName    string
    ProcessCmdline string
    StartTime      time.Time
    EndTime        time.Time
    BytesRead      int64
    BytesWritten   int64
    EventCount     int64
    AlertCount     int64
}

// StreamOptions configures stream monitoring
type StreamOptions struct {
    Mode           string   // tap, record, block
    FollowChildren bool
    CaptureSize    int
    Timeout        time.Duration
    Patterns       []SecurityPattern
    OutputFormat   string   // json, jsonl, binary
}