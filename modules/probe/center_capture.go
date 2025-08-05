package probe

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CaptureEngine handles stream capture from processes.
type CaptureEngine struct {
	mode          string // procfs, strace, auto
	activeProcs   map[int]*ProcessCapture
	straceEnabled bool // Allow strace fallback
	straceCaptors map[int]*StraceCapture
	captureStats  map[int]*CaptureStats
	mu            sync.RWMutex
}

// CaptureStats tracks capture method performance.
type CaptureStats struct {
	Method           string
	Attempts         int
	Successful       int
	BytesCapured     int64
	LastSuccess      time.Time
	ConsecutiveFails int
	// Track bytes over time to detect static vs stream data
	LastStdinBytes  int64
	LastStdoutBytes int64
	LastStderrBytes int64
	StaticDataCount int  // How many times we got same data
	IsUsingPTY      bool // Cached PTY detection result
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
		mode:          mode,
		activeProcs:   make(map[int]*ProcessCapture),
		straceEnabled: false, // Opt-in only
		straceCaptors: make(map[int]*StraceCapture),
		captureStats:  make(map[int]*CaptureStats),
	}
}

// EnableStrace enables strace fallback (opt-in).
func (e *CaptureEngine) EnableStrace() error {
	if err := checkStraceAvailable(); err != nil {
		return fmt.Errorf("cannot enable strace: %w", err)
	}
	e.straceEnabled = true
	fmt.Println("\033[33mWarning: Strace capture enabled. This may impact performance.\033[0m")
	return nil
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

	// Stop strace if running
	if captor, ok := e.straceCaptors[pid]; ok {
		if err := captor.Stop(); err != nil {
			fmt.Printf("Warning: failed to stop strace for PID %d: %v\n", pid, err)
		}
		delete(e.straceCaptors, pid)
	}

	capture, exists := e.activeProcs[pid]
	if exists {
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
	}

	// Clean up stats
	delete(e.captureStats, pid)

	return nil
}

// startStraceCapture initializes strace capture for a process.
func (e *CaptureEngine) startStraceCapture(pid int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if already running
	if _, exists := e.straceCaptors[pid]; exists {
		return nil
	}

	// Create strace captor
	captor, err := NewStraceCapture(pid)
	if err != nil {
		return err
	}

	// Start capture
	if err := captor.Start(); err != nil {
		return err
	}

	e.straceCaptors[pid] = captor
	return nil
}

// GetCaptureMethod returns the active capture method for a PID.
func (e *CaptureEngine) GetCaptureMethod(pid int) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if _, ok := e.straceCaptors[pid]; ok {
		return "strace"
	}

	if stats, ok := e.captureStats[pid]; ok {
		return stats.Method
	}

	return "procfs"
}

// GetCaptureStats returns capture statistics.
func (e *CaptureEngine) GetCaptureStats() map[int]*CaptureStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return a copy to avoid race conditions
	stats := make(map[int]*CaptureStats)
	for pid, stat := range e.captureStats {
		statCopy := *stat
		stats[pid] = &statCopy
	}
	return stats
}

// isUsingPTY checks if a process is using a pseudo-terminal.
func isUsingPTY(pid int) (bool, error) {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	files, err := os.ReadDir(fdDir)
	if err != nil {
		return false, err
	}

	for _, file := range files {
		// Check stdin (0), stdout (1), stderr (2)
		if file.Name() == "0" || file.Name() == "1" || file.Name() == "2" {
			fdPath := filepath.Join(fdDir, file.Name())
			target, err := os.Readlink(fdPath)
			if err == nil && strings.HasPrefix(target, "/dev/pts/") {
				return true, nil
			}
		}
	}
	return false, nil
}

// ReadStreams attempts to read new data from process streams.
func (e *CaptureEngine) ReadStreams(pid int) (*StreamData, error) {
	// Get or create stats
	e.mu.Lock()
	stats, exists := e.captureStats[pid]
	if !exists {
		stats = &CaptureStats{Method: "procfs"}
		// Check PTY usage on first encounter
		if pty, err := isUsingPTY(pid); err == nil {
			stats.IsUsingPTY = pty
		}
		e.captureStats[pid] = stats
	}
	e.mu.Unlock()

	// Check if we're already using strace for this PID
	e.mu.RLock()
	if captor, ok := e.straceCaptors[pid]; ok {
		e.mu.RUnlock()
		return captor.Read()
	}
	e.mu.RUnlock()

	// Try procfs method
	data, err := e.readProcFS(pid)
	stats.Attempts++

	if err == nil {
		// Calculate new data bytes
		currentStdinBytes := int64(len(data.Stdin))
		currentStdoutBytes := int64(len(data.Stdout))
		currentStderrBytes := int64(len(data.Stderr))

		// Check if we got new stream data (not just static environ)
		newStreamData := false
		if currentStdinBytes > stats.LastStdinBytes ||
			currentStdoutBytes > stats.LastStdoutBytes ||
			(currentStderrBytes > stats.LastStderrBytes && currentStderrBytes > 4096) {
			// Got new data or stderr > 4KB (environ is usually < 4KB)
			newStreamData = true
			stats.StaticDataCount = 0
		} else if currentStdinBytes == 0 && currentStdoutBytes == 0 && currentStderrBytes > 0 && currentStderrBytes < 4096 {
			// Likely just environ data
			stats.StaticDataCount++
		}

		// Update byte counters
		stats.LastStdinBytes = currentStdinBytes
		stats.LastStdoutBytes = currentStdoutBytes
		stats.LastStderrBytes = currentStderrBytes

		if newStreamData {
			// Real stream data detected
			stats.Successful++
			stats.ConsecutiveFails = 0
			stats.LastSuccess = time.Now()
			stats.BytesCapured += currentStdinBytes + currentStdoutBytes + currentStderrBytes
			return data, nil
		}
	}

	// No new stream data or error
	stats.ConsecutiveFails++

	// Check if we should try strace fallback
	shouldUseStrace := false
	if e.straceEnabled && e.mode != "procfs" {
		// Trigger strace if:
		// 1. Process is using PTY
		// 2. Getting only static data repeatedly
		// 3. Too many consecutive failures
		if stats.IsUsingPTY {
			shouldUseStrace = true
			fmt.Printf("\033[33mInfo: Process %d is using PTY, switching to strace\033[0m\n", pid)
		} else if stats.StaticDataCount > 5 {
			shouldUseStrace = true
			fmt.Printf("\033[33mInfo: Only static data from PID %d, switching to strace\033[0m\n", pid)
		} else if stats.ConsecutiveFails > 10 {
			shouldUseStrace = true
			fmt.Printf("\033[33mWarning: Switching to strace for PID %d after %d failed attempts\033[0m\n", pid, stats.ConsecutiveFails)
		}
	}

	if shouldUseStrace {
		// Try to start strace capture
		if err := e.startStraceCapture(pid); err != nil {
			return data, fmt.Errorf("strace fallback failed: %w", err)
		}

		// Update stats
		stats.Method = "strace"
		stats.ConsecutiveFails = 0
		stats.StaticDataCount = 0

		// Try reading from strace
		e.mu.RLock()
		if captor, ok := e.straceCaptors[pid]; ok {
			e.mu.RUnlock()
			return captor.Read()
		}
		e.mu.RUnlock()
	} else if e.straceEnabled && stats.ConsecutiveFails > 0 {
		// Debug output for tracking progress
		if stats.ConsecutiveFails%5 == 0 {
			fmt.Printf("Debug: PID %d - fails: %d, static: %d, PTY: %v\n",
				pid, stats.ConsecutiveFails, stats.StaticDataCount, stats.IsUsingPTY)
		}
	}

	return data, err
}

// readProcFS reads streams using the procfs method.
func (e *CaptureEngine) readProcFS(pid int) (*StreamData, error) {
	e.mu.RLock()
	capture, exists := e.activeProcs[pid]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("not attached to PID %d", pid)
	}

	data := &StreamData{Timestamp: time.Now()}

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
