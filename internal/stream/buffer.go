package stream

import (
	"errors"
	"io"
	"sync"
)

// ringBuffer implements a thread-safe circular buffer
type ringBuffer struct {
	data     []byte
	size     int
	capacity int
	head     int
	tail     int
	mu       sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the specified capacity
func NewRingBuffer(capacity int) RingBuffer {
	return &ringBuffer{
		data:     make([]byte, capacity),
		capacity: capacity,
	}
}

// Write adds data to the buffer
func (rb *ringBuffer) Write(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	if len(p) == 0 {
		return 0, nil
	}
	
	written := 0
	for _, b := range p {
		// If buffer is full, overwrite oldest data
		if rb.size == rb.capacity {
			rb.head = (rb.head + 1) % rb.capacity
		} else {
			rb.size++
		}
		
		rb.data[rb.tail] = b
		rb.tail = (rb.tail + 1) % rb.capacity
		written++
	}
	
	return written, nil
}

// Read reads data from the buffer
func (rb *ringBuffer) Read(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	if rb.size == 0 {
		return 0, io.EOF
	}
	
	read := 0
	for read < len(p) && rb.size > 0 {
		p[read] = rb.data[rb.head]
		rb.head = (rb.head + 1) % rb.capacity
		rb.size--
		read++
	}
	
	return read, nil
}

// ReadAt reads data at a specific offset
func (rb *ringBuffer) ReadAt(p []byte, offset int64) (int, error) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if offset < 0 {
		return 0, errors.New("negative offset")
	}
	
	if int(offset) >= rb.size {
		return 0, io.EOF
	}
	
	// Calculate actual position in circular buffer
	pos := (rb.head + int(offset)) % rb.capacity
	read := 0
	
	for read < len(p) && int(offset)+read < rb.size {
		p[read] = rb.data[pos]
		pos = (pos + 1) % rb.capacity
		read++
	}
	
	return read, nil
}

// Size returns the current size of data in the buffer
func (rb *ringBuffer) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// Capacity returns the maximum capacity of the buffer
func (rb *ringBuffer) Capacity() int {
	return rb.capacity
}

// Reset clears the buffer
func (rb *ringBuffer) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	rb.size = 0
	rb.head = 0
	rb.tail = 0
}

// SmartBuffer extends RingBuffer with dynamic sizing based on threat level
type SmartBuffer struct {
	RingBuffer
	window      int // Current window size
	maxWindow   int
	minWindow   int
	threatLevel ThreatLevel
	mu          sync.RWMutex
}

// ThreatLevel represents the current threat assessment
type ThreatLevel int

const (
	ThreatLevelLow ThreatLevel = iota
	ThreatLevelMedium
	ThreatLevelHigh
	ThreatLevelCritical
)

// NewSmartBuffer creates a buffer that adjusts based on threat level
func NewSmartBuffer(capacity, minWindow, maxWindow int) *SmartBuffer {
	return &SmartBuffer{
		RingBuffer: NewRingBuffer(capacity),
		window:     minWindow,
		minWindow:  minWindow,
		maxWindow:  maxWindow,
	}
}

// AdjustWindow changes buffer window based on threat level
func (sb *SmartBuffer) AdjustWindow(threat ThreatLevel) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	
	sb.threatLevel = threat
	
	switch threat {
	case ThreatLevelLow:
		sb.window = sb.minWindow
	case ThreatLevelMedium:
		sb.window = (sb.minWindow + sb.maxWindow) / 2
	case ThreatLevelHigh:
		sb.window = sb.maxWindow - (sb.maxWindow-sb.minWindow)/4
	case ThreatLevelCritical:
		sb.window = sb.maxWindow
	}
}

// GetContext returns data before and after current position
func (sb *SmartBuffer) GetContext(before, after int) []byte {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	
	// Calculate context window
	contextSize := before + after
	if contextSize > sb.Size() {
		contextSize = sb.Size()
	}
	
	result := make([]byte, contextSize)
	
	// Read from appropriate offset
	offset := sb.Size() - before
	if offset < 0 {
		offset = 0
	}
	
	sb.ReadAt(result, int64(offset))
	return result
}