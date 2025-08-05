package probe

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestWebSocketDissector_Identify(t *testing.T) {
	dissector := NewWebSocketDissector()

	tests := []struct {
		name       string
		data       []byte
		shouldFind bool
		minConf    float64
	}{
		{
			name: "WebSocket handshake request",
			data: []byte("GET /chat HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Upgrade: websocket\r\n" +
				"Connection: Upgrade\r\n" +
				"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
				"Sec-WebSocket-Version: 13\r\n\r\n"),
			shouldFind: true,
			minConf:    0.9,
		},
		{
			name: "WebSocket handshake response",
			data: []byte("HTTP/1.1 101 Switching Protocols\r\n" +
				"Upgrade: websocket\r\n" +
				"Connection: Upgrade\r\n" +
				"Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=\r\n\r\n"),
			shouldFind: true,
			minConf:    0.7,
		},
		{
			name:       "WebSocket text frame",
			data:       createWebSocketFrame(0x81, false, []byte("Hello")),
			shouldFind: true,
			minConf:    0.9,
		},
		{
			name:       "WebSocket binary frame",
			data:       createWebSocketFrame(0x82, false, []byte{0x01, 0x02, 0x03}),
			shouldFind: true,
			minConf:    0.9,
		},
		{
			name:       "WebSocket masked frame",
			data:       createWebSocketFrame(0x81, true, []byte("Masked")),
			shouldFind: true,
			minConf:    0.9,
		},
		{
			name:       "Not WebSocket - regular HTTP",
			data:       []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n"),
			shouldFind: false,
		},
		{
			name:       "Too short data",
			data:       []byte{0x81},
			shouldFind: false,
		},
		{
			name:       "Empty data",
			data:       []byte{},
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, conf := dissector.Identify(tt.data)
			if found != tt.shouldFind {
				t.Errorf("Identify() found = %v, want %v", found, tt.shouldFind)
			}
			if tt.shouldFind && conf < tt.minConf {
				t.Errorf("Identify() confidence = %v, want at least %v", conf, tt.minConf)
			}
		})
	}
}

func TestWebSocketDissector_Dissect(t *testing.T) {
	dissector := NewWebSocketDissector()

	tests := []struct {
		name          string
		data          []byte
		expectedType  string
		expectedField string
		checkValue    func(interface{}) bool
	}{
		{
			name: "WebSocket handshake",
			data: []byte("GET /chat HTTP/1.1\r\n" +
				"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"),
			expectedType:  "handshake",
			expectedField: "ws_key",
			checkValue:    func(v interface{}) bool { return v == "dGhlIHNhbXBsZSBub25jZQ==" },
		},
		{
			name:          "WebSocket text frame",
			data:          createWebSocketFrame(0x81, false, []byte("Hello")),
			expectedType:  "frame",
			expectedField: "opcode",
			checkValue:    func(v interface{}) bool { return v == "text" },
		},
		{
			name:          "WebSocket binary frame",
			data:          createWebSocketFrame(0x82, false, []byte{0x01, 0x02, 0x03}),
			expectedType:  "frame",
			expectedField: "opcode",
			checkValue:    func(v interface{}) bool { return v == "binary" },
		},
		{
			name:          "WebSocket close frame",
			data:          createWebSocketFrame(0x88, false, []byte{0x03, 0xe8}),
			expectedType:  "frame",
			expectedField: "opcode",
			checkValue:    func(v interface{}) bool { return v == "close" },
		},
		{
			name:          "WebSocket ping frame",
			data:          createWebSocketFrame(0x89, false, []byte("ping")),
			expectedType:  "frame",
			expectedField: "opcode",
			checkValue:    func(v interface{}) bool { return v == "ping" },
		},
		{
			name:          "WebSocket pong frame",
			data:          createWebSocketFrame(0x8A, false, []byte("pong")),
			expectedType:  "frame",
			expectedField: "opcode",
			checkValue:    func(v interface{}) bool { return v == "pong" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := dissector.Dissect(tt.data)
			if err != nil {
				t.Fatalf("Dissect() error = %v", err)
			}
			if frame == nil {
				t.Fatal("Dissect() returned nil frame")
			}
			if frame.Protocol != "WebSocket" {
				t.Errorf("Frame protocol = %v, want WebSocket", frame.Protocol)
			}

			// Check type
			if frameType, ok := frame.Fields["type"].(string); !ok || frameType != tt.expectedType {
				t.Errorf("Frame type = %v, want %v", frameType, tt.expectedType)
			}

			// Check specific field
			if tt.expectedField != "" {
				val, ok := frame.Fields[tt.expectedField]
				if !ok {
					t.Errorf("Expected field %s not found", tt.expectedField)
				} else if tt.checkValue != nil && !tt.checkValue(val) {
					t.Errorf("Field %s value check failed, got %v", tt.expectedField, val)
				}
			}
		})
	}
}

func TestWebSocketDissector_FindVulnerabilities(t *testing.T) {
	dissector := NewWebSocketDissector()

	tests := []struct {
		name         string
		frame        *Frame
		minVulns     int
		expectedType string
	}{
		{
			name: "JWT in WebSocket headers",
			frame: &Frame{
				Protocol: "WebSocket",
				Fields: map[string]interface{}{
					"type": "handshake",
					"headers": map[string]string{
						"authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
					},
				},
			},
			minVulns:     1,
			expectedType: "bearer_token",
		},
		{
			name: "API key in WebSocket headers",
			frame: &Frame{
				Protocol: "WebSocket",
				Fields: map[string]interface{}{
					"type": "handshake",
					"headers": map[string]string{
						"x-api-key": "sk-test-1234567890abcdefghijklmnop",
					},
				},
			},
			minVulns:     1,
			expectedType: "api_key",
		},
		{
			name: "Credentials in payload",
			frame: &Frame{
				Protocol: "WebSocket",
				Fields: map[string]interface{}{
					"type":    "frame",
					"payload": []byte(`{"api_key": "secret-key-1234567890", "password": "MySecret123!"}`),
				},
			},
			minVulns: 2,
		},
		{
			name: "JWT in cookie",
			frame: &Frame{
				Protocol: "WebSocket",
				Fields: map[string]interface{}{
					"type": "handshake",
					"headers": map[string]string{
						"cookie": "auth_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MTIzfQ.R0JhPRo5T3K9rXkiV6XzGOmV7DSlrcFKn-vPqQY3Fgw; session_id=abc123456789",
					},
				},
			},
			minVulns:     2,
			expectedType: "jwt_in_cookie",
		},
		{
			name: "Missing origin header",
			frame: &Frame{
				Protocol: "WebSocket",
				Fields: map[string]interface{}{
					"type": "handshake",
					"headers": map[string]string{
						"host": "example.com",
					},
				},
			},
			minVulns:     1,
			expectedType: "missing_origin_check",
		},
		{
			name: "Unencrypted WebSocket",
			frame: &Frame{
				Protocol: "WebSocket",
				Fields: map[string]interface{}{
					"type": "handshake",
					"headers": map[string]string{
						"origin": "http://example.com",
					},
				},
			},
			minVulns:     1,
			expectedType: "unencrypted_websocket",
		},
		{
			name: "No vulnerabilities",
			frame: &Frame{
				Protocol: "WebSocket",
				Fields: map[string]interface{}{
					"type": "handshake",
					"headers": map[string]string{
						"origin": "https://example.com",
						"host":   "example.com",
					},
					"payload": []byte("normal message without secrets"),
				},
			},
			minVulns: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vulns := dissector.FindVulnerabilities(tt.frame)

			if len(vulns) < tt.minVulns {
				t.Errorf("FindVulnerabilities() found %d vulns, want at least %d", len(vulns), tt.minVulns)
			}

			// Check specific vulnerability type if expected
			if tt.expectedType != "" && len(vulns) > 0 {
				found := false
				for _, vuln := range vulns {
					if vuln.Subtype == tt.expectedType {
						found = true
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
				if vuln.Context == "" || !strings.Contains(vuln.Context, "WebSocket") {
					t.Errorf("Vulnerability %d has invalid context: %v", i, vuln.Context)
				}
			}
		})
	}
}

func TestWebSocketDissector_EdgeCases(t *testing.T) {
	dissector := NewWebSocketDissector()

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Malformed WebSocket frame",
			data: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name: "Frame with invalid opcode",
			data: []byte{0x8F, 0x00}, // Opcode 0xF (invalid)
		},
		{
			name: "Frame with extended payload length but insufficient data",
			data: []byte{0x81, 0x7E, 0x00, 0x10}, // Claims 16 bytes but has none
		},
		{
			name: "Frame with 64-bit payload length",
			data: append([]byte{0x81, 0x7F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05}, []byte("Hello")...),
		},
		{
			name: "Mixed handshake and frame data",
			data: append([]byte("GET /ws HTTP/1.1\r\nUpgrade: websocket\r\n\r\n"), createWebSocketFrame(0x81, false, []byte("data"))...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Should not panic
			_, _ = dissector.Identify(tt.data)
			frame, _ := dissector.Dissect(tt.data)
			if frame != nil {
				_ = dissector.FindVulnerabilities(frame)
			}
		})
	}
}

func TestParseWebSocketFrame(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		expectError  bool
		expectedOp   byte
		expectedLen  uint64
		expectedMask bool
	}{
		{
			name:         "Simple text frame",
			data:         createWebSocketFrame(0x81, false, []byte("Hello")),
			expectError:  false,
			expectedOp:   0x1,
			expectedLen:  5,
			expectedMask: false,
		},
		{
			name:         "Masked text frame",
			data:         createWebSocketFrame(0x81, true, []byte("Hello")),
			expectError:  false,
			expectedOp:   0x1,
			expectedLen:  5,
			expectedMask: true,
		},
		{
			name:         "16-bit payload length",
			data:         createWebSocketFrameWithExtendedLength(0x81, false, bytes.Repeat([]byte("A"), 200)),
			expectError:  false,
			expectedOp:   0x1,
			expectedLen:  200,
			expectedMask: false,
		},
		{
			name:        "Too short for frame header",
			data:        []byte{0x81},
			expectError: true,
		},
		{
			name:        "Too short for extended length",
			data:        []byte{0x81, 0x7E, 0x00},
			expectError: true,
		},
		{
			name:        "Too short for mask key",
			data:        []byte{0x81, 0x85, 0x00, 0x00},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := parseWebSocketFrame(tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if frame.Opcode != tt.expectedOp {
				t.Errorf("Opcode = %v, want %v", frame.Opcode, tt.expectedOp)
			}
			if frame.PayloadLength != tt.expectedLen {
				t.Errorf("PayloadLength = %v, want %v", frame.PayloadLength, tt.expectedLen)
			}
			if frame.Masked != tt.expectedMask {
				t.Errorf("Masked = %v, want %v", frame.Masked, tt.expectedMask)
			}
		})
	}
}

// Helper functions for tests

func createWebSocketFrame(header byte, masked bool, payload []byte) []byte {
	frame := []byte{header}

	// Payload length
	payloadLen := len(payload)
	if payloadLen < 126 {
		if masked {
			frame = append(frame, byte(payloadLen|0x80))
		} else {
			frame = append(frame, byte(payloadLen))
		}
	} else if payloadLen < 65536 {
		if masked {
			frame = append(frame, 0xFE) // 126 | 0x80
		} else {
			frame = append(frame, 126)
		}
		lenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lenBytes, uint16(payloadLen))
		frame = append(frame, lenBytes...)
	} else {
		if masked {
			frame = append(frame, 0xFF) // 127 | 0x80
		} else {
			frame = append(frame, 127)
		}
		lenBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lenBytes, uint64(payloadLen))
		frame = append(frame, lenBytes...)
	}

	// Mask key and payload
	if masked {
		maskKey := []byte{0x12, 0x34, 0x56, 0x78}
		frame = append(frame, maskKey...)

		// Mask the payload
		maskedPayload := make([]byte, len(payload))
		for i := range payload {
			maskedPayload[i] = payload[i] ^ maskKey[i%4]
		}
		frame = append(frame, maskedPayload...)
	} else {
		frame = append(frame, payload...)
	}

	return frame
}

func createWebSocketFrameWithExtendedLength(header byte, masked bool, payload []byte) []byte {
	// Force 16-bit length encoding even for small payloads
	frame := []byte{header}

	if masked {
		frame = append(frame, 0xFE) // 126 | 0x80
	} else {
		frame = append(frame, 126)
	}

	lenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBytes, uint16(len(payload)))
	frame = append(frame, lenBytes...)

	if masked {
		maskKey := []byte{0x12, 0x34, 0x56, 0x78}
		frame = append(frame, maskKey...)

		maskedPayload := make([]byte, len(payload))
		for i := range payload {
			maskedPayload[i] = payload[i] ^ maskKey[i%4]
		}
		frame = append(frame, maskedPayload...)
	} else {
		frame = append(frame, payload...)
	}

	return frame
}
