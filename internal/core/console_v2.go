package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

// ConsoleV2 is an improved console with better command parsing
type ConsoleV2 struct {
	framework   *Framework
	rl          *readline.Instance
	writer      io.Writer
	
	// Command system
	parser       *CommandParser
	rootCommand  *CommandNode
	navigator    *ContextNavigator
	aliasManager *AliasManager
	fuzzyMatcher *FuzzyMatcher
	
	// Color scheme
	promptColor  *color.Color
	errorColor   *color.Color
	successColor *color.Color
	infoColor    *color.Color
	warnColor    *color.Color
	
	// Enhanced color scheme for better visual distinction
	dirColor     *color.Color  // Directories (blue with bold)
	cmdColor     *color.Color  // Executable commands (green)
	utilColor    *color.Color  // Utility commands (white/default)
	aliasColor   *color.Color  // Aliases (cyan)
}

// NewConsoleV2 creates a new improved console
func NewConsoleV2(framework *Framework) *ConsoleV2 {
	c := &ConsoleV2{
		framework:    framework,
		writer:       os.Stdout,
		parser:       NewCommandParser(),
		aliasManager: NewAliasManager(),
		promptColor:  color.New(color.FgCyan, color.Bold),
		errorColor:   color.New(color.FgRed, color.Bold),
		successColor: color.New(color.FgGreen, color.Bold),
		infoColor:    color.New(color.FgBlue),
		warnColor:    color.New(color.FgYellow),
		
		// Enhanced colors for visual distinction (following Gemini's recommendations)
		dirColor:     color.New(color.FgBlue, color.Bold),     // Directories in blue with bold
		cmdColor:     color.New(color.FgGreen),               // Commands in green (no bold to differentiate from success messages)
		utilColor:    color.New(color.FgHiWhite),             // Utilities in bright white
		aliasColor:   color.New(color.FgCyan),                // Aliases in cyan (no bold to differentiate from prompt)
	}
	
	// Build command tree with improved navigation model
	c.rebuildCommandTreeForClarity()
	
	// Initialize navigation and completion
	c.navigator = NewContextNavigator(c.rootCommand)
	c.fuzzyMatcher = NewFuzzyMatcher(c.rootCommand)
	
	// Set default aliases
	c.setupDefaultAliases()
	
	return c
}

// buildCommandTree constructs the hierarchical command structure
func (c *ConsoleV2) buildCommandTree() {
	// Root node
	c.rootCommand = NewCommandNode("strigoi", "Strigoi security validation platform")
	
	// Help command
	help := NewCommandNode("help", "Show help information")
	help.Handler = c.handleHelp
	help.AddArg(CommandArg{
		Name:        "command",
		Description: "Command path to get help for",
		Required:    false,
		Multiple:    true,
	})
	c.rootCommand.AddChild(help)
	
	// Exit command
	exit := NewCommandNode("exit", "Exit the console")
	exit.Handler = c.handleExit
	c.rootCommand.AddChild(exit)
	
	// Clear command
	clear := NewCommandNode("clear", "Clear the screen")
	clear.Handler = c.handleClear
	c.rootCommand.AddChild(clear)
	
	// Alias command
	alias := NewCommandNode("alias", "Manage command aliases")
	alias.Handler = c.handleAlias
	alias.AddArg(CommandArg{
		Name:        "alias",
		Description: "Alias name",
		Required:    false,
	})
	alias.AddArg(CommandArg{
		Name:        "command",
		Description: "Command to alias",
		Required:    false,
		Multiple:    true,
	})
	alias.AddFlag(CommandFlag{
		Name:        "description",
		Short:       "d",
		Description: "Alias description",
		Type:        "string",
	})
	alias.AddExample("alias")
	alias.AddExample("alias ls list")
	alias.AddExample("alias tap stream/tap --auto-discover")
	c.rootCommand.AddChild(alias)
	
	// Build stream commands
	c.buildStreamCommands()
	
	// Build integration commands
	c.buildIntegrationCommands()
	
	// Build probe commands
	c.buildProbeCommands()
	
	// Build sense commands
	c.buildSenseCommands()
}

// buildStreamCommands builds the stream command subtree
func (c *ConsoleV2) buildStreamCommands() {
	stream := NewCommandNode("stream", "STDIO stream monitoring & analysis")
	
	// stream/tap command
	tap := NewCommandNode("tap", "Monitor process STDIO in real-time")
	tap.Handler = c.handleStreamTap
	tap.AddFlag(CommandFlag{
		Name:        "auto-discover",
		Short:       "a",
		Description: "Automatically discover Claude/MCP processes",
		Type:        "bool",
		Default:     "false",
	})
	tap.AddFlag(CommandFlag{
		Name:        "pid",
		Short:       "p",
		Description: "Process ID to monitor",
		Type:        "int",
	})
	tap.AddFlag(CommandFlag{
		Name:        "duration",
		Short:       "d",
		Description: "Monitoring duration (e.g., 30s, 5m)",
		Type:        "duration",
		Default:     "30s",
	})
	tap.AddFlag(CommandFlag{
		Name:        "output",
		Short:       "o",
		Description: "Output destination (e.g., file:/tmp/capture.jsonl, tcp:host:port)",
		Type:        "string",
		Default:     "stdout",
	})
	tap.AddFlag(CommandFlag{
		Name:        "filter",
		Short:       "f",
		Description: "BPF-style filter expression",
		Type:        "string",
	})
	tap.AddExample("stream/tap --auto-discover --duration 1m")
	tap.AddExample("stream/tap --pid 12345 --output file:/tmp/capture.jsonl")
	tap.AddExample("stream/tap -a -d 5m --filter 'contains(\"password\")'")
	stream.AddChild(tap)
	
	// stream/record command
	record := NewCommandNode("record", "Record streams for later analysis")
	record.Handler = c.handleStreamRecord
	record.AddFlag(CommandFlag{
		Name:        "name",
		Short:       "n",
		Description: "Recording name",
		Type:        "string",
		Required:    true,
	})
	stream.AddChild(record)
	
	// stream/replay command
	replay := NewCommandNode("replay", "Replay recorded streams")
	replay.Handler = c.handleStreamReplay
	stream.AddChild(replay)
	
	// stream/analyze command
	analyze := NewCommandNode("analyze", "Analyze captured streams")
	analyze.Handler = c.handleStreamAnalyze
	stream.AddChild(analyze)
	
	// stream/patterns command
	patterns := NewCommandNode("patterns", "Manage security patterns")
	patterns.Handler = c.handleStreamPatterns
	stream.AddChild(patterns)
	
	// stream/status command
	status := NewCommandNode("status", "Show stream monitoring status")
	status.Handler = c.handleStreamStatus
	stream.AddChild(status)
	
	c.rootCommand.AddChild(stream)
}

// buildIntegrationCommands builds the integration command subtree
func (c *ConsoleV2) buildIntegrationCommands() {
	integrations := NewCommandNode("integrations", "External system integrations")
	
	// integrations/list
	list := NewCommandNode("list", "List available integrations")
	list.Handler = c.handleIntegrationsList
	integrations.AddChild(list)
	
	// integrations/prometheus
	prometheus := NewCommandNode("prometheus", "Prometheus metrics integration")
	
	promEnable := NewCommandNode("enable", "Enable Prometheus metrics export")
	promEnable.Handler = c.handlePrometheusEnable
	promEnable.AddFlag(CommandFlag{
		Name:        "port",
		Short:       "p",
		Description: "HTTP port for metrics endpoint",
		Type:        "int",
		Default:     "9090",
	})
	prometheus.AddChild(promEnable)
	
	promDisable := NewCommandNode("disable", "Disable Prometheus metrics")
	promDisable.Handler = c.handlePrometheusDisable
	prometheus.AddChild(promDisable)
	
	integrations.AddChild(prometheus)
	
	// integrations/syslog
	syslog := NewCommandNode("syslog", "Syslog integration")
	integrations.AddChild(syslog)
	
	c.rootCommand.AddChild(integrations)
}

// buildProbeCommands builds the probe command subtree
func (c *ConsoleV2) buildProbeCommands() {
	probe := NewCommandNode("probe", "Discovery and reconnaissance")
	
	// Compass directions
	for _, dir := range []string{"north", "south", "east", "west", "center"} {
		node := NewCommandNode(dir, fmt.Sprintf("Probe %s direction", dir))
		node.Handler = c.handleProbeDirection
		probe.AddChild(node)
	}
	
	// probe/all
	all := NewCommandNode("all", "Probe all directions")
	all.Handler = c.handleProbeAll
	probe.AddChild(all)
	
	c.rootCommand.AddChild(probe)
}

// buildSenseCommands builds the sense command subtree
func (c *ConsoleV2) buildSenseCommands() {
	sense := NewCommandNode("sense", "Analysis and interpretation")
	
	// OSI layers
	layers := []string{"network", "transport", "protocol", "application", "data", "trust", "human"}
	for _, layer := range layers {
		node := NewCommandNode(layer, fmt.Sprintf("Analyze %s layer", layer))
		node.Handler = c.handleSenseLayer
		sense.AddChild(node)
	}
	
	c.rootCommand.AddChild(sense)
}

// setupDefaultAliases configures default command aliases
func (c *ConsoleV2) setupDefaultAliases() {
	// Short aliases for common commands
	c.aliasManager.AddAlias("h", "help", "Show help")
	c.aliasManager.AddAlias("?", "help", "Show help")
	c.aliasManager.AddAlias("q", "exit", "Exit console")
	c.aliasManager.AddAlias("tap", "stream/tap", "Quick access to tap command")
	c.aliasManager.AddAlias("monitor", "stream/tap --auto-discover", "Monitor with auto-discovery")
	c.aliasManager.AddAlias("ll", "ls", "List commands (alias for ls)")
	
	// Navigation shortcuts
	c.aliasManager.AddAlias("~", "/", "Go to root directory")
}

// Start starts the improved console
func (c *ConsoleV2) Start() error {
	c.printBanner()
	
	// Get history file path
	paths := GetPaths()
	historyFile := filepath.Join(paths.Home, ".strigoi_history")
	
	// Configure readline
	prompt := c.buildPrompt()
	
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            prompt,
		HistoryFile:       historyFile,
		HistoryLimit:      1000,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
		AutoComplete:      BuildHybridCompleter(c.rootCommand, c.navigator, c.aliasManager),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize readline: %w", err)
	}
	defer rl.Close()
	
	c.rl = rl
	
	// Main command loop
	for {
		input, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}
			if err == io.EOF {
				c.Println("\nExiting...")
				return nil
			}
			return err
		}
		
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		
		// Process the command with improved navigation
		if err := c.processCommandV3(input); err != nil {
			if err.Error() == "exit" {
				return nil
			}
			c.Error(err.Error())
		}
		
		// Update prompt if context changed
		rl.SetPrompt(c.buildPrompt())
	}
}

// processCommandV2 processes a command using the new parser
func (c *ConsoleV2) processCommandV2(input string) error {
	// Expand aliases first
	input = c.aliasManager.ExpandAlias(input)
	
	// Parse the command
	cmd, err := c.parser.Parse(input)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}
	
	// Handle navigation commands
	if len(cmd.Path) == 1 && len(cmd.Args) == 0 && len(cmd.Flags) == 0 {
		switch cmd.Path[0] {
		case "..", "back":
			if err := c.navigator.NavigateUp(); err != nil {
				return err
			}
			c.Info("Moved to: %s", c.navigator.GetBreadcrumb())
			return nil
		case "/", "root":
			c.navigator.NavigateToRoot()
			c.Info("Moved to root context")
			return nil
		case "-":
			if err := c.navigator.NavigateBack(); err != nil {
				return err
			}
			c.Info("Moved to: %s", c.navigator.GetBreadcrumb())
			return nil
		}
	}
	
	// Resolve command path with context
	fullPath, isCommand := c.navigator.ResolveCommand(strings.Join(cmd.Path, "/"))
	if !isCommand {
		return nil // Navigation handled
	}
	
	// Find the command node
	node, err := c.rootCommand.FindCommand(fullPath)
	if err != nil {
		// Try fuzzy matching
		if suggestion, score := c.fuzzyMatcher.SuggestCommand(strings.Join(cmd.Path, "/")); score > 0.7 {
			c.Warn("Command not found. Did you mean '%s'?", suggestion)
		} else {
			// Check if it's a context change
			if len(cmd.Args) == 0 && len(cmd.Flags) == 0 {
				// Try to change context
				contextNode, contextErr := c.rootCommand.FindCommand(fullPath)
				if contextErr == nil && len(contextNode.Children) > 0 {
					c.navigator.NavigateTo(fullPath)
							c.Info("Entered %s context. Type 'help' for available commands.", strings.Join(fullPath, "/"))
					return nil
				}
			}
		}
		return err
	}
	
	// If no handler, it's a category node
	if node.Handler == nil {
		if len(node.Children) > 0 {
			// Show available subcommands
			c.Info("Available subcommands for %s:", strings.Join(fullPath, "/"))
			for name, child := range node.Children {
				if !child.Hidden {
					fmt.Fprintf(c.writer, "  %-20s %s\n", name, child.Description)
				}
			}
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


// buildPrompt builds the console prompt based on current context
func (c *ConsoleV2) buildPrompt() string {
	prompt := "strigoi"
	if breadcrumb := c.navigator.GetBreadcrumb(); breadcrumb != "/" {
		prompt += breadcrumb
	}
	prompt += " > "
	return c.promptColor.Sprint(prompt)
}

// handleAlias handles alias management commands
func (c *ConsoleV2) handleAlias(console interface{}, cmd *ParsedCommand) error {
	if len(cmd.Args) == 0 {
		// List aliases
		c.Info("Configured aliases:")
		aliases := c.aliasManager.ListAliases()
		for _, alias := range aliases {
			c.Printf("  %-15s => %s  # %s\n", alias.Alias, alias.Command, alias.Description)
		}
		return nil
	}
	
	if len(cmd.Args) == 1 {
		// Show specific alias
		if command, exists := c.aliasManager.GetAlias(cmd.Args[0]); exists {
			c.Info("%s => %s", cmd.Args[0], command)
		} else {
			c.Error("Alias not found: %s", cmd.Args[0])
		}
		return nil
	}
	
	if len(cmd.Args) >= 2 {
		// Create new alias
		alias := cmd.Args[0]
		command := strings.Join(cmd.Args[1:], " ")
		description := "User-defined alias"
		if desc, ok := cmd.Flags["description"]; ok {
			description = desc
		}
		
		if err := c.aliasManager.AddAlias(alias, command, description); err != nil {
			return err
		}
		
		c.Success("Alias created: %s => %s", alias, command)
		return nil
	}
	
	return fmt.Errorf("invalid alias command")
}

// createCompleterV2 creates an improved completer
func (c *ConsoleV2) createCompleterV2() *readline.PrefixCompleter {
	// This is a simplified version - in production, build from command tree
	return readline.NewPrefixCompleter(
		readline.PcItem("help"),
		readline.PcItem("exit"),
		readline.PcItem("clear"),
		readline.PcItem("stream/",
			readline.PcItem("tap"),
			readline.PcItem("record"),
			readline.PcItem("replay"),
			readline.PcItem("analyze"),
			readline.PcItem("patterns"),
			readline.PcItem("status"),
		),
		readline.PcItem("integrations/",
			readline.PcItem("list"),
			readline.PcItem("prometheus/",
				readline.PcItem("enable"),
				readline.PcItem("disable"),
			),
		),
		readline.PcItem("probe/",
			readline.PcItem("north"),
			readline.PcItem("south"),
			readline.PcItem("east"),
			readline.PcItem("west"),
			readline.PcItem("center"),
			readline.PcItem("all"),
		),
		readline.PcItem("sense/",
			readline.PcItem("network"),
			readline.PcItem("transport"),
			readline.PcItem("protocol"),
			readline.PcItem("application"),
			readline.PcItem("data"),
			readline.PcItem("trust"),
			readline.PcItem("human"),
		),
	)
}

// Output methods with formatting support
func (c *ConsoleV2) Println(a ...interface{}) { 
	fmt.Fprintln(c.writer, a...) 
}

func (c *ConsoleV2) Printf(format string, a ...interface{}) { 
	fmt.Fprintf(c.writer, format, a...) 
}

func (c *ConsoleV2) Error(format string, a ...interface{}) { 
	if len(a) > 0 {
		c.errorColor.Fprintf(c.writer, "[!] " + format + "\n", a...)
	} else {
		c.errorColor.Fprintln(c.writer, "[!] " + format)
	}
}

func (c *ConsoleV2) Success(format string, a ...interface{}) { 
	if len(a) > 0 {
		c.successColor.Fprintf(c.writer, "[+] " + format + "\n", a...)
	} else {
		c.successColor.Fprintln(c.writer, "[+] " + format)
	}
}

func (c *ConsoleV2) Info(format string, a ...interface{}) { 
	if len(a) > 0 {
		c.infoColor.Fprintf(c.writer, "[*] " + format + "\n", a...)
	} else {
		c.infoColor.Fprintln(c.writer, "[*] " + format)
	}
}

func (c *ConsoleV2) Warn(format string, a ...interface{}) { 
	if len(a) > 0 {
		c.warnColor.Fprintf(c.writer, "[!] " + format + "\n", a...)
	} else {
		c.warnColor.Fprintln(c.writer, "[!] " + format)
	}
}

// Command handlers
func (c *ConsoleV2) handleHelp(console interface{}, cmd *ParsedCommand) error {
	if len(cmd.Args) == 0 {
		// Show root help
		return c.showRootHelp()
	}
	
	// Show help for specific command
	path := strings.Split(cmd.Args[0], "/")
	node, err := c.rootCommand.FindCommand(path)
	if err != nil {
		return err
	}
	
	help := node.GetHelp(strings.Join(path, "/"))
	fmt.Fprint(c.writer, help)
	return nil
}

func (c *ConsoleV2) handleExit(console interface{}, cmd *ParsedCommand) error {
	return fmt.Errorf("exit")
}

func (c *ConsoleV2) handleClear(console interface{}, cmd *ParsedCommand) error {
	fmt.Print("\033[H\033[2J")
	return nil
}

func (c *ConsoleV2) showRootHelp() error {
	fmt.Fprintln(c.writer, "\nAvailable commands:")
	fmt.Fprintln(c.writer)
	
	// Sort commands
	var names []string
	for name := range c.rootCommand.Children {
		if !c.rootCommand.Children[name].Hidden {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	
	for _, name := range names {
		child := c.rootCommand.Children[name]
		displayName := name
		// Add trailing slash for directories
		if len(child.Children) > 0 || child.Handler == nil {
			displayName = name + "/"
		}
		fmt.Fprintf(c.writer, "  %-20s %s\n", displayName, child.Description)
	}
	
	fmt.Fprintln(c.writer, "\nNavigation:")
	fmt.Fprintln(c.writer, "  - Use 'cd <directory>' to navigate")
	fmt.Fprintln(c.writer, "  - Use 'pwd' to show current directory")
	fmt.Fprintln(c.writer, "  - Use 'ls' to list available commands and directories")
	fmt.Fprintln(c.writer, "  - Use 'cd ..' or just '..' to go back")
	fmt.Fprintln(c.writer, "  - Use 'cd /' or just '/' to go to root")
	fmt.Fprintln(c.writer, "  - Commands execute directly without navigation")
	fmt.Fprintln(c.writer, "  - Type 'help <command>' for detailed help")
	fmt.Fprintln(c.writer)
	
	return nil
}

// printBanner prints the welcome banner
func (c *ConsoleV2) printBanner() {
	PrintStrigoiBanner(c.writer, BannerStriGo)
	
	grayColor := color.New(color.FgHiBlack)
	grayColor.Fprintf(c.writer, "Advanced Security Validation Platform v0.4.0-community\n")
	grayColor.Fprintf(c.writer, "Copyright © 2025 Macawi - James R. Saker Jr.\n\n")
	
	c.warnColor.Fprintln(c.writer, "⚠️  Authorized use only - WHITE HAT SECURITY TESTING")
	fmt.Fprintln(c.writer)
	
	c.infoColor.Fprintln(c.writer, "♥  If Strigoi helps secure your systems, consider supporting:")
	fmt.Fprintln(c.writer, "   https://github.com/sponsors/macawi-ai")
	fmt.Fprintln(c.writer)
	
	c.successColor.Fprintln(c.writer, "Quick Start Guide:")
	fmt.Fprintln(c.writer, "  Run './strigoi' to enter interactive mode")
	fmt.Fprintln(c.writer, "  Type 'help' once inside to see available commands")
	fmt.Fprintln(c.writer)
}