package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/macawi-ai/strigoi/pkg/modules"
	"github.com/macawi-ai/strigoi/pkg/session"
	"github.com/spf13/cobra"
)

var (
	sessionManager *session.Manager
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage module configuration sessions",
	Long: `Save and load module configurations for easy reuse.
Sessions allow you to persist complex module setups and reload them later.`,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		// Initialize session manager
		home, _ := os.UserHomeDir()
		sessionPath := filepath.Join(home, ".strigoi", "sessions")

		var err error
		sessionManager, err = session.NewManager(sessionPath)
		if err != nil {
			errorColor.Printf("[-] Failed to initialize session manager: %v\n", err)
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, _ []string) {
		// If no subcommand, show help
		_ = cmd.Help()
	},
}

var sessionSaveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Save current module configuration",
	Long:  `Save the current module configuration to a named session for later use.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		// Get flags
		description, _ := cmd.Flags().GetString("description")
		tags, _ := cmd.Flags().GetStringSlice("tags")
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		passphrase, _ := cmd.Flags().GetString("passphrase")

		// Check if we have a module to save
		// For now, we'll create a test module
		// In real implementation, this would get the current active module

		// Load modules
		if err := modules.LoadBuiltins(nil); err != nil {
			errorColor.Printf("[-] Failed to load modules: %v\n", err)
			return
		}

		// Get a module for testing
		module, err := modules.Get("probe/north")
		if err != nil {
			errorColor.Printf("[-] No active module to save\n")
			fmt.Println(infoColor.Sprint("[*] Load a module first with 'module use <name>'"))
			return
		}

		// Configure some test options
		_ = module.SetOption("target", "example.com")
		_ = module.SetOption("timeout", "30")

		// Save the session
		opts := session.SaveOptions{
			Description: description,
			Tags:        tags,
			Overwrite:   overwrite,
			Passphrase:  passphrase,
		}

		if err := sessionManager.SaveWithSalt(name, module, opts); err != nil {
			errorColor.Printf("[-] Failed to save session: %v\n", err)
			return
		}

		successColor.Printf("[+] Session '%s' saved successfully\n", name)

		if passphrase != "" {
			fmt.Println(warnColor.Sprint("[!] Session is encrypted. You'll need the passphrase to load it."))
		}
	},
}

var sessionLoadCmd = &cobra.Command{
	Use:   "load <name>",
	Short: "Load a saved session",
	Long:  `Load a previously saved module configuration session.`,
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// List available sessions for completion
			sessions, err := sessionManager.List()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var names []string
			for _, s := range sessions {
				names = append(names, s.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		passphrase, _ := cmd.Flags().GetString("passphrase")

		// Load the session
		opts := session.LoadOptions{
			Passphrase: passphrase,
		}

		sess, err := sessionManager.Load(name, opts)
		if err != nil {
			errorColor.Printf("[-] Failed to load session: %v\n", err)
			return
		}

		successColor.Printf("[+] Loaded session '%s'\n", name)
		fmt.Printf("%s Module: %s\n", infoColor.Sprint("[*]"), cmdColor.Sprint(sess.Module.Name))

		if sess.Description != "" {
			fmt.Printf("%s Description: %s\n", infoColor.Sprint("[*]"), sess.Description)
		}

		// Show options
		fmt.Println(infoColor.Sprint("\n[*] Configuration:"))
		for name, value := range sess.Module.Options {
			// Check if sensitive
			isSensitive := false
			for _, s := range sess.Module.Sensitive {
				if s == name {
					isSensitive = true
					break
				}
			}

			if isSensitive {
				fmt.Printf("  %s = %s\n", name, grayColor.Sprint("********"))
			} else {
				fmt.Printf("  %s = %v\n", name, value)
			}
		}

		fmt.Println(infoColor.Sprint("\n[*] Use 'run' to execute the loaded module"))
	},
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved sessions",
	Long:  `Display a list of all saved module configuration sessions.`,
	Run: func(cmd *cobra.Command, _ []string) {
		// Get flags
		long, _ := cmd.Flags().GetBool("long")
		tags, _ := cmd.Flags().GetStringSlice("tags")

		// List sessions
		sessions, err := sessionManager.List()
		if err != nil {
			errorColor.Printf("[-] Failed to list sessions: %v\n", err)
			return
		}

		if len(sessions) == 0 {
			fmt.Println(infoColor.Sprint("[*] No saved sessions found"))
			return
		}

		// Filter by tags if specified
		if len(tags) > 0 {
			var filtered []session.Info
			for _, s := range sessions {
				// Load full session info to check tags
				info, err := sessionManager.Info(s.Name)
				if err != nil {
					continue
				}

				// Check if session has all required tags
				hasAllTags := true
				for _, tag := range tags {
					found := false
					for _, sTag := range info.Tags {
						if sTag == tag {
							found = true
							break
						}
					}
					if !found {
						hasAllTags = false
						break
					}
				}

				if hasAllTags {
					filtered = append(filtered, *info)
				}
			}
			sessions = filtered
		}

		// Sort by modified time (newest first)
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].Modified.After(sessions[j].Modified)
		})

		fmt.Printf("%s Found %d session(s):\n\n", successColor.Sprint("[+]"), len(sessions))

		if long {
			// Detailed view
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "NAME\tMODULE\tMODIFIED\tSIZE\tENCRYPTED\tTAGS\n")

			for _, s := range sessions {
				// Load full info
				info, err := sessionManager.Info(s.Name)
				if err != nil {
					continue
				}

				encrypted := "No"
				if info.Encrypted {
					encrypted = "Yes"
				}

				size := formatSize(info.Size)
				modified := formatTime(info.Modified)
				tags := strings.Join(info.Tags, ", ")
				if tags == "" {
					tags = "-"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					cmdColor.Sprint(info.Name),
					info.Module,
					modified,
					size,
					encrypted,
					tags,
				)
			}
			w.Flush()
		} else {
			// Simple view
			for _, s := range sessions {
				info, _ := sessionManager.Info(s.Name)

				fmt.Printf("  %s", cmdColor.Sprint(s.Name))
				if info != nil && info.Module != "" {
					fmt.Printf(" (%s)", grayColor.Sprint(info.Module))
				}
				if info != nil && info.Encrypted {
					fmt.Printf(" %s", warnColor.Sprint("[encrypted]"))
				}
				fmt.Println()
			}
		}
	},
}

var sessionInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed information about a session",
	Long:  `Display detailed information about a saved session.`,
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			sessions, _ := sessionManager.List()
			var names []string
			for _, s := range sessions {
				names = append(names, s.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		showValues, _ := cmd.Flags().GetBool("show-values")
		passphrase, _ := cmd.Flags().GetString("passphrase")

		// Get basic info
		info, err := sessionManager.Info(name)
		if err != nil {
			errorColor.Printf("[-] Failed to get session info: %v\n", err)
			return
		}

		// Display basic info
		fmt.Printf("\n%s %s\n", successColor.Sprint("Session:"), cmdColor.Sprint(info.Name))
		fmt.Println(strings.Repeat("─", 60))

		fmt.Printf("%s %s\n", infoColor.Sprint("Module:"), info.Module)
		fmt.Printf("%s %s\n", infoColor.Sprint("Modified:"), formatTime(info.Modified))
		fmt.Printf("%s %s\n", infoColor.Sprint("Size:"), formatSize(info.Size))
		fmt.Printf("%s %v\n", infoColor.Sprint("Encrypted:"), info.Encrypted)

		if info.Description != "" {
			fmt.Printf("%s %s\n", infoColor.Sprint("Description:"), info.Description)
		}

		if len(info.Tags) > 0 {
			fmt.Printf("%s %s\n", infoColor.Sprint("Tags:"), strings.Join(info.Tags, ", "))
		}

		// If requested, show full session details
		if showValues && (passphrase != "" || !info.Encrypted) {
			opts := session.LoadOptions{
				Passphrase: passphrase,
			}

			sess, err := sessionManager.Load(name, opts)
			if err != nil {
				errorColor.Printf("\n[-] Failed to load full session: %v\n", err)
				return
			}

			fmt.Println(infoColor.Sprint("\n[*] Configuration:"))
			for optName, value := range sess.Module.Options {
				// Check if sensitive
				isSensitive := false
				for _, s := range sess.Module.Sensitive {
					if s == optName {
						isSensitive = true
						break
					}
				}

				if isSensitive && !showValues {
					fmt.Printf("  %s = %s\n", optName, grayColor.Sprint("********"))
				} else {
					fmt.Printf("  %s = %v\n", optName, value)
				}
			}

			if len(sess.Metadata) > 0 {
				fmt.Println(infoColor.Sprint("\n[*] Metadata:"))
				for k, v := range sess.Metadata {
					if k != "salt" { // Don't show salt
						fmt.Printf("  %s = %v\n", k, v)
					}
				}
			}
		}
	},
}

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a saved session",
	Long:  `Remove a saved session from storage.`,
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			sessions, _ := sessionManager.List()
			var names []string
			for _, s := range sessions {
				names = append(names, s.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		// Confirm deletion if not forced
		if !force {
			fmt.Printf("%s Delete session '%s'? [y/N] ", warnColor.Sprint("[?]"), name)
			var confirm string
			_, _ = fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" {
				fmt.Println("Cancelled")
				return
			}
		}

		// Delete the session
		if err := sessionManager.Delete(name); err != nil {
			errorColor.Printf("[-] Failed to delete session: %v\n", err)
			return
		}

		successColor.Printf("[+] Session '%s' deleted\n", name)
	},
}

// Helper functions.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%d min ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

func init() {
	// Add session command to root
	rootCmd.AddCommand(sessionCmd)

	// Add subcommands
	sessionCmd.AddCommand(sessionSaveCmd)
	sessionCmd.AddCommand(sessionLoadCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionInfoCmd)
	sessionCmd.AddCommand(sessionDeleteCmd)

	// Save command flags
	sessionSaveCmd.Flags().StringP("description", "d", "", "Session description")
	sessionSaveCmd.Flags().StringSliceP("tags", "t", nil, "Tags for organization")
	sessionSaveCmd.Flags().BoolP("overwrite", "o", false, "Overwrite existing session")
	sessionSaveCmd.Flags().StringP("passphrase", "p", "", "Passphrase for encryption")

	// Load command flags
	sessionLoadCmd.Flags().StringP("passphrase", "p", "", "Passphrase for decryption")

	// List command flags
	sessionListCmd.Flags().BoolP("long", "l", false, "Show detailed information")
	sessionListCmd.Flags().StringSliceP("tags", "t", nil, "Filter by tags")

	// Info command flags
	sessionInfoCmd.Flags().BoolP("show-values", "v", false, "Show all values (including sensitive)")
	sessionInfoCmd.Flags().StringP("passphrase", "p", "", "Passphrase for encrypted sessions")

	// Delete command flags
	sessionDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
}
