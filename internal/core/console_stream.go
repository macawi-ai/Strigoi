package core

import (
    "context"
    "fmt"
    "strconv"
    "strings"
    "time"
    
    "github.com/fatih/color"
    "github.com/macawi-ai/strigoi/internal/actors"
    "github.com/macawi-ai/strigoi/internal/actors/west"
    "github.com/macawi-ai/strigoi/internal/stream"
)

// processStreamCommand handles stream/ subcommands
func (c *Console) processStreamCommand(subcommand string, args []string) error {
    if subcommand == "" {
        return c.showStreamHelp()
    }
    
    switch subcommand {
    case "tap":
        return c.streamTap(args)
    case "record":
        return c.streamRecord(args)
    case "replay":
        return c.streamReplay(args)
    case "analyze":
        return c.streamAnalyze(args)
    case "patterns":
        return c.streamPatterns(args)
    case "status":
        return c.streamStatus(args)
    default:
        c.Error("Unknown stream subcommand: %s", subcommand)
        return c.showStreamHelp()
    }
}

// showStreamHelp displays stream command help
func (c *Console) showStreamHelp() error {
    c.Info("Stream Commands - STDIO Monitoring & Analysis")
    fmt.Fprintln(c.writer)
    fmt.Fprintln(c.writer, "  stream/tap        - Start live stream monitoring")
    fmt.Fprintln(c.writer, "  stream/record     - Record streams to file")
    fmt.Fprintln(c.writer, "  stream/replay     - Replay recorded session")
    fmt.Fprintln(c.writer, "  stream/analyze    - Analyze captured streams")
    fmt.Fprintln(c.writer, "  stream/patterns   - Manage security patterns")
    fmt.Fprintln(c.writer, "  stream/status     - Show active monitors")
    fmt.Fprintln(c.writer)
    c.infoColor.Fprintln(c.writer, "Tap Options:")
    fmt.Fprintln(c.writer, "  --pid <PID>       - Monitor specific process")
    fmt.Fprintln(c.writer, "  --auto-discover   - Auto-find Claude & MCP processes")
    fmt.Fprintln(c.writer, "  --follow-children - Include child processes")
    fmt.Fprintln(c.writer, "  --duration <time> - Monitoring duration (default: 30s)")
    fmt.Fprintln(c.writer, "  --output <dest>   - Output destination (default: stdout)")
    fmt.Fprintln(c.writer, "  --format <fmt>    - Output format: json, jsonl, cef, raw")
    fmt.Fprintln(c.writer)
    c.infoColor.Fprintln(c.writer, "Output Destinations:")
    fmt.Fprintln(c.writer, "  stdout              - Display in console (default)")
    fmt.Fprintln(c.writer, "  file:/path/to/file  - Write to file")
    fmt.Fprintln(c.writer, "  tcp:host:port       - Stream to TCP endpoint")
    fmt.Fprintln(c.writer, "  unix:/path/to/sock  - Stream to Unix socket")
    fmt.Fprintln(c.writer, "  pipe:name           - Create named pipe")
    fmt.Fprintln(c.writer, "  integration:name    - Send to integration")
    fmt.Fprintln(c.writer)
    c.infoColor.Fprintln(c.writer, "Examples:")
    fmt.Fprintln(c.writer, "  stream/tap --auto-discover")
    fmt.Fprintln(c.writer, "  stream/tap --pid 1234 --output file:/tmp/capture.jsonl")
    fmt.Fprintln(c.writer, "  stream/tap --auto-discover -o tcp:localhost:9999")
    fmt.Fprintln(c.writer, "  stream/tap --pid 1234 -o integration:prometheus")
    fmt.Fprintln(c.writer, "  stream/record --duration 5m -o file:/var/log/strigoi.jsonl")
    fmt.Fprintln(c.writer)
    return nil
}

// streamTap starts live stream monitoring
func (c *Console) streamTap(args []string) error {
    c.Info("🔍 Starting STDIO stream monitoring...")
    
    // Parse arguments
    var (
        pid            int
        autoDiscover   bool
        followChildren bool
        duration       = 30 * time.Second
        outputDest     = "stdout"
        format         = "jsonl"
    )
    
    for i := 0; i < len(args); i++ {
        switch args[i] {
        case "--pid":
            if i+1 < len(args) {
                var err error
                pid, err = strconv.Atoi(args[i+1])
                if err != nil {
                    return fmt.Errorf("invalid PID: %s", args[i+1])
                }
                i++
            }
        case "--auto-discover":
            autoDiscover = true
        case "--follow-children":
            followChildren = true
        case "--duration":
            if i+1 < len(args) {
                var err error
                duration, err = time.ParseDuration(args[i+1])
                if err != nil {
                    return fmt.Errorf("invalid duration: %s", args[i+1])
                }
                i++
            }
        case "--output", "-o":
            if i+1 < len(args) {
                outputDest = args[i+1]
                i++
            }
        case "--format", "-f":
            if i+1 < len(args) {
                format = args[i+1]
                i++
            }
        }
    }
    
    // Validate options
    if pid == 0 && !autoDiscover {
        c.Error("Must specify either --pid or --auto-discover")
        return fmt.Errorf("no target specified")
    }
    
    // Parse output destination
    outputWriter, err := stream.ParseOutputDestination(outputDest)
    if err != nil {
        c.Error("Invalid output destination: %v", err)
        return err
    }
    defer outputWriter.Close()
    
    // Show output configuration if not stdout
    if outputDest != "stdout" && outputDest != "-" {
        c.Info("📤 Output: %s (format: %s)", outputDest, format)
    }
    
    // Create stream monitor actor
    monitor := west.NewStdioStreamMonitor()
    monitor.SetOutputWriter(outputWriter)
    
    // Configure target
    target := actors.Target{
        Type: "process_tree",
        Metadata: map[string]interface{}{
            "auto_discover":    autoDiscover,
            "follow_children":  followChildren,
            "duration":         duration,
        },
    }
    
    if pid > 0 {
        target.Metadata["claude_pid"] = pid
    }
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    defer cancel()
    
    // Probe phase
    c.Info("📡 Probing for processes...")
    probeResult, err := monitor.Probe(ctx, target)
    if err != nil {
        return fmt.Errorf("probe failed: %w", err)
    }
    
    // Display discovered processes
    processCount := 0
    capabilities := []actors.Discovery{}
    processes := []actors.Discovery{}
    
    // Separate processes from capabilities
    for _, disc := range probeResult.Discoveries {
        switch disc.Type {
        case "process":
            processes = append(processes, disc)
            processCount++
        case "monitoring_method", "info", "error":
            capabilities = append(capabilities, disc)
        }
    }
    
    // Show capabilities first if any
    if len(capabilities) > 0 {
        c.Info("System capabilities:")
        for _, cap := range capabilities {
            if cap.Type == "monitoring_method" {
                if available, ok := cap.Properties["available"].(bool); ok && available {
                    c.Success("  ✓ %s available", cap.Identifier)
                } else {
                    c.Warn("  ✗ %s not available", cap.Identifier)
                }
            } else if cap.Type == "info" {
                if msg, ok := cap.Properties["message"].(string); ok {
                    c.Info("  • %s", msg)
                }
            } else if cap.Type == "error" {
                if err, ok := cap.Properties["error"].(string); ok {
                    c.Error("  ✗ %s", err)
                }
            }
        }
        fmt.Fprintln(c.writer)
    }
    
    // Show discovered processes
    if processCount > 0 {
        c.Success("Found %d process(es):", processCount)
        for _, disc := range processes {
            pid := disc.Properties["pid"]
            name := disc.Properties["name"]
            cmdline := disc.Properties["cmdline"]
            procType := disc.Properties["type"]
            
            // Format process info
            pidStr := fmt.Sprintf("%v", pid)
            nameStr := ""
            if name != nil {
                nameStr = fmt.Sprintf("%v", name)
            }
            cmdlineStr := ""
            if cmdline != nil {
                cmdlineStr = fmt.Sprintf("%v", cmdline)
                // Truncate long command lines
                if len(cmdlineStr) > 80 {
                    cmdlineStr = cmdlineStr[:77] + "..."
                }
            }
            
            // Show process with type indicator
            typeIndicator := ""
            switch procType {
            case "claude":
                typeIndicator = " [Claude]"
            case "mcp_server":
                typeIndicator = " [MCP Server]"
            }
            
            if nameStr != "" && cmdlineStr != "" && nameStr != cmdlineStr {
                c.Success("  • PID %s: %s%s", pidStr, nameStr, typeIndicator)
                c.Printf("    %s\n", cmdlineStr)
            } else if cmdlineStr != "" {
                c.Success("  • PID %s: %s%s", pidStr, cmdlineStr, typeIndicator)
            } else if nameStr != "" {
                c.Success("  • PID %s: %s%s", pidStr, nameStr, typeIndicator)
            }
        }
    } else {
        c.Warn("No matching processes found")
    }
    fmt.Fprintln(c.writer)
    
    // Sense phase (real-time monitoring)
    c.Info("🎯 Starting real-time monitoring for %v...", duration)
    c.Info("Press Ctrl+C to stop early")
    fmt.Fprintln(c.writer)
    
    // Start monitoring in background
    resultChan := make(chan *actors.SenseResult, 1)
    errChan := make(chan error, 1)
    
    go func() {
        result, err := monitor.Sense(ctx, probeResult)
        if err != nil {
            errChan <- err
        } else {
            resultChan <- result
        }
    }()
    
    // Wait for completion or interruption
    select {
    case result := <-resultChan:
        return c.displayStreamResults(result)
    case err := <-errChan:
        return err
    case <-ctx.Done():
        c.Info("⏰ Monitoring duration completed")
        return nil
    }
}

// streamRecord records streams to file
func (c *Console) streamRecord(args []string) error {
    c.Info("📹 Recording STDIO streams to file...")
    
    // Parse arguments
    var (
        outputFile   string
        duration     = 30 * time.Second
    )
    
    for i := 0; i < len(args); i++ {
        switch args[i] {
        case "--output", "-o":
            if i+1 < len(args) {
                outputFile = args[i+1]
                i++
            }
        case "--duration", "-d":
            if i+1 < len(args) {
                var err error
                duration, err = time.ParseDuration(args[i+1])
                if err != nil {
                    return fmt.Errorf("invalid duration: %s", args[i+1])
                }
                i++
            }
        }
    }
    
    if outputFile == "" {
        outputFile = fmt.Sprintf("strigoi_stream_%s.jsonl", 
            time.Now().Format("20060102_150405"))
    }
    
    c.Info("Recording to: %s", outputFile)
    c.Info("Duration: %v", duration)
    
    // TODO: Implement recording logic
    c.Warn("Recording implementation coming soon!")
    
    return nil
}

// streamReplay replays a recorded session
func (c *Console) streamReplay(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("usage: stream/replay <recording-file>")
    }
    
    recordingFile := args[0]
    c.Info("🔄 Replaying session from: %s", recordingFile)
    
    // TODO: Implement replay logic
    c.Warn("Replay implementation coming soon!")
    
    return nil
}

// streamAnalyze analyzes captured streams
func (c *Console) streamAnalyze(args []string) error {
    c.Info("🔬 Analyzing captured streams...")
    
    // TODO: Implement analysis logic
    c.Warn("Analysis implementation coming soon!")
    
    return nil
}

// streamPatterns manages security patterns
func (c *Console) streamPatterns(args []string) error {
    if len(args) == 0 {
        return c.showPatternsHelp()
    }
    
    switch args[0] {
    case "list":
        return c.listPatterns()
    case "add":
        if len(args) < 2 {
            return fmt.Errorf("usage: stream/patterns add <pattern-file>")
        }
        return c.addPattern(args[1])
    case "remove":
        if len(args) < 2 {
            return fmt.Errorf("usage: stream/patterns remove <pattern-name>")
        }
        return c.removePattern(args[1])
    default:
        return c.showPatternsHelp()
    }
}

// streamStatus shows active monitors
func (c *Console) streamStatus(args []string) error {
    c.Info("📊 Stream Monitor Status")
    fmt.Fprintln(c.writer)
    
    // TODO: Get actual status from framework
    fmt.Fprintln(c.writer, "  Active Monitors: 0")
    fmt.Fprintln(c.writer, "  Total Events:    0")
    fmt.Fprintln(c.writer, "  Security Alerts: 0")
    fmt.Fprintln(c.writer)
    
    return nil
}

// Helper functions

func (c *Console) displayStreamResults(result *actors.SenseResult) error {
    fmt.Fprintln(c.writer)
    c.Success("📋 Stream Monitoring Complete")
    fmt.Fprintln(c.writer)
    
    // Count different types of results
    infoCount := 0
    warnCount := 0
    errorCount := 0
    
    for _, obs := range result.Observations {
        switch obs.Severity {
        case "error", "critical", "high":
            errorCount++
        case "warning", "medium":
            warnCount++
        default:
            infoCount++
        }
    }
    
    // Display summary
    c.Info("Summary:")
    c.Printf("  • Observations: %d (", len(result.Observations))
    if infoCount > 0 {
        c.infoColor.Printf("%d info", infoCount)
    }
    if warnCount > 0 {
        if infoCount > 0 {
            c.Printf(", ")
        }
        c.warnColor.Printf("%d warnings", warnCount)
    }
    if errorCount > 0 {
        if infoCount > 0 || warnCount > 0 {
            c.Printf(", ")
        }
        c.errorColor.Printf("%d errors", errorCount)
    }
    c.Printf(")\n")
    
    if len(result.Patterns) > 0 {
        c.Printf("  • Patterns detected: %d\n", len(result.Patterns))
    }
    if len(result.Risks) > 0 {
        c.Printf("  • Security risks: %d\n", len(result.Risks))
    }
    fmt.Fprintln(c.writer)
    
    // Display key observations (skip info level in brief mode)
    if warnCount > 0 || errorCount > 0 {
        c.infoColor.Fprintln(c.writer, "Key Observations:")
        for _, obs := range result.Observations {
            if obs.Severity != "info" {
                severity := c.getSeverityColor(obs.Severity)
                severity.Fprintf(c.writer, "  [%s] %s\n", 
                    strings.ToUpper(string(obs.Severity)), obs.Description)
            }
        }
        fmt.Fprintln(c.writer)
    }
    
    // Display patterns
    if len(result.Patterns) > 0 {
        c.infoColor.Fprintln(c.writer, "Detected Patterns:")
        for _, pattern := range result.Patterns {
            confidenceColor := c.successColor
            if pattern.Confidence < 0.7 {
                confidenceColor = c.warnColor
            }
            c.Printf("  • %s ", pattern.Name)
            confidenceColor.Printf("(%.0f%% confidence)\n", pattern.Confidence*100)
            if pattern.Description != "" {
                c.Printf("    %s\n", pattern.Description)
            }
        }
        fmt.Fprintln(c.writer)
    }
    
    // Display risks
    if len(result.Risks) > 0 {
        c.errorColor.Fprintln(c.writer, "🚨 Security Risks Detected:")
        for _, risk := range result.Risks {
            severity := c.getSeverityColor(risk.Severity)
            severity.Fprintf(c.writer, "\n  [%s] %s\n", 
                strings.ToUpper(string(risk.Severity)), risk.Title)
            
            // Wrap long descriptions
            desc := risk.Description
            if len(desc) > 70 {
                words := strings.Fields(desc)
                line := "    "
                for _, word := range words {
                    if len(line)+len(word)+1 > 74 {
                        fmt.Fprintln(c.writer, line)
                        line = "    " + word
                    } else {
                        if line != "    " {
                            line += " "
                        }
                        line += word
                    }
                }
                if line != "    " {
                    fmt.Fprintln(c.writer, line)
                }
            } else {
                fmt.Fprintf(c.writer, "    %s\n", desc)
            }
            
            if risk.Mitigation != "" {
                c.successColor.Printf("\n    💡 Mitigation: ")
                c.Printf("%s\n", risk.Mitigation)
            }
        }
        fmt.Fprintln(c.writer)
    }
    
    return nil
}

func (c *Console) showPatternsHelp() error {
    c.Info("Pattern Management")
    fmt.Fprintln(c.writer)
    fmt.Fprintln(c.writer, "  stream/patterns list          - List current patterns")
    fmt.Fprintln(c.writer, "  stream/patterns add <file>    - Add pattern file")
    fmt.Fprintln(c.writer, "  stream/patterns remove <name> - Remove pattern")
    fmt.Fprintln(c.writer)
    return nil
}

func (c *Console) listPatterns() error {
    c.Info("Security Patterns:")
    fmt.Fprintln(c.writer)
    
    // Default patterns
    patterns := []struct {
        name     string
        severity string
        desc     string
    }{
        {"AWS_CREDENTIALS", "critical", "AWS access key detection"},
        {"PRIVATE_KEY", "critical", "Private key detection"},
        {"API_KEY", "high", "API key detection"},
        {"COMMAND_INJECTION", "high", "Command injection patterns"},
        {"PATH_TRAVERSAL", "high", "Path traversal attempts"},
        {"BASE64_LARGE", "medium", "Large base64 encoded data"},
        {"SUSPICIOUS_URL", "medium", "Suspicious URL patterns"},
    }
    
    for _, p := range patterns {
        severity := c.getSeverityColor(p.severity)
        severity.Fprintf(c.writer, "  • %-20s [%s] %s\n", 
            p.name, strings.ToUpper(p.severity), p.desc)
    }
    
    fmt.Fprintln(c.writer)
    return nil
}

func (c *Console) addPattern(file string) error {
    c.Info("Adding pattern from: %s", file)
    // TODO: Implement pattern loading
    c.Warn("Pattern loading implementation coming soon!")
    return nil
}

func (c *Console) removePattern(name string) error {
    c.Info("Removing pattern: %s", name)
    // TODO: Implement pattern removal
    c.Warn("Pattern removal implementation coming soon!")
    return nil
}

func (c *Console) getSeverityColor(severity string) *color.Color {
    switch strings.ToLower(severity) {
    case "critical":
        return color.New(color.FgRed, color.Bold)
    case "high":
        return c.errorColor
    case "medium":
        return c.warnColor
    case "low":
        return c.infoColor
    default:
        return color.New(color.FgWhite)
    }
}