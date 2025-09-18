package main

import (
	"testing"
)

func TestBuildCommandTree(t *testing.T) {
	// Build a fresh command tree for testing
	tree := buildCommandTree()

	// Test probe directory exists
	if probe, ok := tree.Children["probe"]; !ok {
		t.Error("probe directory not found in command tree")
	} else {
		if !probe.IsDirectory {
			t.Error("probe should be a directory")
		}
		if probe.Description != "Discovery and reconnaissance tools" {
			t.Errorf("unexpected probe description: %s", probe.Description)
		}

		// Test probe commands
		expectedCommands := []string{"north", "south", "east", "west"}
		for _, cmd := range expectedCommands {
			if _, ok := probe.Children[cmd]; !ok {
				t.Errorf("probe command %s not found", cmd)
			}
		}
	}

	// Note: Stream functionality has been externalized to separate tools
	// (as documented in buildCommandTree function)
}

func TestGetPath(t *testing.T) {
	// Build command tree for testing
	tree := buildCommandTree()

	// Save and restore global state
	oldTree := commandTree
	defer func() {
		commandTree = oldTree
	}()

	commandTree = tree

	tests := []struct {
		name     string
		node     *CommandNode
		expected string
	}{
		{
			name:     "root path",
			node:     tree,
			expected: "/",
		},
		{
			name:     "probe directory",
			node:     tree.Children["probe"],
			expected: "/probe",
		},
		{
			name:     "probe north command",
			node:     tree.Children["probe"].Children["north"],
			expected: "/probe/north",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPath(tt.node)
			if result != tt.expected {
				t.Errorf("getPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHandleCD(t *testing.T) {
	// Build command tree for testing
	tree := buildCommandTree()

	// Save and restore global state
	oldTree := commandTree
	oldNode := currentNode
	defer func() {
		commandTree = oldTree
		currentNode = oldNode
	}()

	commandTree = tree
	currentNode = tree

	tests := []struct {
		name         string
		args         []string
		expectedPath string
	}{
		{
			name:         "cd to root with no args",
			args:         []string{"cd"},
			expectedPath: "/",
		},
		{
			name:         "cd to absolute root",
			args:         []string{"cd", "/"},
			expectedPath: "/",
		},
		{
			name:         "cd to probe",
			args:         []string{"cd", "probe"},
			expectedPath: "/probe",
		},
		{
			name:         "cd to parent",
			args:         []string{"cd", ".."},
			expectedPath: "/",
		},
		{
			name:         "cd to current",
			args:         []string{"cd", "."},
			expectedPath: "/probe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set starting position if needed
			if tt.name == "cd to parent" || tt.name == "cd to current" {
				currentNode = commandTree.Children["probe"]
			} else {
				currentNode = commandTree
			}

			handleCD(tt.args)

			result := getPath(currentNode)
			if result != tt.expectedPath {
				t.Errorf("after handleCD(%v), path = %v, want %v", tt.args, result, tt.expectedPath)
			}
		})
	}
}

func TestBuildFullCommand(t *testing.T) {
	// Build command tree for testing
	tree := buildCommandTree()

	// Save and restore global state
	oldTree := commandTree
	oldNode := currentNode
	defer func() {
		commandTree = oldTree
		currentNode = oldNode
	}()

	commandTree = tree
	currentNode = tree

	tests := []struct {
		name     string
		node     *CommandNode
		expected []string
	}{
		{
			name:     "root command",
			node:     commandTree,
			expected: []string{},
		},
		{
			name:     "probe directory",
			node:     commandTree.Children["probe"],
			expected: []string{"probe"},
		},
		{
			name:     "probe north command",
			node:     commandTree.Children["probe"].Children["north"],
			expected: []string{"probe", "north"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFullCommand(tt.node)

			if len(result) != len(tt.expected) {
				t.Errorf("buildFullCommand() returned %d elements, want %d", len(result), len(tt.expected))
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("buildFullCommand()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestGetPrompt(t *testing.T) {
	// Build command tree for testing
	tree := buildCommandTree()

	// Save and restore global state
	oldTree := commandTree
	oldNode := currentNode
	defer func() {
		commandTree = oldTree
		currentNode = oldNode
	}()

	commandTree = tree

	tests := []struct {
		name     string
		node     *CommandNode
		expected string
	}{
		{
			name:     "root prompt",
			node:     commandTree,
			expected: "strigoi> ",
		},
		{
			name:     "probe prompt",
			node:     commandTree.Children["probe"],
			expected: "strigoi/probe> ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentNode = tt.node
			result := getPrompt()
			if result != tt.expected {
				t.Errorf("getPrompt() = %v, want %v", result, tt.expected)
			}
		})
	}
}
