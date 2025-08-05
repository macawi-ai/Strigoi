package probe

import (
	"fmt"
	"sync"
)

// CircularBuffer implements a thread-safe circular buffer that preserves complete events.
type CircularBuffer struct {
	data      []byte
	size      int
	head      int // Write position
	tail      int // Read position
	count     int // Number of bytes currently in buffer
	eventMark int // Mark the start of current event
	mu        sync.Mutex

	// Event boundary detection
	eventDelimiter []byte
	minEventSize   int
	maxEventSize   int
}

// NewCircularBuffer creates a new circular buffer with the specified size.
func NewCircularBuffer(size int, eventDelimiter []byte) *CircularBuffer {
	return &CircularBuffer{
		data:           make([]byte, size),
		size:           size,
		head:           0,
		tail:           0,
		count:          0,
		eventMark:      -1,
		eventDelimiter: eventDelimiter,
		minEventSize:   1,
		maxEventSize:   size / 4, // Max event is 1/4 of buffer
	}
}

// Write adds data to the buffer, preserving event boundaries.
func (cb *CircularBuffer) Write(p []byte) (n int, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if len(p) == 0 {
		return 0, nil
	}

	// If this write would overflow, make space
	if len(p) > cb.Available() {
		// Drop oldest complete events until we have space
		cb.makeSpace(len(p))
	}

	// Write data in a circular manner
	n = len(p)
	for i := 0; i < n; i++ {
		cb.data[cb.head] = p[i]
		cb.head = (cb.head + 1) % cb.size
		cb.count++

		// Track event boundaries
		if cb.eventDelimiter != nil && cb.isEventBoundary(i, p) {
			cb.eventMark = cb.head
		}
	}

	// Ensure we don't exceed buffer size
	if cb.count > cb.size {
		excess := cb.count - cb.size
		cb.tail = (cb.tail + excess) % cb.size
		cb.count = cb.size
	}

	return n, nil
}

// Read retrieves data from the buffer.
func (cb *CircularBuffer) Read(p []byte) (n int, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.count == 0 {
		return 0, nil
	}

	// Read up to len(p) bytes
	n = len(p)
	if n > cb.count {
		n = cb.count
	}

	for i := 0; i < n; i++ {
		p[i] = cb.data[cb.tail]
		cb.tail = (cb.tail + 1) % cb.size
		cb.count--
	}

	return n, nil
}

// ReadAll returns all data in the buffer without removing it.
func (cb *CircularBuffer) ReadAll() []byte {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.count == 0 {
		return nil
	}

	result := make([]byte, cb.count)

	if cb.tail < cb.head {
		// Data is contiguous
		copy(result, cb.data[cb.tail:cb.head])
	} else {
		// Data wraps around
		n := copy(result, cb.data[cb.tail:])
		copy(result[n:], cb.data[:cb.head])
	}

	return result
}

// Available returns the number of bytes available for writing.
func (cb *CircularBuffer) Available() int {
	return cb.size - cb.count
}

// Len returns the number of bytes currently in the buffer.
func (cb *CircularBuffer) Len() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.count
}

// Clear empties the buffer.
func (cb *CircularBuffer) Clear() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.head = 0
	cb.tail = 0
	cb.count = 0
	cb.eventMark = -1
}

// makeSpace ensures there's enough space by dropping oldest complete events.
func (cb *CircularBuffer) makeSpace(needed int) {
	if needed > cb.size {
		// Can't fit even if empty
		cb.Clear()
		return
	}

	// Find event boundaries and drop complete events from the tail
	droppedBytes := 0
	newTail := cb.tail

	for droppedBytes < needed && cb.count > 0 {
		// Look for next event boundary
		eventEnd := cb.findNextEventBoundary(newTail)
		if eventEnd == -1 {
			// No complete event found, drop everything up to eventMark
			if cb.eventMark != -1 && cb.eventMark != cb.head {
				bytesToDrop := cb.distanceBetween(newTail, cb.eventMark)
				droppedBytes += bytesToDrop
				newTail = cb.eventMark
			} else {
				// No events marked, drop half the buffer
				bytesToDrop := cb.count / 2
				droppedBytes += bytesToDrop
				newTail = (newTail + bytesToDrop) % cb.size
			}
		} else {
			// Drop complete event
			bytesToDrop := cb.distanceBetween(newTail, eventEnd)
			droppedBytes += bytesToDrop
			newTail = eventEnd
		}
	}

	// Update tail and count
	cb.tail = newTail
	cb.count -= droppedBytes
}

// isEventBoundary checks if the current position marks an event boundary.
func (cb *CircularBuffer) isEventBoundary(offset int, data []byte) bool {
	if len(cb.eventDelimiter) == 0 {
		return false
	}

	// Check if we have a delimiter at this position
	if offset+len(cb.eventDelimiter) > len(data) {
		return false
	}

	for i, b := range cb.eventDelimiter {
		if data[offset+i] != b {
			return false
		}
	}

	return true
}

// findNextEventBoundary finds the next event boundary from the given position.
func (cb *CircularBuffer) findNextEventBoundary(start int) int {
	if len(cb.eventDelimiter) == 0 {
		return -1
	}

	pos := start
	checked := 0
	delimIndex := 0

	for checked < cb.count {
		if cb.data[pos] == cb.eventDelimiter[delimIndex] {
			delimIndex++
			if delimIndex == len(cb.eventDelimiter) {
				// Found complete delimiter
				return (pos + 1) % cb.size
			}
		} else {
			delimIndex = 0
		}

		pos = (pos + 1) % cb.size
		checked++
	}

	return -1
}

// distanceBetween calculates the number of bytes between two positions.
func (cb *CircularBuffer) distanceBetween(from, to int) int {
	if from <= to {
		return to - from
	}
	return cb.size - from + to
}

// GetStats returns buffer statistics for monitoring.
func (cb *CircularBuffer) GetStats() map[string]interface{} {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return map[string]interface{}{
		"size":      cb.size,
		"count":     cb.count,
		"available": cb.size - cb.count,
		"head":      cb.head,
		"tail":      cb.tail,
		"usage":     fmt.Sprintf("%.1f%%", float64(cb.count)/float64(cb.size)*100),
	}
}

// SetEventDelimiter updates the event delimiter pattern.
func (cb *CircularBuffer) SetEventDelimiter(delimiter []byte) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.eventDelimiter = delimiter
}

// SetEventSizeLimits sets the minimum and maximum expected event sizes.
func (cb *CircularBuffer) SetEventSizeLimits(min, max int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if min > 0 {
		cb.minEventSize = min
	}
	if max > 0 && max <= cb.size {
		cb.maxEventSize = max
	}
}
