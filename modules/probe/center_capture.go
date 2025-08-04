package probe

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CaptureEngine handles stream capture from processes.
type CaptureEngine struct {
	mode        string
	activeProcs map[int]*ProcessCapture
	mu          sync.RWMutex
}

// ProcessCapture tracks capture state for a single process.
type ProcessCapture struct {
	pid         int
	stdinFile   *os.File
	stdoutFile  *os.File
	stderrFile  *os.File
	lastOffsets map[string]int64 // Track read positions
}

// NewCaptureEngine creates a new capture engine.
func NewCaptureEngine(mode string) *CaptureEngine {
	return &CaptureEngine{
		mode:        mode,
		activeProcs: make(map[int]*ProcessCapture),
	}
}

// Attach begins monitoring a process.
func (e *CaptureEngine) Attach(pid int) error {
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

	capture := &ProcessCapture{
		pid:         pid,
		lastOffsets: make(map[string]int64),
	}

	// Try to open file descriptors
	// Note: We may not have access to stdin (fd/0) as it's often not readable
	// stdout and stderr are typically symlinks to pts/pipe that we can't read directly
	// This is why we'll need to use alternative methods like strace in production

	// For MVP, we'll attempt to read what we can
	stdinPath := filepath.Join(procPath, "fd", "0")
	stdoutPath := filepath.Join(procPath, "fd", "1")
	stderrPath := filepath.Join(procPath, "fd", "2")

	// Try to open files (may fail due to permissions)
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

	e.activeProcs[pid] = capture
	return nil
}

// Detach stops monitoring a process.
func (e *CaptureEngine) Detach(pid int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	capture, exists := e.activeProcs[pid]
	if !exists {
		return fmt.Errorf("not attached to PID %d", pid)
	}

	// Close open files
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
	return nil
}

// ReadStreams attempts to read new data from process streams.
func (e *CaptureEngine) ReadStreams(pid int) (*StreamData, error) {
	e.mu.RLock()
	capture, exists := e.activeProcs[pid]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("not attached to PID %d", pid)
	}

	data := &StreamData{}

	// In a real implementation, we would use more sophisticated methods
	// For MVP, we'll read from /proc/PID/fd/* where possible
	// and fall back to parsing /proc/PID/cmdline and environ for static data

	// Read command line arguments (often contains passwords)
	cmdlineData, err := e.readProcFile(pid, "cmdline")
	if err == nil && len(cmdlineData) > 0 {
		// Command line args are null-separated
		data.Stdout = cmdlineData
	}

	// Read environment (often contains API keys)
	environData, err := e.readProcFile(pid, "environ")
	if err == nil && len(environData) > 0 {
		// Environment vars are null-separated
		data.Stderr = environData
	}

	// Try to read from actual file descriptors if open
	if capture.stdinFile != nil {
		if buf, err := e.readFromOffset(capture.stdinFile, capture.lastOffsets["stdin"]); err == nil {
			data.Stdin = append(data.Stdin, buf...)
			capture.lastOffsets["stdin"] += int64(len(buf))
		}
	}

	if capture.stdoutFile != nil {
		if buf, err := e.readFromOffset(capture.stdoutFile, capture.lastOffsets["stdout"]); err == nil {
			data.Stdout = append(data.Stdout, buf...)
			capture.lastOffsets["stdout"] += int64(len(buf))
		}
	}

	if capture.stderrFile != nil {
		if buf, err := e.readFromOffset(capture.stderrFile, capture.lastOffsets["stderr"]); err == nil {
			data.Stderr = append(data.Stderr, buf...)
			capture.lastOffsets["stderr"] += int64(len(buf))
		}
	}

	return data, nil
}

// readProcFile reads a file from /proc/PID/.
func (e *CaptureEngine) readProcFile(pid int, filename string) ([]byte, error) {
	path := fmt.Sprintf("/proc/%d/%s", pid, filename)
	return os.ReadFile(path)
}

// readFromOffset reads from a file starting at a specific offset.
func (e *CaptureEngine) readFromOffset(file *os.File, offset int64) ([]byte, error) {
	// Seek to last read position
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	// Read available data (non-blocking would be better)
	buf := make([]byte, 4096)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	if n > 0 {
		return buf[:n], nil
	}
	return nil, nil
}

// StraceWrapper provides an alternative capture method using strace.
type StraceWrapper struct {
	pid     int
	cmd     *os.Process
	parser  *StraceParser
	running bool
	mu      sync.Mutex
}

// StraceParser parses strace output.
type StraceParser struct {
	// Parser state
}

// NewStraceWrapper creates a new strace wrapper.
func NewStraceWrapper(pid int) *StraceWrapper {
	return &StraceWrapper{
		pid:    pid,
		parser: &StraceParser{},
	}
}

// Start begins strace monitoring.
func (w *StraceWrapper) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return fmt.Errorf("strace already running")
	}

	// Start strace process
	// Note: This is a simplified version - production would need proper error handling
	// cmd := fmt.Sprintf("strace -p %d -e trace=read,write -s 4096 2>&1", w.pid)
	// Implementation would exec strace and capture output

	w.running = true
	return nil
}

// Stop terminates strace monitoring.
func (w *StraceWrapper) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return nil
	}

	// Kill strace process
	if w.cmd != nil {
		_ = w.cmd.Kill()
	}

	w.running = false
	return nil
}

// ReadEvent reads the next strace event.
func (w *StraceWrapper) ReadEvent() (*StraceEvent, error) {
	// Parse strace output line
	// Extract syscall, fd, and data
	return nil, fmt.Errorf("not implemented")
}

// StraceEvent represents a parsed strace event.
type StraceEvent struct {
	Timestamp time.Time
	Syscall   string
	FD        int
	Data      []byte
	Error     error
}
