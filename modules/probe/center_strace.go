package probe

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// StraceCapture provides PTY-aware capture using strace.
type StraceCapture struct {
	pid        int
	cmd        *exec.Cmd
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	parser     *StraceParser
	active     bool
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	lastError  error
	startTime  time.Time
	syscalls   int64
	bytesTotal int64

	// Accumulated stream data
	stdinBuffer  *StreamBuffer
	stdoutBuffer *StreamBuffer
	stderrBuffer *StreamBuffer
	unknownFDs   map[int]*StreamBuffer

	// Configuration
	maxBufferSize      int           // Max bytes per buffer
	maxTotalBytes      int64         // Max total bytes before stopping
	rateLimit          int           // Max bytes per second (0 = unlimited)
	rateLimitWindow    time.Duration // Rate limit window
	lastRateLimitCheck time.Time
	bytesInWindow      int64
}

// StraceConfig holds configuration for strace capture.
type StraceConfig struct {
	MaxBufferSize int   // Max size per stream buffer (default: 1MB)
	MaxTotalBytes int64 // Max total bytes before stopping (default: 100MB)
	RateLimit     int   // Max bytes per second (0 = unlimited)
}

// DefaultStraceConfig returns default strace configuration.
func DefaultStraceConfig() *StraceConfig {
	return &StraceConfig{
		MaxBufferSize: 1024 * 1024,       // 1MB per buffer
		MaxTotalBytes: 100 * 1024 * 1024, // 100MB total
		RateLimit:     0,                 // Unlimited by default
	}
}

// StraceParser parses strace output for stream data.
type StraceParser struct {
	// Patterns for syscall parsing
	readPattern  *regexp.Regexp
	writePattern *regexp.Regexp
	recvPattern  *regexp.Regexp
	sendPattern  *regexp.Regexp
}

// NewStraceCapture creates a new strace-based capture engine.
func NewStraceCapture(pid int) (*StraceCapture, error) {
	return NewStraceCaptureWithConfig(pid, DefaultStraceConfig())
}

// NewStraceCaptureWithConfig creates a new strace-based capture engine with custom config.
func NewStraceCaptureWithConfig(pid int, config *StraceConfig) (*StraceCapture, error) {
	// Check if strace is available
	if err := checkStraceAvailable(); err != nil {
		return nil, fmt.Errorf("strace not available: %w", err)
	}

	// Check if we can ptrace the target process
	if err := checkPtracePermission(pid); err != nil {
		return nil, fmt.Errorf("insufficient permissions for ptrace: %w", err)
	}

	// Validate configuration
	if config.MaxBufferSize <= 0 {
		config.MaxBufferSize = 1024 * 1024 // 1MB default
	}
	if config.MaxTotalBytes <= 0 {
		config.MaxTotalBytes = 100 * 1024 * 1024 // 100MB default
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Use configured buffer size, but cap at reasonable limit
	bufferSize := config.MaxBufferSize
	if bufferSize > 10*1024*1024 { // Cap at 10MB per buffer
		bufferSize = 10 * 1024 * 1024
	}

	s := &StraceCapture{
		pid:                pid,
		parser:             NewStraceParser(),
		ctx:                ctx,
		cancel:             cancel,
		startTime:          time.Now(),
		stdinBuffer:        NewStreamBuffer(bufferSize),
		stdoutBuffer:       NewStreamBuffer(bufferSize),
		stderrBuffer:       NewStreamBuffer(bufferSize),
		unknownFDs:         make(map[int]*StreamBuffer),
		maxBufferSize:      config.MaxBufferSize,
		maxTotalBytes:      config.MaxTotalBytes,
		rateLimit:          config.RateLimit,
		rateLimitWindow:    time.Second,
		lastRateLimitCheck: time.Now(),
	}

	// Set event delimiters for better context preservation
	// For stdout/stderr, use newline as event boundary
	s.stdoutBuffer.SetEventDelimiter([]byte("\n"))
	s.stderrBuffer.SetEventDelimiter([]byte("\n"))
	// For stdin, could be more complex (e.g., commands might have different delimiters)
	s.stdinBuffer.SetEventDelimiter([]byte("\n"))

	return s, nil
}

// NewStraceParser creates a parser for strace output.
func NewStraceParser() *StraceParser {
	return &StraceParser{
		// Match: read(3, "data", 1024) = 4
		readPattern: regexp.MustCompile(`^read\((\d+),\s*"(.+?)",\s*\d+\)\s*=\s*(\d+)`),
		// Match: write(1, "data", 4) = 4
		writePattern: regexp.MustCompile(`^write\((\d+),\s*"(.+?)",\s*\d+\)\s*=\s*(\d+)`),
		// Match: recvfrom(3, "data", 1024, 0, NULL, NULL) = 4
		recvPattern: regexp.MustCompile(`^recvfrom\((\d+),\s*"(.+?)",\s*\d+,.*\)\s*=\s*(\d+)`),
		// Match: sendto(3, "data", 4, 0, NULL, 0) = 4
		sendPattern: regexp.MustCompile(`^sendto\((\d+),\s*"(.+?)",\s*\d+,.*\)\s*=\s*(\d+)`),
	}
}

// Start begins strace capture.
func (s *StraceCapture) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return fmt.Errorf("strace capture already active")
	}

	// Build strace command
	// -p PID: attach to process
	// -s 1024: string size limit (we'll start conservative)
	// -e trace=read,write,recv,recvfrom,send,sendto: syscalls to trace
	// -f: follow forks
	// -q: suppress attach/detach messages
	// -tt: absolute timestamps with microseconds
	args := []string{
		"-p", strconv.Itoa(s.pid),
		"-s", "1024",
		"-e", "trace=read,write,recv,recvfrom,send,sendto",
		"-f",
		"-q",
		"-tt",
	}

	s.cmd = exec.CommandContext(s.ctx, "strace", args...)

	// Get pipes for output
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	s.stdout = stdout

	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	s.stderr = stderr

	// Start strace
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start strace: %w", err)
	}

	s.active = true
	fmt.Printf("Debug: strace started for PID %d (strace PID: %d)\n", s.pid, s.cmd.Process.Pid)

	// Start output processing
	go s.processOutput()

	// Monitor strace process health
	go func() {
		err := s.cmd.Wait()
		s.mu.Lock()
		defer s.mu.Unlock()

		if err != nil && s.lastError == nil {
			// Only set error if we don't already have one
			exitErr, ok := err.(*exec.ExitError)
			if ok && exitErr.ExitCode() == 1 {
				// Exit code 1 often means the traced process exited
				s.lastError = fmt.Errorf("traced process may have exited")
			} else {
				s.lastError = fmt.Errorf("strace exited with error: %w", err)
			}
		}
		s.active = false
		fmt.Printf("Debug: strace process ended for PID %d\n", s.pid)
	}()

	// Set up signal handling for clean shutdown
	go s.handleSignals()

	// Give strace a moment to attach
	time.Sleep(50 * time.Millisecond)

	// Verify strace is still running
	if s.cmd.ProcessState != nil && s.cmd.ProcessState.Exited() {
		return fmt.Errorf("strace exited immediately - process may not exist")
	}

	return nil
}

// handleSignals sets up signal handling for clean shutdown.
func (s *StraceCapture) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		fmt.Printf("Debug: Received signal %v, stopping strace capture\n", sig)
		_ = s.Stop()
	case <-s.ctx.Done():
		// Context cancelled, stop listening for signals
		signal.Stop(sigChan)
	}
}

// Stop terminates strace capture.
func (s *StraceCapture) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return nil
	}

	// Cancel context to stop processOutput
	s.cancel()

	// Try graceful termination first
	if s.cmd != nil && s.cmd.Process != nil {
		// Send SIGTERM for graceful shutdown
		if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// If SIGTERM fails, try SIGKILL
			if err := s.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to stop strace: %w", err)
			}
		}

		// Give strace a moment to detach cleanly
		done := make(chan error, 1)
		go func() {
			done <- s.cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited cleanly
		case <-time.After(2 * time.Second):
			// Force kill if still running
			_ = s.cmd.Process.Kill()
		}
	}

	s.active = false
	return nil
}

// processOutput reads and parses strace output.
func (s *StraceCapture) processOutput() {
	defer func() {
		// Handle any panics
		if r := recover(); r != nil {
			s.mu.Lock()
			s.lastError = fmt.Errorf("strace processing panic: %v", r)
			s.active = false
			s.mu.Unlock()
			fmt.Printf("Error: Strace processing panic recovered: %v\n", r)
		}
	}()

	// strace outputs to stderr
	scanner := bufio.NewScanner(s.stderr)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	lineCount := 0
	consecutiveErrors := 0
	maxConsecutiveErrors := 10

	for scanner.Scan() {
		// Check if context is cancelled
		select {
		case <-s.ctx.Done():
			fmt.Printf("Debug: strace processOutput cancelled\n")
			return
		default:
		}

		line := scanner.Text()
		lineCount++
		if lineCount <= 5 {
			fmt.Printf("Debug: strace line %d: %s\n", lineCount, line)
		}

		// Parse line with error recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					consecutiveErrors++
					if consecutiveErrors >= maxConsecutiveErrors {
						s.mu.Lock()
						s.lastError = fmt.Errorf("too many parse errors: %v", r)
						s.mu.Unlock()
						fmt.Printf("Error: Too many strace parse errors, last: %v\n", r)
					}
				} else {
					consecutiveErrors = 0 // Reset on successful parse
				}
			}()
			s.parseLine(line)
		}()

		// Stop if too many errors
		if consecutiveErrors >= maxConsecutiveErrors {
			break
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		s.mu.Lock()
		s.lastError = fmt.Errorf("strace output error: %w", err)
		s.mu.Unlock()
		fmt.Printf("Debug: strace scanner error: %v\n", err)
	}

	// Mark as inactive when done
	s.mu.Lock()
	s.active = false
	s.mu.Unlock()

	fmt.Printf("Debug: strace processOutput finished after %d lines\n", lineCount)
}

// parseLine parses a single line of strace output.
func (s *StraceCapture) parseLine(line string) {
	// Skip empty lines and timestamps
	if line == "" || strings.HasPrefix(line, "[pid") {
		return
	}

	// Remove timestamp prefix if present
	if idx := strings.Index(line, " "); idx > 0 && strings.Contains(line[:idx], ".") {
		line = line[idx+1:]
	}

	// Parse syscalls
	parsed := s.parser.ParseSyscall(line)
	if parsed != nil {
		s.mu.Lock()
		defer s.mu.Unlock()

		// Check if we've exceeded max total bytes
		if s.maxTotalBytes > 0 && s.bytesTotal >= s.maxTotalBytes {
			if s.lastError == nil {
				s.lastError = fmt.Errorf("max total bytes limit reached (%d bytes)", s.maxTotalBytes)
				fmt.Printf("Warning: Strace capture stopped - max bytes limit reached\n")
				go func() { _ = s.Stop() }() // Stop capture in background
			}
			return
		}

		// Check rate limit
		if s.rateLimit > 0 {
			now := time.Now()
			if now.Sub(s.lastRateLimitCheck) >= s.rateLimitWindow {
				// Reset window
				s.bytesInWindow = 0
				s.lastRateLimitCheck = now
			}

			// Check if adding this data would exceed rate limit
			if s.bytesInWindow+int64(parsed.DataLen) > int64(s.rateLimit) {
				// Drop this data to maintain rate limit
				return
			}
			s.bytesInWindow += int64(parsed.DataLen)
		}

		s.syscalls++
		s.bytesTotal += int64(parsed.DataLen)

		// Accumulate data in appropriate buffer
		switch parsed.Stream {
		case "stdin":
			s.stdinBuffer.Write(parsed.Data)
		case "stdout":
			s.stdoutBuffer.Write(parsed.Data)
		case "stderr":
			s.stderrBuffer.Write(parsed.Data)
		default:
			// Track unknown FDs with limited buffer size
			if _, ok := s.unknownFDs[parsed.FD]; !ok {
				bufSize := s.maxBufferSize
				if bufSize > 1024*1024 { // Cap unknown FDs at 1MB
					bufSize = 1024 * 1024
				}
				s.unknownFDs[parsed.FD] = NewStreamBuffer(bufSize)
			}
			s.unknownFDs[parsed.FD].Write(parsed.Data)
		}
	}
}

// Read retrieves captured stream data.
func (s *StraceCapture) Read() (*StreamData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for errors first
	if s.lastError != nil {
		// Return any buffered data along with the error
		data := &StreamData{
			Timestamp: time.Now(),
			Stdin:     s.stdinBuffer.ReadAll(),
			Stdout:    s.stdoutBuffer.ReadAll(),
			Stderr:    s.stderrBuffer.ReadAll(),
		}
		// Return data if available, even with error
		if len(data.Stdin) > 0 || len(data.Stdout) > 0 || len(data.Stderr) > 0 {
			return data, nil
		}
		return nil, s.lastError
	}

	if !s.active && s.bytesTotal == 0 {
		return nil, fmt.Errorf("strace capture not active or no data captured")
	}

	// Read and clear buffers
	stdinData := s.stdinBuffer.ReadAll()
	stdoutData := s.stdoutBuffer.ReadAll()
	stderrData := s.stderrBuffer.ReadAll()

	return &StreamData{
		Timestamp: time.Now(),
		Stdin:     stdinData,
		Stdout:    stdoutData,
		Stderr:    stderrData,
	}, nil
}

// GetStats returns capture statistics.
func (s *StraceCapture) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"method":          "strace",
		"pid":             s.pid,
		"active":          s.active,
		"duration":        time.Since(s.startTime),
		"syscalls":        s.syscalls,
		"bytes_total":     s.bytesTotal,
		"max_buffer_size": s.maxBufferSize,
		"max_total_bytes": s.maxTotalBytes,
		"rate_limit":      s.rateLimit,
		"last_error":      s.lastError,
	}

	// Add buffer utilization stats
	if s.stdinBuffer != nil {
		stats["stdin_buffer_used"] = s.stdinBuffer.Available()
	}
	if s.stdoutBuffer != nil {
		stats["stdout_buffer_used"] = s.stdoutBuffer.Available()
	}
	if s.stderrBuffer != nil {
		stats["stderr_buffer_used"] = s.stderrBuffer.Available()
	}

	// Check if we're near limits
	if s.maxTotalBytes > 0 {
		stats["bytes_limit_percent"] = float64(s.bytesTotal) / float64(s.maxTotalBytes) * 100
	}

	return stats
}

// ParsedSyscall represents a parsed syscall with stream data.
type ParsedSyscall struct {
	Syscall   string
	FD        int
	Data      []byte
	DataLen   int
	Timestamp time.Time
	Stream    string // stdin, stdout, stderr, or unknown
}

// ParseSyscall parses a syscall line from strace output.
func (p *StraceParser) ParseSyscall(line string) *ParsedSyscall {
	// Try read syscall
	if matches := p.readPattern.FindStringSubmatch(line); matches != nil {
		return p.parseIOSyscall("read", matches)
	}

	// Try write syscall
	if matches := p.writePattern.FindStringSubmatch(line); matches != nil {
		return p.parseIOSyscall("write", matches)
	}

	// Try recvfrom syscall
	if matches := p.recvPattern.FindStringSubmatch(line); matches != nil {
		return p.parseIOSyscall("recvfrom", matches)
	}

	// Try sendto syscall
	if matches := p.sendPattern.FindStringSubmatch(line); matches != nil {
		return p.parseIOSyscall("sendto", matches)
	}

	return nil
}

// parseIOSyscall parses read/write syscall matches.
func (p *StraceParser) parseIOSyscall(syscall string, matches []string) *ParsedSyscall {
	// Validate input
	if len(matches) < 4 {
		return nil
	}

	// Safely parse FD
	fd, err := strconv.Atoi(matches[1])
	if err != nil || fd < 0 {
		return nil
	}

	// Get data string
	dataStr := matches[2]
	if len(dataStr) > 10*1024*1024 { // Sanity check: 10MB max
		// Truncate extremely long strings
		dataStr = dataStr[:1024] + "...[truncated]"
	}

	// Safely parse data length
	dataLen, err := strconv.Atoi(matches[3])
	if err != nil || dataLen < 0 {
		return nil
	}

	// Decode the data string with error recovery
	data := func() []byte {
		defer func() {
			if r := recover(); r != nil {
				// Return empty data on decode panic
				fmt.Printf("Warning: Failed to decode strace string: %v\n", r)
			}
		}()
		return p.decodeStraceString(dataStr)
	}()

	// Validate decoded data length
	if len(data) > dataLen {
		// Truncate if decoded data is longer than reported
		data = data[:dataLen]
	}

	// Determine stream type
	stream := "unknown"
	switch fd {
	case 0:
		stream = "stdin"
	case 1:
		stream = "stdout"
	case 2:
		stream = "stderr"
	}

	return &ParsedSyscall{
		Syscall:   syscall,
		FD:        fd,
		Data:      data,
		DataLen:   dataLen,
		Timestamp: time.Now(),
		Stream:    stream,
	}
}

// decodeStraceString decodes strace's string representation.
func (p *StraceParser) decodeStraceString(s string) []byte {
	var result bytes.Buffer

	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
				i += 2
			case 'r':
				result.WriteByte('\r')
				i += 2
			case 't':
				result.WriteByte('\t')
				i += 2
			case '\\':
				result.WriteByte('\\')
				i += 2
			case '"':
				result.WriteByte('"')
				i += 2
			case 'x':
				// Hex escape sequence
				if i+3 < len(s) {
					if b, err := hex.DecodeString(s[i+2 : i+4]); err == nil && len(b) > 0 {
						result.WriteByte(b[0])
						i += 4
					} else {
						result.WriteByte(s[i])
						i++
					}
				} else {
					result.WriteByte(s[i])
					i++
				}
			default:
				// Octal escape sequence
				if i+3 < len(s) && s[i+1] >= '0' && s[i+1] <= '7' {
					if val, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
						result.WriteByte(byte(val))
						i += 4
					} else {
						result.WriteByte(s[i])
						i++
					}
				} else {
					result.WriteByte(s[i])
					i++
				}
			}
		} else {
			result.WriteByte(s[i])
			i++
		}
	}

	return result.Bytes()
}

// checkStraceAvailable verifies strace is installed and accessible.
func checkStraceAvailable() error {
	cmd := exec.Command("strace", "-V")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("strace not found or not executable: %w", err)
	}

	if !strings.Contains(string(output), "strace") {
		return fmt.Errorf("unexpected strace output")
	}

	return nil
}

// checkPtracePermission verifies we can ptrace the target process.
func checkPtracePermission(pid int) error {
	// Check if process exists
	// #nosec G204 - pid is validated integer from our own process tracking
	if err := exec.Command("kill", "-0", strconv.Itoa(pid)).Run(); err != nil {
		return fmt.Errorf("process %d not found or not accessible", pid)
	}

	// Check ptrace_scope
	// Note: This is a simplified check. Real implementation would need
	// more sophisticated permission checking

	return nil
}
