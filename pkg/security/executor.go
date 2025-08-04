package security

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// SecureExecutor provides safe command execution with strict validation.
type SecureExecutor struct {
	allowedCommands map[string]*CommandSpec
	defaultTimeout  time.Duration
	maxOutputSize   int64
}

// CommandSpec defines allowed commands and their validation rules.
type CommandSpec struct {
	Path          string               `json:"path"`
	AllowedArgs   []ArgumentSpec       `json:"allowed_args"`
	RequiresPrivs bool                 `json:"requires_privs"`
	Timeout       time.Duration        `json:"timeout"`
	MaxOutputSize int64                `json:"max_output_size"`
	Validator     func([]string) error `json:"-"` // Custom validation function
}

// ArgumentSpec defines validation for command arguments.
type ArgumentSpec struct {
	Pattern     string `json:"pattern"`     // Regex pattern for validation
	Required    bool   `json:"required"`    // Whether this argument is required
	Description string `json:"description"` // Human-readable description
}

// ExecutionResult contains the result of command execution.
type ExecutionResult struct {
	Command  string        `json:"command"`
	Args     []string      `json:"args"`
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
	Error    error         `json:"error,omitempty"`
}

// NewSecureExecutor creates a new SecureExecutor with default safe commands.
func NewSecureExecutor() *SecureExecutor {
	executor := &SecureExecutor{
		allowedCommands: make(map[string]*CommandSpec),
		defaultTimeout:  30 * time.Second,
		maxOutputSize:   10 * 1024 * 1024, // 10MB default limit
	}

	// Register default safe commands
	executor.registerDefaultCommands()
	return executor
}

// registerDefaultCommands registers commonly used safe commands.
func (se *SecureExecutor) registerDefaultCommands() {
	// Package managers - ignore errors as these are hardcoded safe commands
	_ = se.RegisterCommand("apt", &CommandSpec{
		Path: "/usr/bin/apt",
		AllowedArgs: []ArgumentSpec{
			{Pattern: "^(list|show|policy)$", Required: true, Description: "apt command"},
			{Pattern: "^--installed$", Required: false, Description: "list installed packages"},
			{Pattern: "^[a-zA-Z0-9._+-]+$", Required: false, Description: "package name"},
		},
		Timeout:       15 * time.Second,
		MaxOutputSize: 5 * 1024 * 1024,
	})

	_ = se.RegisterCommand("yum", &CommandSpec{
		Path: "/usr/bin/yum",
		AllowedArgs: []ArgumentSpec{
			{Pattern: "^(list|info)$", Required: true, Description: "yum command"},
			{Pattern: "^(installed|available)$", Required: false, Description: "package status"},
			{Pattern: "^[a-zA-Z0-9._+-]+$", Required: false, Description: "package name"},
		},
		Timeout:       15 * time.Second,
		MaxOutputSize: 5 * 1024 * 1024,
	})

	_ = se.RegisterCommand("brew", &CommandSpec{
		Path: "/usr/local/bin/brew",
		AllowedArgs: []ArgumentSpec{
			{Pattern: "^(list|info)$", Required: true, Description: "brew command"},
			{Pattern: "^[a-zA-Z0-9._+-]+$", Required: false, Description: "package name"},
		},
		Timeout:       15 * time.Second,
		MaxOutputSize: 5 * 1024 * 1024,
	})

	// System information
	_ = se.RegisterCommand("ps", &CommandSpec{
		Path: "/bin/ps",
		AllowedArgs: []ArgumentSpec{
			{Pattern: "^(aux|ef)$", Required: true, Description: "ps format"},
		},
		Timeout:       10 * time.Second,
		MaxOutputSize: 2 * 1024 * 1024,
	})

	_ = se.RegisterCommand("ldd", &CommandSpec{
		Path: "/usr/bin/ldd",
		AllowedArgs: []ArgumentSpec{
			{Pattern: "^/[a-zA-Z0-9/_.-]+$", Required: true, Description: "binary path"},
		},
		Timeout:       5 * time.Second,
		MaxOutputSize: 1024 * 1024,
	})

	// Network tools
	_ = se.RegisterCommand("netstat", &CommandSpec{
		Path: "/bin/netstat",
		AllowedArgs: []ArgumentSpec{
			{Pattern: "^-[tuplna]+$", Required: true, Description: "netstat options"},
		},
		Timeout:       10 * time.Second,
		MaxOutputSize: 1024 * 1024,
	})
}

// RegisterCommand registers a new allowed command with validation rules.
func (se *SecureExecutor) RegisterCommand(name string, spec *CommandSpec) error {
	if name == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	if spec == nil {
		return fmt.Errorf("command spec cannot be nil")
	}

	if spec.Path == "" {
		return fmt.Errorf("command path cannot be empty")
	}

	// Validate path doesn't contain shell metacharacters
	if strings.ContainsAny(spec.Path, "|&;<>()$`\\\"' \t\n*?[") {
		return fmt.Errorf("command path contains unsafe characters: %s", spec.Path)
	}

	// Set defaults if not specified
	if spec.Timeout == 0 {
		spec.Timeout = se.defaultTimeout
	}
	if spec.MaxOutputSize == 0 {
		spec.MaxOutputSize = se.maxOutputSize
	}

	se.allowedCommands[name] = spec
	return nil
}

// Execute safely executes a command with strict validation.
func (se *SecureExecutor) Execute(ctx context.Context, command string, args ...string) (*ExecutionResult, error) {
	startTime := time.Now()

	// Validate command is allowed
	spec, exists := se.allowedCommands[command]
	if !exists {
		return nil, fmt.Errorf("command not allowed: %s", command)
	}

	// Validate arguments
	if err := se.validateArguments(spec, args); err != nil {
		return nil, fmt.Errorf("argument validation failed: %w", err)
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, spec.Timeout)
	defer cancel()

	// Execute command - gosec G204: This is safe because:
	// 1. spec.Path is whitelisted and validated during RegisterCommand
	// 2. args have been validated against strict regex patterns
	// 3. No shell interpretation is used (direct command execution)
	cmd := exec.CommandContext(execCtx, spec.Path, args...) // #nosec G204

	// Capture output
	stdout, err := cmd.Output()
	duration := time.Since(startTime)

	result := &ExecutionResult{
		Command:  command,
		Args:     args,
		Duration: duration,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
			result.Stderr = string(exitError.Stderr)
		}
		result.Error = err
		return result, err
	}

	// Check output size limits
	if int64(len(stdout)) > spec.MaxOutputSize {
		return nil, fmt.Errorf("command output exceeds size limit: %d bytes", len(stdout))
	}

	result.Stdout = string(stdout)
	return result, nil
}

// validateArguments validates command arguments against the spec.
func (se *SecureExecutor) validateArguments(spec *CommandSpec, args []string) error {
	// Check for shell metacharacters in all arguments
	for _, arg := range args {
		if strings.ContainsAny(arg, "|&;<>()$`\\\"'\t\n*") {
			return fmt.Errorf("argument contains unsafe characters: %s", arg)
		}
	}

	// Apply custom validator if present
	if spec.Validator != nil {
		if err := spec.Validator(args); err != nil {
			return fmt.Errorf("custom validation failed: %w", err)
		}
	}

	// Validate against argument specifications
	requiredCount := 0
	for _, argSpec := range spec.AllowedArgs {
		if argSpec.Required {
			requiredCount++
		}
	}

	if len(args) < requiredCount {
		return fmt.Errorf("insufficient arguments: got %d, need at least %d", len(args), requiredCount)
	}

	// Validate each argument against patterns
	for i, arg := range args {
		if i >= len(spec.AllowedArgs) {
			return fmt.Errorf("too many arguments: got %d, max %d allowed", len(args), len(spec.AllowedArgs))
		}

		argSpec := spec.AllowedArgs[i]
		if argSpec.Pattern != "" {
			matched, err := regexp.MatchString(argSpec.Pattern, arg)
			if err != nil {
				return fmt.Errorf("invalid regex pattern for argument %d: %w", i, err)
			}
			if !matched {
				return fmt.Errorf("argument %d '%s' doesn't match pattern '%s'", i, arg, argSpec.Pattern)
			}
		}
	}

	return nil
}

// ExecuteQuiet executes a command and returns only success/failure.
func (se *SecureExecutor) ExecuteQuiet(ctx context.Context, command string, args ...string) error {
	result, err := se.Execute(ctx, command, args...)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}
	return nil
}

// GetAllowedCommands returns a list of all allowed commands.
func (se *SecureExecutor) GetAllowedCommands() []string {
	commands := make([]string, 0, len(se.allowedCommands))
	for cmd := range se.allowedCommands {
		commands = append(commands, cmd)
	}
	return commands
}

// ValidatePath ensures a file path is safe to access.
func (se *SecureExecutor) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Block dangerous path patterns
	dangerous := []string{
		"..",     // Path traversal
		"//",     // Double slash
		"/proc/", // Proc filesystem
		"/sys/",  // Sys filesystem
		"/dev/",  // Device files
	}

	lowerPath := strings.ToLower(path)
	for _, pattern := range dangerous {
		if strings.Contains(lowerPath, pattern) {
			return fmt.Errorf("path contains dangerous pattern '%s': %s", pattern, path)
		}
	}

	// Block shell metacharacters
	if strings.ContainsAny(path, "|&;<>()$`\\\"' \t\n*?") {
		return fmt.Errorf("path contains unsafe characters: %s", path)
	}

	return nil
}

// CommandExists checks if a command is available and allowed.
func (se *SecureExecutor) CommandExists(cmd string) bool {
	// Check if command is in our allowed list
	if _, exists := se.allowedCommands[cmd]; !exists {
		return false
	}

	// Check if command exists in system PATH
	_, err := exec.LookPath(cmd)
	return err == nil
}
