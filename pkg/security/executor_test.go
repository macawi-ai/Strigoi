package security

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewSecureExecutor(t *testing.T) {
	executor := NewSecureExecutor()

	if executor.defaultTimeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", executor.defaultTimeout)
	}

	if executor.maxOutputSize != 10*1024*1024 {
		t.Errorf("Expected max output size 10MB, got %d", executor.maxOutputSize)
	}

	// Check that default commands are registered
	allowedCommands := executor.GetAllowedCommands()
	expectedCommands := []string{"apt", "yum", "brew", "ps", "ldd", "netstat"}

	for _, expected := range expectedCommands {
		found := false
		for _, allowed := range allowedCommands {
			if allowed == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command '%s' not found in allowed commands", expected)
		}
	}
}

func TestRegisterCommand(t *testing.T) {
	executor := NewSecureExecutor()

	tests := []struct {
		name        string
		commandName string
		spec        *CommandSpec
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid command",
			commandName: "testcmd",
			spec: &CommandSpec{
				Path: "/bin/testcmd",
				AllowedArgs: []ArgumentSpec{
					{Pattern: "^test$", Required: true},
				},
			},
			expectError: false,
		},
		{
			name:        "empty command name",
			commandName: "",
			spec:        &CommandSpec{Path: "/bin/test"},
			expectError: true,
			errorMsg:    "command name cannot be empty",
		},
		{
			name:        "nil spec",
			commandName: "test",
			spec:        nil,
			expectError: true,
			errorMsg:    "command spec cannot be nil",
		},
		{
			name:        "empty path",
			commandName: "test",
			spec:        &CommandSpec{Path: ""},
			expectError: true,
			errorMsg:    "command path cannot be empty",
		},
		{
			name:        "unsafe path characters",
			commandName: "test",
			spec:        &CommandSpec{Path: "/bin/test; rm -rf /"},
			expectError: true,
			errorMsg:    "command path contains unsafe characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.RegisterCommand(tt.commandName, tt.spec)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateArguments(t *testing.T) {
	executor := NewSecureExecutor()

	spec := &CommandSpec{
		Path: "/bin/test",
		AllowedArgs: []ArgumentSpec{
			{Pattern: "^(list|show)$", Required: true, Description: "action"},
			{Pattern: "^[a-zA-Z0-9._-]+$", Required: false, Description: "package name"},
		},
	}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid args",
			args:        []string{"list", "package-name"},
			expectError: false,
		},
		{
			name:        "valid single required arg",
			args:        []string{"list"},
			expectError: false,
		},
		{
			name:        "shell injection attempt",
			args:        []string{"list; rm -rf /"},
			expectError: true,
			errorMsg:    "argument contains unsafe characters",
		},
		{
			name:        "pipe injection attempt",
			args:        []string{"list | cat /etc/passwd"},
			expectError: true,
			errorMsg:    "argument contains unsafe characters",
		},
		{
			name:        "insufficient arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "insufficient arguments",
		},
		{
			name:        "invalid pattern",
			args:        []string{"invalid-action"},
			expectError: true,
			errorMsg:    "doesn't match pattern",
		},
		{
			name:        "too many arguments",
			args:        []string{"list", "package", "extra"},
			expectError: true,
			errorMsg:    "too many arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validateArguments(spec, tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	executor := NewSecureExecutor()

	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid path",
			path:        "/usr/bin/test",
			expectError: false,
		},
		{
			name:        "valid relative path",
			path:        "test/file.txt",
			expectError: false,
		},
		{
			name:        "empty path",
			path:        "",
			expectError: true,
			errorMsg:    "path cannot be empty",
		},
		{
			name:        "path traversal",
			path:        "/usr/bin/../../../etc/passwd",
			expectError: true,
			errorMsg:    "path contains dangerous pattern",
		},
		{
			name:        "proc filesystem",
			path:        "/proc/1/mem",
			expectError: true,
			errorMsg:    "path contains dangerous pattern",
		},
		{
			name:        "shell metacharacters",
			path:        "/usr/bin/test; rm -rf /",
			expectError: true,
			errorMsg:    "path contains unsafe characters",
		},
		{
			name:        "command substitution",
			path:        "/usr/bin/$(whoami)",
			expectError: true,
			errorMsg:    "path contains unsafe characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidatePath(tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExecute_CommandNotAllowed(t *testing.T) {
	executor := NewSecureExecutor()
	ctx := context.Background()

	_, err := executor.Execute(ctx, "forbidden-command", "arg1")

	if err == nil {
		t.Errorf("Expected error for forbidden command")
	}

	if !strings.Contains(err.Error(), "command not allowed") {
		t.Errorf("Expected 'command not allowed' error, got: %v", err)
	}
}

func TestExecute_SafeCommand(t *testing.T) {
	executor := NewSecureExecutor()
	ctx := context.Background()

	// Register a safe test command (echo is commonly available)
	err := executor.RegisterCommand("echo", &CommandSpec{
		Path: "/bin/echo",
		AllowedArgs: []ArgumentSpec{
			{Pattern: "^[a-zA-Z0-9 ._-]+$", Required: false, Description: "text to echo"},
		},
		Timeout:       5 * time.Second,
		MaxOutputSize: 1024,
	})

	if err != nil {
		t.Fatalf("Failed to register echo command: %v", err)
	}

	result, err := executor.Execute(ctx, "echo", "hello world")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("Expected result but got nil")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	expectedOutput := "hello world\n"
	if result.Stdout != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, result.Stdout)
	}
}

func TestExecute_Timeout(t *testing.T) {
	executor := NewSecureExecutor()
	ctx := context.Background()

	// Register a command with very short timeout
	err := executor.RegisterCommand("sleep", &CommandSpec{
		Path: "/usr/bin/sleep",
		AllowedArgs: []ArgumentSpec{
			{Pattern: "^[0-9]+$", Required: true, Description: "seconds to sleep"},
		},
		Timeout:       100 * time.Millisecond, // Very short timeout
		MaxOutputSize: 1024,
	})

	if err != nil {
		t.Fatalf("Failed to register sleep command: %v", err)
	}

	result, err := executor.Execute(ctx, "sleep", "1")

	// Should timeout and return an error
	if err == nil {
		t.Errorf("Expected timeout error but got none")
	}

	// Check that it actually timed out quickly
	if result != nil && result.Duration > 200*time.Millisecond {
		t.Errorf("Expected quick timeout, but took %v", result.Duration)
	}
}

func TestExecuteQuiet(t *testing.T) {
	executor := NewSecureExecutor()
	ctx := context.Background()

	// Register true command (should always succeed)
	err := executor.RegisterCommand("true", &CommandSpec{
		Path:          "/bin/true",
		AllowedArgs:   []ArgumentSpec{},
		Timeout:       5 * time.Second,
		MaxOutputSize: 1024,
	})

	if err != nil {
		t.Fatalf("Failed to register true command: %v", err)
	}

	err = executor.ExecuteQuiet(ctx, "true")

	if err != nil {
		t.Errorf("Unexpected error from ExecuteQuiet: %v", err)
	}
}

func TestMaliciousInputs(t *testing.T) {
	executor := NewSecureExecutor()
	ctx := context.Background()

	maliciousInputs := []struct {
		name string
		cmd  string
		args []string
	}{
		{
			name: "command injection via semicolon",
			cmd:  "ps",
			args: []string{"aux; rm -rf /"},
		},
		{
			name: "command injection via pipe",
			cmd:  "ps",
			args: []string{"aux | cat /etc/passwd"},
		},
		{
			name: "command injection via backticks",
			cmd:  "ps",
			args: []string{"aux `whoami`"},
		},
		{
			name: "command injection via dollar",
			cmd:  "ps",
			args: []string{"aux $(whoami)"},
		},
		{
			name: "path traversal",
			cmd:  "ldd",
			args: []string{"../../../etc/passwd"},
		},
	}

	for _, tt := range maliciousInputs {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.Execute(ctx, tt.cmd, tt.args...)

			if err == nil {
				t.Errorf("Expected error for malicious input but got none")
			}

			// Should be validation error, not execution error
			if !strings.Contains(err.Error(), "validation failed") &&
				!strings.Contains(err.Error(), "unsafe characters") &&
				!strings.Contains(err.Error(), "doesn't match pattern") {
				t.Errorf("Expected validation error, got: %v", err)
			}
		})
	}
}
