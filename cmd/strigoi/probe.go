package main

import (
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/macawi-ai/strigoi/modules/probe" // Import for init registration
	"github.com/macawi-ai/strigoi/pkg/modules"
	"github.com/macawi-ai/strigoi/pkg/output"
	"github.com/spf13/cobra"
)

var probeCmd = &cobra.Command{
	Use:   "probe",
	Short: "Discovery and reconnaissance tools",
	Long: `Probe in cardinal directions to discover attack surfaces and vulnerabilities.

The probe commands follow cardinal directions:
  - North: API endpoints and external interfaces
  - South: Dependencies and supply chain
  - East: Data flows and integrations
  - West: Authentication and access controls`,
	Run: func(cmd *cobra.Command, _ []string) {
		// If no subcommand, show help
		_ = cmd.Help()
	},
}

// newProbeNorthCommand creates a fresh north command instance
func newProbeNorthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "north [target]",
		Short: "Probe north direction (endpoints)",
		Long:  `Discover and analyze API endpoints, web interfaces, and external attack surfaces.`,
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			// Provide target suggestions
			if len(args) == 0 {
				return []string{"localhost", "api.example.com", "https://target.com"}, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Check for progressive disclosure flags first
			brief, _ := cmd.Flags().GetBool("brief")
			full, _ := cmd.Flags().GetBool("full")
			examples, _ := cmd.Flags().GetBool("examples")

			// Handle help modes
			if brief {
				showBriefHelp(cmd)
				return
			}
			if examples {
				showExamples(cmd)
				return
			}
			if full {
				showFullHelp(cmd)
				return
			}

			// Handle normal execution - require a target
			if len(args) == 0 {
				_ = cmd.Help()
				return
			}

			target := args[0]

			// Get flags
			outputFormat, _ := cmd.Flags().GetString("output")
			timeout, _ := cmd.Flags().GetString("timeout")
			followRedirects, _ := cmd.Flags().GetBool("follow-redirects")
			headers, _ := cmd.Flags().GetStringSlice("headers")
			aiPreset, _ := cmd.Flags().GetString("ai-preset")
			includeLocal, _ := cmd.Flags().GetBool("include-local")
			delay, _ := cmd.Flags().GetInt("delay")

			// Load modules if not already loaded
			_ = modules.LoadBuiltins(nil)

			// Get the module from registry
			module, err := modules.Get("probe/north")
			if err != nil {
				errorColor.Printf("[-] Failed to load module: %v\n", err)
				return
			}

			// Configure the module
			if err := module.SetOption("target", target); err != nil {
				errorColor.Printf("[-] Failed to set target: %v\n", err)
				return
			}

			// Parse timeout
			if timeout != "" {
				if d, err := time.ParseDuration(timeout); err == nil {
					if err := module.SetOption("timeout", fmt.Sprintf("%d", int(d.Seconds()))); err != nil {
						errorColor.Printf("[-] Failed to set timeout: %v\n", err)
						return
					}
				}
			}

			// Set AI-specific options
			if err := module.SetOption("ai-preset", aiPreset); err != nil {
				errorColor.Printf("[-] Failed to set ai-preset: %v\n", err)
				return
			}

			if err := module.SetOption("include-local", fmt.Sprintf("%v", includeLocal)); err != nil {
				errorColor.Printf("[-] Failed to set include-local: %v\n", err)
				return
			}

			if err := module.SetOption("delay", fmt.Sprintf("%d", delay)); err != nil {
				errorColor.Printf("[-] Failed to set delay: %v\n", err)
				return
			}

			// Run the module
			if verbose {
				fmt.Println(infoColor.Sprint("[*] Verbose mode enabled"))
				fmt.Printf("%s Target: %s\n", infoColor.Sprint("[*]"), target)
				fmt.Printf("%s Timeout: %s\n", infoColor.Sprint("[*]"), timeout)
				fmt.Printf("%s Follow redirects: %v\n", infoColor.Sprint("[*]"), followRedirects)
				if len(headers) > 0 {
					fmt.Printf("%s Custom headers: %v\n", infoColor.Sprint("[*]"), headers)
				}
			}

			// Only print status messages if not JSON output
			if outputFormat != "json" {
				fmt.Println(successColor.Sprint("[+] Starting endpoint discovery..."))
			}

			result, err := module.Run()
			if err != nil {
				errorColor.Printf("[-] Error: %v\n", err)
				return
			}

			// Convert module result to standard output format
			standardOutput := output.ConvertModuleResult(*result)
			standardOutput.Target = target

			// Extract and enhance summary from results
			if standardOutput.Summary == nil {
				standardOutput.Summary = output.ExtractSummaryFromResults(standardOutput.Results)
			}

			// Format and display output
			noColor, _ := cmd.Flags().GetBool("no-color")
			severityFilter, _ := cmd.Flags().GetStringSlice("severity")
			verbosity := "normal"
			if verbose {
				verbosity = "verbose"
			}

			formatted, err := output.FormatOutput(
				standardOutput,
				outputFormat,   // format
				verbosity,      // verbosity
				noColor,        // no-color
				severityFilter, // severity filter
			)

			if err != nil {
				errorColor.Printf("[-] Failed to format output: %v\n", err)
				// Fallback to JSON
				data, _ := json.MarshalIndent(result.Data, "", "  ")
				fmt.Println(string(data))
				return
			}

			fmt.Print(formatted)
		},
	}
}

var probeNorthCmd = newProbeNorthCommand()

var probeSouthCmd = &cobra.Command{
	Use:   "south [target]",
	Short: "Probe south direction (dependencies)",
	Long:  `Analyze dependencies, libraries, and supply chain vulnerabilities.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Check for progressive disclosure flags first
		brief, _ := cmd.Flags().GetBool("brief")
		full, _ := cmd.Flags().GetBool("full")
		examples, _ := cmd.Flags().GetBool("examples")

		// Handle help modes
		if brief {
			showBriefHelp(cmd)
			return
		}
		if examples {
			showExamples(cmd)
			return
		}
		if full {
			showFullHelp(cmd)
			return
		}

		// Handle normal execution with default target
		target := "./"
		if len(args) > 0 {
			target = args[0]
		}

		// Get flags
		outputFormat, _ := cmd.Flags().GetString("output")
		scanMCP, _ := cmd.Flags().GetBool("scan-mcp")
		includeSelf, _ := cmd.Flags().GetBool("include-self")

		// Load modules if not already loaded
		_ = modules.LoadBuiltins(nil)

		// Get the module from registry
		module, err := modules.Get("probe/south")
		if err != nil {
			errorColor.Printf("[-] Failed to load module: %v\n", err)
			return
		}

		// Configure the module
		if err := module.SetOption("target", target); err != nil {
			errorColor.Printf("[-] Failed to set target: %v\n", err)
			return
		}

		// Set MCP scanning option
		if err := module.SetOption("scan_mcp", fmt.Sprintf("%t", scanMCP)); err != nil {
			errorColor.Printf("[-] Failed to set scan_mcp: %v\n", err)
			return
		}

		// Set include-self option
		if err := module.SetOption("include_self", fmt.Sprintf("%t", includeSelf)); err != nil {
			errorColor.Printf("[-] Failed to set include_self: %v\n", err)
			return
		}

		// Run the module
		if verbose {
			fmt.Printf("%s Target: %s\n", infoColor.Sprint("[*]"), target)
		}

		// Only print status messages if not JSON output
		if outputFormat != "json" {
			fmt.Println(successColor.Sprint("[+] Analyzing dependencies..."))
		}

		result, err := module.Run()
		if err != nil {
			errorColor.Printf("[-] Error: %v\n", err)
			return
		}

		// Convert module result to standard output format
		standardOutput := output.ConvertModuleResult(*result)
		standardOutput.Target = target

		// Extract and enhance summary from results
		if standardOutput.Summary == nil {
			standardOutput.Summary = output.ExtractSummaryFromResults(standardOutput.Results)
		}

		// Format and display output
		noColor, _ := cmd.Flags().GetBool("no-color")
		severityFilter, _ := cmd.Flags().GetStringSlice("severity")
		verbosity := "normal"
		if verbose {
			verbosity = "verbose"
		}

		formatted, err := output.FormatOutput(
			standardOutput,
			outputFormat,   // format
			verbosity,      // verbosity
			noColor,        // no-color
			severityFilter, // severity filter
		)

		if err != nil {
			errorColor.Printf("[-] Failed to format output: %v\n", err)
			// Fallback to JSON
			data, _ := json.MarshalIndent(result.Data, "", "  ")
			fmt.Println(string(data))
			return
		}

		fmt.Print(formatted)
	},
}

var probeEastCmd = &cobra.Command{
	Use:   "east [target]",
	Short: "Probe east direction (data flows)",
	Long:  `Trace data flows, API integrations, and information leakage.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := "./"
		if len(args) > 0 {
			target = args[0]
		}

		// Get flags
		outputFormat, _ := cmd.Flags().GetString("output")
		includeSelf, _ := cmd.Flags().GetBool("include-self")

		// Load modules if not already loaded
		_ = modules.LoadBuiltins(nil)

		// Get the module from registry
		module, err := modules.Get("probe/east")
		if err != nil {
			errorColor.Printf("[-] Failed to load module: %v\n", err)
			return
		}

		// Configure the module
		if err := module.SetOption("target", target); err != nil {
			errorColor.Printf("[-] Failed to set target: %v\n", err)
			return
		}

		// Set include-self option
		if err := module.SetOption("include_self", fmt.Sprintf("%t", includeSelf)); err != nil {
			errorColor.Printf("[-] Failed to set include_self: %v\n", err)
			return
		}

		// Run the module
		if verbose {
			fmt.Printf("%s Target: %s\n", infoColor.Sprint("[*]"), target)
		}

		// Only print status messages if not JSON output
		if outputFormat != "json" {
			fmt.Println(successColor.Sprint("[+] Tracing data flows..."))
		}

		result, err := module.Run()
		if err != nil {
			errorColor.Printf("[-] Error: %v\n", err)
			return
		}

		// Convert module result to standard output format
		standardOutput := output.ConvertModuleResult(*result)
		standardOutput.Target = target

		// Extract and enhance summary from results
		if standardOutput.Summary == nil {
			standardOutput.Summary = output.ExtractSummaryFromResults(standardOutput.Results)
		}

		// Format and display output
		noColor, _ := cmd.Flags().GetBool("no-color")
		severityFilter, _ := cmd.Flags().GetStringSlice("severity")
		verbosity := "normal"
		if verbose {
			verbosity = "verbose"
		}

		formatted, err := output.FormatOutput(
			standardOutput,
			outputFormat,   // format
			verbosity,      // verbosity
			noColor,        // no-color
			severityFilter, // severity filter
		)

		if err != nil {
			errorColor.Printf("[-] Failed to format output: %v\n", err)
			// Fallback to JSON
			data, _ := json.MarshalIndent(result.Data, "", "  ")
			fmt.Println(string(data))
			return
		}

		fmt.Print(formatted)
	},
}

var probeWestCmd = &cobra.Command{
	Use:   "west [target]",
	Short: "Probe west direction (authentication)",
	Long:  `Examine authentication, authorization, and access control mechanisms.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := "https://localhost"
		if len(args) > 0 {
			target = args[0]
		}

		// Get flags
		outputFormat, _ := cmd.Flags().GetString("output")
		timeout, _ := cmd.Flags().GetString("timeout")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		allowPrivate, _ := cmd.Flags().GetBool("allow-private")

		// Load modules if not already loaded
		_ = modules.LoadBuiltins(nil)

		// Get the module from registry
		module, err := modules.Get("probe/west")
		if err != nil {
			errorColor.Printf("[-] Failed to load module: %v\n", err)
			return
		}

		// Configure the module
		if err := module.SetOption("target", target); err != nil {
			errorColor.Printf("[-] Failed to set target: %v\n", err)
			return
		}

		// Parse timeout
		if timeout != "" {
			if d, err := time.ParseDuration(timeout); err == nil {
				if err := module.SetOption("timeout", fmt.Sprintf("%d", int(d.Seconds()))); err != nil {
					errorColor.Printf("[-] Failed to set timeout: %v\n", err)
					return
				}
			}
		}

		// Set dry run mode
		if err := module.SetOption("dry_run", fmt.Sprintf("%v", dryRun)); err != nil {
			errorColor.Printf("[-] Failed to set dry_run: %v\n", err)
			return
		}

		// Set allow private mode
		if err := module.SetOption("allow_private", fmt.Sprintf("%v", allowPrivate)); err != nil {
			errorColor.Printf("[-] Failed to set allow_private: %v\n", err)
			return
		}

		// Run the module
		if verbose {
			fmt.Printf("%s Target: %s\n", infoColor.Sprint("[*]"), target)
			fmt.Printf("%s Timeout: %s\n", infoColor.Sprint("[*]"), timeout)
			if dryRun {
				fmt.Printf("%s Mode: Dry run (passive analysis only)\n", infoColor.Sprint("[*]"))
			}
		}

		// Only print status messages if not JSON output
		if outputFormat != "json" {
			fmt.Println(successColor.Sprint("[+] Testing authentication boundaries..."))
		}

		result, err := module.Run()
		if err != nil {
			errorColor.Printf("[-] Error: %v\n", err)
			return
		}

		// Convert module result to standard output format
		standardOutput := output.ConvertModuleResult(*result)
		standardOutput.Target = target

		// Extract and enhance summary from results
		if standardOutput.Summary == nil {
			standardOutput.Summary = output.ExtractSummaryFromResults(standardOutput.Results)
		}

		// Format and display output
		noColor, _ := cmd.Flags().GetBool("no-color")
		severityFilter, _ := cmd.Flags().GetStringSlice("severity")
		verbosity := "normal"
		if verbose {
			verbosity = "verbose"
		}

		formatted, err := output.FormatOutput(
			standardOutput,
			outputFormat,   // format
			verbosity,      // verbosity
			noColor,        // no-color
			severityFilter, // severity filter
		)

		if err != nil {
			errorColor.Printf("[-] Failed to format output: %v\n", err)
			// Fallback to JSON
			data, _ := json.MarshalIndent(result.Data, "", "  ")
			fmt.Println(string(data))
			return
		}

		fmt.Print(formatted)
	},
}

var probeAllCmd = &cobra.Command{
	Use:   "all [target]",
	Short: "Probe all directions",
	Long:  `Execute probes in all cardinal directions for comprehensive reconnaissance.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get flags from probe all command
		outputFormat, _ := cmd.Flags().GetString("output")
		timeout, _ := cmd.Flags().GetString("timeout")
		noColor, _ := cmd.Flags().GetBool("no-color")
		severity, _ := cmd.Flags().GetStringSlice("severity")

		// Only print status messages if not JSON output
		if outputFormat != "json" && outputFormat != "yaml" {
			fmt.Println(successColor.Sprint("[+] Probing all directions..."))
		}

		// Determine target - use provided or sensible defaults
		target := "." // Default to current directory
		if len(args) > 0 {
			target = args[0]
		}

		// Pass flags to each probe command
		for _, probeCmd := range []*cobra.Command{probeNorthCmd, probeSouthCmd, probeEastCmd, probeWestCmd} {
			_ = probeCmd.Flags().Set("output", outputFormat)
			_ = probeCmd.Flags().Set("timeout", timeout)
			_ = probeCmd.Flags().Set("no-color", fmt.Sprintf("%v", noColor))
			if len(severity) > 0 {
				for _, s := range severity {
					_ = probeCmd.Flags().Set("severity", s)
				}
			}
		}

		// Run NORTH with target for AI config discovery
		if outputFormat != "json" && outputFormat != "yaml" {
			fmt.Printf("\n%s %s\n", warnColor.Sprint("→"), "NORTH")
		}
		probeNorthCmd.Run(probeNorthCmd, []string{target})

		// Run SOUTH with target directory
		if outputFormat != "json" && outputFormat != "yaml" {
			fmt.Printf("\n%s %s\n", warnColor.Sprint("→"), "SOUTH")
		}
		probeSouthCmd.Run(probeSouthCmd, []string{target})

		// Run EAST with target directory
		if outputFormat != "json" && outputFormat != "yaml" {
			fmt.Printf("\n%s %s\n", warnColor.Sprint("→"), "EAST")
		}
		probeEastCmd.Run(probeEastCmd, []string{target})

		// Run WEST with special handling
		// For probe all, always allow private scanning
		if outputFormat != "json" && outputFormat != "yaml" {
			fmt.Printf("\n%s %s\n", warnColor.Sprint("→"), "WEST")
		}
		// Temporarily set the allow-private flag for West
		_ = probeWestCmd.Flags().Set("allow-private", "true")
		westTarget := target
		if target == "." {
			// West needs a host, not a directory
			westTarget = "localhost"
		}
		probeWestCmd.Run(probeWestCmd, []string{westTarget})
	},
}

func init() {
	// Add probe to root
	rootCmd.AddCommand(probeCmd)

	// Add subcommands to probe
	probeCmd.AddCommand(probeNorthCmd)
	probeCmd.AddCommand(probeSouthCmd)
	probeCmd.AddCommand(probeEastCmd)
	probeCmd.AddCommand(probeWestCmd)
	probeCmd.AddCommand(probeAllCmd)

	// Common flags for all probe commands
	for _, cmd := range []*cobra.Command{probeNorthCmd, probeSouthCmd, probeEastCmd, probeWestCmd, probeAllCmd} {
		cmd.Flags().StringP("output", "o", "pretty", "Output format (pretty, json, yaml, markdown)")
		cmd.Flags().StringP("timeout", "t", "30s", "Timeout for probe operations")
		cmd.Flags().Bool("no-color", false, "Disable colored output")
		cmd.Flags().StringSlice("severity", nil, "Filter by severity (critical, high, medium, low, info)")
	}

	// Special flags for north
	probeNorthCmd.Flags().Bool("follow-redirects", true, "Follow HTTP redirects")
	probeNorthCmd.Flags().StringSlice("headers", nil, "Custom headers for HTTP requests")
	probeNorthCmd.Flags().String("ai-preset", "basic", "AI endpoint preset (basic, comprehensive, local)")
	probeNorthCmd.Flags().Bool("include-local", false, "Include local model server ports")
	probeNorthCmd.Flags().Int("delay", 100, "Delay between requests in milliseconds")

	// Progressive disclosure flags for north
	probeNorthCmd.Flags().Bool("brief", false, "Show brief one-line help")
	probeNorthCmd.Flags().Bool("full", false, "Show comprehensive help with advanced options")
	probeNorthCmd.Flags().Bool("examples", false, "Show usage examples only")

	// Special flags for south
	probeSouthCmd.Flags().Bool("scan-mcp", false, "Enable MCP tools scanning")
	probeSouthCmd.Flags().Bool("include-self", false, "Include Strigoi's own files and processes in scan")

	// Progressive disclosure flags for south
	probeSouthCmd.Flags().Bool("brief", false, "Show brief one-line help")
	probeSouthCmd.Flags().Bool("full", false, "Show comprehensive help with advanced options")
	probeSouthCmd.Flags().Bool("examples", false, "Show usage examples only")

	// Special flags for east
	probeEastCmd.Flags().Bool("include-self", false, "Include Strigoi's own files in scan")

	// Special flags for west
	probeWestCmd.Flags().Bool("dry-run", false, "Perform passive analysis only (no active probing)")
	probeWestCmd.Flags().Float64("rate-limit", 10.0, "Requests per second")
	probeWestCmd.Flags().Int("max-concurrent", 5, "Maximum concurrent requests")
	probeWestCmd.Flags().Bool("allow-private", false, "Allow scanning private/local addresses")
}
