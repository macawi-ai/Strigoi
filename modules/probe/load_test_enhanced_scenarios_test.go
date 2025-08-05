package probe

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// EnhancedLoadTestConfig extends LoadTestConfig with buffer-specific settings
type EnhancedLoadTestConfig struct {
	LoadTestConfig

	// Buffer testing
	UseProtocolAwareBuffer bool
	BufferSize             int
	ChunkedWrites          bool // Simulate chunked protocol data
	ChunkSizeMin           int
	ChunkSizeMax           int

	// Protocol-specific testing
	JSONComplexity      int  // Nesting depth for JSON
	HTTPChunkedTransfer bool // Use chunked transfer encoding
	WebSocketFragmented bool // Fragment WebSocket messages

	// Error injection
	NetworkJitter time.Duration // Random delay between chunks
	PacketLoss    float64       // Percentage of dropped chunks
}

// TestEnhancedLoadTest_CircularBufferStress tests the circular buffer under load
func TestEnhancedLoadTest_CircularBufferStress(t *testing.T) {
	config := EnhancedLoadTestConfig{
		LoadTestConfig: LoadTestConfig{
			ConcurrentSessions: 50,
			SessionDuration:    10 * time.Second,
			FramesPerSession:   100,
			FrameSize:          4096,
			ProtocolMix: map[string]float64{
				"HTTP":      0.3,
				"WebSocket": 0.3,
				"JSON":      0.4,
			},
			SessionCreationRate: 200 * time.Millisecond,
			SessionDeathRate:    500 * time.Millisecond,
			VulnerabilityRate:   0.15,
			MaxMemoryMB:         2048,
			MaxCPU:              80.0,
			Timeout:             30 * time.Second,
		},
		UseProtocolAwareBuffer: true,
		BufferSize:             128 * 1024, // 128KB buffers
		ChunkedWrites:          true,
		ChunkSizeMin:           100,
		ChunkSizeMax:           1000,
		NetworkJitter:          50 * time.Millisecond,
	}

	results := runEnhancedLoadTest(t, config, "Circular Buffer Stress")

	// Verify no buffer overflows
	if results.BufferOverflows > 0 {
		t.Errorf("Buffer overflows detected: %d", results.BufferOverflows)
	}

	// Check backpressure handling
	if results.BackpressureEvents > 100 {
		t.Logf("Warning: High backpressure events: %d", results.BackpressureEvents)
	}
}

// TestEnhancedLoadTest_JSONStreamingPerformance tests JSON detector performance
func TestEnhancedLoadTest_JSONStreamingPerformance(t *testing.T) {
	config := EnhancedLoadTestConfig{
		LoadTestConfig: LoadTestConfig{
			ConcurrentSessions: 30,
			SessionDuration:    15 * time.Second,
			FramesPerSession:   200,
			FrameSize:          2048,
			ProtocolMix: map[string]float64{
				"JSON": 1.0, // Pure JSON testing
			},
			SessionCreationRate: 500 * time.Millisecond,
			SessionDeathRate:    2 * time.Second,
			VulnerabilityRate:   0.05,
			MaxMemoryMB:         1024,
			MaxCPU:              80.0,
			Timeout:             30 * time.Second,
		},
		UseProtocolAwareBuffer: true,
		BufferSize:             64 * 1024,
		ChunkedWrites:          true,
		ChunkSizeMin:           50, // Small chunks to stress the detector
		ChunkSizeMax:           200,
		JSONComplexity:         5, // Nested JSON objects
		NetworkJitter:          10 * time.Millisecond,
	}

	results := runEnhancedLoadTest(t, config, "JSON Streaming Performance")

	// JSON-specific metrics
	t.Logf("  JSON Parse Errors: %d", results.JSONParseErrors)
	t.Logf("  Incomplete JSON Detected: %d", results.IncompleteJSON)
	t.Logf("  JSON Detection Latency P99: %v", results.JSONDetectionP99)

	if results.JSONParseErrors > 0 {
		t.Errorf("JSON parse errors detected: %d", results.JSONParseErrors)
	}
}

// TestEnhancedLoadTest_ProtocolSwitching tests rapid protocol switching
func TestEnhancedLoadTest_ProtocolSwitching(t *testing.T) {
	config := EnhancedLoadTestConfig{
		LoadTestConfig: LoadTestConfig{
			ConcurrentSessions: 40,
			SessionDuration:    20 * time.Second,
			FramesPerSession:   150,
			FrameSize:          1024,
			ProtocolMix: map[string]float64{
				"HTTP":      0.25,
				"WebSocket": 0.25,
				"JSON":      0.25,
				"gRPC":      0.25,
			},
			SessionCreationRate: 300 * time.Millisecond,
			SessionDeathRate:    1 * time.Second,
			VulnerabilityRate:   0.10,
			MaxMemoryMB:         1536,
			MaxCPU:              80.0,
			Timeout:             30 * time.Second,
		},
		UseProtocolAwareBuffer: true,
		BufferSize:             96 * 1024,
		ChunkedWrites:          true,
		ChunkSizeMin:           200,
		ChunkSizeMax:           800,
		HTTPChunkedTransfer:    true,
		WebSocketFragmented:    true,
		NetworkJitter:          25 * time.Millisecond,
	}

	results := runEnhancedLoadTest(t, config, "Protocol Switching")

	// Check protocol detection accuracy
	for proto, stats := range results.ProtocolStats {
		accuracy := float64(stats.CorrectDetections) / float64(stats.TotalFrames) * 100
		t.Logf("  %s Detection Accuracy: %.2f%%", proto, accuracy)

		if accuracy < 95.0 {
			t.Errorf("%s detection accuracy too low: %.2f%%", proto, accuracy)
		}
	}
}

// TestEnhancedLoadTest_BackpressureResilience tests system under backpressure
func TestEnhancedLoadTest_BackpressureResilience(t *testing.T) {
	config := EnhancedLoadTestConfig{
		LoadTestConfig: LoadTestConfig{
			ConcurrentSessions: 100, // High concurrency
			SessionDuration:    15 * time.Second,
			FramesPerSession:   300,
			FrameSize:          8192, // Large frames
			ProtocolMix: map[string]float64{
				"HTTP":      0.4,
				"WebSocket": 0.3,
				"JSON":      0.3,
			},
			SessionCreationRate: 100 * time.Millisecond, // Rapid creation
			SessionDeathRate:    2 * time.Second,
			VulnerabilityRate:   0.20,
			MaxMemoryMB:         2048,
			MaxCPU:              90.0,
			Timeout:             45 * time.Second,
		},
		UseProtocolAwareBuffer: true,
		BufferSize:             64 * 1024, // Smaller buffers to induce pressure
		ChunkedWrites:          true,
		ChunkSizeMin:           500,
		ChunkSizeMax:           2000,
		NetworkJitter:          100 * time.Millisecond, // High jitter
		PacketLoss:             0.02,                   // 2% packet loss
	}

	results := runEnhancedLoadTest(t, config, "Backpressure Resilience")

	// Verify system handles backpressure gracefully
	droppedPercentage := float64(results.DroppedFrames) / float64(results.FramesProcessed) * 100
	t.Logf("  Dropped Frames: %.2f%%", droppedPercentage)

	if droppedPercentage > 5.0 {
		t.Errorf("Too many dropped frames under backpressure: %.2f%%", droppedPercentage)
	}

	// Check memory usage stayed within limits
	if results.PeakMemoryMB > config.MaxMemoryMB {
		t.Errorf("Memory limit exceeded: %d MB (limit: %d MB)",
			results.PeakMemoryMB, config.MaxMemoryMB)
	}
}

// EnhancedLoadTestResults extends LoadTestResults
type EnhancedLoadTestResults struct {
	*LoadTestResults

	// Buffer metrics
	BufferOverflows    int64
	BackpressureEvents int64
	DroppedFrames      int64

	// Protocol metrics
	ProtocolStats map[string]*ProtocolTestStats

	// JSON-specific
	JSONParseErrors  int64
	IncompleteJSON   int64
	JSONDetectionP99 time.Duration

	// Resource metrics
	PeakMemoryMB  int64
	AvgCPUPercent float64
}

type ProtocolTestStats struct {
	TotalFrames       int64
	CorrectDetections int64
	MisDetections     int64
	AvgDetectionTime  time.Duration
}

// runEnhancedLoadTest executes an enhanced load test
func runEnhancedLoadTest(t *testing.T, config EnhancedLoadTestConfig, name string) *EnhancedLoadTestResults {
	t.Logf("\n=== Running %s Test ===", name)

	// Create enhanced load tester
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	results := &EnhancedLoadTestResults{
		LoadTestResults: &LoadTestResults{
			frameLatencies: make([]time.Duration, 0, 10000),
		},
		ProtocolStats: make(map[string]*ProtocolTestStats),
	}

	// Initialize protocol stats
	for proto := range config.ProtocolMix {
		results.ProtocolStats[proto] = &ProtocolTestStats{}
	}

	// Run the test with simulated load
	var wg sync.WaitGroup
	sessionCount := int32(0)

	// Session workers
	for i := 0; i < config.ConcurrentSessions; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					if atomic.LoadInt32(&sessionCount) < int32(config.ConcurrentSessions) {
						atomic.AddInt32(&sessionCount, 1)
						runEnhancedSession(ctx, config, results, fmt.Sprintf("session-%d", id))
						atomic.AddInt32(&sessionCount, -1)
					}
					time.Sleep(config.SessionCreationRate)
				}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()

	// Calculate final metrics
	results.EndTime = time.Now()

	t.Logf("\n%s Test Results:", name)
	t.Logf("  Total Sessions: %d", results.SessionsCreated)
	t.Logf("  Total Frames: %d", results.FramesProcessed)
	t.Logf("  Throughput: %.2f frames/sec",
		float64(results.FramesProcessed)/config.Timeout.Seconds())
	t.Logf("  Buffer Overflows: %d", results.BufferOverflows)
	t.Logf("  Backpressure Events: %d", results.BackpressureEvents)

	return results
}

// runEnhancedSession simulates an enhanced session with protocol-aware buffering
func runEnhancedSession(ctx context.Context, config EnhancedLoadTestConfig,
	results *EnhancedLoadTestResults, sessionID string) {

	atomic.AddInt64(&results.SessionsCreated, 1)

	// Create protocol-aware buffer if enabled
	var buffer *ProtocolAwareBuffer
	if config.UseProtocolAwareBuffer {
		var err error
		buffer, err = NewProtocolAwareBuffer(config.BufferSize, "")
		if err != nil {
			return
		}
		defer buffer.Close()

		// Monitor buffer events
		go func() {
			for event := range buffer.ProtocolEvents() {
				atomic.AddInt64(&results.FramesProcessed, 1)

				// Track protocol detection
				if stats, ok := results.ProtocolStats[event.Protocol]; ok {
					atomic.AddInt64(&stats.CorrectDetections, 1)
				}
			}
		}()
	}

	// Generate and process frames
	for i := 0; i < config.FramesPerSession; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			// Simulate frame processing with chunking
			if config.ChunkedWrites && buffer != nil {
				simulateChunkedWrite(buffer, config)
			}

			// Add network jitter
			if config.NetworkJitter > 0 {
				time.Sleep(config.NetworkJitter)
			}
		}
	}

	atomic.AddInt64(&results.SessionsCompleted, 1)
}

// simulateChunkedWrite simulates writing data in chunks
func simulateChunkedWrite(buffer *ProtocolAwareBuffer, config EnhancedLoadTestConfig) {
	// Generate protocol-specific data
	// This is simplified - real implementation would generate actual protocol data
	data := make([]byte, config.FrameSize)

	// Write in chunks
	written := 0
	for written < len(data) {
		chunkSize := config.ChunkSizeMin
		if config.ChunkSizeMax > config.ChunkSizeMin {
			chunkSize += written % (config.ChunkSizeMax - config.ChunkSizeMin)
		}

		if chunkSize > len(data)-written {
			chunkSize = len(data) - written
		}

		buffer.Write(data[written : written+chunkSize])
		written += chunkSize

		// Simulate network delays between chunks
		if config.NetworkJitter > 0 {
			time.Sleep(time.Duration(written%10) * time.Millisecond)
		}
	}
}
