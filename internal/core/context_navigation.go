package core

import (
	"fmt"
	"strings"
)

// ContextNavigator handles context-aware command navigation
type ContextNavigator struct {
	currentPath []string
	rootNode    *CommandNode
	history     [][]string // Navigation history for back/forward
	historyPos  int
}

// NewContextNavigator creates a new context navigator
func NewContextNavigator(root *CommandNode) *ContextNavigator {
	return &ContextNavigator{
		currentPath: []string{},
		rootNode:    root,
		history:     [][]string{{}}, // Start with root
		historyPos:  0,
	}
}

// GetCurrentPath returns the current context path
func (n *ContextNavigator) GetCurrentPath() []string {
	return append([]string{}, n.currentPath...)
}

// GetCurrentNode returns the current context node
func (n *ContextNavigator) GetCurrentNode() (*CommandNode, error) {
	if len(n.currentPath) == 0 {
		return n.rootNode, nil
	}
	return n.rootNode.FindCommand(n.currentPath)
}

// NavigateTo navigates to a specific path
func (n *ContextNavigator) NavigateTo(path []string) error {
	// Validate the path exists
	if len(path) > 0 {
		if _, err := n.rootNode.FindCommand(path); err != nil {
			return err
		}
	}
	
	// Update current path
	n.currentPath = append([]string{}, path...)
	
	// Add to history
	n.addToHistory(path)
	
	return nil
}

// NavigateRelative navigates relative to current context
func (n *ContextNavigator) NavigateRelative(relativePath string) error {
	switch relativePath {
	case "..", "back":
		return n.NavigateUp()
	case "/", "root":
		return n.NavigateToRoot()
	case "-":
		return n.NavigateBack()
	default:
		// Parse relative path
		parts := strings.Split(relativePath, "/")
		newPath := append(n.currentPath, parts...)
		return n.NavigateTo(newPath)
	}
}

// NavigateUp moves up one level
func (n *ContextNavigator) NavigateUp() error {
	if len(n.currentPath) == 0 {
		return fmt.Errorf("already at root")
	}
	
	newPath := n.currentPath[:len(n.currentPath)-1]
	return n.NavigateTo(newPath)
}

// NavigateToRoot returns to root context
func (n *ContextNavigator) NavigateToRoot() error {
	return n.NavigateTo([]string{})
}

// NavigateBack goes back in history
func (n *ContextNavigator) NavigateBack() error {
	if n.historyPos > 0 {
		n.historyPos--
		n.currentPath = append([]string{}, n.history[n.historyPos]...)
		return nil
	}
	return fmt.Errorf("no previous location in history")
}

// NavigateForward goes forward in history
func (n *ContextNavigator) NavigateForward() error {
	if n.historyPos < len(n.history)-1 {
		n.historyPos++
		n.currentPath = append([]string{}, n.history[n.historyPos]...)
		return nil
	}
	return fmt.Errorf("no next location in history")
}

// addToHistory adds a path to navigation history
func (n *ContextNavigator) addToHistory(path []string) {
	// If we're not at the end of history, truncate forward history
	if n.historyPos < len(n.history)-1 {
		n.history = n.history[:n.historyPos+1]
	}
	
	// Don't add duplicate of current position
	if n.historyPos >= 0 && pathsEqual(n.history[n.historyPos], path) {
		return
	}
	
	// Add new path
	n.history = append(n.history, append([]string{}, path...))
	n.historyPos = len(n.history) - 1
	
	// Limit history size
	const maxHistory = 50
	if len(n.history) > maxHistory {
		n.history = n.history[len(n.history)-maxHistory:]
		n.historyPos = len(n.history) - 1
	}
}

// GetBreadcrumb returns a formatted breadcrumb string
func (n *ContextNavigator) GetBreadcrumb() string {
	if len(n.currentPath) == 0 {
		return "/"
	}
	return "/" + strings.Join(n.currentPath, "/")
}

// GetPathString returns the current path as a string (for TAB completion)
func (n *ContextNavigator) GetPathString() string {
	if len(n.currentPath) == 0 {
		return ""
	}
	return "/" + strings.Join(n.currentPath, "/")
}

// GetAvailableCommands returns available commands in current context
func (n *ContextNavigator) GetAvailableCommands() ([]string, error) {
	node, err := n.GetCurrentNode()
	if err != nil {
		return nil, err
	}
	
	commands := []string{}
	
	// Add navigation commands if not at root
	if len(n.currentPath) > 0 {
		commands = append(commands, "..", "back", "/")
	}
	
	// Add child commands
	for name, child := range node.Children {
		if !child.Hidden {
			if len(child.Children) > 0 {
				commands = append(commands, name+"/")
			} else {
				commands = append(commands, name)
			}
		}
	}
	
	return commands, nil
}

// ResolveCommand resolves a command considering current context
func (n *ContextNavigator) ResolveCommand(input string) ([]string, bool) {
	// Check for absolute path
	if strings.HasPrefix(input, "/") {
		path := strings.TrimPrefix(input, "/")
		if path == "" {
			return []string{}, true
		}
		return strings.Split(path, "/"), true
	}
	
	// Check for navigation commands
	switch input {
	case "..", "back", "-":
		return nil, false // These are navigation, not commands
	}
	
	// Relative path from current context
	if n.currentPath != nil && len(n.currentPath) > 0 {
		return append(n.currentPath, strings.Split(input, "/")...), true
	}
	
	return strings.Split(input, "/"), true
}

// pathsEqual compares two paths for equality
func pathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}