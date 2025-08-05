package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/macawi-ai/strigoi/modules/probe"
)

// LoadTestRunner orchestrates load testing
type LoadTestRunner struct {
	scenario   string
	duration   time.Duration
	sessions   int
	bufferSize int
	output     string
	verbose    bool
}

func main() {
	runner := &LoadTestRunner{}

	flag.StringVar(&runner.scenario, "scenario", "mixed",
		"Test scenario: breadth, depth, mixed, lifecycle, buffer, json, protocol, backpressure, all")
	flag.DurationVar(&runner.duration, "duration", 30*time.Second,
		"Test duration")
	flag.IntVar(&runner.sessions, "sessions", 50,
		"Number of concurrent sessions")
	flag.IntVar(&runner.bufferSize, "buffer", 128*1024,
		"Buffer size in bytes")
	flag.StringVar(&runner.output, "output", "",
		"Output file for results (JSON)")
	flag.BoolVar(&runner.verbose, "verbose", false,
		"Verbose output")

	flag.Parse()

	if err := runner.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Load test failed: %v\n", err)
		os.Exit(1)
	}
}

func (r *LoadTestRunner) Run() error {
	fmt.Printf("Strigoi Load Test Runner\n")
	fmt.Printf("========================\n")
	fmt.Printf("Scenario: %s\n", r.scenario)
	fmt.Printf("Duration: %v\n", r.duration)
	fmt.Printf("Sessions: %d\n", r.sessions)
	fmt.Printf("Buffer Size: %d KB\n", r.bufferSize/1024)
	fmt.Printf("\n")

	// System info
	fmt.Printf("System Info:\n")
	fmt.Printf("  CPUs: %d\n", runtime.NumCPU())
	fmt.Printf("  GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("  Go Version: %s\n", runtime.Version())
	fmt.Printf("\n")

	// Run scenarios
	results := make(map[string]*probe.LoadTestResults)

	scenarios := r.getScenarios()
	for _, scenario := range scenarios {
		fmt.Printf("Running %s scenario...\n", scenario)

		config := r.getConfig(scenario)
		tester := probe.NewLoadTester(config)

		start := time.Now()
		result, err := tester.Run()
		if err != nil {
			return fmt.Errorf("%s scenario failed: %w", scenario, err)
		}

		elapsed := time.Since(start)
		results[scenario] = result

		// Print summary
		r.printSummary(scenario, result, elapsed)
	}

	// Save results if requested
	if r.output != "" {
		if err := r.saveResults(results); err != nil {
			return fmt.Errorf("failed to save results: %w", err)
		}
		fmt.Printf("\nResults saved to: %s\n", r.output)
	}

	return nil
}

func (r *LoadTestRunner) getScenarios() []string {
	if r.scenario == "all" {
		return []string{
			"breadth", "depth", "mixed", "lifecycle",
			"buffer", "json", "protocol", "backpressure",
		}
	}
	return []string{r.scenario}
}

func (r *LoadTestRunner) getConfig(scenario string) probe.LoadTestConfig {
	base := probe.LoadTestConfig{
		ConcurrentSessions: r.sessions,
		SessionDuration:    r.duration,
		MaxMemoryMB:        2048,
		MaxCPU:             80.0,
		Timeout:            r.duration + 10*time.Second,
	}

	switch scenario {
	case "breadth":
		// Many concurrent sessions, moderate depth
		base.FramesPerSession = 50
		base.FrameSize = 1024
		base.ProtocolMix = map[string]float64{
			"HTTP": 0.5, "WebSocket": 0.3, "JSON": 0.2,
		}
		base.SessionCreationRate = 200 * time.Millisecond
		base.SessionDeathRate = 1 * time.Second
		base.VulnerabilityRate = 0.1

	case "depth":
		// Fewer sessions, many frames each
		base.ConcurrentSessions = r.sessions / 5
		base.FramesPerSession = 1000
		base.FrameSize = 4096
		base.ProtocolMix = map[string]float64{"HTTP": 1.0}
		base.SessionCreationRate = 2 * time.Second
		base.SessionDeathRate = 10 * time.Second
		base.VulnerabilityRate = 0.2

	case "mixed":
		// Mixed protocols
		base.FramesPerSession = 100
		base.FrameSize = 2048
		base.ProtocolMix = map[string]float64{
			"HTTP": 0.25, "WebSocket": 0.25,
			"JSON": 0.25, "gRPC": 0.25,
		}
		base.SessionCreationRate = 500 * time.Millisecond
		base.SessionDeathRate = 2 * time.Second
		base.VulnerabilityRate = 0.15

	case "lifecycle":
		// Rapid session creation/destruction
		base.FramesPerSession = 20
		base.FrameSize = 512
		base.ProtocolMix = map[string]float64{"HTTP": 1.0}
		base.SessionCreationRate = 100 * time.Millisecond
		base.SessionDeathRate = 200 * time.Millisecond
		base.VulnerabilityRate = 0.05

	case "buffer":
		// Circular buffer stress test
		base.FramesPerSession = 200
		base.FrameSize = r.bufferSize / 100 // Variable frame sizes
		base.ProtocolMix = map[string]float64{
			"HTTP": 0.4, "JSON": 0.6,
		}
		base.SessionCreationRate = 300 * time.Millisecond
		base.SessionDeathRate = 1 * time.Second
		base.VulnerabilityRate = 0.1

	case "json":
		// JSON streaming focus
		base.FramesPerSession = 300
		base.FrameSize = 2048
		base.ProtocolMix = map[string]float64{"JSON": 1.0}
		base.SessionCreationRate = 500 * time.Millisecond
		base.SessionDeathRate = 2 * time.Second
		base.VulnerabilityRate = 0.05

	case "protocol":
		// Protocol switching stress
		base.FramesPerSession = 150
		base.FrameSize = 1536
		base.ProtocolMix = map[string]float64{
			"HTTP": 0.2, "WebSocket": 0.2,
			"JSON": 0.2, "gRPC": 0.2,
			"SQL": 0.1, "PlainText": 0.1,
		}
		base.SessionCreationRate = 400 * time.Millisecond
		base.SessionDeathRate = 1500 * time.Millisecond
		base.VulnerabilityRate = 0.12

	case "backpressure":
		// Backpressure resilience
		base.ConcurrentSessions = r.sessions * 2 // Double sessions
		base.FramesPerSession = 500
		base.FrameSize = 8192 // Large frames
		base.ProtocolMix = map[string]float64{
			"HTTP": 0.5, "JSON": 0.5,
		}
		base.SessionCreationRate = 50 * time.Millisecond // Very fast
		base.SessionDeathRate = 3 * time.Second
		base.VulnerabilityRate = 0.25

	default:
		// Default mixed scenario
		base.FramesPerSession = 100
		base.FrameSize = 2048
		base.ProtocolMix = map[string]float64{
			"HTTP": 0.4, "WebSocket": 0.3, "JSON": 0.3,
		}
		base.SessionCreationRate = 500 * time.Millisecond
		base.SessionDeathRate = 2 * time.Second
		base.VulnerabilityRate = 0.1
	}

	return base
}

func (r *LoadTestRunner) printSummary(scenario string, result *probe.LoadTestResults, elapsed time.Duration) {
	fmt.Printf("\n%s Scenario Results:\n", scenario)
	fmt.Printf("  Duration: %v\n", elapsed)
	fmt.Printf("  Sessions: %d created, %d completed\n",
		result.SessionsCreated, result.SessionsCompleted)
	fmt.Printf("  Frames: %d processed (%.2f frames/sec)\n",
		result.FramesProcessed,
		float64(result.FramesProcessed)/elapsed.Seconds())
	fmt.Printf("  Data: %.2f MB processed (%.2f MB/sec)\n",
		float64(result.BytesProcessed)/1024/1024,
		float64(result.BytesProcessed)/1024/1024/elapsed.Seconds())
	fmt.Printf("  Vulnerabilities: %d detected\n", result.VulnsDetected)

	if r.verbose {
		fmt.Printf("  Latencies:\n")
		fmt.Printf("    Average: %v\n", result.AvgFrameLatency)
		fmt.Printf("    P95: %v\n", result.P95FrameLatency)
		fmt.Printf("    P99: %v\n", result.P99FrameLatency)
		fmt.Printf("    Max: %v\n", result.MaxFrameLatency)

		if result.MaxMemoryUsed > 0 {
			fmt.Printf("  Resources:\n")
			fmt.Printf("    Peak Memory: %d MB\n", result.MaxMemoryUsed/1024/1024)
			fmt.Printf("    Avg CPU: %.1f%%\n", result.AvgCPUUsed)
		}

		if len(result.Errors) > 0 {
			fmt.Printf("  Errors: %d\n", len(result.Errors))
			for i, err := range result.Errors[:min(5, len(result.Errors))] {
				fmt.Printf("    %d: %v\n", i+1, err)
			}
		}
	}
}

func (r *LoadTestRunner) saveResults(results map[string]*probe.LoadTestResults) error {
	file, err := os.Create(r.output)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	output := map[string]interface{}{
		"timestamp": time.Now(),
		"config": map[string]interface{}{
			"scenario":    r.scenario,
			"duration":    r.duration,
			"sessions":    r.sessions,
			"buffer_size": r.bufferSize,
		},
		"system": map[string]interface{}{
			"cpus":       runtime.NumCPU(),
			"gomaxprocs": runtime.GOMAXPROCS(0),
			"go_version": runtime.Version(),
		},
		"results": results,
	}

	return encoder.Encode(output)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
