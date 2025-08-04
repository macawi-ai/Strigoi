package probe

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// TerminalDisplay provides a real-time UI for stream monitoring.
type TerminalDisplay struct {
	vulns        []StreamVulnerability
	activities   []ActivityEvent
	stats        DisplayStats
	running      bool
	mu           sync.RWMutex
	lastUpdate   time.Time
	ShowActivity bool
}

// DisplayStats tracks display statistics.
type DisplayStats struct {
	ProcessCount  int
	BytesCaptured int64
	EventsCount   int64
	VulnsCount    int64
	StartTime     time.Time
	LastActivity  time.Time
}

// ActivityEvent represents stream activity for display.
type ActivityEvent struct {
	Timestamp time.Time
	Process   StreamTarget
	Stream    string
	Preview   string
	Bytes     int
}

// NewTerminalDisplay creates a new terminal display.
func NewTerminalDisplay() *TerminalDisplay {
	return &TerminalDisplay{
		vulns:      make([]StreamVulnerability, 0),
		activities: make([]ActivityEvent, 0),
		stats:      DisplayStats{StartTime: time.Now()},
		lastUpdate: time.Now(),
	}
}

// Start initializes the display.
func (d *TerminalDisplay) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("display already running")
	}

	d.running = true
	d.stats.StartTime = time.Now()

	// Clear screen and show initial display
	d.render()

	// Start refresh loop
	go d.refreshLoop()

	return nil
}

// Stop terminates the display.
func (d *TerminalDisplay) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.running = false
	return nil
}

// Update refreshes display data.
func (d *TerminalDisplay) Update(streams map[int]*StreamCapture) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Update statistics
	d.stats.ProcessCount = len(streams)
	d.stats.BytesCaptured = 0
	d.stats.EventsCount = 0
	d.stats.VulnsCount = 0

	for _, capture := range streams {
		d.stats.BytesCaptured += capture.Statistics.BytesCaptured
		d.stats.EventsCount += capture.Statistics.EventsCount
		d.stats.VulnsCount += capture.Statistics.VulnsFound

		if capture.Statistics.LastActivity.After(d.stats.LastActivity) {
			d.stats.LastActivity = capture.Statistics.LastActivity
		}
	}

	d.lastUpdate = time.Now()
}

// AddVulnerability adds a new vulnerability to the display.
func (d *TerminalDisplay) AddVulnerability(vuln StreamVulnerability) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.vulns = append(d.vulns, vuln)

	// Keep only last 20 vulnerabilities for display
	if len(d.vulns) > 20 {
		d.vulns = d.vulns[len(d.vulns)-20:]
	}

	d.stats.VulnsCount++
	d.lastUpdate = time.Now()
}

// AddActivity adds a new activity event to the display.
func (d *TerminalDisplay) AddActivity(target StreamTarget, stream string, data []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Create preview - first 50 chars, sanitized
	preview := sanitizePreview(data, 50)

	activity := ActivityEvent{
		Timestamp: time.Now(),
		Process:   target,
		Stream:    stream,
		Preview:   preview,
		Bytes:     len(data),
	}

	d.activities = append(d.activities, activity)

	// Keep only last 10 activities for display
	if len(d.activities) > 10 {
		d.activities = d.activities[len(d.activities)-10:]
	}

	d.lastUpdate = time.Now()
}

// sanitizePreview creates a safe preview of data for display.
func sanitizePreview(data []byte, maxLen int) string {
	if len(data) == 0 {
		return "[empty]"
	}

	// Take first maxLen bytes
	preview := data
	if len(preview) > maxLen {
		preview = preview[:maxLen]
	}

	// Convert to string and replace non-printable characters
	result := make([]byte, 0, len(preview))
	for _, b := range preview {
		if b >= 32 && b <= 126 { // Printable ASCII
			result = append(result, b)
		} else if b == '\n' {
			result = append(result, []byte("\\n")...)
		} else if b == '\r' {
			result = append(result, []byte("\\r")...)
		} else if b == '\t' {
			result = append(result, []byte("\\t")...)
		} else {
			result = append(result, '.')
		}
	}

	s := string(result)
	if len(data) > maxLen {
		s += "..."
	}

	return s
}

// refreshLoop continuously updates the display.
func (d *TerminalDisplay) refreshLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for d.running {
		<-ticker.C
		d.render()
	}
}

// render draws the current display state.
func (d *TerminalDisplay) render() {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Clear screen (ANSI escape codes)
	fmt.Print("\033[2J\033[H")

	// Build display
	var sb strings.Builder

	// Header
	d.renderHeader(&sb)

	// Activity table (if enabled)
	if d.ShowActivity {
		d.renderActivities(&sb)
	}

	// Vulnerability table
	d.renderVulnerabilities(&sb)

	// Statistics
	d.renderStats(&sb)

	// Controls
	d.renderControls(&sb)

	// Output
	fmt.Print(sb.String())
}

// renderHeader draws the header section.
func (d *TerminalDisplay) renderHeader(sb *strings.Builder) {
	duration := time.Since(d.stats.StartTime)
	status := "MONITORING"
	if !d.running {
		status = "STOPPED"
	}

	sb.WriteString("\033[1;36m") // Cyan bold
	sb.WriteString("═══════════════ Strigoi Center - Stream Monitor ═══════════════\n")
	sb.WriteString("\033[0m") // Reset

	sb.WriteString(fmt.Sprintf("Processes: %d | Status: %s | Duration: %s\n",
		d.stats.ProcessCount,
		status,
		formatDuration(duration),
	))
	sb.WriteString("\n")
}

// renderActivities draws the activity table.
func (d *TerminalDisplay) renderActivities(sb *strings.Builder) {
	sb.WriteString("\033[1;36m▼ Stream Activity\033[0m\n")

	if len(d.activities) == 0 {
		sb.WriteString("\033[2m  No stream activity captured yet...\033[0m\n\n")
		return
	}

	// Table header
	sb.WriteString("╭────────────────┬────────┬───────────┬──────────────────────────╮\n")
	sb.WriteString("│ Time           │ Stream │ Bytes     │ Preview                  │\n")
	sb.WriteString("├────────────────┼────────┼───────────┼──────────────────────────┤\n")

	// Show activities
	for _, activity := range d.activities {
		d.renderActivityRow(sb, &activity)
	}

	sb.WriteString("╰────────────────┴────────┴───────────┴──────────────────────────╯\n")
	sb.WriteString("\n")
}

// renderActivityRow draws a single activity row.
func (d *TerminalDisplay) renderActivityRow(sb *strings.Builder, activity *ActivityEvent) {
	timeStr := activity.Timestamp.Format("15:04:05.000")

	// Format stream name
	streamStr := activity.Stream
	switch activity.Stream {
	case "stdin":
		streamStr = "\033[32mstdin \033[0m" // Green
	case "stdout":
		streamStr = "\033[34mstdout\033[0m" // Blue
	case "stderr":
		streamStr = "\033[33mstderr\033[0m" // Yellow
	}

	// Format bytes
	bytesStr := fmt.Sprintf("%d", activity.Bytes)
	if activity.Bytes > 1024 {
		bytesStr = formatBytes(int64(activity.Bytes))
	}

	// Truncate preview if needed
	preview := activity.Preview
	if len(preview) > 25 {
		preview = preview[:22] + "..."
	}

	sb.WriteString(fmt.Sprintf("│ %s │ %s │ %-9s │ %-24s │\n",
		timeStr,
		streamStr,
		bytesStr,
		preview,
	))
}

// renderVulnerabilities draws the vulnerability table.
func (d *TerminalDisplay) renderVulnerabilities(sb *strings.Builder) {
	sb.WriteString("\033[1;33m▼ Live Vulnerabilities Detected\033[0m\n")

	if len(d.vulns) == 0 {
		sb.WriteString("\033[2m  No vulnerabilities detected yet...\033[0m\n\n")
		return
	}

	// Table header
	sb.WriteString("╭────────────────┬──────────┬────────────┬─────────────────────╮\n")
	sb.WriteString("│ Time           │ Severity │ Type       │ Evidence            │\n")
	sb.WriteString("├────────────────┼──────────┼────────────┼─────────────────────┤\n")

	// Show last 10 vulnerabilities
	start := 0
	if len(d.vulns) > 10 {
		start = len(d.vulns) - 10
	}

	for i := start; i < len(d.vulns); i++ {
		vuln := d.vulns[i]
		d.renderVulnRow(sb, &vuln)
	}

	sb.WriteString("╰────────────────┴──────────┴────────────┴─────────────────────╯\n")
	sb.WriteString("\n")
}

// renderVulnRow draws a single vulnerability row.
func (d *TerminalDisplay) renderVulnRow(sb *strings.Builder, vuln *StreamVulnerability) {
	timeStr := vuln.Timestamp.Format("15:04:05.000")

	// Color code severity
	severityStr := vuln.Severity
	switch vuln.Severity {
	case "critical":
		severityStr = "\033[1;31mCRITICAL\033[0m" // Red bold
	case "high":
		severityStr = "\033[1;33mHIGH    \033[0m" // Yellow bold
	case "medium":
		severityStr = "\033[1;34mMEDIUM  \033[0m" // Blue bold
	case "low":
		severityStr = "\033[1;32mLOW     \033[0m" // Green bold
	}

	// Truncate evidence if too long
	evidence := vuln.Evidence
	if len(evidence) > 20 {
		evidence = evidence[:17] + "..."
	}

	// Truncate type if too long
	typeStr := vuln.Subtype
	if len(typeStr) > 10 {
		typeStr = typeStr[:10]
	}

	sb.WriteString(fmt.Sprintf("│ %s │ %s │ %-10s │ %-19s │\n",
		timeStr,
		severityStr,
		typeStr,
		evidence,
	))
}

// renderStats draws the statistics section.
func (d *TerminalDisplay) renderStats(sb *strings.Builder) {
	sb.WriteString("╭─────────────────────────────────────────────────────────────╮\n")
	sb.WriteString(fmt.Sprintf("│ Stats: %s captured | %d vulns | %d events │\n",
		formatBytes(d.stats.BytesCaptured),
		d.stats.VulnsCount,
		d.stats.EventsCount,
	))

	if !d.stats.LastActivity.IsZero() {
		idleTime := time.Since(d.stats.LastActivity)
		sb.WriteString(fmt.Sprintf("│ Last activity: %s ago                              │\n",
			formatDuration(idleTime),
		))
	}

	sb.WriteString("╰─────────────────────────────────────────────────────────────╯\n")
}

// renderControls draws the control hints.
func (d *TerminalDisplay) renderControls(sb *strings.Builder) {
	sb.WriteString("\n")
	sb.WriteString("\033[2m[Press 'q' to quit, 'p' to pause, 'c' to clear]\033[0m\n")
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// formatBytes formats byte count for display.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// SimpleDisplay provides a non-interactive display option.
type SimpleDisplay struct {
	mu sync.Mutex
}

// NewSimpleDisplay creates a basic display.
func NewSimpleDisplay() *SimpleDisplay {
	return &SimpleDisplay{}
}

// LogVulnerability prints a vulnerability to stdout.
func (d *SimpleDisplay) LogVulnerability(vuln StreamVulnerability) {
	d.mu.Lock()
	defer d.mu.Unlock()

	timestamp := vuln.Timestamp.Format("2006-01-02 15:04:05")

	// Color code based on severity
	var color string
	switch vuln.Severity {
	case "critical":
		color = "\033[31m" // Red
	case "high":
		color = "\033[33m" // Yellow
	case "medium":
		color = "\033[34m" // Blue
	default:
		color = "\033[32m" // Green
	}

	fmt.Printf("%s[%s] %sSEVERITY: %s\033[0m | TYPE: %s | EVIDENCE: %s | PID: %d\n",
		timestamp,
		vuln.ID,
		color,
		strings.ToUpper(vuln.Severity),
		vuln.Subtype,
		vuln.Evidence,
		vuln.ProcessInfo.PID,
	)
}
