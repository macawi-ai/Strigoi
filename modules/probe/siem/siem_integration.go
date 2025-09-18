package siem

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/modules/probe"
)

// Integration defines the interface for SIEM systems
type Integration interface {
	// Initialize sets up the connection to the SIEM
	Initialize(config Config) error

	// SendEvent sends a single security event
	SendEvent(event *SecurityEvent) error

	// SendBatch sends multiple events in a batch
	SendBatch(events []*SecurityEvent) error

	// HealthCheck verifies the connection is working
	HealthCheck() error

	// Close cleans up resources
	Close() error
}

// Config holds SIEM connection configuration
type Config struct {
	Type          string                 `json:"type"`           // elasticsearch, splunk, etc.
	Endpoint      string                 `json:"endpoint"`       // SIEM endpoint URL
	APIKey        string                 `json:"api_key"`        // API key for authentication
	Username      string                 `json:"username"`       // Username if using basic auth
	Password      string                 `json:"password"`       // Password if using basic auth
	Index         string                 `json:"index"`          // Index/source name
	BatchSize     int                    `json:"batch_size"`     // Max events per batch
	FlushInterval time.Duration          `json:"flush_interval"` // Auto-flush interval
	TLS           TLSConfig              `json:"tls"`            // TLS configuration
	Extra         map[string]interface{} `json:"extra"`          // Additional config
}

// TLSConfig holds TLS settings
type TLSConfig struct {
	Enabled            bool   `json:"enabled"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	CACert             string `json:"ca_cert"`
	ClientCert         string `json:"client_cert"`
	ClientKey          string `json:"client_key"`
}

// SecurityEvent represents a normalized security event for SIEM
type SecurityEvent struct {
	// Core fields (ECS compliant)
	Timestamp     time.Time `json:"@timestamp"`
	Message       string    `json:"message"`
	EventType     string    `json:"event.type"`
	EventCategory []string  `json:"event.category"`
	EventSeverity int       `json:"event.severity"`
	EventRisk     string    `json:"event.risk_score_norm"`

	// Strigoi specific
	StrigoiVersion string `json:"strigoi.version"`
	SessionID      string `json:"strigoi.session_id"`
	Protocol       string `json:"strigoi.protocol"`
	Dissector      string `json:"strigoi.dissector"`

	// Network fields
	SourceIP        string `json:"source.ip,omitempty"`
	SourcePort      int    `json:"source.port,omitempty"`
	DestinationIP   string `json:"destination.ip,omitempty"`
	DestinationPort int    `json:"destination.port,omitempty"`
	NetworkProtocol string `json:"network.protocol,omitempty"`
	NetworkBytes    int64  `json:"network.bytes,omitempty"`

	// Vulnerability details
	VulnerabilityType  string  `json:"vulnerability.type,omitempty"`
	VulnerabilityName  string  `json:"vulnerability.name,omitempty"`
	VulnerabilityCVE   string  `json:"vulnerability.id,omitempty"`
	VulnerabilityOWASP string  `json:"vulnerability.category,omitempty"`
	VulnerabilityScore float64 `json:"vulnerability.score,omitempty"`

	// Evidence and context
	Evidence    map[string]string      `json:"strigoi.evidence,omitempty"`
	Remediation string                 `json:"strigoi.remediation,omitempty"`
	FrameData   map[string]interface{} `json:"strigoi.frame,omitempty"`

	// User and process info
	UserName    string `json:"user.name,omitempty"`
	UserID      string `json:"user.id,omitempty"`
	ProcessName string `json:"process.name,omitempty"`
	ProcessPID  int    `json:"process.pid,omitempty"`

	// File information
	FileName string `json:"file.name,omitempty"`
	FilePath string `json:"file.path,omitempty"`
	FileHash string `json:"file.hash.sha256,omitempty"`

	// HTTP specific
	HTTPMethod     string `json:"http.request.method,omitempty"`
	HTTPURL        string `json:"url.full,omitempty"`
	HTTPStatusCode int    `json:"http.response.status_code,omitempty"`
	HTTPUserAgent  string `json:"user_agent.original,omitempty"`

	// DNS specific
	DNSQuery   string   `json:"dns.question.name,omitempty"`
	DNSType    string   `json:"dns.question.type,omitempty"`
	DNSAnswers []string `json:"dns.answers,omitempty"`

	// TLS specific
	TLSVersion string `json:"tls.version,omitempty"`
	TLSCipher  string `json:"tls.cipher,omitempty"`
	TLSSubject string `json:"tls.server.subject,omitempty"`

	// Tags and labels
	Tags   []string          `json:"tags,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`

	// Masked/redacted fields tracking
	MaskedFields []string `json:"strigoi.masked_fields,omitempty"`
}

// EventConverter converts Strigoi events to SIEM format
type EventConverter struct {
	config        ConverterConfig
	maskingEngine *DataMaskingEngine
}

// ConverterConfig holds conversion settings
type ConverterConfig struct {
	IncludeRawFrame    bool
	MaskSensitiveData  bool
	EnrichWithGeoIP    bool
	EnrichWithThreat   bool
	CustomFieldMapping map[string]string
}

// NewEventConverter creates a new event converter
func NewEventConverter(config ConverterConfig) *EventConverter {
	return &EventConverter{
		config:        config,
		maskingEngine: NewDataMaskingEngine(),
	}
}

// ConvertFrame converts a Strigoi frame to a security event
func (c *EventConverter) ConvertFrame(frame *probe.Frame, sessionID string, vulns []*probe.Vulnerability) []*SecurityEvent {
	var events []*SecurityEvent

	// Create base event from frame
	baseEvent := &SecurityEvent{
		Timestamp:       time.Now(), // Frame doesn't have timestamp
		SessionID:       sessionID,
		Protocol:        frame.Protocol,
		StrigoiVersion:  "1.0.0",
		EventCategory:   []string{"network", "intrusion_detection"},
		NetworkProtocol: frame.Protocol,
		Tags:            []string{"strigoi", frame.Protocol},
		Labels:          make(map[string]string),
	}

	// Extract network information
	c.extractNetworkInfo(frame, baseEvent)

	// Extract protocol-specific fields
	c.extractProtocolFields(frame, baseEvent)

	// Mask sensitive data if configured
	if c.config.MaskSensitiveData {
		c.maskSensitiveFields(frame, baseEvent)
	}

	// Convert vulnerabilities to events
	for _, vuln := range vulns {
		event := c.createVulnerabilityEvent(baseEvent, vuln)
		events = append(events, event)
	}

	// If no vulnerabilities, create an observation event
	if len(vulns) == 0 {
		event := c.createObservationEvent(baseEvent, frame)
		events = append(events, event)
	}

	return events
}

// createVulnerabilityEvent creates an event for a detected vulnerability
func (c *EventConverter) createVulnerabilityEvent(base *SecurityEvent, vuln *probe.Vulnerability) *SecurityEvent {
	event := *base // Copy base event

	event.EventType = "vulnerability_detected"
	event.EventCategory = append(event.EventCategory, "vulnerability")
	event.EventSeverity = c.severityToECS(vuln.Severity)
	event.EventRisk = fmt.Sprintf("%d", event.EventSeverity*20) // Simple risk score

	event.Message = fmt.Sprintf("Vulnerability detected: %s - %s",
		vuln.Package, vuln.Description)

	event.VulnerabilityType = vuln.Package
	event.VulnerabilityName = vuln.Description
	event.VulnerabilityCVE = vuln.CVE
	event.VulnerabilityOWASP = "" // Not in supply chain vuln
	event.VulnerabilityScore = vuln.CVSSScore

	event.Evidence = map[string]string{
		"package":    vuln.Package,
		"version":    vuln.Version,
		"cwe":        vuln.CWE,
		"confidence": vuln.Confidence,
	}
	event.Remediation = vuln.Remediation

	// Add vulnerability-specific tags
	event.Tags = append(event.Tags,
		fmt.Sprintf("severity:%s", c.severityToString(vuln.Severity)),
		fmt.Sprintf("vuln:%s", vuln.Package),
	)

	if vuln.CVE != "" {
		event.Tags = append(event.Tags, fmt.Sprintf("cve:%s", vuln.CVE))
	}

	return &event
}

// createObservationEvent creates an event for normal traffic observation
func (c *EventConverter) createObservationEvent(base *SecurityEvent, frame *probe.Frame) *SecurityEvent {
	event := *base // Copy base event

	event.EventType = "protocol_observation"
	event.EventSeverity = 0 // Info level
	event.Message = fmt.Sprintf("Protocol traffic observed: %s", frame.Protocol)

	// Include frame data if configured
	if c.config.IncludeRawFrame {
		event.FrameData = c.frameToMap(frame)
	}

	return &event
}

// extractNetworkInfo extracts network-level information
func (c *EventConverter) extractNetworkInfo(frame *probe.Frame, event *SecurityEvent) {
	// Extract from frame fields
	if srcIP, ok := frame.Fields["source_ip"]; ok {
		if ip, ok := srcIP.(string); ok {
			event.SourceIP = ip
		}
	}
	if srcPort, ok := frame.Fields["source_port"]; ok {
		switch v := srcPort.(type) {
		case int:
			event.SourcePort = v
		case int64:
			event.SourcePort = int(v)
		case float64:
			event.SourcePort = int(v)
		}
	}
	if dstIP, ok := frame.Fields["destination_ip"]; ok {
		if ip, ok := dstIP.(string); ok {
			event.DestinationIP = ip
		}
	}
	if dstPort, ok := frame.Fields["destination_port"]; ok {
		switch v := dstPort.(type) {
		case int:
			event.DestinationPort = v
		case int64:
			event.DestinationPort = int(v)
		case float64:
			event.DestinationPort = int(v)
		}
	}

	event.NetworkBytes = int64(len(frame.Raw))
}

// extractProtocolFields extracts protocol-specific fields
func (c *EventConverter) extractProtocolFields(frame *probe.Frame, event *SecurityEvent) {
	switch frame.Protocol {
	case "HTTP":
		c.extractHTTPFields(frame, event)
	case "DNS":
		c.extractDNSFields(frame, event)
	case "TLS":
		c.extractTLSFields(frame, event)
	}
}

// extractHTTPFields extracts HTTP-specific fields
func (c *EventConverter) extractHTTPFields(frame *probe.Frame, event *SecurityEvent) {
	if method, ok := frame.Fields["method"]; ok {
		if m, ok := method.(string); ok {
			event.HTTPMethod = m
		}
	}
	if uri, ok := frame.Fields["uri"]; ok {
		if u, ok := uri.(string); ok {
			event.HTTPURL = u
		}
	}
	if status, ok := frame.Fields["status_code"]; ok {
		switch v := status.(type) {
		case string:
			if _, err := fmt.Sscanf(v, "%d", &event.HTTPStatusCode); err != nil {
				// Invalid status code format, leave as default
			}
		case int:
			event.HTTPStatusCode = v
		case float64:
			event.HTTPStatusCode = int(v)
		}
	}
	if ua, ok := frame.Fields["user-agent"]; ok {
		if u, ok := ua.(string); ok {
			event.HTTPUserAgent = u
		}
	}
}

// extractDNSFields extracts DNS-specific fields
func (c *EventConverter) extractDNSFields(frame *probe.Frame, event *SecurityEvent) {
	if query, ok := frame.Fields["query"]; ok {
		if q, ok := query.(string); ok {
			event.DNSQuery = q
		}
	}
	if qtype, ok := frame.Fields["query_type"]; ok {
		if qt, ok := qtype.(string); ok {
			event.DNSType = qt
		}
	}
	if answers, ok := frame.Fields["answers"]; ok {
		if answerList, ok := answers.([]string); ok {
			event.DNSAnswers = answerList
		}
	}
}

// extractTLSFields extracts TLS-specific fields
func (c *EventConverter) extractTLSFields(frame *probe.Frame, event *SecurityEvent) {
	if version, ok := frame.Fields["tls_version"]; ok {
		if v, ok := version.(string); ok {
			event.TLSVersion = v
		}
	}
	if cipher, ok := frame.Fields["cipher_suite"]; ok {
		if c, ok := cipher.(string); ok {
			event.TLSCipher = c
		}
	}
	if subject, ok := frame.Fields["certificate_subject"]; ok {
		if s, ok := subject.(string); ok {
			event.TLSSubject = s
		}
	}
}

// maskSensitiveFields applies data masking to sensitive fields
func (c *EventConverter) maskSensitiveFields(frame *probe.Frame, event *SecurityEvent) {
	maskedFields := []string{}

	// Check for common sensitive field names
	sensitiveFieldNames := []string{"password", "auth_token", "api_key", "email", "credit_card", "ssn"}

	for name, value := range frame.Fields {
		isSensitive := false
		for _, sensitive := range sensitiveFieldNames {
			if strings.Contains(strings.ToLower(name), sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			// Mask the value in the event
			switch name {
			case "password", "auth_token", "api_key":
				// Completely redact these
				frame.Fields[name] = "[REDACTED]"
				maskedFields = append(maskedFields, name)
			case "email":
				// Partially mask email
				if email, ok := value.(string); ok {
					frame.Fields[name] = c.maskingEngine.MaskEmail(email)
					maskedFields = append(maskedFields, name)
				}
			case "credit_card":
				// Mask credit card
				if cc, ok := value.(string); ok {
					frame.Fields[name] = c.maskingEngine.MaskCreditCard(cc)
					maskedFields = append(maskedFields, name)
				}
			default:
				// Generic masking
				frame.Fields[name] = c.maskingEngine.MaskGeneric(fmt.Sprintf("%v", value))
				maskedFields = append(maskedFields, name)
			}
		}
	}

	if len(maskedFields) > 0 {
		event.MaskedFields = maskedFields
		event.Tags = append(event.Tags, "masked_data")
	}
}

// frameToMap converts frame fields to a map
func (c *EventConverter) frameToMap(frame *probe.Frame) map[string]interface{} {
	result := make(map[string]interface{})

	// Check for common sensitive field names when masking is enabled
	sensitiveFieldNames := []string{"password", "auth_token", "api_key", "email", "credit_card", "ssn"}

	for name, value := range frame.Fields {
		if c.config.MaskSensitiveData {
			isSensitive := false
			for _, sensitive := range sensitiveFieldNames {
				if strings.Contains(strings.ToLower(name), sensitive) {
					isSensitive = true
					break
				}
			}
			if isSensitive {
				result[name] = "[MASKED]"
			} else {
				result[name] = value
			}
		} else {
			result[name] = value
		}
	}

	return result
}

// severityToECS converts Strigoi severity to ECS severity
func (c *EventConverter) severityToECS(severity string) int {
	switch severity {
	case "critical":
		return 100
	case "high":
		return 73
	case "medium":
		return 47
	case "low":
		return 21
	case "info":
		return 0
	default:
		return 0
	}
}

// severityToString converts severity to string
func (c *EventConverter) severityToString(severity string) string {
	// Already a string, just return it
	return severity
}

// calculateCVSS estimates a CVSS score based on vulnerability
func (c *EventConverter) calculateCVSS(vuln *probe.Vulnerability) float64 {
	// Simplified CVSS calculation
	baseScore := 0.0

	switch vuln.Severity {
	case "critical":
		baseScore = 9.0
	case "high":
		baseScore = 7.5
	case "medium":
		baseScore = 5.0
	case "low":
		baseScore = 3.0
	case "info":
		baseScore = 0.0
	}

	// Adjust based on vulnerability CWE
	if vuln.CWE != "" {
		switch {
		case strings.Contains(vuln.CWE, "CWE-89"), strings.Contains(vuln.CWE, "CWE-94"):
			baseScore += 1.0 // Injection
		case strings.Contains(vuln.CWE, "CWE-287"):
			baseScore += 0.8 // Authentication
		case strings.Contains(vuln.CWE, "CWE-200"):
			baseScore += 0.5 // Information disclosure
		}
	}

	// Cap at 10.0
	if baseScore > 10.0 {
		baseScore = 10.0
	}

	return baseScore
}

// BatchProcessor handles batching of events for efficient sending
type BatchProcessor struct {
	integration Integration
	config      Config
	eventChan   chan *SecurityEvent
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(integration Integration, config Config) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	bp := &BatchProcessor{
		integration: integration,
		config:      config,
		eventChan:   make(chan *SecurityEvent, config.BatchSize*2),
		ctx:         ctx,
		cancel:      cancel,
	}

	go bp.processLoop()

	return bp
}

// Send queues an event for sending
func (bp *BatchProcessor) Send(event *SecurityEvent) error {
	select {
	case bp.eventChan <- event:
		return nil
	case <-bp.ctx.Done():
		return fmt.Errorf("batch processor stopped")
	default:
		return fmt.Errorf("event queue full")
	}
}

// processLoop handles batching and sending
func (bp *BatchProcessor) processLoop() {
	batch := make([]*SecurityEvent, 0, bp.config.BatchSize)
	ticker := time.NewTicker(bp.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case event := <-bp.eventChan:
			batch = append(batch, event)
			if len(batch) >= bp.config.BatchSize {
				bp.flushBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				bp.flushBatch(batch)
				batch = batch[:0]
			}

		case <-bp.ctx.Done():
			// Flush remaining events
			if len(batch) > 0 {
				bp.flushBatch(batch)
			}
			return
		}
	}
}

// flushBatch sends a batch of events
func (bp *BatchProcessor) flushBatch(batch []*SecurityEvent) {
	if err := bp.integration.SendBatch(batch); err != nil {
		// Log error - in production, implement retry logic
		fmt.Printf("Failed to send batch: %v\n", err)
	}
}

// Close stops the batch processor
func (bp *BatchProcessor) Close() error {
	bp.cancel()
	close(bp.eventChan)
	return nil
}
