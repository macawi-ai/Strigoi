package probe

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
)

// GRPCDissector parses gRPC protocol data from streams.
type GRPCDissector struct {
	// Patterns for detecting gRPC traffic
	grpcPathPattern *regexp.Regexp

	// Sensitive data patterns
	jwtPattern    *regexp.Regexp
	apiKeyPattern *regexp.Regexp
	tokenPattern  *regexp.Regexp
	bearerPattern *regexp.Regexp
	secretPattern *regexp.Regexp
}

// NewGRPCDissector creates a new gRPC protocol dissector.
func NewGRPCDissector() *GRPCDissector {
	return &GRPCDissector{
		// gRPC paths must have format: /package.Service/Method
		grpcPathPattern: regexp.MustCompile(`^/[\w.]+\.[\w]+/[\w]+$`),

		// Sensitive data patterns
		jwtPattern:    regexp.MustCompile(`eyJ[A-Za-z0-9_\-]+\.eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+`),
		apiKeyPattern: regexp.MustCompile(`(?i)(api[_-]?key|apikey|x-api-key)[:=\s]+([A-Za-z0-9_\-]{20,})`),
		tokenPattern:  regexp.MustCompile(`(?i)(token|access[_-]?token|auth[_-]?token)[:=\s]+([A-Za-z0-9_\-\.]{20,})`),
		bearerPattern: regexp.MustCompile(`(?i)bearer\s+([A-Za-z0-9_\-\.]+)`),
		secretPattern: regexp.MustCompile(`(?i)(secret|private[_-]?key|credential)[:=\s]+([A-Za-z0-9_\-]{16,})`),
	}
}

// Identify checks if the data contains gRPC protocol traffic.
func (d *GRPCDissector) Identify(data []byte) (bool, float64) {
	if len(data) < 9 { // Minimum gRPC frame size
		return false, 0
	}

	// Check for HTTP/2 frame header (9 bytes)
	// Frame format: Length(3) + Type(1) + Flags(1) + StreamID(4)
	if len(data) >= 9 {
		// Check for common HTTP/2 frame types used by gRPC
		frameType := data[3]

		// DATA frames (0x0) or HEADERS frames (0x1) are most common in gRPC
		if frameType == 0x0 || frameType == 0x1 {
			// Look for gRPC-specific patterns
			dataStr := string(data)

			// Check for gRPC headers
			if strings.Contains(dataStr, ":path") {
				path := extractPath(dataStr)
				// gRPC paths must match service/method pattern
				if path != "" && d.grpcPathPattern.MatchString(path) && strings.Contains(path, "/") {
					// Additional check: path should have exactly one slash after service name
					parts := strings.Split(path[1:], "/") // Skip leading slash
					if len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
						return true, 0.9
					}
				}
			}

			// Check for gRPC status headers
			if strings.Contains(dataStr, "grpc-status") || strings.Contains(dataStr, "grpc-message") {
				return true, 0.85
			}

			// Check for content-type: application/grpc
			if strings.Contains(dataStr, "application/grpc") {
				return true, 0.9
			}

			// For DATA frames, check for protobuf-like binary data
			if frameType == 0x0 && len(data) > 9 && containsProtobufMarkers(data[9:]) {
				return true, 0.7
			}
		}
	}

	// Check for gRPC magic bytes in unframed mode (less common)
	if bytes.Contains(data, []byte("\x00\x00\x00\x00")) && containsProtobufMarkers(data) {
		return true, 0.6
	}

	return false, 0
}

// Dissect parses gRPC data into a structured frame.
func (d *GRPCDissector) Dissect(data []byte) (*Frame, error) {
	frame := &Frame{
		Protocol: "gRPC",
		Fields:   make(map[string]interface{}),
		Raw:      data,
	}

	// Try to parse HTTP/2 frame structure
	if len(data) >= 9 {
		// Parse frame header
		frameLength := binary.BigEndian.Uint32(append([]byte{0}, data[0:3]...))
		frameType := data[3]
		frameFlags := data[4]
		streamID := binary.BigEndian.Uint32(data[5:9]) & 0x7FFFFFFF // Clear reserved bit

		frame.Fields["frame_length"] = frameLength
		frame.Fields["frame_type"] = frameTypeToString(frameType)
		frame.Fields["frame_flags"] = frameFlags
		frame.Fields["stream_id"] = streamID

		// Parse frame payload if available
		if len(data) > 9 && frameLength > 0 {
			payload := data[9:]
			if int(frameLength) <= len(payload) {
				payload = payload[:frameLength]
			}

			switch frameType {
			case 0x0: // DATA frame
				frame.Fields["type"] = "data"
				// Try to parse gRPC message
				if grpcMsg := parseGRPCMessage(payload); grpcMsg != nil {
					frame.Fields["grpc_message"] = grpcMsg
				}
				frame.Fields["payload"] = payload

			case 0x1: // HEADERS frame
				frame.Fields["type"] = "headers"
				// Parse headers (simplified - real implementation would use HPACK)
				headers := parseSimpleHeaders(payload)
				frame.Fields["headers"] = headers

			default:
				frame.Fields["type"] = "other"
				frame.Fields["payload"] = payload
			}
		}
	} else {
		// Fallback: treat as raw gRPC message
		if len(data) > 5 && containsProtobufMarkers(data) {
			frame.Fields["type"] = "raw"
		} else {
			frame.Fields["type"] = "other"
		}
		frame.Fields["payload"] = data
	}

	return frame, nil
}

// FindVulnerabilities analyzes a gRPC frame for security issues.
func (d *GRPCDissector) FindVulnerabilities(frame *Frame) []StreamVulnerability {
	var vulns []StreamVulnerability

	// Check headers for sensitive data
	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		vulns = append(vulns, d.checkHeadersForSecrets(headers)...)
	}

	// Check payload for sensitive data
	if payload, ok := frame.Fields["payload"].([]byte); ok {
		vulns = append(vulns, d.checkPayloadForSecrets(payload)...)
	}

	// Check gRPC message if parsed
	if grpcMsg, ok := frame.Fields["grpc_message"].(map[string]interface{}); ok {
		if msgData, ok := grpcMsg["data"].([]byte); ok {
			vulns = append(vulns, d.checkPayloadForSecrets(msgData)...)
		}
	}

	// Add protocol context to all vulnerabilities
	for i := range vulns {
		if frameType, ok := frame.Fields["frame_type"].(string); ok {
			vulns[i].Context = fmt.Sprintf("gRPC %s frame", frameType)
		} else {
			vulns[i].Context = "gRPC communication"
		}
	}

	return vulns
}

// checkHeadersForSecrets looks for sensitive data in gRPC headers.
func (d *GRPCDissector) checkHeadersForSecrets(headers map[string]string) []StreamVulnerability {
	var vulns []StreamVulnerability

	for name, value := range headers {
		nameLower := strings.ToLower(name)

		// Check authorization headers
		if nameLower == "authorization" || nameLower == "grpc-authorization" {
			if matches := d.bearerPattern.FindStringSubmatch(value); matches != nil {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "bearer_token",
					Severity:   "high",
					Confidence: 1.0,
					Evidence:   fmt.Sprintf("%s: Bearer ****", name),
					Location:   "gRPC header",
					Context:    "Bearer token in gRPC authorization header",
				})
			}
		}

		// Check for API keys in headers
		if strings.Contains(nameLower, "api") && strings.Contains(nameLower, "key") ||
			nameLower == "x-api-key" || nameLower == "apikey" {
			if len(value) >= 20 {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "api_key",
					Severity:   "high",
					Confidence: 0.95,
					Evidence:   fmt.Sprintf("%s: %s", name, maskValue(value)),
					Location:   "gRPC header",
					Context:    fmt.Sprintf("API key in gRPC header '%s'", name),
				})
			}
		}

		// Check for JWT tokens
		if d.jwtPattern.MatchString(value) {
			vulns = append(vulns, StreamVulnerability{
				Type:       "credential",
				Subtype:    "jwt_token",
				Severity:   "high",
				Confidence: 0.95,
				Evidence:   fmt.Sprintf("%s: %s", name, maskJWT(value)),
				Location:   "gRPC header",
				Context:    fmt.Sprintf("JWT token in gRPC header '%s'", name),
			})
		}

		// Check custom auth headers
		if strings.Contains(nameLower, "token") || strings.Contains(nameLower, "auth") {
			if len(value) >= 20 && !strings.Contains(nameLower, "content") {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "auth_token",
					Severity:   "medium",
					Confidence: 0.8,
					Evidence:   fmt.Sprintf("%s: %s", name, maskValue(value)),
					Location:   "gRPC header",
					Context:    fmt.Sprintf("Authentication token in gRPC header '%s'", name),
				})
			}
		}
	}

	return vulns
}

// checkPayloadForSecrets looks for sensitive data in gRPC payload.
func (d *GRPCDissector) checkPayloadForSecrets(payload []byte) []StreamVulnerability {
	var vulns []StreamVulnerability

	// Convert to string for pattern matching (handle binary data gracefully)
	payloadStr := string(payload)

	// Look for JWT tokens
	if matches := d.jwtPattern.FindAllString(payloadStr, -1); matches != nil {
		for _, match := range matches {
			vulns = append(vulns, StreamVulnerability{
				Type:       "credential",
				Subtype:    "jwt_in_payload",
				Severity:   "high",
				Confidence: 0.9,
				Evidence:   maskJWT(match),
				Location:   "gRPC payload",
				Context:    "JWT token found in gRPC message payload",
			})
		}
	}

	// Look for API keys - use more flexible pattern for JSON data
	apiKeyPatterns := []*regexp.Regexp{
		d.apiKeyPattern,
		regexp.MustCompile(`"api_key"\s*:\s*"([^"]{20,})"`),
		regexp.MustCompile(`api_key=([A-Za-z0-9_\-]{20,})`),
	}

	for _, pattern := range apiKeyPatterns {
		if matches := pattern.FindAllStringSubmatch(payloadStr, -1); matches != nil {
			for _, match := range matches {
				key := match[len(match)-1] // Get the last capture group
				if len(key) >= 20 {
					vulns = append(vulns, StreamVulnerability{
						Type:       "credential",
						Subtype:    "api_key_in_payload",
						Severity:   "high",
						Confidence: 0.85,
						Evidence:   fmt.Sprintf("api_key: %s", maskValue(key)),
						Location:   "gRPC payload",
						Context:    "API key found in gRPC message payload",
					})
				}
			}
		}
	}

	// Look for tokens - use more flexible patterns
	tokenPatterns := []*regexp.Regexp{
		d.tokenPattern,
		regexp.MustCompile(`"token"\s*:\s*"([^"]{20,})"`),
		regexp.MustCompile(`token=([A-Za-z0-9_\-]{20,})`),
	}

	for _, pattern := range tokenPatterns {
		if matches := pattern.FindAllStringSubmatch(payloadStr, -1); matches != nil {
			for _, match := range matches {
				token := match[len(match)-1] // Get the last capture group
				if len(token) >= 20 {
					vulns = append(vulns, StreamVulnerability{
						Type:       "credential",
						Subtype:    "token_in_payload",
						Severity:   "high",
						Confidence: 0.85,
						Evidence:   fmt.Sprintf("token: %s", maskValue(token)),
						Location:   "gRPC payload",
						Context:    "Authentication token found in gRPC message payload",
					})
				}
			}
		}
	}

	// Look for secrets
	if matches := d.secretPattern.FindAllStringSubmatch(payloadStr, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 3 {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "secret_in_payload",
					Severity:   "critical",
					Confidence: 0.8,
					Evidence:   fmt.Sprintf("%s: %s", match[1], maskValue(match[2])),
					Location:   "gRPC payload",
					Context:    "Secret or private key found in gRPC message payload",
				})
			}
		}
	}

	// Check for common password patterns in protobuf
	passwordPatterns := []string{
		`password["\s]*[:=]\s*"([^"]+)"`,
		`pass["\s]*[:=]\s*"([^"]+)"`,
		`pwd["\s]*[:=]\s*"([^"]+)"`,
		`\x0apassword\x12.([^\x1a\x22]+)`, // Protobuf field pattern
	}

	for _, pattern := range passwordPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindAllStringSubmatch(payloadStr, -1); matches != nil {
			for range matches {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "password_in_payload",
					Severity:   "critical",
					Confidence: 0.85,
					Evidence:   "password: ***",
					Location:   "gRPC payload",
					Context:    "Password found in gRPC message payload",
				})
			}
		}
	}

	return vulns
}

// Helper functions

func frameTypeToString(frameType byte) string {
	switch frameType {
	case 0x0:
		return "DATA"
	case 0x1:
		return "HEADERS"
	case 0x2:
		return "PRIORITY"
	case 0x3:
		return "RST_STREAM"
	case 0x4:
		return "SETTINGS"
	case 0x5:
		return "PUSH_PROMISE"
	case 0x6:
		return "PING"
	case 0x7:
		return "GOAWAY"
	case 0x8:
		return "WINDOW_UPDATE"
	case 0x9:
		return "CONTINUATION"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", frameType)
	}
}

func extractPath(data string) string {
	// Simple extraction - real implementation would parse HPACK properly
	if idx := strings.Index(data, ":path"); idx >= 0 {
		start := idx + 5
		// Skip any separator characters
		for start < len(data) && (data[start] == ' ' || data[start] == ':' || data[start] == '\x00') {
			start++
		}
		// Find the end of the path
		end := start
		for end < len(data) && data[end] != '\x00' && data[end] != '\n' && data[end] != ' ' && data[end] != '\r' {
			end++
		}
		if end > start {
			path := data[start:end]
			// Validate it's a proper path
			if strings.HasPrefix(path, "/") {
				return path
			}
		}
	}
	return ""
}

func containsProtobufMarkers(data []byte) bool {
	// Look for common protobuf field markers
	// Field format: (field_number << 3) | wire_type

	// Need at least 2 bytes for a valid protobuf field
	if len(data) < 2 {
		return false
	}

	validFieldCount := 0
	for i := 0; i < len(data)-1; i++ {
		fieldByte := data[i]
		if fieldByte == 0 {
			continue // Skip null bytes
		}

		wireType := fieldByte & 0x07
		fieldNumber := fieldByte >> 3

		// Valid field numbers are typically 1-127
		if fieldNumber > 0 && fieldNumber < 128 {
			// Common wire types: 0 (varint), 2 (length-delimited)
			if wireType == 0 || wireType == 2 {
				// Check if next byte could be valid protobuf data
				if i+1 < len(data) && data[i+1] > 0 {
					validFieldCount++
					if validFieldCount >= 2 { // Require at least 2 valid fields
						return true
					}
				}
			}
		}
	}
	return false
}

func parseGRPCMessage(data []byte) map[string]interface{} {
	if len(data) < 5 {
		return nil
	}

	// gRPC message format: Compressed(1) + MessageLength(4) + Message
	compressed := data[0]
	messageLength := binary.BigEndian.Uint32(data[1:5])

	result := make(map[string]interface{})
	result["compressed"] = compressed == 1
	result["message_length"] = messageLength

	if len(data) >= 5+int(messageLength) {
		result["data"] = data[5 : 5+messageLength]
	}

	return result
}

func parseSimpleHeaders(data []byte) map[string]string {
	// Simplified header parsing - real implementation would use HPACK
	headers := make(map[string]string)

	// Look for common header patterns
	dataStr := string(data)
	patterns := []string{":path", ":method", ":status", ":authority", ":scheme",
		"content-type", "grpc-status", "grpc-message", "authorization"}

	for _, pattern := range patterns {
		if idx := strings.Index(dataStr, pattern); idx >= 0 {
			// Try to extract value (very simplified)
			start := idx + len(pattern)
			for start < len(dataStr) && (dataStr[start] == ' ' || dataStr[start] == ':') {
				start++
			}
			end := start
			for end < len(dataStr) && dataStr[end] != '\x00' && dataStr[end] != '\n' {
				end++
			}
			if end > start {
				headers[pattern] = dataStr[start:end]
			}
		}
	}

	return headers
}

func maskJWT(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		// Show first few chars of header and signature
		header := parts[0]
		if len(header) > 10 {
			header = header[:10] + "..."
		}
		return fmt.Sprintf("%s.****.**", header)
	}
	return maskValue(token)
}

// GetSessionID extracts session identifier from gRPC frame.
func (d *GRPCDissector) GetSessionID(frame *Frame) (string, error) {
	// gRPC uses HTTP/2 streams, so we use stream ID as the base
	streamID, ok := frame.Fields["stream_id"].(uint32)
	if !ok {
		return "", fmt.Errorf("no stream ID in gRPC frame")
	}

	// For stream ID 0, this is connection-level, not a request
	if streamID == 0 {
		return "", fmt.Errorf("stream ID 0 is connection-level, not a session")
	}

	// Check if we have connection info to make session ID unique
	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		// Check for authorization which can group multiple streams
		if auth := headers["authorization"]; auth != "" {
			// Use auth + stream ID for unique session per authenticated stream
			hash := sha256.Sum256([]byte(auth))
			return fmt.Sprintf("grpc_auth_%x_stream_%d", hash[:4], streamID), nil
		}

		// Check for custom session headers
		sessionHeaders := []string{"x-session-id", "x-request-id", "x-correlation-id"}
		for _, hdr := range sessionHeaders {
			if val := headers[hdr]; val != "" {
				return fmt.Sprintf("grpc_header_%s", val), nil
			}
		}

		// Use :path as part of session ID if available
		if path := headers[":path"]; path != "" {
			// Combine path and stream ID
			pathHash := sha256.Sum256([]byte(path))
			return fmt.Sprintf("grpc_path_%x_stream_%d", pathHash[:4], streamID), nil
		}
	}

	// Fallback to just stream ID
	return fmt.Sprintf("grpc_stream_%d", streamID), nil
}
