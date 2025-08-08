# Strigoi Help System Improvements

## Overview

Based on user testing feedback, we've completely redesigned Strigoi's help system to be more intuitive, context-aware, and helpful. The new system addresses the confusion users experienced with commands like `probe south --help` in interactive mode.

## Key Improvements

### 1. Context-Aware Help

The help system now understands whether you're in:
- **Command-line mode**: Running `strigoi probe south --help`
- **Interactive mode**: Navigating the command hierarchy with `cd` and executing commands

When in interactive mode at `/probe>`, the system recognizes that typing `probe south --help` is redundant and provides gentle guidance.

### 2. Smart Error Recovery

When users type incorrect commands, the system now:
- Suggests similar commands based on partial matches
- Provides clear guidance on correct syntax
- Shows contextual help based on the current directory

Example:
```bash
strigoi/probe> sout
✗ Command not found: sout
   Did you mean one of these?
   • south
   
   Type help to see all available commands
```

### 3. Progressive Disclosure

Multiple help levels provide the right amount of information:

```bash
# Brief help (one-liner)
south -h

# Standard help (default)
south --help

# Full help with advanced options
south --help-full

# Just examples
south --examples
```

### 4. Rich Interactive Mode Help

The interactive `help` command now shows:
- Current context/directory
- Available commands and subdirectories
- Navigation commands
- Utility commands
- Contextual tips

### 5. Built-in Command Documentation

Interactive navigation commands now have proper help:
```bash
strigoi> help cd
cd: Built-in command

Change directory within Strigoi's command hierarchy
  Usage: cd <directory>
  Examples:
    cd probe     - Enter probe directory
    cd ..        - Go up one level
    cd /         - Go to root
```

## Implementation Details

### New Files
- `cmd/strigoi/help.go` - Enhanced help system with multiple modes and context awareness

### Modified Files
- `cmd/strigoi/interactive.go` - Improved error handling and command suggestions
- `cmd/strigoi/root.go` - Integration of the new help system

### Key Features

1. **HelpMode Enumeration**: Brief, Standard, Full, Examples
2. **InteractiveContext Tracking**: Maintains state for context-aware help
3. **Smart Command Suggestions**: Levenshtein-like distance for similar commands
4. **Categorized Command Display**: Groups commands by type (Directions, Monitoring, Utilities)
5. **Dynamic Help Based on State**: Different help when scanning, errors occurred, etc.

## Usage Examples

### Interactive Mode - Handling Confusion
```bash
strigoi> probe south --help
💡 You can navigate to 'probe' first, or use the command directly:
   Option 1: cd probe then south --help
   Option 2: Execute directly: probe south --help
```

### Interactive Mode - Context Help
```bash
strigoi/probe> help
Current context: /probe

📁 Available here:
  Commands: north, south, east, west, all

🧭 Navigation:
  cd <dir>     Navigate to directory
  cd ..        Go up one level
  ls           List current directory
  pwd          Show current path

💡 Tips:
  • Type command names directly to execute them
  • Use TAB for auto-completion
  • Add --help to any command for detailed info
```

### Command-Line Mode
```bash
$ strigoi probe south --help
Analyze dependencies, libraries, and supply chain vulnerabilities.

Usage:
  strigoi probe south [target] [flags]

Flags:
  --scan-mcp        Enable MCP tools scanning
  --include-self    Include Strigoi's own files
  ...

Quick Examples:
  strigoi probe south --scan-mcp
  strigoi probe south --output json > deps.json

💡 Hint: Add --scan-mcp to detect MCP server vulnerabilities
```

## Benefits

1. **Reduced User Confusion**: Clear guidance when commands are mistyped
2. **Faster Learning Curve**: Progressive disclosure and examples
3. **Better Error Recovery**: Smart suggestions help users find the right command
4. **Context Awareness**: Help adapts to where you are and what you're doing
5. **Professional UX**: Modern CLI experience comparable to tools like Git, Docker, kubectl

## Future Enhancements

- [ ] Add command history suggestions based on frequency
- [ ] Implement fuzzy search for command discovery
- [ ] Add interactive tutorials for new users
- [ ] Support for custom help templates per command
- [ ] Integration with man pages generation

## Testing

Run the demo script to see all improvements:
```bash
./demo/demo_help_system.sh
```

## Credits

Designed in collaboration with Sister Gemini, incorporating best practices from:
- Git's comprehensive help system
- Docker's clear command structure
- kubectl's context-aware assistance
- Modern CLI design principles