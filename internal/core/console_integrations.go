package core

import (
    "context"
    "fmt"
    "strings"
    
    "github.com/macawi-ai/strigoi/internal/actors"
    "github.com/macawi-ai/strigoi/internal/actors/integrations"
)

// processIntegrationsCommand handles integrations/ subcommands
func (c *Console) processIntegrationsCommand(subcommand string, args []string) error {
    if subcommand == "" {
        return c.showIntegrationsHelp()
    }
    
    switch subcommand {
    case "list":
        return c.listIntegrations()
    case "enable":
        if len(args) == 0 {
            return fmt.Errorf("usage: integrations/enable <integration-name>")
        }
        return c.enableIntegration(args[0], args[1:])
    case "prometheus":
        return c.prometheusIntegration(args)
    case "syslog":
        return c.syslogIntegration(args)
    case "file":
        return c.fileIntegration(args)
    default:
        c.Error("Unknown integration: %s", subcommand)
        return c.showIntegrationsHelp()
    }
}

// showIntegrationsHelp displays integrations help
func (c *Console) showIntegrationsHelp() error {
    c.Info("Integration Commands - External System Connections")
    fmt.Fprintln(c.writer)
    fmt.Fprintln(c.writer, "  integrations/list            - List available integrations")
    fmt.Fprintln(c.writer, "  integrations/enable <name>   - Quick enable integration")
    fmt.Fprintln(c.writer)
    c.infoColor.Fprintln(c.writer, "Available Integrations:")
    fmt.Fprintln(c.writer, "  integrations/prometheus      - Prometheus metrics export")
    fmt.Fprintln(c.writer, "  integrations/syslog          - Local syslog integration")
    fmt.Fprintln(c.writer, "  integrations/file            - File logger integration")
    fmt.Fprintln(c.writer)
    c.infoColor.Fprintln(c.writer, "Examples:")
    fmt.Fprintln(c.writer, "  integrations/prometheus start")
    fmt.Fprintln(c.writer, "  integrations/syslog connect --facility security")
    fmt.Fprintln(c.writer, "  integrations/file configure --log-dir /var/log/strigoi")
    fmt.Fprintln(c.writer)
    return nil
}

// listIntegrations shows available integrations
func (c *Console) listIntegrations() error {
    c.Info("Available Integrations")
    fmt.Fprintln(c.writer)
    
    integrations := []struct {
        name   string
        desc   string
        status string
    }{
        {
            name:   "prometheus",
            desc:   "Export metrics in Prometheus format",
            status: "inactive",
        },
        {
            name:   "syslog",
            desc:   "Send events to local syslog daemon",
            status: "inactive",
        },
        {
            name:   "file",
            desc:   "Log events to custom directory",
            status: "inactive",
        },
    }
    
    // Table header
    fmt.Fprintf(c.writer, "  %-15s %-45s %s\n", "NAME", "DESCRIPTION", "STATUS")
    fmt.Fprintf(c.writer, "  %-15s %-45s %s\n", 
        strings.Repeat("-", 15), 
        strings.Repeat("-", 45), 
        strings.Repeat("-", 10))
    
    // List integrations
    for _, intg := range integrations {
        statusColor := c.errorColor
        if intg.status == "active" {
            statusColor = c.successColor
        }
        
        fmt.Fprintf(c.writer, "  %-15s %-45s ", intg.name, intg.desc)
        statusColor.Fprintln(c.writer, intg.status)
    }
    
    fmt.Fprintln(c.writer)
    return nil
}

// enableIntegration quickly enables an integration
func (c *Console) enableIntegration(name string, args []string) error {
    c.Info("Enabling %s integration...", name)
    
    switch name {
    case "prometheus":
        return c.prometheusStart(args)
    case "syslog":
        return c.syslogConnect(args)
    case "file":
        return c.fileStart(args)
    default:
        return fmt.Errorf("unknown integration: %s", name)
    }
}

// prometheusIntegration handles Prometheus subcommands
func (c *Console) prometheusIntegration(args []string) error {
    if len(args) == 0 {
        return c.showPrometheusHelp()
    }
    
    switch args[0] {
    case "start":
        return c.prometheusStart(args[1:])
    case "status":
        return c.prometheusStatus()
    case "configure":
        return c.prometheusConfigure(args[1:])
    case "stop":
        return c.prometheusStop()
    default:
        return c.showPrometheusHelp()
    }
}

func (c *Console) showPrometheusHelp() error {
    c.Info("Prometheus Integration")
    fmt.Fprintln(c.writer)
    fmt.Fprintln(c.writer, "  integrations/prometheus start      - Start metrics endpoint")
    fmt.Fprintln(c.writer, "  integrations/prometheus status     - Check metrics status")
    fmt.Fprintln(c.writer, "  integrations/prometheus configure  - Set endpoint/port")
    fmt.Fprintln(c.writer, "  integrations/prometheus stop       - Stop metrics export")
    fmt.Fprintln(c.writer)
    c.infoColor.Fprintln(c.writer, "Options:")
    fmt.Fprintln(c.writer, "  --port <port>          - Metrics port (default: 9100)")
    fmt.Fprintln(c.writer, "  --push-gateway <url>   - Push gateway URL")
    fmt.Fprintln(c.writer)
    return nil
}

func (c *Console) prometheusStart(args []string) error {
    // Parse arguments
    config := map[string]interface{}{
        "listen_addr": ":9100",
    }
    
    for i := 0; i < len(args); i++ {
        switch args[i] {
        case "--port":
            if i+1 < len(args) {
                config["listen_addr"] = ":" + args[i+1]
                i++
            }
        case "--push-gateway":
            if i+1 < len(args) {
                config["push_gateway"] = args[i+1]
                i++
            }
        }
    }
    
    // Create Prometheus actor
    actor := integrations.NewPrometheusActor()
    
    // Configure
    if err := actor.Configure(config); err != nil {
        return fmt.Errorf("configuration failed: %w", err)
    }
    
    // Probe
    target := actors.Target{Type: "prometheus"}
    probeResult, err := actor.Probe(context.Background(), target)
    if err != nil {
        return fmt.Errorf("probe failed: %w", err)
    }
    
    // Check discoveries
    for _, disc := range probeResult.Discoveries {
        if disc.Type == "port_availability" {
            if !disc.Properties["available"].(bool) {
                return fmt.Errorf("port not available: %v", disc.Properties["error"])
            }
        }
    }
    
    // Start
    senseResult, err := actor.Sense(context.Background(), probeResult)
    if err != nil {
        return fmt.Errorf("failed to start: %w", err)
    }
    
    // Display result
    for _, obs := range senseResult.Observations {
        c.Success("%s", obs.Description)
        if evidence, ok := obs.Evidence.(map[string]interface{}); ok {
            if url, ok := evidence["url"].(string); ok {
                c.Info("Metrics available at: %s", url)
            }
        }
    }
    
    // Store actor in framework
    // TODO: Framework needs to track active actors
    
    return nil
}

func (c *Console) prometheusStatus() error {
    c.Info("Prometheus Integration Status")
    fmt.Fprintln(c.writer)
    
    // TODO: Get actual status from framework
    fmt.Fprintln(c.writer, "  Status:        inactive")
    fmt.Fprintln(c.writer, "  Endpoint:      -")
    fmt.Fprintln(c.writer, "  Events:        0")
    fmt.Fprintln(c.writer, "  Last Updated:  -")
    fmt.Fprintln(c.writer)
    
    return nil
}

func (c *Console) prometheusConfigure(args []string) error {
    c.Info("Configuring Prometheus integration...")
    // TODO: Implement configuration updates
    c.Warn("Configuration update coming soon!")
    return nil
}

func (c *Console) prometheusStop() error {
    c.Info("Stopping Prometheus integration...")
    // TODO: Get actor from framework and stop it
    c.Warn("Stop functionality coming soon!")
    return nil
}

// syslogIntegration handles syslog subcommands
func (c *Console) syslogIntegration(args []string) error {
    if len(args) == 0 {
        return c.showSyslogHelp()
    }
    
    switch args[0] {
    case "connect":
        return c.syslogConnect(args[1:])
    case "filter":
        return c.syslogFilter(args[1:])
    case "test":
        return c.syslogTest()
    case "disconnect":
        return c.syslogDisconnect()
    default:
        return c.showSyslogHelp()
    }
}

func (c *Console) showSyslogHelp() error {
    c.Info("Syslog Integration")
    fmt.Fprintln(c.writer)
    fmt.Fprintln(c.writer, "  integrations/syslog connect     - Connect to syslog daemon")
    fmt.Fprintln(c.writer, "  integrations/syslog filter      - Set severity filters")
    fmt.Fprintln(c.writer, "  integrations/syslog test        - Send test message")
    fmt.Fprintln(c.writer, "  integrations/syslog disconnect  - Stop syslog forwarding")
    fmt.Fprintln(c.writer)
    c.infoColor.Fprintln(c.writer, "Options:")
    fmt.Fprintln(c.writer, "  --facility <name>      - Syslog facility (default: security)")
    fmt.Fprintln(c.writer, "  --tag <tag>            - Syslog tag (default: strigoi)")
    fmt.Fprintln(c.writer, "  --min-severity <level> - Minimum severity (default: medium)")
    fmt.Fprintln(c.writer)
    return nil
}

func (c *Console) syslogConnect(args []string) error {
    // Parse arguments
    config := map[string]interface{}{
        "facility":     "security",
        "tag":          "strigoi",
        "min_severity": "medium",
    }
    
    for i := 0; i < len(args); i++ {
        switch args[i] {
        case "--facility":
            if i+1 < len(args) {
                config["facility"] = args[i+1]
                i++
            }
        case "--tag":
            if i+1 < len(args) {
                config["tag"] = args[i+1]
                i++
            }
        case "--min-severity":
            if i+1 < len(args) {
                config["min_severity"] = args[i+1]
                i++
            }
        }
    }
    
    c.Info("Connecting to syslog daemon...")
    c.Info("Facility: %s", config["facility"])
    c.Info("Tag: %s", config["tag"])
    
    // TODO: Create and start syslog actor
    c.Warn("Syslog integration coming soon!")
    
    return nil
}

func (c *Console) syslogFilter(args []string) error {
    c.Info("Setting syslog filters...")
    // TODO: Implement filter configuration
    c.Warn("Filter configuration coming soon!")
    return nil
}

func (c *Console) syslogTest() error {
    c.Info("Sending test message to syslog...")
    // TODO: Send test message
    c.Warn("Test message coming soon!")
    return nil
}

func (c *Console) syslogDisconnect() error {
    c.Info("Disconnecting from syslog...")
    // TODO: Stop syslog actor
    c.Warn("Disconnect coming soon!")
    return nil
}

// fileIntegration handles file logger subcommands
func (c *Console) fileIntegration(args []string) error {
    if len(args) == 0 {
        return c.showFileHelp()
    }
    
    switch args[0] {
    case "configure":
        return c.fileConfigure(args[1:])
    case "start":
        return c.fileStart(args[1:])
    case "rotate":
        return c.fileRotate()
    case "stop":
        return c.fileStop()
    default:
        return c.showFileHelp()
    }
}

func (c *Console) showFileHelp() error {
    c.Info("File Logger Integration")
    fmt.Fprintln(c.writer)
    fmt.Fprintln(c.writer, "  integrations/file configure  - Set log directory")
    fmt.Fprintln(c.writer, "  integrations/file start      - Start file logging")
    fmt.Fprintln(c.writer, "  integrations/file rotate     - Force log rotation")
    fmt.Fprintln(c.writer, "  integrations/file stop       - Stop file logging")
    fmt.Fprintln(c.writer)
    c.infoColor.Fprintln(c.writer, "Options:")
    fmt.Fprintln(c.writer, "  --log-dir <path>       - Log directory (default: /var/log/strigoi)")
    fmt.Fprintln(c.writer, "  --format <type>        - Output format: json, jsonl, csv, text")
    fmt.Fprintln(c.writer, "  --rotate-size <bytes>  - Rotation size (default: 100MB)")
    fmt.Fprintln(c.writer, "  --max-files <num>      - Max log files (default: 10)")
    fmt.Fprintln(c.writer)
    return nil
}

func (c *Console) fileConfigure(args []string) error {
    c.Info("Configuring file logger...")
    
    // Parse arguments
    for i := 0; i < len(args); i++ {
        switch args[i] {
        case "--log-dir":
            if i+1 < len(args) {
                c.Success("Log directory: %s", args[i+1])
                i++
            }
        case "--format":
            if i+1 < len(args) {
                c.Success("Format: %s", args[i+1])
                i++
            }
        }
    }
    
    // TODO: Store configuration
    c.Warn("Configuration storage coming soon!")
    
    return nil
}

func (c *Console) fileStart(args []string) error {
    c.Info("Starting file logger...")
    
    // Parse arguments
    config := map[string]interface{}{
        "log_dir": "/var/log/strigoi",
        "format":  "jsonl",
    }
    
    for i := 0; i < len(args); i++ {
        switch args[i] {
        case "--log-dir":
            if i+1 < len(args) {
                config["log_dir"] = args[i+1]
                i++
            }
        case "--format":
            if i+1 < len(args) {
                config["format"] = args[i+1]
                i++
            }
        }
    }
    
    c.Success("Logging to: %s", config["log_dir"])
    c.Success("Format: %s", config["format"])
    
    // TODO: Create and start file logger actor
    c.Warn("File logger implementation coming soon!")
    
    return nil
}

func (c *Console) fileRotate() error {
    c.Info("Rotating log files...")
    // TODO: Trigger rotation
    c.Warn("Rotation coming soon!")
    return nil
}

func (c *Console) fileStop() error {
    c.Info("Stopping file logger...")
    // TODO: Stop file logger actor
    c.Warn("Stop functionality coming soon!")
    return nil
}