# Phase 1: Local STDIO Implementation Guide

## Overview

This guide provides concrete implementation details for Phase 1 of Strigoi's stream infrastructure. We'll build a working system that monitors local process I/O streams and applies multi-LLM analysis for real-time threat detection.

## Architecture for Phase 1

```
┌─────────────────────────────────────────────────────┐
│                  CLI Interface                       │
│  stream setup stdio <process>                       │
│  stream list | start | stop | filter               │
└─────────────────────┬───────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────┐
│              Stream Manager (S3)                     │
│  • Stream lifecycle control                         │
│  • Resource governance                              │
│  • Configuration management                         │
└─────────────────────┬───────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────┐
│           Stream Router (S2)                         │
│  • Event distribution                               │
│  • Filter application                               │
│  • Load balancing                                   │
└─────────────────────┬───────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────┐
│         STDIO Streams (S1)                          │
│  • Process attachment                               │
│  • I/O capture                                      │
│  • Buffer management                                │
└─────────────────────┬───────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────┐
│         Multi-LLM Analysis                          │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐              │
│  │ Claude  │ │ Gemini  │ │ Patterns │              │
│  │ Direct  │ │   A2A   │ │ Library  │              │
│  └─────────┘ └─────────┘ └─────────┘              │
└─────────────────────────────────────────────────────┘
```

## Implementation Components

### 1. Core Stream Interface

```go
// internal/stream/types.go
package stream

import (
    "context"
    "time"
)

type StreamType string

const (
    StreamTypeSTDIO   StreamType = "stdio"
    StreamTypeRemote  StreamType = "remote"  // Future
    StreamTypeSerial  StreamType = "serial"  // Future
    StreamTypeNetwork StreamType = "network" // Future
)

type StreamConfig struct {
    ID          string
    Type        StreamType
    Target      string      // Process name/PID for STDIO
    Filters     []Filter
    BufferSize  int
    Timeout     time.Duration
}

type StreamData struct {
    StreamID  string
    Timestamp time.Time
    Source    string      // stdin/stdout/stderr
    Data      []byte
    Metadata  map[string]interface{}
}

type Stream interface {
    ID() string
    Type() StreamType
    Config() StreamConfig
    
    // Lifecycle
    Start(ctx context.Context) error
    Stop() error
    Status() StreamStatus
    
    // Data flow
    Subscribe(handler StreamHandler) (Subscription, error)
    SetFilter(filter Filter) error
    
    // Metrics
    Stats() StreamStats
}

type StreamHandler func(data StreamData) error

type Subscription interface {
    ID() string
    Unsubscribe() error
}
```

### 2. STDIO Stream Implementation

```go
// internal/stream/stdio/stream.go
package stdio

import (
    "bufio"
    "context"
    "fmt"
    "io"
    "os/exec"
    "sync"
    
    "strigoi/internal/stream"
)

type STDIOStream struct {
    config      stream.StreamConfig
    cmd         *exec.Cmd
    stdin       io.WriteCloser
    stdout      io.ReadCloser
    stderr      io.ReadCloser
    
    subscribers map[string]stream.StreamHandler
    filters     []stream.Filter
    
    ctx         context.Context
    cancel      context.CancelFunc
    wg          sync.WaitGroup
    mu          sync.RWMutex
    
    stats       *StreamStats
    status      stream.StreamStatus
}

func NewSTDIOStream(config stream.StreamConfig) (*STDIOStream, error) {
    ctx, cancel := context.WithCancel(context.Background())
    
    return &STDIOStream{
        config:      config,
        subscribers: make(map[string]stream.StreamHandler),
        ctx:         ctx,
        cancel:      cancel,
        stats:       &StreamStats{},
        status:      stream.StatusCreated,
    }, nil
}

func (s *STDIOStream) Start(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.status != stream.StatusCreated {
        return fmt.Errorf("stream already started")
    }
    
    // Create command
    s.cmd = exec.CommandContext(s.ctx, s.config.Target)
    
    // Set up pipes
    var err error
    s.stdin, err = s.cmd.StdinPipe()
    if err != nil {
        return fmt.Errorf("stdin pipe: %w", err)
    }
    
    s.stdout, err = s.cmd.StdoutPipe()
    if err != nil {
        return fmt.Errorf("stdout pipe: %w", err)
    }
    
    s.stderr, err = s.cmd.StderrPipe()
    if err != nil {
        return fmt.Errorf("stderr pipe: %w", err)
    }
    
    // Start process
    if err := s.cmd.Start(); err != nil {
        return fmt.Errorf("start process: %w", err)
    }
    
    // Start monitoring goroutines
    s.wg.Add(3)
    go s.monitorStream("stdout", s.stdout)
    go s.monitorStream("stderr", s.stderr)
    go s.monitorProcess()
    
    s.status = stream.StatusRunning
    return nil
}

func (s *STDIOStream) monitorStream(source string, reader io.Reader) {
    defer s.wg.Done()
    
    scanner := bufio.NewScanner(reader)
    scanner.Buffer(make([]byte, s.config.BufferSize), s.config.BufferSize)
    
    for scanner.Scan() {
        data := scanner.Bytes()
        
        // Apply filters
        if !s.shouldProcess(data) {
            continue
        }
        
        // Create stream data
        streamData := stream.StreamData{
            StreamID:  s.config.ID,
            Timestamp: time.Now(),
            Source:    source,
            Data:      append([]byte{}, data...), // Copy
            Metadata: map[string]interface{}{
                "process": s.config.Target,
                "pid":     s.cmd.Process.Pid,
            },
        }
        
        // Notify subscribers
        s.notifySubscribers(streamData)
        
        // Update stats
        s.stats.Update(len(data))
    }
}
```

### 3. Multi-LLM Analysis Engine

```go
// internal/analysis/multi_llm.go
package analysis

import (
    "context"
    "sync"
    "time"
    
    "strigoi/internal/stream"
)

type LLMAnalyzer interface {
    Name() string
    Analyze(ctx context.Context, data stream.StreamData) (*AnalysisResult, error)
    Priority() int
}

type AnalysisResult struct {
    Analyzer    string
    Confidence  float64
    ThreatLevel ThreatLevel
    Findings    []Finding
    Timestamp   time.Time
}

type MultiLLMEngine struct {
    analyzers []LLMAnalyzer
    consensus *ConsensusEngine
    cache     *AnalysisCache
    
    concurrency int
    timeout     time.Duration
}

func (e *MultiLLMEngine) Analyze(data stream.StreamData) (*ConsensusResult, error) {
    ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
    defer cancel()
    
    // Check cache
    if cached := e.cache.Get(data); cached != nil {
        return cached, nil
    }
    
    // Run parallel analysis
    results := make(chan *AnalysisResult, len(e.analyzers))
    var wg sync.WaitGroup
    
    for _, analyzer := range e.analyzers {
        wg.Add(1)
        go func(a LLMAnalyzer) {
            defer wg.Done()
            
            result, err := a.Analyze(ctx, data)
            if err != nil {
                log.Warnf("analyzer %s failed: %v", a.Name(), err)
                return
            }
            
            results <- result
        }(analyzer)
    }
    
    // Wait for results
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Collect results
    var analysisResults []*AnalysisResult
    for result := range results {
        analysisResults = append(analysisResults, result)
    }
    
    // Build consensus
    consensus := e.consensus.Build(analysisResults)
    
    // Cache result
    e.cache.Set(data, consensus)
    
    return consensus, nil
}
```

### 4. Claude Direct Analyzer

```go
// internal/analysis/claude/analyzer.go
package claude

import (
    "context"
    "fmt"
    
    "strigoi/internal/analysis"
    "strigoi/internal/stream"
)

type ClaudeAnalyzer struct {
    client *ClaudeClient
    prompts map[string]string
}

func (a *ClaudeAnalyzer) Analyze(ctx context.Context, data stream.StreamData) (*analysis.AnalysisResult, error) {
    // Build analysis prompt
    prompt := a.buildPrompt(data)
    
    // Call Claude API
    response, err := a.client.Analyze(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("claude analysis: %w", err)
    }
    
    // Parse response
    result := a.parseResponse(response, data)
    
    return result, nil
}

func (a *ClaudeAnalyzer) buildPrompt(data stream.StreamData) string {
    return fmt.Sprintf(`Analyze this process I/O for security threats:

Stream: %s
Source: %s
Data: %s

Look for:
1. Command injection attempts
2. Data exfiltration patterns
3. Privilege escalation
4. Suspicious system calls
5. Anomalous behavior

Respond with:
- Threat level (none/low/medium/high/critical)
- Confidence (0-100)
- Specific findings
- Recommended actions
`, data.StreamID, data.Source, string(data.Data))
}
```

### 5. Gemini A2A Bridge Analyzer

```go
// internal/analysis/gemini/analyzer.go
package gemini

import (
    "context"
    "encoding/json"
    
    "strigoi/internal/analysis"
    "strigoi/internal/stream"
)

type GeminiAnalyzer struct {
    bridge *A2ABridge
    contextWindow int
    history *ContextHistory
}

func (a *GeminiAnalyzer) Analyze(ctx context.Context, data stream.StreamData) (*analysis.AnalysisResult, error) {
    // Add to context history
    a.history.Add(data)
    
    // Build context-aware prompt
    contextData := a.history.GetRecentContext(a.contextWindow)
    
    query := map[string]interface{}{
        "action": "analyze_security",
        "stream_data": data,
        "context": contextData,
        "focus_areas": []string{
            "temporal_patterns",
            "cross_stream_correlation",
            "historical_anomalies",
            "attack_campaigns",
        },
    }
    
    // Query via A2A bridge
    response, err := a.bridge.Query(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("gemini a2a: %w", err)
    }
    
    // Parse response
    var result analysis.AnalysisResult
    if err := json.Unmarshal(response, &result); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }
    
    return &result, nil
}
```

### 6. Consensus Engine

```go
// internal/analysis/consensus.go
package analysis

type ConsensusEngine struct {
    threshold float64
    weights   map[string]float64
}

type ConsensusResult struct {
    FinalThreatLevel ThreatLevel
    Confidence       float64
    Agreement        float64
    Findings         []Finding
    Dissent          []Disagreement
    Action           ResponseAction
}

func (e *ConsensusEngine) Build(results []*AnalysisResult) *ConsensusResult {
    if len(results) == 0 {
        return &ConsensusResult{
            FinalThreatLevel: ThreatLevelNone,
            Confidence:       0,
        }
    }
    
    // Calculate weighted threat scores
    var totalWeight, threatScore float64
    threatCounts := make(map[ThreatLevel]int)
    
    for _, result := range results {
        weight := e.weights[result.Analyzer]
        if weight == 0 {
            weight = 1.0
        }
        
        totalWeight += weight
        threatScore += float64(result.ThreatLevel) * weight * result.Confidence
        threatCounts[result.ThreatLevel]++
    }
    
    // Determine consensus threat level
    avgThreatScore := threatScore / totalWeight / 100
    finalThreat := ThreatLevel(avgThreatScore)
    
    // Calculate agreement
    var maxCount int
    for _, count := range threatCounts {
        if count > maxCount {
            maxCount = count
        }
    }
    agreement := float64(maxCount) / float64(len(results))
    
    // Determine action based on consensus
    action := e.determineAction(finalThreat, agreement)
    
    return &ConsensusResult{
        FinalThreatLevel: finalThreat,
        Confidence:       avgThreatScore * 100,
        Agreement:        agreement,
        Action:           action,
    }
}
```

### 7. CLI Commands

```go
// internal/core/console_stream.go
package core

func (c *Console) initStreamCommands() {
    // stream setup stdio <process>
    c.RegisterCommand("stream setup stdio", c.cmdStreamSetupSTDIO,
        "Setup STDIO monitoring for a process")
    
    // stream list
    c.RegisterCommand("stream list", c.cmdStreamList,
        "List active streams")
    
    // stream start <id>
    c.RegisterCommand("stream start", c.cmdStreamStart,
        "Start monitoring a stream")
    
    // stream stop <id>
    c.RegisterCommand("stream stop", c.cmdStreamStop,
        "Stop monitoring a stream")
    
    // stream filter <id> <pattern>
    c.RegisterCommand("stream filter", c.cmdStreamFilter,
        "Add filter to stream")
    
    // stream analyze <id>
    c.RegisterCommand("stream analyze", c.cmdStreamAnalyze,
        "Run multi-LLM analysis on stream")
}

func (c *Console) cmdStreamSetupSTDIO(args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: stream setup stdio <process>")
    }
    
    config := stream.StreamConfig{
        ID:     fmt.Sprintf("stdio-%s-%d", args[0], time.Now().Unix()),
        Type:   stream.StreamTypeSTDIO,
        Target: args[0],
        BufferSize: 64 * 1024,
        Timeout: 30 * time.Second,
    }
    
    // Create stream
    s, err := stdio.NewSTDIOStream(config)
    if err != nil {
        return fmt.Errorf("create stream: %w", err)
    }
    
    // Register with manager
    if err := c.streamManager.Register(s); err != nil {
        return fmt.Errorf("register stream: %w", err)
    }
    
    // Subscribe analyzer
    sub, err := s.Subscribe(c.analyzeStream)
    if err != nil {
        return fmt.Errorf("subscribe analyzer: %w", err)
    }
    
    c.Printf("Stream created: %s\n", config.ID)
    c.Printf("Subscription: %s\n", sub.ID())
    
    return nil
}
```

## Testing Strategy

### 1. Unit Tests

```go
// internal/stream/stdio/stream_test.go
func TestSTDIOStreamCapture(t *testing.T) {
    // Create test stream
    config := stream.StreamConfig{
        ID:     "test-stream",
        Type:   stream.StreamTypeSTDIO,
        Target: "echo",
        BufferSize: 1024,
    }
    
    s, err := NewSTDIOStream(config)
    require.NoError(t, err)
    
    // Subscribe to events
    var captured []stream.StreamData
    _, err = s.Subscribe(func(data stream.StreamData) error {
        captured = append(captured, data)
        return nil
    })
    require.NoError(t, err)
    
    // Start stream
    ctx := context.Background()
    require.NoError(t, s.Start(ctx))
    
    // Write test data
    s.stdin.Write([]byte("test data\n"))
    
    // Wait for capture
    time.Sleep(100 * time.Millisecond)
    
    // Verify
    assert.Len(t, captured, 1)
    assert.Equal(t, "test data", string(captured[0].Data))
}
```

### 2. Attack Simulations

```go
// tests/attacks/injection_test.go
func TestCommandInjectionDetection(t *testing.T) {
    engine := setupTestEngine(t)
    
    attacks := []struct {
        name     string
        input    string
        expected ThreatLevel
    }{
        {
            name:     "SQL injection",
            input:    "'; DROP TABLE users; --",
            expected: ThreatLevelHigh,
        },
        {
            name:     "Command injection",
            input:    "test; cat /etc/passwd",
            expected: ThreatLevelCritical,
        },
        {
            name:     "Path traversal",
            input:    "../../../etc/passwd",
            expected: ThreatLevelMedium,
        },
    }
    
    for _, tt := range attacks {
        t.Run(tt.name, func(t *testing.T) {
            data := stream.StreamData{
                Data: []byte(tt.input),
            }
            
            result, err := engine.Analyze(data)
            require.NoError(t, err)
            
            assert.GreaterOrEqual(t, result.FinalThreatLevel, tt.expected)
        })
    }
}
```

## Deployment Checklist

### Week 1 Tasks
- [ ] Implement core stream interface
- [ ] Build STDIO stream capture
- [ ] Create stream manager with governors
- [ ] Add basic CLI commands
- [ ] Write unit tests

### Week 2 Tasks
- [ ] Integrate Claude analyzer
- [ ] Build Gemini A2A bridge
- [ ] Implement consensus engine
- [ ] Add pattern library
- [ ] Run attack simulations

### Performance Targets
- Stream capture: <1ms latency
- LLM analysis: <100ms for critical paths
- Throughput: 1000+ events/second
- Memory: <100MB for typical workload
- CPU: <10% overhead on monitored process

## Next Steps

After Phase 1 is stable:
1. Begin Phase 2 planning (Remote STDIO)
2. Gather feedback from early users
3. Refine LLM prompts based on results
4. Build pattern library from detected attacks
5. Prepare for distributed architecture

---

*"Start with working code, iterate to excellence"*