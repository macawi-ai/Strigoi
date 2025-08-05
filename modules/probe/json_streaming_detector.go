package probe

// StreamingJSONDetector handles JSON boundary detection across multiple chunks
type StreamingJSONDetector struct {
	// State tracking
	stack    []byte // Track nesting with '{' and '['
	inString bool
	escaped  bool
}

// NewStreamingJSONDetector creates a new streaming JSON detector
func NewStreamingJSONDetector() *StreamingJSONDetector {
	return &StreamingJSONDetector{
		stack: make([]byte, 0, 32), // Pre-allocate for efficiency
	}
}

// DetectBoundary implements the BoundaryDetector interface for streaming JSON
func (d *StreamingJSONDetector) DetectBoundary(data []byte, offset int) (pos int, size int, found bool) {
	// Skip leading whitespace only if we're not already in a JSON object
	start := offset
	if len(d.stack) == 0 {
		for offset < len(data) && isWhitespace(data[offset]) {
			offset++
		}

		if offset >= len(data) {
			return -1, 0, false
		}

		// Must start with { or [
		if data[offset] != '{' && data[offset] != '[' {
			return -1, 0, false
		}
		start = offset
	}

	// Process from current position
	for offset < len(data) {
		ch := data[offset]

		if d.inString {
			if d.escaped {
				d.escaped = false
			} else if ch == '\\' {
				d.escaped = true
			} else if ch == '"' {
				d.inString = false
			}
		} else {
			switch ch {
			case '"':
				d.inString = true
				d.escaped = false
			case '{', '[':
				d.stack = append(d.stack, ch)
			case '}':
				if len(d.stack) > 0 && d.stack[len(d.stack)-1] == '{' {
					d.stack = d.stack[:len(d.stack)-1]
				} else {
					// Mismatched brackets - invalid JSON
					d.Reset()
					return -1, 0, false
				}
			case ']':
				if len(d.stack) > 0 && d.stack[len(d.stack)-1] == '[' {
					d.stack = d.stack[:len(d.stack)-1]
				} else {
					// Mismatched brackets - invalid JSON
					d.Reset()
					return -1, 0, false
				}
			}
		}

		offset++

		// Check if we have a complete JSON object
		if len(d.stack) == 0 && !d.inString {
			// Skip trailing whitespace
			endPos := offset
			for offset < len(data) && isWhitespace(data[offset]) {
				offset++
			}

			// We found a complete JSON object
			d.Reset() // Reset for next detection
			return offset, endPos - start, true
		}
	}

	// Incomplete JSON - need more data
	return -1, 0, false
}

// Reset clears the detector state for a new JSON object
func (d *StreamingJSONDetector) Reset() {
	d.stack = d.stack[:0]
	d.inString = false
	d.escaped = false
}

// MinMessageSize returns minimum valid JSON size
func (d *StreamingJSONDetector) MinMessageSize() int {
	return 2 // "{}" or "[]"
}

// MaxMessageSize returns maximum expected JSON size
func (d *StreamingJSONDetector) MaxMessageSize() int {
	return 10 * 1024 * 1024 // 10MB
}

// Name returns the detector name
func (d *StreamingJSONDetector) Name() string {
	return "JSON-Streaming"
}

// isWhitespace checks if byte is JSON whitespace
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// EnhancedJSONBoundaryDetector wraps streaming detector with state preservation
type EnhancedJSONBoundaryDetector struct {
	detector      *StreamingJSONDetector
	buffer        []byte
	maxBufferSize int
}

// NewEnhancedJSONBoundaryDetector creates a stateful JSON detector
func NewEnhancedJSONBoundaryDetector() *EnhancedJSONBoundaryDetector {
	return &EnhancedJSONBoundaryDetector{
		detector:      NewStreamingJSONDetector(),
		buffer:        make([]byte, 0, 4096),
		maxBufferSize: 1024 * 1024, // 1MB max buffer
	}
}

// DetectBoundary handles chunked JSON detection with buffering
func (e *EnhancedJSONBoundaryDetector) DetectBoundary(data []byte, offset int) (pos int, size int, found bool) {
	// The protocol-aware buffer does its own accumulation of partial messages.
	// When we're called with accumulated data, we shouldn't buffer again.
	// We detect this by checking if we've been buffering but suddenly get
	// data that's significantly larger than what we'd expect from a chunk.

	// If offset > 0, we're definitely being called by protocol-aware buffer
	if offset > 0 {
		e.buffer = e.buffer[:0] // Clear any buffer
		e.detector.Reset()
		return e.detector.DetectBoundary(data, offset)
	}

	// Check if this looks like accumulated data (much larger than our buffer)
	if len(e.buffer) > 0 && len(data) > len(e.buffer)*2 {
		// This is accumulated data from protocol-aware buffer
		e.buffer = e.buffer[:0]
		e.detector.Reset()
		return e.detector.DetectBoundary(data, offset)
	}

	// This is a chunk - use buffering logic for standalone use
	if len(e.buffer) > 0 {
		// Combine buffer with new data
		combined := make([]byte, len(e.buffer)+len(data))
		copy(combined, e.buffer)
		copy(combined[len(e.buffer):], data)

		// Try to detect in combined data
		e.detector.Reset()
		pos, size, found = e.detector.DetectBoundary(combined, 0)

		if found {
			// Clear buffer since we found complete JSON
			e.buffer = e.buffer[:0]
			return pos, size, true
		}

		// Still incomplete - update buffer
		if len(combined) > e.maxBufferSize {
			e.buffer = e.buffer[:0]
			e.detector.Reset()
			return -1, 0, false
		}

		e.buffer = e.buffer[:0]
		e.buffer = append(e.buffer, combined...)
		return -1, 0, false
	}

	// No buffered data - try direct detection first
	e.detector.Reset()
	pos, size, found = e.detector.DetectBoundary(data, 0)

	if !found && pos == -1 {
		// Incomplete JSON - buffer the data
		if len(data) > 0 && len(data) < e.maxBufferSize {
			e.buffer = append(e.buffer[:0], data...)
		}
	}

	return pos, size, found
}

// MinMessageSize returns minimum valid JSON size
func (e *EnhancedJSONBoundaryDetector) MinMessageSize() int {
	return e.detector.MinMessageSize()
}

// MaxMessageSize returns maximum expected JSON size
func (e *EnhancedJSONBoundaryDetector) MaxMessageSize() int {
	return e.detector.MaxMessageSize()
}

// Name returns the detector name
func (e *EnhancedJSONBoundaryDetector) Name() string {
	return "JSON-Enhanced"
}
