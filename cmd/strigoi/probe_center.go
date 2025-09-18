package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	_ "github.com/macawi-ai/strigoi/modules/probe"
	"github.com/macawi-ai/strigoi/pkg/modules"
	"github.com/macawi-ai/strigoi/pkg/output"
)

var (
	centerTarget       string
	centerDuration     string
	centerOutput       string
	centerNoDisplay    bool
	centerFilter       string
	centerBufferSize   int
	centerPollInterval int
	centerShowActivity bool
	centerEnableStrace bool
)

var probeCenterCmd = &cobra.Command{
	Use:   "center [flags]",
	Short: "Monitor STDIO streams for security vulnerabilities",
	Long: `The Center probe monitors STDIO streams (stdin, stdout, stderr) of processes
for security vulnerabilities in real-time. It detects credentials, API keys,
tokens, and other sensitive data flowing through process streams.

Key capabilities:
- Real-time stream monitoring at user privilege level
- Multi-protocol analysis (JSON, SQL, plaintext)
- Credential and secret detection
- Live terminal display with vulnerability alerts
- Structured logging for forensic analysis`,
	Example: `  # Monitor a process by name
  strigoi probe center --target nginx

  # Monitor by PID with custom output
  strigoi probe center --target 12345 --output vulns.jsonl

  # Monitor with filter and duration limit
  strigoi probe center --target mysql --filter "password|token" --duration 1h

  # Monitor without terminal UI (log only)
  strigoi probe center --target api-server --no-display`,
	RunE: runProbeCenter,
}

func init() {
	// Target specification
	probeCenterCmd.Flags().StringVarP(&centerTarget, "target", "t", "", "Process name or PID to monitor (required)")
	_ = probeCenterCmd.MarkFlagRequired("target")

	// Monitoring options
	probeCenterCmd.Flags().StringVarP(&centerDuration, "duration", "d", "0", "Maximum monitoring duration (0 = unlimited)")
	probeCenterCmd.Flags().StringVarP(&centerOutput, "output", "o", "stream-monitor.jsonl", "Output log file (JSONL format)")
	probeCenterCmd.Flags().BoolVar(&centerNoDisplay, "no-display", false, "Disable terminal UI (log only mode)")
	probeCenterCmd.Flags().StringVarP(&centerFilter, "filter", "f", "", "Regex filter for stream data")

	// Performance tuning
	probeCenterCmd.Flags().IntVar(&centerBufferSize, "buffer-size", 64, "Buffer size per stream in KB")
	probeCenterCmd.Flags().IntVar(&centerPollInterval, "poll-interval", 10, "Stream polling interval in milliseconds")

	// Activity display
	probeCenterCmd.Flags().BoolVar(&centerShowActivity, "show-activity", false, "Show stream activity even when no vulnerabilities detected")

	// Strace fallback
	probeCenterCmd.Flags().BoolVar(&centerEnableStrace, "enable-strace", false, "Enable strace fallback for PTY capture (requires permissions, impacts performance)")

	// Note: For real-time visualization, use GoScope (github.com/macawi-ai/GoScope)
	// A clean Go-native alternative without Qt dependencies

	// Add to probe command
	probeCmd.AddCommand(probeCenterCmd)
}

func runProbeCenter(cmd *cobra.Command, _ []string) error {
	// Validate target
	if centerTarget == "" {
		return fmt.Errorf("target is required")
	}

	// Load modules
	_ = modules.LoadBuiltins(nil)

	// Get the module from registry
	centerModule, err := modules.Get("probe/center")
	if err != nil {
		return fmt.Errorf("failed to load center module: %w", err)
	}

	// Set module options
	if err := centerModule.SetOption("target", centerTarget); err != nil {
		return fmt.Errorf("failed to set target: %w", err)
	}
	if err := centerModule.SetOption("duration", centerDuration); err != nil {
		return fmt.Errorf("failed to set duration: %w", err)
	}
	if err := centerModule.SetOption("output", centerOutput); err != nil {
		return fmt.Errorf("failed to set output: %w", err)
	}
	if err := centerModule.SetOption("no-display", fmt.Sprintf("%t", centerNoDisplay)); err != nil {
		return fmt.Errorf("failed to set no-display: %w", err)
	}
	if err := centerModule.SetOption("filter", centerFilter); err != nil {
		return fmt.Errorf("failed to set filter: %w", err)
	}
	if err := centerModule.SetOption("buffer-size", fmt.Sprintf("%d", centerBufferSize)); err != nil {
		return fmt.Errorf("failed to set buffer-size: %w", err)
	}
	if err := centerModule.SetOption("poll-interval", fmt.Sprintf("%d", centerPollInterval)); err != nil {
		return fmt.Errorf("failed to set poll-interval: %w", err)
	}
	if err := centerModule.SetOption("show-activity", fmt.Sprintf("%t", centerShowActivity)); err != nil {
		return fmt.Errorf("failed to set show-activity: %w", err)
	}
	if err := centerModule.SetOption("enable-strace", fmt.Sprintf("%t", centerEnableStrace)); err != nil {
		return fmt.Errorf("failed to set enable-strace: %w", err)
	}

	// Check if module can run
	if !centerModule.Check() {
		return fmt.Errorf("center module check failed: ensure /proc is accessible")
	}

	// Show monitoring info
	fmt.Printf("\n► Starting STDIO stream monitoring...\n")
	fmt.Printf("  Target: %s\n", centerTarget)
	fmt.Printf("  Output: %s\n", centerOutput)
	if centerDuration != "0" {
		fmt.Printf("  Duration: %s\n", centerDuration)
	}
	if centerFilter != "" {
		fmt.Printf("  Filter: %s\n", centerFilter)
	}
	if centerNoDisplay {
		fmt.Printf("  Mode: Log-only (no terminal UI)\n")
	} else {
		fmt.Printf("  Mode: Interactive terminal UI\n")
	}
	if centerEnableStrace {
		fmt.Printf("  \033[33mStrace: Enabled (performance impact)\033[0m\n")
	}
	fmt.Println()

	// Handle interrupt for clean shutdown
	if !centerNoDisplay {
		fmt.Println("Press Ctrl+C to stop monitoring...")
		fmt.Println()
	}

	// Run the module
	fmt.Printf("Starting center probe for target: %s\n", centerTarget)
	result, err := centerModule.Run()
	if err != nil {
		return fmt.Errorf("center probe failed: %w", err)
	}

	// If terminal UI was disabled, display results
	if centerNoDisplay {
		// Create v2 output
		standardOutput := output.ConvertModuleResult(*result)

		// Check for vulnerabilities in results
		vulnCount := 0
		if stats, ok := result.Data["statistics"].(map[string]interface{}); ok {
			if count, ok := stats["total_vulns"].(int64); ok {
				vulnCount = int(count)
			}
		}

		// Display summary based on findings
		if vulnCount > 0 {
			fmt.Printf("\n✗ Stream monitoring completed - %d vulnerabilities detected!\n", vulnCount)
		} else {
			fmt.Printf("\n✓ Stream monitoring completed - no vulnerabilities detected\n")
		}

		// Get output format from parent command flags
		outputFormat, _ := cmd.Parent().Flags().GetString("output")

		// Format and display results
		verbosity := "normal"
		noColor := false
		var severityFilter []string

		formatted, err := output.FormatOutput(
			standardOutput,
			outputFormat,
			verbosity,
			noColor,
			severityFilter,
		)
		if err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}

		fmt.Print(formatted)

		// Show log file location
		fmt.Printf("\nFull results saved to: %s\n", centerOutput)
	}

	// Exit with error if vulnerabilities found
	if result.Status == "completed" {
		if stats, ok := result.Data["statistics"].(map[string]interface{}); ok {
			if vulns, ok := stats["total_vulns"].(int64); ok && vulns > 0 {
				os.Exit(1)
			}
		}
	}

	return nil
}
