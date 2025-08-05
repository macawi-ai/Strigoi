package probe

import (
	"time"
)

// NewLockFreeCircularBufferV3NoProcessor creates a buffer without starting the processor.
// This is used as a base for protocol-aware buffers that implement their own processing.
func NewLockFreeCircularBufferV3NoProcessor(size int, delimiter []byte) (*LockFreeCircularBufferV3, error) {
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

	// Don't start event processor - let derived types handle this

	return cb, nil
}
