package main

import (
	"fmt"
	"sort"
	"strings"

	_ "github.com/macawi-ai/strigoi/modules/probe" // Import for side effects (init)
	"github.com/macawi-ai/strigoi/pkg/modules"
	"github.com/spf13/cobra"
)

var moduleCmd = &cobra.Command{
	Use:   "module",
	Short: "Module management commands",
	Long:  `List, search, and get information about available modules.`,
	Run: func(cmd *cobra.Command, _ []string) {
		// If no subcommand, show help
		_ = cmd.Help()
	},
}

var moduleListCmd = &cobra.Command{
	Use:   "list [type]",
	Short: "List available modules",
	Long:  `List all available modules or filter by type (probe, stream, sense, exploit).`,
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{"probe", "stream", "sense", "exploit"}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(_ *cobra.Command, args []string) {
		// Load all modules
		if err := modules.LoadBuiltins(nil); err != nil {
			errorColor.Printf("[-] Failed to load modules: %v\n", err)
			return
		}

		var moduleList []modules.Module

		if len(args) > 0 {
			// Filter by type
			moduleType := modules.ModuleType(args[0])
			moduleList = modules.ListByType(moduleType)
			fmt.Printf("%s Modules of type '%s':\n\n", infoColor.Sprint("[*]"), moduleType)
		} else {
			// List all modules
			moduleList = modules.List()
			fmt.Printf("%s Available modules (%d):\n\n", infoColor.Sprint("[*]"), modules.GlobalRegistry.Count())
		}

		if len(moduleList) == 0 {
			fmt.Println("  No modules found")
			return
		}

		// Group by type
		byType := make(map[modules.ModuleType][]modules.Module)
		for _, mod := range moduleList {
			byType[mod.Type()] = append(byType[mod.Type()], mod)
		}

		// Sort types
		var types []modules.ModuleType
		for t := range byType {
			types = append(types, t)
		}
		sort.Slice(types, func(i, j int) bool {
			return string(types[i]) < string(types[j])
		})

		// Display modules
		for _, moduleType := range types {
			typeColor := cmdColor
			switch moduleType {
			case modules.ProbeModule:
				typeColor = dirColor
			case modules.StreamModule:
				typeColor = infoColor
			case modules.SenseModule:
				typeColor = warnColor
			case modules.ExploitModule:
				typeColor = errorColor
			}

			fmt.Printf("%s %s\n", typeColor.Sprintf("[%s]", strings.ToUpper(string(moduleType))), grayColor.Sprint("─────────────────────────────────"))

			for _, mod := range byType[moduleType] {
				fmt.Printf("  %s  %s\n",
					cmdColor.Sprintf("%-20s", mod.Name()),
					mod.Description())
			}
			fmt.Println()
		}
	},
}

var moduleInfoCmd = &cobra.Command{
	Use:   "info <module>",
	Short: "Show detailed information about a module",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Load modules to provide completion
			_ = modules.LoadBuiltins(nil)
			var names []string
			for _, mod := range modules.List() {
				names = append(names, mod.Name())
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(_ *cobra.Command, args []string) {
		// Load all modules
		if err := modules.LoadBuiltins(nil); err != nil {
			errorColor.Printf("[-] Failed to load modules: %v\n", err)
			return
		}

		moduleName := args[0]
		module, err := modules.Get(moduleName)
		if err != nil {
			errorColor.Printf("[-] Module not found: %s\n", moduleName)
			return
		}

		info := module.Info()

		// Module header
		fmt.Printf("\n%s %s\n", successColor.Sprint("Module:"), cmdColor.Sprint(module.Name()))
		fmt.Printf("%s %s\n", successColor.Sprint("Type:"), string(module.Type()))
		fmt.Println(strings.Repeat("─", 60))

		// Description
		fmt.Printf("\n%s\n%s\n", infoColor.Sprint("Description:"), module.Description())

		// Module info
		if info != nil {
			if info.Author != "" {
				fmt.Printf("\n%s %s\n", infoColor.Sprint("Author:"), info.Author)
			}
			if info.Version != "" {
				fmt.Printf("%s %s\n", infoColor.Sprint("Version:"), info.Version)
			}

			// Tags
			if len(info.Tags) > 0 {
				fmt.Printf("\n%s\n", infoColor.Sprint("Tags:"))
				for _, tag := range info.Tags {
					fmt.Printf("  • %s\n", tag)
				}
			}

			// References
			if len(info.References) > 0 {
				fmt.Printf("\n%s\n", infoColor.Sprint("References:"))
				for _, ref := range info.References {
					fmt.Printf("  • %s\n", ref)
				}
			}
		}

		// Options
		options := module.Options()
		if len(options) > 0 {
			fmt.Printf("\n%s\n", infoColor.Sprint("Options:"))
			fmt.Println(strings.Repeat("─", 60))

			// Sort options for consistent display
			var optNames []string
			for name := range options {
				optNames = append(optNames, name)
			}
			sort.Strings(optNames)

			for _, name := range optNames {
				opt := options[name]
				required := ""
				if opt.Required {
					required = errorColor.Sprint(" (required)")
				}

				fmt.Printf("\n  %s%s\n", cmdColor.Sprint(name), required)
				fmt.Printf("    %s\n", opt.Description)
				fmt.Printf("    Type: %s", opt.Type)
				if opt.Default != nil && opt.Default != "" {
					fmt.Printf("  Default: %v", opt.Default)
				}
				fmt.Println()
			}
		}
	},
}

var moduleSearchCmd = &cobra.Command{
	Use:   "search <term>",
	Short: "Search for modules",
	Long:  `Search for modules by name, description, or tags.`,
	Args:  cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		// Load all modules
		if err := modules.LoadBuiltins(nil); err != nil {
			errorColor.Printf("[-] Failed to load modules: %v\n", err)
			return
		}

		term := args[0]
		matches := modules.Search(term)

		if len(matches) == 0 {
			fmt.Printf("%s No modules found matching '%s'\n", warnColor.Sprint("[!]"), term)
			return
		}

		fmt.Printf("%s Found %d modules matching '%s':\n\n",
			successColor.Sprint("[+]"), len(matches), term)

		for _, mod := range matches {
			fmt.Printf("  %s  %s  %s\n",
				cmdColor.Sprintf("%-20s", mod.Name()),
				grayColor.Sprintf("[%s]", mod.Type()),
				mod.Description())
		}
	},
}

var moduleUseCmd = &cobra.Command{
	Use:   "use <module>",
	Short: "Load a module for interactive use",
	Long:  `Load a module and enter interactive mode to configure and run it.`,
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Load modules to provide completion
			_ = modules.LoadBuiltins(nil)
			var names []string
			for _, mod := range modules.List() {
				names = append(names, mod.Name())
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(_ *cobra.Command, args []string) {
		// Load all modules
		if err := modules.LoadBuiltins(nil); err != nil {
			errorColor.Printf("[-] Failed to load modules: %v\n", err)
			return
		}

		moduleName := args[0]
		module, err := modules.Get(moduleName)
		if err != nil {
			errorColor.Printf("[-] Module not found: %s\n", moduleName)
			return
		}

		fmt.Printf("%s Loaded module: %s\n", successColor.Sprint("[+]"), cmdColor.Sprint(module.Name()))
		fmt.Printf("%s %s\n\n", infoColor.Sprint("[*]"), module.Description())

		// TODO: Enter interactive module configuration mode
		// For now, just show the options
		fmt.Println("Module options:")
		options := module.Options()
		for name, opt := range options {
			required := ""
			if opt.Required {
				required = errorColor.Sprint(" *")
			}
			fmt.Printf("  %s%s = %v\n", name, required, opt.Default)
		}

		fmt.Printf("\n%s Interactive module mode not yet implemented\n", warnColor.Sprint("[!]"))
		fmt.Println("Use the direct command instead:")
		fmt.Printf("  strigoi %s <target>\n", module.Name())
	},
}

func init() {
	// Add module command to root
	rootCmd.AddCommand(moduleCmd)

	// Add subcommands
	moduleCmd.AddCommand(moduleListCmd)
	moduleCmd.AddCommand(moduleInfoCmd)
	moduleCmd.AddCommand(moduleSearchCmd)
	moduleCmd.AddCommand(moduleUseCmd)
}
