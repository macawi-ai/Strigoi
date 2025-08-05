package probe

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestAuthSessionChecker(t *testing.T) {
	checker := NewAuthSessionChecker()

	tests := []struct {
		name              string
		session           *Session
		expectedVulnTypes []string
		minExpectedVulns  int
	}{
		{
			name: "Session fixation detection",
			session: &Session{
				ID:         "test-session-1",
				Protocol:   "HTTP",
				StartTime:  time.Now(),
				LastActive: time.Now(),
				Frames: []*Frame{
					{
						Fields: map[string]interface{}{
							"url": "/login",
							"headers": map[string]string{
								"cookie": "JSESSIONID=ABC123",
							},
							"payload": []byte("username=test&password=secret"),
						},
					},
					{
						Fields: map[string]interface{}{
							"status": 200,
							"headers": map[string]string{
								"set-cookie":    "JSESSIONID=ABC123; HttpOnly",
								"authorization": "Bearer token123",
							},
						},
					},
				},
			},
			expectedVulnTypes: []string{"session_fixation"},
			minExpectedVulns:  1,
		},
		{
			name: "Token reuse detection",
			session: &Session{
				ID:         "test-session-2",
				Protocol:   "HTTP",
				StartTime:  time.Now(),
				LastActive: time.Now(),
				Frames: []*Frame{
					{
						Fields: map[string]interface{}{
							"url": "/api/auth",
							"headers": map[string]string{
								"authorization": "Bearer secret-token-123",
							},
						},
					},
					{
						Fields: map[string]interface{}{
							"status": 200,
							"headers": map[string]string{
								"authorization": "Bearer secret-token-123",
							},
						},
					},
					{
						Fields: map[string]interface{}{
							"url":       "/api/data",
							"source_ip": "192.168.1.100",
							"headers": map[string]string{
								"authorization": "Bearer secret-token-123",
								"user-agent":    "Mozilla/5.0",
							},
						},
					},
					{
						Fields: map[string]interface{}{
							"url":       "/api/data",
							"source_ip": "10.0.0.50", // Different IP
							"headers": map[string]string{
								"authorization": "Bearer secret-token-123",
								"user-agent":    "Chrome/99.0", // Different UA
							},
						},
					},
				},
			},
			expectedVulnTypes: []string{"token_reuse", "session_hijacking_risk"},
			minExpectedVulns:  2,
		},
		{
			name: "Excessive session reuse",
			session: func() *Session {
				s := &Session{
					ID:         "test-session-3",
					Protocol:   "HTTP",
					StartTime:  time.Now(),
					LastActive: time.Now(),
					Frames:     []*Frame{},
				}
				// Add many frames with same session ID
				for i := 0; i < 15; i++ {
					s.Frames = append(s.Frames, &Frame{
						Fields: map[string]interface{}{
							"headers": map[string]string{
								"cookie": "JSESSIONID=SAME123",
							},
						},
					})
				}
				return s
			}(),
			expectedVulnTypes: []string{"excessive_session_reuse"},
			minExpectedVulns:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vulns := checker.CheckSession(tt.session)

			if len(vulns) < tt.minExpectedVulns {
				t.Errorf("Expected at least %d vulnerabilities, got %d", tt.minExpectedVulns, len(vulns))
			}

			// Check that expected vulnerability types are found
			foundTypes := make(map[string]bool)
			for _, vuln := range vulns {
				foundTypes[vuln.Type] = true
			}

			for _, expectedType := range tt.expectedVulnTypes {
				if !foundTypes[expectedType] {
					t.Errorf("Expected vulnerability type %s not found", expectedType)
				}
			}
		})
	}
}

func TestTokenLeakageChecker(t *testing.T) {
	checker := NewTokenLeakageChecker()

	tests := []struct {
		name             string
		session          *Session
		expectedSeverity string
		minExpectedVulns int
	}{
		{
			name: "API key in multiple locations",
			session: &Session{
				ID:         "test-session-4",
				Protocol:   "HTTP",
				StartTime:  time.Now(),
				LastActive: time.Now(),
				Frames: []*Frame{
					{
						Fields: map[string]interface{}{
							"headers": map[string]string{
								"x-api-key": "sk_live_1234567890abcdef1234567890abcdef",
							},
						},
					},
					{
						Fields: map[string]interface{}{
							"url":     "/api/endpoint?api_key=sk_live_1234567890abcdef1234567890abcdef",
							"payload": []byte(`{"data": "test"}`),
						},
					},
					{
						Fields: map[string]interface{}{
							"payload": []byte(`{"error": "Invalid API key: sk_live_1234567890abcdef1234567890abcdef"}`),
						},
					},
				},
			},
			expectedSeverity: "high",
			minExpectedVulns: 1,
		},
		{
			name: "JWT token exposure",
			session: &Session{
				ID:         "test-session-5",
				Protocol:   "HTTP",
				StartTime:  time.Now(),
				LastActive: time.Now(),
				Frames: []*Frame{
					{
						Fields: map[string]interface{}{
							"headers": map[string]string{
								"authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
							},
						},
					},
					{
						Fields: map[string]interface{}{
							"payload": []byte(`{"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"}`),
						},
					},
				},
			},
			expectedSeverity: "high",
			minExpectedVulns: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vulns := checker.CheckSession(tt.session)

			if len(vulns) < tt.minExpectedVulns {
				t.Errorf("Expected at least %d vulnerabilities, got %d", tt.minExpectedVulns, len(vulns))
			}

			// Check severity
			for _, vuln := range vulns {
				if vuln.Type == "token_leakage" && vuln.Severity != tt.expectedSeverity {
					t.Errorf("Expected severity %s, got %s", tt.expectedSeverity, vuln.Severity)
				}
			}
		})
	}
}

func TestSessionTimeoutChecker(t *testing.T) {
	checker := NewSessionTimeoutChecker()

	tests := []struct {
		name             string
		session          *Session
		expectedVulnType string
		shouldFindVuln   bool
	}{
		{
			name: "Excessive session duration",
			session: &Session{
				ID:         "test-session-6",
				Protocol:   "HTTP",
				StartTime:  time.Now().Add(-25 * time.Hour),
				LastActive: time.Now(),
				Frames:     []*Frame{{}, {}},
			},
			expectedVulnType: "excessive_session_duration",
			shouldFindVuln:   true,
		},
		{
			name: "Long session timeout configuration",
			session: &Session{
				ID:         "test-session-7",
				Protocol:   "HTTP",
				StartTime:  time.Now(),
				LastActive: time.Now(),
				Frames: []*Frame{
					{
						Fields: map[string]interface{}{
							"headers": map[string]string{
								"set-cookie": "JSESSIONID=ABC123; Max-Age=86400", // 24 hours
							},
						},
					},
				},
			},
			expectedVulnType: "long_session_timeout",
			shouldFindVuln:   true,
		},
		{
			name: "Short session timeout",
			session: &Session{
				ID:         "test-session-8",
				Protocol:   "HTTP",
				StartTime:  time.Now(),
				LastActive: time.Now(),
				Frames: []*Frame{
					{
						Fields: map[string]interface{}{
							"headers": map[string]string{
								"set-cookie": "JSESSIONID=ABC123; Max-Age=120", // 2 minutes
							},
						},
					},
				},
			},
			expectedVulnType: "short_session_timeout",
			shouldFindVuln:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vulns := checker.CheckSession(tt.session)

			found := false
			for _, vuln := range vulns {
				if vuln.Type == tt.expectedVulnType {
					found = true
					break
				}
			}

			if found != tt.shouldFindVuln {
				t.Errorf("Expected to find vulnerability: %v, but got: %v", tt.shouldFindVuln, found)
			}
		})
	}
}

func TestWeakSessionChecker(t *testing.T) {
	checker := NewWeakSessionChecker()

	tests := []struct {
		name             string
		session          *Session
		expectedVulnType string
		shouldFindVuln   bool
	}{
		{
			name: "Weak session ID - too short",
			session: &Session{
				ID:         "test-session-9",
				Protocol:   "HTTP",
				StartTime:  time.Now(),
				LastActive: time.Now(),
				Frames: []*Frame{
					{
						Fields: map[string]interface{}{
							"headers": map[string]string{
								"cookie": "JSESSIONID=12345",
							},
						},
					},
				},
			},
			expectedVulnType: "weak_session_id",
			shouldFindVuln:   true,
		},
		{
			name: "Sequential session ID",
			session: &Session{
				ID:         "test-session-10",
				Protocol:   "HTTP",
				StartTime:  time.Now(),
				LastActive: time.Now(),
				Frames: []*Frame{
					{
						Fields: map[string]interface{}{
							"headers": map[string]string{
								"cookie": "JSESSIONID=USER_12345678",
							},
						},
					},
				},
			},
			expectedVulnType: "weak_session_id",
			shouldFindVuln:   true,
		},
		{
			name: "Missing secure flags",
			session: &Session{
				ID:         "test-session-11",
				Protocol:   "HTTP",
				StartTime:  time.Now(),
				LastActive: time.Now(),
				Frames: []*Frame{
					{
						Fields: map[string]interface{}{
							"headers": map[string]string{
								"set-cookie": "JSESSIONID=abc123def456ghi789",
							},
						},
					},
				},
			},
			expectedVulnType: "insecure_session_cookie",
			shouldFindVuln:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vulns := checker.CheckSession(tt.session)

			found := false
			for _, vuln := range vulns {
				if vuln.Type == tt.expectedVulnType {
					found = true

					// Verify specific weakness reasons
					if tt.expectedVulnType == "weak_session_id" {
						if !strings.Contains(vuln.Evidence, "Session ID weakness:") {
							t.Errorf("Expected weakness reason in evidence, got: %s", vuln.Evidence)
						}
					}

					if tt.expectedVulnType == "insecure_session_cookie" {
						if !strings.Contains(vuln.Evidence, "Missing security flags:") {
							t.Errorf("Expected missing flags in evidence, got: %s", vuln.Evidence)
						}
					}
					break
				}
			}

			if found != tt.shouldFindVuln {
				t.Errorf("Expected to find vulnerability: %v, but got: %v", tt.shouldFindVuln, found)
			}
		})
	}
}

func TestCrossSessionChecker(t *testing.T) {
	checker := NewCrossSessionChecker()

	// Create first session with sensitive data
	session1 := &Session{
		ID:         "session-1",
		Protocol:   "HTTP",
		StartTime:  time.Now(),
		LastActive: time.Now(),
		Frames: []*Frame{
			{
				Fields: map[string]interface{}{
					"headers": map[string]string{
						"authorization": "Bearer secret-token-user1",
						"x-api-key":     "user1-api-key-1234567890",
					},
					"payload": []byte(`{"userId": "user1", "data": "sensitive-data-1"}`),
				},
			},
		},
	}

	// Add first session to checker
	vulns1 := checker.CheckSession(session1)
	if len(vulns1) > 0 {
		t.Errorf("First session should not have cross-session vulnerabilities")
	}

	// Create second session that leaks data from first session
	session2 := &Session{
		ID:         "session-2",
		Protocol:   "HTTP",
		StartTime:  time.Now(),
		LastActive: time.Now(),
		Frames: []*Frame{
			{
				Fields: map[string]interface{}{
					"headers": map[string]string{
						"authorization": "Bearer different-token",
					},
					"payload": []byte(`{"error": "Invalid token: Bearer secret-token-user1"}`),
				},
			},
			{
				Raw: []byte("Debug info: user1-api-key-1234567890"),
			},
		},
	}

	// Check second session for cross-session data leakage
	vulns2 := checker.CheckSession(session2)

	foundCrossSessionLeak := false
	for _, vuln := range vulns2 {
		if vuln.Type == "cross_session_data_leak" {
			foundCrossSessionLeak = true
			if vuln.Severity != "critical" {
				t.Errorf("Cross-session data leak should be critical severity, got: %s", vuln.Severity)
			}
		}
	}

	if !foundCrossSessionLeak {
		t.Errorf("Expected to find cross-session data leak vulnerability")
	}
}

// Test helper functions.
func TestExtractSessionIDFromFrame(t *testing.T) {
	tests := []struct {
		name       string
		frame      *Frame
		expectedID string
		shouldFind bool
	}{
		{
			name: "Session ID in cookie",
			frame: &Frame{
				Fields: map[string]interface{}{
					"headers": map[string]string{
						"cookie": "JSESSIONID=ABC123; Path=/; HttpOnly",
					},
				},
			},
			expectedID: "ABC123",
			shouldFind: true,
		},
		{
			name: "Session ID in custom header",
			frame: &Frame{
				Fields: map[string]interface{}{
					"headers": map[string]string{
						"x-session-id": "CUSTOM-SESSION-456",
					},
				},
			},
			expectedID: "CUSTOM-SESSION-456",
			shouldFind: true,
		},
		{
			name: "Session ID in URL",
			frame: &Frame{
				Fields: map[string]interface{}{
					"url": "https://example.com/api?sid=URL-SESSION-789",
				},
			},
			expectedID: "URL-SESSION-789",
			shouldFind: true,
		},
		{
			name: "No session ID",
			frame: &Frame{
				Fields: map[string]interface{}{
					"headers": map[string]string{
						"content-type": "application/json",
					},
				},
			},
			expectedID: "",
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID := extractSessionIDFromFrame(tt.frame)

			if tt.shouldFind {
				if sessionID != tt.expectedID {
					t.Errorf("Expected session ID %s, got %s", tt.expectedID, sessionID)
				}
			} else {
				if sessionID != "" {
					t.Errorf("Expected no session ID, got %s", sessionID)
				}
			}
		})
	}
}

func TestAnalyzeSessionIDStrength(t *testing.T) {
	tests := []struct {
		name         string
		sessionID    string
		expectedWeak string
	}{
		{
			name:         "Too short",
			sessionID:    "12345",
			expectedWeak: "too short",
		},
		{
			name:         "Sequential pattern",
			sessionID:    "USER_12345678901234",
			expectedWeak: "sequential",
		},
		{
			name:         "Timestamp pattern",
			sessionID:    "session_1609459200", // Unix timestamp embedded
			expectedWeak: "predictable",
		},
		{
			name:         "Low entropy",
			sessionID:    "AAAAAAAAAAAAAAAA",
			expectedWeak: "low entropy",
		},
		{
			name:         "Strong session ID",
			sessionID:    "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
			expectedWeak: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weakness := analyzeSessionIDStrength(tt.sessionID)

			if tt.expectedWeak != "" {
				if !strings.Contains(weakness, tt.expectedWeak) {
					t.Errorf("Expected weakness containing '%s', got '%s'", tt.expectedWeak, weakness)
				}
			} else {
				if weakness != "" {
					t.Errorf("Expected no weakness, got '%s'", weakness)
				}
			}
		})
	}
}

func TestCheckHTTPSessionSecurity(t *testing.T) {
	tests := []struct {
		name           string
		frame          *Frame
		expectedIssues []string
	}{
		{
			name: "Missing all security flags",
			frame: &Frame{
				Fields: map[string]interface{}{
					"headers": map[string]string{
						"set-cookie": "JSESSIONID=ABC123",
					},
				},
			},
			expectedIssues: []string{"missing Secure flag", "missing HttpOnly flag", "missing SameSite attribute"},
		},
		{
			name: "Has Secure flag only",
			frame: &Frame{
				Fields: map[string]interface{}{
					"headers": map[string]string{
						"set-cookie": "JSESSIONID=ABC123; Secure",
					},
				},
			},
			expectedIssues: []string{"missing HttpOnly flag", "missing SameSite attribute"},
		},
		{
			name: "All security flags present",
			frame: &Frame{
				Fields: map[string]interface{}{
					"headers": map[string]string{
						"set-cookie": "JSESSIONID=ABC123; Secure; HttpOnly; SameSite=Strict",
					},
				},
			},
			expectedIssues: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checkHTTPSessionSecurity(tt.frame)

			if len(issues) != len(tt.expectedIssues) {
				t.Errorf("Expected %d issues, got %d", len(tt.expectedIssues), len(issues))
			}

			// Check each expected issue is present
			for _, expectedIssue := range tt.expectedIssues {
				found := false
				for _, issue := range issues {
					if issue == expectedIssue {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected issue '%s' not found in %v", expectedIssue, issues)
				}
			}
		})
	}
}

// Benchmark tests.
func BenchmarkAuthSessionChecker(b *testing.B) {
	checker := NewAuthSessionChecker()
	session := createLargeSession(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.CheckSession(session)
	}
}

func BenchmarkTokenLeakageChecker(b *testing.B) {
	checker := NewTokenLeakageChecker()
	session := createLargeSession(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.CheckSession(session)
	}
}

// Helper to create large session for benchmarks.
func createLargeSession(frameCount int) *Session {
	session := &Session{
		ID:         "bench-session",
		Protocol:   "HTTP",
		StartTime:  time.Now(),
		LastActive: time.Now(),
		Frames:     make([]*Frame, 0, frameCount),
	}

	for i := 0; i < frameCount; i++ {
		session.Frames = append(session.Frames, &Frame{
			Fields: map[string]interface{}{
				"headers": map[string]string{
					"cookie":        fmt.Sprintf("JSESSIONID=SESSION_%d", i),
					"authorization": fmt.Sprintf("Bearer token_%d", i),
					"x-api-key":     fmt.Sprintf("api_key_%d_1234567890abcdef", i),
				},
				"url":     fmt.Sprintf("/api/endpoint/%d", i),
				"payload": []byte(fmt.Sprintf(`{"data": "test_%d", "token": "secret_%d"}`, i, i)),
			},
		})
	}

	return session
}
