package core

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/internal/actors"
	"github.com/macawi-ai/strigoi/internal/actors/west"
	"github.com/macawi-ai/strigoi/internal/stream"
)

// Stream command handlers

func (c *ConsoleV2) handleStreamTap(console interface{}, cmd *ParsedCommand) error {
	c.Info("🔍 Starting STDIO stream monitoring...")
	
	// Parse flags
	autoDiscover := cmd.Flags["auto-discover"] == "true" || cmd.Flags["a"] == "true"
	pidStr := cmd.Flags["pid"]
	if pidStr == "" {
		pidStr = cmd.Flags["p"]
	}
	
	durationStr := cmd.Flags["duration"]
	if durationStr == "" {
		durationStr = cmd.Flags["d"]
	}
	if durationStr == "" {
		durationStr = "30s" // Default
	}
	
	outputDest := cmd.Flags["output"]
	if outputDest == "" {
		outputDest = cmd.Flags["o"]
	}
	if outputDest == "" {
		outputDest = "stdout"
	}
	
	// Parse duration
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}
	
	// Create output writer
	outputWriter, err := stream.ParseOutputDestination(outputDest)
	if err != nil {
		return fmt.Errorf("invalid output destination: %w", err)
	}
	defer outputWriter.Close()
	
	// Create the stream monitor actor
	monitor := west.NewStdioStreamMonitor()
	
	// Set up monitoring based on mode
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	if autoDiscover {
		// Auto-discover mode
		c.Info("📡 Probing for processes...")
		
		probeResult, err := monitor.Probe(ctx, actors.Target{
			Type: "system",
			Metadata: map[string]interface{}{
				"auto_discover": true,
			},
		})
		if err != nil {
			return fmt.Errorf("probe failed: %w", err)
		}
		
		// Extract processes from probe result
		processes := west.ExtractProcesses(probeResult)
		
		if len(processes) > 0 {
			c.Success("Found %d process(es):", len(processes))
			for _, proc := range processes {
				cmdPreview := proc.Command
				if len(cmdPreview) > 80 {
					cmdPreview = cmdPreview[:77] + "..."
				}
				c.Printf("  • PID %d: %s [%s]\n", proc.PID, proc.Name, proc.Category)
				if cmdPreview != "" && cmdPreview != proc.Name {
					c.Printf("    %s\n", cmdPreview)
				}
			}
			
			// Start monitoring all discovered processes
			c.Info("🎯 Starting real-time monitoring for %s...", durationStr)
			c.Info("Press Ctrl+C to stop early")
			fmt.Fprintln(c.writer)
			
			// Monitor each process
			for _, proc := range processes {
				go monitor.MonitorProcess(ctx, proc, outputWriter)
			}
			
			// Wait for duration or cancellation
			<-ctx.Done()
			c.Info("⏰ Monitoring duration completed")
		} else {
			c.Warn("No Claude or MCP processes found")
		}
	} else if pidStr != "" {
		// Specific PID mode
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			return fmt.Errorf("invalid PID: %w", err)
		}
		
		c.Info("Monitoring PID %d for %s...", pid, durationStr)
		c.Info("Press Ctrl+C to stop early")
		
		// Create process info
		proc := west.ProcessInfo{
			PID:      pid,
			Name:     fmt.Sprintf("pid_%d", pid),
			Category: "Manual",
		}
		
		// Start monitoring
		err = monitor.MonitorProcess(ctx, proc, outputWriter)
		if err != nil {
			return fmt.Errorf("monitoring failed: %w", err)
		}
		
		<-ctx.Done()
		c.Info("⏰ Monitoring completed")
	} else {
		return fmt.Errorf("either --auto-discover or --pid must be specified")
	}
	
	return nil
}

func (c *ConsoleV2) handleStreamRecord(console interface{}, cmd *ParsedCommand) error {
	name := cmd.Flags["name"]
	if name == "" {
		name = cmd.Flags["n"]
	}
	
	c.Info("Recording streams to: %s", name)
	c.Warn("stream/record not yet implemented")
	return nil
}

func (c *ConsoleV2) handleStreamReplay(console interface{}, cmd *ParsedCommand) error {
	c.Warn("stream/replay not yet implemented")
	return nil
}

func (c *ConsoleV2) handleStreamAnalyze(console interface{}, cmd *ParsedCommand) error {
	c.Warn("stream/analyze not yet implemented")
	return nil
}

func (c *ConsoleV2) handleStreamPatterns(console interface{}, cmd *ParsedCommand) error {
	c.Info("Current security patterns:")
	
	for _, pattern := range stream.DefaultSecurityPatterns() {
		c.Printf("  • %-20s [%s] %s\n", pattern.Name, pattern.Severity, pattern.Description)
	}
	
	c.Info("\nPattern categories: %s", strings.Join([]string{
		"credentials", "injection", "traversal", "network",
	}, ", "))
	
	return nil
}

func (c *ConsoleV2) handleStreamStatus(console interface{}, cmd *ParsedCommand) error {
	c.Info("Stream monitoring status:")
	c.Printf("  • Active monitors: 0\n")
	c.Printf("  • Events captured: 0\n")
	c.Printf("  • Alerts raised: 0\n")
	
	return nil
}

// Integration command handlers

func (c *ConsoleV2) handleIntegrationsList(console interface{}, cmd *ParsedCommand) error {
	c.Info("Available integrations:")
	c.Printf("  • prometheus    - Export metrics to Prometheus\n")
	c.Printf("  • syslog        - Send events to syslog\n")
	c.Printf("  • file          - Log events to file\n")
	c.Printf("  • elasticsearch - Send to Elasticsearch (planned)\n")
	c.Printf("  • splunk        - Send to Splunk (planned)\n")
	
	return nil
}

func (c *ConsoleV2) handlePrometheusEnable(console interface{}, cmd *ParsedCommand) error {
	// TEMPORARY: Comment out messages to debug navigation issue
	// This handler is being called on every command for some reason
	
	// DEBUG: Log when this handler is called
	c.Error("DEBUG: handlePrometheusEnable called! Command path: %v, Raw: %s", cmd.Path, cmd.RawInput)
	
	portStr := cmd.Flags["port"]
	if portStr == "" {
		portStr = cmd.Flags["p"]
	}
	if portStr == "" {
		portStr = "9090"
	}
	
	_, err := strconv.Atoi(portStr) // port variable temporarily unused
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}
	
	// COMMENTED OUT TO DEBUG NAVIGATION
	// c.Success("Prometheus metrics enabled on port %d", port)
	// c.Printf("[*] Metrics endpoint: http://localhost:%d/metrics\n", port)
	
	return nil
}

func (c *ConsoleV2) handlePrometheusDisable(console interface{}, cmd *ParsedCommand) error {
	c.Success("Prometheus metrics disabled")
	return nil
}

// Probe command handlers

func (c *ConsoleV2) handleProbeDirection(console interface{}, cmd *ParsedCommand) error {
	// Extract direction from command path
	direction := cmd.Path[len(cmd.Path)-1]
	
	c.Info("Probing %s direction...", direction)
	c.Warn("probe/%s not yet implemented", direction)
	
	return nil
}

func (c *ConsoleV2) handleProbeAll(console interface{}, cmd *ParsedCommand) error {
	c.Info("Probing all directions...")
	directions := []string{"north", "south", "east", "west", "center"}
	
	for _, dir := range directions {
		c.Printf("  • %s: pending\n", dir)
	}
	
	c.Warn("probe/all not yet implemented")
	return nil
}

// Sense command handlers

func (c *ConsoleV2) handleSenseLayer(console interface{}, cmd *ParsedCommand) error {
	// Extract layer from command path
	layer := cmd.Path[len(cmd.Path)-1]
	
	c.Info("Analyzing %s layer...", layer)
	c.Warn("sense/%s not yet implemented", layer)
	
	return nil
}