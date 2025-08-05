package probe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/macawi-ai/strigoi/pkg/modules"
)

func init() {
	modules.RegisterBuiltin("probe/center", NewCenterModule)
}

// CenterModule analyzes STDIO streams for security vulnerabilities.
type CenterModule struct {
	modules.BaseModule

	// Capture components
	captureEngine *CaptureEngine
	activeStreams map[int]*StreamCapture

	// Analysis components
	dissectors   []Dissector
	vulnDetector *VulnerabilityDetector
	credHunter   *CredentialHunter

	// Output components
	logger  *EventLogger
	display *TerminalDisplay

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// Configuration
	config CenterConfig
}

// CenterConfig holds configuration for the center probe.
type CenterConfig struct {
	CaptureMode    string        `json:"capture_mode"`    // procfs, strace, auto
	PollInterval   time.Duration `json:"poll_interval"`   // How often to check streams
	BufferSize     int           `json:"buffer_size"`     // Per-stream buffer size
	LogFile        string        `json:"log_file"`        // JSONL output file
	DisplayEnabled bool          `json:"display_enabled"` // Show terminal UI
	Filters        []string      `json:"filters"`         // Regex filters
	MaxDuration    time.Duration `json:"max_duration"`    // Maximum monitoring time
	ShowActivity   bool          `json:"show_activity"`   // Show all stream activity
	EnableStrace   bool          `json:"enable_strace"`   // Enable strace fallback (opt-in)
}

// StreamTarget represents a process to monitor.
type StreamTarget struct {
	PID         int
	Name        string
	CommandLine string
	StartTime   time.Time
}

// StreamCapture represents active stream monitoring.
type StreamCapture struct {
	Target      StreamTarget
	Stdin       *StreamBuffer
	Stdout      *StreamBuffer
	Stderr      *StreamBuffer
	CaptureMode string
	Statistics  StreamStats
}

// StreamStats tracks capture statistics.
type StreamStats struct {
	BytesCaptured int64
	EventsCount   int64
	VulnsFound    int64
	LastActivity  time.Time
}

// StreamVulnerability represents a detected vulnerability.
type StreamVulnerability struct {
	ID          string       `json:"id"`
	Timestamp   time.Time    `json:"timestamp"`
	Severity    string       `json:"severity"`   // critical, high, medium, low
	Type        string       `json:"type"`       // credential, api_key, token, etc.
	Subtype     string       `json:"subtype"`    // password, oauth, jwt, etc.
	Evidence    string       `json:"evidence"`   // Redacted by default
	Location    string       `json:"location"`   // stdin, stdout, stderr
	Context     string       `json:"context"`    // Surrounding data
	Confidence  float64      `json:"confidence"` // 0.0 to 1.0
	ProcessInfo StreamTarget `json:"process_info"`
}

// NewCenterModule creates a new stream analysis module.
func NewCenterModule() modules.Module {
	return &CenterModule{
		BaseModule: modules.BaseModule{
			ModuleName:        "probe/center",
			ModuleDescription: "Stream analysis and vulnerability detection",
			ModuleType:        modules.ProbeModule,
			ModuleOptions: map[string]*modules.ModuleOption{
				"target": {
					Name:        "target",
					Description: "Process name or PID to monitor",
					Required:    true,
					Type:        "string",
				},
				"duration": {
					Name:        "duration",
					Description: "Maximum monitoring duration",
					Required:    false,
					Type:        "string",
					Default:     "0", // Unlimited
				},
				"output": {
					Name:        "output",
					Description: "Log file path (JSONL format)",
					Required:    false,
					Type:        "string",
					Default:     "stream-monitor.jsonl",
				},
				"no-display": {
					Name:        "no-display",
					Description: "Disable terminal UI",
					Required:    false,
					Type:        "bool",
					Default:     false,
				},
				"filter": {
					Name:        "filter",
					Description: "Regex filter for stream data",
					Required:    false,
					Type:        "string",
					Default:     "",
				},
				"buffer-size": {
					Name:        "buffer-size",
					Description: "Buffer size per stream (KB)",
					Required:    false,
					Type:        "int",
					Default:     64,
				},
				"poll-interval": {
					Name:        "poll-interval",
					Description: "Stream polling interval (ms)",
					Required:    false,
					Type:        "int",
					Default:     10,
				},
				"show-activity": {
					Name:        "show-activity",
					Description: "Show stream activity even without vulnerabilities",
					Required:    false,
					Type:        "bool",
					Default:     false,
				},
				"enable-strace": {
					Name:        "enable-strace",
					Description: "Enable strace fallback for PTY capture (performance impact)",
					Required:    false,
					Type:        "bool",
					Default:     false,
				},
			},
		},
		activeStreams: make(map[int]*StreamCapture),
	}
}

// Check verifies the module can run.
func (m *CenterModule) Check() bool {
	// Check if we can access /proc
	if _, err := os.Stat("/proc"); err != nil {
		return false
	}
	return true
}

// Configure sets up the module before execution.
func (m *CenterModule) Configure() error {
	// Parse duration
	duration := time.Duration(0)
	if d, ok := m.ModuleOptions["duration"]; ok && d.Value != nil {
		if dStr, ok := d.Value.(string); ok && dStr != "0" {
			parsed, err := time.ParseDuration(dStr)
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			duration = parsed
		}
	}

	// Parse other options
	bufferSize := 64
	if bs, ok := m.ModuleOptions["buffer-size"]; ok && bs.Value != nil {
		if bsInt, ok := bs.Value.(int); ok {
			bufferSize = bsInt
		}
	}

	pollInterval := 10
	if pi, ok := m.ModuleOptions["poll-interval"]; ok && pi.Value != nil {
		if piInt, ok := pi.Value.(int); ok {
			pollInterval = piInt
		}
	}

	logFile := "stream-monitor.jsonl"
	if lf, ok := m.ModuleOptions["output"]; ok && lf.Value != nil {
		if lfStr, ok := lf.Value.(string); ok {
			logFile = lfStr
		}
	}

	noDisplay := false
	if nd, ok := m.ModuleOptions["no-display"]; ok && nd.Value != nil {
		if ndBool, ok := nd.Value.(bool); ok {
			noDisplay = ndBool
		}
	}

	filters := []string{}
	if f, ok := m.ModuleOptions["filter"]; ok && f.Value != nil {
		if fStr, ok := f.Value.(string); ok && fStr != "" {
			filters = append(filters, fStr)
		}
	}

	showActivity := false
	if sa, ok := m.ModuleOptions["show-activity"]; ok && sa.Value != nil {
		if saBool, ok := sa.Value.(bool); ok {
			showActivity = saBool
		}
	}

	enableStrace := false
	if es, ok := m.ModuleOptions["enable-strace"]; ok && es.Value != nil {
		if esBool, ok := es.Value.(bool); ok {
			enableStrace = esBool
		}
	}

	m.config = CenterConfig{
		CaptureMode:    "auto",
		PollInterval:   time.Duration(pollInterval) * time.Millisecond,
		BufferSize:     bufferSize * 1024, // Convert to bytes
		LogFile:        logFile,
		DisplayEnabled: !noDisplay,
		Filters:        filters,
		MaxDuration:    duration,
		ShowActivity:   showActivity,
		EnableStrace:   enableStrace,
	}

	// Initialize components
	m.captureEngine = NewCaptureEngine(m.config.CaptureMode)
	if m.config.EnableStrace {
		if err := m.captureEngine.EnableStrace(); err != nil {
			fmt.Printf("Warning: Failed to enable strace: %v\n", err)
		}
	}
	m.vulnDetector = NewVulnerabilityDetector()
	m.credHunter = NewCredentialHunter()

	// Initialize dissectors
	m.dissectors = []Dissector{
		NewHTTPDissector(),
		NewGRPCDissectorV2(), // Use improved version with better performance and security
		NewWebSocketDissector(),
		NewJSONDissector(),
		NewSQLDissector(),
		NewPlainTextDissector(),
	}

	// Initialize logger
	var err error
	m.logger, err = NewEventLogger(m.config.LogFile)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	// Initialize display if enabled
	if m.config.DisplayEnabled {
		m.display = NewTerminalDisplay()
		m.display.ShowActivity = m.config.ShowActivity
	}

	return nil
}

// Run executes the stream monitoring.
func (m *CenterModule) Run() (*modules.ModuleResult, error) {
	// Configure module
	if err := m.Configure(); err != nil {
		return &modules.ModuleResult{
			Module: m.Name(),
			Status: "failed",
			Error:  err.Error(),
		}, err
	}

	// Validate options
	if err := m.ValidateOptions(); err != nil {
		return &modules.ModuleResult{
			Module: m.Name(),
			Status: "failed",
			Error:  err.Error(),
		}, err
	}

	// Get target
	target, ok := m.ModuleOptions["target"]
	if !ok || target.Value == nil {
		return &modules.ModuleResult{
			Module: m.Name(),
			Status: "failed",
			Error:  "target not set",
		}, fmt.Errorf("target not set")
	}
	targetStr := fmt.Sprintf("%v", target.Value)

	// Initialize context
	m.ctx, m.cancel = context.WithCancel(context.Background())
	if m.config.MaxDuration > 0 {
		m.ctx, m.cancel = context.WithTimeout(m.ctx, m.config.MaxDuration)
	}
	defer m.cancel()

	startTime := time.Now()

	// Find target processes
	targets, err := m.findTargets(targetStr)
	if err != nil {
		return &modules.ModuleResult{
			Module:    m.Name(),
			Status:    "failed",
			StartTime: startTime,
			Error:     err.Error(),
		}, err
	}

	if len(targets) == 0 {
		return &modules.ModuleResult{
			Module:    m.Name(),
			Status:    "failed",
			StartTime: startTime,
			Error:     "no matching processes found",
		}, fmt.Errorf("no processes matching target: %s", targetStr)
	}

	// Log start event
	if err := m.logger.LogEvent("start", map[string]interface{}{
		"targets": targets,
		"config":  m.config,
	}); err != nil {
		// Non-critical error, just log it
		fmt.Printf("Warning: failed to log start event: %v\n", err)
	}

	// Initialize display
	if m.display != nil {
		if err := m.display.Start(); err != nil {
			return &modules.ModuleResult{
				Module:    m.Name(),
				Status:    "failed",
				StartTime: startTime,
				Error:     fmt.Sprintf("display error: %v", err),
			}, err
		}
		defer func() {
			if err := m.display.Stop(); err != nil {
				fmt.Printf("Warning: failed to stop display: %v\n", err)
			}
		}()
	}

	// Start monitoring each target
	for _, target := range targets {
		m.wg.Add(1)
		go m.monitorTarget(target)
	}

	// Wait for completion or interruption
	m.wg.Wait()

	// Collect results
	results := m.collectResults()

	// Log stop event
	if err := m.logger.LogEvent("stop", map[string]interface{}{
		"reason":   "completed",
		"duration": time.Since(startTime),
		"stats":    results["statistics"],
	}); err != nil {
		// Non-critical error, just log it
		fmt.Printf("Warning: failed to log stop event: %v\n", err)
	}

	return &modules.ModuleResult{
		Module:    m.Name(),
		Status:    "completed",
		StartTime: startTime,
		EndTime:   time.Now(),
		Data:      results,
	}, nil
}

// findTargets locates processes matching the target specification.
func (m *CenterModule) findTargets(target string) ([]StreamTarget, error) {
	targets := []StreamTarget{}

	// Check if target is a PID
	if pid, err := strconv.Atoi(target); err == nil {
		// Direct PID
		if process, err := m.getProcessInfo(pid); err == nil {
			targets = append(targets, process)
		}
		return targets, nil
	}

	// Search by name
	procDir, err := os.Open("/proc")
	if err != nil {
		return nil, fmt.Errorf("cannot access /proc: %w", err)
	}
	defer procDir.Close()

	entries, err := procDir.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("cannot read /proc: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory name is a PID
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		// Read process info
		process, err := m.getProcessInfo(pid)
		if err != nil {
			continue
		}

		// Check if name matches
		if strings.Contains(process.Name, target) || strings.Contains(process.CommandLine, target) {
			targets = append(targets, process)
		}
	}

	return targets, nil
}

// getProcessInfo retrieves information about a process.
func (m *CenterModule) getProcessInfo(pid int) (StreamTarget, error) {
	target := StreamTarget{PID: pid}

	// Read command line
	cmdlineFile := fmt.Sprintf("/proc/%d/cmdline", pid)
	cmdlineData, err := os.ReadFile(cmdlineFile)
	if err != nil {
		return target, err
	}
	target.CommandLine = strings.ReplaceAll(string(cmdlineData), "\x00", " ")

	// Extract process name
	parts := strings.Split(target.CommandLine, " ")
	if len(parts) > 0 {
		target.Name = filepath.Base(parts[0])
	}

	// Get start time
	statFile := fmt.Sprintf("/proc/%d/stat", pid)
	statData, err := os.ReadFile(statFile)
	if err == nil {
		// Parse stat file for start time (field 22)
		fields := strings.Fields(string(statData))
		if len(fields) >= 22 {
			// Convert jiffies to time
			// Note: This is simplified, would need proper calculation
			target.StartTime = time.Now()
		}
	}

	return target, nil
}

// monitorTarget monitors a single process.
func (m *CenterModule) monitorTarget(target StreamTarget) {
	defer m.wg.Done()

	// Create stream capture
	capture := &StreamCapture{
		Target:      target,
		Stdin:       NewStreamBuffer(m.config.BufferSize),
		Stdout:      NewStreamBuffer(m.config.BufferSize),
		Stderr:      NewStreamBuffer(m.config.BufferSize),
		CaptureMode: m.config.CaptureMode,
	}

	m.mu.Lock()
	m.activeStreams[target.PID] = capture
	m.mu.Unlock()

	// Start capture
	if err := m.captureEngine.Attach(target.PID); err != nil {
		if err := m.logger.LogError(fmt.Errorf("failed to attach to PID %d: %w", target.PID, err)); err != nil {
			fmt.Printf("Warning: failed to log error: %v\n", err)
		}
		return
	}
	defer func() {
		if err := m.captureEngine.Detach(target.PID); err != nil {
			fmt.Printf("Warning: failed to detach from PID %d: %v\n", target.PID, err)
		}
	}()

	// Monitoring loop
	ticker := time.NewTicker(m.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// Read streams
			data, err := m.captureEngine.ReadStreams(target.PID)
			if err != nil {
				// Process might have exited
				if os.IsNotExist(err) {
					return
				}
				continue
			}

			// Process data
			if len(data.Stdin) > 0 {
				m.processStreamData(capture, "stdin", data.Stdin)
			}
			if len(data.Stdout) > 0 {
				m.processStreamData(capture, "stdout", data.Stdout)
			}
			if len(data.Stderr) > 0 {
				m.processStreamData(capture, "stderr", data.Stderr)
			}

			// Update display
			if m.display != nil {
				m.display.Update(m.activeStreams)
			}
		}
	}
}

// processStreamData analyzes stream data for vulnerabilities.
func (m *CenterModule) processStreamData(capture *StreamCapture, stream string, data []byte) {
	// Update statistics
	capture.Statistics.BytesCaptured += int64(len(data))
	capture.Statistics.EventsCount++
	capture.Statistics.LastActivity = time.Now()

	// Store in buffer
	switch stream {
	case "stdin":
		capture.Stdin.Write(data)
	case "stdout":
		capture.Stdout.Write(data)
	case "stderr":
		capture.Stderr.Write(data)
	}

	// Apply filters
	if len(m.config.Filters) > 0 {
		matched := false
		for _, filter := range m.config.Filters {
			if regexp.MustCompile(filter).Match(data) {
				matched = true
				break
			}
		}
		if !matched {
			return
		}
	}

	// Show activity if enabled (after filter check)
	if m.config.ShowActivity {
		// Add to display
		if m.display != nil {
			m.display.AddActivity(capture.Target, stream, data)
		}

		// Log to JSONL
		preview := sanitizePreview(data, 50)
		if err := m.logger.LogActivity(capture.Target, stream, preview, len(data)); err != nil {
			fmt.Printf("Warning: failed to log activity: %v\n", err)
		}
	}

	// Detect protocol
	var dissector Dissector
	var confidence float64
	for _, d := range m.dissectors {
		if match, conf := d.Identify(data); match && conf > confidence {
			dissector = d
			confidence = conf
		}
	}

	// Extract vulnerabilities
	vulns := []StreamVulnerability{}

	// Use dissector if found
	if dissector != nil {
		frame, err := dissector.Dissect(data)
		if err == nil {
			vulns = append(vulns, dissector.FindVulnerabilities(frame)...)
		}
	}

	// Always run credential hunter
	creds := m.credHunter.Hunt(data)
	for _, cred := range creds {
		vuln := StreamVulnerability{
			ID:          fmt.Sprintf("VULN-%d", time.Now().UnixNano()),
			Timestamp:   time.Now(),
			Severity:    cred.Severity,
			Type:        "credential",
			Subtype:     cred.Type,
			Evidence:    cred.Redacted,
			Location:    stream,
			Context:     string(data),
			Confidence:  cred.Confidence,
			ProcessInfo: capture.Target,
		}
		vulns = append(vulns, vuln)
	}

	// Process vulnerabilities
	for _, vuln := range vulns {
		capture.Statistics.VulnsFound++

		// Log vulnerability
		if err := m.logger.LogVulnerability(vuln); err != nil {
			fmt.Printf("Warning: failed to log vulnerability: %v\n", err)
		}

		// Update display
		if m.display != nil {
			m.display.AddVulnerability(vuln)
		}
	}
}

// collectResults aggregates monitoring results.
func (m *CenterModule) collectResults() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalBytes := int64(0)
	totalEvents := int64(0)
	totalVulns := int64(0)
	processes := []map[string]interface{}{}

	for pid, capture := range m.activeStreams {
		totalBytes += capture.Statistics.BytesCaptured
		totalEvents += capture.Statistics.EventsCount
		totalVulns += capture.Statistics.VulnsFound

		processes = append(processes, map[string]interface{}{
			"pid":            pid,
			"name":           capture.Target.Name,
			"command_line":   capture.Target.CommandLine,
			"bytes_captured": capture.Statistics.BytesCaptured,
			"vulns_found":    capture.Statistics.VulnsFound,
		})
	}

	return map[string]interface{}{
		"processes": processes,
		"statistics": map[string]interface{}{
			"total_bytes":     totalBytes,
			"total_events":    totalEvents,
			"total_vulns":     totalVulns,
			"processes_count": len(m.activeStreams),
		},
		"log_file": m.config.LogFile,
	}
}

// Info returns module information.
func (m *CenterModule) Info() *modules.ModuleInfo {
	return &modules.ModuleInfo{
		Name:        m.ModuleName,
		Description: m.ModuleDescription,
		Author:      "Strigoi Security",
		Version:     "1.0.0",
		Tags:        []string{"stream", "stdio", "monitoring", "vulnerability"},
		References: []string{
			"STDIO Attack Chain Documentation",
			"Stream Analysis Best Practices",
		},
	}
}
