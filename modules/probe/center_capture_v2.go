package probe

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CaptureEngineV2 handles stream capture with circular buffers.
type CaptureEngineV2 struct {
	mode          string // procfs, strace, auto
	activeProcs   map[int]*ProcessCaptureV2
	straceEnabled bool
	straceCaptors map[int]*StraceCapture
	captureStats  map[int]*CaptureStats

	// Buffer configuration
	bufferSize     int
	eventDelimiter []byte

	mu sync.RWMutex
}

// ProcessCaptureV2 tracks capture state with circular buffers.
type ProcessCaptureV2 struct {
	pid         int
	stdinFile   *os.File
	stdoutFile  *os.File
	stderrFile  *os.File
	lastOffsets map[string]int64

	// Circular buffers for each stream
	stdinBuffer  *LockFreeCircularBufferV3
	stdoutBuffer *LockFreeCircularBufferV3
	stderrBuffer *LockFreeCircularBufferV3

	// Event channels aggregated from all streams
	events chan StreamEvent
	done   chan struct{}
}

// StreamEvent represents a captured event from any stream.
type StreamEvent struct {
	PID       int
	Stream    string // "stdin", "stdout", "stderr"
	Data      []byte
	Timestamp time.Time
	Sequence  uint64
}

// NewCaptureEngineV2 creates a new capture engine with circular buffers.
func NewCaptureEngineV2(bufferSize int, delimiter []byte) *CaptureEngineV2 {
	if bufferSize == 0 {
		bufferSize = 1024 * 1024 // Default 1MB per stream
	}

	return &CaptureEngineV2{
		mode:           "auto",
		activeProcs:    make(map[int]*ProcessCaptureV2),
		straceEnabled:  false,
		straceCaptors:  make(map[int]*StraceCapture),
		captureStats:   make(map[int]*CaptureStats),
		bufferSize:     bufferSize,
		eventDelimiter: delimiter,
	}
}

// EnableStrace enables strace fallback.
func (e *CaptureEngineV2) EnableStrace() error {
	if err := checkStraceAvailable(); err != nil {
		return fmt.Errorf("cannot enable strace: %w", err)
	}
	e.straceEnabled = true
	fmt.Println("\033[33mWarning: Strace capture enabled. This may impact performance.\033[0m")
	return nil
}

// Attach begins monitoring a process with circular buffers.
func (e *CaptureEngineV2) Attach(pid int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.activeProcs[pid]; exists {
		return fmt.Errorf("already attached to PID %d", pid)
	}

	// Check if process exists
	procPath := fmt.Sprintf("/proc/%d", pid)
	if _, err := os.Stat(procPath); err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}

	// Create circular buffers for each stream
	stdinBuf, err := NewLockFreeCircularBufferV3(e.bufferSize, e.eventDelimiter)
	if err != nil {
		return fmt.Errorf("failed to create stdin buffer: %w", err)
	}

	stdoutBuf, err := NewLockFreeCircularBufferV3(e.bufferSize, e.eventDelimiter)
	if err != nil {
		return fmt.Errorf("failed to create stdout buffer: %w", err)
	}

	stderrBuf, err := NewLockFreeCircularBufferV3(e.bufferSize, e.eventDelimiter)
	if err != nil {
		return fmt.Errorf("failed to create stderr buffer: %w", err)
	}

	capture := &ProcessCaptureV2{
		pid:          pid,
		lastOffsets:  make(map[string]int64),
		stdinBuffer:  stdinBuf,
		stdoutBuffer: stdoutBuf,
		stderrBuffer: stderrBuf,
		events:       make(chan StreamEvent, 1024),
		done:         make(chan struct{}),
	}

	// Try to open file descriptors
	stdinPath := filepath.Join(procPath, "fd", "0")
	stdoutPath := filepath.Join(procPath, "fd", "1")
	stderrPath := filepath.Join(procPath, "fd", "2")

	if file, err := os.Open(stdinPath); err == nil {
		capture.stdinFile = file
		capture.lastOffsets["stdin"] = 0
	}

	if file, err := os.Open(stdoutPath); err == nil {
		capture.stdoutFile = file
		capture.lastOffsets["stdout"] = 0
	}

	if file, err := os.Open(stderrPath); err == nil {
		capture.stderrFile = file
		capture.lastOffsets["stderr"] = 0
	}

	// Start event aggregator
	go capture.aggregateEvents()

	e.activeProcs[pid] = capture

	// Initialize stats
	if e.captureStats[pid] == nil {
		e.captureStats[pid] = &CaptureStats{Method: "procfs"}
	}

	return nil
}

// aggregateEvents combines events from all stream buffers.
func (pc *ProcessCaptureV2) aggregateEvents() {
	for {
		select {
		case <-pc.done:
			close(pc.events)
			return

		case event := <-pc.stdinBuffer.Events():
			pc.events <- StreamEvent{
				PID:       pc.pid,
				Stream:    "stdin",
				Data:      event.Data,
				Timestamp: event.Timestamp,
				Sequence:  event.Sequence,
			}

		case event := <-pc.stdoutBuffer.Events():
			pc.events <- StreamEvent{
				PID:       pc.pid,
				Stream:    "stdout",
				Data:      event.Data,
				Timestamp: event.Timestamp,
				Sequence:  event.Sequence,
			}

		case event := <-pc.stderrBuffer.Events():
			pc.events <- StreamEvent{
				PID:       pc.pid,
				Stream:    "stderr",
				Data:      event.Data,
				Timestamp: event.Timestamp,
				Sequence:  event.Sequence,
			}
		}
	}
}

// CaptureStreams reads new data from process streams into circular buffers.
func (e *CaptureEngineV2) CaptureStreams(pid int) error {
	e.mu.RLock()
	capture, exists := e.activeProcs[pid]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("not attached to PID %d", pid)
	}

	stats := e.captureStats[pid]
	stats.Attempts++

	// Try procfs method first
	bytesRead := int64(0)

	// Read stdin
	if capture.stdinFile != nil {
		offset := capture.lastOffsets["stdin"]
		if n, err := e.readStreamToBuffer(capture.stdinFile, capture.stdinBuffer,
			&offset); err == nil {
			bytesRead += n
			capture.lastOffsets["stdin"] = offset
		}
	}

	// Read stdout
	if capture.stdoutFile != nil {
		offset := capture.lastOffsets["stdout"]
		if n, err := e.readStreamToBuffer(capture.stdoutFile, capture.stdoutBuffer,
			&offset); err == nil {
			bytesRead += n
			capture.lastOffsets["stdout"] = offset
		}
	}

	// Read stderr
	if capture.stderrFile != nil {
		offset := capture.lastOffsets["stderr"]
		if n, err := e.readStreamToBuffer(capture.stderrFile, capture.stderrBuffer,
			&offset); err == nil {
			bytesRead += n
			capture.lastOffsets["stderr"] = offset
		}
	}

	// Update stats
	if bytesRead > 0 {
		stats.Successful++
		stats.BytesCapured += bytesRead
		stats.ConsecutiveFails = 0
		stats.LastSuccess = time.Now()
	} else {
		stats.ConsecutiveFails++

		// Check if we should try strace
		if e.shouldUseStrace(pid, stats) {
			return e.captureWithStrace(pid)
		}
	}

	return nil
}

// readStreamToBuffer reads from a file into a circular buffer.
func (e *CaptureEngineV2) readStreamToBuffer(file *os.File, buffer *LockFreeCircularBufferV3,
	lastOffset *int64) (int64, error) {

	// Read in chunks
	chunk := make([]byte, 4096)
	totalRead := int64(0)

	for {
		n, err := file.ReadAt(chunk, *lastOffset)
		if n > 0 {
			// Write to circular buffer
			if _, writeErr := buffer.Write(chunk[:n]); writeErr != nil {
				// Buffer full or other error, but we still read the data
				// The buffer will handle backpressure appropriately
				log.Printf("Buffer write error (continuing): %v", writeErr)
			}

			*lastOffset += int64(n)
			totalRead += int64(n)
		}

		if err != nil {
			break // EOF or error
		}
	}

	return totalRead, nil
}

// shouldUseStrace determines if we should fallback to strace.
func (e *CaptureEngineV2) shouldUseStrace(pid int, stats *CaptureStats) bool {
	if !e.straceEnabled {
		return false
	}

	// Use strace if:
	// 1. We've failed multiple times
	// 2. Process is using PTY
	// 3. No successful reads in the last minute
	return stats.ConsecutiveFails > 5 ||
		stats.IsUsingPTY ||
		(stats.Successful == 0 && stats.Attempts > 10)
}

// captureWithStrace uses strace to capture streams.
func (e *CaptureEngineV2) captureWithStrace(pid int) error {
	e.mu.RLock()
	capture := e.activeProcs[pid]
	strace, straceExists := e.straceCaptors[pid]
	e.mu.RUnlock()

	if !straceExists {
		if err := e.startStraceCapture(pid); err != nil {
			return err
		}
		strace = e.straceCaptors[pid]
	}

	// Read from strace output
	if streamData, err := strace.Read(); err == nil && streamData != nil {
		// Write captured data to appropriate buffers
		if len(streamData.Stdin) > 0 {
			capture.stdinBuffer.Write(streamData.Stdin)
		}
		if len(streamData.Stdout) > 0 {
			capture.stdoutBuffer.Write(streamData.Stdout)
		}
		if len(streamData.Stderr) > 0 {
			capture.stderrBuffer.Write(streamData.Stderr)
		}

		stats := e.captureStats[pid]
		stats.Method = "strace"
		stats.Successful++
		stats.BytesCapured += int64(len(streamData.Stdin) + len(streamData.Stdout) + len(streamData.Stderr))
	}

	return nil
}

// startStraceCapture initializes strace capture.
func (e *CaptureEngineV2) startStraceCapture(pid int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.straceCaptors[pid]; exists {
		return nil
	}

	captor, err := NewStraceCapture(pid)
	if err != nil {
		return err
	}

	if err := captor.Start(); err != nil {
		return err
	}

	e.straceCaptors[pid] = captor
	return nil
}

// GetEvents returns the event channel for a process.
func (e *CaptureEngineV2) GetEvents(pid int) (<-chan StreamEvent, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	capture, exists := e.activeProcs[pid]
	if !exists {
		return nil, fmt.Errorf("not attached to PID %d", pid)
	}

	return capture.events, nil
}

// GetBufferStats returns buffer statistics for monitoring.
func (e *CaptureEngineV2) GetBufferStats(pid int) (map[string]interface{}, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	capture, exists := e.activeProcs[pid]
	if !exists {
		return nil, fmt.Errorf("not attached to PID %d", pid)
	}

	stats := make(map[string]interface{})
	stats["stdin"] = capture.stdinBuffer.Stats()
	stats["stdout"] = capture.stdoutBuffer.Stats()
	stats["stderr"] = capture.stderrBuffer.Stats()
	stats["capture_stats"] = e.captureStats[pid]

	return stats, nil
}

// Detach stops monitoring a process.
func (e *CaptureEngineV2) Detach(pid int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop strace if running
	if captor, ok := e.straceCaptors[pid]; ok {
		if err := captor.Stop(); err != nil {
			fmt.Printf("Warning: failed to stop strace for PID %d: %v\n", pid, err)
		}
		delete(e.straceCaptors, pid)
	}

	capture, exists := e.activeProcs[pid]
	if !exists {
		return fmt.Errorf("not attached to PID %d", pid)
	}

	// Signal shutdown
	close(capture.done)

	// Close circular buffers
	capture.stdinBuffer.Close()
	capture.stdoutBuffer.Close()
	capture.stderrBuffer.Close()

	// Close file handles
	if capture.stdinFile != nil {
		capture.stdinFile.Close()
	}
	if capture.stdoutFile != nil {
		capture.stdoutFile.Close()
	}
	if capture.stderrFile != nil {
		capture.stderrFile.Close()
	}

	delete(e.activeProcs, pid)
	delete(e.captureStats, pid)

	return nil
}
