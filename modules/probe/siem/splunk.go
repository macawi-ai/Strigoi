package siem

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SplunkIntegration implements SIEM integration for Splunk
type SplunkIntegration struct {
	config      Config
	httpClient  *http.Client
	hecEndpoint string // HTTP Event Collector endpoint
	sourcetype  string
}

// NewSplunkIntegration creates a new Splunk integration
func NewSplunkIntegration() *SplunkIntegration {
	return &SplunkIntegration{}
}

// Initialize sets up the Splunk connection
func (s *SplunkIntegration) Initialize(config Config) error {
	s.config = config

	// Parse endpoint to ensure it's properly formatted
	u, err := url.Parse(config.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}

	// Build HEC endpoint
	s.hecEndpoint = fmt.Sprintf("%s://%s/services/collector/event",
		u.Scheme, u.Host)

	// Set sourcetype
	s.sourcetype = "strigoi:security"
	if st, ok := config.Extra["sourcetype"].(string); ok {
		s.sourcetype = st
	}

	// Configure HTTP client
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	}

	// Configure TLS
	if config.TLS.Enabled || u.Scheme == "https" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.TLS.InsecureSkipVerify,
		}

		if config.TLS.ClientCert != "" && config.TLS.ClientKey != "" {
			cert, err := tls.LoadX509KeyPair(config.TLS.ClientCert, config.TLS.ClientKey)
			if err != nil {
				return fmt.Errorf("failed to load client certificates: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		transport.TLSClientConfig = tlsConfig
	}

	s.httpClient = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Test connection
	return s.HealthCheck()
}

// SendEvent sends a single event to Splunk
func (s *SplunkIntegration) SendEvent(event *SecurityEvent) error {
	// Convert to Splunk event format
	splunkEvent := s.toSplunkEvent(event)

	// Marshal to JSON
	data, err := json.Marshal(splunkEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", s.hecEndpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	s.setHeaders(req)

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("splunk returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var hecResp HECResponse
	if err := json.NewDecoder(resp.Body).Decode(&hecResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if hecResp.Code != 0 {
		return fmt.Errorf("splunk HEC error %d: %s", hecResp.Code, hecResp.Text)
	}

	return nil
}

// SendBatch sends multiple events to Splunk
func (s *SplunkIntegration) SendBatch(events []*SecurityEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Build batch payload
	var payload bytes.Buffer

	for _, event := range events {
		splunkEvent := s.toSplunkEvent(event)
		data, err := json.Marshal(splunkEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}
		payload.Write(data)
		payload.WriteString("\n")
	}

	// Create request
	req, err := http.NewRequest("POST", s.hecEndpoint, &payload)
	if err != nil {
		return fmt.Errorf("failed to create batch request: %w", err)
	}

	// Set headers
	s.setHeaders(req)

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send batch request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("splunk returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var hecResp HECResponse
	if err := json.NewDecoder(resp.Body).Decode(&hecResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if hecResp.Code != 0 {
		return fmt.Errorf("splunk HEC error %d: %s", hecResp.Code, hecResp.Text)
	}

	return nil
}

// HealthCheck verifies Splunk connection
func (s *SplunkIntegration) HealthCheck() error {
	// Use HEC health endpoint
	healthURL := strings.Replace(s.hecEndpoint, "/event", "/health", 1)

	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	s.setHeaders(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to splunk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("splunk health check failed with status %d: %s",
			resp.StatusCode, string(body))
	}

	return nil
}

// Close cleans up resources
func (s *SplunkIntegration) Close() error {
	// No persistent connections to close
	return nil
}

// setHeaders sets common headers for Splunk requests
func (s *SplunkIntegration) setHeaders(req *http.Request) {
	// HEC token authentication
	if s.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.config.APIKey))
	}

	req.Header.Set("Content-Type", "application/json")
}

// toSplunkEvent converts SecurityEvent to Splunk HEC format
func (s *SplunkIntegration) toSplunkEvent(event *SecurityEvent) SplunkEvent {
	// Build event data with Splunk-friendly field names
	eventData := map[string]interface{}{
		// Core fields
		"message":          event.Message,
		"severity":         event.EventSeverity,
		"risk_score":       event.EventRisk,
		"event_type":       event.EventType,
		"event_categories": event.EventCategory,

		// Strigoi fields
		"strigoi_version": event.StrigoiVersion,
		"session_id":      event.SessionID,
		"protocol":        event.Protocol,
		"dissector":       event.Dissector,

		// Network fields
		"src_ip":           event.SourceIP,
		"src_port":         event.SourcePort,
		"dest_ip":          event.DestinationIP,
		"dest_port":        event.DestinationPort,
		"network_protocol": event.NetworkProtocol,
		"bytes":            event.NetworkBytes,

		// Vulnerability fields
		"vuln_type":   event.VulnerabilityType,
		"vuln_name":   event.VulnerabilityName,
		"vuln_cve":    event.VulnerabilityCVE,
		"vuln_owasp":  event.VulnerabilityOWASP,
		"vuln_score":  event.VulnerabilityScore,
		"remediation": event.Remediation,
	}

	// Add evidence if present
	if len(event.Evidence) > 0 {
		eventData["evidence"] = event.Evidence
	}

	// Add HTTP fields if present
	if event.HTTPMethod != "" {
		eventData["http_method"] = event.HTTPMethod
		eventData["url"] = event.HTTPURL
		eventData["status_code"] = event.HTTPStatusCode
		eventData["user_agent"] = event.HTTPUserAgent
	}

	// Add DNS fields if present
	if event.DNSQuery != "" {
		eventData["dns_query"] = event.DNSQuery
		eventData["dns_type"] = event.DNSType
		eventData["dns_answers"] = event.DNSAnswers
	}

	// Add TLS fields if present
	if event.TLSVersion != "" {
		eventData["tls_version"] = event.TLSVersion
		eventData["tls_cipher"] = event.TLSCipher
		eventData["tls_subject"] = event.TLSSubject
	}

	// Add tags
	if len(event.Tags) > 0 {
		eventData["tags"] = strings.Join(event.Tags, ",")
	}

	// Add masked fields info
	if len(event.MaskedFields) > 0 {
		eventData["masked_fields"] = event.MaskedFields
	}

	// Build Splunk event
	splunkEvent := SplunkEvent{
		Time:       event.Timestamp.Unix(),
		Host:       s.getHostname(),
		Source:     "strigoi",
		Sourcetype: s.sourcetype,
		Index:      s.config.Index,
		Event:      eventData,
	}

	return splunkEvent
}

// getHostname gets the hostname for events
func (s *SplunkIntegration) getHostname() string {
	if host, ok := s.config.Extra["hostname"].(string); ok {
		return host
	}
	return "strigoi-probe"
}

// Splunk-specific structures

// SplunkEvent represents a Splunk HEC event
type SplunkEvent struct {
	Time       int64                  `json:"time"`
	Host       string                 `json:"host"`
	Source     string                 `json:"source"`
	Sourcetype string                 `json:"sourcetype"`
	Index      string                 `json:"index,omitempty"`
	Event      map[string]interface{} `json:"event"`
}

// HECResponse represents Splunk HEC response
type HECResponse struct {
	Text string `json:"text"`
	Code int    `json:"code"`
}

// CreateSplunkApp creates a Splunk app configuration for Strigoi
func CreateSplunkApp() map[string]interface{} {
	return map[string]interface{}{
		"app": map[string]interface{}{
			"name":        "strigoi_security",
			"label":       "Strigoi Security Monitoring",
			"version":     "1.0.0",
			"description": "Real-time security monitoring and vulnerability detection with Strigoi",
		},
		"props.conf": map[string]string{
			"[strigoi:security]": `
SHOULD_LINEMERGE = false
KV_MODE = json
TIME_PREFIX = "time":
MAX_TIMESTAMP_LOOKAHEAD = 20
TRUNCATE = 0

# Field extractions
EXTRACT-strigoi = (?<strigoi_fields>.*)

# Field aliases for CIM compliance
FIELDALIAS-src = src_ip AS src
FIELDALIAS-dest = dest_ip AS dest
FIELDALIAS-dvc = host AS dvc
FIELDALIAS-severity = severity AS severity_id
`,
		},
		"eventtypes.conf": map[string]string{
			"[strigoi_vulnerability]": `
search = sourcetype=strigoi:security vuln_type=*
`,
			"[strigoi_high_severity]": `
search = sourcetype=strigoi:security severity>=73
`,
			"[strigoi_injection]": `
search = sourcetype=strigoi:security vuln_type=*INJECTION*
`,
		},
		"tags.conf": map[string]string{
			"[eventtype=strigoi_vulnerability]": `
attack = enabled
vulnerability = enabled
`,
			"[eventtype=strigoi_injection]": `
attack = enabled
injection = enabled
`,
		},
		"savedsearches.conf": map[string]string{
			"[Strigoi - Critical Vulnerabilities]": `
search = sourcetype=strigoi:security severity>=90 | stats count by vuln_type, dest_ip | sort -count
dispatch.earliest_time = -24h@h
dispatch.latest_time = now
display.visualizations.charting.chart = bar
`,
			"[Strigoi - Top Attack Patterns]": `
search = sourcetype=strigoi:security vuln_type=* | timechart span=1h count by vuln_type
dispatch.earliest_time = -7d@d
dispatch.latest_time = now
display.visualizations.charting.chart = line
`,
			"[Strigoi - Vulnerable Services]": `
search = sourcetype=strigoi:security | stats dc(vuln_type) as vuln_count, values(vuln_type) as vulnerabilities by dest_ip, dest_port, protocol | sort -vuln_count
dispatch.earliest_time = -24h@h
dispatch.latest_time = now
`,
		},
		"macros.conf": map[string]string{
			"[strigoi_critical]": `
definition = severity>=90
`,
			"[strigoi_injection_attacks]": `
definition = vuln_type="*INJECTION*" OR vuln_type="*XSS*"
`,
		},
		"data/ui/views/strigoi_overview.xml": `
<dashboard>
  <label>Strigoi Security Overview</label>
  <row>
    <panel>
      <title>Vulnerability Trend (24h)</title>
      <chart>
        <search>
          <query>sourcetype=strigoi:security | timechart span=1h count by severity</query>
          <earliest>-24h@h</earliest>
          <latest>now</latest>
        </search>
        <option name="charting.chart">area</option>
        <option name="charting.axisTitleX.text">Time</option>
        <option name="charting.axisTitleY.text">Count</option>
      </chart>
    </panel>
    <panel>
      <title>Top Vulnerability Types</title>
      <chart>
        <search>
          <query>sourcetype=strigoi:security vuln_type=* | top vuln_type</query>
          <earliest>-24h@h</earliest>
          <latest>now</latest>
        </search>
        <option name="charting.chart">pie</option>
      </chart>
    </panel>
  </row>
  <row>
    <panel>
      <title>Recent Critical Vulnerabilities</title>
      <table>
        <search>
          <query>sourcetype=strigoi:security severity>=90 | table _time, dest_ip, vuln_type, vuln_name, remediation | sort -_time</query>
          <earliest>-24h@h</earliest>
          <latest>now</latest>
        </search>
        <option name="count">10</option>
        <option name="drilldown">cell</option>
      </table>
    </panel>
  </row>
  <row>
    <panel>
      <title>Network Activity Map</title>
      <map>
        <search>
          <query>sourcetype=strigoi:security | iplocation src_ip | geostats count by vuln_type</query>
          <earliest>-24h@h</earliest>
          <latest>now</latest>
        </search>
      </map>
    </panel>
  </row>
</dashboard>
`,
	}
}
