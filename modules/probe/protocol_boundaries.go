package probe

import (
	"bytes"
	"encoding/binary"
	"sync"
)

// ProtocolBoundaryDetector identifies protocol-specific message boundaries.
type ProtocolBoundaryDetector struct {
	detectors map[string]BoundaryDetector
	order     []string // Detection order for auto-detection
	mu        sync.RWMutex
}

// BoundaryDetector defines the interface for protocol-specific boundary detection.
type BoundaryDetector interface {
	// DetectBoundary scans buffer for next message boundary
	// Returns: boundary position, message size, found
	DetectBoundary(data []byte, offset int) (pos int, size int, found bool)

	// MinMessageSize returns minimum valid message size
	MinMessageSize() int

	// MaxMessageSize returns maximum expected message size
	MaxMessageSize() int

	// Name returns the protocol name
	Name() string
}

// NewProtocolBoundaryDetector creates a detector with common protocols.
func NewProtocolBoundaryDetector() *ProtocolBoundaryDetector {
	detector := &ProtocolBoundaryDetector{
		detectors: make(map[string]BoundaryDetector),
		order:     []string{},
	}

	// Register default detectors - order matters for auto-detection
	// More specific protocols first
	detector.Register("http", NewHTTPBoundaryDetector())
	detector.Register("grpc", NewGRPCBoundaryDetector())
	detector.Register("websocket", NewWebSocketBoundaryDetector())
	detector.Register("json", NewEnhancedJSONBoundaryDetector())
	detector.Register("line", NewLineBoundaryDetector()) // Most generic last

	return detector
}

// Register adds a new protocol detector.
func (p *ProtocolBoundaryDetector) Register(name string, detector BoundaryDetector) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.detectors[name] = detector
	p.order = append(p.order, name)
}

// DetectProtocol attempts to identify the protocol and find boundaries.
func (p *ProtocolBoundaryDetector) DetectProtocol(data []byte) (protocol string, boundary int, size int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Try each detector in order
	for _, name := range p.order {
		detector := p.detectors[name]
		if pos, msgSize, found := detector.DetectBoundary(data, 0); found {
			return name, pos, msgSize
		}
	}

	return "", -1, 0
}

// HTTPBoundaryDetector detects HTTP message boundaries.
type HTTPBoundaryDetector struct {
	requestPattern  []byte
	responsePattern []byte
	headerEnd       []byte
}

func NewHTTPBoundaryDetector() *HTTPBoundaryDetector {
	return &HTTPBoundaryDetector{
		requestPattern:  []byte("HTTP/"),
		responsePattern: []byte("HTTP/"),
		headerEnd:       []byte("\r\n\r\n"),
	}
}

func (h *HTTPBoundaryDetector) DetectBoundary(data []byte, offset int) (int, int, bool) {
	// Look for HTTP request or response
	if len(data) < offset+10 {
		return -1, 0, false
	}

	// Check for HTTP methods
	methods := [][]byte{
		[]byte("GET "), []byte("POST "), []byte("PUT "),
		[]byte("DELETE "), []byte("HEAD "), []byte("OPTIONS "),
		[]byte("PATCH "), []byte("CONNECT "), []byte("TRACE "),
	}

	isHTTP := false
	start := offset

	// Check if it's a request
	for _, method := range methods {
		if bytes.HasPrefix(data[offset:], method) {
			isHTTP = true
			break
		}
	}

	// Check if it's a response
	if !isHTTP && bytes.HasPrefix(data[offset:], h.responsePattern) {
		isHTTP = true
	}

	if !isHTTP {
		return -1, 0, false
	}

	// Find header end
	headerEndPos := bytes.Index(data[offset:], h.headerEnd)
	if headerEndPos == -1 {
		return -1, 0, false // Incomplete headers
	}

	headerEndPos += offset + len(h.headerEnd)

	// Parse Content-Length if present
	contentLength := h.parseContentLength(data[offset:headerEndPos])

	if contentLength >= 0 {
		// Fixed content length
		totalSize := (headerEndPos - start) + contentLength
		return headerEndPos + contentLength, totalSize, true
	}

	// Check for chunked encoding
	if h.isChunkedEncoding(data[offset:headerEndPos]) {
		// Find end of chunked body
		endPos := h.findChunkedEnd(data, headerEndPos)
		if endPos > 0 {
			return endPos, endPos - start, true
		}
		return -1, 0, false // Incomplete chunked body
	}

	// No body (GET, HEAD, etc)
	return headerEndPos, headerEndPos - start, true
}

func (h *HTTPBoundaryDetector) parseContentLength(headers []byte) int {
	pattern := []byte("Content-Length: ")
	idx := bytes.Index(headers, pattern)
	if idx == -1 {
		pattern = []byte("content-length: ")
		idx = bytes.Index(headers, pattern)
	}

	if idx == -1 {
		return -1
	}

	start := idx + len(pattern)
	end := bytes.IndexByte(headers[start:], '\r')
	if end == -1 {
		return -1
	}

	length := 0
	for i := start; i < start+end; i++ {
		if headers[i] >= '0' && headers[i] <= '9' {
			length = length*10 + int(headers[i]-'0')
		}
	}

	return length
}

func (h *HTTPBoundaryDetector) isChunkedEncoding(headers []byte) bool {
	return bytes.Contains(headers, []byte("Transfer-Encoding: chunked")) ||
		bytes.Contains(headers, []byte("transfer-encoding: chunked"))
}

func (h *HTTPBoundaryDetector) findChunkedEnd(data []byte, offset int) int {
	// Look for final chunk (0\r\n\r\n)
	finalChunk := []byte("0\r\n\r\n")
	idx := bytes.Index(data[offset:], finalChunk)
	if idx != -1 {
		return offset + idx + len(finalChunk)
	}
	return -1
}

func (h *HTTPBoundaryDetector) MinMessageSize() int { return 16 }               // "GET / HTTP/1.0\r\n\r\n"
func (h *HTTPBoundaryDetector) MaxMessageSize() int { return 10 * 1024 * 1024 } // 10MB
func (h *HTTPBoundaryDetector) Name() string        { return "HTTP" }

// GRPCBoundaryDetector detects gRPC message boundaries.
type GRPCBoundaryDetector struct{}

func NewGRPCBoundaryDetector() *GRPCBoundaryDetector {
	return &GRPCBoundaryDetector{}
}

func (g *GRPCBoundaryDetector) DetectBoundary(data []byte, offset int) (int, int, bool) {
	// gRPC uses length-prefixed messages
	// Format: [compression_flag:1][length:4][data:length]

	if len(data) < offset+5 {
		return -1, 0, false // Need at least 5 bytes for header
	}

	// Check if this looks like HTTP/2 (gRPC transport)
	if offset == 0 && len(data) >= 24 {
		// HTTP/2 preface: "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
		if bytes.HasPrefix(data, []byte("PRI * HTTP/2.0")) {
			return -1, 0, false // This is HTTP/2 negotiation, not gRPC data
		}
	}

	// Read gRPC message header
	compressionFlag := data[offset]
	if compressionFlag > 1 {
		return -1, 0, false // Invalid compression flag
	}

	// Read 4-byte big-endian length
	messageLength := binary.BigEndian.Uint32(data[offset+1 : offset+5])

	// Sanity check
	if messageLength > 4*1024*1024 { // 4MB max
		return -1, 0, false
	}

	totalSize := 5 + int(messageLength)
	endPos := offset + totalSize

	if len(data) < endPos {
		return -1, 0, false // Incomplete message
	}

	return endPos, totalSize, true
}

func (g *GRPCBoundaryDetector) MinMessageSize() int { return 5 }
func (g *GRPCBoundaryDetector) MaxMessageSize() int { return 4 * 1024 * 1024 }
func (g *GRPCBoundaryDetector) Name() string        { return "gRPC" }

// WebSocketBoundaryDetector detects WebSocket frame boundaries.
type WebSocketBoundaryDetector struct{}

func NewWebSocketBoundaryDetector() *WebSocketBoundaryDetector {
	return &WebSocketBoundaryDetector{}
}

func (w *WebSocketBoundaryDetector) DetectBoundary(data []byte, offset int) (int, int, bool) {
	if len(data) < offset+2 {
		return -1, 0, false // Need at least 2 bytes for header
	}

	// Parse WebSocket frame header
	b0 := data[offset]
	b1 := data[offset+1]

	// Check FIN bit and opcode
	// fin := (b0 & 0x80) != 0
	opcode := b0 & 0x0F

	// Valid opcodes: 0-2 (data), 8-10 (control)
	if opcode > 10 || (opcode > 2 && opcode < 8) {
		return -1, 0, false
	}

	// Check mask bit (client->server must be masked)
	masked := (b1 & 0x80) != 0
	payloadLen := int(b1 & 0x7F)

	headerSize := 2
	if masked {
		headerSize += 4 // 4-byte masking key
	}

	// Extended payload length
	if payloadLen == 126 {
		if len(data) < offset+headerSize+2 {
			return -1, 0, false
		}
		payloadLen = int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		headerSize += 2
	} else if payloadLen == 127 {
		if len(data) < offset+headerSize+8 {
			return -1, 0, false
		}
		payloadLen64 := binary.BigEndian.Uint64(data[offset+2 : offset+10])
		if payloadLen64 > uint64(w.MaxMessageSize()) {
			return -1, 0, false
		}
		payloadLen = int(payloadLen64)
		headerSize += 8
	}

	totalSize := headerSize + payloadLen
	endPos := offset + totalSize

	if len(data) < endPos {
		return -1, 0, false // Incomplete frame
	}

	return endPos, totalSize, true
}

func (w *WebSocketBoundaryDetector) MinMessageSize() int { return 2 }
func (w *WebSocketBoundaryDetector) MaxMessageSize() int { return 64 * 1024 * 1024 }
func (w *WebSocketBoundaryDetector) Name() string        { return "WebSocket" }

// JSONBoundaryDetector detects complete JSON objects/arrays.
type JSONBoundaryDetector struct{}

func NewJSONBoundaryDetector() *JSONBoundaryDetector {
	return &JSONBoundaryDetector{}
}

func (j *JSONBoundaryDetector) DetectBoundary(data []byte, offset int) (int, int, bool) {
	// Skip whitespace
	for offset < len(data) && (data[offset] == ' ' || data[offset] == '\t' ||
		data[offset] == '\n' || data[offset] == '\r') {
		offset++
	}

	if offset >= len(data) {
		return -1, 0, false
	}

	start := offset

	// Check start of JSON
	if data[offset] != '{' && data[offset] != '[' {
		return -1, 0, false
	}

	// Track nesting
	stack := []byte{data[offset]}
	offset++
	inString := false
	escaped := false

	for offset < len(data) && len(stack) > 0 {
		ch := data[offset]

		if inString {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
		} else {
			switch ch {
			case '"':
				inString = true
			case '{', '[':
				stack = append(stack, ch)
			case '}':
				if len(stack) > 0 && stack[len(stack)-1] == '{' {
					stack = stack[:len(stack)-1]
				} else {
					return -1, 0, false // Mismatched brackets
				}
			case ']':
				if len(stack) > 0 && stack[len(stack)-1] == '[' {
					stack = stack[:len(stack)-1]
				} else {
					return -1, 0, false // Mismatched brackets
				}
			}
		}

		offset++
	}

	if len(stack) == 0 {
		return offset, offset - start, true
	}

	return -1, 0, false // Incomplete JSON
}

func (j *JSONBoundaryDetector) MinMessageSize() int { return 2 } // "{}"
func (j *JSONBoundaryDetector) MaxMessageSize() int { return 10 * 1024 * 1024 }
func (j *JSONBoundaryDetector) Name() string        { return "JSON" }

// LineBoundaryDetector detects newline-delimited messages.
type LineBoundaryDetector struct {
	delimiter []byte
}

func NewLineBoundaryDetector() *LineBoundaryDetector {
	return &LineBoundaryDetector{
		delimiter: []byte("\n"),
	}
}

func (l *LineBoundaryDetector) DetectBoundary(data []byte, offset int) (int, int, bool) {
	idx := bytes.Index(data[offset:], l.delimiter)
	if idx == -1 {
		return -1, 0, false
	}

	endPos := offset + idx + len(l.delimiter)
	return endPos, idx + len(l.delimiter), true
}

func (l *LineBoundaryDetector) MinMessageSize() int { return 1 }
func (l *LineBoundaryDetector) MaxMessageSize() int { return 1024 * 1024 } // 1MB lines
func (l *LineBoundaryDetector) Name() string        { return "Line" }
