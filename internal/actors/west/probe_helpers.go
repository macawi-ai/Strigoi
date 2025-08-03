package west

import (
	"context"
	"fmt"
	
	"github.com/macawi-ai/strigoi/internal/actors"
	"github.com/macawi-ai/strigoi/internal/stream"
)

// ExtractProcesses converts ProbeResult discoveries to ProcessInfo slice
func ExtractProcesses(result *actors.ProbeResult) []ProcessInfo {
	var processes []ProcessInfo
	
	for _, disc := range result.Discoveries {
		if disc.Type == "process" {
			proc := ProcessInfo{}
			
			if pid, ok := disc.Properties["pid"].(int); ok {
				proc.PID = pid
			}
			
			if ppid, ok := disc.Properties["ppid"].(int); ok {
				proc.PPID = ppid
			}
			
			if name, ok := disc.Properties["name"].(string); ok {
				proc.Name = name
			}
			
			if cmdline, ok := disc.Properties["cmdline"].(string); ok {
				proc.Command = cmdline
			}
			
			// Determine category
			if procType, ok := disc.Properties["type"].(string); ok {
				switch procType {
				case "claude":
					proc.Category = "Claude"
				case "mcp_server":
					proc.Category = "MCP"
				default:
					proc.Category = "Unknown"
				}
			}
			
			processes = append(processes, proc)
		}
	}
	
	return processes
}

// MonitorProcess monitors a single process with strace
func (s *StdioStreamMonitor) MonitorProcess(ctx context.Context, proc ProcessInfo, output stream.OutputWriter) error {
	// Create strace monitor
	monitor := stream.NewStraceMonitor(proc.PID, proc.Name, output, stream.DefaultSecurityPatterns())
	
	// Start monitoring
	if err := monitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start monitoring PID %d: %w", proc.PID, err)
	}
	
	// Wait for context cancellation
	<-ctx.Done()
	
	// Stop monitoring
	return monitor.Stop()
}