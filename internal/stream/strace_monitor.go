package stream

import (
    "bufio"
    "context"
    "fmt"
    "io"
    "os"
    "os/exec"
    "regexp"
    "strconv"
    "strings"
    "sync"
    "time"
)

// StraceMonitor monitors process STDIO using strace
type StraceMonitor struct {
    pid          int
    processName  string
    outputWriter OutputWriter
    patterns     []SecurityPattern
    
    mu           sync.Mutex
    cmd          *exec.Cmd
    cancelFunc   context.CancelFunc
    running      bool
    eventCount   int
    alertCount   int
    doneChan     chan struct{}
}

// NewStraceMonitor creates a new strace monitor
func NewStraceMonitor(pid int, name string, writer OutputWriter, patterns []SecurityPattern) *StraceMonitor {
    return &StraceMonitor{
        pid:          pid,
        processName:  name,
        outputWriter: writer,
        patterns:     patterns,
    }
}

// Start begins monitoring the process
func (m *StraceMonitor) Start(ctx context.Context) error {
    m.mu.Lock()
    if m.running {
        m.mu.Unlock()
        return fmt.Errorf("monitor already running")
    }
    
    // Create cancellable context
    ctx, cancel := context.WithCancel(ctx)
    m.cancelFunc = cancel
    m.doneChan = make(chan struct{})
    m.running = true
    m.mu.Unlock()
    
    // Build strace command
    // -p PID: attach to process
    // -s 1024: string size (increase for larger payloads)
    // -e trace=read,write,send,recv,sendto,recvfrom: trace I/O calls
    // -f: follow forks
    // -tt: absolute timestamps
    args := []string{
        "-p", strconv.Itoa(m.pid),
        "-s", "1024",
        "-e", "trace=read,write,send,recv,sendto,recvfrom",
        "-f",
        "-tt",
    }
    
    m.cmd = exec.CommandContext(ctx, "strace", args...)
    
    // Get stderr pipe (strace outputs to stderr)
    stderr, err := m.cmd.StderrPipe()
    if err != nil {
        m.mu.Lock()
        m.running = false
        m.cancelFunc()
        m.mu.Unlock()
        return fmt.Errorf("failed to get stderr pipe: %w", err)
    }
    
    // Start strace
    if err := m.cmd.Start(); err != nil {
        m.mu.Lock()
        m.running = false
        m.cancelFunc()
        m.mu.Unlock()
        return fmt.Errorf("failed to start strace: %w", err)
    }
    
    // Process output in goroutine
    go func() {
        defer func() {
            m.mu.Lock()
            m.running = false
            if m.cancelFunc != nil {
                m.cancelFunc()
            }
            m.mu.Unlock()
            close(m.doneChan)
        }()
        
        m.processOutput(ctx, stderr)
        
        // Wait for process to complete
        m.cmd.Wait()
    }()
    
    return nil
}

// Stop halts monitoring
func (m *StraceMonitor) Stop() error {
    m.mu.Lock()
    if !m.running {
        m.mu.Unlock()
        return nil
    }
    
    // Cancel context to stop reading
    if m.cancelFunc != nil {
        m.cancelFunc()
    }
    m.mu.Unlock()
    
    // Send interrupt signal
    if m.cmd != nil && m.cmd.Process != nil {
        // Try graceful interrupt first
        m.cmd.Process.Signal(os.Interrupt)
        
        // Wait for graceful shutdown
        select {
        case <-m.doneChan:
            return nil
        case <-time.After(3 * time.Second):
            // Force kill if still running
            m.cmd.Process.Kill()
        }
    }
    
    // Wait for done signal
    select {
    case <-m.doneChan:
    case <-time.After(5 * time.Second):
        return fmt.Errorf("timeout waiting for monitor to stop")
    }
    
    return nil
}

// GetStats returns monitoring statistics
func (m *StraceMonitor) GetStats() (events int, alerts int) {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.eventCount, m.alertCount
}

// parseSyscall parses an strace line into a stream event
func (m *StraceMonitor) parseSyscall(line string) *StreamEvent {
    // Example strace output:
    // 12:34:56.123456 [pid 1234] write(1, "Hello, World!\n", 14) = 14
    // 12:34:56.123456 read(0, "input data", 1024) = 10
    
    // Skip non-syscall lines
    if !strings.Contains(line, "(") || !strings.Contains(line, ")") {
        return nil
    }
    
    // Parse PID if present
    pid := m.pid
    if strings.Contains(line, "[pid ") {
        pidMatch := regexp.MustCompile(`\[pid (\d+)\]`).FindStringSubmatch(line)
        if len(pidMatch) > 1 {
            if p, err := strconv.Atoi(pidMatch[1]); err == nil {
                pid = p
            }
        }
    }
    
    // Simple pattern to extract syscall data
    // Match: syscall(fd, "data", ...) = result
    syscallRe := regexp.MustCompile(`(read|write|send|recv|sendto|recvfrom)\((\d+),\s*"([^"]*)"`)
    matches := syscallRe.FindStringSubmatch(line)
    if len(matches) < 4 {
        return nil
    }
    
    syscall := matches[1]
    fd, _ := strconv.Atoi(matches[2])
    data := matches[3]
    
    // Extract result size
    resultRe := regexp.MustCompile(`\)\s*=\s*(\d+)`)
    resultMatch := resultRe.FindStringSubmatch(line)
    size := 0
    if len(resultMatch) > 1 {
        size, _ = strconv.Atoi(resultMatch[1])
    }
    
    // Skip failed syscalls
    if size <= 0 {
        return nil
    }
    
    // Determine direction based on syscall
    direction := DirectionInbound
    eventType := StreamEventRead
    
    switch syscall {
    case "write", "send", "sendto":
        direction = DirectionOutbound
        eventType = StreamEventWrite
    case "read", "recv", "recvfrom":
        direction = DirectionInbound
        eventType = StreamEventRead
    }
    
    // Unescape string data
    data = unescapeStraceString(data)
    
    return &StreamEvent{
        Timestamp:   time.Now(),
        Type:        eventType,
        Direction:   direction,
        PID:         pid,
        ProcessName: m.processName,
        FD:          fd,
        Data:        []byte(data),
        Size:        size,
        Summary:     fmt.Sprintf("%s(%d) %d bytes", syscall, fd, size),
        Metadata: map[string]interface{}{
            "syscall": syscall,
            "fd_type": getFDType(fd),
        },
    }
}

// processOutput reads and processes strace output
func (m *StraceMonitor) processOutput(ctx context.Context, reader io.Reader) {
    scanner := bufio.NewScanner(reader)
    // Increase buffer size for large syscall outputs
    scanner.Buffer(make([]byte, 64*1024), 256*1024)
    
    for scanner.Scan() {
        select {
        case <-ctx.Done():
            return
        default:
            line := scanner.Text()
            if event := m.parseSyscall(line); event != nil {
                m.processEvent(event)
            }
        }
    }
    
    if err := scanner.Err(); err != nil {
        fmt.Printf("Error reading strace output: %v\n", err)
    }
}

// processEvent handles a parsed event
func (m *StraceMonitor) processEvent(event *StreamEvent) {
    m.mu.Lock()
    m.eventCount++
    m.mu.Unlock()
    
    // Write event
    if err := m.outputWriter.WriteEvent(event); err != nil {
        // Log error but continue
        fmt.Printf("Error writing event: %v\n", err)
    }
    
    // Check for security patterns
    for _, pattern := range m.patterns {
        if pattern.Matches(event.Data) {
            alert := &SecurityAlert{
                Timestamp:   event.Timestamp,
                EventID:     fmt.Sprintf("evt_%d_%d", event.PID, event.Timestamp.UnixNano()),
                Pattern:     pattern.Name,
                Severity:    pattern.Severity,
                Title:       fmt.Sprintf("Security Pattern Detected: %s", pattern.Name),
                Description: pattern.Description,
                PID:         event.PID,
                ProcessName: event.ProcessName,
                Evidence:    string(event.Data),
                Category:    pattern.Category,
                Blocked:     false,
            }
            
            m.mu.Lock()
            m.alertCount++
            m.mu.Unlock()
            
            if err := m.outputWriter.WriteAlert(alert); err != nil {
                fmt.Printf("Error writing alert: %v\n", err)
            }
        }
    }
}

// unescapeStraceString converts strace escaped strings to normal strings
func unescapeStraceString(s string) string {
    // Handle common escape sequences
    s = strings.ReplaceAll(s, `\n`, "\n")
    s = strings.ReplaceAll(s, `\r`, "\r")
    s = strings.ReplaceAll(s, `\t`, "\t")
    s = strings.ReplaceAll(s, `\\`, "\\")
    s = strings.ReplaceAll(s, `\"`, "\"")
    
    // Handle hex escapes like \x0a
    hexRe := regexp.MustCompile(`\\x([0-9a-fA-F]{2})`)
    s = hexRe.ReplaceAllStringFunc(s, func(match string) string {
        hex := match[2:]
        if b, err := strconv.ParseUint(hex, 16, 8); err == nil {
            return string(byte(b))
        }
        return match
    })
    
    // Handle octal escapes like \012
    octalRe := regexp.MustCompile(`\\([0-7]{1,3})`)
    s = octalRe.ReplaceAllStringFunc(s, func(match string) string {
        octal := match[1:]
        if b, err := strconv.ParseUint(octal, 8, 8); err == nil {
            return string(byte(b))
        }
        return match
    })
    
    return s
}

// getFDType returns the type of file descriptor
func getFDType(fd int) string {
    switch fd {
    case 0:
        return "stdin"
    case 1:
        return "stdout"
    case 2:
        return "stderr"
    default:
        if fd <= 2 {
            return "stdio"
        }
        return "socket" // Could be socket, file, pipe, etc.
    }
}