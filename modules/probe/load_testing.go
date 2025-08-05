package probe

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// LoadTestConfig defines configuration for load testing.
type LoadTestConfig struct {
	// Breadth testing
	ConcurrentSessions int           // Number of concurrent sessions
	SessionDuration    time.Duration // How long each session lasts

	// Depth testing
	FramesPerSession int // Number of frames per session
	FrameSize        int // Average size of each frame

	// Mixed protocol testing
	ProtocolMix map[string]float64 // Protocol distribution (e.g., "HTTP": 0.5)

	// Session lifecycle testing
	SessionCreationRate time.Duration // How often to create new sessions
	SessionDeathRate    time.Duration // How often to complete sessions

	// Vulnerability injection
	VulnerabilityRate float64 // Percentage of frames with vulnerabilities

	// Resource limits
	MaxMemoryMB int64         // Maximum memory usage
	MaxCPU      float64       // Maximum CPU percentage
	Timeout     time.Duration // Overall test timeout
}

// LoadTestResults contains the results of a load test.
type LoadTestResults struct {
	StartTime time.Time
	EndTime   time.Time

	// Performance metrics
	SessionsCreated   int64
	SessionsCompleted int64
	FramesProcessed   int64
	BytesProcessed    int64
	VulnsDetected     int64

	// Timing metrics
	AvgSessionDuration time.Duration
	AvgFrameLatency    time.Duration
	MaxFrameLatency    time.Duration
	P95FrameLatency    time.Duration
	P99FrameLatency    time.Duration

	// Resource metrics
	MaxMemoryUsed int64
	AvgCPUUsed    float64

	// Error metrics
	Errors []error

	// Detailed latencies for percentile calculation
	frameLatencies []time.Duration
	mu             sync.Mutex
}

// LoadTester performs load testing on the Strigoi platform.
type LoadTester struct {
	config         LoadTestConfig
	sessionManager *SessionManager
	dissectors     []Dissector
	results        *LoadTestResults
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewLoadTester creates a new load tester.
func NewLoadTester(config LoadTestConfig) *LoadTester {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)

	// Create session manager with reasonable timeout
	sessionManager := NewSessionManager(30*time.Second, 5*time.Second)

	// Add vulnerability checkers
	sessionManager.vulnerabilityCheckers = []SessionVulnerabilityChecker{
		NewAuthSessionChecker(),
		NewTokenLeakageChecker(),
		NewSessionTimeoutChecker(),
		NewWeakSessionChecker(),
		NewCrossSessionChecker(),
	}

	// Initialize dissectors
	dissectors := []Dissector{
		NewHTTPDissector(),
		NewGRPCDissectorV2(),
		NewWebSocketDissector(),
		NewJSONDissector(),
		NewSQLDissector(),
		NewPlainTextDissector(),
	}

	return &LoadTester{
		config:         config,
		sessionManager: sessionManager,
		dissectors:     dissectors,
		results: &LoadTestResults{
			frameLatencies: make([]time.Duration, 0, config.ConcurrentSessions*config.FramesPerSession),
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Run executes the load test.
func (lt *LoadTester) Run() (*LoadTestResults, error) {
	lt.results.StartTime = time.Now()
	defer func() {
		lt.results.EndTime = time.Now()
	}()

	// Start resource monitoring
	lt.wg.Add(1)
	go lt.monitorResources()

	// Start session creation
	lt.wg.Add(1)
	go lt.sessionCreator()

	// Start session completion
	lt.wg.Add(1)
	go lt.sessionCompleter()

	// Start concurrent sessions
	sessionStarted := make(chan struct{}, lt.config.ConcurrentSessions)
	for i := 0; i < lt.config.ConcurrentSessions; i++ {
		lt.wg.Add(1)
		sessionID := fmt.Sprintf("session-%d", i)
		go func(id string) {
			sessionStarted <- struct{}{}
			lt.runSession(id)
		}(sessionID)
	}

	// Wait for all initial sessions to start
	for i := 0; i < lt.config.ConcurrentSessions; i++ {
		<-sessionStarted
	}
	close(sessionStarted)

	// Wait for completion or timeout
	done := make(chan struct{})
	go func() {
		lt.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Normal completion
	case <-lt.ctx.Done():
		// Timeout
		lt.cancel()
	}

	// Calculate final metrics
	lt.calculateMetrics()

	return lt.results, nil
}

// runSession simulates a single session.
func (lt *LoadTester) runSession(sessionID string) {

	protocol := lt.selectProtocol()
	atomic.AddInt64(&lt.results.SessionsCreated, 1)

	for i := 0; i < lt.config.FramesPerSession; i++ {
		select {
		case <-lt.ctx.Done():
			return
		default:
		}

		// Generate frame data
		frameData := lt.generateFrameData(protocol, i)

		// Process frame
		start := time.Now()
		err := lt.processFrame(sessionID, protocol, frameData)
		latency := time.Since(start)

		// Record metrics
		lt.results.mu.Lock()
		lt.results.frameLatencies = append(lt.results.frameLatencies, latency)
		lt.results.mu.Unlock()

		atomic.AddInt64(&lt.results.FramesProcessed, 1)
		atomic.AddInt64(&lt.results.BytesProcessed, int64(len(frameData)))

		if err != nil {
			lt.results.mu.Lock()
			lt.results.Errors = append(lt.results.Errors, err)
			lt.results.mu.Unlock()
		}

		// Simulate inter-frame delay
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
	}

	// Complete session
	lt.sessionManager.CompleteSession(sessionID)
	atomic.AddInt64(&lt.results.SessionsCompleted, 1)
}

// processFrame processes a single frame through the dissection pipeline.
func (lt *LoadTester) processFrame(sessionID, protocol string, data []byte) error {
	// Find appropriate dissector
	var selectedDissector Dissector
	var maxConfidence float64

	for _, dissector := range lt.dissectors {
		if identified, confidence := dissector.Identify(data); identified && confidence > maxConfidence {
			selectedDissector = dissector
			maxConfidence = confidence
		}
	}

	if selectedDissector == nil {
		return fmt.Errorf("no dissector found for data")
	}

	// Dissect frame
	frame, err := selectedDissector.Dissect(data)
	if err != nil {
		return fmt.Errorf("dissection failed: %w", err)
	}

	// Find vulnerabilities
	vulns := selectedDissector.FindVulnerabilities(frame)
	atomic.AddInt64(&lt.results.VulnsDetected, int64(len(vulns)))

	// Add to session
	return lt.sessionManager.AddFrame(sessionID, protocol, frame)
}

// selectProtocol selects a protocol based on the configured mix.
func (lt *LoadTester) selectProtocol() string {
	r := rand.Float64()
	cumulative := 0.0

	for protocol, weight := range lt.config.ProtocolMix {
		cumulative += weight
		if r <= cumulative {
			return protocol
		}
	}

	// Default to HTTP
	return "HTTP"
}

// generateFrameData generates realistic frame data for testing.
func (lt *LoadTester) generateFrameData(protocol string, frameNum int) []byte {
	switch protocol {
	case "HTTP":
		return lt.generateHTTPFrame(frameNum)
	case "WebSocket":
		return lt.generateWebSocketFrame(frameNum)
	case "gRPC":
		return lt.generateGRPCFrame(frameNum)
	default:
		return lt.generateGenericFrame(frameNum)
	}
}

// generateHTTPFrame generates HTTP request/response data.
func (lt *LoadTester) generateHTTPFrame(frameNum int) []byte {
	// Randomly choose request or response
	if frameNum%2 == 0 {
		// HTTP Request
		sessionID := fmt.Sprintf("SID%d", rand.Intn(1000))

		// Inject vulnerabilities based on rate
		var authHeader string
		if rand.Float64() < lt.config.VulnerabilityRate {
			authHeader = "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.signature\r\n"
		}

		return []byte(fmt.Sprintf(
			"GET /api/data/%d HTTP/1.1\r\n"+
				"Host: example.com\r\n"+
				"Cookie: JSESSIONID=%s\r\n"+
				"%s"+
				"User-Agent: LoadTest/1.0\r\n"+
				"\r\n",
			frameNum, sessionID, authHeader,
		))
	}

	// HTTP Response
	body := fmt.Sprintf(`{"frame":%d,"data":"test"}`, frameNum)

	// Inject API key in response for vulnerability testing
	if rand.Float64() < lt.config.VulnerabilityRate {
		body = fmt.Sprintf(`{"frame":%d,"api_key":"sk_test_1234567890abcdef"}`, frameNum)
	}

	return []byte(fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Type: application/json\r\n"+
			"Content-Length: %d\r\n"+
			"\r\n"+
			"%s",
		len(body), body,
	))
}

// generateWebSocketFrame generates WebSocket frame data.
func (lt *LoadTester) generateWebSocketFrame(frameNum int) []byte {
	if frameNum == 0 {
		// WebSocket handshake
		return []byte(
			"GET /ws HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Upgrade: websocket\r\n" +
				"Connection: Upgrade\r\n" +
				"Sec-WebSocket-Key: x3JJHMbDL1EzLkh9GBhXDw==\r\n" +
				"Sec-WebSocket-Version: 13\r\n" +
				"\r\n",
		)
	}

	// WebSocket data frame
	payload := []byte(fmt.Sprintf(`{"frame":%d,"type":"data"}`, frameNum))
	frame := make([]byte, 2+len(payload))
	frame[0] = 0x81 // FIN=1, Text frame
	frame[1] = byte(len(payload))
	copy(frame[2:], payload)

	return frame
}

// generateGRPCFrame generates gRPC/HTTP2 frame data.
func (lt *LoadTester) generateGRPCFrame(frameNum int) []byte {
	// Simplified HTTP/2 DATA frame
	payload := []byte(fmt.Sprintf("\x00\x00\x00\x00\x10{\"id\":%d}", frameNum))

	frame := make([]byte, 9+len(payload))
	// Frame header
	frame[0] = 0x00 // Length (24-bit)
	frame[1] = 0x00
	frame[2] = byte(len(payload))
	frame[3] = 0x00 // Type: DATA
	frame[4] = 0x00 // Flags
	// Stream ID (32-bit)
	frame[5] = 0x00
	frame[6] = 0x00
	frame[7] = 0x00
	frame[8] = 0x01

	copy(frame[9:], payload)
	return frame
}

// generateGenericFrame generates generic test data.
func (lt *LoadTester) generateGenericFrame(frameNum int) []byte {
	data := make([]byte, lt.config.FrameSize)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}

	// Add some structure for parsing
	copy(data, []byte(fmt.Sprintf("FRAME:%d:", frameNum)))

	return data
}

// sessionCreator creates new sessions at the configured rate.
func (lt *LoadTester) sessionCreator() {
	defer lt.wg.Done()

	ticker := time.NewTicker(lt.config.SessionCreationRate)
	defer ticker.Stop()

	sessionNum := lt.config.ConcurrentSessions

	for {
		select {
		case <-lt.ctx.Done():
			return
		case <-ticker.C:
			sessionNum++
			lt.wg.Add(1)
			newSessionID := fmt.Sprintf("session-%d", sessionNum)
			go func(id string) {
				defer lt.wg.Done()
				lt.runSession(id)
			}(newSessionID)
		}
	}
}

// sessionCompleter completes sessions at the configured rate.
func (lt *LoadTester) sessionCompleter() {
	defer lt.wg.Done()

	ticker := time.NewTicker(lt.config.SessionDeathRate)
	defer ticker.Stop()

	for {
		select {
		case <-lt.ctx.Done():
			return
		case <-ticker.C:
			// Get active sessions
			sessions := lt.sessionManager.GetActiveSessions()
			if len(sessions) > 0 {
				// Complete a random session
				session := sessions[rand.Intn(len(sessions))]
				lt.sessionManager.CompleteSession(session.ID)
			}
		}
	}
}

// monitorResources monitors resource usage during the test.
func (lt *LoadTester) monitorResources() {
	defer lt.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-lt.ctx.Done():
			return
		case <-ticker.C:
			// In a real implementation, we would use runtime.MemStats
			// and system CPU monitoring. For now, we'll use placeholders.

			// This is where you would collect actual metrics
			// For example:
			// var m runtime.MemStats
			// runtime.ReadMemStats(&m)
			// memory := int64(m.Alloc / 1024 / 1024) // MB
		}
	}
}

// calculateMetrics calculates final test metrics.
func (lt *LoadTester) calculateMetrics() {
	lt.results.mu.Lock()
	defer lt.results.mu.Unlock()

	if len(lt.results.frameLatencies) == 0 {
		return
	}

	// Calculate average latency
	var totalLatency time.Duration
	maxLatency := time.Duration(0)

	for _, latency := range lt.results.frameLatencies {
		totalLatency += latency
		if latency > maxLatency {
			maxLatency = latency
		}
	}

	lt.results.AvgFrameLatency = totalLatency / time.Duration(len(lt.results.frameLatencies))
	lt.results.MaxFrameLatency = maxLatency

	// Calculate percentiles
	lt.results.P95FrameLatency = calculatePercentile(lt.results.frameLatencies, 0.95)
	lt.results.P99FrameLatency = calculatePercentile(lt.results.frameLatencies, 0.99)

	// Calculate session duration
	duration := lt.results.EndTime.Sub(lt.results.StartTime)
	if lt.results.SessionsCompleted > 0 {
		lt.results.AvgSessionDuration = duration / time.Duration(lt.results.SessionsCompleted)
	}
}

// calculatePercentile calculates the percentile of a slice of durations.
func calculatePercentile(latencies []time.Duration, percentile float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Sort latencies
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Simple bubble sort for now (could use sort.Slice)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)-1) * percentile)
	return sorted[index]
}
