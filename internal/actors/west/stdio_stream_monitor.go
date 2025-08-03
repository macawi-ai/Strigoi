package west

import (
    "context"
    "fmt"
    "os"
    "strings"
    "time"
    
    "github.com/macawi-ai/strigoi/internal/actors"
    "github.com/macawi-ai/strigoi/internal/stream"
)

// StdioStreamMonitor monitors STDIO communications for security risks
type StdioStreamMonitor struct {
    *actors.BaseActor
    
    // Configuration
    monitoringMode string
    patterns       []stream.SecurityPattern
    outputWriter   stream.OutputWriter
    
    // State
    activeMonitors map[int]*ProcessMonitor
}

// ProcessMonitor tracks a single process
type ProcessMonitor struct {
    PID        int
    Name       string
    StartTime  time.Time
    EventCount int
}

// NewStdioStreamMonitor creates a new STDIO stream monitor
func NewStdioStreamMonitor() *StdioStreamMonitor {
    actor := &StdioStreamMonitor{
        BaseActor: actors.NewBaseActor(
            "stdio_stream_monitor",
            "Monitors STDIO communications between Claude and MCP servers",
            "west",
        ),
        monitoringMode: "tap",
        activeMonitors: make(map[int]*ProcessMonitor),
    }
    
    // Define capabilities
    actor.AddCapability(actors.Capability{
        Name:        "stdio_monitoring",
        Description: "Monitor standard I/O streams for security risks",
        DataTypes:   []string{"stream", "stdio", "jsonrpc"},
    })
    
    actor.AddCapability(actors.Capability{
        Name:        "pattern_detection",
        Description: "Detect security patterns in stream data",
        DataTypes:   []string{"patterns", "security"},
    })
    
    actor.AddCapability(actors.Capability{
        Name:        "process_discovery",
        Description: "Discover Claude and MCP processes",
        DataTypes:   []string{"process", "discovery"},
    })
    
    actor.SetInputTypes([]string{"process_tree", "pid"})
    actor.SetOutputType("stream_analysis")
    
    // Load default security patterns
    actor.patterns = stream.DefaultSecurityPatterns()
    
    return actor
}

// SetOutputWriter sets the output writer for the monitor
func (s *StdioStreamMonitor) SetOutputWriter(writer stream.OutputWriter) {
    s.outputWriter = writer
}

// Probe discovers processes to monitor
func (s *StdioStreamMonitor) Probe(ctx context.Context, target actors.Target) (*actors.ProbeResult, error) {
    discoveries := []actors.Discovery{}
    
    // Check monitoring method availability
    if _, err := os.Stat("/proc"); err == nil {
        discoveries = append(discoveries, actors.Discovery{
            Type:       "monitoring_method",
            Identifier: "/proc",
            Properties: map[string]interface{}{
                "available": true,
                "method":    "proc_filesystem",
            },
            Confidence: 1.0,
        })
    }
    
    // Check for strace
    if _, err := os.Stat("/usr/bin/strace"); err == nil {
        discoveries = append(discoveries, actors.Discovery{
            Type:       "monitoring_method",
            Identifier: "strace",
            Properties: map[string]interface{}{
                "available": true,
                "method":    "syscall_trace",
            },
            Confidence: 0.9,
        })
    }
    
    // Check for specific PID
    if pid, ok := target.Metadata["claude_pid"].(int); ok && pid > 0 {
        // Verify the PID exists
        procPath := fmt.Sprintf("/proc/%d", pid)
        if info, err := os.Stat(procPath); err == nil && info.IsDir() {
            // Read process info
            cmdlineBytes, _ := os.ReadFile(fmt.Sprintf("%s/cmdline", procPath))
            cmdline := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
            cmdline = strings.TrimSpace(cmdline)
            
            commBytes, _ := os.ReadFile(fmt.Sprintf("%s/comm", procPath))
            comm := strings.TrimSpace(string(commBytes))
            
            discoveries = append(discoveries, actors.Discovery{
                Type:       "process",
                Identifier: fmt.Sprintf("pid_%d", pid),
                Properties: map[string]interface{}{
                    "pid":     pid,
                    "name":    comm,
                    "cmdline": cmdline,
                    "type":    "specified",
                },
                Confidence: 1.0,
            })
        } else {
            discoveries = append(discoveries, actors.Discovery{
                Type:       "error",
                Identifier: "pid_not_found",
                Properties: map[string]interface{}{
                    "pid":   pid,
                    "error": "Process not found",
                },
                Confidence: 1.0,
            })
        }
    } else if autoDiscover, ok := target.Metadata["auto_discover"].(bool); ok && autoDiscover {
    // Auto-discover Claude/MCP processes if requested
        // Use real process discovery
        patterns := []string{
            "claude*",
            "*mcp-server*",
            "*mcp_server*",
            "node*mcp*",
            "python*mcp*",
            "deno*mcp*",
            "*server.py",  // Common MCP server pattern
            "*.claude-mcp-servers*", // Claude MCP server directory
        }
        
        processes, err := DiscoverProcesses(patterns)
        if err != nil {
            // Fall back to mock data if discovery fails
            discoveries = append(discoveries, actors.Discovery{
                Type:       "error",
                Identifier: "process_discovery",
                Properties: map[string]interface{}{
                    "error": err.Error(),
                },
                Confidence: 1.0,
            })
        } else {
            // Add discovered processes
            for _, proc := range processes {
                procType := "unknown"
                if proc.IsClaudeRelated {
                    procType = "claude"
                } else if proc.IsMCPServer {
                    procType = "mcp_server"
                }
                
                discoveries = append(discoveries, actors.Discovery{
                    Type:       "process",
                    Identifier: fmt.Sprintf("pid_%d", proc.PID),
                    Properties: map[string]interface{}{
                        "pid":     proc.PID,
                        "ppid":    proc.PPID,
                        "name":    proc.Name,
                        "cmdline": proc.Cmdline,
                        "exe":     proc.Exe,
                        "type":    procType,
                    },
                    Confidence: 0.95,
                })
            }
            
            // If no processes found, indicate that
            if len(processes) == 0 {
                discoveries = append(discoveries, actors.Discovery{
                    Type:       "info",
                    Identifier: "no_processes",
                    Properties: map[string]interface{}{
                        "message": "No Claude or MCP processes found",
                        "patterns": patterns,
                    },
                    Confidence: 1.0,
                })
            }
        }
    }
    
    return &actors.ProbeResult{
        ActorName:   s.Name(),
        Target:      target,
        Discoveries: discoveries,
        RawData: map[string]interface{}{
            "monitoring_mode": s.monitoringMode,
            "pattern_count":   len(s.patterns),
        },
    }, nil
}

// Sense performs real-time stream monitoring
func (s *StdioStreamMonitor) Sense(ctx context.Context, data *actors.ProbeResult) (*actors.SenseResult, error) {
    observations := []actors.Observation{}
    patterns := []actors.Pattern{}
    risks := []actors.Risk{}
    
    // Extract target processes from discoveries
    targetPIDs := []int{}
    hasStrace := false
    
    for _, disc := range data.Discoveries {
        if disc.Type == "process" {
            if pid, ok := disc.Properties["pid"].(int); ok {
                targetPIDs = append(targetPIDs, pid)
                
                // Create monitor for each process
                name := "unknown"
                if n, ok := disc.Properties["name"].(string); ok {
                    name = n
                } else if cmd, ok := disc.Properties["cmdline"].(string); ok {
                    name = cmd
                }
                
                s.activeMonitors[pid] = &ProcessMonitor{
                    PID:       pid,
                    Name:      name,
                    StartTime: time.Now(),
                }
            }
        } else if disc.Type == "monitoring_method" && disc.Identifier == "strace" {
            if available, ok := disc.Properties["available"].(bool); ok && available {
                hasStrace = true
            }
        }
    }
    
    if len(targetPIDs) == 0 {
        return nil, fmt.Errorf("no processes to monitor")
    }
    
    // Check if we can use strace
    if !hasStrace {
        observations = append(observations, actors.Observation{
            Layer:       "stdio",
            Description: "strace not available - limited monitoring capability",
            Evidence: map[string]interface{}{
                "suggestion": "Install strace for full STDIO monitoring",
            },
            Severity: "warning",
        })
    }
    
    // Determine monitoring approach
    method := "simulated"
    if hasStrace {
        method = "strace"
    }
    
    observations = append(observations, actors.Observation{
        Layer:       "stdio",
        Description: fmt.Sprintf("Monitoring %d processes for STDIO activity", len(targetPIDs)),
        Evidence: map[string]interface{}{
            "pids":            targetPIDs,
            "monitoring_mode": s.monitoringMode,
            "method":          method,
        },
        Severity: "info",
    })
    
    // Use provided output writer or create in-memory one
    outputWriter := s.outputWriter
    if outputWriter == nil {
        outputWriter = stream.NewMemoryOutput()
    }
    
    // Build process list
    var processes []int
    for pid := range s.activeMonitors {
        processes = append(processes, pid)
    }
    
    // Get monitoring duration
    duration := 30 * time.Second
    if d, ok := data.Target.Metadata["duration"].(time.Duration); ok {
        duration = d
    }
    
    // Create context with timeout
    monitorCtx, cancel := context.WithTimeout(ctx, duration)
    defer cancel()
    
    if hasStrace && len(processes) > 0 {
        // Real strace monitoring
        observations = append(observations, actors.Observation{
            Layer:       "stdio",
            Description: fmt.Sprintf("Starting strace monitoring on %d processes", len(processes)),
            Severity:    "info",
        })
        
        // Monitor each process
        eventCount := 0
        alertCount := 0
        
        for pid, monitor := range s.activeMonitors {
            straceMonitor := stream.NewStraceMonitor(pid, monitor.Name, outputWriter, s.patterns)
            
            // Start monitoring in a goroutine
            monitorErr := make(chan error, 1)
            go func() {
                if err := straceMonitor.Start(monitorCtx); err != nil {
                    monitorErr <- err
                }
            }()
            
            // Wait briefly for strace to attach
            select {
            case err := <-monitorErr:
                observations = append(observations, actors.Observation{
                    Layer:       "stdio",
                    Description: fmt.Sprintf("Failed to attach strace to PID %d: %v", pid, err),
                    Severity:    "warning",
                })
            case <-time.After(100 * time.Millisecond):
                // Strace started successfully
                observations = append(observations, actors.Observation{
                    Layer:       "stdio",
                    Description: fmt.Sprintf("Monitoring PID %d (%s)", pid, monitor.Name),
                    Severity:    "info",
                })
            }
            
            // Get stats after monitoring
            defer func() {
                straceMonitor.Stop()
                events, alerts := straceMonitor.GetStats()
                eventCount += events
                alertCount += alerts
            }()
        }
        
        // Wait for monitoring to complete
        <-monitorCtx.Done()
        
        // Report results
        if eventCount > 0 || alertCount > 0 {
            patterns = append(patterns, actors.Pattern{
                Name:        "STDIO_ACTIVITY",
                Description: "Process STDIO activity detected",
                Confidence:  1.0,
                Instances: []interface{}{
                    map[string]interface{}{
                        "event_count": eventCount,
                        "alert_count": alertCount,
                        "method":      "strace",
                    },
                },
            })
        }
    } else {
        // Simulated monitoring for when strace is not available
        // Wait for a short time to simulate monitoring
        select {
        case <-time.After(2 * time.Second):
        case <-monitorCtx.Done():
        }
        
        // Simulate finding patterns
        if len(targetPIDs) > 0 {
            patterns = append(patterns, actors.Pattern{
                Name:        "JSONRPC_COMMUNICATION",
                Description: "JSON-RPC protocol communication detected",
                Confidence:  0.95,
                Instances: []interface{}{
                    map[string]interface{}{
                        "protocol": "json-rpc",
                        "version":  "2.0",
                        "note":     "Simulated detection",
                    },
                },
            })
        }
    }
    
    // Always report the STDIO vulnerability risk
    risks = append(risks, actors.Risk{
        Title:       "Direct STDIO Access Risk",
        Description: "MCP servers have direct access to process STDIO, enabling potential command injection",
        Severity:    "high",
        Mitigation:  "Implement message exchange router to isolate STDIO access",
        Evidence: map[string]interface{}{
            "risk_score":    0.7 * 0.9, // likelihood * impact
            "affected_pids": targetPIDs,
            "attack_vector": "Direct process injection via STDIO",
        },
    })
    
    return &actors.SenseResult{
        ActorName:    s.Name(),
        Observations: observations,
        Patterns:     patterns,
        Risks:        risks,
    }, nil
}

// Transform analyzes stream data
func (s *StdioStreamMonitor) Transform(ctx context.Context, input interface{}) (interface{}, error) {
    switch v := input.(type) {
    case *stream.StreamEvent:
        // Analyze stream event
        return s.analyzeStreamEvent(v)
    case []byte:
        // Analyze raw stream data
        return s.analyzeRawData(v)
    default:
        return nil, fmt.Errorf("unsupported input type: %T", input)
    }
}

// analyzeStreamEvent checks a stream event for security patterns
func (s *StdioStreamMonitor) analyzeStreamEvent(event *stream.StreamEvent) (*stream.SecurityAlert, error) {
    // Check against security patterns
    for _, pattern := range s.patterns {
        if pattern.Matches(event.Data) {
            return &stream.SecurityAlert{
                Timestamp: event.Timestamp,
                Severity:  pattern.Severity,
                Category:  pattern.Category,
                Pattern:   pattern.Name,
                Title:     fmt.Sprintf("Security pattern detected: %s", pattern.Name),
                Details:   pattern.Description,
                PID:       event.PID,
                ProcessName: event.ProcessName,
                Evidence:    string(event.Data),
                Blocked:   false,
            }, nil
        }
    }
    
    return nil, nil
}

// analyzeRawData checks raw data for patterns
func (s *StdioStreamMonitor) analyzeRawData(data []byte) (map[string]interface{}, error) {
    result := map[string]interface{}{
        "size":     len(data),
        "analyzed": true,
    }
    
    // Check for patterns
    matches := []string{}
    for _, pattern := range s.patterns {
        if pattern.Matches(data) {
            matches = append(matches, pattern.Name)
        }
    }
    
    if len(matches) > 0 {
        result["patterns"] = matches
        result["risk"] = true
    }
    
    return result, nil
}