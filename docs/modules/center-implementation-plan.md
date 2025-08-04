# Center Module Implementation Plan

## Overview
The Center module is Strigoi's stream analysis component, focusing on STDIO monitoring, vulnerability detection, and real-time security analysis. This document details the phased implementation approach.

## Phase 1: Core MVP (Weeks 1-2)

### Goals
- Establish basic stream capture and monitoring
- Implement real-time vulnerability detection
- Create live terminal display
- Enable structured logging

### Deliverables

#### 1.1 User-Level Capture Engine
```go
// modules/probe/center.go
type CaptureEngine struct {
    // Primary capture methods (no special privileges)
    procFS      *ProcFSCapture    // /proc/PID/fd/* monitoring
    strace      *StraceWrapper    // Limited but effective
    
    // Target management
    targets     []CaptureTarget
    buffers     map[int]*RingBuffer
}

// Implementation details:
// - Read /proc/PID/fd/0,1,2 for STDIO
// - Poll for new data every 10ms
// - Buffer size: 64KB per stream
// - Handle process termination gracefully
```

**File Structure:**
```
modules/probe/
├── center.go           # Main module
├── capture_procfs.go   # ProcFS implementation
├── capture_strace.go   # Strace wrapper
└── types.go           # Data structures
```

#### 1.2 Basic Protocol Dissectors
```go
// Initial dissectors for MVP
type JSONDissector struct{}
type SQLDissector struct{}
type PlainTextDissector struct{}

// Simple pattern matching
func (j *JSONDissector) DetectCredentials(data []byte) []Credential {
    // Regex patterns for common credential formats
    patterns := []string{
        `"password"\s*:\s*"([^"]+)"`,
        `"api_key"\s*:\s*"([^"]+)"`,
        `"token"\s*:\s*"([^"]+)"`,
    }
}
```

#### 1.3 Real-Time Terminal Display
```go
type TerminalDisplay struct {
    // Uses termbox-go or similar
    screen      *Screen
    layout      *Layout
    
    // Display components
    headerBox   *HeaderWidget    // Target info, stats
    vulnTable   *TableWidget     // Live vulnerabilities
    statsBox    *StatsWidget     // Capture statistics
    
    // Update rate
    refreshRate time.Duration // 100ms default
}
```

**Display Layout:**
```
┌─────────────────────────────────────────────────┐
│ Target: nginx (12345) | Time: 00:05:23 | ACTIVE │
├─────────────────────────────────────────────────┤
│ VULNERABILITIES DETECTED                        │
│ Time     Severity  Type        Evidence         │
│ 10:23:45 CRITICAL  Password    [REDACTED]       │
│ 10:23:47 HIGH      API Key     sk-****          │
├─────────────────────────────────────────────────┤
│ Stats: 1.2MB captured | 3 vulns | 2 protocols  │
└─────────────────────────────────────────────────┘
```

#### 1.4 Structured Logging
```go
type EventLogger struct {
    file       *os.File
    encoder    *json.Encoder
    
    // Log everything as JSONL
    LogVulnerability(v Vulnerability) error
    LogStatistics(s Statistics) error
    LogError(e error) error
}

// Output format (one JSON object per line)
{"time":"2024-01-20T10:23:45Z","type":"vuln","severity":"critical","data":{...}}
{"time":"2024-01-20T10:23:46Z","type":"stats","bytes":1234567,"events":543}
```

### Testing & Validation
- Unit tests for each capture method
- Integration test with sample processes
- Performance baseline (handle 1MB/s streams)
- Manual testing with known vulnerable applications

### Success Criteria
- [ ] Capture STDIO from any user process
- [ ] Detect passwords/API keys in JSON and SQL
- [ ] Display vulnerabilities within 1 second
- [ ] Log all events to structured format
- [ ] Run continuously for 1 hour without crashes

---

## Phase 2: Security Hardening (Weeks 3-4)

### Goals
- Implement comprehensive input validation
- Add data sanitization and redaction
- Create ACL system for process monitoring
- Secure configuration management

### Deliverables

#### 2.1 Input Validation Framework
```go
type Validator struct {
    // Validate all user inputs
    ValidateTarget(target string) error
    ValidateFilter(filter string) error
    ValidateConfig(config Config) error
}

// Validation rules
- No path traversal in file targets
- Regex filters must compile
- Config values within acceptable ranges
- Process names sanitized
```

#### 2.2 Data Sanitization Pipeline
```go
type Sanitizer struct {
    rules []RedactionRule
    
    // Redaction strategies
    RedactPassword(s string) string    // Replace with ****
    RedactAPIKey(s string) string      // Show first 4 chars
    RedactCreditCard(s string) string  // Show last 4 digits
    
    // Configurable rules
    LoadRules(file string) error
}

// Default redaction rules
rules:
  - pattern: "password.*?['\"]([^'\"]+)"
    action: full_redact
  - pattern: "sk-[a-zA-Z0-9]{48}"
    action: partial_redact
    show: prefix:4
```

#### 2.3 Access Control Lists
```go
type ACLManager struct {
    rules   []ACLRule
    default ACLAction
    
    // Check if monitoring is allowed
    CanMonitor(process Process) (bool, string)
    
    // Audit all decisions
    auditLog *AuditLogger
}

// Config example
acl:
  default: deny
  rules:
    - name: "web_servers"
      pattern: "nginx|apache|httpd"
      action: allow
    - name: "databases"
      pattern: "mysql|postgres|mongo"
      action: allow
      alert_on_monitor: true
    - name: "system"
      uid: 0
      action: deny
      reason: "Cannot monitor root processes"
```

#### 2.4 Secure Configuration
```go
type SecureConfig struct {
    // Encrypted at rest
    encryptedFile string
    key           []byte
    
    // Validation on load
    validator     *ConfigValidator
    
    // Change tracking
    version       int
    lastModified  time.Time
}

// Config encryption using AES-256-GCM
// Key derivation from passphrase using Argon2
```

### Testing & Validation
- Security audit of all input paths
- Fuzzing tests for validators
- Verify ACL enforcement
- Test config encryption/decryption

### Success Criteria
- [ ] No injection vulnerabilities
- [ ] All sensitive data redacted
- [ ] ACL rules properly enforced
- [ ] Configuration encrypted at rest
- [ ] Security audit passed

---

## Phase 3: Performance & Scale (Weeks 5-6)

### Goals
- Integrate eBPF for efficient capture
- Implement asynchronous processing pipeline
- Add performance monitoring
- Optimize for high-volume streams

### Deliverables

#### 3.1 eBPF Integration
```go
type eBPFCapture struct {
    program  *ebpf.Program
    perfMap  *ebpf.Map
    
    // BPF program attaches to:
    // - sys_write/sys_read
    // - Process creation/termination
    // - File descriptor operations
}

// BPF program (simplified)
int trace_stdio(struct pt_regs *ctx) {
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    
    // Filter by target PIDs
    if (!is_target_pid(pid)) return 0;
    
    // Capture data
    struct event e = {};
    e.pid = pid;
    e.fd = PT_REGS_PARM1(ctx);
    e.size = PT_REGS_PARM3(ctx);
    
    // Send to userspace
    events.perf_submit(ctx, &e, sizeof(e));
    return 0;
}
```

#### 3.2 Async Processing Pipeline
```go
type Pipeline struct {
    // Stage 1: Capture (10k buffer)
    captureWorkers  int
    captureQueue    chan RawEvent
    
    // Stage 2: Parse (5k buffer)  
    parseWorkers    int
    parseQueue      chan ParsedEvent
    
    // Stage 3: Analyze (1k buffer)
    analyzeWorkers  int
    vulnQueue       chan Vulnerability
    
    // Metrics
    processed       atomic.Uint64
    dropped         atomic.Uint64
}

// Worker pool management
// Back-pressure handling
// Graceful degradation
```

#### 3.3 Performance Monitoring
```go
type Metrics struct {
    // Capture metrics
    EventsPerSecond   float64
    BytesPerSecond    float64
    
    // Processing metrics
    ParseLatencyP50   time.Duration
    ParseLatencyP99   time.Duration
    
    // Detection metrics
    VulnDetectLatency time.Duration
    
    // Resource usage
    CPUPercent        float64
    MemoryMB          int64
}

// Prometheus export
// Real-time dashboard
```

#### 3.4 Optimization Techniques
```go
// Memory pooling
var eventPool = sync.Pool{
    New: func() interface{} {
        return &Event{}
    },
}

// Zero-copy techniques
// SIMD for pattern matching
// Bloom filters for quick checks
```

### Testing & Validation
- Load testing (10k events/sec)
- Memory leak detection
- CPU profiling
- Latency measurements

### Success Criteria
- [ ] Handle 10k events/second
- [ ] <1s vulnerability detection
- [ ] <500MB memory usage
- [ ] No memory leaks in 24h
- [ ] CPU usage <50% at peak

---

## Phase 4: Advanced Features (Weeks 7-8)

### Goals
- Implement plugin sandboxing
- Add graph-based sudo detection
- Create Serial Studio integration
- Production deployment readiness

### Deliverables

#### 4.1 Plugin Sandboxing
```go
type PluginSandbox struct {
    // Container isolation
    runtime    ContainerRuntime
    image      string
    
    // Resource limits
    cpuShares  int
    memoryMB   int
    
    // Security
    seccomp    []string
    capabilities []string
}

// Plugin communication via gRPC
service DissectorPlugin {
    rpc Identify(Data) returns (MatchResult);
    rpc Dissect(Data) returns (DissectedFrame);
    rpc GetMetadata(Empty) returns (PluginInfo);
}
```

#### 4.2 Sudo Chain Detection
```go
type SudoChainAnalyzer struct {
    graph    *ProcessGraph
    chains   []*PrivilegeChain
    
    // Detection algorithms
    DetectDirectEscalation() []Chain
    DetectIndirectEscalation() []Chain
    DetectTimingAttacks() []Chain
}

// Graph analysis using:
// - Dijkstra for shortest privilege path
// - Community detection for related processes
// - Temporal analysis for timing patterns
```

#### 4.3 Serial Studio Integration
```go
type SerialStudioExporter struct {
    // Real-time export
    websocket  *WSConnection
    
    // Format conversion
    ConvertToSerialStudio(stream Stream) *Project
    
    // Widget mapping
    MapVulnToWidget(vuln Vulnerability) Widget
}

// Serial Studio project format
{
  "version": "3.0",
  "widgets": [
    {
      "type": "console",
      "title": "Vulnerability Stream",
      "data": "$.vulnerabilities"
    }
  ]
}
```

#### 4.4 Production Features
```go
// High availability
type HAManager struct {
    primary   *CenterModule
    secondary *CenterModule
    failover  *FailoverController
}

// Monitoring integration
type MonitoringExporter struct {
    prometheus *PrometheusExporter
    grafana    *GrafanaDashboard
    alerts     *AlertManager
}

// Deployment
type Deployment struct {
    kubernetes *K8sManifests
    docker     *DockerCompose
    systemd    *SystemdUnits
}
```

### Testing & Validation
- Plugin security audit
- Graph algorithm correctness
- Serial Studio compatibility
- Production load testing
- Deployment validation

### Success Criteria
- [ ] Zero plugin escapes
- [ ] Sudo chains detected accurately
- [ ] Serial Studio integration working
- [ ] Production deployment ready
- [ ] Documentation complete

---

## Risk Management

### Technical Risks
1. **eBPF compatibility**: Not all kernels support it
   - Mitigation: Robust fallback chain
   
2. **Performance bottlenecks**: High-volume streams
   - Mitigation: Profiling and optimization
   
3. **Plugin security**: Malicious plugins
   - Mitigation: Strict sandboxing

### Schedule Risks
1. **Complexity creep**: Features take longer
   - Mitigation: Strict phase boundaries
   
2. **Testing time**: Security testing extensive
   - Mitigation: Automated test suite

---

## Success Metrics

### Phase 1
- Basic monitoring operational
- 3+ vulnerabilities detected
- 1 hour stability

### Phase 2  
- Security audit passed
- Zero vulnerabilities
- ACL system working

### Phase 3
- 10k events/second
- <1s detection latency
- 24h stability

### Phase 4
- Plugin system secure
- All features integrated
- Production ready

---

## Next Steps
1. Begin Phase 1 implementation
2. Weekly progress reviews
3. Adjust timeline as needed
4. Document lessons learned