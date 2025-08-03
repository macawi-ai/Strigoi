package integrations

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"
    
    "github.com/macawi-ai/strigoi/internal/actors"
    "github.com/macawi-ai/strigoi/internal/stream"
)

// FileLoggerActor writes events to a specified local folder
type FileLoggerActor struct {
    *actors.BaseActor
    
    // Configuration
    logDir        string
    fileFormat    string // json, jsonl, csv, text
    rotateSize    int64  // bytes
    maxFiles      int
    
    // Current file
    currentFile   *os.File
    currentSize   int64
    fileIndex     int
    
    // State
    mu            sync.Mutex
    active        bool
    eventCount    int64
}

// NewFileLoggerActor creates a new file logger integration actor
func NewFileLoggerActor() *FileLoggerActor {
    actor := &FileLoggerActor{
        BaseActor: actors.NewBaseActor(
            "file_logger_integration",
            "Log events to local filesystem with rotation",
            "integration",
        ),
        logDir:     "/var/log/strigoi",
        fileFormat: "jsonl",
        rotateSize: 100 * 1024 * 1024, // 100MB
        maxFiles:   10,
    }
    
    // Define capabilities
    actor.AddCapability(actors.Capability{
        Name:        "file_write",
        Description: "Write events to local files",
        DataTypes:   []string{"event", "alert", "log"},
    })
    
    actor.AddCapability(actors.Capability{
        Name:        "log_rotation",
        Description: "Automatic log file rotation",
        DataTypes:   []string{"rotation"},
    })
    
    actor.AddCapability(actors.Capability{
        Name:        "format_flexibility",
        Description: "Multiple output formats (JSON, CSV, text)",
        DataTypes:   []string{"json", "csv", "text"},
    })
    
    actor.SetInputTypes([]string{"stream_event", "security_alert", "log_message"})
    actor.SetOutputType("file")
    
    return actor
}

// Probe checks file system access
func (f *FileLoggerActor) Probe(ctx context.Context, target actors.Target) (*actors.ProbeResult, error) {
    discoveries := []actors.Discovery{}
    
    // Override log directory if specified in target
    if dir, ok := target.Metadata["log_dir"].(string); ok && dir != "" {
        f.logDir = dir
    }
    
    // Check if directory exists
    info, err := os.Stat(f.logDir)
    if err != nil {
        if os.IsNotExist(err) {
            // Try to create it
            if mkErr := os.MkdirAll(f.logDir, 0755); mkErr != nil {
                discoveries = append(discoveries, actors.Discovery{
                    Type:       "log_directory",
                    Identifier: f.logDir,
                    Properties: map[string]interface{}{
                        "exists":    false,
                        "creatable": false,
                        "error":     mkErr.Error(),
                    },
                    Confidence: 1.0,
                })
            } else {
                discoveries = append(discoveries, actors.Discovery{
                    Type:       "log_directory",
                    Identifier: f.logDir,
                    Properties: map[string]interface{}{
                        "exists":    false,
                        "created":   true,
                        "writable":  true,
                    },
                    Confidence: 1.0,
                })
            }
        }
    } else {
        // Directory exists
        discoveries = append(discoveries, actors.Discovery{
            Type:       "log_directory",
            Identifier: f.logDir,
            Properties: map[string]interface{}{
                "exists":    true,
                "writable":  info.Mode().Perm()&0200 != 0,
                "size":      f.getDirSize(f.logDir),
                "free_space": f.getFreeSpace(f.logDir),
            },
            Confidence: 1.0,
        })
    }
    
    // Check existing log files
    files, _ := filepath.Glob(filepath.Join(f.logDir, "strigoi_*.log*"))
    discoveries = append(discoveries, actors.Discovery{
        Type:       "existing_logs",
        Identifier: "log_files",
        Properties: map[string]interface{}{
            "count":       len(files),
            "total_size":  f.getTotalFileSize(files),
            "oldest_file": f.getOldestFile(files),
            "newest_file": f.getNewestFile(files),
        },
        Confidence: 0.9,
    })
    
    return &actors.ProbeResult{
        ActorName:   f.Name(),
        Target:      target,
        Discoveries: discoveries,
        RawData: map[string]interface{}{
            "log_dir":     f.logDir,
            "format":      f.fileFormat,
            "rotate_size": f.rotateSize,
            "max_files":   f.maxFiles,
        },
    }, nil
}

// Sense starts file logging
func (f *FileLoggerActor) Sense(ctx context.Context, data *actors.ProbeResult) (*actors.SenseResult, error) {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    if f.active {
        return nil, fmt.Errorf("file logger already active")
    }
    
    // Ensure directory exists
    if err := os.MkdirAll(f.logDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create log directory: %w", err)
    }
    
    // Open initial log file
    if err := f.openNewFile(); err != nil {
        return nil, fmt.Errorf("failed to open log file: %w", err)
    }
    
    f.active = true
    
    observations := []actors.Observation{
        {
            Layer:       "integration",
            Description: fmt.Sprintf("File logging started in %s", f.logDir),
            Evidence: map[string]interface{}{
                "file":   f.currentFile.Name(),
                "format": f.fileFormat,
            },
            Severity: "info",
        },
    }
    
    return &actors.SenseResult{
        ActorName:    f.Name(),
        Observations: observations,
        Patterns:     []actors.Pattern{},
        Risks:        []actors.Risk{},
    }, nil
}

// Transform processes events and writes to file
func (f *FileLoggerActor) Transform(ctx context.Context, input interface{}) (interface{}, error) {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    if !f.active || f.currentFile == nil {
        return nil, fmt.Errorf("file logger not active")
    }
    
    // Check rotation
    if f.currentSize >= f.rotateSize {
        if err := f.rotateLog(); err != nil {
            return nil, fmt.Errorf("log rotation failed: %w", err)
        }
    }
    
    var written int
    var err error
    
    switch v := input.(type) {
    case *stream.StreamEvent:
        written, err = f.writeEvent(v)
        
    case *stream.SecurityAlert:
        written, err = f.writeAlert(v)
        
    case string:
        written, err = f.writeString(v)
        
    default:
        return nil, fmt.Errorf("unsupported input type: %T", input)
    }
    
    if err != nil {
        return nil, err
    }
    
    f.currentSize += int64(written)
    f.eventCount++
    
    return map[string]interface{}{
        "written":     written,
        "total_count": f.eventCount,
        "file_size":   f.currentSize,
    }, nil
}

// Write stream event
func (f *FileLoggerActor) writeEvent(event *stream.StreamEvent) (int, error) {
    switch f.fileFormat {
    case "json", "jsonl":
        data, err := json.Marshal(event)
        if err != nil {
            return 0, err
        }
        return f.currentFile.Write(append(data, '\n'))
        
    case "csv":
        line := fmt.Sprintf("%s,%s,%s,%d,%s,%d,%s\n",
            event.Timestamp.Format(time.RFC3339),
            event.Type,
            event.Direction,
            event.PID,
            event.ProcessName,
            event.Size,
            event.Summary,
        )
        return f.currentFile.WriteString(line)
        
    default: // text
        line := fmt.Sprintf("[%s] %s %s (PID:%d) %s (%d bytes)\n",
            event.Timestamp.Format("15:04:05"),
            event.Type,
            event.Direction,
            event.PID,
            event.Summary,
            event.Size,
        )
        return f.currentFile.WriteString(line)
    }
}

// Write security alert
func (f *FileLoggerActor) writeAlert(alert *stream.SecurityAlert) (int, error) {
    switch f.fileFormat {
    case "json", "jsonl":
        data, err := json.Marshal(alert)
        if err != nil {
            return 0, err
        }
        return f.currentFile.Write(append(data, '\n'))
        
    case "csv":
        line := fmt.Sprintf("%s,ALERT,%s,%s,%d,%s,%t\n",
            alert.Timestamp.Format(time.RFC3339),
            alert.Severity,
            alert.Category,
            alert.PID,
            alert.Title,
            alert.Blocked,
        )
        return f.currentFile.WriteString(line)
        
    default: // text
        line := fmt.Sprintf("[%s] ALERT [%s] %s - %s (PID:%d) Blocked:%t\n",
            alert.Timestamp.Format("15:04:05"),
            alert.Severity,
            alert.Category,
            alert.Title,
            alert.PID,
            alert.Blocked,
        )
        return f.currentFile.WriteString(line)
    }
}

// Write generic string
func (f *FileLoggerActor) writeString(s string) (int, error) {
    timestamp := time.Now()
    
    switch f.fileFormat {
    case "json", "jsonl":
        entry := map[string]interface{}{
            "timestamp": timestamp,
            "message":   s,
        }
        data, err := json.Marshal(entry)
        if err != nil {
            return 0, err
        }
        return f.currentFile.Write(append(data, '\n'))
        
    default:
        line := fmt.Sprintf("[%s] %s\n", timestamp.Format("15:04:05"), s)
        return f.currentFile.WriteString(line)
    }
}

// Open new log file
func (f *FileLoggerActor) openNewFile() error {
    timestamp := time.Now().Format("20060102_150405")
    ext := f.getFileExtension()
    
    filename := filepath.Join(f.logDir, fmt.Sprintf("strigoi_%s_%d%s", 
        timestamp, f.fileIndex, ext))
    
    file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return err
    }
    
    // Write header for CSV
    if f.fileFormat == "csv" && f.currentSize == 0 {
        file.WriteString("timestamp,type,direction,pid,process,size,summary\n")
    }
    
    f.currentFile = file
    f.currentSize = 0
    
    return nil
}

// Rotate log file
func (f *FileLoggerActor) rotateLog() error {
    // Close current file
    if f.currentFile != nil {
        f.currentFile.Close()
    }
    
    f.fileIndex++
    
    // Clean up old files if needed
    if err := f.cleanupOldFiles(); err != nil {
        // Non-fatal - log and continue
        fmt.Printf("Failed to cleanup old files: %v\n", err)
    }
    
    // Open new file
    return f.openNewFile()
}

// Get file extension based on format
func (f *FileLoggerActor) getFileExtension() string {
    switch f.fileFormat {
    case "json":
        return ".json"
    case "jsonl":
        return ".jsonl"
    case "csv":
        return ".csv"
    default:
        return ".log"
    }
}

// Clean up old log files
func (f *FileLoggerActor) cleanupOldFiles() error {
    pattern := filepath.Join(f.logDir, "strigoi_*")
    files, err := filepath.Glob(pattern)
    if err != nil {
        return err
    }
    
    if len(files) <= f.maxFiles {
        return nil
    }
    
    // Sort by modification time and remove oldest
    // Simplified - in production, implement proper sorting
    toRemove := len(files) - f.maxFiles
    for i := 0; i < toRemove && i < len(files); i++ {
        os.Remove(files[i])
    }
    
    return nil
}

// Helper functions

func (f *FileLoggerActor) getDirSize(path string) int64 {
    var size int64
    filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
        if err == nil && !info.IsDir() {
            size += info.Size()
        }
        return nil
    })
    return size
}

func (f *FileLoggerActor) getFreeSpace(path string) int64 {
    // Simplified - in production use syscall.Statfs
    return 1024 * 1024 * 1024 // 1GB placeholder
}

func (f *FileLoggerActor) getTotalFileSize(files []string) int64 {
    var total int64
    for _, file := range files {
        if info, err := os.Stat(file); err == nil {
            total += info.Size()
        }
    }
    return total
}

func (f *FileLoggerActor) getOldestFile(files []string) string {
    if len(files) == 0 {
        return ""
    }
    // Simplified - return first
    return filepath.Base(files[0])
}

func (f *FileLoggerActor) getNewestFile(files []string) string {
    if len(files) == 0 {
        return ""
    }
    // Simplified - return last
    return filepath.Base(files[len(files)-1])
}

// Stop the file logger
func (f *FileLoggerActor) Stop() error {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    if !f.active || f.currentFile == nil {
        return nil
    }
    
    // Write final entry
    f.writeString(fmt.Sprintf("File logger stopped. Total events: %d", f.eventCount))
    
    // Close file
    err := f.currentFile.Close()
    f.currentFile = nil
    f.active = false
    
    return err
}

// Configure updates actor configuration
func (f *FileLoggerActor) Configure(config map[string]interface{}) error {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    if dir, ok := config["log_dir"].(string); ok {
        f.logDir = dir
    }
    
    if format, ok := config["format"].(string); ok {
        f.fileFormat = format
    }
    
    if size, ok := config["rotate_size"].(int64); ok {
        f.rotateSize = size
    }
    
    if max, ok := config["max_files"].(int); ok {
        f.maxFiles = max
    }
    
    return nil
}