package probe

import (
	"fmt"
	"testing"
	"time"
)

// TestLoadTest_BreadthScenario tests many concurrent sessions.
func TestLoadTest_BreadthScenario(t *testing.T) {
	config := LoadTestConfig{
		// Breadth: Many concurrent sessions (reduced for testing)
		ConcurrentSessions: 20,
		SessionDuration:    5 * time.Second,

		// Moderate depth per session
		FramesPerSession: 10,
		FrameSize:        1024,

		// Mixed protocols
		ProtocolMix: map[string]float64{
			"HTTP":      0.5,
			"WebSocket": 0.3,
			"gRPC":      0.2,
		},

		// Session lifecycle
		SessionCreationRate: 500 * time.Millisecond,
		SessionDeathRate:    1 * time.Second,

		// Moderate vulnerability injection
		VulnerabilityRate: 0.1, // 10% of frames

		// Resource limits
		MaxMemoryMB: 1024,
		MaxCPU:      80.0,
		Timeout:     10 * time.Second,
	}

	tester := NewLoadTester(config)
	results, err := tester.Run()
	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	// Verify results
	t.Logf("Breadth Test Results:")
	t.Logf("  Sessions Created: %d", results.SessionsCreated)
	t.Logf("  Sessions Completed: %d", results.SessionsCompleted)
	t.Logf("  Frames Processed: %d", results.FramesProcessed)
	t.Logf("  Vulnerabilities Detected: %d", results.VulnsDetected)
	t.Logf("  Average Frame Latency: %v", results.AvgFrameLatency)
	t.Logf("  P95 Frame Latency: %v", results.P95FrameLatency)
	t.Logf("  P99 Frame Latency: %v", results.P99FrameLatency)

	// Performance assertions
	if results.AvgFrameLatency > 100*time.Millisecond {
		t.Errorf("Average frame latency too high: %v", results.AvgFrameLatency)
	}

	if results.P99FrameLatency > 500*time.Millisecond {
		t.Errorf("P99 frame latency too high: %v", results.P99FrameLatency)
	}

	if len(results.Errors) > 10 {
		t.Errorf("Too many errors: %d", len(results.Errors))
		for i, err := range results.Errors[:10] {
			t.Logf("  Error %d: %v", i, err)
		}
	}
}

// TestLoadTest_DepthScenario tests large individual sessions.
func TestLoadTest_DepthScenario(t *testing.T) {
	config := LoadTestConfig{
		// Fewer sessions but deeper
		ConcurrentSessions: 10,
		SessionDuration:    30 * time.Second,

		// Many frames per session
		FramesPerSession: 1000,
		FrameSize:        4096, // Larger frames

		// Single protocol for consistency
		ProtocolMix: map[string]float64{
			"HTTP": 1.0,
		},

		// Slow session lifecycle
		SessionCreationRate: 5 * time.Second,
		SessionDeathRate:    10 * time.Second,

		// Higher vulnerability rate for stress testing
		VulnerabilityRate: 0.25,

		// Resource limits
		MaxMemoryMB: 2048,
		MaxCPU:      80.0,
		Timeout:     60 * time.Second,
	}

	tester := NewLoadTester(config)
	results, err := tester.Run()
	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	// Verify results
	t.Logf("Depth Test Results:")
	t.Logf("  Total Frames: %d", results.FramesProcessed)
	t.Logf("  Total Bytes: %d MB", results.BytesProcessed/1024/1024)
	t.Logf("  Vulnerabilities: %d", results.VulnsDetected)
	t.Logf("  Detection Rate: %.2f%%", float64(results.VulnsDetected)/float64(results.FramesProcessed)*100)

	// Check vulnerability detection rate
	expectedVulns := float64(results.FramesProcessed) * config.VulnerabilityRate * 0.8 // 80% detection rate
	if float64(results.VulnsDetected) < expectedVulns {
		t.Errorf("Low vulnerability detection: got %d, expected at least %.0f",
			results.VulnsDetected, expectedVulns)
	}
}

// TestLoadTest_MixedProtocolScenario tests protocol diversity.
func TestLoadTest_MixedProtocolScenario(t *testing.T) {
	config := LoadTestConfig{
		ConcurrentSessions: 50,
		SessionDuration:    20 * time.Second,
		FramesPerSession:   100,
		FrameSize:          2048,

		// Even protocol distribution
		ProtocolMix: map[string]float64{
			"HTTP":      0.25,
			"WebSocket": 0.25,
			"gRPC":      0.25,
			"Generic":   0.25,
		},

		SessionCreationRate: 1 * time.Second,
		SessionDeathRate:    2 * time.Second,
		VulnerabilityRate:   0.15,

		MaxMemoryMB: 1024,
		MaxCPU:      80.0,
		Timeout:     30 * time.Second,
	}

	tester := NewLoadTester(config)
	results, err := tester.Run()
	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	t.Logf("Mixed Protocol Test Results:")
	t.Logf("  Sessions: %d", results.SessionsCreated)
	t.Logf("  Frames: %d", results.FramesProcessed)
	t.Logf("  Throughput: %.2f frames/sec",
		float64(results.FramesProcessed)/results.EndTime.Sub(results.StartTime).Seconds())
}

// TestLoadTest_RapidLifecycleScenario tests rapid session creation/destruction.
func TestLoadTest_RapidLifecycleScenario(t *testing.T) {
	config := LoadTestConfig{
		ConcurrentSessions: 20,
		SessionDuration:    5 * time.Second,
		FramesPerSession:   10,
		FrameSize:          512,

		ProtocolMix: map[string]float64{
			"HTTP": 1.0,
		},

		// Very rapid lifecycle
		SessionCreationRate: 100 * time.Millisecond,
		SessionDeathRate:    200 * time.Millisecond,

		VulnerabilityRate: 0.05,

		MaxMemoryMB: 512,
		MaxCPU:      80.0,
		Timeout:     20 * time.Second,
	}

	tester := NewLoadTester(config)
	results, err := tester.Run()
	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	t.Logf("Rapid Lifecycle Test Results:")
	t.Logf("  Sessions Created: %d", results.SessionsCreated)
	t.Logf("  Sessions Completed: %d", results.SessionsCompleted)
	t.Logf("  Session Churn Rate: %.2f sessions/sec",
		float64(results.SessionsCreated)/results.EndTime.Sub(results.StartTime).Seconds())

	// Should create many sessions in short time
	if results.SessionsCreated < 50 {
		t.Errorf("Expected more sessions created, got %d", results.SessionsCreated)
	}
}

// BenchmarkLoadTest_Throughput benchmarks frame processing throughput.
func BenchmarkLoadTest_Throughput(b *testing.B) {
	config := LoadTestConfig{
		ConcurrentSessions:  10,
		SessionDuration:     10 * time.Second,
		FramesPerSession:    b.N / 10, // Distribute b.N frames across sessions
		FrameSize:           1024,
		ProtocolMix:         map[string]float64{"HTTP": 1.0},
		SessionCreationRate: 10 * time.Second, // Don't create new sessions
		SessionDeathRate:    10 * time.Second, // Don't kill sessions
		VulnerabilityRate:   0.1,
		MaxMemoryMB:         1024,
		MaxCPU:              80.0,
		Timeout:             1 * time.Hour,
	}

	tester := NewLoadTester(config)

	b.ResetTimer()
	results, err := tester.Run()
	if err != nil {
		b.Fatalf("Load test failed: %v", err)
	}

	b.StopTimer()

	framesPerSecond := float64(results.FramesProcessed) / results.EndTime.Sub(results.StartTime).Seconds()
	b.ReportMetric(framesPerSecond, "frames/sec")
	b.ReportMetric(float64(results.BytesProcessed)/1024/1024, "MB_processed")
	b.ReportMetric(float64(results.AvgFrameLatency.Microseconds()), "avg_latency_us")
}

// TestLoadTest_StressScenario is a comprehensive stress test.
func TestLoadTest_StressScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	config := LoadTestConfig{
		// High concurrency
		ConcurrentSessions: 200,
		SessionDuration:    30 * time.Second,

		// Many frames
		FramesPerSession: 500,
		FrameSize:        2048,

		// All protocols
		ProtocolMix: map[string]float64{
			"HTTP":      0.3,
			"WebSocket": 0.3,
			"gRPC":      0.2,
			"Generic":   0.2,
		},

		// Aggressive lifecycle
		SessionCreationRate: 250 * time.Millisecond,
		SessionDeathRate:    500 * time.Millisecond,

		// High vulnerability rate
		VulnerabilityRate: 0.3,

		// Resource limits
		MaxMemoryMB: 4096,
		MaxCPU:      90.0,
		Timeout:     2 * time.Minute,
	}

	tester := NewLoadTester(config)
	results, err := tester.Run()
	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	// Generate report
	report := fmt.Sprintf(`
Stress Test Report:
==================
Duration: %v
Sessions Created: %d
Sessions Completed: %d
Frames Processed: %d
Bytes Processed: %d MB
Vulnerabilities Detected: %d

Performance Metrics:
- Throughput: %.2f frames/sec
- Avg Latency: %v
- P95 Latency: %v
- P99 Latency: %v
- Max Latency: %v

Session Metrics:
- Creation Rate: %.2f sessions/sec
- Completion Rate: %.2f sessions/sec
- Avg Session Duration: %v

Error Summary:
- Total Errors: %d
`,
		results.EndTime.Sub(results.StartTime),
		results.SessionsCreated,
		results.SessionsCompleted,
		results.FramesProcessed,
		results.BytesProcessed/1024/1024,
		results.VulnsDetected,
		float64(results.FramesProcessed)/results.EndTime.Sub(results.StartTime).Seconds(),
		results.AvgFrameLatency,
		results.P95FrameLatency,
		results.P99FrameLatency,
		results.MaxFrameLatency,
		float64(results.SessionsCreated)/results.EndTime.Sub(results.StartTime).Seconds(),
		float64(results.SessionsCompleted)/results.EndTime.Sub(results.StartTime).Seconds(),
		results.AvgSessionDuration,
		len(results.Errors),
	)

	t.Log(report)

	// Stress test should handle load without crashing
	if len(results.Errors) > int(float64(results.FramesProcessed)*0.01) { // 1% error rate
		t.Errorf("Error rate too high: %.2f%%",
			float64(len(results.Errors))/float64(results.FramesProcessed)*100)
	}
}
