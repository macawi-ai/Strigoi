package main

import (
	"testing"
)

func TestCompletionCmd(t *testing.T) {
	// Test completion command exists and is configured correctly
	if completionCmd.Use != "completion [bash|zsh|fish|powershell]" {
		t.Errorf("Completion command Use = %v", completionCmd.Use)
	}

	if completionCmd.Short != "Generate shell completion script" {
		t.Error("Completion command should have correct short description")
	}

	// Test valid arguments
	validShells := []string{"bash", "zsh", "fish", "powershell"}
	for _, shell := range validShells {
		err := completionCmd.Args(completionCmd, []string{shell})
		if err != nil {
			t.Errorf("Completion command should accept %s as valid argument", shell)
		}
	}

	// Test invalid argument
	err := completionCmd.Args(completionCmd, []string{"invalid"})
	if err == nil {
		t.Error("Completion command should reject invalid shell type")
	}

	// Test no arguments
	err = completionCmd.Args(completionCmd, []string{})
	if err == nil {
		t.Error("Completion command should require an argument")
	}

	// Test too many arguments
	err = completionCmd.Args(completionCmd, []string{"bash", "extra"})
	if err == nil {
		t.Error("Completion command should reject extra arguments")
	}
}

func TestCompletionValidArgs(t *testing.T) {
	// Test the ValidArgs field contains correct shells
	expectedShells := []string{"bash", "zsh", "fish", "powershell"}

	if len(completionCmd.ValidArgs) != len(expectedShells) {
		t.Errorf("Expected %d valid shells, got %d", len(expectedShells), len(completionCmd.ValidArgs))
	}

	// Check all expected shells are present
	shellMap := make(map[string]bool)
	for _, s := range completionCmd.ValidArgs {
		shellMap[s] = true
	}

	for _, expected := range expectedShells {
		if !shellMap[expected] {
			t.Errorf("Expected shell %s not found in ValidArgs", expected)
		}
	}
}
