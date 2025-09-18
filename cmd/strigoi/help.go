package main

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Color variables for help system (defined here to avoid circular dependency).
var (
	helpErrorColor   = color.New(color.FgRed, color.Bold)
	helpSuccessColor = color.New(color.FgGreen, color.Bold)
	helpInfoColor    = color.New(color.FgBlue)
	helpWarnColor    = color.New(color.FgYellow)
	helpGrayColor    = color.New(color.FgHiBlack)
	helpDirColor     = color.New(color.FgBlue, color.Bold)
	helpCmdColor     = color.New(color.FgGreen)
	helpUtilColor    = color.New(color.FgHiWhite)
)

// CommandNode structure to avoid circular dependency.
type HelpCommandNode struct {
	Name        string
	Description string
	IsDirectory bool
	Children    map[string]*HelpCommandNode
	Parent      *HelpCommandNode
}

// HelpMode represents the level of help detail.
type HelpMode int

const (
	HelpModeBrief HelpMode = iota
	HelpModeStandard
	HelpModeFull
	HelpModeExamples
)

// InteractiveContext tracks the current state in interactive mode.
type InteractiveContext struct {
	IsInteractive  bool
	CurrentPath    string
	CurrentNode    interface{} // Can be *CommandNode or *HelpCommandNode
	LastCommand    string
	LastError      error
	ScanningActive bool
	HasErrors      bool
}

var globalContext = &InteractiveContext{
	IsInteractive: false,
}

// convertToHelpNode converts CommandNode to HelpCommandNode for help display.
func convertToHelpNode(node interface{}) *HelpCommandNode {
	if node == nil {
		return nil
	}

	// Use type assertion to check if it's a CommandNode (from interactive.go)
	// Since we can't import CommandNode due to circular dependency, we'll use reflection
	// or just return nil for now - the help system will handle it gracefully
	return nil
}

// SetInteractiveContext updates the global interactive context.
func SetInteractiveContext(ctx *InteractiveContext) {
	globalContext = ctx
}

// GetHelpMode determines the help mode based on flags.
func GetHelpMode(cmd *cobra.Command) HelpMode {
	// Check for --brief flag
	if brief, _ := cmd.Flags().GetBool("brief"); brief {
		return HelpModeBrief
	}

	// Check for --full flag
	if full, _ := cmd.Flags().GetBool("full"); full {
		return HelpModeFull
	}

	// Check for --examples flag
	if examples, _ := cmd.Flags().GetBool("examples"); examples {
		return HelpModeExamples
	}

	// Default to standard help (triggered by -h or --help)
	return HelpModeStandard
}

// EnhancedHelpFunc provides context-aware help with progressive disclosure.
func EnhancedHelpFunc(cmd *cobra.Command, args []string) {
	mode := GetHelpMode(cmd)

	// If this is a specific command help request (not root), always show command help
	// regardless of interactive mode
	if cmd.Name() != "strigoi" || !globalContext.IsInteractive {
		// Standard command-line help
		switch mode {
		case HelpModeBrief:
			showBriefHelp(cmd)
		case HelpModeExamples:
			showExamples(cmd)
		case HelpModeFull:
			showFullHelp(cmd)
		default:
			showStandardHelp(cmd)
		}
		return
	}

	// Only show interactive context help for root command in interactive mode
	if globalContext.IsInteractive {
		handleInteractiveHelp(cmd, args, mode)
	}
}

// handleInteractiveHelp provides context-aware help in interactive mode.
func handleInteractiveHelp(cmd *cobra.Command, args []string, mode HelpMode) {
	// Clear any command prefix confusion
	if len(args) > 0 && strings.HasPrefix(args[0], globalContext.CurrentPath) {
		suggestion := strings.TrimPrefix(args[0], globalContext.CurrentPath+"/")
		fmt.Printf("%s In interactive mode, you're already in '%s' context.\n",
			helpWarnColor.Sprint("→"), globalContext.CurrentPath)
		fmt.Printf("  Did you mean: %s\n\n", helpCmdColor.Sprint(suggestion))
	}

	// Show contextual commands based on state
	if globalContext.ScanningActive {
		showScanningHelp()
		return
	}

	if globalContext.HasErrors {
		showErrorRecoveryHelp()
		return
	}

	// Default interactive help
	showInteractiveContextHelp(cmd, mode)
}

// showBriefHelp displays one-line help.
func showBriefHelp(cmd *cobra.Command) {
	fmt.Printf("%s: %s\n", helpCmdColor.Sprint(cmd.Name()), cmd.Short)

	if cmd.HasAvailableSubCommands() {
		fmt.Print("\nSubcommands: ")
		var names []string
		for _, sub := range cmd.Commands() {
			if !sub.Hidden {
				names = append(names, sub.Name())
			}
		}
		fmt.Println(strings.Join(names, ", "))
	}

	fmt.Printf("\nUse %s for more info\n", helpInfoColor.Sprint(cmd.Name()+" --help"))
}

// showStandardHelp displays the standard help (current default).
func showStandardHelp(cmd *cobra.Command) {
	// For specific commands, show the actual command help with enhancements
	if cmd.Long != "" {
		fmt.Println(cmd.Long)
	} else if cmd.Short != "" {
		fmt.Println(cmd.Short)
	}

	// Show usage
	if cmd.HasAvailableSubCommands() || cmd.HasAvailableFlags() {
		fmt.Println("\nUsage:")
		fmt.Printf("  %s\n", cmd.UseLine())
	}

	// Show flags if available
	if cmd.HasAvailableLocalFlags() {
		fmt.Println("\nFlags:")
		fmt.Print(cmd.LocalFlags().FlagUsages())
	}

	if cmd.HasAvailableInheritedFlags() {
		fmt.Println("\nGlobal Flags:")
		fmt.Print(cmd.InheritedFlags().FlagUsages())
	}

	// Add quick examples if available
	if examples := getQuickExamples(cmd); len(examples) > 0 {
		fmt.Println(helpInfoColor.Sprint("\nQuick Examples:"))
		for _, ex := range examples {
			fmt.Printf("  %s\n", ex)
		}
	}

	// Add contextual hints
	if hint := getContextualHint(cmd); hint != "" {
		fmt.Printf("\n%s %s\n", helpInfoColor.Sprint("💡 Hint:"), hint)
	}
}

// showFullHelp displays comprehensive help including advanced options.
func showFullHelp(cmd *cobra.Command) {
	showStandardHelp(cmd)

	// Add advanced options
	fmt.Println(helpWarnColor.Sprint("\n📚 Advanced Options:"))
	showAdvancedOptions(cmd)

	// Add configuration details
	fmt.Println(helpWarnColor.Sprint("\n⚙️  Configuration:"))
	showConfigurationDetails(cmd)

	// Add related commands
	fmt.Println(helpWarnColor.Sprint("\n🔗 Related Commands:"))
	showRelatedCommands(cmd)
}

// showExamples displays only examples.
func showExamples(cmd *cobra.Command) {
	fmt.Printf("%s for %s\n\n", helpSuccessColor.Sprint("Examples"), helpCmdColor.Sprint(cmd.Name()))

	examples := getDetailedExamples(cmd)
	if len(examples) == 0 {
		fmt.Println("No examples available for this command.")
		return
	}

	for i, ex := range examples {
		fmt.Printf("%s %d: %s\n", helpInfoColor.Sprint("Example"), i+1, ex.Description)
		fmt.Printf("  %s\n", helpCmdColor.Sprint(ex.Command))
		if ex.Output != "" {
			fmt.Printf("  %s %s\n", helpGrayColor.Sprint("Output:"), ex.Output)
		}
		fmt.Println()
	}
}

// showInteractiveContextHelp shows help relevant to the current interactive context.
func showInteractiveContextHelp(_ *cobra.Command, _ HelpMode) {
	fmt.Printf("%s %s\n", helpInfoColor.Sprint("Current context:"),
		helpCmdColor.Sprint(globalContext.CurrentPath))

	// Show available commands in this context
	fmt.Println(helpSuccessColor.Sprint("\n📁 Available here:"))

	helpNode := convertToHelpNode(globalContext.CurrentNode)
	if helpNode != nil {
		// Show directories
		var dirs []string
		var cmds []string

		for name, child := range helpNode.Children {
			if child.IsDirectory {
				dirs = append(dirs, name+"/")
			} else {
				cmds = append(cmds, name)
			}
		}

		if len(dirs) > 0 {
			fmt.Print("  Directories: ")
			fmt.Println(helpDirColor.Sprint(strings.Join(dirs, " ")))
		}

		if len(cmds) > 0 {
			fmt.Print("  Commands: ")
			fmt.Println(helpCmdColor.Sprint(strings.Join(cmds, " ")))
		}
	} else {
		// Fallback when no node information is available
		fmt.Println("  Use 'ls' to see available commands")
	}

	// Show navigation commands
	fmt.Println(helpUtilColor.Sprint("\n🧭 Navigation:"))
	fmt.Println("  cd <dir>    - Enter directory")
	fmt.Println("  cd ..       - Go up one level")
	fmt.Println("  ls          - List current directory")
	fmt.Println("  pwd         - Show current path")

	// Show command-specific help hint
	fmt.Printf("\n%s Type %s for detailed help on any command\n",
		helpInfoColor.Sprint("💡"), helpCmdColor.Sprint("<command> --help"))
}

// showScanningHelp shows help when scanning is active.
func showScanningHelp() {
	fmt.Println(helpWarnColor.Sprint("⚠️  Scanning in progress"))
	fmt.Println("\nAvailable commands:")
	fmt.Println("  " + helpCmdColor.Sprint("stop") + "     - Stop the current scan")
	fmt.Println("  " + helpCmdColor.Sprint("pause") + "    - Pause the scan")
	fmt.Println("  " + helpCmdColor.Sprint("resume") + "   - Resume a paused scan")
	fmt.Println("  " + helpCmdColor.Sprint("status") + "   - Show scan progress")
}

// showErrorRecoveryHelp shows help for error recovery.
func showErrorRecoveryHelp() {
	fmt.Println(helpErrorColor.Sprint("❌ Previous command encountered errors"))
	fmt.Println("\nSuggested actions:")
	fmt.Println("  " + helpCmdColor.Sprint("logs") + "     - View detailed error logs")
	fmt.Println("  " + helpCmdColor.Sprint("retry") + "    - Retry the last command")
	fmt.Println("  " + helpCmdColor.Sprint("reset") + "    - Reset to clean state")
	fmt.Println("  " + helpCmdColor.Sprint("help") + "     - Show command help")
}

// Example structure for detailed examples.
type Example struct {
	Description string
	Command     string
	Output      string
}

// getQuickExamples returns quick examples for a command.
func getQuickExamples(cmd *cobra.Command) []string {
	examples := map[string][]string{
		"probe": {
			"strigoi probe north localhost",
			"strigoi probe south --scan-mcp",
			"strigoi probe all --output json",
		},
		"south": {
			"south --scan-mcp",
			"south --output json > deps.json",
			"south --severity high,critical",
		},
		"north": {
			"north https://api.example.com",
			"north localhost --ai-preset comprehensive",
			"north --include-local --delay 500",
		},
		"stream": {
			"strigoi stream tap nginx",
			"strigoi stream record 1234 -o capture.log",
			"strigoi stream status",
		},
		"tap": {
			"tap nginx",
			"tap 1234 --filter error",
			"tap python --no-color",
		},
		"record": {
			"record nginx -o nginx.log",
			"record 5678 --format json",
			"record api-server --duration 5m",
		},
	}

	if ex, ok := examples[cmd.Name()]; ok {
		return ex
	}
	return nil
}

// getDetailedExamples returns detailed examples for a command.
func getDetailedExamples(cmd *cobra.Command) []Example {
	examples := map[string][]Example{
		"tap": {
			{
				Description: "Monitor nginx process by name",
				Command:     "tap nginx",
				Output:      "[+] Tapping into process: nginx",
			},
			{
				Description: "Monitor process by PID with error filter",
				Command:     "tap 1234 --filter error",
				Output:      "[*] Filter pattern: error\n[WARN] Error in request handler",
			},
			{
				Description: "Monitor without colors and timestamps",
				Command:     "tap python --no-color --no-timestamps",
				Output:      "Raw stream output without formatting",
			},
		},
		"record": {
			{
				Description: "Record nginx streams to file",
				Command:     "record nginx -o nginx-streams.log",
				Output:      "[+] Recording streams from: nginx\n[*] Output file: nginx-streams.log",
			},
			{
				Description: "Record with JSON format for analysis",
				Command:     "record 5678 --format json",
				Output:      "[*] Recording in JSON format for structured analysis",
			},
			{
				Description: "Time-limited recording session",
				Command:     "record api-server --duration 5m",
				Output:      "[*] Recording for 5 minutes...",
			},
		},
		"south": {
			{
				Description: "Scan for MCP vulnerabilities",
				Command:     "south --scan-mcp --output json",
				Output:      "Found 3 MCP servers with 2 vulnerabilities",
			},
			{
				Description: "Check dependencies with high severity filter",
				Command:     "south --severity high,critical",
				Output:      "2 critical vulnerabilities found in dependencies",
			},
			{
				Description: "Include self-analysis in scan",
				Command:     "south --include-self",
				Output:      "Analyzing 42 packages including Strigoi",
			},
		},
		"north": {
			{
				Description: "Scan local development server",
				Command:     "north localhost:3000",
				Output:      "Found 15 endpoints, 3 with potential issues",
			},
			{
				Description: "Comprehensive AI endpoint discovery",
				Command:     "north --ai-preset comprehensive --include-local",
				Output:      "Discovered 8 AI model endpoints",
			},
		},
	}

	if ex, ok := examples[cmd.Name()]; ok {
		return ex
	}
	return nil
}

// getContextualHint provides smart hints based on context.
func getContextualHint(cmd *cobra.Command) string {
	hints := map[string]string{
		"probe":  "Use 'probe all' for comprehensive scanning",
		"south":  "Add --scan-mcp to detect MCP server vulnerabilities",
		"north":  "Use --ai-preset comprehensive for thorough AI endpoint discovery",
		"stream": "Stream commands require a running process to monitor",
	}

	if hint, ok := hints[cmd.Name()]; ok {
		return hint
	}

	// Dynamic hints based on last error
	if globalContext.LastError != nil {
		return getSuggestionForError(globalContext.LastError)
	}

	return ""
}

// getSuggestionForError provides suggestions based on the error.
func getSuggestionForError(err error) string {
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "permission denied"):
		return "Try running with elevated permissions or check file access rights"
	case strings.Contains(errStr, "connection refused"):
		return "Ensure the target service is running and accessible"
	case strings.Contains(errStr, "timeout"):
		return "Try increasing the timeout with -t flag (e.g., -t 60s)"
	case strings.Contains(errStr, "not found"):
		return "Check the command syntax or use 'help' to see available commands"
	default:
		return ""
	}
}

// showAdvancedOptions displays advanced configuration options.
func showAdvancedOptions(cmd *cobra.Command) {
	// Command-specific advanced options
	advanced := map[string][]string{
		"probe": {
			"Environment Variables:",
			"  STRIGOI_TIMEOUT     - Default timeout for all operations",
			"  STRIGOI_PARALLEL    - Number of parallel workers",
			"  STRIGOI_DEBUG       - Enable debug logging",
		},
		"south": {
			"MCP Scanning Options:",
			"  --scan-mcp          - Deep scan for MCP servers",
			"  --mcp-timeout       - Timeout for MCP discovery",
			"  --mcp-depth         - Recursion depth for scanning",
		},
	}

	if opts, ok := advanced[cmd.Name()]; ok {
		for _, opt := range opts {
			fmt.Println("  " + opt)
		}
	}
}

// showConfigurationDetails shows configuration file options.
func showConfigurationDetails(_ *cobra.Command) {
	fmt.Println("  Config file: ~/.strigoi/config.yaml")
	fmt.Println("  Log directory: ~/.strigoi/logs/")
	fmt.Println("  Cache directory: ~/.strigoi/cache/")
}

// showRelatedCommands suggests related commands.
func showRelatedCommands(cmd *cobra.Command) {
	related := map[string][]string{
		"north": {"south", "probe all"},
		"south": {"north", "probe east"},
		"probe": {"stream tap", "stream record"},
	}

	if rels, ok := related[cmd.Name()]; ok {
		fmt.Println("  " + strings.Join(rels, ", "))
	}
}

// SmartErrorHandler provides intelligent error messages with suggestions.
func SmartErrorHandler(cmd *cobra.Command, err error) error {
	// Check if in interactive mode and handle command confusion
	if globalContext.IsInteractive {
		errStr := err.Error()

		// Handle "command not found" errors
		if strings.Contains(errStr, "command not found") {
			parts := strings.Fields(errStr)
			if len(parts) > 3 {
				badCmd := parts[3]

				// Check if user typed the context prefix
				if strings.HasPrefix(badCmd, "probe") && globalContext.CurrentPath == "/probe" {
					suggestion := strings.TrimPrefix(badCmd, "probe")
					fmt.Printf("\n%s You're already in the 'probe' context.\n",
						helpErrorColor.Sprint("✗"))
					fmt.Printf("   Did you mean: %s\n", helpCmdColor.Sprint(suggestion))
					fmt.Printf("   Type %s to see available commands\n\n",
						helpInfoColor.Sprint("help"))
					return nil
				}

				// Suggest similar commands
				if suggestion := findSimilarCommand(badCmd, cmd); suggestion != "" {
					fmt.Printf("\n%s Command not found: %s\n",
						helpErrorColor.Sprint("✗"), badCmd)
					fmt.Printf("   Did you mean: %s\n", helpCmdColor.Sprint(suggestion))
					fmt.Printf("   Type %s to see all commands\n\n",
						helpInfoColor.Sprint("help"))
					return nil
				}
			}
		}
	}

	// Default error handling
	return err
}

// findSimilarCommand uses Levenshtein distance to find similar commands.
func findSimilarCommand(input string, cmd *cobra.Command) string {
	// This is a simplified version - you'd want a proper Levenshtein implementation
	commands := []string{}
	for _, sub := range cmd.Commands() {
		if !sub.Hidden {
			commands = append(commands, sub.Name())
		}
	}

	// Simple prefix matching for now
	for _, c := range commands {
		if strings.HasPrefix(c, input[:min(len(input), 2)]) {
			return c
		}
	}

	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// InitializeHelpSystem sets up the enhanced help system.
func InitializeHelpSystem(rootCmd *cobra.Command) {
	// Add help level flags to all commands
	addHelpFlags(rootCmd)

	// Set custom help function
	rootCmd.SetHelpFunc(EnhancedHelpFunc)

	// Apply to all subcommands recursively
	applyHelpSystemToCommand(rootCmd)

	// Help flag handlers removed - they were interfering with normal command execution
}

// setupHelpFlagHandlers - REMOVED: This was interfering with normal command execution

// addHelpFlags adds the multi-level help flags to a command.
func addHelpFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("brief", false, "Show brief one-line help")
	cmd.Flags().Bool("full", false, "Show comprehensive help with advanced options")
	cmd.Flags().Bool("examples", false, "Show usage examples only")
}

// applyHelpSystemToCommand recursively applies the help system.
func applyHelpSystemToCommand(cmd *cobra.Command) {
	cmd.SetHelpFunc(EnhancedHelpFunc)

	for _, child := range cmd.Commands() {
		// Only add help flags if they don't already exist
		if child.Flags().Lookup("brief") == nil {
			addHelpFlags(child)
		}
		applyHelpSystemToCommand(child)
	}
}
