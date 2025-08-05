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

	// Event boundary preservation
	eventDelimiter []byte
	eventMarks     []int // Positions of event boundaries
	maxEvents      int   // Maximum number of events to track
}

// NewStreamBuffer creates a new stream buffer with the specified size.
func NewStreamBuffer(size int) *StreamBuffer {
	return &StreamBuffer{
		data:       make([]byte, size),
		size:       size,
		eventMarks: make([]int, 0, 100),
		maxEvents:  100,
	}
}

// SetEventDelimiter sets the delimiter for event boundary detection.
func (b *StreamBuffer) SetEventDelimiter(delimiter []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.eventDelimiter = delimiter
}

// Write adds data to the buffer.
func (b *StreamBuffer) Write(data []byte) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	written := 0
	for i, byte := range data {
		b.data[b.writePos] = byte
		b.writePos = (b.writePos + 1) % b.size
		written++

		// Check for event delimiter at the end position
		if b.eventDelimiter != nil && i >= len(b.eventDelimiter)-1 && b.checkEventBoundary(i-len(b.eventDelimiter)+1, data) {
			b.addEventMark(b.writePos)
		}

		// Handle overflow by moving read position
		if b.writePos == b.readPos {
			// When overflowing, try to preserve complete events
			b.handleOverflow()
		}
	}
	return written
}

// checkEventBoundary checks if current position in data is an event boundary.
func (b *StreamBuffer) checkEventBoundary(offset int, data []byte) bool {
	if len(b.eventDelimiter) == 0 || offset+len(b.eventDelimiter) > len(data) {
		return false
	}

	for i, delim := range b.eventDelimiter {
		if data[offset+i] != delim {
			return false
		}
	}
	return true
}

// addEventMark records the position of an event boundary.
func (b *StreamBuffer) addEventMark(position int) {
	// Remove old marks that are behind readPos
	validMarks := b.eventMarks[:0]
	for _, mark := range b.eventMarks {
		if b.isPositionAhead(mark, b.readPos) {
			validMarks = append(validMarks, mark)
		}
	}
	b.eventMarks = validMarks

	// Add new mark
	if len(b.eventMarks) < b.maxEvents {
		b.eventMarks = append(b.eventMarks, position)
	}
}

// handleOverflow manages buffer overflow by preserving complete events.
func (b *StreamBuffer) handleOverflow() {
	if len(b.eventMarks) > 0 {
		// Find the first event mark after current readPos
		for _, mark := range b.eventMarks {
			if b.isPositionAhead(mark, b.readPos) {
				b.readPos = (mark + 1) % b.size
				return
			}
		}
	}
	// No event marks, just advance read position
	b.readPos = (b.readPos + 1) % b.size
}

// isPositionAhead checks if pos1 is ahead of pos2 in circular buffer.
func (b *StreamBuffer) isPositionAhead(pos1, _ int) bool {
	if b.writePos > b.readPos {
		// Normal case: no wrap
		return pos1 >= b.readPos && pos1 < b.writePos
	} else if b.writePos < b.readPos {
		// Wrapped case
		return pos1 >= b.readPos || pos1 < b.writePos
	}
	// Empty buffer
	return false
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

// ReadAll reads all available data and resets the buffer.
func (b *StreamBuffer) ReadAll() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.readPos == b.writePos {
		return nil
	}

	var result []byte
	if b.writePos > b.readPos {
		result = make([]byte, b.writePos-b.readPos)
		copy(result, b.data[b.readPos:b.writePos])
	} else {
		// Wrapped around
		result = make([]byte, b.size-b.readPos+b.writePos)
		copy(result, b.data[b.readPos:])
		copy(result[b.size-b.readPos:], b.data[:b.writePos])
	}

	b.readPos = b.writePos
	return result
}

// StreamData holds captured stream data.
type StreamData struct {
	Timestamp time.Time
	Stdin     []byte
	Stdout    []byte
	Stderr    []byte
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
	// GetSessionID extracts the session identifier from a frame.
	GetSessionID(frame *Frame) (string, error)
}

// Frame represents parsed protocol data.
type Frame struct {
	Protocol string                 `json:"protocol"`
	Fields   map[string]interface{} `json:"fields"`
	Raw      []byte                 `json:"-"`
}
