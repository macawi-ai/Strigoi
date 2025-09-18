package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "STDIO stream monitoring & analysis",
	Long: `Monitor and analyze input/output streams from processes, network connections, 
and other sources for security validation.`,
	Run: func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	},
}

var streamTapCmd = &cobra.Command{
	Use:   "tap <pid|name>",
	Short: "Monitor process STDIO in real-time",
	Long:  `Attach to a running process and monitor its standard input/output streams in real-time.`,
	Args:  cobra.MaximumNArgs(1), // Changed to allow 0 args for help modes
	ValidArgsFunction: func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// In real implementation, this would list running processes
		if len(args) == 0 {
			suggestions := []string{
				"1234", "5678", // PIDs
				"nginx", "apache", "node", "python", // Process names
			}

			filtered := []string{}
			for _, s := range suggestions {
				if strings.HasPrefix(s, toComplete) {
					filtered = append(filtered, s)
				}
			}
			return filtered, cobra.ShellCompDirectiveNoFileComp
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

		// Normal execution requires a target
		if len(args) == 0 {
			_ = cmd.Help()
			return
		}

		target := args[0]
		filter, _ := cmd.Flags().GetString("filter")

		fmt.Printf("%s Tapping into process: %s\n", successColor.Sprint("[+]"), target)

		if filter != "" {
			fmt.Printf("%s Filter pattern: %s\n", infoColor.Sprint("[*]"), filter)
		}

		// Simulate stream output
		fmt.Println(infoColor.Sprint("\n[*] Stream output:"))
		fmt.Println(grayColor.Sprint("2024-01-15 10:23:45 [INFO] Application started"))
		fmt.Println(grayColor.Sprint("2024-01-15 10:23:46 [DEBUG] Connected to database"))

		if filter == "" || strings.Contains("api request", filter) {
			fmt.Println(warnColor.Sprint("2024-01-15 10:23:47 [WARN] Suspicious API request detected"))
		}

		fmt.Println(grayColor.Sprint("2024-01-15 10:23:48 [INFO] Request processed"))
	},
}

var streamRecordCmd = &cobra.Command{
	Use:   "record <pid|name>",
	Short: "Record streams for later analysis",
	Long:  `Record process streams to a file for offline analysis and forensics.`,
	Args:  cobra.MaximumNArgs(1), // Changed to allow 0 args for help modes
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

		// Normal execution requires a target
		if len(args) == 0 {
			_ = cmd.Help()
			return
		}

		target := args[0]
		output, _ := cmd.Flags().GetString("output")

		fmt.Printf("%s Recording streams from: %s\n", successColor.Sprint("[+]"), target)
		fmt.Printf("%s Output file: %s\n", infoColor.Sprint("[*]"), output)

		// Simulate recording
		fmt.Println(infoColor.Sprint("[*] Recording... Press Ctrl+C to stop"))
	},
}

var streamStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show stream monitoring status",
	Long:  `Display the current status of all active stream monitoring sessions.`,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(successColor.Sprint("[+] Active stream monitors:"))
		fmt.Println()

		// Simulate status output
		fmt.Printf("  %s  PID: 1234  Duration: 5m23s  Events: 1,523\n", cmdColor.Sprint("nginx"))
		fmt.Printf("  %s  PID: 5678  Duration: 2m11s  Events: 423\n", cmdColor.Sprint("api-server"))

		fmt.Println()
		fmt.Println(infoColor.Sprint("[*] Total events captured: 1,946"))
	},
}

func init() {
	// Add stream to root
	rootCmd.AddCommand(streamCmd)

	// Add subcommands
	streamCmd.AddCommand(streamTapCmd)
	streamCmd.AddCommand(streamRecordCmd)
	streamCmd.AddCommand(streamStatusCmd)

	// Tap command flags
	streamTapCmd.Flags().StringP("filter", "f", "", "Filter pattern (regex)")
	streamTapCmd.Flags().Bool("color", true, "Colorize output")
	streamTapCmd.Flags().Bool("timestamps", true, "Show timestamps")

	// Record command flags
	streamRecordCmd.Flags().StringP("output", "o", "stream.log", "Output file path")
	streamRecordCmd.Flags().String("format", "json", "Output format (json, raw, pcap)")
	streamRecordCmd.Flags().Duration("duration", 0, "Recording duration (0 for unlimited)")
}
