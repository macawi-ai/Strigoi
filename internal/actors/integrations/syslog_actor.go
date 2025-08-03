package integrations

import (
    "context"
    "fmt"
    "log/syslog"
    "os"
    "sync"
    
    "github.com/macawi-ai/strigoi/internal/actors"
    "github.com/macawi-ai/strigoi/internal/stream"
)

// SyslogActor sends events to local syslog
type SyslogActor struct {
    *actors.BaseActor
    
    // Syslog writer
    writer       *syslog.Writer
    
    // Configuration
    facility     syslog.Priority
    tag          string
    
    // Filtering
    minSeverity  string
    includeTypes []string
    
    // State
    mu           sync.RWMutex
    connected    bool
    eventCount   int64
}

// facilityToString converts syslog.Priority to string
func facilityToString(p syslog.Priority) string {
    switch p {
    case syslog.LOG_KERN:
        return "kern"
    case syslog.LOG_USER:
        return "user"
    case syslog.LOG_MAIL:
        return "mail"
    case syslog.LOG_DAEMON:
        return "daemon"
    case syslog.LOG_AUTH:
        return "auth"
    case syslog.LOG_SYSLOG:
        return "syslog"
    case syslog.LOG_LPR:
        return "lpr"
    case syslog.LOG_NEWS:
        return "news"
    case syslog.LOG_UUCP:
        return "uucp"
    case syslog.LOG_CRON:
        return "cron"
    case syslog.LOG_AUTHPRIV:
        return "authpriv"
    case syslog.LOG_FTP:
        return "ftp"
    case syslog.LOG_LOCAL0:
        return "local0"
    case syslog.LOG_LOCAL1:
        return "local1"
    case syslog.LOG_LOCAL2:
        return "local2"
    case syslog.LOG_LOCAL3:
        return "local3"
    case syslog.LOG_LOCAL4:
        return "local4"
    case syslog.LOG_LOCAL5:
        return "local5"
    case syslog.LOG_LOCAL6:
        return "local6"
    case syslog.LOG_LOCAL7:
        return "local7"
    default:
        return fmt.Sprintf("facility(%d)", p)
    }
}

// NewSyslogActor creates a new syslog integration actor
func NewSyslogActor() *SyslogActor {
    actor := &SyslogActor{
        BaseActor: actors.NewBaseActor(
            "syslog_integration",
            "Send security events to local syslog daemon",
            "integration",
        ),
        facility:    syslog.LOG_LOCAL0, // LOG_SECURITY not available in Go's syslog
        tag:         "strigoi",
        minSeverity: "medium",
    }
    
    // Define capabilities
    actor.AddCapability(actors.Capability{
        Name:        "syslog_write",
        Description: "Write events to local syslog",
        DataTypes:   []string{"event", "alert", "log"},
    })
    
    actor.AddCapability(actors.Capability{
        Name:        "severity_filter",
        Description: "Filter events by severity level",
        DataTypes:   []string{"filter"},
    })
    
    actor.AddCapability(actors.Capability{
        Name:        "format_cef",
        Description: "Format events in Common Event Format",
        DataTypes:   []string{"cef"},
    })
    
    actor.SetInputTypes([]string{"stream_event", "security_alert", "log_message"})
    actor.SetOutputType("syslog")
    
    return actor
}

// Probe checks syslog connectivity
func (s *SyslogActor) Probe(ctx context.Context, target actors.Target) (*actors.ProbeResult, error) {
    discoveries := []actors.Discovery{}
    
    // Try to connect to syslog
    writer, err := syslog.New(s.facility, s.tag)
    if err != nil {
        discoveries = append(discoveries, actors.Discovery{
            Type:       "syslog_daemon",
            Identifier: "local",
            Properties: map[string]interface{}{
                "available": false,
                "error":     err.Error(),
            },
            Confidence: 1.0,
        })
    } else {
        // Test write
        testErr := writer.Info("Strigoi syslog integration test")
        writer.Close()
        
        discoveries = append(discoveries, actors.Discovery{
            Type:       "syslog_daemon",
            Identifier: "local",
            Properties: map[string]interface{}{
                "available":  true,
                "writable":   testErr == nil,
                "facility":   facilityToString(s.facility),
                "tag":        s.tag,
            },
            Confidence: 1.0,
        })
    }
    
    // Check rsyslog/syslog-ng configuration
    configs := []string{
        "/etc/rsyslog.conf",
        "/etc/syslog-ng/syslog-ng.conf",
        "/etc/syslog.conf",
    }
    
    for _, config := range configs {
        if info, err := os.Stat(config); err == nil {
            discoveries = append(discoveries, actors.Discovery{
                Type:       "syslog_config",
                Identifier: config,
                Properties: map[string]interface{}{
                    "exists":   true,
                    "readable": info.Mode().Perm()&0400 != 0,
                    "modified": info.ModTime(),
                },
                Confidence: 0.8,
            })
        }
    }
    
    return &actors.ProbeResult{
        ActorName:   s.Name(),
        Target:      target,
        Discoveries: discoveries,
        RawData: map[string]interface{}{
            "facility":     facilityToString(s.facility),
            "tag":          s.tag,
            "min_severity": s.minSeverity,
        },
    }, nil
}

// Sense starts syslog forwarding
func (s *SyslogActor) Sense(ctx context.Context, data *actors.ProbeResult) (*actors.SenseResult, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.connected {
        return nil, fmt.Errorf("already connected to syslog")
    }
    
    // Connect to syslog
    writer, err := syslog.New(s.facility, s.tag)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to syslog: %w", err)
    }
    
    s.writer = writer
    s.connected = true
    
    // Log startup
    s.writer.Info("Strigoi syslog integration started")
    
    observations := []actors.Observation{
        {
            Layer:       "integration",
            Description: "Connected to local syslog daemon",
            Evidence: map[string]interface{}{
                "facility": facilityToString(s.facility),
                "tag":      s.tag,
            },
            Severity: "info",
        },
    }
    
    return &actors.SenseResult{
        ActorName:    s.Name(),
        Observations: observations,
        Patterns:     []actors.Pattern{},
        Risks:        []actors.Risk{},
    }, nil
}

// Transform processes events and sends to syslog
func (s *SyslogActor) Transform(ctx context.Context, input interface{}) (interface{}, error) {
    s.mu.RLock()
    if !s.connected || s.writer == nil {
        s.mu.RUnlock()
        return nil, fmt.Errorf("not connected to syslog")
    }
    s.mu.RUnlock()
    
    var sent bool
    
    switch v := input.(type) {
    case *stream.StreamEvent:
        if s.shouldLog(v.Severity, string(v.Type)) {
            msg := s.formatStreamEvent(v)
            sent = s.sendToSyslog(v.Severity, msg)
        }
        
    case *stream.SecurityAlert:
        if s.shouldLog(v.Severity, "alert") {
            msg := s.formatSecurityAlert(v)
            sent = s.sendToSyslog(v.Severity, msg)
        }
        
    case string:
        // Generic log message
        sent = s.sendToSyslog("info", v)
        
    default:
        return nil, fmt.Errorf("unsupported input type: %T", input)
    }
    
    if sent {
        s.mu.Lock()
        s.eventCount++
        s.mu.Unlock()
    }
    
    return map[string]interface{}{
        "sent":        sent,
        "total_count": s.eventCount,
    }, nil
}

// Format stream event for syslog
func (s *SyslogActor) formatStreamEvent(event *stream.StreamEvent) string {
    // Common Event Format (CEF)
    return fmt.Sprintf(
        "CEF:0|Macawi|Strigoi|1.0|%s|%s|%s|pid=%d direction=%s size=%d",
        event.Type,
        event.Summary,
        s.mapSeverity(event.Severity),
        event.PID,
        event.Direction,
        event.Size,
    )
}

// Format security alert for syslog
func (s *SyslogActor) formatSecurityAlert(alert *stream.SecurityAlert) string {
    return fmt.Sprintf(
        "CEF:0|Macawi|Strigoi|1.0|SecurityAlert|%s|%s|cat=%s pid=%d pattern=%s blocked=%t",
        alert.Title,
        s.mapSeverity(alert.Severity),
        alert.Category,
        alert.PID,
        alert.Pattern,
        alert.Blocked,
    )
}

// Send to syslog with appropriate priority
func (s *SyslogActor) sendToSyslog(severity, message string) bool {
    var err error
    
    switch severity {
    case "critical":
        err = s.writer.Crit(message)
    case "high":
        err = s.writer.Alert(message)
    case "medium":
        err = s.writer.Warning(message)
    case "low":
        err = s.writer.Notice(message)
    default:
        err = s.writer.Info(message)
    }
    
    return err == nil
}

// Check if event should be logged based on filters
func (s *SyslogActor) shouldLog(severity string, eventType string) bool {
    // Check severity threshold
    if !s.meetsMinSeverity(severity) {
        return false
    }
    
    // Check type filter if configured
    if len(s.includeTypes) > 0 {
        found := false
        for _, t := range s.includeTypes {
            if t == eventType {
                found = true
                break
            }
        }
        if !found {
            return false
        }
    }
    
    return true
}

// Check if severity meets minimum threshold
func (s *SyslogActor) meetsMinSeverity(severity string) bool {
    severityLevels := map[string]int{
        "critical": 5,
        "high":     4,
        "medium":   3,
        "low":      2,
        "info":     1,
    }
    
    eventLevel := severityLevels[severity]
    minLevel := severityLevels[s.minSeverity]
    
    return eventLevel >= minLevel
}

// Map severity to CEF severity (0-10)
func (s *SyslogActor) mapSeverity(severity string) string {
    severityMap := map[string]string{
        "critical": "10",
        "high":     "8",
        "medium":   "5",
        "low":      "3",
        "info":     "1",
    }
    
    if cef, ok := severityMap[severity]; ok {
        return cef
    }
    return "0"
}

// Stop the syslog actor
func (s *SyslogActor) Stop() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if !s.connected || s.writer == nil {
        return nil
    }
    
    // Log shutdown
    s.writer.Info(fmt.Sprintf("Strigoi syslog integration stopped (sent %d events)", s.eventCount))
    
    // Close connection
    err := s.writer.Close()
    s.writer = nil
    s.connected = false
    
    return err
}

// Configure updates actor configuration
func (s *SyslogActor) Configure(config map[string]interface{}) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if facility, ok := config["facility"].(string); ok {
        // Parse facility string to syslog.Priority
        // This is simplified - in production, implement full mapping
        switch facility {
        case "security", "local0":
            s.facility = syslog.LOG_LOCAL0
        case "daemon":
            s.facility = syslog.LOG_DAEMON
        case "local1":
            s.facility = syslog.LOG_LOCAL1
        case "auth":
            s.facility = syslog.LOG_AUTH
        case "authpriv":
            s.facility = syslog.LOG_AUTHPRIV
        }
    }
    
    if tag, ok := config["tag"].(string); ok {
        s.tag = tag
    }
    
    if severity, ok := config["min_severity"].(string); ok {
        s.minSeverity = severity
    }
    
    if types, ok := config["include_types"].([]string); ok {
        s.includeTypes = types
    }
    
    return nil
}