package stream

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

// StdioStream captures stdin/stdout/stderr from a process
type StdioStream struct {
	id         string
	pid        int
	cmd        *exec.Cmd
	pty        *os.File
	buffer     RingBuffer
	filters    []Filter
	handlers   map[string]StreamHandler
	ctx        context.Context
	cancel     context.CancelFunc
	active     bool
	stats      StreamStats
	mu         sync.RWMutex
	handlersMu sync.RWMutex
}

// NewStdioStream creates a new STDIO stream capture
func NewStdioStream(pid int, bufferSize int) (*StdioStream, error) {
	return &StdioStream{
		id:       fmt.Sprintf("STDIO-%s", uuid.New().String()[:8]),
		pid:      pid,
		buffer:   NewRingBuffer(bufferSize),
		filters:  make([]Filter, 0),
		handlers: make(map[string]StreamHandler),
		stats: StreamStats{
			StartTime: time.Now(),
		},
	}, nil
}

// NewStdioStreamCommand creates a stream for a command
func NewStdioStreamCommand(command string, args []string, bufferSize int) (*StdioStream, error) {
	cmd := exec.Command(command, args...)
	
	return &StdioStream{
		id:       fmt.Sprintf("STDIO-%s", uuid.New().String()[:8]),
		cmd:      cmd,
		buffer:   NewRingBuffer(bufferSize),
		filters:  make([]Filter, 0),
		handlers: make(map[string]StreamHandler),
		stats: StreamStats{
			StartTime: time.Now(),
		},
	}, nil
}

// Start begins capturing stream data
func (s *StdioStream) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.active {
		return fmt.Errorf("stream already active")
	}
	
	s.ctx, s.cancel = context.WithCancel(ctx)
	
	// If we have a command, start it with PTY
	if s.cmd != nil {
		var err error
		s.pty, err = pty.Start(s.cmd)
		if err != nil {
			return fmt.Errorf("failed to start PTY: %w", err)
		}
		s.pid = s.cmd.Process.Pid
	} else {
		// Attach to existing process
		if err := s.attachToProcess(); err != nil {
			return fmt.Errorf("failed to attach to process: %w", err)
		}
	}
	
	s.active = true
	
	// Start capture goroutine
	go s.captureLoop()
	
	return nil
}

// Stop halts stream capture
func (s *StdioStream) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.active {
		return nil
	}
	
	s.cancel()
	s.active = false
	
	if s.pty != nil {
		s.pty.Close()
	}
	
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	
	return nil
}

// attachToProcess attaches to an existing process for monitoring
func (s *StdioStream) attachToProcess() error {
	// For existing processes, we'll use /proc/[pid]/fd/* on Linux
	// This is a simplified version - full implementation would handle
	// different attachment methods
	
	// Check if process exists
	process, err := os.FindProcess(s.pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}
	
	// Send signal 0 to check if process is alive
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("process not accessible: %w", err)
	}
	
	// Note: Full implementation would set up monitoring via
	// ptrace, /proc filesystem, or other platform-specific methods
	return nil
}

// captureLoop continuously reads from the stream
func (s *StdioStream) captureLoop() {
	reader := bufio.NewReader(s.pty)
	buffer := make([]byte, 4096)
	
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			n, err := reader.Read(buffer)
			if err != nil {
				if err != io.EOF {
					s.mu.Lock()
					s.stats.ErrorCount++
					s.mu.Unlock()
				}
				continue
			}
			
			if n > 0 {
				data := buffer[:n]
				
				// Apply filters
				if s.shouldProcess(data) {
					// Write to ring buffer
					s.buffer.Write(data)
					
					// Update stats
					s.mu.Lock()
					s.stats.BytesProcessed += uint64(n)
					s.stats.EventsCount++
					s.stats.LastEventTime = time.Now()
					s.mu.Unlock()
					
					// Notify handlers
					s.notifyHandlers(data)
				}
			}
		}
	}
}

// shouldProcess checks if data passes all filters
func (s *StdioStream) shouldProcess(data []byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, filter := range s.filters {
		if !filter.Match(data) {
			return false
		}
	}
	return true
}

// notifyHandlers sends data to all subscribed handlers
func (s *StdioStream) notifyHandlers(data []byte) {
	s.handlersMu.RLock()
	handlers := make([]StreamHandler, 0, len(s.handlers))
	for _, h := range s.handlers {
		handlers = append(handlers, h)
	}
	s.handlersMu.RUnlock()
	
	streamData := StreamData{
		ID:        s.id,
		Type:      StreamTypeSTDIO,
		Source:    fmt.Sprintf("pid:%d", s.pid),
		Timestamp: time.Now(),
		Data:      make([]byte, len(data)),
		Metadata: map[string]interface{}{
			"pid": s.pid,
		},
	}
	copy(streamData.Data, data)
	
	// Notify handlers in parallel based on priority
	var wg sync.WaitGroup
	for _, handler := range handlers {
		wg.Add(1)
		go func(h StreamHandler) {
			defer wg.Done()
			if err := h.OnData(streamData); err != nil {
				// Log error but continue
				s.mu.Lock()
				s.stats.ErrorCount++
				s.mu.Unlock()
			}
		}(handler)
	}
	
	// Wait with timeout to prevent blocking
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		// Timeout - handlers taking too long
	}
}

// Subscribe adds a handler to receive stream data
func (s *StdioStream) Subscribe(handler StreamHandler) error {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()
	
	if _, exists := s.handlers[handler.GetID()]; exists {
		return fmt.Errorf("handler already subscribed: %s", handler.GetID())
	}
	
	s.handlers[handler.GetID()] = handler
	return nil
}

// Unsubscribe removes a handler
func (s *StdioStream) Unsubscribe(handlerID string) error {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()
	
	if _, exists := s.handlers[handlerID]; !exists {
		return fmt.Errorf("handler not found: %s", handlerID)
	}
	
	delete(s.handlers, handlerID)
	return nil
}

// AddFilter adds a filter to the stream
func (s *StdioStream) AddFilter(filter Filter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.filters = append(s.filters, filter)
	return nil
}

// RemoveFilter removes a filter by name
func (s *StdioStream) RemoveFilter(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for i, filter := range s.filters {
		if filter.GetName() == name {
			s.filters = append(s.filters[:i], s.filters[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("filter not found: %s", name)
}

// GetID returns the stream ID
func (s *StdioStream) GetID() string {
	return s.id
}

// GetType returns the stream type
func (s *StdioStream) GetType() StreamType {
	return StreamTypeSTDIO
}

// IsActive returns whether the stream is active
func (s *StdioStream) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// GetStats returns stream statistics
func (s *StdioStream) GetStats() StreamStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}