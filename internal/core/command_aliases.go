package core

import (
	"fmt"
	"strings"
	"sync"
)

// CommandAlias represents a command alias
type CommandAlias struct {
	Alias       string
	Command     string
	Description string
}

// AliasManager manages command aliases
type AliasManager struct {
	mu      sync.RWMutex
	aliases map[string]*CommandAlias
}

// NewAliasManager creates a new alias manager
func NewAliasManager() *AliasManager {
	am := &AliasManager{
		aliases: make(map[string]*CommandAlias),
	}
	
	// Add default aliases
	am.AddDefaultAliases()
	
	return am
}

// AddDefaultAliases adds common default aliases
func (am *AliasManager) AddDefaultAliases() {
	defaults := []CommandAlias{
		// Navigation shortcuts
		{Alias: "cd", Command: "context", Description: "Change context (cd stream)"},
		{Alias: "pwd", Command: "context", Description: "Print working context"},
		{Alias: "ls", Command: "list", Description: "List available commands"},
		
		// Common shortcuts
		{Alias: "h", Command: "help", Description: "Show help"},
		{Alias: "?", Command: "help", Description: "Show help"},
		{Alias: "q", Command: "exit", Description: "Quit/exit"},
		{Alias: "quit", Command: "exit", Description: "Quit/exit"},
		
		// Stream shortcuts
		{Alias: "tap", Command: "stream/tap", Description: "Quick stream tap"},
		{Alias: "monitor", Command: "stream/tap --auto-discover", Description: "Auto-monitor processes"},
		
		// Integration shortcuts
		{Alias: "prom", Command: "integrations/prometheus", Description: "Prometheus integration"},
		{Alias: "metrics", Command: "integrations/prometheus/enable", Description: "Enable metrics"},
	}
	
	for _, alias := range defaults {
		am.aliases[alias.Alias] = &alias
	}
}

// AddAlias adds a new alias
func (am *AliasManager) AddAlias(alias, command, description string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	// Validate alias doesn't conflict with core commands
	if am.isReservedCommand(alias) {
		return fmt.Errorf("cannot alias reserved command: %s", alias)
	}
	
	am.aliases[alias] = &CommandAlias{
		Alias:       alias,
		Command:     command,
		Description: description,
	}
	
	return nil
}

// RemoveAlias removes an alias
func (am *AliasManager) RemoveAlias(alias string) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	if _, exists := am.aliases[alias]; !exists {
		return fmt.Errorf("alias not found: %s", alias)
	}
	
	delete(am.aliases, alias)
	return nil
}

// GetAlias returns the command for an alias
func (am *AliasManager) GetAlias(alias string) (string, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	if cmd, exists := am.aliases[alias]; exists {
		return cmd.Command, true
	}
	
	return "", false
}

// ExpandAlias expands an alias in the input string
func (am *AliasManager) ExpandAlias(input string) string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	// Split input to get first word
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return input
	}
	
	// Check if first word is an alias
	if alias, exists := am.aliases[parts[0]]; exists {
		// Replace alias with command
		expandedParts := []string{alias.Command}
		if len(parts) > 1 {
			expandedParts = append(expandedParts, parts[1:]...)
		}
		return strings.Join(expandedParts, " ")
	}
	
	return input
}

// ListAliases returns all configured aliases
func (am *AliasManager) ListAliases() []CommandAlias {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	aliases := make([]CommandAlias, 0, len(am.aliases))
	for _, alias := range am.aliases {
		aliases = append(aliases, *alias)
	}
	
	return aliases
}

// GetAllAliases returns a map of alias name to command for TAB completion
func (am *AliasManager) GetAllAliases() map[string]string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	result := make(map[string]string)
	for name, alias := range am.aliases {
		result[name] = alias.Command
	}
	
	return result
}

// isReservedCommand checks if a name is a reserved command
func (am *AliasManager) isReservedCommand(name string) bool {
	reserved := []string{
		"help", "exit", "clear", "stream", "integrations", 
		"probe", "sense", "respond", "report", "support",
		"state", "jobs", "alias", "unalias",
	}
	
	for _, cmd := range reserved {
		if cmd == name {
			return true
		}
	}
	
	return false
}

// SaveAliases saves aliases to a file
func (am *AliasManager) SaveAliases(filename string) error {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	// TODO: Implement alias persistence
	return nil
}

// LoadAliases loads aliases from a file
func (am *AliasManager) LoadAliases(filename string) error {
	// TODO: Implement alias persistence
	return nil
}