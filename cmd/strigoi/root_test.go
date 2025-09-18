package main

import (
	"strings"
	"testing"
)

func TestGetBanner(t *testing.T) {
	banner := getBanner()

	// Check banner contains key elements from ASCII art
	if !strings.Contains(banner, "███████╗") {
		t.Error("Banner should contain STRIGOI ASCII art")
	}

	if !strings.Contains(banner, "Advanced Security Validation Platform") {
		t.Error("Banner should contain platform description")
	}

	if !strings.Contains(banner, "Copyright © 2025 Macawi") {
		t.Error("Banner should contain copyright")
	}

	if !strings.Contains(banner, "WHITE HAT SECURITY TESTING") {
		t.Error("Banner should contain security warning")
	}
}

func TestGetColoredUsage(t *testing.T) {
	// Test the actual rootCmd which has proper structure
	usage := getColoredUsage(rootCmd)

	// Check usage contains expected sections
	if !strings.Contains(usage, "Usage:") {
		t.Errorf("Usage should contain Usage section. Got: %v", usage)
	}

	// rootCmd has both directories (probe, stream) and commands (completion, help)
	if !strings.Contains(usage, "Directories:") {
		t.Errorf("Usage should contain Directories section for commands with subcommands. Got: %v", usage)
	}

	if !strings.Contains(usage, "Commands:") {
		t.Errorf("Usage should contain Commands section. Got: %v", usage)
	}

	// Also check that it contains expected commands
	if !strings.Contains(usage, "probe/") {
		t.Error("Usage should show probe as a directory")
	}

	if !strings.Contains(usage, "completion") {
		t.Error("Usage should show completion command")
	}
}

func TestRootCmdStructure(t *testing.T) {
	// Test root command is properly configured
	if rootCmd.Use != "strigoi" {
		t.Errorf("Root command Use = %v, want strigoi", rootCmd.Use)
	}

	if rootCmd.Short != "Advanced Security Validation Platform" {
		t.Error("Root command should have correct short description")
	}

	// Test global flags exist
	helpFlag := rootCmd.PersistentFlags().Lookup("help")
	if helpFlag == nil {
		t.Error("Root command should have help flag")
	}

	versionFlag := rootCmd.PersistentFlags().Lookup("version")
	if versionFlag == nil {
		t.Error("Root command should have version flag")
	}
}

func TestInit(t *testing.T) {
	// Test that init() sets up executeCobraCommand
	if executeCobraCommand == nil {
		t.Error("executeCobraCommand should be set by init()")
	}

	// Test executing a simple command through the function
	err := executeCobraCommand([]string{"--help"})
	if err != nil {
		t.Errorf("executeCobraCommand with --help failed: %v", err)
	}
}
