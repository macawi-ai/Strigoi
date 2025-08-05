package probe

import (
	"errors"
	"sync/atomic"
	"time"
	"unsafe"
)

// LockFreeCircularBufferV3 implements a production-ready lock-free circular buffer
// with ABA protection, adaptive scanning, and proper event boundary handling.
type LockFreeCircularBufferV3 struct {
	// Buffer data and metadata
	data []byte
	size uint64
	mask uint64 // For fast modulo operations (size must be power of 2)

	// Versioned atomic indices for ABA protection
	// Each index is packed: [32-bit version][32-bit index]
	writeIndex atomic.Uint64 // Current write position with version
	readIndex  atomic.Uint64 // Current read position with version

	// Event tracking
	eventDelimiter []byte
	maxEventSize   int
	delimiterLen   int

	// Adaptive scanning
	scanInterval   atomic.Int64  // Current scan interval in nanoseconds
	writeRate      atomic.Uint64 // Writes per second (moving average)
	lastWriteCount atomic.Uint64
	lastRateUpdate atomic.Int64

	// Event delivery
	events chan EventV3
	done   chan struct{}
	closed atomic.Bool

	// Stats
	dropped    atomic.Uint64
	written    atomic.Uint64
	eventsSent atomic.Uint64
	scans      atomic.Uint64

	// Backpressure
	highWaterMark uint64 // Threshold for backpressure (90% of size)
}

// EventV3 represents a complete event with metadata.
type EventV3 struct {
	Data      []byte
	Timestamp time.Time
	Offset    uint64
	Sequence  uint64 // Monotonic sequence number
}

// NewLockFreeCircularBufferV3 creates a production-ready lock-free buffer.
func NewLockFreeCircularBufferV3(size int, delimiter []byte) (*LockFreeCircularBufferV3, error) {
	// Ensure size is power of 2
	if size&(size-1) != 0 {
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

	if size < 65536 {
		size = 65536 // Minimum 64KB for version packing
	}

	cb := &LockFreeCircularBufferV3{
		data:           make([]byte, size),
		size:           uint64(size),
		mask:           uint64(size - 1),
		eventDelimiter: delimiter,
		delimiterLen:   len(delimiter),
		maxEventSize:   size / 4,
		events:         make(chan EventV3, 4096),
		done:           make(chan struct{}),
		highWaterMark:  uint64(float64(size) * 0.9),
	}

	// Start with 1ms scan interval
	cb.scanInterval.Store(1_000_000) // 1ms in nanoseconds
	cb.lastRateUpdate.Store(time.Now().UnixNano())

	// Start event processor
	go cb.processEvents()

	return cb, nil
}

// packIndex combines version and index into a single uint64.
func packIndex(version uint32, index uint32) uint64 {
	return (uint64(version) << 32) | uint64(index)
}

// unpackIndex extracts version and index from packed value.
func unpackIndex(packed uint64) (version uint32, index uint32) {
	return uint32(packed >> 32), uint32(packed & 0xFFFFFFFF)
}

// Write adds data to the buffer with ABA protection.
func (cb *LockFreeCircularBufferV3) Write(p []byte) (int, error) {
	if cb.closed.Load() {
		return 0, errors.New("buffer closed")
	}

	n := uint32(len(p))
	if n == 0 {
		return 0, nil
	}

	// Check for backpressure
	if cb.shouldBackpressure() {
		return 0, errors.New("buffer full - backpressure")
	}

	// Reserve space with versioned CAS
	for {
		packedWrite := cb.writeIndex.Load()
		writeVer, writeIdx := unpackIndex(packedWrite)

		packedRead := cb.readIndex.Load()
		_, readIdx := unpackIndex(packedRead)

		// Check available space
		available := cb.size - uint64(writeIdx-readIdx)
		if available < uint64(n) {
			cb.dropped.Add(uint64(n))
			return 0, errors.New("buffer full")
		}

		// Try to advance write index with version increment
		newIdx := writeIdx + n
		newPacked := packIndex(writeVer+1, newIdx)

		if cb.writeIndex.CompareAndSwap(packedWrite, newPacked) {
			// Successfully reserved space, now copy data
			cb.copyToBuffer(p, uint64(writeIdx))
			cb.written.Add(uint64(n))
			cb.updateWriteRate()
			return int(n), nil
		}
		// Retry if CAS failed
	}
}

// shouldBackpressure checks if we should apply backpressure.
func (cb *LockFreeCircularBufferV3) shouldBackpressure() bool {
	packedWrite := cb.writeIndex.Load()
	_, writeIdx := unpackIndex(packedWrite)

	packedRead := cb.readIndex.Load()
	_, readIdx := unpackIndex(packedRead)

	used := uint64(writeIdx - readIdx)
	return used >= cb.highWaterMark
}

// updateWriteRate updates the moving average write rate.
func (cb *LockFreeCircularBufferV3) updateWriteRate() {
	now := time.Now().UnixNano()
	lastUpdate := cb.lastRateUpdate.Load()

	// Update rate every 100ms
	if now-lastUpdate > 100_000_000 {
		if cb.lastRateUpdate.CompareAndSwap(lastUpdate, now) {
			currentCount := cb.written.Load()
			lastCount := cb.lastWriteCount.Swap(currentCount)

			elapsed := float64(now-lastUpdate) / 1e9 // Convert to seconds
			rate := float64(currentCount-lastCount) / elapsed

			// Exponential moving average
			oldRate := cb.writeRate.Load()
			newRate := uint64(float64(oldRate)*0.7 + rate*0.3)
			cb.writeRate.Store(newRate)

			// Adapt scan interval based on rate
			cb.adaptScanInterval(newRate)
		}
	}
}

// adaptScanInterval adjusts scanning frequency based on write rate.
func (cb *LockFreeCircularBufferV3) adaptScanInterval(bytesPerSec uint64) {
	var interval int64

	switch {
	case bytesPerSec > 100_000_000: // >100MB/s
		interval = 100_000 // 100μs
	case bytesPerSec > 10_000_000: // >10MB/s
		interval = 500_000 // 500μs
	case bytesPerSec > 1_000_000: // >1MB/s
		interval = 1_000_000 // 1ms
	case bytesPerSec > 100_000: // >100KB/s
		interval = 5_000_000 // 5ms
	default:
		interval = 10_000_000 // 10ms
	}

	cb.scanInterval.Store(interval)
}

// copyToBuffer copies data handling wraparound.
func (cb *LockFreeCircularBufferV3) copyToBuffer(data []byte, pos uint64) {
	n := uint64(len(data))
	start := pos & cb.mask

	if start+n <= cb.size {
		copy(cb.data[start:], data)
	} else {
		firstPart := cb.size - start
		copy(cb.data[start:], data[:firstPart])
		copy(cb.data[0:], data[firstPart:])
	}
}

// processEvents continuously scans for complete events with adaptive timing.
func (cb *LockFreeCircularBufferV3) processEvents() {
	var eventStart uint64
	var inEvent bool
	var partialDelimiter []byte // Handle split delimiters
	var sequence uint64

	scanBuffer := make([]byte, 65536) // 64KB scan buffer

	for {
		select {
		case <-cb.done:
			close(cb.events)
			return
		default:
			interval := time.Duration(cb.scanInterval.Load())
			time.Sleep(interval)

			cb.scanForEvents(scanBuffer, &eventStart, &inEvent, &partialDelimiter, &sequence)
			cb.scans.Add(1)
		}
	}
}

// scanForEvents processes available data with delimiter splitting support.
func (cb *LockFreeCircularBufferV3) scanForEvents(scanBuf []byte, eventStart *uint64,
	inEvent *bool, partialDelim *[]byte, sequence *uint64) {

	packedRead := cb.readIndex.Load()
	readVer, readIdx := unpackIndex(packedRead)

	packedWrite := cb.writeIndex.Load()
	_, writeIdx := unpackIndex(packedWrite)

	if readIdx >= writeIdx {
		return // No new data
	}

	available := writeIdx - readIdx
	scanSize := uint32(len(scanBuf))
	if available < scanSize {
		scanSize = available
	}

	// Copy data for scanning
	start := uint64(readIdx) & cb.mask
	if start+uint64(scanSize) <= cb.size {
		copy(scanBuf[:scanSize], cb.data[start:])
	} else {
		firstPart := cb.size - start
		copy(scanBuf[:firstPart], cb.data[start:])
		copy(scanBuf[firstPart:scanSize], cb.data[0:])
	}

	// Process data with delimiter handling
	var consumed uint32
	offset := uint32(0)

	// Check for partial delimiter from previous scan
	if len(*partialDelim) > 0 {
		needed := cb.delimiterLen - len(*partialDelim)
		if int(scanSize) >= needed {
			combined := append(*partialDelim, scanBuf[:needed]...)
			if cb.isDelimiter(combined) {
				// Complete delimiter found
				if *inEvent {
					cb.sendEvent(*eventStart, uint64(readIdx), *sequence)
					*sequence++
				}
				offset = uint32(needed)
				*eventStart = uint64(readIdx) + uint64(offset)
				*inEvent = true
			}
		}
		*partialDelim = nil
	}

	// Scan for events
	for offset < scanSize {
		// Check for delimiter
		if cb.delimiterLen > 0 && offset+uint32(cb.delimiterLen) <= scanSize {
			if cb.isDelimiter(scanBuf[offset : offset+uint32(cb.delimiterLen)]) {
				if *inEvent {
					eventEnd := uint64(readIdx) + uint64(offset)
					cb.sendEvent(*eventStart, eventEnd, *sequence)
					*sequence++
				}
				offset += uint32(cb.delimiterLen)
				*eventStart = uint64(readIdx) + uint64(offset)
				consumed = offset
				continue
			}
		} else if cb.delimiterLen > 0 && offset < scanSize {
			// Potential partial delimiter at end
			remaining := scanSize - offset
			if remaining < uint32(cb.delimiterLen) {
				*partialDelim = make([]byte, remaining)
				copy(*partialDelim, scanBuf[offset:])
			}
		}

		if !*inEvent {
			*eventStart = uint64(readIdx) + uint64(offset)
			*inEvent = true
		}

		offset++
		consumed = offset
	}

	// Update read index with version increment
	if consumed > 0 {
		newIdx := readIdx + consumed
		newPacked := packIndex(readVer+1, newIdx)
		cb.readIndex.CompareAndSwap(packedRead, newPacked)
	}
}

// isDelimiter checks if data matches delimiter.
func (cb *LockFreeCircularBufferV3) isDelimiter(data []byte) bool {
	if len(data) != cb.delimiterLen {
		return false
	}
	for i := 0; i < cb.delimiterLen; i++ {
		if data[i] != cb.eventDelimiter[i] {
			return false
		}
	}
	return true
}

// sendEvent extracts and sends an event.
func (cb *LockFreeCircularBufferV3) sendEvent(start, end, seq uint64) {
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

	event := EventV3{
		Data:      eventData,
		Timestamp: time.Now(),
		Offset:    start,
		Sequence:  seq,
	}

	select {
	case cb.events <- event:
		cb.eventsSent.Add(1)
	default:
		cb.dropped.Add(1)
	}
}

// Events returns the channel for receiving events.
func (cb *LockFreeCircularBufferV3) Events() <-chan EventV3 {
	return cb.events
}

// Stats returns detailed buffer statistics.
func (cb *LockFreeCircularBufferV3) Stats() map[string]interface{} {
	packedWrite := cb.writeIndex.Load()
	writeVer, writeIdx := unpackIndex(packedWrite)

	packedRead := cb.readIndex.Load()
	readVer, readIdx := unpackIndex(packedRead)

	used := uint64(writeIdx - readIdx)

	return map[string]interface{}{
		"size":           cb.size,
		"used":           used,
		"available":      cb.size - used,
		"usage_pct":      float64(used) / float64(cb.size) * 100,
		"written":        cb.written.Load(),
		"dropped":        cb.dropped.Load(),
		"events_sent":    cb.eventsSent.Load(),
		"write_idx":      writeIdx,
		"write_version":  writeVer,
		"read_idx":       readIdx,
		"read_version":   readVer,
		"write_rate_bps": cb.writeRate.Load(),
		"scan_interval":  time.Duration(cb.scanInterval.Load()),
		"total_scans":    cb.scans.Load(),
		"backpressure":   cb.shouldBackpressure(),
	}
}

// Close gracefully shuts down the buffer.
func (cb *LockFreeCircularBufferV3) Close() error {
	if !cb.closed.CompareAndSwap(false, true) {
		return errors.New("already closed")
	}
	close(cb.done)
	return nil
}

// Ensure proper memory alignment
var _ = unsafe.Sizeof(LockFreeCircularBufferV3{})
