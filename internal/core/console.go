package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

// Console provides a simplified interactive console interface
type Console struct {
	framework *Framework
	prompt    string
	rl        *readline.Instance
	writer    io.Writer
	history   []string
	mu        sync.Mutex
	
	// Context navigation
	context      []string // Current context path (e.g., ["probe", "north"])
	basePrompt   string   // Base prompt without context
	
	// Color scheme
	promptColor  *color.Color
	errorColor   *color.Color
	successColor *color.Color
	infoColor    *color.Color
	warnColor    *color.Color
}

// NewConsole creates a new console instance
func NewConsole(framework *Framework) *Console {
	return &Console{
		framework:    framework,
		prompt:       "strigoi > ",
		basePrompt:   "strigoi",
		context:      []string{},
		writer:       os.Stdout,
		history:      make([]string, 0),
		promptColor:  color.New(color.FgCyan, color.Bold),
		errorColor:   color.New(color.FgRed, color.Bold),
		successColor: color.New(color.FgGreen, color.Bold),
		infoColor:    color.New(color.FgBlue),
		warnColor:    color.New(color.FgYellow),
	}
}

// Start starts the interactive console
func (c *Console) Start() error {
	c.printBanner()
	
	// Get history file path
	paths := GetPaths()
	historyFile := filepath.Join(paths.Home, ".strigoi_history")
	
	// Configure readline with colored prompt
	coloredPrompt := c.promptColor.Sprint(c.prompt)
	
	// Create readline instance
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            coloredPrompt,
		HistoryFile:       historyFile,
		HistoryLimit:      1000,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
		AutoComplete:      c.createCompleter(),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize readline: %w", err)
	}
	defer rl.Close()
	
	c.rl = rl
	
	for {
		// Read input with history support
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
		
		// Clean input
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		
		// Process command based on context
		if len(c.context) > 0 {
			// We're in a context, handle context commands
			if err := c.processContextCommand(input); err != nil {
				if err.Error() == "exit" {
					return nil
				}
				c.Error(err.Error())
			}
		} else {
			// Main context
			if err := c.processCommand(input); err != nil {
				if err.Error() == "exit" {
					return nil
				}
				c.Error(err.Error())
			}
		}
	}
}

// processCommand processes a single command
func (c *Console) processCommand(input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	
	// Check if this is a slash command by looking for the first slash
	slashIndex := strings.Index(input, "/")
	if slashIndex > 0 && slashIndex < len(input)-1 {
		// This is a slash command - find where the command ends
		// We need to handle cases like "stream/tap --auto-discover"
		spaceAfterSlash := strings.Index(input[slashIndex:], " ")
		if spaceAfterSlash == -1 {
			// No arguments, just the slash command
			return c.processSlashCommand(input, []string{})
		}
		
		// Split into command and args
		commandEnd := slashIndex + spaceAfterSlash
		command := input[:commandEnd]
		remainingInput := strings.TrimSpace(input[commandEnd:])
		
		// Parse the remaining arguments
		args := strings.Fields(remainingInput)
		return c.processSlashCommand(command, args)
	}
	
	// Not a slash command, process normally
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}
	
	command := parts[0]
	args := parts[1:]
	
	switch command {
	case "help", "?":
		return c.showHelp()
	case "exit", "quit":
		return fmt.Errorf("exit")
	case "clear", "cls":
		return c.clearScreen()
	case "jobs":
		return c.showJobs(args)
	case "probe":
		// Enter probe context
		return c.enterProbeContext()
	case "sense":
		// Enter sense context
		return c.enterSenseContext()
	case "respond":
		c.Info("Respond context not yet implemented")
		return nil
	case "report":
		c.Info("Report context not yet implemented")
		return nil
	case "support":
		// Enter support context
		return c.enterSupportContext()
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// processSlashCommand handles hierarchical slash commands
func (c *Console) processSlashCommand(command string, args []string) error {
	parts := strings.Split(command, "/")
	if len(parts) < 1 {
		return fmt.Errorf("invalid command format")
	}
	
	primary := parts[0]
	subcommand := ""
	if len(parts) > 1 {
		// Join remaining parts to handle nested slashes (e.g., sense/network/local)
		subcommand = strings.Join(parts[1:], "/")
	}
	
	switch primary {
	case "probe":
		return c.processProbeCommand(subcommand, args)
	case "sense":
		return c.processSenseCommand(subcommand, args)
	case "respond":
		return c.processRespondCommand(subcommand, args)
	case "report":
		return c.processReportCommand(subcommand, args)
	case "support":
		return c.processSupportCommand(subcommand, args)
	case "state":
		return c.handleStateCommand(strings.Split(command, "/"))
	case "stream":
		return c.processStreamCommand(subcommand, args)
	case "integrations":
		return c.processIntegrationsCommand(subcommand, args)
	default:
		return fmt.Errorf("unknown command: %s", primary)
	}
}

// showHelp displays available commands
func (c *Console) showHelp() error {
	help := [][]string{
		{"help, ?", "Show this help menu"},
		{"probe", "Enter probe context for discovery"},
		{"sense", "Enter sense context for analysis"},
		{"stream", "🔍 STDIO stream monitoring & analysis"},
		{"integrations", "📊 External system integrations"},
		{"state", "🌟 Consciousness collaboration state management"},
		{"respond", "Enter respond context (future)"},
		{"report", "Enter report context"},
		{"support", "Enter support context (attribution, etc)"},
		{"jobs", "List running jobs"},
		{"clear, cls", "Clear the screen"},
		{"exit, quit", "Exit the console"},
	}
	
	fmt.Fprintln(c.writer, "\nAvailable commands:")
	fmt.Fprintln(c.writer)
	for _, cmd := range help {
		fmt.Fprintf(c.writer, "  %-20s %s\n", cmd[0], cmd[1])
	}
	fmt.Fprintln(c.writer)
	
	c.infoColor.Fprintln(c.writer, "Navigation:")
	fmt.Fprintln(c.writer, "  - Type a command to enter its context")
	fmt.Fprintln(c.writer, "  - Use 'back' or '..' to go back")
	fmt.Fprintln(c.writer, "  - Use '/' for direct paths (e.g., probe/north)")
	fmt.Fprintln(c.writer)
	
	return nil
}


// showJobs displays running jobs
func (c *Console) showJobs(args []string) error {
	jobs := c.framework.sessionMgr.GetJobs()
	
	if len(jobs) == 0 {
		c.Info("No active jobs")
		fmt.Fprintln(c.writer, "\nJobs run in the background for long-running operations.")
		fmt.Fprintln(c.writer, "Some modules create jobs when scanning multiple targets.")
		return nil
	}
	
	fmt.Fprintln(c.writer)
	fmt.Fprintf(c.writer, "  %-16s %-15s %-20s %-10s %s\n", "ID", "Type", "Module", "Status", "Progress")
	fmt.Fprintf(c.writer, "  %-16s %-15s %-20s %-10s %s\n", "--", "----", "------", "------", "--------")
	
	for _, job := range jobs {
		fmt.Fprintf(c.writer, "  %-16s %-15s %-20s %-10s %d%%\n",
			job.ID,
			job.Type,
			job.Module,
			job.Status,
			job.Progress)
	}
	return nil
}

// clearScreen clears the console screen
func (c *Console) clearScreen() error {
	// ANSI escape code to clear screen
	fmt.Print("\033[H\033[2J")
	return nil
}

// createCompleter creates the readline completer for tab completion
func (c *Console) createCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("help"),
		readline.PcItem("?"),
		readline.PcItem("probe/",
			readline.PcItem("info"),
			readline.PcItem("north"),
			readline.PcItem("east"),
			readline.PcItem("south"),
			readline.PcItem("west"),
			readline.PcItem("center"),
			readline.PcItem("quick"),
			readline.PcItem("all"),
		),
		readline.PcItem("sense/",
			readline.PcItem("network/"),
			readline.PcItem("transport/"),
			readline.PcItem("protocol/"),
			readline.PcItem("application/"),
			readline.PcItem("data/"),
			readline.PcItem("trust/"),
			readline.PcItem("human/"),
		),
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
			readline.PcItem("enable"),
			readline.PcItem("prometheus"),
			readline.PcItem("syslog"),
			readline.PcItem("file"),
		),
		readline.PcItem("respond/"),
		readline.PcItem("report/"),
		readline.PcItem("jobs"),
		readline.PcItem("clear"),
		readline.PcItem("cls"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
	)
}

// printBanner prints the Strigoi banner
func (c *Console) printBanner() {
	// Use the new STRI-GO banner with white STRI and blue GO!
	PrintStrigoiBanner(c.writer, BannerStriGo)
	
	// Framework info
	grayColor := color.New(color.FgHiBlack)
	grayColor.Fprintf(c.writer, "Advanced Security Validation Platform v0.4.0-community\n")
	grayColor.Fprintf(c.writer, "Copyright © 2025 Macawi - James R. Saker Jr.\n\n")
	
	// Warning message
	yellowColor := color.New(color.FgYellow)
	yellowColor.Fprintln(c.writer, "⚠️  Authorized use only - WHITE HAT SECURITY TESTING")
	
	// Support message
	cyanColor := color.New(color.FgCyan)
	cyanColor.Fprintln(c.writer, "\n♥  If Strigoi helps secure your systems, consider supporting:")
	c.infoColor.Fprintln(c.writer, "   https://github.com/sponsors/macawi-ai")
	
	// Quick start guide
	fmt.Fprintln(c.writer)
	c.successColor.Fprintln(c.writer, "Quick Start Guide:")
	fmt.Fprintln(c.writer, "  Type 'help' to see available commands")
	fmt.Fprintln(c.writer)
	
	// Module count
	c.infoColor.Fprintf(c.writer, "Modules loaded: %d\n", len(c.framework.modules))
	grayColor.Fprintln(c.writer, "Type 'help' for available commands")
	fmt.Fprintln(c.writer)
}

// Output methods
// Println prints values to the console
func (c *Console) Println(args ...interface{}) {
	fmt.Fprintln(c.writer, args...)
}

func (c *Console) Error(format string, args ...interface{}) {
	c.errorColor.Fprintf(c.writer, "[!] "+format+"\n", args...)
}

func (c *Console) Success(format string, args ...interface{}) {
	c.successColor.Fprintf(c.writer, "[+] "+format+"\n", args...)
}

func (c *Console) Info(format string, args ...interface{}) {
	c.infoColor.Fprintf(c.writer, "[*] "+format+"\n", args...)
}

func (c *Console) Warn(format string, args ...interface{}) {
	c.warnColor.Fprintf(c.writer, "[!] "+format+"\n", args...)
}

func (c *Console) Print(text string) {
	fmt.Fprint(c.writer, text)
}

func (c *Console) Printf(format string, args ...interface{}) {
	fmt.Fprintf(c.writer, format, args...)
}

// updatePrompt updates the prompt based on current context
func (c *Console) updatePrompt() {
	if len(c.context) == 0 {
		c.prompt = c.basePrompt + " > "
	} else {
		c.prompt = c.basePrompt + "/" + strings.Join(c.context, "/") + " > "
	}
	
	// Update readline prompt
	if c.rl != nil {
		c.rl.SetPrompt(c.promptColor.Sprint(c.prompt))
	}
}

// enterProbeContext enters the probe navigation context
func (c *Console) enterProbeContext() error {
	c.context = []string{"probe"}
	c.updatePrompt()
	
	// Show probe options
	c.Info("Entered probe context")
	fmt.Fprintln(c.writer)
	fmt.Fprintln(c.writer, "Available directions:")
	fmt.Fprintln(c.writer, "  north    - Probe LLM/AI platforms")
	fmt.Fprintln(c.writer, "  east     - Probe human interaction layers")
	fmt.Fprintln(c.writer, "  south    - Probe tool and data protocols")
	fmt.Fprintln(c.writer, "  west     - Probe VCP-MCP broker systems")
	fmt.Fprintln(c.writer, "  center   - Probe routing/orchestration layer")
	fmt.Fprintln(c.writer, "  quick    - Quick scan across all directions")
	fmt.Fprintln(c.writer, "  all      - Exhaustive enumeration")
	fmt.Fprintln(c.writer, "  info     - Explain the cardinal directions model")
	fmt.Fprintln(c.writer, "  back     - Return to main context")
	fmt.Fprintln(c.writer)
	
	return nil
}

// enterSenseContext enters the sense navigation context
func (c *Console) enterSenseContext() error {
	c.context = []string{"sense"}
	c.updatePrompt()
	
	// Show sense options
	c.Info("Entered sense context")
	fmt.Fprintln(c.writer)
	fmt.Fprintln(c.writer, "Available layers:")
	fmt.Fprintln(c.writer, "  network      - Network layer analysis")
	fmt.Fprintln(c.writer, "  transport    - Transport layer analysis")
	fmt.Fprintln(c.writer, "  protocol     - Protocol analysis (MCP, A2A)")
	fmt.Fprintln(c.writer, "  application  - Application layer analysis")
	fmt.Fprintln(c.writer, "  data         - Data flow and content analysis")
	fmt.Fprintln(c.writer, "  trust        - Trust and authentication analysis")
	fmt.Fprintln(c.writer, "  human        - Human interaction security")
	fmt.Fprintln(c.writer, "  back         - Return to main context")
	fmt.Fprintln(c.writer)
	
	return nil
}

// enterSupportContext enters the support navigation context
func (c *Console) enterSupportContext() error {
	c.context = []string{"support"}
	c.updatePrompt()
	
	// Show support options
	c.Info("Entered support context")
	fmt.Fprintln(c.writer)
	fmt.Fprintln(c.writer, "Available support actors:")
	fmt.Fprintln(c.writer, "  attribution  - Honor the thinkers who made this possible")
	fmt.Fprintln(c.writer, "  back         - Return to main context")
	fmt.Fprintln(c.writer)
	
	return nil
}

// processContextCommand handles commands within a context
func (c *Console) processContextCommand(input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}
	
	command := parts[0]
	args := parts[1:]
	
	// Handle back navigation
	if command == "back" || command == ".." {
		if len(c.context) > 0 {
			c.context = c.context[:len(c.context)-1]
			c.updatePrompt()
			if len(c.context) == 0 {
				c.Info("Returned to main context")
			} else {
				c.Info("Returned to %s context", strings.Join(c.context, "/"))
			}
		}
		return nil
	}
	
	// Handle exit from any context
	if command == "exit" || command == "quit" {
		return fmt.Errorf("exit")
	}
	
	// Handle help from any context
	if command == "help" || command == "?" {
		// Show context-specific help
		if len(c.context) > 0 {
			switch c.context[0] {
			case "probe":
				return c.showProbeHelp()
			case "sense":
				return c.showSenseHelp()
			case "support":
				return c.showSupportHelp()
			}
		}
		return c.showHelp()
	}
	
	// Handle context-specific commands
	if len(c.context) > 0 {
		switch c.context[0] {
		case "probe":
			return c.handleProbeContext(command, args)
		case "sense":
			return c.handleSenseContext(command, args)
		case "support":
			return c.handleSupportContext(command, args)
		}
	}
	
	return fmt.Errorf("unknown command in context: %s", command)
}

// handleProbeContext handles commands within probe context
func (c *Console) handleProbeContext(command string, args []string) error {
	switch command {
	case "info":
		return c.showProbeInfo()
	case "north", "east", "south", "west", "center":
		// Navigate deeper or execute actor
		c.context = append(c.context, command)
		c.updatePrompt()
		c.Info("Entering probe/%s", command)
		// This is where we'd show available actors or execute if it's an actor
		c.Warn("Actor execution not yet implemented")
		return nil
	case "quick":
		return c.probeQuick(args)
	case "all":
		return c.probeAll(args)
	default:
		return fmt.Errorf("unknown probe command: %s", command)
	}
}

// handleSenseContext handles commands within sense context
func (c *Console) handleSenseContext(command string, args []string) error {
	switch command {
	case "network", "transport", "protocol", "application", "data", "trust", "human":
		// Navigate deeper
		c.context = append(c.context, command)
		c.updatePrompt()
		c.Info("Entering sense/%s", command)
		// This is where we'd show available sub-options or actors
		c.Warn("Layer analysis not yet implemented")
		return nil
	default:
		return fmt.Errorf("unknown sense command: %s", command)
	}
}

// processSenseCommand is now implemented in console_sense.go

func (c *Console) processRespondCommand(subcommand string, args []string) error {
	c.Info("Respond command not yet implemented")
	return nil
}

func (c *Console) processReportCommand(subcommand string, args []string) error {
	c.Info("Report command not yet implemented")
	return nil
}

// handleSupportContext handles commands within support context
func (c *Console) handleSupportContext(command string, args []string) error {
	switch command {
	case "attribution":
		return c.showAttribution(args)
	default:
		return fmt.Errorf("unknown support command: %s", command)
	}
}

// showSupportHelp displays help for support commands
func (c *Console) showSupportHelp() error {
	c.Info("Support Context - Meta and Attribution")
	fmt.Fprintln(c.writer)
	fmt.Fprintln(c.writer, "  support/attribution         - Show full attributions")
	fmt.Fprintln(c.writer, "  support/attribution --brief - List thinkers and contributions")
	fmt.Fprintln(c.writer, "  support/attribution --random - Daily inspiration")
	fmt.Fprintln(c.writer)
	return nil
}

// processSupportCommand handles support/ subcommands
func (c *Console) processSupportCommand(subcommand string, args []string) error {
	if subcommand == "" {
		return c.enterSupportContext()
	}
	
	switch subcommand {
	case "attribution":
		return c.showAttribution(args)
	default:
		c.Error("Unknown support subcommand: %s", subcommand)
		return c.showSupportHelp()
	}
}

// showAttribution displays intellectual attributions
func (c *Console) showAttribution(args []string) error {
	// For now, show a simple version until we integrate the full actor
	fmt.Fprintln(c.writer)
	fmt.Fprintln(c.writer, "=== Standing on the Shoulders of Giants ===")
	fmt.Fprintln(c.writer)
	
	// Show brief list for now
	attributions := [][]string{
		{"Gregory Bateson", "Ecology of Mind"},
		{"Stafford Beer", "Viable System Model"},
		{"Clayton Christensen", "Disruptive Innovation"},
		{"Donna Haraway", "Cyborg Manifesto"},
		{"Bruno Latour", "Actor-Network Theory"},
		{"Humberto Maturana", "Autopoiesis"},
		{"Jean-Luc Nancy", "Being-With"},
		{"Jacques Rancière", "Radical Equality"},
		{"Wolfgang Schirmacher", "Homo Generator"},
		{"David Snowden", "Cynefin Framework"},
		{"Bill Washburn", "Commercial Internet eXchange"},
		{"Norbert Wiener", "Cybernetics"},
	}
	
	for _, attr := range attributions {
		fmt.Fprintf(c.writer, "• %s - %s\n", attr[0], attr[1])
	}
	
	fmt.Fprintln(c.writer)
	fmt.Fprintln(c.writer, "Their ideas live on in every actor, every connection,")
	fmt.Fprintln(c.writer, "every ecology we create together.")
	fmt.Fprintln(c.writer)
	
	return nil
}