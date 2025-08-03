# Phase 1: Local STDIO Implementation Guide

## Week 1: Core Infrastructure

### Day 1-2: Stream Capture Foundation

```go
// internal/stream/capture.go
type StreamCapture interface {
    Start(pid int) error
    Stop() error
    Subscribe(handler StreamHandler) error
    SetFilter(filter Filter) error
}

// internal/stream/stdio.go
type StdioStream struct {
    pid      int
    pty      *os.File
    buffer   *RingBuffer
    filters  []Filter
    handlers []StreamHandler
}
```

**Key Tasks:**
- Implement PTY attachment for process monitoring
- Create ring buffer for efficient stream storage
- Build subscription mechanism for modules
- Add basic filtering (regex, keywords)

### Day 3-4: Hierarchical Processing Pipeline

```go
// Three-stage processing hierarchy
type ProcessingPipeline struct {
    S1 EdgeFilter    // Microsecond filtering
    S2 ShallowAnalyzer // Millisecond analysis  
    S3 DeepAnalyzer    // Second-level analysis
}
```

**S1: Edge Filtering**
- Pattern matching (SQL injection, command injection)
- Known malicious signatures
- Rate limiting and deduplication

**S2: Shallow Analysis**
- Context extraction
- Threat scoring
- Quick classification

**S3: Deep Analysis**
- Multi-LLM consensus
- Historical correlation
- Advanced threat detection

### Day 5: Buffer Management

```go
type SmartBuffer struct {
    window   time.Duration
    maxSize  int
    priority Priority
    
    // Dynamic sizing based on threat level
    AdjustWindow(threat ThreatLevel)
    
    // Context preservation
    GetContext(before, after time.Duration) []byte
}
```

## Week 2: Multi-LLM Integration

### Day 6-7: LLM Interface & Mock Implementation

```go
// internal/ai/analyzer.go
type StreamAnalyzer interface {
    Analyze(ctx context.Context, stream StreamData) (*Analysis, error)
    GetCapabilities() []Capability
    GetConfidence(analysisType string) float64
}

// Mock for testing
type MockAnalyzer struct {
    responses map[string]*Analysis
    latency   time.Duration
}
```

### Day 8-9: Consensus Engine

```go
type ConsensusEngine struct {
    analyzers []StreamAnalyzer
    weights   map[string]float64
    
    // Voting strategies
    SimpleVote(analyses []*Analysis) *Decision
    WeightedVote(analyses []*Analysis) *Decision
    RequireQuorum(analyses []*Analysis, quorum float64) *Decision
}
```

**Consensus Strategies:**
1. **Unanimous**: All agree → immediate action
2. **Majority**: >50% agree → action with logging
3. **Weighted**: Based on analyzer expertise
4. **Escalation**: Disagreement → human review

### Day 10: Real LLM Integration

**Claude Integration** (Direct):
```go
type ClaudeAnalyzer struct {
    client *anthropic.Client
    model  string
}
```

**Gemini Integration** (via A2A):
```go
type GeminiAnalyzer struct {
    a2aClient *cyreal.A2AClient
    agentID   string
}
```

## Testing Strategy

### Unit Tests
```bash
# Stream capture
go test ./internal/stream/... -v

# Filtering accuracy
go test ./internal/filter/... -bench=.

# LLM mocks
go test ./internal/ai/... -cover
```

### Integration Tests
```go
// test/integration/attack_simulation_test.go
func TestSQLInjectionDetection(t *testing.T) {
    stream := setupTestStream()
    
    // Simulate attack
    stream.Write([]byte("'; DROP TABLE users; --"))
    
    // Verify detection
    alert := waitForAlert(t, 5*time.Second)
    assert.Equal(t, "SQL_INJECTION", alert.Type)
    assert.Greater(t, alert.Confidence, 0.95)
}
```

### Attack Simulations
1. **Command Injection**: `; cat /etc/passwd`
2. **SQL Injection**: `' OR '1'='1`
3. **Path Traversal**: `../../../etc/passwd`
4. **Prompt Injection**: `Ignore previous instructions`
5. **Data Exfiltration**: Base64 encoded data

## Performance Targets

### Latency Requirements
- S1 Filtering: < 1ms
- S2 Analysis: < 10ms  
- S3 Consensus: < 100ms
- Total pipeline: < 150ms

### Throughput Targets
- 10,000 commands/second per stream
- 100 concurrent streams
- 1 million events/hour

### Resource Limits
- Memory: < 1GB per 100 streams
- CPU: < 1 core per 100 streams
- Disk: Configurable buffer size

## Module Integration

### Stream Command
```bash
# Start monitoring
strigoi> stream setup stdio local --pid 12345

# List active streams
strigoi> stream list

# View stream details
strigoi> stream info STREAM-001

# Subscribe module
strigoi> stream subscribe STREAM-001 sql-injection-detector
```

### Module API
```go
type StreamModule interface {
    OnStreamData(data StreamData) error
    GetSubscriptionFilter() Filter
    GetPriority() Priority
}
```

## Success Metrics

### Week 1 Deliverables
- [ ] Working STDIO capture
- [ ] Basic filtering operational
- [ ] Buffer management tested
- [ ] Pipeline architecture proven

### Week 2 Deliverables
- [ ] Mock LLM analysis working
- [ ] Consensus engine tested
- [ ] Real LLM integration (1+ model)
- [ ] Attack detection demonstrated

### End of Phase 1
- [ ] Detects 95% of test attacks
- [ ] < 1% false positive rate
- [ ] < 150ms detection latency
- [ ] Clean, extensible architecture

---

*"Start simple, build solid, extend infinitely"*