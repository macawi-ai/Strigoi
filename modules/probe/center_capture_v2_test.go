package probe

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestCaptureEngineV2_BasicCapture(t *testing.T) {
	// Create test engine with newline delimiter
	engine := NewCaptureEngineV2(64*1024, []byte("\n")) // 64KB buffers

	// Enable strace for capturing terminal output
	if err := engine.EnableStrace(); err != nil {
		t.Skipf("Strace not available: %v", err)
	}

	// Start a test process that outputs data
	cmd := exec.Command("bash", "-c", `
		for i in {1..10}; do
			echo "stdout line $i"
			echo "stderr line $i" >&2
			sleep 0.1
		done
	`)

	// Start the process
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid
	t.Logf("Started test process with PID: %d", pid)

	// Attach to process
	if err := engine.Attach(pid); err != nil {
		t.Fatalf("Failed to attach to process: %v", err)
	}
	defer engine.Detach(pid)

	// Get event channel
	events, err := engine.GetEvents(pid)
	if err != nil {
		t.Fatalf("Failed to get event channel: %v", err)
	}

	// Start capture loop
	done := make(chan struct{})
	go func() {
		for i := 0; i < 20; i++ { // Capture for 2 seconds
			engine.CaptureStreams(pid)
			time.Sleep(100 * time.Millisecond)
		}
		close(done)
	}()

	// Collect events
	var collectedEvents []StreamEvent
	timeout := time.After(3 * time.Second)

loop:
	for {
		select {
		case event := <-events:
			collectedEvents = append(collectedEvents, event)
			t.Logf("Event: [%s] %q", event.Stream, event.Data)
		case <-done:
			// Give a bit more time for final events
			time.Sleep(200 * time.Millisecond)
			break loop
		case <-timeout:
			t.Log("Timeout reached")
			break loop
		}
	}

	// Wait for process to complete
	cmd.Wait()

	// Verify we got events
	if len(collectedEvents) == 0 {
		t.Error("No events captured")
	}

	// Count by stream
	streamCounts := make(map[string]int)
	for _, e := range collectedEvents {
		streamCounts[e.Stream]++
	}

	t.Logf("Events captured: stdout=%d, stderr=%d",
		streamCounts["stdout"], streamCounts["stderr"])

	// Get buffer stats
	stats, err := engine.GetBufferStats(pid)
	if err != nil {
		t.Errorf("Failed to get buffer stats: %v", err)
	}

	// Log buffer performance
	for stream, stat := range stats {
		if m, ok := stat.(map[string]interface{}); ok {
			t.Logf("%s buffer: %+v", stream, m)
		}
	}
}

func TestCaptureEngineV2_HighThroughput(t *testing.T) {
	// Create engine with small delimiter for stress testing
	engine := NewCaptureEngineV2(256*1024, []byte("\n")) // 256KB buffers

	// Start a process that generates lots of output
	cmd := exec.Command("bash", "-c", `
		for i in {1..1000}; do
			echo "High throughput test line $i with some padding to make it longer"
		done
	`)

	// Capture output to verify later
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run test command: %v", err)
	}

	expectedLines := len(output) / 50 // Rough estimate
	t.Logf("Expected approximately %d lines", expectedLines)

	// Now run the same command with capture
	cmd = exec.Command("bash", "-c", `
		for i in {1..1000}; do
			echo "High throughput test line $i with some padding to make it longer"
		done
	`)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid

	// Attach and capture
	if err := engine.Attach(pid); err != nil {
		t.Fatalf("Failed to attach: %v", err)
	}
	defer engine.Detach(pid)

	events, _ := engine.GetEvents(pid)

	// Aggressive capture loop
	done := make(chan struct{})
	go func() {
		start := time.Now()
		for time.Since(start) < 2*time.Second {
			engine.CaptureStreams(pid)
			time.Sleep(10 * time.Millisecond) // 100Hz capture rate
		}
		close(done)
	}()

	// Count events
	eventCount := 0
	for {
		select {
		case <-events:
			eventCount++
		case <-done:
			// Drain remaining events
			timeout := time.After(500 * time.Millisecond)
		drain:
			for {
				select {
				case <-events:
					eventCount++
				case <-timeout:
					break drain
				}
			}
			goto finished
		}
	}

finished:
	cmd.Wait()

	t.Logf("Captured %d events", eventCount)

	// Get final stats
	stats, _ := engine.GetBufferStats(pid)
	if stdoutStats, ok := stats["stdout"].(map[string]interface{}); ok {
		t.Logf("Stdout buffer stats: written=%v, dropped=%v, events=%v",
			stdoutStats["written"], stdoutStats["dropped"], stdoutStats["events_sent"])

		// Check adaptive scan interval
		if interval, ok := stdoutStats["scan_interval"].(time.Duration); ok {
			t.Logf("Adaptive scan interval: %v", interval)
			if interval > time.Millisecond {
				t.Log("Warning: Scan interval might be too high for real-time capture")
			}
		}
	}
}

func TestCaptureEngineV2_Backpressure(t *testing.T) {
	// Very small buffers to test backpressure
	engine := NewCaptureEngineV2(4096, []byte("\n")) // 4KB buffers

	// Generate more data than buffer can hold
	cmd := exec.Command("bash", "-c", `
		# Generate 100KB of data quickly
		for i in {1..2000}; do
			echo "Line $i: $(printf '=%.0s' {1..40})"
		done
	`)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid

	if err := engine.Attach(pid); err != nil {
		t.Fatalf("Failed to attach: %v", err)
	}
	defer engine.Detach(pid)

	// Slow reader to create backpressure
	events, _ := engine.GetEvents(pid)

	// Capture but read slowly
	go func() {
		for i := 0; i < 10; i++ {
			engine.CaptureStreams(pid)
			time.Sleep(50 * time.Millisecond)
		}
	}()

	time.Sleep(600 * time.Millisecond)

	// Now read some events
	readCount := 0
	timeout := time.After(100 * time.Millisecond)
loop:
	for {
		select {
		case <-events:
			readCount++
		case <-timeout:
			break loop
		}
	}

	cmd.Wait()

	stats, _ := engine.GetBufferStats(pid)
	if stdoutStats, ok := stats["stdout"].(map[string]interface{}); ok {
		t.Logf("Backpressure test - Stdout stats: %+v", stdoutStats)

		// We expect some drops due to backpressure
		if dropped, ok := stdoutStats["dropped"].(uint64); ok && dropped == 0 {
			t.Log("Warning: Expected some dropped data due to backpressure")
		}

		// Check if backpressure was triggered
		if bp, ok := stdoutStats["backpressure"].(bool); ok && bp {
			t.Log("Backpressure was triggered as expected")
		}
	}
}

func TestCaptureEngineV2_ProcessTermination(t *testing.T) {
	engine := NewCaptureEngineV2(64*1024, []byte("\n"))

	// Start a process that terminates quickly
	cmd := exec.Command("echo", "quick test")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	pid := cmd.Process.Pid

	// Attach
	if err := engine.Attach(pid); err != nil {
		t.Fatalf("Failed to attach: %v", err)
	}

	// Process might already be done
	cmd.Wait()

	// Try to capture from terminated process
	err := engine.CaptureStreams(pid)
	if err != nil {
		t.Logf("Expected error capturing from terminated process: %v", err)
	}

	// Detach should still work
	if err := engine.Detach(pid); err != nil {
		t.Errorf("Failed to detach from terminated process: %v", err)
	}
}

// Helper interface for test and benchmark compatibility
type testHelper interface {
	Fatalf(format string, args ...interface{})
}

// Helper to create a test script file
func createTestScript(t testHelper, content string) string {
	file, err := os.CreateTemp("", "test_script_*.sh")
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	if _, err := file.WriteString("#!/bin/bash\n" + content); err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}

	if err := file.Chmod(0755); err != nil {
		t.Fatalf("Failed to chmod test script: %v", err)
	}

	name := file.Name()
	file.Close()
	return name
}

func BenchmarkCaptureEngineV2(b *testing.B) {
	engine := NewCaptureEngineV2(1024*1024, []byte("\n")) // 1MB buffers

	// Create a process that generates continuous output
	script := createTestScript(b, `
		while true; do
			echo "Benchmark line with timestamp: $(date +%s.%N)"
		done
	`)
	defer os.Remove(script)

	cmd := exec.Command(script)
	if err := cmd.Start(); err != nil {
		b.Fatalf("Failed to start benchmark process: %v", err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid

	if err := engine.Attach(pid); err != nil {
		b.Fatalf("Failed to attach: %v", err)
	}
	defer engine.Detach(pid)

	b.ResetTimer()

	// Benchmark capture rate
	for i := 0; i < b.N; i++ {
		engine.CaptureStreams(pid)
	}

	b.StopTimer()

	// Report stats
	stats, _ := engine.GetBufferStats(pid)
	if stdoutStats, ok := stats["stdout"].(map[string]interface{}); ok {
		if written, ok := stdoutStats["written"].(uint64); ok {
			b.Logf("Total bytes written: %d", written)
			b.Logf("Throughput: %.2f MB/s", float64(written)/b.Elapsed().Seconds()/1024/1024)
		}
	}
}
