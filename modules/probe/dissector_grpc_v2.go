package probe

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// GRPCDissectorV2 is an improved gRPC protocol dissector with better performance and security.
type GRPCDissectorV2 struct {
	// Pre-compiled patterns for better performance
	patterns struct {
		grpcPath   *regexp.Regexp
		jwt        *regexp.Regexp
		apiKey     *regexp.Regexp
		token      *regexp.Regexp
		bearer     *regexp.Regexp
		secret     *regexp.Regexp
		oauth      *regexp.Regexp
		sessionID  *regexp.Regexp
		customAuth *regexp.Regexp
		basicAuth  *regexp.Regexp
	}

	// Cache for performance
	cache struct {
		mu      sync.RWMutex
		results map[string]cachedResult
	}

	// Byte patterns for efficient searching
	bytePatterns struct {
		grpcStatus  []byte
		grpcMessage []byte
		contentType []byte
		grpcType    []byte
		pathPrefix  []byte
	}
}

type cachedResult struct {
	isGRPC     bool
	confidence float64
}

// NewGRPCDissectorV2 creates an improved gRPC protocol dissector.
func NewGRPCDissectorV2() *GRPCDissectorV2 {
	d := &GRPCDissectorV2{}

	// Initialize pre-compiled regex patterns (with ReDoS prevention in mind)
	d.patterns.grpcPath = regexp.MustCompile(`^/[\w]{1,50}\.[\w]{1,50}/[\w]{1,50}$`)
	d.patterns.jwt = regexp.MustCompile(`^eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}$`)
	d.patterns.apiKey = regexp.MustCompile(`(?i)^(api[_-]?key|x-api-key)$`)
	d.patterns.token = regexp.MustCompile(`(?i)^(auth|access|refresh)[_-]?token$`)
	d.patterns.bearer = regexp.MustCompile(`^Bearer\s+[\w.-]{20,200}$`)
	d.patterns.oauth = regexp.MustCompile(`^OAuth\s+[\w.-]{20,200}$`)
	d.patterns.sessionID = regexp.MustCompile(`(?i)^(session[_-]?id|phpsessid|jsessionid)$`)
	d.patterns.customAuth = regexp.MustCompile(`(?i)^x-auth[_-]?token$`)
	d.patterns.basicAuth = regexp.MustCompile(`^Basic\s+[A-Za-z0-9+/]{20,}={0,2}$`)

	// Initialize byte patterns for efficient searching
	d.bytePatterns.grpcStatus = []byte("grpc-status")
	d.bytePatterns.grpcMessage = []byte("grpc-message")
	d.bytePatterns.contentType = []byte("content-type")
	d.bytePatterns.grpcType = []byte("application/grpc")
	d.bytePatterns.pathPrefix = []byte(":path")

	// Initialize cache
	d.cache.results = make(map[string]cachedResult)

	return d
}

// Identify checks if the data contains gRPC protocol traffic with improved detection.
func (d *GRPCDissectorV2) Identify(data []byte) (bool, float64) {
	// Check cache first
	cacheKey := fmt.Sprintf("%x", data[:min(64, len(data))])
	d.cache.mu.RLock()
	if cached, ok := d.cache.results[cacheKey]; ok {
		d.cache.mu.RUnlock()
		return cached.isGRPC, cached.confidence
	}
	d.cache.mu.RUnlock()

	// Minimum size check
	if len(data) < 9 {
		return d.cacheAndReturn(cacheKey, false, 0)
	}

	// Parse HTTP/2 frame header
	frame, err := parseHTTP2FrameHeader(data)
	if err != nil {
		return d.cacheAndReturn(cacheKey, false, 0)
	}

	// Calculate confidence based on multiple factors
	confidence := 0.0
	indicators := 0

	// Check frame type
	if frame.Type == frameTypeData || frame.Type == frameTypeHeaders {
		confidence += 0.1
		indicators++

		// Extract payload safely
		payload := extractFramePayload(data, frame)
		if payload == nil {
			return d.cacheAndReturn(cacheKey, false, confidence)
		}

		// Check for gRPC-specific patterns using byte operations
		if bytes.Contains(payload, d.bytePatterns.grpcStatus) {
			confidence += 0.2
			indicators++
		}

		if bytes.Contains(payload, d.bytePatterns.grpcMessage) {
			confidence += 0.2
			indicators++
		}

		// Check for application/grpc content-type
		if d.hasGRPCContentType(payload) {
			confidence += 0.3
			indicators++
		}

		// Check path format
		if path := d.extractPathSafely(payload); path != "" {
			if d.patterns.grpcPath.MatchString(path) {
				confidence += 0.3
				indicators++
			}
		}

		// For DATA frames, check for protobuf markers
		if frame.Type == frameTypeData && d.hasProtobufMarkersV2(payload) {
			confidence += 0.2
			indicators++
		}
	}

	// Determine if it's gRPC based on confidence
	isGRPC := confidence >= 0.5 && indicators >= 2

	return d.cacheAndReturn(cacheKey, isGRPC, confidence)
}

// Dissect parses gRPC data into a structured frame with improved parsing.
func (d *GRPCDissectorV2) Dissect(data []byte) (*Frame, error) {
	frame := &Frame{
		Protocol: "gRPC",
		Fields:   make(map[string]interface{}),
		Raw:      data,
	}

	// Parse HTTP/2 frame
	h2frame, err := parseHTTP2FrameHeader(data)
	if err != nil {
		frame.Fields["type"] = "unknown"
		frame.Fields["error"] = err.Error()
		return frame, nil
	}

	frame.Fields["frame_length"] = h2frame.Length
	frame.Fields["frame_type"] = frameTypeToString(h2frame.Type)
	frame.Fields["frame_flags"] = h2frame.Flags
	frame.Fields["stream_id"] = h2frame.StreamID

	// Extract and process payload
	payload := extractFramePayload(data, h2frame)
	if payload == nil {
		return frame, nil
	}

	switch h2frame.Type {
	case frameTypeData:
		frame.Fields["type"] = "data"
		// Parse gRPC message
		if grpcMsg := d.parseGRPCMessageV2(payload); grpcMsg != nil {
			frame.Fields["grpc_message"] = grpcMsg
		}
		frame.Fields["payload"] = payload

	case frameTypeHeaders:
		frame.Fields["type"] = "headers"
		headers := d.parseHeadersSafely(payload)
		frame.Fields["headers"] = headers

	default:
		frame.Fields["type"] = "other"
		frame.Fields["payload"] = payload
	}

	return frame, nil
}

// FindVulnerabilities analyzes a gRPC frame for security issues with improved detection.
func (d *GRPCDissectorV2) FindVulnerabilities(frame *Frame) []StreamVulnerability {
	var vulns []StreamVulnerability

	// Check headers
	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		vulns = append(vulns, d.checkHeadersForSecretsV2(headers)...)
	}

	// Check payload
	if payload, ok := frame.Fields["payload"].([]byte); ok {
		vulns = append(vulns, d.checkPayloadForSecretsV2(payload)...)
	}

	// Check parsed gRPC message
	if grpcMsg, ok := frame.Fields["grpc_message"].(map[string]interface{}); ok {
		if msgData, ok := grpcMsg["data"].([]byte); ok {
			vulns = append(vulns, d.checkPayloadForSecretsV2(msgData)...)
		}
	}

	// Add protocol context
	for i := range vulns {
		if frameType, ok := frame.Fields["frame_type"].(string); ok {
			vulns[i].Context = fmt.Sprintf("gRPC %s frame", frameType)
		} else {
			vulns[i].Context = "gRPC communication"
		}
	}

	return vulns
}

// Helper functions

type http2Frame struct {
	Length   uint32
	Type     byte
	Flags    byte
	StreamID uint32
}

const (
	frameTypeData         = 0x0
	frameTypeHeaders      = 0x1
	frameTypePriority     = 0x2
	frameTypeRstStream    = 0x3
	frameTypeSettings     = 0x4
	frameTypePushPromise  = 0x5
	frameTypePing         = 0x6
	frameTypeGoaway       = 0x7
	frameTypeWindowUpdate = 0x8
	frameTypeContinuation = 0x9
)

func parseHTTP2FrameHeader(data []byte) (*http2Frame, error) {
	if len(data) < 9 {
		return nil, fmt.Errorf("insufficient data for HTTP/2 frame header")
	}

	frame := &http2Frame{}
	frame.Length = uint32(data[0])<<16 | uint32(data[1])<<8 | uint32(data[2])
	frame.Type = data[3]
	frame.Flags = data[4]
	frame.StreamID = binary.BigEndian.Uint32(data[5:9]) & 0x7FFFFFFF

	// Validate frame
	if frame.Length > 16777215 { // Max frame size
		return nil, fmt.Errorf("invalid frame length: %d", frame.Length)
	}

	return frame, nil
}

func extractFramePayload(data []byte, frame *http2Frame) []byte {
	if len(data) < 9+int(frame.Length) {
		return nil
	}
	return data[9 : 9+frame.Length]
}

func (d *GRPCDissectorV2) hasGRPCContentType(data []byte) bool {
	// Look for content-type header
	idx := bytes.Index(data, d.bytePatterns.contentType)
	if idx < 0 {
		return false
	}

	// Check if followed by application/grpc
	start := idx + len(d.bytePatterns.contentType)
	if start+20 < len(data) {
		// Skip separators
		for start < len(data) && (data[start] == ':' || data[start] == ' ') {
			start++
		}
		if bytes.HasPrefix(data[start:], d.bytePatterns.grpcType) {
			return true
		}
	}
	return false
}

func (d *GRPCDissectorV2) extractPathSafely(data []byte) string {
	idx := bytes.Index(data, d.bytePatterns.pathPrefix)
	if idx < 0 {
		return ""
	}

	start := idx + len(d.bytePatterns.pathPrefix)
	// Skip separators
	for start < len(data) && (data[start] == ':' || data[start] == ' ' || data[start] == 0) {
		start++
	}

	// Find end of path
	end := start
	for end < len(data) && data[end] != 0 && data[end] != '\n' && data[end] != ' ' {
		end++
		// Prevent excessive scanning
		if end-start > 256 {
			return ""
		}
	}

	if end > start && data[start] == '/' {
		return string(data[start:end])
	}
	return ""
}

func (d *GRPCDissectorV2) hasProtobufMarkersV2(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	validFields := 0
	i := 0

	for i < len(data)-1 && i < 100 { // Limit scanning
		if data[i] == 0 {
			i++
			continue
		}

		wireType := data[i] & 0x07
		fieldNum := data[i] >> 3

		// Valid protobuf field
		if fieldNum > 0 && fieldNum <= 100 && (wireType == 0 || wireType == 2) {
			// Verify next byte is plausible
			if i+1 < len(data) {
				nextByte := data[i+1]
				if wireType == 0 && nextByte < 128 { // Varint
					validFields++
				} else if wireType == 2 && nextByte > 0 && nextByte < 100 { // Length-delimited
					validFields++
				}
			}
		}

		i++

		if validFields >= 2 {
			return true
		}
	}

	return false
}

func (d *GRPCDissectorV2) parseGRPCMessageV2(data []byte) map[string]interface{} {
	if len(data) < 5 {
		return nil
	}

	// Validate gRPC message format
	compressed := data[0]
	if compressed > 1 {
		return nil // Invalid compression flag
	}

	messageLength := binary.BigEndian.Uint32(data[1:5])
	remaining := len(data) - 5
	if remaining < 0 {
		return nil // Invalid length
	}
	if messageLength > uint32(remaining) { //nolint:gosec // remaining is checked to be non-negative
		return nil // Invalid length
	}

	result := make(map[string]interface{})
	result["compressed"] = compressed == 1
	result["message_length"] = messageLength

	if messageLength > 0 {
		result["data"] = data[5 : 5+messageLength]
	}

	return result
}

func (d *GRPCDissectorV2) parseHeadersSafely(data []byte) map[string]string {
	headers := make(map[string]string)

	// Common gRPC headers to look for
	headerNames := []string{
		":path", ":method", ":status", ":authority", ":scheme",
		"content-type", "grpc-status", "grpc-message", "grpc-timeout",
		"authorization", "x-api-key", "x-auth-token",
	}

	for _, name := range headerNames {
		nameBytes := []byte(name)
		idx := bytes.Index(data, nameBytes)
		if idx >= 0 {
			start := idx + len(nameBytes)
			// Skip separators
			for start < len(data) && (data[start] == ':' || data[start] == ' ') {
				start++
			}

			// Find value end
			end := start
			for end < len(data) && data[end] != 0 && data[end] != '\n' {
				end++
				// Prevent excessive scanning
				if end-start > 512 {
					break
				}
			}

			if end > start {
				headers[name] = string(data[start:end])
			}
		}
	}

	return headers
}

func (d *GRPCDissectorV2) checkHeadersForSecretsV2(headers map[string]string) []StreamVulnerability {
	var vulns []StreamVulnerability

	for name, value := range headers {
		// Skip empty values
		if len(value) == 0 {
			continue
		}

		// Normalize header name
		nameLower := strings.ToLower(name)

		// Check authorization patterns
		if nameLower == "authorization" || nameLower == "grpc-authorization" {
			if d.patterns.bearer.MatchString(value) {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "bearer_token",
					Severity:   "high",
					Confidence: 1.0,
					Evidence:   fmt.Sprintf("%s: Bearer ****", name),
					Location:   "gRPC header",
					Context:    "Bearer token in authorization header",
				})
			} else if d.patterns.oauth.MatchString(value) {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "oauth_token",
					Severity:   "high",
					Confidence: 0.95,
					Evidence:   fmt.Sprintf("%s: OAuth ****", name),
					Location:   "gRPC header",
					Context:    "OAuth token in authorization header",
				})
			} else if d.patterns.basicAuth.MatchString(value) {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "basic_auth",
					Severity:   "critical",
					Confidence: 1.0,
					Evidence:   fmt.Sprintf("%s: Basic ****", name),
					Location:   "gRPC header",
					Context:    "Basic authentication credentials",
				})
			}
		}

		// Check for API key headers
		if d.patterns.apiKey.MatchString(nameLower) && len(value) >= 20 {
			vulns = append(vulns, StreamVulnerability{
				Type:       "credential",
				Subtype:    "api_key",
				Severity:   "high",
				Confidence: 0.95,
				Evidence:   fmt.Sprintf("%s: %s", name, maskValue(value)),
				Location:   "gRPC header",
				Context:    fmt.Sprintf("API key in header '%s'", name),
			})
		}

		// Check for custom auth headers
		if d.patterns.customAuth.MatchString(nameLower) && len(value) >= 20 {
			vulns = append(vulns, StreamVulnerability{
				Type:       "credential",
				Subtype:    "custom_auth_token",
				Severity:   "high",
				Confidence: 0.9,
				Evidence:   fmt.Sprintf("%s: %s", name, maskValue(value)),
				Location:   "gRPC header",
				Context:    fmt.Sprintf("Custom auth token in header '%s'", name),
			})
		}

		// Check for JWT in any header value
		if d.patterns.jwt.MatchString(value) {
			vulns = append(vulns, StreamVulnerability{
				Type:       "credential",
				Subtype:    "jwt_token",
				Severity:   "high",
				Confidence: 0.95,
				Evidence:   fmt.Sprintf("%s: %s", name, maskJWT(value)),
				Location:   "gRPC header",
				Context:    fmt.Sprintf("JWT token in header '%s'", name),
			})
		}
	}

	return vulns
}

func (d *GRPCDissectorV2) checkPayloadForSecretsV2(payload []byte) []StreamVulnerability {
	var vulns []StreamVulnerability

	// Limit scan size for performance
	scanSize := min(len(payload), 10240)
	scanData := payload[:scanSize]

	// Convert to string for pattern matching
	payloadStr := string(scanData)

	// Look for JWT tokens with word boundaries
	jwtRegex := regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`)
	if matches := jwtRegex.FindAllString(payloadStr, 3); matches != nil {
		for _, match := range matches {
			vulns = append(vulns, StreamVulnerability{
				Type:       "credential",
				Subtype:    "jwt_in_payload",
				Severity:   "high",
				Confidence: 0.9,
				Evidence:   maskJWT(match),
				Location:   "gRPC payload",
				Context:    "JWT token in message payload",
			})
		}
	}

	// Look for API keys with context
	apiKeyRegex := regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*["']?([A-Za-z0-9_-]{20,})["']?`)
	if matches := apiKeyRegex.FindAllStringSubmatch(payloadStr, 3); matches != nil {
		for _, match := range matches {
			if len(match) >= 3 {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "api_key_in_payload",
					Severity:   "high",
					Confidence: 0.85,
					Evidence:   fmt.Sprintf("%s: %s", match[1], maskValue(match[2])),
					Location:   "gRPC payload",
					Context:    "API key in message payload",
				})
			}
		}
	}

	// Look for password patterns
	passwordRegex := regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["']?([^"'\s]{6,})["']?`)
	if matches := passwordRegex.FindAllStringSubmatch(payloadStr, 3); matches != nil {
		for range matches {
			vulns = append(vulns, StreamVulnerability{
				Type:       "credential",
				Subtype:    "password_in_payload",
				Severity:   "critical",
				Confidence: 0.9,
				Evidence:   "password: ***",
				Location:   "gRPC payload",
				Context:    "Password in message payload",
			})
		}
	}

	return vulns
}

func (d *GRPCDissectorV2) cacheAndReturn(key string, isGRPC bool, confidence float64) (bool, float64) {
	d.cache.mu.Lock()
	defer d.cache.mu.Unlock()

	// Limit cache size
	if len(d.cache.results) > 1000 {
		// Simple eviction - clear cache
		d.cache.results = make(map[string]cachedResult)
	}

	d.cache.results[key] = cachedResult{isGRPC: isGRPC, confidence: confidence}
	return isGRPC, confidence
}

// GetSessionID extracts session identifier from gRPC frame.
func (d *GRPCDissectorV2) GetSessionID(frame *Frame) (string, error) {
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
	// In a real implementation, we'd extract this from the network layer
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
	// In practice, we'd want to include connection info (IP:port)
	return fmt.Sprintf("grpc_stream_%d", streamID), nil
}

// min function is already defined in dissector_http.go
