package probe

import (
	"fmt"
	"sync/atomic"
	"time"
)

// ProtocolAwareBuffer extends our circular buffer with protocol detection.
type ProtocolAwareBuffer struct {
	*LockFreeCircularBufferV3

	// Protocol detection
	detector        *ProtocolBoundaryDetector
	currentProtocol atomic.Value // string
	protocolStats   map[string]*ProtocolStats
	autoDetect      atomic.Bool
	preferredProto  string

	// Enhanced event channel with protocol info
	protocolEvents chan ProtocolEvent
}

// ProtocolEvent includes protocol information with the data.
type ProtocolEvent struct {
	EventV3
	Protocol  string
	FrameType string // For protocols with multiple frame types
	Metadata  map[string]interface{}
}

// ProtocolStats tracks protocol-specific statistics.
type ProtocolStats struct {
	DetectedCount  uint64
	BytesProcessed uint64
	AvgMessageSize uint64
	LastDetected   time.Time
}

// NewProtocolAwareBuffer creates a buffer with protocol detection.
func NewProtocolAwareBuffer(size int, preferredProtocol string) (*ProtocolAwareBuffer, error) {
	// Create base circular buffer with no delimiter (we'll detect boundaries)
	base, err := NewLockFreeCircularBufferV3NoProcessor(size, nil)
	if err != nil {
		return nil, err
	}

	pab := &ProtocolAwareBuffer{
		LockFreeCircularBufferV3: base,
		detector:                 NewProtocolBoundaryDetector(),
		protocolStats:            make(map[string]*ProtocolStats),
		protocolEvents:           make(chan ProtocolEvent, 4096),
		preferredProto:           preferredProtocol,
	}

	// Set initial protocol if specified
	if preferredProtocol != "" {
		pab.currentProtocol.Store(preferredProtocol)
	} else {
		pab.autoDetect.Store(true)
		pab.currentProtocol.Store("unknown")
	}

	// Start protocol-aware processor
	go pab.processProtocolEvents()

	return pab, nil
}

// processProtocolEvents scans for protocol-specific boundaries.
func (pab *ProtocolAwareBuffer) processProtocolEvents() {
	var lastProtocol string
	scanBuffer := make([]byte, 65536) // 64KB scan buffer
	var partialMessage []byte
	var sequence uint64

	for {
		select {
		case <-pab.done:
			close(pab.protocolEvents)
			return
		default:
			interval := time.Duration(pab.scanInterval.Load())
			time.Sleep(interval)

			pab.scanForProtocolEvents(scanBuffer, &partialMessage, &sequence, &lastProtocol)
			pab.scans.Add(1)
		}
	}
}

// scanForProtocolEvents looks for protocol-specific message boundaries.
func (pab *ProtocolAwareBuffer) scanForProtocolEvents(scanBuf []byte,
	partialMessage *[]byte, sequence *uint64, lastProtocol *string) {

	packedRead := pab.readIndex.Load()
	readVer, readIdx := unpackIndex(packedRead)

	packedWrite := pab.writeIndex.Load()
	_, writeIdx := unpackIndex(packedWrite)

	if readIdx >= writeIdx {
		return // No new data
	}

	available := writeIdx - readIdx
	scanSize := uint32(len(scanBuf))
	if available < scanSize {
		scanSize = available
	}

	// Include partial message from previous scan
	totalSize := len(*partialMessage) + int(scanSize)
	workBuffer := make([]byte, totalSize)
	copy(workBuffer, *partialMessage)

	// Copy new data
	start := uint64(readIdx) & pab.mask
	newDataStart := len(*partialMessage)

	if start+uint64(scanSize) <= pab.size {
		copy(workBuffer[newDataStart:], pab.data[start:start+uint64(scanSize)])
	} else {
		firstPart := pab.size - start
		copy(workBuffer[newDataStart:newDataStart+int(firstPart)], pab.data[start:])
		copy(workBuffer[newDataStart+int(firstPart):], pab.data[:scanSize-uint32(firstPart)])
	}

	// Process messages
	offset := 0
	consumed := 0
	currentProto := pab.getCurrentProtocol()

	for offset < len(workBuffer) {
		var protocol string
		var boundary, msgSize int

		// Try preferred protocol first
		if currentProto != "unknown" && currentProto != "" {
			if detector, ok := pab.detector.detectors[currentProto]; ok {
				if b, s, found := detector.DetectBoundary(workBuffer, offset); found {
					protocol = currentProto
					boundary = b
					msgSize = s
				}
			}
		}

		// Auto-detect if needed
		if protocol == "" && pab.autoDetect.Load() {
			protocol, boundary, msgSize = pab.detector.DetectProtocol(workBuffer[offset:])
			if protocol != "" {
				boundary += offset

				// Update detected protocol
				if protocol != *lastProtocol {
					pab.setCurrentProtocol(protocol)
					*lastProtocol = protocol
				}
			}
		}

		if protocol == "" {
			// No protocol detected, save as partial
			break
		}

		// Extract and send message
		if boundary <= len(workBuffer) {
			message := workBuffer[offset:boundary]

			event := ProtocolEvent{
				EventV3: EventV3{
					Data:      message[:msgSize], // Actual message without padding
					Timestamp: time.Now(),
					Offset:    uint64(readIdx) + uint64(consumed),
					Sequence:  *sequence,
				},
				Protocol: protocol,
				Metadata: pab.extractMetadata(protocol, message),
			}

			// Determine frame type for certain protocols
			event.FrameType = pab.detectFrameType(protocol, message)

			select {
			case pab.protocolEvents <- event:
				pab.eventsSent.Add(1)
				pab.updateProtocolStats(protocol, msgSize)
			default:
				pab.dropped.Add(1)
			}

			*sequence++
			offset = boundary
			consumed = boundary - len(*partialMessage)
			if consumed < 0 {
				consumed = 0
			}
		} else {
			// Incomplete message
			break
		}
	}

	// Save any partial message
	if offset < len(workBuffer) {
		*partialMessage = make([]byte, len(workBuffer)-offset)
		copy(*partialMessage, workBuffer[offset:])

		// If we have partial data, we still need to advance the read index
		// to avoid re-reading the same data. The partial message will be
		// prepended on the next scan.
		if consumed == 0 && len(*partialMessage) > 0 {
			// Advance by the amount of new data we scanned (not including old partial)
			consumed = int(scanSize)
		}
	} else {
		*partialMessage = nil
	}

	// Update read index
	if consumed > 0 {
		newIdx := readIdx + uint32(consumed)
		newPacked := packIndex(readVer+1, newIdx)
		pab.readIndex.CompareAndSwap(packedRead, newPacked)
	}
}

// extractMetadata extracts protocol-specific metadata.
func (pab *ProtocolAwareBuffer) extractMetadata(protocol string, data []byte) map[string]interface{} {
	metadata := make(map[string]interface{})

	switch protocol {
	case "http":
		// Extract method, path, status code, etc.
		if len(data) > 10 {
			// Simple extraction - real implementation would be more robust
			if data[0] >= 'A' && data[0] <= 'Z' {
				// Request
				endMethod := 0
				for i := 0; i < len(data) && i < 10; i++ {
					if data[i] == ' ' {
						endMethod = i
						break
					}
				}
				if endMethod > 0 {
					metadata["method"] = string(data[:endMethod])
				}
			} else if string(data[:5]) == "HTTP/" {
				// Response
				if len(data) > 12 {
					metadata["status"] = string(data[9:12])
				}
			}
		}

	case "grpc":
		// Extract compression flag
		if len(data) > 0 {
			metadata["compressed"] = data[0] == 1
		}

	case "websocket":
		// Extract opcode and mask info
		if len(data) > 1 {
			metadata["opcode"] = data[0] & 0x0F
			metadata["masked"] = (data[1] & 0x80) != 0
			metadata["fin"] = (data[0] & 0x80) != 0
		}
	}

	return metadata
}

// detectFrameType identifies specific frame types within a protocol.
func (pab *ProtocolAwareBuffer) detectFrameType(protocol string, data []byte) string {
	switch protocol {
	case "websocket":
		if len(data) > 0 {
			opcode := data[0] & 0x0F
			switch opcode {
			case 0x0:
				return "continuation"
			case 0x1:
				return "text"
			case 0x2:
				return "binary"
			case 0x8:
				return "close"
			case 0x9:
				return "ping"
			case 0xA:
				return "pong"
			}
		}

	case "http":
		if len(data) > 4 {
			switch string(data[:4]) {
			case "GET ":
				return "GET"
			case "POST":
				return "POST"
			case "PUT ":
				return "PUT"
			case "DELE":
				return "DELETE"
			case "HTTP":
				return "response"
			}
		}
	}

	return ""
}

// updateProtocolStats updates statistics for a protocol.
func (pab *ProtocolAwareBuffer) updateProtocolStats(protocol string, messageSize int) {
	stats, exists := pab.protocolStats[protocol]
	if !exists {
		stats = &ProtocolStats{}
		pab.protocolStats[protocol] = stats
	}

	stats.DetectedCount++
	stats.BytesProcessed += uint64(messageSize)
	stats.AvgMessageSize = stats.BytesProcessed / stats.DetectedCount
	stats.LastDetected = time.Now()
}

// SetProtocol sets the preferred protocol for detection.
func (pab *ProtocolAwareBuffer) SetProtocol(protocol string) error {
	pab.detector.mu.RLock()
	_, exists := pab.detector.detectors[protocol]
	pab.detector.mu.RUnlock()

	if !exists && protocol != "" {
		return fmt.Errorf("unknown protocol: %s", protocol)
	}

	pab.preferredProto = protocol
	pab.setCurrentProtocol(protocol)

	if protocol == "" {
		pab.autoDetect.Store(true)
	} else {
		pab.autoDetect.Store(false)
	}

	return nil
}

// EnableAutoDetect enables automatic protocol detection.
func (pab *ProtocolAwareBuffer) EnableAutoDetect() {
	pab.autoDetect.Store(true)
}

// getCurrentProtocol safely gets the current protocol.
func (pab *ProtocolAwareBuffer) getCurrentProtocol() string {
	if v := pab.currentProtocol.Load(); v != nil {
		return v.(string)
	}
	return "unknown"
}

// setCurrentProtocol safely sets the current protocol.
func (pab *ProtocolAwareBuffer) setCurrentProtocol(protocol string) {
	pab.currentProtocol.Store(protocol)
}

// ProtocolEvents returns the protocol-aware event channel.
func (pab *ProtocolAwareBuffer) ProtocolEvents() <-chan ProtocolEvent {
	return pab.protocolEvents
}

// GetProtocolStats returns statistics by protocol.
func (pab *ProtocolAwareBuffer) GetProtocolStats() map[string]*ProtocolStats {
	stats := make(map[string]*ProtocolStats)
	for k, v := range pab.protocolStats {
		// Create a copy to avoid race conditions
		statsCopy := &ProtocolStats{
			DetectedCount:  v.DetectedCount,
			BytesProcessed: v.BytesProcessed,
			AvgMessageSize: v.AvgMessageSize,
			LastDetected:   v.LastDetected,
		}
		stats[k] = statsCopy
	}
	return stats
}

// Stats extends base stats with protocol information.
func (pab *ProtocolAwareBuffer) Stats() map[string]interface{} {
	stats := pab.LockFreeCircularBufferV3.Stats()
	stats["current_protocol"] = pab.getCurrentProtocol()
	stats["auto_detect"] = pab.autoDetect.Load()
	stats["protocol_stats"] = pab.GetProtocolStats()
	return stats
}

// RegisterProtocol adds a custom protocol detector.
func (pab *ProtocolAwareBuffer) RegisterProtocol(name string, detector BoundaryDetector) {
	pab.detector.Register(name, detector)
}
