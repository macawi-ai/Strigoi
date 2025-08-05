package probe

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
)

// WebSocketDissector parses WebSocket protocol data from streams.
type WebSocketDissector struct {
	// WebSocket magic GUID for handshake
	wsGUID []byte

	// Patterns for detecting WebSocket traffic
	upgradePattern *regexp.Regexp
	wsKeyPattern   *regexp.Regexp

	// Sensitive data patterns
	jwtPattern      *regexp.Regexp
	apiKeyPattern   *regexp.Regexp
	tokenPattern    *regexp.Regexp
	bearerPattern   *regexp.Regexp
	secretPattern   *regexp.Regexp
	passwordPattern *regexp.Regexp
}

// NewWebSocketDissector creates a new WebSocket protocol dissector.
func NewWebSocketDissector() *WebSocketDissector {
	return &WebSocketDissector{
		// WebSocket magic GUID
		wsGUID: []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"),

		// WebSocket patterns
		upgradePattern: regexp.MustCompile(`(?i)upgrade:\s*websocket`),
		wsKeyPattern:   regexp.MustCompile(`(?i)sec-websocket-key:\s*([A-Za-z0-9+/=]{24})`),

		// Sensitive data patterns
		jwtPattern:      regexp.MustCompile(`eyJ[A-Za-z0-9_\-]+\.eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+`),
		apiKeyPattern:   regexp.MustCompile(`(?i)(api[_-]?key|apikey)[:=\s]+([A-Za-z0-9_\-]{20,})`),
		tokenPattern:    regexp.MustCompile(`(?i)(token|access[_-]?token|auth[_-]?token)[:=\s]+([A-Za-z0-9_\-\.]{20,})`),
		bearerPattern:   regexp.MustCompile(`(?i)bearer\s+([A-Za-z0-9_\-\.]+)`),
		secretPattern:   regexp.MustCompile(`(?i)(secret|private[_-]?key)[:=\s]+([A-Za-z0-9_\-]{16,})`),
		passwordPattern: regexp.MustCompile(`(?i)(password|passwd|pwd)[:=\s]+['"]*([^'"\s]{6,})['"]*`),
	}
}

// Identify checks if the data contains WebSocket protocol traffic.
func (d *WebSocketDissector) Identify(data []byte) (bool, float64) {
	if len(data) < 2 {
		return false, 0
	}

	dataStr := string(data)
	confidence := 0.0

	// Check for WebSocket upgrade handshake
	if d.upgradePattern.MatchString(dataStr) {
		confidence += 0.4

		// Check for Sec-WebSocket-Key header
		if d.wsKeyPattern.MatchString(dataStr) {
			confidence += 0.4
		}

		// Check for other WebSocket headers
		if strings.Contains(dataStr, "Sec-WebSocket-Version") {
			confidence += 0.1
		}
		if strings.Contains(dataStr, "Sec-WebSocket-Accept") {
			confidence += 0.3
		}

		return true, confidence
	}

	// Check for WebSocket frame structure
	if isWebSocketFrame(data) {
		return true, 0.9
	}

	return false, 0
}

// Dissect parses WebSocket data into a structured frame.
func (d *WebSocketDissector) Dissect(data []byte) (*Frame, error) {
	frame := &Frame{
		Protocol: "WebSocket",
		Fields:   make(map[string]interface{}),
		Raw:      data,
	}

	// Check if this is a handshake or frame data
	if bytes.Contains(data, []byte("HTTP/1.1")) || bytes.Contains(data, []byte("GET ")) {
		// WebSocket handshake
		frame.Fields["type"] = "handshake"
		headers := d.parseHandshakeHeaders(data)
		frame.Fields["headers"] = headers

		// Extract WebSocket-specific headers
		if key, ok := headers["sec-websocket-key"]; ok {
			frame.Fields["ws_key"] = key
		}
		if accept, ok := headers["sec-websocket-accept"]; ok {
			frame.Fields["ws_accept"] = accept
		}
		if version, ok := headers["sec-websocket-version"]; ok {
			frame.Fields["ws_version"] = version
		}
		if protocol, ok := headers["sec-websocket-protocol"]; ok {
			frame.Fields["ws_protocol"] = protocol
		}
	} else if len(data) >= 2 {
		// Try to parse as WebSocket frame
		wsFrame, err := parseWebSocketFrame(data)
		if err != nil {
			frame.Fields["type"] = "unknown"
			frame.Fields["error"] = err.Error()
			return frame, nil
		}

		frame.Fields["type"] = "frame"
		frame.Fields["fin"] = wsFrame.Fin
		frame.Fields["opcode"] = opcodeToString(wsFrame.Opcode)
		frame.Fields["masked"] = wsFrame.Masked
		frame.Fields["payload_length"] = wsFrame.PayloadLength

		if wsFrame.Payload != nil {
			frame.Fields["payload"] = wsFrame.Payload
		}
	} else {
		frame.Fields["type"] = "unknown"
	}

	return frame, nil
}

// FindVulnerabilities analyzes a WebSocket frame for security issues.
func (d *WebSocketDissector) FindVulnerabilities(frame *Frame) []StreamVulnerability {
	var vulns []StreamVulnerability

	// Check handshake headers
	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		vulns = append(vulns, d.checkHeadersForSecrets(headers)...)

		// Check for missing security headers
		if _, hasOrigin := headers["origin"]; !hasOrigin {
			vulns = append(vulns, StreamVulnerability{
				Type:       "configuration",
				Subtype:    "missing_origin_check",
				Severity:   "medium",
				Confidence: 0.8,
				Evidence:   "No Origin header in WebSocket handshake",
				Location:   "WebSocket handshake",
				Context:    "Missing origin validation could allow cross-site WebSocket hijacking",
			})
		}
	}

	// Check frame payload
	if payload, ok := frame.Fields["payload"].([]byte); ok {
		vulns = append(vulns, d.checkPayloadForSecrets(payload)...)
	}

	// Check for unencrypted WebSocket (ws:// instead of wss://)
	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		if origin, hasOrigin := headers["origin"]; hasOrigin && strings.HasPrefix(origin, "http://") {
			vulns = append(vulns, StreamVulnerability{
				Type:       "configuration",
				Subtype:    "unencrypted_websocket",
				Severity:   "high",
				Confidence: 0.9,
				Evidence:   fmt.Sprintf("WebSocket origin: %s", origin),
				Location:   "WebSocket handshake",
				Context:    "Unencrypted WebSocket connection vulnerable to eavesdropping",
			})
		}
	}

	// Add protocol context
	for i := range vulns {
		if frameType, ok := frame.Fields["type"].(string); ok {
			if !strings.Contains(vulns[i].Context, "WebSocket") {
				vulns[i].Context = fmt.Sprintf("WebSocket %s - %s", frameType, vulns[i].Context)
			}
		}
	}

	return vulns
}

// WebSocket frame structure.
type wsFrame struct {
	Fin           bool
	RSV1          bool
	RSV2          bool
	RSV3          bool
	Opcode        byte
	Masked        bool
	PayloadLength uint64
	MaskKey       []byte
	Payload       []byte
}

// Check if data looks like a WebSocket frame.
func isWebSocketFrame(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	// Check opcode
	opcode := data[0] & 0x0F
	if opcode > 0x0F {
		return false
	}

	// Valid opcodes: 0x0 (continuation), 0x1 (text), 0x2 (binary), 0x8 (close), 0x9 (ping), 0xA (pong)
	validOpcodes := []byte{0x0, 0x1, 0x2, 0x8, 0x9, 0xA}
	validOpcode := false
	for _, valid := range validOpcodes {
		if opcode == valid {
			validOpcode = true
			break
		}
	}

	if !validOpcode {
		return false
	}

	// Check payload length encoding
	payloadLen := data[1] & 0x7F
	minLen := 2
	if payloadLen == 126 {
		minLen = 4
	} else if payloadLen == 127 {
		minLen = 10
	}

	// Check mask bit
	masked := (data[1] & 0x80) != 0
	if masked {
		minLen += 4
	}

	return len(data) >= minLen
}

// Parse WebSocket frame.
func parseWebSocketFrame(data []byte) (*wsFrame, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("insufficient data for WebSocket frame")
	}

	frame := &wsFrame{}

	// Parse first byte
	frame.Fin = (data[0] & 0x80) != 0
	frame.RSV1 = (data[0] & 0x40) != 0
	frame.RSV2 = (data[0] & 0x20) != 0
	frame.RSV3 = (data[0] & 0x10) != 0
	frame.Opcode = data[0] & 0x0F

	// Parse second byte
	frame.Masked = (data[1] & 0x80) != 0
	payloadLen := uint64(data[1] & 0x7F)

	offset := 2

	// Extended payload length
	if payloadLen == 126 {
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for 16-bit payload length")
		}
		payloadLen = uint64(binary.BigEndian.Uint16(data[2:4]))
		offset = 4
	} else if payloadLen == 127 {
		if len(data) < 10 {
			return nil, fmt.Errorf("insufficient data for 64-bit payload length")
		}
		payloadLen = binary.BigEndian.Uint64(data[2:10])
		offset = 10
	}

	frame.PayloadLength = payloadLen

	// Masking key
	if frame.Masked {
		if len(data) < offset+4 {
			return nil, fmt.Errorf("insufficient data for mask key")
		}
		frame.MaskKey = data[offset : offset+4]
		offset += 4
	}

	// Payload
	if payloadLen > 0 {
		if uint64(len(data)) < uint64(offset)+payloadLen { //nolint:gosec
			return nil, fmt.Errorf("insufficient data for payload")
		}
		frame.Payload = make([]byte, payloadLen)
		copy(frame.Payload, data[offset:offset+int(payloadLen)])

		// Unmask payload if needed
		if frame.Masked && frame.MaskKey != nil {
			for i := range frame.Payload {
				frame.Payload[i] ^= frame.MaskKey[i%4]
			}
		}
	}

	return frame, nil
}

// Convert opcode to string.
func opcodeToString(opcode byte) string {
	switch opcode {
	case 0x0:
		return "continuation"
	case 0x1:
		return "text"
	case 0x2:
		return "binary"
	case 0x8:
		return "close"
	case 0x9:
		return "ping"
	case 0xA:
		return "pong"
	default:
		return fmt.Sprintf("unknown(0x%X)", opcode)
	}
}

// Parse handshake headers.
func (d *WebSocketDissector) parseHandshakeHeaders(data []byte) map[string]string {
	headers := make(map[string]string)
	lines := bytes.Split(data, []byte("\r\n"))

	for _, line := range lines {
		if len(line) == 0 {
			break
		}

		parts := bytes.SplitN(line, []byte(":"), 2)
		if len(parts) == 2 {
			key := strings.ToLower(string(bytes.TrimSpace(parts[0])))
			value := string(bytes.TrimSpace(parts[1]))
			headers[key] = value
		}
	}

	return headers
}

// Check headers for secrets.
func (d *WebSocketDissector) checkHeadersForSecrets(headers map[string]string) []StreamVulnerability {
	var vulns []StreamVulnerability

	for name, value := range headers {
		nameLower := strings.ToLower(name)

		// Check authorization headers
		if nameLower == "authorization" {
			if matches := d.bearerPattern.FindStringSubmatch(value); matches != nil {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "bearer_token",
					Severity:   "high",
					Confidence: 1.0,
					Evidence:   fmt.Sprintf("%s: Bearer ****", name),
					Location:   "WebSocket header",
					Context:    "Bearer token in authorization header",
				})
			}
		}

		// Check for API keys
		if strings.Contains(nameLower, "api") && strings.Contains(nameLower, "key") {
			if len(value) >= 20 {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "api_key",
					Severity:   "high",
					Confidence: 0.95,
					Evidence:   fmt.Sprintf("%s: %s", name, maskValue(value)),
					Location:   "WebSocket header",
					Context:    fmt.Sprintf("API key in header '%s'", name),
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
				Location:   "WebSocket header",
				Context:    fmt.Sprintf("JWT token in header '%s'", name),
			})
		}

		// Check cookies for sensitive data
		if nameLower == "cookie" {
			cookies := parseCookies(value)
			for cookieName, cookieValue := range cookies {
				if d.jwtPattern.MatchString(cookieValue) {
					vulns = append(vulns, StreamVulnerability{
						Type:       "credential",
						Subtype:    "jwt_in_cookie",
						Severity:   "high",
						Confidence: 0.9,
						Evidence:   fmt.Sprintf("Cookie %s: %s", cookieName, maskJWT(cookieValue)),
						Location:   "WebSocket cookie",
						Context:    "JWT token in cookie",
					})
				} else if strings.Contains(strings.ToLower(cookieName), "session") && len(cookieValue) >= 20 {
					vulns = append(vulns, StreamVulnerability{
						Type:       "credential",
						Subtype:    "session_cookie",
						Severity:   "medium",
						Confidence: 0.8,
						Evidence:   fmt.Sprintf("Cookie %s: %s", cookieName, maskValue(cookieValue)),
						Location:   "WebSocket cookie",
						Context:    "Session identifier in cookie",
					})
				}
			}
		}
	}

	return vulns
}

// Check payload for secrets.
func (d *WebSocketDissector) checkPayloadForSecrets(payload []byte) []StreamVulnerability {
	var vulns []StreamVulnerability
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
				Location:   "WebSocket payload",
				Context:    "JWT token in message payload",
			})
		}
	}

	// Look for API keys - check multiple patterns
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
						Location:   "WebSocket payload",
						Context:    "API key in message payload",
					})
					break // Only report once per type
				}
			}
		}
	}

	// Look for tokens
	if matches := d.tokenPattern.FindAllStringSubmatch(payloadStr, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 3 {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "token_in_payload",
					Severity:   "high",
					Confidence: 0.85,
					Evidence:   fmt.Sprintf("%s: %s", match[1], maskValue(match[2])),
					Location:   "WebSocket payload",
					Context:    "Authentication token in message payload",
				})
			}
		}
	}

	// Look for passwords - check multiple patterns
	passwordPatterns := []*regexp.Regexp{
		d.passwordPattern,
		regexp.MustCompile(`"password"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`password=([^&\s]+)`),
	}

	for _, pattern := range passwordPatterns {
		if matches := pattern.FindAllStringSubmatch(payloadStr, -1); matches != nil {
			vulns = append(vulns, StreamVulnerability{
				Type:       "credential",
				Subtype:    "password_in_payload",
				Severity:   "critical",
				Confidence: 0.9,
				Evidence:   "password: ***",
				Location:   "WebSocket payload",
				Context:    "Password in message payload",
			})
			break // Only report once
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
					Confidence: 0.85,
					Evidence:   fmt.Sprintf("%s: %s", match[1], maskValue(match[2])),
					Location:   "WebSocket payload",
					Context:    "Secret or private key in message payload",
				})
			}
		}
	}

	return vulns
}

// Parse cookie string - reuse the function from dissector_http.go
// parseCookies is already defined in dissector_http.go

// GetSessionID extracts session identifier from WebSocket frame.
func (d *WebSocketDissector) GetSessionID(frame *Frame) (string, error) {
	// For handshake frames, extract from headers
	if frameType, ok := frame.Fields["type"].(string); ok && frameType == "handshake" {
		headers, ok := frame.Fields["headers"].(map[string]string)
		if !ok {
			return "", fmt.Errorf("no headers in WebSocket handshake")
		}

		// Use Sec-WebSocket-Key as primary session identifier
		if wsKey := headers["sec-websocket-key"]; wsKey != "" {
			return fmt.Sprintf("ws_key_%s", wsKey), nil
		}

		// Check cookies for session info
		if cookie := headers["cookie"]; cookie != "" {
			cookies := parseCookies(cookie)
			sessionNames := []string{"JSESSIONID", "PHPSESSID", "session_id", "sid", "sessionid"}

			for _, name := range sessionNames {
				if val, ok := cookies[name]; ok && val != "" {
					return fmt.Sprintf("ws_cookie_%s", val), nil
				}
			}
		}

		// Use authorization header if present
		if auth := headers["authorization"]; auth != "" {
			hash := sha256.Sum256([]byte(auth))
			return fmt.Sprintf("ws_auth_%x", hash[:8]), nil
		}
	}

	// For data frames, we need connection tracking
	// This is a limitation - we'd need connection state management
	// For now, return error indicating we need session context
	return "", fmt.Errorf("WebSocket data frames require connection tracking for session ID")
}
