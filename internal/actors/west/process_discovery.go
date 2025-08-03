package west

import (
    "bufio"
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
)

// Process represents a discovered process
type Process struct {
    PID     int
    PPID    int
    Name    string
    Cmdline string
    Exe     string
    IsClaudeRelated bool
    IsMCPServer     bool
}

// DiscoverProcesses finds Claude and MCP-related processes on Linux
func DiscoverProcesses(patterns []string) ([]Process, error) {
    var processes []Process
    
    // Default patterns if none provided
    if len(patterns) == 0 {
        patterns = []string{
            "claude",
            "mcp-server",
            "mcp_server",
            "node.*mcp",
            "python.*mcp",
            "deno.*mcp",
        }
    }
    
    // Read /proc to find processes
    procDir, err := os.Open("/proc")
    if err != nil {
        return nil, fmt.Errorf("failed to open /proc: %w", err)
    }
    defer procDir.Close()
    
    entries, err := procDir.Readdir(0)
    if err != nil {
        return nil, fmt.Errorf("failed to read /proc: %w", err)
    }
    
    for _, entry := range entries {
        // Skip non-numeric directories
        pid, err := strconv.Atoi(entry.Name())
        if err != nil {
            continue
        }
        
        process := Process{PID: pid}
        
        // Read cmdline
        cmdlinePath := filepath.Join("/proc", entry.Name(), "cmdline")
        cmdlineBytes, err := os.ReadFile(cmdlinePath)
        if err != nil {
            continue // Process might have exited
        }
        
        // cmdline uses null bytes as separators
        cmdline := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
        cmdline = strings.TrimSpace(cmdline)
        if cmdline == "" {
            continue
        }
        
        process.Cmdline = cmdline
        
        // Read comm (process name)
        commPath := filepath.Join("/proc", entry.Name(), "comm")
        commBytes, err := os.ReadFile(commPath)
        if err == nil {
            process.Name = strings.TrimSpace(string(commBytes))
        }
        
        // Read exe (symlink to executable)
        exePath := filepath.Join("/proc", entry.Name(), "exe")
        if exe, err := os.Readlink(exePath); err == nil {
            process.Exe = exe
        }
        
        // Read status for PPID
        statusPath := filepath.Join("/proc", entry.Name(), "status")
        if statusFile, err := os.Open(statusPath); err == nil {
            scanner := bufio.NewScanner(statusFile)
            for scanner.Scan() {
                line := scanner.Text()
                if strings.HasPrefix(line, "PPid:") {
                    fields := strings.Fields(line)
                    if len(fields) >= 2 {
                        process.PPID, _ = strconv.Atoi(fields[1])
                    }
                    break
                }
            }
            statusFile.Close()
        }
        
        // Check if process matches our patterns
        lowerCmdline := strings.ToLower(cmdline)
        lowerName := strings.ToLower(process.Name)
        
        // Check each pattern
        matched := false
        for _, pattern := range patterns {
            lowerPattern := strings.ToLower(pattern)
            
            // Try glob matching first
            if m, _ := filepath.Match(lowerPattern, lowerCmdline); m {
                matched = true
            } else if m, _ := filepath.Match(lowerPattern, lowerName); m {
                matched = true
            } else if strings.Contains(lowerCmdline, strings.Trim(lowerPattern, "*")) {
                matched = true
            }
            
            if matched {
                break
            }
        }
        
        // Also check for MCP servers by looking at directory structure
        if !matched && strings.Contains(lowerCmdline, ".claude-mcp-servers") {
            matched = true
            process.IsMCPServer = true
        }
        
        // Also check for claude processes
        if !matched && strings.Contains(lowerCmdline, "claude") {
            matched = true
            process.IsClaudeRelated = true
        }
        
        if matched {
            // Categorize the process
            if strings.Contains(lowerCmdline, "claude") || strings.Contains(lowerName, "claude") {
                process.IsClaudeRelated = true
            }
            if strings.Contains(lowerCmdline, "mcp") || strings.Contains(lowerCmdline, "mcp-server") || 
               strings.Contains(lowerCmdline, ".claude-mcp-servers") {
                process.IsMCPServer = true
            }
            processes = append(processes, process)
        }
    }
    
    return processes, nil
}

// GetProcessChildren finds all child processes of a given PID
func GetProcessChildren(parentPID int) ([]int, error) {
    var children []int
    
    procDir, err := os.Open("/proc")
    if err != nil {
        return nil, err
    }
    defer procDir.Close()
    
    entries, err := procDir.Readdir(0)
    if err != nil {
        return nil, err
    }
    
    for _, entry := range entries {
        pid, err := strconv.Atoi(entry.Name())
        if err != nil {
            continue
        }
        
        // Read status for PPID
        statusPath := filepath.Join("/proc", entry.Name(), "status")
        statusFile, err := os.Open(statusPath)
        if err != nil {
            continue
        }
        
        scanner := bufio.NewScanner(statusFile)
        for scanner.Scan() {
            line := scanner.Text()
            if strings.HasPrefix(line, "PPid:") {
                fields := strings.Fields(line)
                if len(fields) >= 2 {
                    ppid, _ := strconv.Atoi(fields[1])
                    if ppid == parentPID {
                        children = append(children, pid)
                    }
                }
                break
            }
        }
        statusFile.Close()
    }
    
    return children, nil
}