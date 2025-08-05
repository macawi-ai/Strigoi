package probe

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestGRPCDissector_Identify(t *testing.T) {
	dissector := NewGRPCDissector()

	tests := []struct {
		name       string
		data       []byte
		shouldFind bool
		minConf    float64
	}{
		{
			name:       "HTTP/2 DATA frame with gRPC",
			data:       createHTTP2Frame(0x0, []byte("\x00\x00\x00\x00\x05hello")), // DATA frame
			shouldFind: true,
			minConf:    0.6,
		},
		{
			name:       "HTTP/2 HEADERS frame with gRPC path",
			data:       createHTTP2Frame(0x1, []byte(":path:/grpc.Service/Method")),
			shouldFind: true,
			minConf:    0.9,
		},
		{
			name:       "HTTP/2 with grpc-status header",
			data:       createHTTP2Frame(0x1, []byte("grpc-status:0\x00grpc-message:OK")),
			shouldFind: true,
			minConf:    0.85,
		},
		{
			name:       "HTTP/2 with application/grpc content-type",
			data:       createHTTP2Frame(0x1, []byte("content-type:application/grpc+proto")),
			shouldFind: true,
			minConf:    0.9,
		},
		{
			name:       "Not gRPC - regular HTTP/2",
			data:       createHTTP2Frame(0x1, []byte(":method:GET\x00:path:/api/users")),
			shouldFind: false,
		},
		{
			name:       "Too short data",
			data:       []byte("short"),
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

func TestGRPCDissector_Dissect(t *testing.T) {
	dissector := NewGRPCDissector()

	tests := []struct {
		name          string
		data          []byte
		expectedType  string
		expectedField string
		checkValue    func(interface{}) bool
	}{
		{
			name:          "HTTP/2 DATA frame",
			data:          createHTTP2Frame(0x0, []byte("\x00\x00\x00\x00\x05hello")),
			expectedType:  "data",
			expectedField: "frame_type",
			checkValue:    func(v interface{}) bool { return v == "DATA" },
		},
		{
			name:          "HTTP/2 HEADERS frame",
			data:          createHTTP2Frame(0x1, []byte(":path/grpc.Service/Method")),
			expectedType:  "headers",
			expectedField: "frame_type",
			checkValue:    func(v interface{}) bool { return v == "HEADERS" },
		},
		{
			name:          "gRPC message in DATA frame",
			data:          createHTTP2Frame(0x0, createGRPCMessage(false, []byte("test message"))),
			expectedType:  "data",
			expectedField: "grpc_message",
			checkValue:    func(v interface{}) bool { return v != nil },
		},
		{
			name:          "Raw data (not HTTP/2)",
			data:          []byte("raw grpc data"),
			expectedType:  "raw",
			expectedField: "type",
			checkValue:    func(v interface{}) bool { return v == "raw" },
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
			if frame.Protocol != "gRPC" {
				t.Errorf("Frame protocol = %v, want gRPC", frame.Protocol)
			}

			// Check type if specified
			if tt.expectedType != "" {
				if frameType, ok := frame.Fields["type"].(string); !ok || frameType != tt.expectedType {
					t.Errorf("Frame type = %v, want %v", frameType, tt.expectedType)
				}
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

func TestGRPCDissector_FindVulnerabilities(t *testing.T) {
	dissector := NewGRPCDissector()

	tests := []struct {
		name         string
		frame        *Frame
		minVulns     int
		expectedType string
	}{
		{
			name: "JWT in gRPC headers",
			frame: &Frame{
				Protocol: "gRPC",
				Fields: map[string]interface{}{
					"headers": map[string]string{
						"authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
					},
				},
			},
			minVulns:     1,
			expectedType: "bearer_token",
		},
		{
			name: "API key in gRPC headers",
			frame: &Frame{
				Protocol: "gRPC",
				Fields: map[string]interface{}{
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
				Protocol: "gRPC",
				Fields: map[string]interface{}{
					"payload": []byte(`{"api_key": "secret-key-1234567890abcdef", "token": "auth-token-xyz123456789"}`),
				},
			},
			minVulns: 2,
		},
		{
			name: "JWT in gRPC message data",
			frame: &Frame{
				Protocol: "gRPC",
				Fields: map[string]interface{}{
					"grpc_message": map[string]interface{}{
						"data": []byte(`{"jwt": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MTIzfQ.R0JhPRo9T3K9rXkiV6XzGOmV7DSlrcFKn-vPqQY3Fgw"}`),
					},
				},
			},
			minVulns:     1,
			expectedType: "jwt_in_payload",
		},
		{
			name: "Password in protobuf-like payload",
			frame: &Frame{
				Protocol: "gRPC",
				Fields: map[string]interface{}{
					"payload": []byte("\x0apassword\x12\x0cSecretPass123"),
				},
			},
			minVulns:     1,
			expectedType: "password_in_payload",
		},
		{
			name: "No vulnerabilities",
			frame: &Frame{
				Protocol: "gRPC",
				Fields: map[string]interface{}{
					"headers": map[string]string{
						":path":        "/grpc.Service/Method",
						"content-type": "application/grpc",
					},
					"payload": []byte("normal data without secrets"),
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
				if vuln.Context == "" || !strings.Contains(vuln.Context, "gRPC") {
					t.Errorf("Vulnerability %d has invalid context: %v", i, vuln.Context)
				}
			}
		})
	}
}

func TestGRPCDissector_EdgeCases(t *testing.T) {
	dissector := NewGRPCDissector()

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Malformed HTTP/2 frame",
			data: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name: "Binary data with null bytes",
			data: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
		},
		{
			name: "Very large frame length",
			data: append([]byte{0xFF, 0xFF, 0xFF, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01}, bytes.Repeat([]byte{0x41}, 100)...),
		},
		{
			name: "Mixed valid and invalid data",
			data: append(createHTTP2Frame(0x1, []byte(":path/grpc.Test/Method")), []byte("garbage data")...),
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

func TestContainsProtobufMarkers(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "Valid protobuf varint field",
			data:     []byte{0x08, 0x96, 0x01}, // Field 1, varint 150
			expected: true,
		},
		{
			name:     "Valid protobuf string field",
			data:     []byte{0x12, 0x07, 0x74, 0x65, 0x73, 0x74, 0x69, 0x6E, 0x67}, // Field 2, string "testing"
			expected: true,
		},
		{
			name:     "No protobuf markers",
			data:     []byte("plain text data"),
			expected: false,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsProtobufMarkers(tt.data)
			if result != tt.expected {
				t.Errorf("containsProtobufMarkers() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseGRPCMessage(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		expectNil     bool
		compressed    bool
		messageLength uint32
	}{
		{
			name:          "Uncompressed message",
			data:          createGRPCMessage(false, []byte("test")),
			expectNil:     false,
			compressed:    false,
			messageLength: 4,
		},
		{
			name:          "Compressed message",
			data:          createGRPCMessage(true, []byte("compressed")),
			expectNil:     false,
			compressed:    true,
			messageLength: 10,
		},
		{
			name:      "Too short",
			data:      []byte{0x00, 0x00},
			expectNil: true,
		},
		{
			name:      "Empty",
			data:      []byte{},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGRPCMessage(tt.data)

			if tt.expectNil {
				if result != nil {
					t.Error("Expected nil result")
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			if compressed, ok := result["compressed"].(bool); !ok || compressed != tt.compressed {
				t.Errorf("Compressed = %v, want %v", compressed, tt.compressed)
			}

			if msgLen, ok := result["message_length"].(uint32); !ok || msgLen != tt.messageLength {
				t.Errorf("Message length = %v, want %v", msgLen, tt.messageLength)
			}
		})
	}
}

// Helper functions for tests

func createHTTP2Frame(frameType byte, payload []byte) []byte {
	frame := make([]byte, 9+len(payload))

	// Frame length (24 bits)
	binary.BigEndian.PutUint32(frame[0:4], uint32(len(payload)))
	frame[0] = 0 // Clear first byte (24-bit length)

	// Frame type
	frame[3] = frameType

	// Flags
	frame[4] = 0x00

	// Stream ID (31 bits + 1 reserved bit)
	binary.BigEndian.PutUint32(frame[5:9], 1)

	// Payload
	copy(frame[9:], payload)

	return frame
}

func createGRPCMessage(compressed bool, data []byte) []byte {
	msg := make([]byte, 5+len(data))

	// Compressed flag
	if compressed {
		msg[0] = 1
	}

	// Message length
	binary.BigEndian.PutUint32(msg[1:5], uint32(len(data)))

	// Message data
	copy(msg[5:], data)

	return msg
}
