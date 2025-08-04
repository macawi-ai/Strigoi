package probe

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// EventLogger handles structured logging to JSONL format.
type EventLogger struct {
	file     *os.File
	encoder  *json.Encoder
	mu       sync.Mutex
	filename string
}

// NewEventLogger creates a new event logger.
func NewEventLogger(filename string) (*EventLogger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &EventLogger{
		file:     file,
		encoder:  json.NewEncoder(file),
		filename: filename,
	}, nil
}

// Close closes the log file.
func (l *EventLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// LogEvent logs a generic event.
func (l *EventLogger) LogEvent(eventType string, data map[string]interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	event := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"type":      eventType,
		"data":      data,
	}

	return l.encoder.Encode(event)
}

// LogVulnerability logs a vulnerability finding.
func (l *EventLogger) LogVulnerability(vuln StreamVulnerability) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	event := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"type":      "vulnerability",
		"vuln": map[string]interface{}{
			"id":         vuln.ID,
			"severity":   vuln.Severity,
			"type":       vuln.Type,
			"subtype":    vuln.Subtype,
			"evidence":   vuln.Evidence,
			"location":   vuln.Location,
			"context":    vuln.Context,
			"confidence": vuln.Confidence,
			"process": map[string]interface{}{
				"pid":  vuln.ProcessInfo.PID,
				"name": vuln.ProcessInfo.Name,
				"cmd":  vuln.ProcessInfo.CommandLine,
			},
		},
	}

	return l.encoder.Encode(event)
}

// LogError logs an error event.
func (l *EventLogger) LogError(err error) error {
	return l.LogEvent("error", map[string]interface{}{
		"error":   err.Error(),
		"context": "stream_monitoring",
	})
}

// LogStatistics logs monitoring statistics.
func (l *EventLogger) LogStatistics(stats map[string]interface{}) error {
	return l.LogEvent("statistics", stats)
}

// LogActivity logs stream activity (when show-activity is enabled).
func (l *EventLogger) LogActivity(target StreamTarget, stream string, preview string, bytes int) error {
	return l.LogEvent("activity", map[string]interface{}{
		"process": map[string]interface{}{
			"pid":  target.PID,
			"name": target.Name,
		},
		"stream":  stream,
		"preview": preview,
		"bytes":   bytes,
	})
}

// VulnerabilityDetector manages vulnerability detection logic.
type VulnerabilityDetector struct {
	patterns   []VulnPattern
	statistics DetectorStats
	mu         sync.RWMutex
}

// VulnPattern defines a vulnerability detection pattern.
type VulnPattern struct {
	Name        string
	Description string
	Detector    func([]byte) []StreamVulnerability
}

// DetectorStats tracks detection statistics.
type DetectorStats struct {
	TotalScans     int64
	VulnsDetected  int64
	FalsePositives int64
	LastDetection  time.Time
}

// NewVulnerabilityDetector creates a new vulnerability detector.
func NewVulnerabilityDetector() *VulnerabilityDetector {
	v := &VulnerabilityDetector{
		patterns: make([]VulnPattern, 0),
	}

	// Initialize detection patterns
	v.initializePatterns()
	return v
}

// initializePatterns sets up vulnerability patterns.
func (v *VulnerabilityDetector) initializePatterns() {
	// Add detection patterns
	v.patterns = append(v.patterns, VulnPattern{
		Name:        "sudo_chain",
		Description: "Detects potential sudo privilege escalation chains",
		Detector:    v.detectSudoChain,
	})

	v.patterns = append(v.patterns, VulnPattern{
		Name:        "exposed_secrets",
		Description: "Detects exposed secrets in environment variables",
		Detector:    v.detectExposedSecrets,
	})

	v.patterns = append(v.patterns, VulnPattern{
		Name:        "insecure_protocols",
		Description: "Detects use of insecure protocols",
		Detector:    v.detectInsecureProtocols,
	})
}

// Detect runs all detection patterns on data.
func (v *VulnerabilityDetector) Detect(data []byte) []StreamVulnerability {
	v.mu.Lock()
	v.statistics.TotalScans++
	v.mu.Unlock()

	allVulns := []StreamVulnerability{}

	for _, pattern := range v.patterns {
		vulns := pattern.Detector(data)
		allVulns = append(allVulns, vulns...)
	}

	if len(allVulns) > 0 {
		v.mu.Lock()
		v.statistics.VulnsDetected += int64(len(allVulns))
		v.statistics.LastDetection = time.Now()
		v.mu.Unlock()
	}

	return allVulns
}

// detectSudoChain checks for sudo privilege escalation patterns.
func (v *VulnerabilityDetector) detectSudoChain(data []byte) []StreamVulnerability {
	vulns := []StreamVulnerability{}
	dataStr := string(data)

	// Check for sudo invocations with NOPASSWD
	if strings.Contains(dataStr, "NOPASSWD") && strings.Contains(dataStr, "sudo") {
		vuln := StreamVulnerability{
			ID:         fmt.Sprintf("SUDO-%d", time.Now().UnixNano()),
			Type:       "privilege_escalation",
			Subtype:    "sudo_nopasswd",
			Evidence:   "[NOPASSWD sudo detected]",
			Context:    "Potential sudo chain vulnerability",
			Confidence: 0.8,
			Severity:   "high",
			Timestamp:  time.Now(),
		}
		vulns = append(vulns, vuln)
	}

	// Check for sudo -l output showing exploitable entries
	if strings.Contains(dataStr, "may run the following commands") {
		vuln := StreamVulnerability{
			ID:         fmt.Sprintf("SUDO-%d", time.Now().UnixNano()),
			Type:       "information_disclosure",
			Subtype:    "sudo_permissions",
			Evidence:   "[sudo -l output exposed]",
			Context:    "Sudo permissions enumeration",
			Confidence: 0.9,
			Severity:   "medium",
			Timestamp:  time.Now(),
		}
		vulns = append(vulns, vuln)
	}

	return vulns
}

// detectExposedSecrets checks for secrets in environment variables.
func (v *VulnerabilityDetector) detectExposedSecrets(data []byte) []StreamVulnerability {
	vulns := []StreamVulnerability{}
	dataStr := string(data)

	// Check for environment variable patterns
	envPatterns := map[string]string{
		"AWS_SECRET_ACCESS_KEY": "aws_credentials",
		"GITHUB_TOKEN":          "github_token",
		"NPM_TOKEN":             "npm_token",
		"DOCKER_PASSWORD":       "docker_credentials",
		"DATABASE_URL":          "database_connection",
	}

	for pattern, subtype := range envPatterns {
		if strings.Contains(dataStr, pattern) {
			vuln := StreamVulnerability{
				ID:         fmt.Sprintf("ENV-%d", time.Now().UnixNano()),
				Type:       "credential_exposure",
				Subtype:    subtype,
				Evidence:   pattern + "=****",
				Context:    "Environment variable exposed",
				Confidence: 0.95,
				Severity:   "critical",
				Timestamp:  time.Now(),
			}
			vulns = append(vulns, vuln)
		}
	}

	return vulns
}

// detectInsecureProtocols checks for use of insecure protocols.
func (v *VulnerabilityDetector) detectInsecureProtocols(data []byte) []StreamVulnerability {
	vulns := []StreamVulnerability{}
	dataStr := strings.ToLower(string(data))

	// Check for insecure protocols
	insecureProtos := map[string]string{
		"http://":   "http_plaintext",
		"ftp://":    "ftp_plaintext",
		"telnet://": "telnet_plaintext",
		"smtp://":   "smtp_plaintext",
	}

	for proto, subtype := range insecureProtos {
		if strings.Contains(dataStr, proto) && !strings.Contains(dataStr, "http://localhost") {
			vuln := StreamVulnerability{
				ID:         fmt.Sprintf("PROTO-%d", time.Now().UnixNano()),
				Type:       "insecure_protocol",
				Subtype:    subtype,
				Evidence:   "[" + proto + " URL detected]",
				Context:    "Insecure protocol in use",
				Confidence: 0.85,
				Severity:   "medium",
				Timestamp:  time.Now(),
			}
			vulns = append(vulns, vuln)
		}
	}

	return vulns
}

// GetStatistics returns detector statistics.
func (v *VulnerabilityDetector) GetStatistics() DetectorStats {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.statistics
}

// StreamExporter handles exporting stream data to various formats.
type StreamExporter struct {
	format string
	output string
}

// NewStreamExporter creates a new stream exporter.
func NewStreamExporter(format, output string) *StreamExporter {
	return &StreamExporter{
		format: format,
		output: output,
	}
}

// Export exports stream data in the specified format.
func (e *StreamExporter) Export(data interface{}) error {
	switch e.format {
	case "serial-studio":
		return e.exportSerialStudio(data)
	case "wireshark":
		return e.exportWireshark(data)
	case "csv":
		return e.exportCSV(data)
	default:
		return fmt.Errorf("unsupported export format: %s", e.format)
	}
}

// exportSerialStudio exports to Serial Studio format.
func (e *StreamExporter) exportSerialStudio(_ interface{}) error {
	// Implementation would create Serial Studio project file
	return fmt.Errorf("serial studio export not yet implemented")
}

// exportWireshark exports to PCAP format.
func (e *StreamExporter) exportWireshark(_ interface{}) error {
	// Implementation would create PCAP file
	return fmt.Errorf("wireshark export not yet implemented")
}

// exportCSV exports to CSV format.
func (e *StreamExporter) exportCSV(_ interface{}) error {
	// Implementation would create CSV file
	return fmt.Errorf("CSV export not yet implemented")
}
