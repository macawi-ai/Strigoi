package probe

import (
	"strings"
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

// Write adds data to the buffer, preserving event boundaries.
func (b *StreamBuffer) Write(data []byte) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(data) == 0 {
		return 0
	}

	// Split data into complete events if delimiter is set
	if b.eventDelimiter != nil {
		return b.writeEvents(data)
	}

	// No delimiter - write as single unit
	return b.writeData(data)
}

// writeEvents splits data by delimiter and writes complete events.
func (b *StreamBuffer) writeEvents(data []byte) int {
	delimiter := string(b.eventDelimiter)
	dataStr := string(data)

	// Split by delimiter but keep delimiter with each event
	var events []string
	start := 0
	for {
		idx := strings.Index(dataStr[start:], delimiter)
		if idx == -1 {
			// No more delimiters - add remaining data if any
			if start < len(dataStr) {
				events = append(events, dataStr[start:])
			}
			break
		}
		// Include delimiter in the event
		end := start + idx + len(delimiter)
		events = append(events, dataStr[start:end])
		start = end
	}

	// Write each complete event
	totalWritten := 0
	for _, event := range events {
		eventBytes := []byte(event)
		written := b.writeData(eventBytes)
		totalWritten += written

		// Mark event boundary if this event ends with delimiter
		if strings.HasSuffix(event, delimiter) {
			b.addEventMark(b.writePos)
		}
	}

	return totalWritten
}

// writeData writes raw data to buffer, making room if needed.
func (b *StreamBuffer) writeData(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	// If data is larger than total buffer, truncate to most recent part
	if len(data) >= b.size {
		data = data[len(data)-(b.size-1):]
	}

	// Make room if needed - apply 2π regulation with minimal intervention
	available := b.getAvailableSpace()
	if len(data) > available {
		if len(data) >= b.size-1 {
			// Data is too large for buffer - clear completely (extreme case)
			b.readPos = 0
			b.writePos = 0
			b.eventMarks = nil
		} else {
			// Apply minimal 2π intervention - remove just enough oldest data
			needed := len(data) - available
			b.removeOldestData(needed)
		}
	}

	// Write the data
	written := 0
	for _, byte := range data {
		b.data[b.writePos] = byte
		b.writePos = (b.writePos + 1) % b.size
		written++
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

// getAvailableSpace calculates available space in the buffer.
func (b *StreamBuffer) getAvailableSpace() int {
	if b.writePos >= b.readPos {
		return b.size - (b.writePos - b.readPos) - 1
	} else {
		return b.readPos - b.writePos - 1
	}
}

// makeRoom frees up space by removing old events until we have enough space.
func (b *StreamBuffer) makeRoom(needed int) {
	for b.getAvailableSpace() < needed && b.readPos != b.writePos {
		// Find the next event boundary after readPos
		nextEvent := b.findNextEventBoundary()
		if nextEvent != -1 {
			b.readPos = nextEvent
			// Clean up old event marks
			b.cleanupEventMarks()
		} else {
			// No event boundaries, just advance readPos
			b.readPos = (b.readPos + 1) % b.size
		}
	}
}

// findNextEventBoundary finds the next event boundary after readPos.
func (b *StreamBuffer) findNextEventBoundary() int {
	if len(b.eventMarks) == 0 {
		return -1
	}

	// Find the closest event mark after readPos
	closestMark := -1
	for _, mark := range b.eventMarks {
		if b.isPositionAhead(mark, b.readPos) {
			if closestMark == -1 || !b.isPositionAhead(mark, closestMark) {
				closestMark = mark
			}
		}
	}

	if closestMark != -1 {
		return (closestMark + 1) % b.size
	}
	return -1
}

// removeOldestData removes oldest data to free up at least bytesToRemove space.
func (b *StreamBuffer) removeOldestData(bytesToRemove int) {
	if bytesToRemove <= 0 {
		return
	}

	// If we have event marks, try to remove complete events first
	if len(b.eventMarks) > 0 {
		for bytesToRemove > 0 && len(b.eventMarks) > 0 {
			// Find the oldest event mark
			oldestMark := b.eventMarks[0]
			for _, mark := range b.eventMarks {
				if !b.isPositionAhead(mark, oldestMark) {
					oldestMark = mark
				}
			}

			// Calculate how many bytes removing this event would free
			if b.isPositionAhead(oldestMark, b.readPos) {
				// Move readPos to after this event mark
				bytesFreed := b.calculateDistance(b.readPos, (oldestMark+1)%b.size)
				b.readPos = (oldestMark + 1) % b.size
				bytesToRemove -= bytesFreed
			} else {
				// This event is already past readPos, move byte by byte
				break
			}

			// Clean up event marks
			b.cleanupEventMarks()
		}
	}

	// If still need more space, remove byte by byte
	removed := 0
	for removed < bytesToRemove && b.readPos != b.writePos {
		b.readPos = (b.readPos + 1) % b.size
		removed++
	}

	// Final cleanup
	b.cleanupEventMarks()
}

// calculateDistance calculates distance between two positions in circular buffer.
func (b *StreamBuffer) calculateDistance(from, to int) int {
	if to >= from {
		return to - from
	} else {
		return b.size - from + to
	}
}

// cleanupEventMarks removes event marks that are now behind readPos.
func (b *StreamBuffer) cleanupEventMarks() {
	validMarks := b.eventMarks[:0]
	for _, mark := range b.eventMarks {
		if b.isPositionAhead(mark, b.readPos) || mark == b.readPos {
			validMarks = append(validMarks, mark)
		}
	}
	b.eventMarks = validMarks
}

// isPositionAhead checks if pos1 is ahead of pos2 in circular buffer.
func (b *StreamBuffer) isPositionAhead(pos1, pos2 int) bool {
	if pos1 == pos2 {
		return false
	}

	// Calculate distance considering circular nature
	if pos1 > pos2 {
		return (pos1 - pos2) <= (b.size / 2)
	} else {
		return (pos2 - pos1) > (b.size / 2)
	}
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
