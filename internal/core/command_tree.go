package core

import (
	"fmt"
	"sort"
	"strings"
)

// CommandHandler is a function that handles a command
type CommandHandler func(c interface{}, cmd *ParsedCommand) error

// CommandNode represents a node in the command tree
type CommandNode struct {
	Name        string                    // Command name
	Description string                    // Command description
	Handler     CommandHandler            // Handler function (nil for category nodes)
	Children    map[string]*CommandNode   // Child commands
	Flags       []CommandFlag             // Supported flags
	Args        []CommandArg              // Positional arguments
	Examples    []string                  // Usage examples
	Hidden      bool                      // Hide from help listing
	
	// TAB completion optimization fields
	IsDirectory    bool     // True if this is a navigable directory
	CompletionData []string // Pre-computed completions for this node
}

// CommandFlag represents a command flag
type CommandFlag struct {
	Name        string   // Long name (e.g., "output")
	Short       string   // Short name (e.g., "o")
	Description string   // Description
	Type        string   // Type: string, bool, int, duration
	Default     string   // Default value
	Required    bool     // Is required?
	Choices     []string // Valid choices (for enum types)
}

// CommandArg represents a positional argument
type CommandArg struct {
	Name        string // Argument name
	Description string // Description
	Required    bool   // Is required?
	Multiple    bool   // Can accept multiple values?
}

// NewCommandNode creates a new command node
func NewCommandNode(name, description string) *CommandNode {
	return &CommandNode{
		Name:        name,
		Description: description,
		Children:    make(map[string]*CommandNode),
		Flags:       []CommandFlag{},
		Args:        []CommandArg{},
		Examples:    []string{},
	}
}

// AddChild adds a child command
func (n *CommandNode) AddChild(child *CommandNode) {
	n.Children[child.Name] = child
}

// AddFlag adds a flag to the command
func (n *CommandNode) AddFlag(flag CommandFlag) {
	n.Flags = append(n.Flags, flag)
}

// AddArg adds a positional argument
func (n *CommandNode) AddArg(arg CommandArg) {
	n.Args = append(n.Args, arg)
}

// AddExample adds a usage example
func (n *CommandNode) AddExample(example string) {
	n.Examples = append(n.Examples, example)
}

// FindCommand finds a command node by path
func (n *CommandNode) FindCommand(path []string) (*CommandNode, error) {
	if len(path) == 0 {
		return n, nil
	}

	childName := path[0]
	child, exists := n.Children[childName]
	if !exists {
		// Try case-insensitive match
		for name, node := range n.Children {
			if strings.EqualFold(name, childName) {
				child = node
				exists = true
				break
			}
		}
		
		if !exists {
			return nil, fmt.Errorf("unknown command: %s", childName)
		}
	}

	if len(path) == 1 {
		return child, nil
	}

	return child.FindCommand(path[1:])
}

// GetHelp generates help text for the command
func (n *CommandNode) GetHelp(fullPath string) string {
	var help strings.Builder

	// Command name and description
	help.WriteString(fmt.Sprintf("\n%s - %s\n", fullPath, n.Description))

	// Usage
	help.WriteString("\nUSAGE:\n")
	if n.Handler != nil {
		help.WriteString(fmt.Sprintf("  %s", fullPath))
		
		// Add required flags
		for _, flag := range n.Flags {
			if flag.Required {
				if flag.Short != "" {
					help.WriteString(fmt.Sprintf(" -%s", flag.Short))
				} else {
					help.WriteString(fmt.Sprintf(" --%s", flag.Name))
				}
				if flag.Type != "bool" {
					help.WriteString(fmt.Sprintf(" <%s>", flag.Type))
				}
			}
		}
		
		// Add positional arguments
		for _, arg := range n.Args {
			if arg.Required {
				help.WriteString(fmt.Sprintf(" <%s>", arg.Name))
			} else {
				help.WriteString(fmt.Sprintf(" [%s]", arg.Name))
			}
			if arg.Multiple {
				help.WriteString("...")
			}
		}
		
		help.WriteString(" [OPTIONS]\n")
	} else {
		// Category node
		help.WriteString(fmt.Sprintf("  %s <SUBCOMMAND> [OPTIONS]\n", fullPath))
	}

	// Subcommands
	if len(n.Children) > 0 {
		help.WriteString("\nSUBCOMMANDS:\n")
		
		// Sort children by name
		var names []string
		for name := range n.Children {
			if !n.Children[name].Hidden {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		
		// Find max name length for alignment
		maxLen := 0
		for _, name := range names {
			if len(name) > maxLen {
				maxLen = len(name)
			}
		}
		
		for _, name := range names {
			child := n.Children[name]
			help.WriteString(fmt.Sprintf("  %-*s  %s\n", maxLen+2, name, child.Description))
		}
	}

	// Arguments
	if len(n.Args) > 0 {
		help.WriteString("\nARGUMENTS:\n")
		for _, arg := range n.Args {
			req := ""
			if arg.Required {
				req = " (required)"
			}
			help.WriteString(fmt.Sprintf("  %-15s  %s%s\n", arg.Name, arg.Description, req))
		}
	}

	// Options/Flags
	if len(n.Flags) > 0 {
		help.WriteString("\nOPTIONS:\n")
		for _, flag := range n.Flags {
			// Format flag line
			flagStr := ""
			if flag.Short != "" {
				flagStr = fmt.Sprintf("-%s, ", flag.Short)
			}
			flagStr += fmt.Sprintf("--%s", flag.Name)
			
			if flag.Type != "bool" {
				flagStr += fmt.Sprintf(" <%s>", flag.Type)
			}
			
			desc := flag.Description
			if flag.Default != "" {
				desc += fmt.Sprintf(" (default: %s)", flag.Default)
			}
			if flag.Required {
				desc += " (required)"
			}
			if len(flag.Choices) > 0 {
				desc += fmt.Sprintf(" [choices: %s]", strings.Join(flag.Choices, ", "))
			}
			
			help.WriteString(fmt.Sprintf("  %-25s  %s\n", flagStr, desc))
		}
	}

	// Examples
	if len(n.Examples) > 0 {
		help.WriteString("\nEXAMPLES:\n")
		for _, example := range n.Examples {
			help.WriteString(fmt.Sprintf("  %s\n", example))
		}
	}

	return help.String()
}

// ValidateCommand validates a parsed command against this node
func (n *CommandNode) ValidateCommand(cmd *ParsedCommand) error {
	// Check required flags
	for _, flag := range n.Flags {
		if flag.Required {
			found := false
			if _, ok := cmd.Flags[flag.Name]; ok {
				found = true
			}
			if flag.Short != "" {
				if _, ok := cmd.Flags[flag.Short]; ok {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("required flag --%s is missing", flag.Name)
			}
		}
	}

	// Check required arguments
	requiredArgs := 0
	for _, arg := range n.Args {
		if arg.Required {
			requiredArgs++
		}
	}
	
	if len(cmd.Args) < requiredArgs {
		return fmt.Errorf("expected at least %d arguments, got %d", requiredArgs, len(cmd.Args))
	}

	// Validate flag values
	for flagName, value := range cmd.Flags {
		var flag *CommandFlag
		for i := range n.Flags {
			if n.Flags[i].Name == flagName || n.Flags[i].Short == flagName {
				flag = &n.Flags[i]
				break
			}
		}
		
		if flag == nil {
			return fmt.Errorf("unknown flag: %s", flagName)
		}
		
		// Check choices
		if len(flag.Choices) > 0 {
			valid := false
			for _, choice := range flag.Choices {
				if value == choice {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid value for --%s: %s (valid choices: %s)", 
					flag.Name, value, strings.Join(flag.Choices, ", "))
			}
		}
	}

	return nil
}