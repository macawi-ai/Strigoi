package west

import (
    "fmt"
    "strings"
    "time"
    
    "github.com/macawi-ai/strigoi/internal/stream"
)

// SimulateStreamEvents generates example stream events for demonstration
func SimulateStreamEvents(output stream.OutputWriter, pids []int, duration time.Duration) error {
    start := time.Now()
    eventCount := 0
    
    // Simulate some realistic MCP communication patterns
    patterns := []struct {
        direction stream.Direction
        data      string
        size      int
    }{
        {
            direction: stream.DirectionInbound,
            data:      `{"jsonrpc":"2.0","method":"tools/list","id":"1"}`,
            size:      48,
        },
        {
            direction: stream.DirectionOutbound,
            data:      `{"jsonrpc":"2.0","result":{"tools":[{"name":"read_file","description":"Read file contents"}]},"id":"1"}`,
            size:      104,
        },
        {
            direction: stream.DirectionInbound,
            data:      `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read_file","arguments":{"path":"/etc/passwd"}},"id":"2"}`,
            size:      115,
        },
        {
            direction: stream.DirectionOutbound,
            data:      `{"jsonrpc":"2.0","result":{"content":"root:x:0:0:root:/root:/bin/bash\n..."},"id":"2"}`,
            size:      88,
        },
        {
            direction: stream.DirectionInbound,
            data:      `{"jsonrpc":"2.0","method":"execute","params":{"command":"cat /home/user/.ssh/id_rsa"},"id":"3"}`,
            size:      96,
        },
    }
    
    // Generate events over the duration
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if time.Since(start) >= duration {
                return nil
            }
            
            // Pick a pattern
            pattern := patterns[eventCount%len(patterns)]
            
            // Create event
            event := &stream.StreamEvent{
                Timestamp:   time.Now(),
                Type:        stream.StreamEventRead,
                Direction:   pattern.direction,
                PID:         pids[eventCount%len(pids)],
                ProcessName: fmt.Sprintf("mcp-server-%d", pids[eventCount%len(pids)]),
                FD:          3, // stdin/stdout
                Data:        []byte(pattern.data),
                Size:        pattern.size,
                Summary:     fmt.Sprintf("JSON-RPC %s", pattern.direction),
            }
            
            // Write event
            if err := output.WriteEvent(event); err != nil {
                return fmt.Errorf("failed to write event: %w", err)
            }
            
            // Check for security patterns
            if eventCount == 2 || eventCount == 4 {
                // Simulate security alerts
                alert := &stream.SecurityAlert{
                    Timestamp:   event.Timestamp,
                    EventID:     fmt.Sprintf("evt_%d_%d", event.PID, event.Timestamp.UnixNano()),
                    Severity:    "high",
                    Category:    "data_access",
                    Pattern:     "SENSITIVE_FILE_ACCESS",
                    Title:       "Sensitive file access detected",
                    Description: "MCP server attempting to access sensitive system files",
                    Details:     fmt.Sprintf("File path: %s", pattern.data),
                    PID:         event.PID,
                    ProcessName: event.ProcessName,
                    Evidence:    string(event.Data),
                    Blocked:     false,
                    Mitigation:  "Review MCP server permissions and implement path restrictions",
                }
                
                if err := output.WriteAlert(alert); err != nil {
                    return fmt.Errorf("failed to write alert: %w", err)
                }
            }
            
            eventCount++
        }
    }
}

// FormatEventForDisplay formats an event for console display
func FormatEventForDisplay(event *stream.StreamEvent) string {
    timestamp := event.Timestamp.Format("15:04:05.000")
    
    direction := "→"
    dirColor := "\033[32m" // green for inbound
    if event.Direction == stream.DirectionOutbound {
        direction = "←"
        dirColor = "\033[31m" // red for outbound
    }
    
    // Truncate data for display
    data := string(event.Data)
    if len(data) > 80 {
        data = data[:77] + "..."
    }
    
    return fmt.Sprintf("%s %s%s\033[0m [PID:%d] %s",
        timestamp,
        dirColor,
        direction,
        event.PID,
        data,
    )
}

// FormatAlertForDisplay formats an alert for console display
func FormatAlertForDisplay(alert *stream.SecurityAlert) string {
    timestamp := alert.Timestamp.Format("15:04:05.000")
    
    severityColor := "\033[33m" // yellow
    if alert.Severity == "critical" || alert.Severity == "high" {
        severityColor = "\033[31m" // red
    }
    
    return fmt.Sprintf("%s %s⚠ ALERT [%s]\033[0m %s - %s",
        timestamp,
        severityColor,
        strings.ToUpper(alert.Severity),
        alert.Title,
        alert.Details,
    )
}