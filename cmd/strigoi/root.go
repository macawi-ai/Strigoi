// Strigoi Security Validation Platform
// Copyright © September 2025 Macawi LLC. All Rights Reserved.
// Licensed under CC BY-NC-SA 4.0: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

var (
	// Version info.
	version = "0.5.2"
	build   = "dev"

	// Global verbose flag.
	verbose bool

	// Color scheme.
	errorColor   = color.New(color.FgRed, color.Bold)
	successColor = color.New(color.FgGreen, color.Bold)
	infoColor    = color.New(color.FgBlue)
	warnColor    = color.New(color.FgYellow)
	grayColor    = color.New(color.FgHiBlack)

	// Enhanced colors for visual distinction.
	dirColor  = color.New(color.FgBlue, color.Bold)
	cmdColor  = color.New(color.FgGreen)
	utilColor = color.New(color.FgHiWhite)
	// aliasColor = color.New(color.FgCyan) // Reserved for future use.
)

var rootCmd = &cobra.Command{
	Use:   "strigoi",
	Short: "Advanced Security Validation Platform",
	Long:  getBanner(),
	Run: func(cmd *cobra.Command, _ []string) {
		// Check for help display flags first
		brief, _ := cmd.Flags().GetBool("brief")
		full, _ := cmd.Flags().GetBool("full")
		examples, _ := cmd.Flags().GetBool("examples")

		// Handle brief help
		if brief {
			fmt.Println("strigoi - Advanced Security Validation Platform for AI/LLM infrastructure")
			fmt.Println("Usage: strigoi [command] [flags]")
			fmt.Println("Try 'strigoi --help' for more information")
			return
		}

		// Handle examples
		if examples {
			fmt.Println("Examples:")
			fmt.Println("  strigoi probe all                  # Scan all directions with defaults")
			fmt.Println("  strigoi probe north localhost       # Scan AI endpoints on localhost")
			fmt.Println("  strigoi probe south .               # Check dependencies in current directory")
			fmt.Println("  strigoi probe all --output json     # Output results in JSON format")
			fmt.Println("  strigoi --version                   # Show version information")
			return
		}

		// Handle full help
		if full {
			_ = cmd.Help()
			return
		}

		// Print banner and exit if not in a TTY (for testing)
		if !isInteractive() {
			fmt.Print(getBanner())
			return
		}
		// If no subcommand, start interactive mode
		if err := startInteractiveMode(); err != nil {
			errorColor.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// isInteractive checks if we're running in an interactive terminal
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// Execute runs the root command.
func Execute() error {
	// Disable default completion command (we'll add our own)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Initialize the enhanced help system BEFORE execution
	InitializeHelpSystem(rootCmd)

	// Set custom usage function for all commands
	cobra.OnInitialize(func() {
		// Apply custom help to all commands (keeping backward compatibility)
		applyCustomHelp(rootCmd)
	})

	return rootCmd.Execute()
}

func init() {
	// Set up the command executor for interactive mode
	executeCobraCommand = func(args []string) error {
		// SOLUTION: Completely reset Cobra's flag state to prevent state persistence
		// This fixes the bug where running --help leaves commands in a "help mode"
		rootCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			_ = flag.Value.Set(flag.DefValue)
			flag.Changed = false
		})

		// Reset probe command flags recursively
		if probeCmd != nil {
			probeCmd.Flags().VisitAll(func(flag *pflag.Flag) {
				_ = flag.Value.Set(flag.DefValue)
				flag.Changed = false
			})

			// Reset all probe subcommand flags
			for _, subCmd := range []*cobra.Command{probeNorthCmd, probeSouthCmd, probeEastCmd, probeWestCmd} {
				if subCmd != nil {
					subCmd.Flags().VisitAll(func(flag *pflag.Flag) {
						_ = flag.Value.Set(flag.DefValue)
						flag.Changed = false
					})
				}
			}
		}

		// Execute with the provided arguments using the main command structure
		rootCmd.SetArgs(args)
		rootCmd.SilenceUsage = true
		return rootCmd.Execute()
	}

	// Set up the cobra command finder for interactive mode
	findCobraCommand = func(path []string) *cobra.Command {
		if len(path) == 0 {
			return rootCmd
		}

		current := rootCmd
		for _, part := range path {
			found := false
			for _, cmd := range current.Commands() {
				if cmd.Name() == part {
					current = cmd
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}
		return current
	}

	// Global flags
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Show help for command")
	rootCmd.PersistentFlags().Bool("version", false, "Show version information")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	// Override version flag behavior and configure logging
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Printf("Strigoi v%s (build: %s)\n", version, build)
			os.Exit(0)
		}

		// Configure logging based on verbose flag
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.WarnLevel) // Only show warnings and errors by default
		}

		// Always output logs to stderr to keep stdout clean
		logrus.SetOutput(os.Stderr)

		// Set formatter to text with timestamp
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableColors: false,
			FullTimestamp: true,
		})
	}

	// Add commands
	rootCmd.AddCommand(completionCmd)
	// More commands will be added here
}

func getBanner() string {
	redColor := color.New(color.FgRed, color.Bold)

	banner := redColor.Sprint(`███████╗████████╗██████╗ ██╗ ██████╗  ██████╗ ██╗
██╔════╝╚══██╔══╝██╔══██╗██║██╔════╝ ██╔═══██╗██║
███████╗   ██║   ██████╔╝██║██║  ███╗██║   ██║██║
╚════██║   ██║   ██╔══██╗██║██║   ██║██║   ██║██║
███████║   ██║   ██║  ██║██║╚██████╔╝╚██████╔╝██║
╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝ ╚═════╝  ╚═════╝ ╚═╝`) + "\n\n"

	banner += grayColor.Sprint("Advanced Security Validation Platform v" + version + "\n")
	banner += grayColor.Sprint("Copyright © 2025 Macawi - James R. Saker Jr.\n\n")
	banner += warnColor.Sprint("⚠️  Authorized use only - WHITE HAT SECURITY TESTING\n\n")
	banner += infoColor.Sprint("♥  If Strigoi helps secure your systems, consider supporting:\n")
	banner += "   https://github.com/sponsors/macawi-ai\n"

	return banner
}

// applyCustomHelp recursively applies custom help formatting to all commands.
func applyCustomHelp(cmd *cobra.Command) {
	// Override the usage function for this command
	cmd.SetUsageFunc(func(c *cobra.Command) error {
		fmt.Fprint(c.OutOrStderr(), getColoredUsage(c))
		return nil
	})

	// Apply to all subcommands
	for _, child := range cmd.Commands() {
		applyCustomHelp(child)
	}
}

// getColoredUsage generates our custom colored usage.
func getColoredUsage(c *cobra.Command) string {
	var b strings.Builder

	// Usage line
	if c.HasAvailableSubCommands() || c.HasAvailableFlags() {
		b.WriteString("\nUsage:\n")
		b.WriteString("  " + c.UseLine() + "\n")
	}

	// Available Commands section with colors
	if c.HasAvailableSubCommands() {
		// First, separate directories from commands
		var dirs, cmds []*cobra.Command

		for _, cmd := range c.Commands() {
			if !cmd.Hidden && cmd.IsAvailableCommand() {
				if cmd.HasAvailableSubCommands() {
					dirs = append(dirs, cmd)
				} else {
					cmds = append(cmds, cmd)
				}
			}
		}

		// Show directories
		if len(dirs) > 0 {
			b.WriteString(dirColor.Sprint("\nDirectories:\n"))
			for _, cmd := range dirs {
				b.WriteString(fmt.Sprintf("  %s  %s\n",
					dirColor.Sprintf("%-15s", cmd.Name()+"/"),
					cmd.Short))
			}
		}

		// Show commands
		if len(cmds) > 0 {
			b.WriteString(cmdColor.Sprint("\nCommands:\n"))
			for _, cmd := range cmds {
				cmdType := cmdColor
				switch cmd.Name() {
				case "help", "completion", "version":
					cmdType = utilColor
				}
				b.WriteString(fmt.Sprintf("  %s  %s\n",
					cmdType.Sprintf("%-15s", cmd.Name()),
					cmd.Short))
			}
		}
	}

	// Flags section
	if c.HasAvailableLocalFlags() {
		b.WriteString("\nFlags:\n")
		b.WriteString(c.LocalFlags().FlagUsages())
	}

	if c.HasAvailableInheritedFlags() {
		b.WriteString("\nGlobal Flags:\n")
		b.WriteString(c.InheritedFlags().FlagUsages())
	}

	// Help line
	b.WriteString("\nUse \"" + c.CommandPath() + " [command] --help\" for more information about a command.\n")

	return b.String()
}
