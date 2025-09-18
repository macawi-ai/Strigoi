package main

import (
	"fmt"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

// Command tree structure for navigation.
type CommandNode struct {
	Name        string
	Description string
	IsDirectory bool
	Children    map[string]*CommandNode
	Parent      *CommandNode
}

var (
	// Build command tree.
	commandTree = buildCommandTree()
	currentNode = commandTree
)

func buildCommandTree() *CommandNode {
	root := &CommandNode{
		Name:        "/",
		Description: "Strigoi root",
		IsDirectory: true,
		Children:    make(map[string]*CommandNode),
	}

	// Add probe directory
	probe := &CommandNode{
		Name:        "probe",
		Description: "Discovery and reconnaissance tools",
		IsDirectory: true,
		Children:    make(map[string]*CommandNode),
		Parent:      root,
	}
	root.Children["probe"] = probe

	// Add probe subcommands
	probeCommands := map[string]string{
		"north": "Probe north direction (endpoints)",
		"south": "Probe south direction (dependencies)",
		"east":  "Probe east direction (data flows)",
		"west":  "Probe west direction (integrations)",
		"all":   "Probe all directions",
	}
	for name, desc := range probeCommands {
		probe.Children[name] = &CommandNode{
			Name:        name,
			Description: desc,
			IsDirectory: false,
			Parent:      probe,
		}
	}

	// Stream functionality has been externalized to separate tools

	// Add utility commands at root
	root.Children["help"] = &CommandNode{
		Name:        "help",
		Description: "Show help information",
		IsDirectory: false,
		Parent:      root,
	}

	return root
}

func getPrompt() string {
	path := getPath(currentNode)
	if path == "/" {
		path = ""
	}
	return fmt.Sprintf("strigoi%s> ", path)
}

func getPath(node *CommandNode) string {
	if node == commandTree {
		return "/"
	}

	parts := []string{}
	current := node
	for current != nil && current != commandTree {
		parts = append([]string{current.Name}, parts...)
		current = current.Parent
	}
	return "/" + strings.Join(parts, "/")
}

func startInteractiveMode() error {
	fmt.Println(getBanner())
	fmt.Println("Entering interactive mode. Type 'help' for commands, 'exit' to quit.")

	completer := buildCompleter()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:              getPrompt(),
		HistoryFile:         "/tmp/strigoi-history",
		AutoComplete:        completer,
		InterruptPrompt:     "^C",
		EOFPrompt:           "exit",
		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF or interrupt
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse command
		args := strings.Fields(line)
		cmd := args[0]

		// Special handling for "probe south --help" type commands in interactive mode
		if currentNode == commandTree && len(args) >= 3 {
			// Check if user is trying to use full path in root context
			if dirNode, ok := commandTree.Children[cmd]; ok && dirNode.IsDirectory {
				if subCmd, ok := dirNode.Children[args[1]]; ok && !subCmd.IsDirectory {
					// User typed something like "probe south --help" at root
					if len(args) > 2 && (args[2] == "--help" || args[2] == "-h") {
						fmt.Printf("%s You can navigate to '%s' first, or use the command directly:\n",
							infoColor.Sprint("💡"), cmd)
						fmt.Printf("   Option 1: %s then %s\n",
							cmdColor.Sprintf("cd %s", cmd),
							cmdColor.Sprintf("%s --help", args[1]))
						fmt.Printf("   Option 2: Execute directly: %s\n\n",
							cmdColor.Sprint(strings.Join(args, " ")))

						// Still execute the command to show help
						if err := executeFullPathCommand(args); err != nil {
							errorColor.Printf("Error: %v\n", err)
						}
						continue
					}
				}
			}
		}

		// Check if user typed context prefix while already in that context
		if currentNode != commandTree && cmd == currentNode.Name {
			// User typed "probe" while already in /probe
			if len(args) > 1 {
				fmt.Printf("%s You're already in '%s' context. ",
					warnColor.Sprint("→"), currentNode.Name)
				fmt.Printf("Did you mean: %s\n",
					cmdColor.Sprint(strings.Join(args[1:], " ")))

				// Try to execute without the redundant prefix
				if err := executeCommand(args[1:]); err != nil {
					errorColor.Printf("Error: %v\n", err)
				}
				continue
			}
		}

		// Handle built-in commands
		switch cmd {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return nil
		case "cd":
			handleCD(args)
			rl.SetPrompt(getPrompt())
			// Update completer for new context
			rl.Config.AutoComplete = buildCompleter()
		case "ls":
			handleLS(args)
		case "pwd":
			handlePWD()
		case "help", "?":
			handleHelp(args)
		case "clear":
			fmt.Print("\033[H\033[2J")
		default:
			// Try to execute as a command
			if err := executeCommand(args); err != nil {
				errorColor.Printf("Error: %v\n", err)
			}
		}
	}

	return nil
}

func handleCD(args []string) {
	if len(args) < 2 {
		// Go to root
		currentNode = commandTree
		return
	}

	target := args[1]

	// Handle special cases
	if target == "/" {
		currentNode = commandTree
		return
	}
	if target == ".." {
		if currentNode.Parent != nil {
			currentNode = currentNode.Parent
		}
		return
	}
	if target == "." {
		return
	}

	// Navigate to child
	if child, ok := currentNode.Children[target]; ok && child.IsDirectory {
		currentNode = child
	} else {
		errorColor.Printf("cd: %s: No such directory\n", target)
	}
}

func handleLS(args []string) {
	// Determine which node to list
	node := currentNode
	if len(args) > 1 {
		// TODO: Handle path argument
		node = currentNode
	}

	// Separate directories and commands
	var dirs, cmds []*CommandNode
	for _, child := range node.Children {
		if child.IsDirectory {
			dirs = append(dirs, child)
		} else {
			cmds = append(cmds, child)
		}
	}

	// Show directories first
	if len(dirs) > 0 {
		for _, dir := range dirs {
			dirColor.Printf("  %-20s", dir.Name+"/")
			fmt.Printf("  %s\n", dir.Description)
		}
	}

	// Show commands
	if len(cmds) > 0 {
		if len(dirs) > 0 {
			fmt.Println() // Separator
		}
		for _, cmd := range cmds {
			cmdColor.Printf("  %-20s", cmd.Name)
			fmt.Printf("  %s\n", cmd.Description)
		}
	}

	if len(dirs) == 0 && len(cmds) == 0 {
		fmt.Println("  (empty)")
	}
}

func handlePWD() {
	fmt.Println(getPath(currentNode))
}

func handleHelp(args []string) {
	// Update global context for help system
	globalContext.IsInteractive = true
	globalContext.CurrentPath = getPath(currentNode)
	globalContext.CurrentNode = currentNode

	if len(args) > 1 {
		// Show help for specific command
		cmdName := args[1]

		// Check if it's a command in current directory
		if node, ok := currentNode.Children[cmdName]; ok && !node.IsDirectory {
			// Build the full cobra command path and get help
			fullCmd := buildFullCommand(node)
			if cobraCmd := findCobraCommand(fullCmd); cobraCmd != nil {
				// Make sure we're showing command-specific help, not interactive context
				globalContext.IsInteractive = false // Temporarily disable to get command help
				if err := cobraCmd.Help(); err != nil {
					errorColor.Printf("Error displaying help: %v\n", err)
				}
				globalContext.IsInteractive = true // Re-enable
				return
			}
		}

		// Check for built-in commands
		switch cmdName {
		case "cd", "ls", "pwd", "clear", "exit", "quit":
			showBuiltinHelp(cmdName)
			return
		}

		fmt.Printf("%s Command not found: %s\n", errorColor.Sprint("✗"), cmdName)
		fmt.Printf("Type %s to see available commands\n", infoColor.Sprint("help"))
		return
	}

	// Show contextual interactive help
	showInteractiveHelp()
}

func showInteractiveHelp() {
	fmt.Printf("\n%s %s\n", infoColor.Sprint("Current context:"),
		cmdColor.Sprint(getPath(currentNode)))

	// Show available items in current directory
	var dirs, cmds []string
	for name, child := range currentNode.Children {
		if child.IsDirectory {
			dirs = append(dirs, name)
		} else {
			cmds = append(cmds, name)
		}
	}

	if len(dirs) > 0 || len(cmds) > 0 {
		fmt.Println(successColor.Sprint("\n📁 Available here:"))
		if len(dirs) > 0 {
			fmt.Print("  Directories: ")
			for i, d := range dirs {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(dirColor.Sprint(d + "/"))
			}
			fmt.Println()
		}
		if len(cmds) > 0 {
			fmt.Print("  Commands: ")
			for i, c := range cmds {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(cmdColor.Sprint(c))
			}
			fmt.Println()
		}
	}

	// Show navigation commands
	fmt.Println(utilColor.Sprint("\n🧭 Navigation:"))
	fmt.Println("  cd <dir>     Navigate to directory")
	fmt.Println("  cd ..        Go up one level")
	fmt.Println("  ls           List current directory")
	fmt.Println("  pwd          Show current path")

	// Show utility commands
	fmt.Println(utilColor.Sprint("\n🔧 Utilities:"))
	fmt.Println("  help         Show this help")
	fmt.Println("  help <cmd>   Show help for specific command")
	fmt.Println("  ?            Quick help (same as help)")
	fmt.Println("  clear        Clear the screen")
	fmt.Println("  exit         Exit Strigoi")

	// Show execution hint
	fmt.Println(infoColor.Sprint("\n💡 Tips:"))
	fmt.Println("  • Type command names directly to execute them")
	fmt.Println("  • Press TAB for auto-completion")
	fmt.Println("  • Utilize the --help option to any command for detailed info")
	fmt.Println("  • Utilize the --examples option to see usage examples")
}

func showBuiltinHelp(cmd string) {
	helps := map[string]string{
		"cd":    "Change directory within Strigoi's command hierarchy\n  Usage: cd <directory>\n  Examples:\n    cd probe     - Enter probe directory\n    cd ..        - Go up one level\n    cd /         - Go to root",
		"ls":    "List contents of current or specified directory\n  Usage: ls [directory]\n  Shows available commands and subdirectories",
		"pwd":   "Print working directory - shows your current location\n  Usage: pwd",
		"clear": "Clear the terminal screen\n  Usage: clear",
		"exit":  "Exit Strigoi interactive mode\n  Usage: exit or quit\n  You can also use Ctrl+C or Ctrl+D",
	}

	if help, ok := helps[cmd]; ok {
		fmt.Printf("\n%s %s\n\n%s\n", cmdColor.Sprint(cmd+":"),
			"Built-in command", help)
	}
}

// findCobraCommand finds the cobra command for a given path.
// This will be implemented as a function variable to avoid circular dependency.
var findCobraCommand func([]string) *cobra.Command

func executeCommand(args []string) error {
	cmd := args[0]

	// Update global context for error handling
	globalContext.IsInteractive = true
	globalContext.CurrentPath = getPath(currentNode)
	globalContext.CurrentNode = currentNode
	globalContext.LastCommand = strings.Join(args, " ")

	// Check if it's a command in current directory
	if node, ok := currentNode.Children[cmd]; ok && !node.IsDirectory {
		// Build full command path
		fullCmd := buildFullCommand(node)
		fullArgs := append(fullCmd, args[1:]...)

		// Execute through Cobra by calling it externally
		err := executeCobraCommand(fullArgs)
		globalContext.LastError = err
		globalContext.HasErrors = (err != nil)
		return err
	}

	// Enhanced error message with suggestions
	err := fmt.Errorf("command not found: %s", cmd)
	globalContext.LastError = err

	// Check if user might have meant a different command
	suggestions := findSuggestions(cmd, currentNode)
	if len(suggestions) > 0 {
		fmt.Printf("\n%s Command not found: %s\n", errorColor.Sprint("✗"), cmd)
		fmt.Printf("   Did you mean one of these?\n")
		for _, suggestion := range suggestions {
			fmt.Printf("   • %s\n", cmdColor.Sprint(suggestion))
		}
		fmt.Printf("\n   Type %s to see all available commands\n", infoColor.Sprint("help"))
		return nil // Don't show the raw error
	}

	return err
}

// findSuggestions finds similar commands based on the input.
func findSuggestions(input string, node *CommandNode) []string {
	var suggestions []string

	// Check for commands that start with the same letter
	for name, child := range node.Children {
		if !child.IsDirectory {
			if strings.HasPrefix(name, input[:min(len(input), 1)]) {
				suggestions = append(suggestions, name)
			}
		}
	}

	// If no suggestions yet, check for commands containing the input
	if len(suggestions) == 0 {
		for name, child := range node.Children {
			if !child.IsDirectory {
				if strings.Contains(name, input) {
					suggestions = append(suggestions, name)
				}
			}
		}
	}

	// Limit to 3 suggestions
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

// executeCobraCommand will be set by root.go to avoid circular dependency.
var executeCobraCommand func([]string) error

// executeFullPathCommand executes a command with full path from root.
func executeFullPathCommand(args []string) error {
	// Update global context
	globalContext.IsInteractive = true
	globalContext.CurrentPath = getPath(currentNode)
	globalContext.LastCommand = strings.Join(args, " ")

	// Execute through Cobra
	err := executeCobraCommand(args)
	globalContext.LastError = err
	globalContext.HasErrors = (err != nil)
	return err
}

func buildFullCommand(node *CommandNode) []string {
	parts := []string{}
	current := node

	// Traverse up to build the full command path
	for current != nil && current != commandTree {
		parts = append([]string{current.Name}, parts...)
		current = current.Parent
	}

	return parts
}

func buildCompleter() *readline.PrefixCompleter {
	// Build dynamic completer based on current context
	items := []readline.PrefixCompleterInterface{
		readline.PcItem("cd",
			readline.PcItemDynamic(func(string) []string {
				var dirs []string
				for name, child := range currentNode.Children {
					if child.IsDirectory {
						dirs = append(dirs, name)
					}
				}
				dirs = append(dirs, "..", ".", "/")
				return dirs
			}),
		),
		readline.PcItem("ls"),
		readline.PcItem("pwd"),
		readline.PcItem("help"),
		readline.PcItem("?"),
		readline.PcItem("clear"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
	}

	// Add current directory commands dynamically
	for name, child := range currentNode.Children {
		if !child.IsDirectory {
			// For commands with subcommands, add dynamic completion
			if name == "north" || name == "south" || name == "east" || name == "west" {
				items = append(items, readline.PcItem(name,
					readline.PcItem("localhost"),
					readline.PcItem("api.example.com"),
					readline.PcItem("https://target.com"),
				))
			} else {
				items = append(items, readline.PcItem(name))
			}
		}
	}

	return readline.NewPrefixCompleter(items...)
}

func filterInput(r rune) (rune, bool) {
	switch r {
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}
