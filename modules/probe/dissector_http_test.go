package probe

import (
	"strings"
	"testing"
)

func TestHTTPDissector_Identify(t *testing.T) {
	dissector := NewHTTPDissector()

	tests := []struct {
		name       string
		data       string
		shouldFind bool
		minConf    float64
	}{
		{
			name: "HTTP GET request",
			data: `GET /api/v1/users?token=secret123 HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
Content-Type: application/json

`,
			shouldFind: true,
			minConf:    0.9,
		},
		{
			name: "HTTP POST request",
			data: `POST /login HTTP/1.1
Host: secure.example.com
Content-Type: application/x-www-form-urlencoded
Content-Length: 29

username=admin&password=secret`,
			shouldFind: true,
			minConf:    0.9,
		},
		{
			name: "HTTP response",
			data: `HTTP/1.1 200 OK
Content-Type: application/json
Set-Cookie: sessionId=abc123def456

{"status": "success", "token": "secret-token-xyz"}`,
			shouldFind: true,
			minConf:    0.9,
		},
		{
			name: "HTTP response with status",
			data: `HTTP/1.0 404 Not Found
Server: nginx/1.19.0
Date: Mon, 01 Jan 2024 12:00:00 GMT

Page not found`,
			shouldFind: true,
			minConf:    0.9,
		},
		{
			name: "Partial HTTP headers",
			data: `Content-Type: application/json
Authorization: Bearer token123
X-API-Key: secret-key-456
Accept: */*

Some body content`,
			shouldFind: true,
			minConf:    0.7,
		},
		{
			name:       "Not HTTP - JSON data",
			data:       `{"key": "value", "nested": {"data": "here"}}`,
			shouldFind: false,
		},
		{
			name:       "Not HTTP - Plain text",
			data:       "This is just plain text with no HTTP structure",
			shouldFind: false,
		},
		{
			name:       "Empty data",
			data:       "",
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, conf := dissector.Identify([]byte(tt.data))
			if found != tt.shouldFind {
				t.Errorf("Identify() found = %v, want %v", found, tt.shouldFind)
			}
			if tt.shouldFind && conf < tt.minConf {
				t.Errorf("Identify() confidence = %v, want at least %v", conf, tt.minConf)
			}
		})
	}
}

func TestHTTPDissector_Dissect(t *testing.T) {
	dissector := NewHTTPDissector()

	tests := []struct {
		name          string
		data          string
		expectedType  string
		expectedField string
		expectedValue interface{}
	}{
		{
			name: "HTTP GET request",
			data: `GET /api/v1/users HTTP/1.1
Host: api.example.com
Accept: application/json

`,
			expectedType:  "request",
			expectedField: "method",
			expectedValue: "GET",
		},
		{
			name: "HTTP POST with body",
			data: `POST /api/login HTTP/1.1
Host: api.example.com
Content-Type: application/json
Content-Length: 35

{"username":"admin","password":"x"}`,
			expectedType:  "request",
			expectedField: "body",
			expectedValue: `{"username":"admin","password":"x"}`,
		},
		{
			name: "HTTP response",
			data: `HTTP/1.1 200 OK
Content-Type: text/html
Server: Apache

<html><body>Hello</body></html>`,
			expectedType:  "response",
			expectedField: "status_code",
			expectedValue: 200,
		},
		{
			name: "Partial request (manual parsing)",
			data: `PUT /update HTTP/1.1
Host: example.com`,
			expectedType:  "request",
			expectedField: "method",
			expectedValue: "PUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := dissector.Dissect([]byte(tt.data))
			if err != nil {
				t.Fatalf("Dissect() error = %v", err)
			}
			if frame == nil {
				t.Fatal("Dissect() returned nil frame")
			}
			if frame.Protocol != "HTTP" {
				t.Errorf("Frame protocol = %v, want HTTP", frame.Protocol)
			}
			if frameType, ok := frame.Fields["type"].(string); !ok || frameType != tt.expectedType {
				t.Errorf("Frame type = %v, want %v", frameType, tt.expectedType)
			}
			if val, ok := frame.Fields[tt.expectedField]; !ok || val != tt.expectedValue {
				t.Errorf("Frame field %s = %v, want %v", tt.expectedField, val, tt.expectedValue)
			}
		})
	}
}

func TestHTTPDissector_FindVulnerabilities(t *testing.T) {
	dissector := NewHTTPDissector()

	tests := []struct {
		name          string
		data          string
		minVulns      int
		expectedType  string
		checkEvidence string
	}{
		{
			name: "API key in URL",
			data: `GET /api/v1/users?api_key=sk-test-1234567890abcdef HTTP/1.1
Host: api.example.com

`,
			minVulns:      1,
			expectedType:  "api_key_in_url",
			checkEvidence: "api_key=",
		},
		{
			name: "Bearer token in header",
			data: `GET /api/users HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.R0JhPRo9T3K

`,
			minVulns:      1,
			expectedType:  "bearer_token",
			checkEvidence: "Authorization:",
		},
		{
			name: "Multiple credentials",
			data: `POST /api/login HTTP/1.1
Host: secure.example.com
X-API-Key: secret-api-key-1234567890
Cookie: sessionId=abc123def456789012345; PHPSESSID=xyz789
Content-Type: application/json

{"password": "SuperSecret123!", "api_token": "tok_1234567890abcdef"}`,
			minVulns: 3, // API key, session cookie(s), password - token might not match pattern
		},
		{
			name: "Basic auth in URL",
			data: `GET http://admin:password123@api.example.com/data HTTP/1.1
Host: api.example.com

`,
			minVulns:      1,
			expectedType:  "basic_auth_in_url",
			checkEvidence: "admin:***@",
		},
		{
			name: "No vulnerabilities",
			data: `GET /public/info HTTP/1.1
Host: example.com
Accept: text/html

`,
			minVulns: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First dissect the data
			frame, err := dissector.Dissect([]byte(tt.data))
			if err != nil {
				t.Fatalf("Dissect() error = %v", err)
			}

			// Find vulnerabilities
			vulns := dissector.FindVulnerabilities(frame)

			if len(vulns) < tt.minVulns {
				t.Errorf("FindVulnerabilities() found %d vulns, want at least %d", len(vulns), tt.minVulns)
			}

			// Check specific vulnerability if expected
			if tt.expectedType != "" && len(vulns) > 0 {
				found := false
				for _, vuln := range vulns {
					if vuln.Subtype == tt.expectedType {
						found = true
						if tt.checkEvidence != "" && !strings.Contains(vuln.Evidence, tt.checkEvidence) {
							t.Errorf("Evidence does not contain %q, got %q", tt.checkEvidence, vuln.Evidence)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected vulnerability type %q not found", tt.expectedType)
				}
			}

			// Verify all vulnerabilities have required fields
			for i, vuln := range vulns {
				if vuln.Type == "" {
					t.Errorf("Vulnerability %d missing Type", i)
				}
				if vuln.Subtype == "" {
					t.Errorf("Vulnerability %d missing Subtype", i)
				}
				if vuln.Severity == "" {
					t.Errorf("Vulnerability %d missing Severity", i)
				}
				if vuln.Evidence == "" {
					t.Errorf("Vulnerability %d missing Evidence", i)
				}
				if vuln.Location == "" {
					t.Errorf("Vulnerability %d missing Location", i)
				}
				if vuln.Confidence <= 0 || vuln.Confidence > 1 {
					t.Errorf("Vulnerability %d has invalid confidence: %v", i, vuln.Confidence)
				}
			}
		})
	}
}

func TestHTTPDissector_EdgeCases(t *testing.T) {
	dissector := NewHTTPDissector()

	tests := []struct {
		name string
		data string
	}{
		{
			name: "Malformed HTTP request",
			data: "GET /path\nSomeHeader",
		},
		{
			name: "Binary data",
			data: "\x00\x01\x02\x03\x04\x05",
		},
		{
			name: "Very long header value",
			data: "GET / HTTP/1.1\nX-Long: " + strings.Repeat("A", 10000),
		},
		{
			name: "Unicode in headers",
			data: "GET /用户 HTTP/1.1\nHost: 例え.com\n\n",
		},
		{
			name: "Mixed case methods",
			data: "gEt /path HTTP/1.1\nHost: example.com\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Should not panic
			_, _ = dissector.Identify([]byte(tt.data))
			frame, _ := dissector.Dissect([]byte(tt.data))
			if frame != nil {
				_ = dissector.FindVulnerabilities(frame)
			}
		})
	}
}

func TestParseCookies(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "Single cookie",
			input: "sessionId=abc123",
			expected: map[string]string{
				"sessionId": "abc123",
			},
		},
		{
			name:  "Multiple cookies",
			input: "sessionId=abc123; userId=456; token=xyz",
			expected: map[string]string{
				"sessionId": "abc123",
				"userId":    "456",
				"token":     "xyz",
			},
		},
		{
			name:  "Cookies with spaces",
			input: "session = abc123 ; user = admin",
			expected: map[string]string{
				"session": "abc123",
				"user":    "admin",
			},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:  "Cookie with attributes",
			input: "sessionId=abc123; Path=/; HttpOnly",
			expected: map[string]string{
				"sessionId": "abc123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCookies(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseCookies() returned %d cookies, want %d", len(result), len(tt.expected))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("parseCookies()[%q] = %q, want %q", k, result[k], v)
				}
			}
		})
	}
}

func TestMaskValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"short", "***"},
		{"12345678", "***"},
		{"123456789", "1234****6789"},
		{"verylongsecretvalue", "very****alue"},
	}

	for _, tt := range tests {
		result := maskValue(tt.input)
		if result != tt.expected {
			t.Errorf("maskValue(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMaskAuthValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Bearer abc123def456", "Bearer abc1****f456"},
		{"Basic dXNlcjpwYXNz", "Basic dXNl****YXNz"},
		{"simpletokenvalue", "simp****alue"},
	}

	for _, tt := range tests {
		result := maskAuthValue(tt.input)
		if result != tt.expected {
			t.Errorf("maskAuthValue(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
