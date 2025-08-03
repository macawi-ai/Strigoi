package core

import (
	"fmt"
	"strings"
	"unicode"
)

// CommandParser handles parsing of hierarchical commands with arguments
type CommandParser struct {
	// Nothing needed for now, but allows for future configuration
}

// NewCommandParser creates a new command parser
func NewCommandParser() *CommandParser {
	return &CommandParser{}
}

// ParsedCommand represents a parsed command with its components
type ParsedCommand struct {
	Path      []string          // Full command path (e.g., ["stream", "tap"])
	Args      []string          // Positional arguments
	Flags     map[string]string // Named flags and their values
	RawInput  string            // Original input string
}

// Parse parses a command line into its components
func (p *CommandParser) Parse(input string) (*ParsedCommand, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty command")
	}

	result := &ParsedCommand{
		Path:     []string{},
		Args:     []string{},
		Flags:    make(map[string]string),
		RawInput: input,
	}

	// Tokenize the input while respecting quotes
	tokens, err := p.tokenize(input)
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("no command specified")
	}

	// First token is the command path
	commandToken := tokens[0]
	
	// Split command path by forward slashes
	if strings.Contains(commandToken, "/") {
		result.Path = strings.Split(commandToken, "/")
	} else {
		result.Path = []string{commandToken}
	}

	// Process remaining tokens as arguments and flags
	i := 1
	for i < len(tokens) {
		token := tokens[i]

		// Check if it's a flag
		if strings.HasPrefix(token, "--") {
			// Long flag
			flagName := strings.TrimPrefix(token, "--")
			
			// Check for = syntax (--flag=value)
			if idx := strings.Index(flagName, "="); idx != -1 {
				result.Flags[flagName[:idx]] = flagName[idx+1:]
			} else {
				// Next token is the value (unless it's another flag or end of tokens)
				if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
					result.Flags[flagName] = tokens[i+1]
					i++ // Skip the value token
				} else {
					// Boolean flag
					result.Flags[flagName] = "true"
				}
			}
		} else if strings.HasPrefix(token, "-") && len(token) > 1 {
			// Short flag(s)
			flags := strings.TrimPrefix(token, "-")
			
			// Handle combined short flags (e.g., -abc)
			for j, flag := range flags {
				flagStr := string(flag)
				
				// Last flag in a combination might have a value
				if j == len(flags)-1 && i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
					result.Flags[flagStr] = tokens[i+1]
					i++ // Skip the value token
				} else {
					// Boolean flag
					result.Flags[flagStr] = "true"
				}
			}
		} else {
			// Positional argument
			result.Args = append(result.Args, token)
		}
		
		i++
	}

	return result, nil
}

// tokenize splits input into tokens while respecting quotes
func (p *CommandParser) tokenize(input string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	var inQuote rune
	var escape bool

	runes := []rune(input)
	
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if escape {
			// Handle escaped characters
			switch r {
			case 'n':
				current.WriteRune('\n')
			case 't':
				current.WriteRune('\t')
			case 'r':
				current.WriteRune('\r')
			case '\\', '"', '\'':
				current.WriteRune(r)
			default:
				// Unknown escape sequence, write as-is
				current.WriteRune('\\')
				current.WriteRune(r)
			}
			escape = false
			continue
		}

		if r == '\\' {
			escape = true
			continue
		}

		// Handle quotes
		if inQuote != 0 {
			if r == inQuote {
				// End quote
				inQuote = 0
			} else {
				current.WriteRune(r)
			}
		} else {
			// Not in quote
			if r == '"' || r == '\'' {
				// Start quote
				inQuote = r
			} else if unicode.IsSpace(r) {
				// End of token
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(r)
			}
		}
	}

	// Check for unclosed quotes
	if inQuote != 0 {
		return nil, fmt.Errorf("unclosed quote: %c", inQuote)
	}

	// Add final token
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}

// FormatCommand formats a parsed command back to a string (useful for logging)
func (p *CommandParser) FormatCommand(cmd *ParsedCommand) string {
	var parts []string
	
	// Add command path
	parts = append(parts, strings.Join(cmd.Path, "/"))
	
	// Add positional arguments
	for _, arg := range cmd.Args {
		if strings.Contains(arg, " ") || strings.Contains(arg, "\"") {
			// Quote arguments containing spaces or quotes
			parts = append(parts, fmt.Sprintf("%q", arg))
		} else {
			parts = append(parts, arg)
		}
	}
	
	// Add flags
	for flag, value := range cmd.Flags {
		if len(flag) == 1 {
			// Short flag
			if value == "true" {
				parts = append(parts, fmt.Sprintf("-%s", flag))
			} else {
				parts = append(parts, fmt.Sprintf("-%s", flag), value)
			}
		} else {
			// Long flag
			if value == "true" {
				parts = append(parts, fmt.Sprintf("--%s", flag))
			} else {
				parts = append(parts, fmt.Sprintf("--%s", flag), value)
			}
		}
	}
	
	return strings.Join(parts, " ")
}