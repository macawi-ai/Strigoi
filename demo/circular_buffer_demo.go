//go:build demo
// +build demo

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/macawi-ai/strigoi/modules/probe"
)

func main() {
	fmt.Println("🦊 Strigoi Circular Buffer Demo")
	fmt.Println("================================")

	// Create capture engine with 256KB buffers and newline delimiter
	engine := probe.NewCaptureEngineV2(256*1024, []byte("\n"))

	// Optional: Enable strace fallback
	// engine.EnableStrace()

	// Start a demo process
	fmt.Println("\n📋 Starting demo process...")
	cmd := exec.Command("bash", "-c", `
		echo "Demo process started"
		for i in {1..100}; do
			echo "[$(date +%H:%M:%S)] stdout: Processing item $i"
			echo "[$(date +%H:%M:%S)] stderr: Status update $i" >&2
			
			# Simulate variable output rate
			if [ $((i % 10)) -eq 0 ]; then
				echo "=== Checkpoint $i reached ==="
				sleep 0.5
			else
				sleep 0.1
			fi
		done
		echo "Demo process completed"
	`)

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start demo process: %v", err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid
	fmt.Printf("✅ Process started with PID: %d\n", pid)

	// Attach to the process
	fmt.Printf("\n🔗 Attaching to process %d...\n", pid)
	if err := engine.Attach(pid); err != nil {
		log.Fatalf("Failed to attach: %v", err)
	}
	defer engine.Detach(pid)

	// Get event channel
	events, err := engine.GetEvents(pid)
	if err != nil {
		log.Fatalf("Failed to get events: %v", err)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start capture loop
	fmt.Println("\n📊 Starting stream capture...")
	fmt.Println("Press Ctrl+C to stop\n")

	captureStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond) // 20Hz capture
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := engine.CaptureStreams(pid); err != nil {
					fmt.Printf("⚠️  Capture error: %v\n", err)
				}
			case <-captureStop:
				return
			}
		}
	}()

	// Stats display ticker
	statsTicker := time.NewTicker(2 * time.Second)
	defer statsTicker.Stop()

	// Event processing
	eventCount := make(map[string]int)

	fmt.Println("📝 Stream events:")
	fmt.Println("─────────────────")

loop:
	for {
		select {
		case event := <-events:
			eventCount[event.Stream]++

			// Display event with color coding
			switch event.Stream {
			case "stdout":
				fmt.Printf("\033[32m[STDOUT]\033[0m %s", event.Data)
			case "stderr":
				fmt.Printf("\033[33m[STDERR]\033[0m %s", event.Data)
			case "stdin":
				fmt.Printf("\033[36m[STDIN]\033[0m %s", event.Data)
			}

		case <-statsTicker.C:
			// Display buffer statistics
			stats, err := engine.GetBufferStats(pid)
			if err != nil {
				continue
			}

			fmt.Println("\n📈 Buffer Statistics:")
			fmt.Println("────────────────────")

			for stream, stat := range stats {
				if m, ok := stat.(map[string]interface{}); ok {
					if stream == "capture_stats" {
						if cs, ok := m.(*probe.CaptureStats); ok {
							fmt.Printf("Capture: method=%s, success=%d/%d, bytes=%d\n",
								cs.Method, cs.Successful, cs.Attempts, cs.BytesCapured)
						}
					} else {
						usage := m["usage_pct"].(float64)
						written := m["written"].(uint64)
						dropped := m["dropped"].(uint64)
						events := m["events_sent"].(uint64)
						interval := m["scan_interval"].(time.Duration)
						rate := m["write_rate_bps"].(uint64)

						fmt.Printf("%s: usage=%.1f%%, written=%d, events=%d, dropped=%d, rate=%.1fKB/s, scan=%v\n",
							stream, usage, written, events, dropped, float64(rate)/1024, interval)
					}
				}
			}
			fmt.Printf("\nEvents captured: stdout=%d, stderr=%d, stdin=%d\n",
				eventCount["stdout"], eventCount["stderr"], eventCount["stdin"])
			fmt.Println()

		case <-sigChan:
			fmt.Println("\n\n🛑 Shutting down...")
			break loop
		}
	}

	// Stop capture
	close(captureStop)
	time.Sleep(100 * time.Millisecond)

	// Final stats
	fmt.Println("\n📊 Final Statistics:")
	fmt.Println("───────────────────")

	finalStats, _ := engine.GetBufferStats(pid)
	for stream, stat := range finalStats {
		if m, ok := stat.(map[string]interface{}); ok && stream != "capture_stats" {
			fmt.Printf("\n%s buffer:\n", stream)
			for k, v := range m {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
	}

	fmt.Printf("\n✅ Total events processed: %d\n",
		eventCount["stdout"]+eventCount["stderr"]+eventCount["stdin"])

	fmt.Println("\n🦊 Demo completed!")
}
