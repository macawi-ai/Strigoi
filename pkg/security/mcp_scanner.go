package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// MCPScanner discovers and analyzes MCP (Model Context Protocol) tools.
type MCPScanner struct {
	executor    *SecureExecutor
	logger      *logrus.Logger
	rules       *SecurityRuleEngine
	findings    []MCPTool
	risks       []SecurityFinding
	includeSelf bool
	mu          sync.Mutex
}

// MCPTool represents a discovered MCP server/client/bridge.
type MCPTool struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"` // "server", "client", "bridge"
	Version       string                 `json:"version,omitempty"`
	ConfigPath    string                 `json:"config_path,omitempty"`
	ProcessID     int                    `json:"process_id,omitempty"`
	Port          int                    `json:"port,omitempty"`
	Status        string                 `json:"status"` // "running", "stopped", "configured"
	Dependencies  []MCPDependency        `json:"dependencies,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	SecurityRisks []SecurityFinding      `json:"security_risks,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	// Enhanced fields for Phase 1
	ExecutablePath  string              `json:"executable_path,omitempty"`
	StartTime       time.Time           `json:"start_time,omitempty"`
	User            string              `json:"user,omitempty"`
	CommandLine     string              `json:"command_line,omitempty"`
	BuildInfo       map[string]string   `json:"build_info,omitempty"`
	NetworkExposure NetworkExposureInfo `json:"network_exposure,omitempty"`
}

// MCPDependency represents a dependency of an MCP tool.
type MCPDependency struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Type    string `json:"type"` // "database", "library", "service"
	Path    string `json:"path,omitempty"`
}

// ProcessInfo contains information about a running process.
type ProcessInfo struct {
	PID         int
	Command     string
	Args        []string
	User        string
	ProcessName string
	// Enhanced fields for Phase 1
	StartTime   time.Time         `json:"start_time,omitempty"`
	ParentPID   int               `json:"parent_pid,omitempty"`
	ExePath     string            `json:"exe_path,omitempty"`
	WorkingDir  string            `json:"working_dir,omitempty"`
	CmdLine     string            `json:"cmdline,omitempty"`
	Environment map[string]string `json:"-"` // Not exported for security
}

// NetworkConnection represents a network connection.
type NetworkConnection struct {
	Protocol string
	Port     int
	Address  string
	State    string
	PID      int
}

// NetworkExposureInfo contains network security assessment data.
type NetworkExposureInfo struct {
	ListeningPorts []int    `json:"listening_ports,omitempty"`
	BindAddress    string   `json:"bind_address,omitempty"`
	ExposureLevel  string   `json:"exposure_level,omitempty"` // "local", "network", "internet"
	TLSEnabled     bool     `json:"tls_enabled,omitempty"`
	RiskFactors    []string `json:"risk_factors,omitempty"`
}

// NewMCPScanner creates a new MCP scanner instance.
func NewMCPScanner(executor *SecureExecutor) *MCPScanner {
	// Use the global logger instance to respect the configured log level
	logger := logrus.StandardLogger()

	return &MCPScanner{
		executor:    executor,
		logger:      logger,
		rules:       NewSecurityRuleEngine(),
		findings:    []MCPTool{},
		risks:       []SecurityFinding{},
		includeSelf: false, // Default: exclude self-scanning
	}
}

// SetIncludeSelf configures whether to include Strigoi's own files and processes in scans.
func (ms *MCPScanner) SetIncludeSelf(include bool) {
	ms.includeSelf = include
}

// DiscoverMCPTools performs comprehensive MCP tool discovery and analysis.
func (ms *MCPScanner) DiscoverMCPTools(ctx context.Context) ([]MCPTool, error) {
	ms.logger.Info("Starting MCP tool discovery")

	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// Phase 1: Configuration Discovery (Parallel)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ms.scanConfigFiles(ctx); err != nil {
			ms.logger.WithError(err).Warn("Config file scanning encountered errors")
			errChan <- fmt.Errorf("config scanning: %w", err)
		}
	}()

	// Phase 2: Process Detection (Parallel)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ms.scanRunningProcesses(ctx); err != nil {
			ms.logger.WithError(err).Warn("Process scanning encountered errors")
			errChan <- fmt.Errorf("process scanning: %w", err)
		}
	}()

	// Phase 3: Network Analysis (Parallel)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ms.scanNetworkConnections(ctx); err != nil {
			ms.logger.WithError(err).Warn("Network scanning encountered errors")
			errChan <- fmt.Errorf("network scanning: %w", err)
		}
	}()

	// Wait for all scans to complete
	wg.Wait()
	close(errChan)

	// Collect any errors (non-fatal)
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		ms.logger.Warnf("Some scanning operations had errors: %v", errors)
	}

	// Phase 4: Correlation and Analysis
	ms.correlateFindings()
	ms.analyzeSecurityRisks()

	ms.logger.Infof("MCP discovery completed. Found %d tools with %d security findings",
		len(ms.findings), len(ms.risks))

	return ms.getFindings(), nil
}

// scanConfigFiles discovers MCP configuration files.
func (ms *MCPScanner) scanConfigFiles(ctx context.Context) error {
	ms.logger.Debug("Scanning MCP configuration files")

	// Known MCP configuration locations
	configPaths := []string{
		// Claude Code MCP configurations
		filepath.Join(os.Getenv("HOME"), ".config", "claude-code", "mcp_servers.json"),

		// System-wide configurations
		"/etc/mcp/",
		"/opt/mcp/",

		// Local project configurations
		"./mcp.json",
		"./mcp.yaml",
		"./docker-compose.yml",
		"./.env",
	}

	for _, configPath := range configPaths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			ms.scanConfigPath(configPath)
		}
	}

	return nil
}

// scanConfigPath scans a specific configuration path.
func (ms *MCPScanner) scanConfigPath(configPath string) {
	// Check if we should exclude this path
	if ms.shouldExcludePath(configPath) {
		return
	}

	// Validate path
	if err := ms.executor.ValidatePath(configPath); err != nil {
		ms.logger.WithField("path", configPath).Debug("Skipping invalid path")
		return
	}

	// Check if path exists
	info, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		return
	}

	if info.IsDir() {
		ms.scanDirectory(configPath)
	} else {
		ms.scanFile(configPath)
	}
}

// scanDirectory scans a directory for MCP configuration files.
func (ms *MCPScanner) scanDirectory(dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		ms.logger.WithError(err).WithField("dir", dirPath).Debug("Failed to read directory")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		if ms.isConfigFile(fileName) {
			fullPath := filepath.Join(dirPath, fileName)
			ms.scanFile(fullPath)
		}
	}
}

// scanFile scans a specific configuration file.
func (ms *MCPScanner) scanFile(filePath string) {
	filePath = filepath.Clean(filePath) // Standardize the file path
	ms.logger.WithField("file", filePath).Debug("Scanning config file")

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		ms.logger.WithError(err).WithField("file", filePath).Debug("Failed to read file")
		return
	}

	// Check file size limit (10MB)
	if len(content) > 10*1024*1024 {
		ms.logger.WithField("file", filePath).Warn("File too large, skipping")
		return
	}

	// Parse and analyze configuration
	tool := ms.parseConfigFile(filePath, content)
	if tool != nil {
		ms.logger.WithField("file", filePath).WithField("tool", tool.Name).Debug("Adding MCP tool finding")
		ms.addFinding(*tool)

		// Scan for security issues
		ms.logger.WithField("file", filePath).Debug("Scanning for security risks...")
		findings := ms.rules.ScanContent(string(content), filePath)
		ms.logger.WithField("file", filePath).WithField("findings_count", len(findings)).Debug("Security scan completed")
		ms.addSecurityFindings(findings)

		// Analyze and correlate security risks for this specific scan
		ms.analyzeSecurityRisks()
	}
}

// parseConfigFile parses a configuration file and extracts MCP tool information.
func (ms *MCPScanner) parseConfigFile(filePath string, content []byte) *MCPTool {
	fileName := filepath.Base(filePath)

	// Handle different file types
	switch {
	case strings.Contains(fileName, "mcp_servers.json"):
		return ms.parseClaudeCodeConfig(filePath, content)
	case strings.HasSuffix(fileName, ".json"):
		return ms.parseJSONConfig(filePath, content)
	case strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml"):
		return ms.parseYAMLConfig(filePath, content)
	case strings.HasSuffix(fileName, ".env"):
		return ms.parseEnvConfig(filePath, content)
	case strings.Contains(fileName, "docker-compose"):
		return ms.parseDockerComposeConfig(filePath, content)
	}

	return nil
}

// parseClaudeCodeConfig parses Claude Code MCP server configuration.
func (ms *MCPScanner) parseClaudeCodeConfig(filePath string, content []byte) *MCPTool {
	filePath = filepath.Clean(filePath) // Standardize the file path
	var config map[string]interface{}
	if err := json.Unmarshal(content, &config); err != nil {
		ms.logger.WithError(err).WithField("file", filePath).Debug("Failed to parse JSON")
		return nil
	}

	// Extract MCP servers from Claude Code config
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Create a tool entry for the configuration
	tool := &MCPTool{
		ID:            uuid.New().String(),
		Name:          "Claude Code MCP Configuration",
		Type:          "client",
		Status:        "configured",
		ConfigPath:    filePath,
		Configuration: config,
		Dependencies:  []MCPDependency{},
		Timestamp:     time.Now(),
	}

	ms.logger.WithField("tool_config_path", tool.ConfigPath).WithField("file_path", filePath).Debug("Created Claude Code config tool")

	// Extract server dependencies
	for serverName, serverConfig := range mcpServers {
		if serverMap, ok := serverConfig.(map[string]interface{}); ok {
			dep := MCPDependency{
				Name: serverName,
				Type: "server",
			}

			if command, ok := serverMap["command"].(string); ok {
				dep.Path = command
			}

			tool.Dependencies = append(tool.Dependencies, dep)
		}
	}

	return tool
}

// parseJSONConfig parses a generic JSON configuration file.
func (ms *MCPScanner) parseJSONConfig(filePath string, content []byte) *MCPTool {
	var config map[string]interface{}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil
	}

	// Check if this looks like an MCP configuration
	if ms.containsMCPKeywords(config) {
		return &MCPTool{
			ID:            uuid.New().String(),
			Name:          filepath.Base(filePath),
			Type:          "unknown",
			Status:        "configured",
			ConfigPath:    filePath,
			Configuration: config,
			Timestamp:     time.Now(),
		}
	}

	return nil
}

// parseYAMLConfig parses YAML configuration files.
func (ms *MCPScanner) parseYAMLConfig(filePath string, content []byte) *MCPTool {
	// For now, do basic string matching for MCP patterns
	contentStr := string(content)
	if strings.Contains(contentStr, "mcp") ||
		strings.Contains(contentStr, "neo4j") ||
		strings.Contains(contentStr, "duckdb") {
		return &MCPTool{
			ID:         uuid.New().String(),
			Name:       filepath.Base(filePath),
			Type:       "unknown",
			Status:     "configured",
			ConfigPath: filePath,
			Timestamp:  time.Now(),
		}
	}

	return nil
}

// parseEnvConfig parses environment configuration files.
func (ms *MCPScanner) parseEnvConfig(filePath string, content []byte) *MCPTool {
	contentStr := string(content)

	// Look for MCP-related environment variables
	mcpPatterns := []string{
		"MCP_", "NEO4J_", "DUCKDB_", "CHROMA_",
		"CLAUDE_", "API_KEY", "DATABASE_URL",
	}

	for _, pattern := range mcpPatterns {
		if strings.Contains(contentStr, pattern) {
			return &MCPTool{
				ID:         uuid.New().String(),
				Name:       "Environment Configuration",
				Type:       "config",
				Status:     "configured",
				ConfigPath: filePath,
				Timestamp:  time.Now(),
			}
		}
	}

	return nil
}

// parseDockerComposeConfig parses Docker Compose files for MCP services.
func (ms *MCPScanner) parseDockerComposeConfig(filePath string, content []byte) *MCPTool {
	contentStr := string(content)

	// Look for MCP-related services
	if strings.Contains(contentStr, "neo4j") ||
		strings.Contains(contentStr, "duckdb") ||
		strings.Contains(contentStr, "mcp") {
		return &MCPTool{
			ID:         uuid.New().String(),
			Name:       "Docker Compose MCP Services",
			Type:       "container",
			Status:     "configured",
			ConfigPath: filePath,
			Timestamp:  time.Now(),
		}
	}

	return nil
}

// scanRunningProcesses discovers running MCP processes.
func (ms *MCPScanner) scanRunningProcesses(ctx context.Context) error {
	ms.logger.Debug("Scanning running processes for MCP tools")

	// Execute ps command safely
	result, err := ms.executor.Execute(ctx, "ps", "aux")
	if err != nil {
		return fmt.Errorf("failed to execute ps command: %w", err)
	}

	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		// Check if we should exclude this process
		if ms.shouldExcludeProcess(line) {
			continue
		}

		if ms.rules.MatchProcess(line) {
			process := ms.parseProcessLine(line)
			if process != nil {
				tool := ms.processToMCPTool(*process)
				ms.addFinding(tool)
			}
		}
	}

	return nil
}

// parseProcessLine parses a process line from ps output.
func (ms *MCPScanner) parseProcessLine(line string) *ProcessInfo {
	fields := strings.Fields(line)
	if len(fields) < 11 {
		return nil
	}

	// Parse PID
	pid, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil
	}

	// Extract command and arguments
	command := fields[10]
	args := fields[11:]

	procInfo := &ProcessInfo{
		PID:         pid,
		User:        fields[0],
		Command:     command,
		Args:        args,
		ProcessName: filepath.Base(command),
	}

	// Enhance with additional process information
	ms.enhanceProcessInfo(procInfo)

	return procInfo
}

// processToMCPTool converts a ProcessInfo to an MCPTool.
func (ms *MCPScanner) processToMCPTool(process ProcessInfo) MCPTool {
	toolType := "server"
	if strings.Contains(process.ProcessName, "client") {
		toolType = "client"
	} else if strings.Contains(process.ProcessName, "bridge") {
		toolType = "bridge"
	}

	// Enhanced process information
	tool := MCPTool{
		ID:             uuid.New().String(),
		Name:           process.ProcessName,
		Type:           toolType,
		Status:         "running",
		ProcessID:      process.PID,
		Timestamp:      time.Now(),
		User:           process.User,
		ExecutablePath: process.ExePath,
		StartTime:      process.StartTime,
		CommandLine:    ms.redactSensitive(process.CmdLine),
	}

	// Detect version from various sources
	tool.Version = ms.detectVersion(process)

	// Analyze network exposure
	tool.NetworkExposure = ms.analyzeNetworkExposure(process.PID)

	// Extract build information if available
	tool.BuildInfo = ms.extractBuildInfo(process.ExePath)

	return tool
}

// scanNetworkConnections discovers network connections related to MCP tools.
func (ms *MCPScanner) scanNetworkConnections(ctx context.Context) error {
	ms.logger.Debug("Scanning network connections")

	// Execute netstat command safely
	result, err := ms.executor.Execute(ctx, "netstat", "-tulpn")
	if err != nil {
		// netstat might not be available, continue without error
		ms.logger.Debug("netstat not available, skipping network scanning")
		return nil
	}

	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		conn := ms.parseNetworkLine(line)
		if conn != nil && ms.isMCPPort(conn.Port) {
			ms.updateToolWithPort(conn.PID, conn.Port)
		}
	}

	return nil
}

// parseNetworkLine parses a network connection line from netstat output.
func (ms *MCPScanner) parseNetworkLine(line string) *NetworkConnection {
	fields := strings.Fields(line)
	if len(fields) < 6 {
		return nil
	}

	// Parse address and port
	addressPort := fields[3]
	parts := strings.Split(addressPort, ":")
	if len(parts) < 2 {
		return nil
	}

	port, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return nil
	}

	return &NetworkConnection{
		Protocol: fields[0],
		Port:     port,
		Address:  addressPort,
		State:    fields[5],
	}
}

// Helper methods

func (ms *MCPScanner) isConfigFile(fileName string) bool {
	configExtensions := []string{".json", ".yaml", ".yml", ".conf", ".config", ".env"}
	for _, ext := range configExtensions {
		if strings.HasSuffix(fileName, ext) {
			return true
		}
	}
	return false
}

func (ms *MCPScanner) containsMCPKeywords(config map[string]interface{}) bool {
	configStr := fmt.Sprintf("%v", config)
	keywords := []string{"mcp", "neo4j", "duckdb", "chroma", "claude"}

	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(configStr), keyword) {
			return true
		}
	}
	return false
}

func (ms *MCPScanner) isMCPPort(port int) bool {
	// Common MCP-related ports
	mcpPorts := []int{7687, 7688, 7474, 7473, 8080, 8000, 3000, 5000}
	for _, p := range mcpPorts {
		if port == p {
			return true
		}
	}
	return false
}

func (ms *MCPScanner) addFinding(tool MCPTool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.findings = append(ms.findings, tool)
}

func (ms *MCPScanner) addSecurityFindings(findings []SecurityFinding) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	for _, finding := range findings {
		ms.logger.WithField("finding_file_path", finding.FilePath).WithField("finding_category", finding.Category).WithField("finding_name", finding.Name).Debug("Adding security finding")
	}
	ms.risks = append(ms.risks, findings...)
	ms.logger.WithField("total_risks", len(ms.risks)).Debug("Security findings added to risks collection")
}

func (ms *MCPScanner) updateToolWithPort(pid, port int) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for i, tool := range ms.findings {
		if tool.ProcessID == pid {
			ms.findings[i].Port = port
			break
		}
	}
}

func (ms *MCPScanner) correlateFindings() {
	// Correlate process and configuration findings
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Simple correlation - match by name patterns
	for i, tool := range ms.findings {
		if tool.Status == "running" && tool.ConfigPath == "" {
			// Try to find matching config
			for _, configTool := range ms.findings {
				if configTool.Status == "configured" &&
					strings.Contains(strings.ToLower(configTool.Name), strings.ToLower(tool.Name)) {
					ms.findings[i].ConfigPath = configTool.ConfigPath
					ms.findings[i].Configuration = configTool.Configuration
					break
				}
			}
		}
	}
}

func (ms *MCPScanner) analyzeSecurityRisks() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.logger.WithField("total_tools", len(ms.findings)).WithField("total_risks", len(ms.risks)).Debug("Starting security risk analysis")

	// Attach security findings to tools
	for i, tool := range ms.findings {
		ms.logger.WithField("tool_name", tool.Name).WithField("tool_config_path", tool.ConfigPath).Debug("Analyzing risks for tool")
		var toolRisks []SecurityFinding

		for _, risk := range ms.risks {
			ms.logger.WithField("tool_config_path", tool.ConfigPath).WithField("risk_file_path", risk.FilePath).Debug("Comparing paths for risk correlation")
			if risk.FilePath == tool.ConfigPath {
				toolRisks = append(toolRisks, risk)
				ms.logger.WithField("risk_category", risk.Category).Debug("Risk correlated with tool")
			}
		}

		ms.findings[i].SecurityRisks = toolRisks
		ms.logger.WithField("tool_name", tool.Name).WithField("risks_count", len(toolRisks)).Debug("Security risks attached to tool")
	}
}

// shouldExcludePath determines if a path should be excluded from scanning based on includeSelf setting.
func (ms *MCPScanner) shouldExcludePath(path string) bool {
	if ms.includeSelf {
		return false // Don't exclude anything if self-scanning is enabled
	}

	// Normalize path for comparison
	path = filepath.Clean(strings.ToLower(path))

	// Strigoi-related patterns to exclude
	strigoiPatterns := []string{
		"strigoi",                      // Binary name
		"/cmd/strigoi",                 // Command directory
		"/pkg/security",                // Security package
		"/pkg/modules",                 // Modules package
		"/modules/probe",               // Probe modules
		"github.com/macawi-ai/strigoi", // Go module path
		".git/macawi-ai/strigoi",       // Git repo path
	}

	for _, pattern := range strigoiPatterns {
		if strings.Contains(path, pattern) {
			ms.logger.WithField("path", path).WithField("pattern", pattern).Debug("Excluding Strigoi-related path")
			return true
		}
	}

	return false
}

// shouldExcludeProcess determines if a process should be excluded from scanning.
func (ms *MCPScanner) shouldExcludeProcess(processLine string) bool {
	if ms.includeSelf {
		return false // Don't exclude anything if self-scanning is enabled
	}

	// Check for Strigoi process patterns
	processLine = strings.ToLower(processLine)
	strigoiProcessPatterns := []string{
		"strigoi",
		"./strigoi",
		"/strigoi",
		"probe south",
		"scan-mcp",
	}

	for _, pattern := range strigoiProcessPatterns {
		if strings.Contains(processLine, pattern) {
			ms.logger.WithField("process", processLine).WithField("pattern", pattern).Debug("Excluding Strigoi-related process")
			return true
		}
	}

	return false
}

func (ms *MCPScanner) getFindings() []MCPTool {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Create a copy to avoid race conditions
	findings := make([]MCPTool, len(ms.findings))
	copy(findings, ms.findings)
	return findings
}

// Enhanced process information gathering methods

// enhanceProcessInfo enriches ProcessInfo with data from /proc filesystem.
func (ms *MCPScanner) enhanceProcessInfo(info *ProcessInfo) {
	if info == nil || info.PID == 0 {
		return
	}

	// Read executable path
	exePath, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", info.PID))
	if err == nil {
		info.ExePath = exePath
	}

	// Read working directory
	cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", info.PID))
	if err == nil {
		info.WorkingDir = cwd
	}

	// Read command line (full, including arguments)
	cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", info.PID))
	if err == nil {
		// Replace null bytes with spaces for display
		info.CmdLine = strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
	}

	// Read process status for additional info
	statusBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", info.PID))
	if err == nil {
		statusLines := strings.Split(string(statusBytes), "\n")
		for _, line := range statusLines {
			if strings.HasPrefix(line, "PPid:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					info.ParentPID, _ = strconv.Atoi(fields[1])
				}
			}
		}
	}

	// Parse start time from /proc/[pid]/stat (field 22)
	_, err = os.ReadFile(fmt.Sprintf("/proc/%d/stat", info.PID))
	if err == nil {
		// This is complex due to process names potentially containing spaces/parens
		// For now, we'll skip start time parsing to avoid complexity
		ms.logger.Debug("Process stat data available for enhanced analysis")
	}
}

// detectVersion attempts to detect the version of an MCP tool.
func (ms *MCPScanner) detectVersion(process ProcessInfo) string {
	// Check command-line arguments for version flags
	for _, arg := range process.Args {
		if strings.Contains(arg, "--version=") {
			parts := strings.Split(arg, "=")
			if len(parts) >= 2 {
				return parts[1]
			}
		}
		if strings.Contains(arg, "-v=") {
			parts := strings.Split(arg, "=")
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}

	// Try to extract version from binary if we have the path
	if process.ExePath != "" {
		if version := ms.extractVersionFromBinary(process.ExePath); version != "" {
			return version
		}
	}

	// Check environment variables (safely)
	if process.Environment != nil {
		if version, ok := process.Environment["VERSION"]; ok {
			return version
		}
		if version, ok := process.Environment["APP_VERSION"]; ok {
			return version
		}
	}

	return "unknown"
}

// extractVersionFromBinary attempts to extract version info from the binary.
func (ms *MCPScanner) extractVersionFromBinary(exePath string) string {
	if exePath == "" {
		return ""
	}

	// Use strings command to extract printable strings
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := ms.executor.Execute(ctx, "strings", exePath)
	if err != nil {
		ms.logger.Debug("Could not run strings on binary")
		return ""
	}

	// Look for version patterns in the output
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		// Common version patterns
		if strings.Contains(line, "version:") ||
			strings.Contains(line, "Version:") ||
			strings.Contains(line, "VERSION:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				version := strings.TrimSpace(parts[1])
				if version != "" && len(version) < 50 { // Sanity check
					return version
				}
			}
		}
		// Semantic version pattern (e.g., v1.2.3)
		if strings.HasPrefix(line, "v") && len(line) < 20 {
			// Simple check for version-like string
			if strings.Count(line, ".") >= 1 {
				return strings.TrimSpace(line)
			}
		}
	}

	return ""
}

// extractBuildInfo extracts build information from the binary.
func (ms *MCPScanner) extractBuildInfo(exePath string) map[string]string {
	buildInfo := make(map[string]string)

	if exePath == "" {
		return buildInfo
	}

	// Get file info
	fileInfo, err := os.Stat(exePath)
	if err == nil {
		buildInfo["modified"] = fileInfo.ModTime().Format(time.RFC3339)
		buildInfo["size"] = fmt.Sprintf("%d", fileInfo.Size())
		buildInfo["mode"] = fileInfo.Mode().String()
	}

	// Calculate hash for integrity checking
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := ms.executor.Execute(ctx, "sha256sum", exePath)
	if err == nil {
		parts := strings.Fields(result.Stdout)
		if len(parts) >= 1 {
			buildInfo["sha256"] = parts[0]
		}
	}

	return buildInfo
}

// analyzeNetworkExposure analyzes the network exposure of a process.
func (ms *MCPScanner) analyzeNetworkExposure(pid int) NetworkExposureInfo {
	exposure := NetworkExposureInfo{
		ListeningPorts: []int{},
		RiskFactors:    []string{},
	}

	// Use ss or netstat to find listening ports for this PID
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try ss first (more modern)
	result, err := ms.executor.Execute(ctx, "ss", "-tlnp")
	if err != nil {
		// Fall back to netstat
		result, err = ms.executor.Execute(ctx, "netstat", "-tlnp")
		if err != nil {
			ms.logger.Debug("Could not analyze network connections")
			return exposure
		}
	}

	// Parse output looking for our PID
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf("pid=%d", pid)) {
			// Extract port and bind address
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.Contains(field, ":") && !strings.Contains(field, "::") {
					parts := strings.Split(field, ":")
					if len(parts) >= 2 {
						port, err := strconv.Atoi(parts[len(parts)-1])
						if err == nil {
							exposure.ListeningPorts = append(exposure.ListeningPorts, port)

							// Analyze bind address
							if strings.HasPrefix(field, "0.0.0.0:") || strings.HasPrefix(field, "*:") {
								exposure.BindAddress = "0.0.0.0"
								exposure.ExposureLevel = "network"
								exposure.RiskFactors = append(exposure.RiskFactors, "Listening on all interfaces")
							} else if strings.HasPrefix(field, "127.0.0.1:") || strings.HasPrefix(field, "localhost:") {
								exposure.BindAddress = "127.0.0.1"
								exposure.ExposureLevel = "local"
							} else {
								exposure.BindAddress = field
								exposure.ExposureLevel = "network"
							}
						}
					}
				}
			}
		}
	}

	// Check for common MCP ports without TLS
	for _, port := range exposure.ListeningPorts {
		// Common database ports that might need TLS
		if port == 5432 || port == 3306 || port == 27017 || port == 6379 {
			exposure.RiskFactors = append(exposure.RiskFactors, fmt.Sprintf("Database port %d may need TLS", port))
		}
	}

	return exposure
}

// redactSensitive removes sensitive information from strings.
func (ms *MCPScanner) redactSensitive(input string) string {
	if input == "" {
		return input
	}

	// Patterns to redact
	patterns := []struct {
		pattern     string
		replacement string
	}{
		{`password=[\S]+`, "password=[REDACTED]"},
		{`PASSWORD=[\S]+`, "PASSWORD=[REDACTED]"},
		{`pass=[\S]+`, "pass=[REDACTED]"},
		{`api[_-]?key=[\S]+`, "api_key=[REDACTED]"},
		{`API[_-]?KEY=[\S]+`, "API_KEY=[REDACTED]"},
		{`token=[\S]+`, "token=[REDACTED]"},
		{`TOKEN=[\S]+`, "TOKEN=[REDACTED]"},
		{`secret=[\S]+`, "secret=[REDACTED]"},
		{`SECRET=[\S]+`, "SECRET=[REDACTED]"},
		{`private[_-]?key=[\S]+`, "private_key=[REDACTED]"},
		{`bearer\s+[\S]+`, "bearer [REDACTED]"},
	}

	result := input
	for _, p := range patterns {
		re := regexp.MustCompile(p.pattern)
		result = re.ReplaceAllString(result, p.replacement)
	}

	return result
}
