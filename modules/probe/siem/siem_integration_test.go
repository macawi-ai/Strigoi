package siem

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/macawi-ai/strigoi/modules/probe"
)

func TestEventConverter_ConvertFrame(t *testing.T) {
	converter := NewEventConverter(ConverterConfig{
		MaskSensitiveData: true,
		IncludeRawFrame:   true,
	})

	// Create test frame
	frame := &probe.Frame{
		Protocol: "HTTP",
		Fields: map[string]interface{}{
			"method":    "POST",
			"uri":       "/api/login",
			"password":  "secret123",
			"source_ip": "192.168.1.100",
		},
		Raw: []byte("POST /api/login HTTP/1.1\r\n..."),
	}

	// Create test vulnerabilities
	vulns := []*probe.Vulnerability{
		{
			CVE:         "",
			CWE:         "CWE-521",
			Package:     "login-module",
			Version:     "1.0.0",
			Severity:    "high",
			CVSSScore:   7.5,
			Description: "Weak password detected in login request",
			Remediation: "Use strong passwords with minimum 12 characters",
			Confidence:  "high",
			References:  []string{"OWASP A07:2021"},
		},
	}

	// Convert
	events := converter.ConvertFrame(frame, "session-123", vulns)

	// Verify
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]

	// Check basic fields
	if event.Protocol != "HTTP" {
		t.Errorf("Protocol = %s, want HTTP", event.Protocol)
	}

	if event.SessionID != "session-123" {
		t.Errorf("SessionID = %s, want session-123", event.SessionID)
	}

	// Check vulnerability fields
	if event.VulnerabilityType != "login-module" {
		t.Errorf("VulnerabilityType = %s, want login-module", event.VulnerabilityType)
	}

	// Check masking
	if len(event.MaskedFields) == 0 {
		t.Error("Expected masked fields")
	}

	// Check HTTP fields
	if event.HTTPMethod != "POST" {
		t.Errorf("HTTPMethod = %s, want POST", event.HTTPMethod)
	}
}

func TestDataMaskingEngine(t *testing.T) {
	engine := NewDataMaskingEngine()

	tests := []struct {
		name     string
		input    string
		expected string
		contains string
	}{
		{
			name:     "Credit card",
			input:    "Payment with card 4111111111111111",
			contains: "4111****1111",
		},
		{
			name:     "SSN",
			input:    "SSN: 123-45-6789",
			contains: "***-**-6789",
		},
		{
			name:     "Email",
			input:    "Contact: john.doe@example.com",
			contains: "jo***@example.com",
		},
		{
			name:     "API key",
			input:    "sk_test_4eC39HqLyjWDarjtT1zdp7dc",
			contains: "sk_***p7dc",
		},
		{
			name:     "Multiple sensitive items",
			input:    "User john@example.com with SSN 123-45-6789 paid with 4111111111111111",
			contains: "jo***@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.MaskString(tt.input)
			if !contains(result, tt.contains) {
				t.Errorf("MaskString() = %s, want to contain %s", result, tt.contains)
			}
		})
	}
}

func TestMaskingAnalyzer(t *testing.T) {
	analyzer := NewMaskingAnalyzer()

	fields := map[string]interface{}{
		"username":    "john_doe",
		"email":       "john@example.com",
		"credit_card": "4111-1111-1111-1111",
		"api_key":     "sk_live_abcdef123456789",
		"safe_field":  "This is safe data",
		"password":    "secret123", // Sensitive field name
	}

	masked, stats := analyzer.AnalyzeAndMask(fields)

	// Check stats
	if stats.TotalFields != 6 {
		t.Errorf("TotalFields = %d, want 6", stats.TotalFields)
	}

	if stats.MaskedFields < 4 {
		t.Errorf("MaskedFields = %d, want at least 4", stats.MaskedFields)
	}

	// Check specific masked values
	if email, ok := masked["email"].(string); ok {
		if email == "john@example.com" {
			t.Error("Email was not masked")
		}
	}

	if cc, ok := masked["credit_card"].(string); ok {
		if contains(cc, "1111-1111") {
			t.Error("Credit card was not properly masked")
		}
	}

	// Safe field should not be masked
	if safe, ok := masked["safe_field"].(string); ok {
		if safe != "This is safe data" {
			t.Error("Safe field was incorrectly masked")
		}
	}
}

func TestJSONSchema_Validation(t *testing.T) {
	// Create a valid event
	event := &SecurityEvent{
		Timestamp:         time.Now(),
		Message:           "Test vulnerability detected",
		EventType:         "vulnerability_detected",
		EventCategory:     []string{"network", "vulnerability"},
		EventSeverity:     73,
		StrigoiVersion:    "1.0.0",
		SessionID:         "test-session",
		Protocol:          "HTTP",
		SourceIP:          "192.168.1.100",
		SourcePort:        54321,
		DestinationIP:     "10.0.0.1",
		DestinationPort:   80,
		VulnerabilityType: "SQL_INJECTION",
		VulnerabilityCVE:  "CVE-2021-1234",
	}

	// Marshal to JSON
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Basic validation - ensure required fields are present
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	// Check required fields (using actual JSON tags)
	requiredFields := []string{"@timestamp", "message", "strigoi.version", "strigoi.session_id"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}
}

func TestSplunkEventConversion(t *testing.T) {
	integration := NewSplunkIntegration()
	integration.sourcetype = "strigoi:security"

	event := &SecurityEvent{
		Timestamp:         time.Now(),
		Message:           "SQL injection detected",
		EventType:         "vulnerability_detected",
		EventSeverity:     90,
		SourceIP:          "192.168.1.100",
		HTTPMethod:        "POST",
		HTTPURL:           "/api/users",
		VulnerabilityType: "login-module",
	}

	splunkEvent := integration.toSplunkEvent(event)

	// Check conversion
	if splunkEvent.Sourcetype != "strigoi:security" {
		t.Errorf("Sourcetype = %s, want strigoi:security", splunkEvent.Sourcetype)
	}

	if splunkEvent.Source != "strigoi" {
		t.Errorf("Source = %s, want strigoi", splunkEvent.Source)
	}

	// Check event data
	eventData := splunkEvent.Event
	if eventData["severity"] != 90 {
		t.Errorf("Severity = %v, want 90", eventData["severity"])
	}

	if eventData["src_ip"] != "192.168.1.100" {
		t.Errorf("src_ip = %v, want 192.168.1.100", eventData["src_ip"])
	}
}

func TestQRadarLEEFFormat(t *testing.T) {
	event := &SecurityEvent{
		Timestamp:         time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		EventType:         "vulnerability_detected",
		EventSeverity:     80,
		EventCategory:     []string{"network", "vulnerability"},
		SourceIP:          "192.168.1.100",
		DestinationIP:     "10.0.0.1",
		SourcePort:        12345,
		DestinationPort:   80,
		NetworkProtocol:   "TCP",
		Message:           "Test message with | special = characters",
		VulnerabilityType: "xss-module",
		VulnerabilityCVE:  "CVE-2024-0001",
	}

	leef := QRadarLEEFFormat(event)

	// Check format
	if !contains(leef, "LEEF:2.0|Strigoi|SecurityMonitor|1.0|vulnerability_detected|") {
		t.Error("Invalid LEEF header")
	}

	// Check escaping
	if !contains(leef, "msg=Test message with \\| special \\= characters") {
		t.Error("Special characters not properly escaped")
	}

	// Check attributes
	expectedAttrs := []string{
		"severity=8",
		"src=192.168.1.100",
		"dst=10.0.0.1",
		"vulnType=xss-module",
		"cve=CVE-2024-0001",
	}

	for _, attr := range expectedAttrs {
		if !contains(leef, attr) {
			t.Errorf("Missing attribute: %s", attr)
		}
	}
}

func TestArcSightCEFFormat(t *testing.T) {
	event := &SecurityEvent{
		Timestamp:          time.Now(),
		EventType:          "injection_attempt",
		EventSeverity:      95,
		Message:            "SQL injection in login form",
		SourceIP:           "192.168.1.100",
		VulnerabilityType:  "sql-injection-module",
		VulnerabilityScore: 8.5,
	}

	cef := ArcSightCEFFormat(event)

	// Check format
	if !contains(cef, "CEF:0|Strigoi|SecurityMonitor|1.0|injection_attempt|") {
		t.Error("Invalid CEF header")
	}

	// Check severity mapping (95 -> 9)
	if !contains(cef, "|9|") {
		t.Error("Severity not properly mapped")
	}

	// Check custom fields
	if !contains(cef, "cs1=sql-injection-module") {
		t.Error("Missing vulnerability type")
	}

	if !contains(cef, "cfp1=8.5") {
		t.Error("Missing CVSS score")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
