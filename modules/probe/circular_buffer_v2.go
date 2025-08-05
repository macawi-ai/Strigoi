package probe

import (
	"errors"
	"sync/atomic"
	"time"
	"unsafe"
)

// LockFreeCircularBuffer implements a high-performance lock-free circular buffer
// optimized for stream capture with event boundary preservation.
type LockFreeCircularBuffer struct {
	// Buffer data and metadata
	data []byte
	size uint64
	mask uint64 // For fast modulo operations (size must be power of 2)

	// Atomic indices for lock-free operations
	writeIndex atomic.Uint64 // Current write position
	readIndex  atomic.Uint64 // Current read position
	reserved   atomic.Uint64 // Reserved space for ongoing writes

	// Event tracking
	eventDelimiter []byte
	maxEventSize   int

	// Channel-based event notification
	events  chan Event
	done    chan struct{}
	writers atomic.Int32 // Track active writers
	closed  atomic.Bool

	// Stats
	dropped    atomic.Uint64
	written    atomic.Uint64
	eventsSent atomic.Uint64
}

// Event represents a complete event extracted from the buffer.
type Event struct {
	Data      []byte
	Timestamp time.Time
	Offset    uint64 // Position in stream
}

// NewLockFreeCircularBuffer creates a new lock-free circular buffer.
// Size must be a power of 2 for optimal performance.
func NewLockFreeCircularBuffer(size int, delimiter []byte) (*LockFreeCircularBuffer, error) {
	// Ensure size is power of 2
	if size&(size-1) != 0 {
		// Round up to next power of 2
		v := uint64(size)
		v--
		v |= v >> 1
		v |= v >> 2
		v |= v >> 4
		v |= v >> 8
		v |= v >> 16
		v |= v >> 32
		v++
		size = int(v)
	}

	if size < 4096 {
		size = 4096 // Minimum 4KB
	}

	cb := &LockFreeCircularBuffer{
		data:           make([]byte, size),
		size:           uint64(size),
		mask:           uint64(size - 1),
		eventDelimiter: delimiter,
		maxEventSize:   size / 4, // Default to 1/4 of buffer
		events:         make(chan Event, 1024),
		done:           make(chan struct{}),
	}

	// Start event processor
	go cb.processEvents()

	return cb, nil
}

// Write adds data to the buffer using lock-free operations.
func (cb *LockFreeCircularBuffer) Write(p []byte) (int, error) {
	if cb.closed.Load() {
		return 0, errors.New("buffer closed")
	}

	cb.writers.Add(1)
	defer cb.writers.Add(-1)

	n := len(p)
	if n == 0 {
		return 0, nil
	}

	// Reserve space atomically
	writePos := cb.reserve(uint64(n))
	if writePos == ^uint64(0) {
		// Buffer full, drop data
		cb.dropped.Add(uint64(n))
		return 0, errors.New("buffer full")
	}

	// Copy data to buffer (lock-free write)
	cb.copyToBuffer(p, writePos)

	// Update write index to make data visible
	cb.commit(writePos, uint64(n))

	cb.written.Add(uint64(n))
	return n, nil
}

// reserve atomically reserves space in the buffer.
func (cb *LockFreeCircularBuffer) reserve(size uint64) uint64 {
	for {
		writeIdx := cb.writeIndex.Load()
		readIdx := cb.readIndex.Load()

		// Calculate available space
		available := cb.size - (writeIdx - readIdx)
		if available < size {
			return ^uint64(0) // Buffer full
		}

		// Try to reserve space
		if cb.reserved.CompareAndSwap(writeIdx, writeIdx+size) {
			return writeIdx
		}
		// Retry if another writer intervened
	}
}

// commit makes written data visible to readers.
func (cb *LockFreeCircularBuffer) commit(pos, size uint64) {
	// Spin until it's our turn to commit
	for {
		current := cb.writeIndex.Load()
		if current == pos {
			cb.writeIndex.Store(pos + size)
			return
		}
		// Small pause to reduce CPU usage
		spinWait()
	}
}

// copyToBuffer copies data to the circular buffer.
func (cb *LockFreeCircularBuffer) copyToBuffer(data []byte, pos uint64) {
	n := uint64(len(data))
	start := pos & cb.mask

	if start+n <= cb.size {
		// Single contiguous write
		copy(cb.data[start:], data)
	} else {
		// Wrap-around write
		firstPart := cb.size - start
		copy(cb.data[start:], data[:firstPart])
		copy(cb.data[0:], data[firstPart:])
	}
}

// processEvents continuously scans for complete events.
func (cb *LockFreeCircularBuffer) processEvents() {
	var eventStart uint64
	var inEvent bool
	scanBuffer := make([]byte, 4096) // Reusable scan buffer

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-cb.done:
			close(cb.events)
			return
		case <-ticker.C:
			cb.scanForEvents(scanBuffer, &eventStart, &inEvent)
		}
	}
}

// scanForEvents looks for complete events in the buffer.
func (cb *LockFreeCircularBuffer) scanForEvents(scanBuf []byte, eventStart *uint64, inEvent *bool) {
	readIdx := cb.readIndex.Load()
	writeIdx := cb.writeIndex.Load()

	if readIdx >= writeIdx {
		return // No new data
	}

	// Limit scan size
	available := writeIdx - readIdx
	scanSize := uint64(len(scanBuf))
	if available < scanSize {
		scanSize = available
	}

	// Copy data for scanning (snapshot for consistency)
	start := readIdx & cb.mask
	if start+scanSize <= cb.size {
		copy(scanBuf[:scanSize], cb.data[start:])
	} else {
		firstPart := cb.size - start
		copy(scanBuf[:firstPart], cb.data[start:])
		copy(scanBuf[firstPart:scanSize], cb.data[0:])
	}

	// If not in event, we're looking for start of data
	if !*inEvent {
		*eventStart = readIdx
		*inEvent = true
	}

	// Scan for events
	var i uint64
	for i = 0; i < scanSize; i++ {
		if cb.matchesDelimiter(scanBuf, i) {
			// Found delimiter - extract event
			eventEnd := readIdx + i
			if eventEnd > *eventStart {
				cb.extractAndSendEvent(*eventStart, eventEnd)
			}
			// Skip delimiter and prepare for next event
			i += uint64(len(cb.eventDelimiter))
			*eventStart = readIdx + i
			if i >= scanSize {
				*inEvent = false
			}
		}
	}

	// Update read index to consume processed data
	if !*inEvent || (readIdx+i) > *eventStart {
		// Only update if we're not in the middle of an event
		newReadIdx := *eventStart
		if !*inEvent {
			newReadIdx = readIdx + i
		}
		cb.readIndex.Store(newReadIdx)
	}
}

// matchesDelimiter checks if delimiter appears at position.
func (cb *LockFreeCircularBuffer) matchesDelimiter(buf []byte, pos uint64) bool {
	if len(cb.eventDelimiter) == 0 {
		return false
	}

	if pos+uint64(len(cb.eventDelimiter)) > uint64(len(buf)) {
		return false
	}

	for i, b := range cb.eventDelimiter {
		if buf[pos+uint64(i)] != b {
			return false
		}
	}
	return true
}

// extractAndSendEvent extracts an event and sends it through the channel.
func (cb *LockFreeCircularBuffer) extractAndSendEvent(start, end uint64) {
	size := end - start
	if size == 0 || size > uint64(cb.maxEventSize) {
		return
	}

	eventData := make([]byte, size)
	startPos := start & cb.mask

	if startPos+size <= cb.size {
		copy(eventData, cb.data[startPos:startPos+size])
	} else {
		firstPart := cb.size - startPos
		copy(eventData[:firstPart], cb.data[startPos:])
		copy(eventData[firstPart:], cb.data[:size-firstPart])
	}

	event := Event{
		Data:      eventData,
		Timestamp: time.Now(),
		Offset:    start,
	}

	select {
	case cb.events <- event:
		cb.eventsSent.Add(1)
	default:
		// Channel full, drop event
		cb.dropped.Add(1)
	}
}

// Events returns the channel for receiving complete events.
func (cb *LockFreeCircularBuffer) Events() <-chan Event {
	return cb.events
}

// ReadAll returns all available data (for compatibility).
func (cb *LockFreeCircularBuffer) ReadAll() []byte {
	readIdx := cb.readIndex.Load()
	writeIdx := cb.writeIndex.Load()

	if readIdx >= writeIdx {
		return nil
	}

	size := writeIdx - readIdx
	result := make([]byte, size)

	start := readIdx & cb.mask
	if start+size <= cb.size {
		copy(result, cb.data[start:])
	} else {
		firstPart := cb.size - start
		copy(result[:firstPart], cb.data[start:])
		copy(result[firstPart:], cb.data[:size-firstPart])
	}

	cb.readIndex.Store(writeIdx)
	return result
}

// Close gracefully shuts down the buffer.
func (cb *LockFreeCircularBuffer) Close() error {
	if !cb.closed.CompareAndSwap(false, true) {
		return errors.New("already closed")
	}

	// Wait for writers to finish
	for cb.writers.Load() > 0 {
		time.Sleep(time.Millisecond)
	}

	close(cb.done)
	return nil
}

// Stats returns buffer statistics.
func (cb *LockFreeCircularBuffer) Stats() map[string]interface{} {
	writeIdx := cb.writeIndex.Load()
	readIdx := cb.readIndex.Load()
	used := writeIdx - readIdx

	return map[string]interface{}{
		"size":        cb.size,
		"used":        used,
		"available":   cb.size - used,
		"usage_pct":   float64(used) / float64(cb.size) * 100,
		"written":     cb.written.Load(),
		"dropped":     cb.dropped.Load(),
		"events_sent": cb.eventsSent.Load(),
		"write_idx":   writeIdx,
		"read_idx":    readIdx,
	}
}

// spinWait provides a CPU-friendly pause for spin locks.
func spinWait() {
	// This is a simplified version. In production, you might want
	// to use runtime.Gosched() or more sophisticated backoff
	for i := 0; i < 10; i++ {
		_ = i
	}
}

// Ensure we're using atomic memory access patterns correctly
var _ = unsafe.Sizeof(LockFreeCircularBuffer{})
