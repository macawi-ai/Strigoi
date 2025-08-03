package west

// ProcessInfo contains information about a discovered process
type ProcessInfo struct {
	PID      int
	PPID     int
	Name     string
	Command  string
	Category string // "Claude", "MCP", etc.
}