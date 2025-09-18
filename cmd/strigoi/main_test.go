package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup code before tests

	// Run tests
	code := m.Run()

	// Cleanup code after tests

	os.Exit(code)
}

func TestMainFunction(t *testing.T) {
	// Test that main doesn't panic
	// Note: We can't easily test the actual main() function
	// but we can ensure all components it uses are available

	if rootCmd == nil {
		t.Error("rootCmd should be initialized")
	}

	if completionCmd == nil {
		t.Error("completionCmd should be initialized")
	}

	if probeCmd == nil {
		t.Error("probeCmd should be initialized")
	}

	if streamCmd == nil {
		t.Error("streamCmd should be initialized")
	}
}
