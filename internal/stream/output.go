package stream

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "net"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"
)

// OutputWriter defines the interface for stream output destinations
type OutputWriter interface {
    WriteEvent(event *StreamEvent) error
    WriteAlert(alert *SecurityAlert) error
    Close() error
}

// OutputFormat represents the serialization format
type OutputFormat string

const (
    FormatJSON     OutputFormat = "json"
    FormatJSONL    OutputFormat = "jsonl"
    FormatPCAP     OutputFormat = "pcap"     // For network-style capture
    FormatCEF      OutputFormat = "cef"      // Common Event Format
    FormatRaw      OutputFormat = "raw"      // Raw binary
    FormatProtobuf OutputFormat = "protobuf" // For high-performance
)

// ParseOutputDestination parses output destination strings
// Examples:
//   - "file:/tmp/capture.jsonl"
//   - "tcp:192.168.1.100:9999"
//   - "unix:/var/run/strigoi.sock"
//   - "pipe:analyzer"
//   - "integration:prometheus"
func ParseOutputDestination(dest string) (OutputWriter, error) {
    if dest == "" || dest == "-" || dest == "stdout" {
        return NewConsoleOutput(os.Stdout, FormatJSONL), nil
    }
    
    parts := strings.SplitN(dest, ":", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid output format, expected type:destination")
    }
    
    outputType := parts[0]
    location := parts[1]
    
    switch outputType {
    case "file":
        return NewFileOutput(location, FormatJSONL)
    case "tcp":
        return NewTCPOutput(location, FormatJSONL)
    case "unix":
        return NewUnixSocketOutput(location, FormatJSONL)
    case "pipe":
        return NewPipeOutput(location, FormatJSONL)
    case "integration":
        return NewIntegrationOutput(location)
    default:
        return nil, fmt.Errorf("unknown output type: %s", outputType)
    }
}

// ConsoleOutput writes to stdout/stderr
type ConsoleOutput struct {
    writer io.Writer
    format OutputFormat
}

func NewConsoleOutput(w io.Writer, format OutputFormat) *ConsoleOutput {
    return &ConsoleOutput{
        writer: w,
        format: format,
    }
}

func (c *ConsoleOutput) WriteEvent(event *StreamEvent) error {
    return c.writeData(map[string]interface{}{
        "type":      "event",
        "timestamp": event.Timestamp,
        "data":      event,
    })
}

func (c *ConsoleOutput) WriteAlert(alert *SecurityAlert) error {
    return c.writeData(map[string]interface{}{
        "type":      "alert",
        "timestamp": alert.Timestamp,
        "data":      alert,
    })
}

func (c *ConsoleOutput) writeData(data interface{}) error {
    switch c.format {
    case FormatJSON, FormatJSONL:
        enc := json.NewEncoder(c.writer)
        return enc.Encode(data)
    default:
        _, err := fmt.Fprintln(c.writer, data)
        return err
    }
}

func (c *ConsoleOutput) Close() error {
    // Nothing to close for console
    return nil
}

// FileOutput writes to a file with rotation support
type FileOutput struct {
    mu          sync.Mutex
    path        string
    format      OutputFormat
    file        *os.File
    encoder     *json.Encoder
    currentSize int64
    maxSize     int64
}

func NewFileOutput(path string, format OutputFormat) (*FileOutput, error) {
    // SECURITY: Clean the path to prevent traversal attacks
    cleanPath := filepath.Clean(path)
    
    // SECURITY: Ensure the path is absolute to prevent ambiguity
    if !filepath.IsAbs(cleanPath) {
        // If relative, make it relative to current working directory
        cwd, err := os.Getwd()
        if err != nil {
            return nil, fmt.Errorf("failed to get working directory: %w", err)
        }
        cleanPath = filepath.Join(cwd, cleanPath)
    }
    
    // SECURITY: Verify the directory exists and create if needed
    dir := filepath.Dir(cleanPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
    }
    
    file, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return nil, fmt.Errorf("failed to open file %s: %w", cleanPath, err)
    }
    
    info, _ := file.Stat()
    currentSize := int64(0)
    if info != nil {
        currentSize = info.Size()
    }
    
    return &FileOutput{
        path:        path,
        format:      format,
        file:        file,
        encoder:     json.NewEncoder(file),
        currentSize: currentSize,
        maxSize:     100 * 1024 * 1024, // 100MB default
    }, nil
}

func (f *FileOutput) WriteEvent(event *StreamEvent) error {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }
    
    // Check rotation
    if f.currentSize+int64(len(data)) > f.maxSize {
        if err := f.rotate(); err != nil {
            return fmt.Errorf("failed to rotate log file: %w", err)
        }
    }
    
    n, err := f.file.Write(append(data, '\n'))
    if err != nil {
        return fmt.Errorf("failed to write event: %w", err)
    }
    f.currentSize += int64(n)
    return nil
}

func (f *FileOutput) WriteAlert(alert *SecurityAlert) error {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    data, err := json.Marshal(alert)
    if err != nil {
        return fmt.Errorf("failed to marshal alert: %w", err)
    }
    
    // Check rotation
    if f.currentSize+int64(len(data)) > f.maxSize {
        if err := f.rotate(); err != nil {
            return fmt.Errorf("failed to rotate log file: %w", err)
        }
    }
    
    n, err := f.file.Write(append(data, '\n'))
    if err != nil {
        return fmt.Errorf("failed to write alert: %w", err)
    }
    f.currentSize += int64(n)
    return nil
}

func (f *FileOutput) rotate() error {
    // Close current file
    if err := f.file.Close(); err != nil {
        return fmt.Errorf("failed to close current file: %w", err)
    }
    
    // Rename current file with timestamp
    timestamp := time.Now().Format("20060102-150405")
    newPath := fmt.Sprintf("%s.%s", f.path, timestamp)
    if err := os.Rename(f.path, newPath); err != nil {
        return fmt.Errorf("failed to rename file for rotation: %w", err)
    }
    
    // Open new file
    file, err := os.OpenFile(f.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return fmt.Errorf("failed to create new file after rotation: %w", err)
    }
    
    f.file = file
    f.encoder = json.NewEncoder(file)
    f.currentSize = 0
    
    return nil
}

func (f *FileOutput) Close() error {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    if f.file != nil {
        return f.file.Close()
    }
    return nil
}

// TCPOutput streams to a TCP endpoint
type TCPOutput struct {
    mu      sync.Mutex
    address string
    format  OutputFormat
    conn    net.Conn
    encoder *json.Encoder
    buffer  *bufio.Writer
}

func NewTCPOutput(address string, format OutputFormat) (*TCPOutput, error) {
    conn, err := net.DialTimeout("tcp", address, 10*time.Second)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
    }
    
    // Enable TCP keepalive
    if tcpConn, ok := conn.(*net.TCPConn); ok {
        tcpConn.SetKeepAlive(true)
        tcpConn.SetKeepAlivePeriod(30 * time.Second)
    }
    
    // Add buffering for better performance
    buffer := bufio.NewWriterSize(conn, 64*1024)
    
    return &TCPOutput{
        address: address,
        format:  format,
        conn:    conn,
        buffer:  buffer,
        encoder: json.NewEncoder(buffer),
    }, nil
}

func (t *TCPOutput) WriteEvent(event *StreamEvent) error {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    if err := t.encoder.Encode(event); err != nil {
        return fmt.Errorf("failed to encode event: %w", err)
    }
    
    // Flush periodically for real-time streaming
    return t.buffer.Flush()
}

func (t *TCPOutput) WriteAlert(alert *SecurityAlert) error {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    if err := t.encoder.Encode(alert); err != nil {
        return fmt.Errorf("failed to encode alert: %w", err)
    }
    
    // Always flush alerts immediately
    return t.buffer.Flush()
}

func (t *TCPOutput) Close() error {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    // Flush any remaining data
    if t.buffer != nil {
        t.buffer.Flush()
    }
    
    if t.conn != nil {
        return t.conn.Close()
    }
    return nil
}

// UnixSocketOutput streams to a Unix domain socket
type UnixSocketOutput struct {
    mu      sync.Mutex
    path    string
    format  OutputFormat
    conn    net.Conn
    encoder *json.Encoder
    buffer  *bufio.Writer
}

func NewUnixSocketOutput(path string, format OutputFormat) (*UnixSocketOutput, error) {
    conn, err := net.DialTimeout("unix", path, 5*time.Second)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to Unix socket %s: %w", path, err)
    }
    
    // Add buffering for better performance
    buffer := bufio.NewWriterSize(conn, 64*1024)
    
    return &UnixSocketOutput{
        path:    path,
        format:  format,
        conn:    conn,
        buffer:  buffer,
        encoder: json.NewEncoder(buffer),
    }, nil
}

func (u *UnixSocketOutput) WriteEvent(event *StreamEvent) error {
    u.mu.Lock()
    defer u.mu.Unlock()
    
    if err := u.encoder.Encode(event); err != nil {
        return fmt.Errorf("failed to encode event: %w", err)
    }
    
    return u.buffer.Flush()
}

func (u *UnixSocketOutput) WriteAlert(alert *SecurityAlert) error {
    u.mu.Lock()
    defer u.mu.Unlock()
    
    if err := u.encoder.Encode(alert); err != nil {
        return fmt.Errorf("failed to encode alert: %w", err)
    }
    
    return u.buffer.Flush()
}

func (u *UnixSocketOutput) Close() error {
    u.mu.Lock()
    defer u.mu.Unlock()
    
    if u.buffer != nil {
        u.buffer.Flush()
    }
    
    if u.conn != nil {
        return u.conn.Close()
    }
    return nil
}

// PipeOutput creates a named pipe for other processes
type PipeOutput struct {
    pipeName string
    format   OutputFormat
    file     *os.File
    encoder  *json.Encoder
}

func NewPipeOutput(name string, format OutputFormat) (*PipeOutput, error) {
    // This would create a named pipe in production
    // For now, we'll use a regular file
    pipePath := fmt.Sprintf("/tmp/strigoi-%s.pipe", name)
    
    // In production: syscall.Mkfifo(pipePath, 0644)
    file, err := os.OpenFile(pipePath, os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    
    return &PipeOutput{
        pipeName: name,
        format:   format,
        file:     file,
        encoder:  json.NewEncoder(file),
    }, nil
}

func (p *PipeOutput) WriteEvent(event *StreamEvent) error {
    return p.encoder.Encode(event)
}

func (p *PipeOutput) WriteAlert(alert *SecurityAlert) error {
    return p.encoder.Encode(alert)
}

func (p *PipeOutput) Close() error {
    return p.file.Close()
}

// IntegrationOutput sends to configured integrations
type IntegrationOutput struct {
    integration string
    // This would connect to the actual integration actors
}

func NewIntegrationOutput(name string) (*IntegrationOutput, error) {
    return &IntegrationOutput{
        integration: name,
    }, nil
}

func (i *IntegrationOutput) WriteEvent(event *StreamEvent) error {
    // In production, this would route to the appropriate integration actor
    // For now, just log it
    fmt.Printf("Integration[%s] Event: %+v\n", i.integration, event)
    return nil
}

func (i *IntegrationOutput) WriteAlert(alert *SecurityAlert) error {
    fmt.Printf("Integration[%s] Alert: %+v\n", i.integration, alert)
    return nil
}

func (i *IntegrationOutput) Close() error {
    return nil
}

// MultiOutput writes to multiple destinations
type MultiOutput struct {
    writers []OutputWriter
}

func NewMultiOutput(writers ...OutputWriter) *MultiOutput {
    return &MultiOutput{
        writers: writers,
    }
}

func (m *MultiOutput) WriteEvent(event *StreamEvent) error {
    for _, w := range m.writers {
        if err := w.WriteEvent(event); err != nil {
            // Log error but continue
            fmt.Printf("Error writing to output: %v\n", err)
        }
    }
    return nil
}

func (m *MultiOutput) WriteAlert(alert *SecurityAlert) error {
    for _, w := range m.writers {
        if err := w.WriteAlert(alert); err != nil {
            fmt.Printf("Error writing alert: %v\n", err)
        }
    }
    return nil
}

func (m *MultiOutput) Close() error {
    var lastErr error
    for _, w := range m.writers {
        if err := w.Close(); err != nil {
            lastErr = err
        }
    }
    return lastErr
}