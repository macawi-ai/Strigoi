package core

import (
	"fmt"
	"strings"
)

// NavigationMode represents how navigation should be handled
type NavigationMode int

const (
	NavigationExplicit NavigationMode = iota // Requires "cd" or "go"
	NavigationImplicit                        // Old behavior (for backward compat)
)

// GlobalCommands that are available everywhere
var GlobalCommands = map[string]bool{
	"cd":    true,
	"pwd":   true,
	"ls":    true,
	"dir":   true,
	"help":  true,
	"exit":  true,
	"clear": true,
	"alias": true,
	"..":    true,
	"back":  true,
	"/":     true,
}

// isNavigationCommand checks if a command is a navigation command
func isNavigationCommand(cmd string) bool {
	switch cmd {
	case "cd", "go", "..", "back", "/":
		return true
	default:
		return false
	}
}

// handleExplicitNavigation handles navigation with explicit cd/go commands
func (c *ConsoleV2) handleExplicitNavigation(cmd *ParsedCommand) error {
	if len(cmd.Path) == 0 {
		return fmt.Errorf("invalid command")
	}
	
	action := cmd.Path[0]
	
	switch action {
	case "cd", "go":
		// cd requires a target
		if len(cmd.Args) == 0 {
			return fmt.Errorf("usage: %s <directory>", action)
		}
		
		target := cmd.Args[0]
		return c.navigateToTarget(target)
		
	case "..", "back":
		if err := c.navigator.NavigateUp(); err != nil {
			return err
		}
		c.Info("Current directory: %s", c.navigator.GetBreadcrumb())
		return nil
		
	case "/":
		c.navigator.NavigateToRoot()
		c.Info("Current directory: /")
		return nil
		
	default:
		return fmt.Errorf("unknown navigation command: %s", action)
	}
}

// navigateToTarget navigates to a specific target
func (c *ConsoleV2) navigateToTarget(target string) error {
	// Handle special targets
	switch target {
	case "..", "back":
		return c.navigator.NavigateUp()
	case "/", "~", "root":
		c.navigator.NavigateToRoot()
		c.Info("Current directory: /")
		return nil
	}
	
	// Handle absolute paths
	if strings.HasPrefix(target, "/") {
		path := strings.TrimPrefix(target, "/")
		parts := strings.Split(path, "/")
		
		// Validate the path exists
		if _, err := c.rootCommand.FindCommand(parts); err != nil {
			return fmt.Errorf("directory not found: %s", target)
		}
		
		c.navigator.NavigateTo(parts)
		c.Info("Current directory: %s", c.navigator.GetBreadcrumb())
		return nil
	}
	
	// Handle relative paths
	currentPath := c.navigator.GetCurrentPath()
	targetParts := strings.Split(target, "/")
	newPath := append(currentPath, targetParts...)
	
	// Validate the path exists and is a directory (has children)
	node, err := c.rootCommand.FindCommand(newPath)
	if err != nil {
		return fmt.Errorf("directory not found: %s", target)
	}
	
	// Check if it's actually a directory (has children)
	if len(node.Children) == 0 && node.Handler != nil {
		return fmt.Errorf("'%s' is a command, not a directory", target)
	}
	
	c.navigator.NavigateTo(newPath)
	c.Info("Current directory: %s", c.navigator.GetBreadcrumb())
	return nil
}

// handlePwd handles the pwd command
func (c *ConsoleV2) handlePwd(cmd *ParsedCommand) error {
	c.Printf("%s\n", c.navigator.GetBreadcrumb())
	return nil
}

// handleLs handles the ls/dir command
func (c *ConsoleV2) handleLs(cmd *ParsedCommand) error {
	// Get current node
	currentPath := c.navigator.GetCurrentPath()
	var node *CommandNode
	var err error
	
	if len(currentPath) == 0 {
		node = c.rootCommand
	} else {
		node, err = c.rootCommand.FindCommand(currentPath)
		if err != nil {
			return err
		}
	}
	
	// Separate commands and directories
	var commands []string
	var directories []string
	
	for name, child := range node.Children {
		if child.Hidden {
			continue
		}
		
		if len(child.Children) > 0 || (child.Handler == nil) {
			// It's a directory (has children OR no handler - empty directory)
			directories = append(directories, name)
		} else if child.Handler != nil {
			// It's an executable command
			commands = append(commands, name)
		}
	}
	
	// Display directories first
	if len(directories) > 0 {
		c.Info("Directories:")
		for _, dir := range directories {
			child := node.Children[dir]
			// Use blue color for directories
			c.dirColor.Printf("  %-20s", dir+"/")
			c.Printf("  %s\n", child.Description)
		}
	}
	
	// Then display commands
	if len(commands) > 0 {
		if len(directories) > 0 {
			c.Println() // Blank line between sections
		}
		c.Info("Commands:")
		for _, cmd := range commands {
			child := node.Children[cmd]
			
			// Determine command type for coloring
			switch cmd {
			case "help", "exit", "clear", "pwd", "ls", "cd":
				// Utility commands in white
				c.utilColor.Printf("  %-20s", cmd)
			case "alias":
				// Alias command in cyan
				c.aliasColor.Printf("  %-20s", cmd)
			default:
				// Action commands in green
				c.cmdColor.Printf("  %-20s", cmd)
			}
			c.Printf("  %s\n", child.Description)
		}
	}
	
	if len(directories) == 0 && len(commands) == 0 {
		c.Info("(empty directory)")
	}
	
	return nil
}

// processCommandV3 is the improved command processor with explicit navigation
func (c *ConsoleV2) processCommandV3(input string) error {
	// Parse the command BEFORE alias expansion to check for global commands
	preCmd, err := c.parser.Parse(input)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}
	
	// Empty command
	if len(preCmd.Path) == 0 {
		return nil
	}
	
	// Check if it's a global navigation command BEFORE alias expansion
	firstCmd := preCmd.Path[0]
	switch firstCmd {
	case "cd", "go", "..", "back", "/", "pwd", "ls", "dir":
		// These are global commands, handle them without alias expansion
		switch firstCmd {
		case "cd", "go", "..", "back", "/":
			return c.handleExplicitNavigation(preCmd)
		case "pwd":
			return c.handlePwd(preCmd)
		case "ls", "dir":
			return c.handleLs(preCmd)
		}
	}
	
	// Not a navigation command, proceed with alias expansion
	expandedInput := c.aliasManager.ExpandAlias(input)
	
	// Re-parse if alias was expanded
	var cmd *ParsedCommand
	if expandedInput != input {
		cmd, err = c.parser.Parse(expandedInput)
		if err != nil {
			return fmt.Errorf("parse error after alias expansion: %w", err)
		}
	} else {
		cmd = preCmd
	}
	
	// Check for other global commands
	firstCmd = cmd.Path[0]
	switch firstCmd {
	case "help":
		return c.handleHelp(c, cmd)
	case "exit":
		return c.handleExit(c, cmd)
	case "clear":
		return c.handleClear(c, cmd)
	case "alias":
		return c.handleAlias(c, cmd)
	default:
		// Not a global command, try to execute in current context
		return c.executeInContext(cmd)
	}
}

// executeInContext executes a command in the current context
func (c *ConsoleV2) executeInContext(cmd *ParsedCommand) error {
	// Build full path from current context
	currentPath := c.navigator.GetCurrentPath()
	fullPath := append(currentPath, cmd.Path...)
	
	// Find the command node
	node, err := c.rootCommand.FindCommand(fullPath)
	if err != nil {
		// Try fuzzy matching
		suggestion, score := c.fuzzyMatcher.SuggestCommand(strings.Join(cmd.Path, "/"))
		if score > 0.7 {
			c.Warn("Command not found. Did you mean '%s'?", suggestion)
			c.Info("Use 'ls' to see available commands")
		} else {
			c.Error("Command not found: %s", strings.Join(cmd.Path, "/"))
			c.Info("Use 'ls' to see available commands")
		}
		return nil
	}
	
	// Check if it has a handler
	if node.Handler == nil {
		if len(node.Children) > 0 {
			// It's a directory, not a command
			c.Error("'%s' is a directory, not a command", strings.Join(cmd.Path, "/"))
			c.Info("Use 'cd %s' to navigate there", strings.Join(cmd.Path, "/"))
			return nil
		}
		return fmt.Errorf("command %s has no handler", strings.Join(fullPath, "/"))
	}
	
	// Validate the command
	if err := node.ValidateCommand(cmd); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	
	// Execute the handler
	return node.Handler(c, cmd)
}