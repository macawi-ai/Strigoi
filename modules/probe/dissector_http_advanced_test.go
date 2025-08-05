package probe

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestHTTPDissector_Methods tests various HTTP methods.
func TestHTTPDissector_Methods(t *testing.T) {
	dissector := NewHTTPDissector()
	methods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH", "CONNECT", "TRACE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			data := fmt.Sprintf(`%s /api/endpoint HTTP/1.1
Host: example.com
X-API-Key: secret-key-123456789012345678

`, method)
			found, conf := dissector.Identify([]byte(data))
			if !found {
				t.Errorf("Failed to identify %s request", method)
			}
			if conf < 0.9 {
				t.Errorf("Low confidence %v for %s request", conf, method)
			}

			// Check vulnerability detection still works
			frame, _ := dissector.Dissect([]byte(data))
			vulns := dissector.FindVulnerabilities(frame)
			if len(vulns) < 1 {
				t.Errorf("Should find API key vulnerability in %s request", method)
			}
		})
	}
}

// TestHTTPDissector_Versions tests different HTTP versions.
func TestHTTPDissector_Versions(t *testing.T) {
	dissector := NewHTTPDissector()
	tests := []struct {
		name    string
		data    string
		version string
	}{
		{
			name: "HTTP/1.0",
			data: `GET /path HTTP/1.0
Host: example.com

`,
			version: "HTTP/1.0",
		},
		{
			name: "HTTP/1.1",
			data: `GET /path HTTP/1.1
Host: example.com

`,
			version: "HTTP/1.1",
		},
		{
			name: "HTTP/2.0",
			data: `GET /path HTTP/2.0
Host: example.com

`,
			version: "HTTP/2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, _ := dissector.Identify([]byte(tt.data))
			if !found {
				t.Errorf("Failed to identify %s", tt.name)
			}
		})
	}
}

// TestHTTPDissector_HeaderVariations tests header edge cases.
func TestHTTPDissector_HeaderVariations(t *testing.T) {
	dissector := NewHTTPDissector()

	tests := []struct {
		name        string
		data        string
		expectVulns int
	}{
		{
			name: "Mixed case headers",
			data: `GET /api HTTP/1.1
host: example.com
AUTHORIZATION: Bearer secret-token-123456789012345678
x-api-KEY: another-secret-key-098765432109876543
Content-TYPE: application/json

`,
			expectVulns: 2,
		},
		{
			name: "Duplicate headers",
			data: `GET /api HTTP/1.1
Host: example.com
X-API-Key: first-key-12345678901234567890
X-API-Key: second-key-09876543210987654321

`,
			expectVulns: 1, // Should detect at least one
		},
		{
			name: "Headers with whitespace",
			data: `GET /api HTTP/1.1
Host:    example.com   
Authorization:   Bearer    token-with-spaces-1234567890  
  X-Custom-Header: value

`,
			expectVulns: 1,
		},
		{
			name: "Folded headers (deprecated but should handle)",
			data: `GET /api HTTP/1.1
Host: example.com
Authorization: Bearer
 very-long-token-that-was-folded-1234567890

`,
			expectVulns: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := dissector.Dissect([]byte(tt.data))
			if err != nil {
				t.Fatalf("Dissect() error = %v", err)
			}
			vulns := dissector.FindVulnerabilities(frame)
			if len(vulns) < tt.expectVulns {
				t.Errorf("Found %d vulnerabilities, expected at least %d", len(vulns), tt.expectVulns)
			}
		})
	}
}

// TestHTTPDissector_ChunkedEncoding tests chunked transfer encoding.
func TestHTTPDissector_ChunkedEncoding(t *testing.T) {
	dissector := NewHTTPDissector()

	tests := []struct {
		name string
		data string
	}{
		{
			name: "Simple chunked response",
			data: `HTTP/1.1 200 OK
Transfer-Encoding: chunked
Content-Type: application/json

1e
{"api_key": "secret-key-12345"}
0

`,
		},
		{
			name: "Multiple chunks with extensions",
			data: `HTTP/1.1 200 OK
Transfer-Encoding: chunked

5;name=value
Hello
a
 World!
0

`,
		},
		{
			name: "Malformed chunks",
			data: `HTTP/1.1 200 OK
Transfer-Encoding: chunked

GGGG
Invalid hex
0

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Should not panic on any input
			frame, _ := dissector.Dissect([]byte(tt.data))
			if frame != nil {
				_ = dissector.FindVulnerabilities(frame)
			}
		})
	}
}

// TestHTTPDissector_Multipart tests multipart/form-data.
func TestHTTPDissector_Multipart(t *testing.T) {
	dissector := NewHTTPDissector()

	boundary := "----WebKitFormBoundary7MA4YWxkTrZu0gW"
	data := fmt.Sprintf(`POST /upload HTTP/1.1
Host: example.com
Content-Type: multipart/form-data; boundary=%s
Content-Length: 248

--%s
Content-Disposition: form-data; name="api_key"

sk-secret-api-key-1234567890abcdef
--%s
Content-Disposition: form-data; name="password"

SuperSecret123!
--%s
Content-Disposition: form-data; name="file"; filename="test.txt"
Content-Type: text/plain

File contents with token=auth-token-xyz789
--%s--`, boundary, boundary, boundary, boundary, boundary)

	frame, err := dissector.Dissect([]byte(data))
	if err != nil {
		t.Fatalf("Failed to dissect multipart: %v", err)
	}

	vulns := dissector.FindVulnerabilities(frame)
	// The dissector should find vulnerabilities in the multipart body
	// Even though it doesn't parse multipart structure, it should still detect patterns
	if len(vulns) < 2 {
		t.Errorf("Expected at least 2 vulnerabilities in multipart body, found %d", len(vulns))
		for i, vuln := range vulns {
			t.Logf("Vuln %d: %s - %s at %s", i, vuln.Type, vuln.Subtype, vuln.Location)
		}
	}
}

// TestHTTPDissector_LargeData tests performance with large payloads.
func TestHTTPDissector_LargeData(t *testing.T) {
	dissector := NewHTTPDissector()

	// Create a large JSON body with credentials
	largeBody := `{"data": "` + strings.Repeat("A", 10*1024) + `", "api_key": "secret-key-1234567890", "padding": "` + strings.Repeat("B", 10*1024) + `"}`

	data := fmt.Sprintf(`POST /api/large HTTP/1.1
Host: example.com
Content-Type: application/json
Content-Length: %d

%s`, len(largeBody), largeBody)

	start := time.Now()
	frame, err := dissector.Dissect([]byte(data))
	if err != nil {
		t.Fatalf("Failed to dissect large data: %v", err)
	}

	vulns := dissector.FindVulnerabilities(frame)
	elapsed := time.Since(start)

	// The dissector should detect the API key pattern in the JSON body
	found := false
	for _, vuln := range vulns {
		if vuln.Subtype == "api_key_in_body" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should find API key in large JSON body")
		t.Logf("Found %d vulnerabilities:", len(vulns))
		for _, vuln := range vulns {
			t.Logf("  - %s: %s", vuln.Subtype, vuln.Evidence)
		}
	}

	// Performance check - should complete within reasonable time
	if elapsed > 100*time.Millisecond {
		t.Errorf("Processing large data took too long: %v", elapsed)
	}
}

// TestHTTPDissector_CharacterEncoding tests various encodings.
func TestHTTPDissector_CharacterEncoding(t *testing.T) {
	dissector := NewHTTPDissector()

	tests := []struct {
		name string
		data string
	}{
		{
			name: "UTF-8 with emoji",
			data: `POST /api/🔐 HTTP/1.1
Host: 例え.com
X-Secret-🔑: api-key-with-emoji-🎯-1234567890

{"message": "Hello 世界! 🌍", "token": "bearer-xyz123"}`,
		},
		{
			name: "Latin-1 encoded",
			data: "GET /café HTTP/1.1\r\nHost: résumé.com\r\nX-API-Key: clé-secrète-1234567890\r\n\r\n",
		},
		{
			name: "Mixed encodings",
			data: `GET /api HTTP/1.1
Host: example.com
X-Token-中文: secret-混合-token-1234567890
Cookie: 用户ID=admin; session=xyz123

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should handle without panic
			frame, _ := dissector.Dissect([]byte(tt.data))
			if frame != nil {
				vulns := dissector.FindVulnerabilities(frame)
				// Should still detect vulnerabilities with special characters
				hasVuln := false
				for _, vuln := range vulns {
					if vuln.Type == "credential" {
						hasVuln = true
						break
					}
				}
				if !hasVuln {
					t.Errorf("Should detect vulnerabilities even with special characters, found %d", len(vulns))
				}
			}
		})
	}
}

// TestHTTPDissector_InjectionAttempts tests resistance to injection.
func TestHTTPDissector_InjectionAttempts(t *testing.T) {
	dissector := NewHTTPDissector()

	tests := []struct {
		name string
		data string
	}{
		{
			name: "Header injection attempt",
			data: `GET /api HTTP/1.1
Host: example.com
X-Injected: value
X-API-Key: key-12345

Another: header
`,
		},
		{
			name: "CRLF injection",
			data: "GET /api HTTP/1.1\r\nHost: example.com\r\nX-Evil: test\r\n\r\nHTTP/1.1 200 OK\r\nX-Injected: true\r\n\r\n",
		},
		{
			name: "SQL injection in headers",
			data: `GET /api HTTP/1.1
Host: example.com
X-User-Id: ' OR '1'='1
Authorization: Bearer token-12345678901234567890

`,
		},
		{
			name: "XSS attempt in body",
			data: `POST /api HTTP/1.1
Host: example.com
Content-Type: application/json

{"user": "<script>alert('xss')</script>", "api_key": "secret-key-1234567890"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic or misbehave
			frame, _ := dissector.Dissect([]byte(tt.data))
			if frame != nil {
				vulns := dissector.FindVulnerabilities(frame)
				// Verify the dissector still works correctly
				for _, vuln := range vulns {
					if vuln.Evidence == "" || vuln.Type == "" {
						t.Error("Vulnerability detection compromised by injection attempt")
					}
				}
			}
		})
	}
}

// TestHTTPDissector_ResponseCodes tests various HTTP response codes.
func TestHTTPDissector_ResponseCodes(t *testing.T) {
	dissector := NewHTTPDissector()

	codes := []struct {
		code int
		text string
	}{
		{200, "OK"},
		{201, "Created"},
		{301, "Moved Permanently"},
		{302, "Found"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{502, "Bad Gateway"},
		{503, "Service Unavailable"},
	}

	for _, tc := range codes {
		t.Run(fmt.Sprintf("HTTP_%d", tc.code), func(t *testing.T) {
			data := fmt.Sprintf(`HTTP/1.1 %d %s
Content-Type: application/json
X-Error-Details: Contains api_key=secret-1234567890 in error

{"error": "%s", "debug_token": "debug-token-xyz789"}`, tc.code, tc.text, tc.text)

			frame, err := dissector.Dissect([]byte(data))
			if err != nil {
				t.Fatalf("Failed to dissect %d response: %v", tc.code, err)
			}

			// Should still detect vulnerabilities in error responses
			vulns := dissector.FindVulnerabilities(frame)
			if len(vulns) < 1 {
				t.Errorf("Should detect at least 1 vulnerability in %d response, found %d", tc.code, len(vulns))
			}
		})
	}
}

// TestHTTPDissector_Concurrent tests thread safety.
func TestHTTPDissector_Concurrent(t *testing.T) {
	dissector := NewHTTPDissector()

	// Run multiple goroutines processing different data
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine processes different data
			data := fmt.Sprintf(`GET /api/user/%d HTTP/1.1
Host: example.com
X-API-Key: key-%d-secret-1234567890
Authorization: Bearer token-%d-abcdefghij

{"user_id": %d, "password": "pass%d"}`, id, id, id, id, id)

			for j := 0; j < 100; j++ {
				frame, err := dissector.Dissect([]byte(data))
				if err != nil {
					errors <- fmt.Errorf("goroutine %d iteration %d: %v", id, j, err)
					return
				}

				vulns := dissector.FindVulnerabilities(frame)
				if len(vulns) < 3 {
					errors <- fmt.Errorf("goroutine %d iteration %d: expected 3 vulns, got %d", id, j, len(vulns))
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent execution error: %v", err)
	}
}

// BenchmarkHTTPDissector_Identify benchmarks identification performance.
func BenchmarkHTTPDissector_Identify(b *testing.B) {
	dissector := NewHTTPDissector()
	data := []byte(`GET /api/v1/users HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
X-API-Key: sk-test-1234567890abcdef

`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dissector.Identify(data)
	}
}

// BenchmarkHTTPDissector_Dissect benchmarks dissection performance.
func BenchmarkHTTPDissector_Dissect(b *testing.B) {
	dissector := NewHTTPDissector()
	data := []byte(`POST /api/login HTTP/1.1
Host: secure.example.com
Content-Type: application/json
Content-Length: 58

{"username": "admin", "password": "secret123", "remember": true}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dissector.Dissect(data)
	}
}

// BenchmarkHTTPDissector_FindVulnerabilities benchmarks vulnerability detection.
func BenchmarkHTTPDissector_FindVulnerabilities(b *testing.B) {
	dissector := NewHTTPDissector()
	data := []byte(`POST /api/data HTTP/1.1
Host: api.example.com
X-API-Key: sk-live-1234567890abcdef
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0
Cookie: sessionId=abc123def456; userId=12345

{"api_token": "tok_1234567890", "password": "MySecretPass123!"}`)

	frame, _ := dissector.Dissect(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dissector.FindVulnerabilities(frame)
	}
}

// TestHTTPDissector_NegativeCases tests cases that should NOT trigger vulnerabilities.
func TestHTTPDissector_NegativeCases(t *testing.T) {
	dissector := NewHTTPDissector()

	tests := []struct {
		name string
		data string
	}{
		{
			name: "Short values that look like keys",
			data: `GET /api HTTP/1.1
Host: example.com
X-API-Version: v2
X-Token-Type: bearer
X-Key-Size: 256

{"api": "public", "key": "value"}`,
		},
		{
			name: "Documentation examples",
			data: `GET /api/docs HTTP/1.1
Host: example.com

Example: Use Authorization: Bearer YOUR_API_KEY_HERE`,
		},
		{
			name: "Hashed/encrypted values",
			data: `POST /api HTTP/1.1
Host: example.com
X-Signature: a7b9c3d2e1f0
Content-MD5: 5d41402abc4b2a76b9719d911017c592

{"checksum": "e99a18c428cb38d5f260853678922e03"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, _ := dissector.Dissect([]byte(tt.data))
			if frame != nil {
				vulns := dissector.FindVulnerabilities(frame)
				if len(vulns) > 0 {
					t.Errorf("False positive: found %d vulnerabilities where none expected", len(vulns))
					for _, vuln := range vulns {
						t.Logf("  - %s: %s", vuln.Subtype, vuln.Evidence)
					}
				}
			}
		})
	}
}
