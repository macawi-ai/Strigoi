package probe

import (
	"sync"
	"time"
)

// StreamBuffer provides a thread-safe circular buffer for stream data.
type StreamBuffer struct {
	data     []byte
	size     int
	writePos int
	readPos  int
	mu       sync.Mutex
}

// NewStreamBuffer creates a new stream buffer with the specified size.
func NewStreamBuffer(size int) *StreamBuffer {
	return &StreamBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Write adds data to the buffer.
func (b *StreamBuffer) Write(data []byte) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	written := 0
	for _, byte := range data {
		b.data[b.writePos] = byte
		b.writePos = (b.writePos + 1) % b.size
		written++

		// Handle overflow by moving read position
		if b.writePos == b.readPos {
			b.readPos = (b.readPos + 1) % b.size
		}
	}
	return written
}

// Read retrieves data from the buffer.
func (b *StreamBuffer) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.readPos == b.writePos {
		return 0, nil // Empty buffer
	}

	n := 0
	for n < len(p) && b.readPos != b.writePos {
		p[n] = b.data[b.readPos]
		b.readPos = (b.readPos + 1) % b.size
		n++
	}
	return n, nil
}

// Available returns the number of bytes available to read.
func (b *StreamBuffer) Available() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.writePos >= b.readPos {
		return b.writePos - b.readPos
	}
	return b.size - b.readPos + b.writePos
}

// StreamData holds captured stream data.
type StreamData struct {
	Stdin  []byte
	Stdout []byte
	Stderr []byte
}

// Credential represents a detected credential.
type Credential struct {
	Type       string  // password, api_key, token, etc.
	Value      string  // Original value (for internal use)
	Redacted   string  // Redacted version for display
	Confidence float64 // 0.0 to 1.0
	Severity   string  // critical, high, medium, low
	Timestamp  time.Time
}

// Dissector interface for protocol analysis.
type Dissector interface {
	// Identify checks if the data matches this dissector's protocol.
	Identify(data []byte) (bool, float64)
	// Dissect parses the data into a structured frame.
	Dissect(data []byte) (*Frame, error)
	// FindVulnerabilities analyzes a frame for security issues.
	FindVulnerabilities(frame *Frame) []StreamVulnerability
}

// Frame represents parsed protocol data.
type Frame struct {
	Protocol string                 `json:"protocol"`
	Fields   map[string]interface{} `json:"fields"`
	Raw      []byte                 `json:"-"`
}
