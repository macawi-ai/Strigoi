# Cobra Migration Example for Strigoi

## Current Structure → Cobra Mapping

### Before (Current)
```go
// Command tree with manual navigation
rootCommand := &CommandNode{
    Children: map[string]*CommandNode{
        "probe": {
            Children: map[string]*CommandNode{
                "north": {Handler: probeNorth},
                "south": {Handler: probeSouth},
            },
        },
        "stream": {
            Children: map[string]*CommandNode{
                "tap": {Handler: streamTap},
            },
        },
    },
}
```

### After (Cobra)
```go
// cmd/root.go
package cmd

import (
    "fmt"
    "os"
    "github.com/spf13/cobra"
    "github.com/fatih/color"
)

var rootCmd = &cobra.Command{
    Use:   "strigoi",
    Short: "Advanced Security Validation Platform",
    Long: coloredBanner + `
⚠️  Authorized use only - WHITE HAT SECURITY TESTING`,
    PersistentPreRun: func(cmd *cobra.Command, args []string) {
        // Initialize framework, logger, etc.
    },
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func init() {
    // Set custom help template with colors
    rootCmd.SetHelpTemplate(coloredHelpTemplate)
    
    // Add all subcommands
    rootCmd.AddCommand(probeCmd)
    rootCmd.AddCommand(streamCmd)
    rootCmd.AddCommand(senseCmd)
    rootCmd.AddCommand(stateCmd)
    
    // Global commands available everywhere
    rootCmd.PersistentFlags().Bool("help", false, "Show help")
}
```

### Probe Command with Subcommands
```go
// cmd/probe.go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/macawi-ai/strigoi/internal/core"
)

var probeCmd = &cobra.Command{
    Use:   "probe",
    Short: "Discovery and reconnaissance tools",
    Long:  `Probe in cardinal directions to discover attack surfaces`,
}

var probeNorthCmd = &cobra.Command{
    Use:   "north [target]",
    Short: "Probe north direction (endpoints)",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // Get framework from context
        framework := cmd.Context().Value("framework").(*core.Framework)
        return framework.ProbeNorth(args)
    },
    ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        // Custom completion for targets
        return []string{"localhost", "api.example.com"}, cobra.ShellCompDirectiveNoFileComp
    },
}

func init() {
    // Add subcommands
    probeCmd.AddCommand(probeNorthCmd)
    probeCmd.AddCommand(probeSouthCmd)
    probeCmd.AddCommand(probeEastCmd)
    probeCmd.AddCommand(probeWestCmd)
    probeCmd.AddCommand(probeAllCmd)
    
    // Probe-specific flags
    probeCmd.PersistentFlags().String("output", "json", "Output format (json, yaml, table)")
}
```

### Stream Command with Context-Aware Completion
```go
// cmd/stream.go
package cmd

import (
    "github.com/spf13/cobra"
)

var streamTapCmd = &cobra.Command{
    Use:   "tap <pid|name>",
    Short: "Monitor process STDIO in real-time",
    Args:  cobra.ExactArgs(1),
    RunE:  runStreamTap,
    ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        // Return running processes for completion
        return getRunningProcesses(toComplete), cobra.ShellCompDirectiveNoFileComp
    },
}

// Example: cd-like navigation (if we want to keep it)
var cdCmd = &cobra.Command{
    Use:   "cd [directory]",
    Short: "Change context (optional, for navigation feel)",
    Args:  cobra.MaximumNArgs(1),
    ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        // Only return "directories" (command groups)
        return []string{"probe", "stream", "sense", "state"}, cobra.ShellCompDirectiveNoFileComp
    },
    RunE: func(cmd *cobra.Command, args []string) error {
        // Could set context or just print current location
        fmt.Printf("Current context: %s\n", args[0])
        return nil
    },
}
```

### Preserving Color Output
```go
// cmd/colors.go
package cmd

import (
    "github.com/fatih/color"
)

var (
    dirColor   = color.New(color.FgBlue, color.Bold)
    cmdColor   = color.New(color.FgGreen)
    utilColor  = color.New(color.FgHiWhite)
    aliasColor = color.New(color.FgCyan)
)

// Custom help template with colors
const coloredHelpTemplate = `{{.Long}}

{{if .HasAvailableSubCommands}}` + dirColor.Sprint("Directories:") + `
{{range .Commands}}{{if (and .IsAvailableCommand (not .Hidden))}}  ` +
    dirColor.Sprint("{{rpad .Name .NamePadding}}") + `  {{.Short}}{{end}}{{end}}{{end}}

{{if .HasAvailableLocalFlags}}` + cmdColor.Sprint("Commands:") + `
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`
```

### Generating Completions
```go
// cmd/completion.go
package cmd

import (
    "os"
    "github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
    Use:   "completion [bash|zsh|fish|powershell]",
    Short: "Generate completion script",
    Long: `To load completions:

Bash:
  $ source <(strigoi completion bash)
  # To load completions for each session, execute once:
  $ strigoi completion bash > /etc/bash_completion.d/strigoi

Zsh:
  $ source <(strigoi completion zsh)
  # To load completions for each session, execute once:
  $ strigoi completion zsh > "${fpath[1]}/_strigoi"`,
    DisableFlagsInUseLine: true,
    ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
    Args:                  cobra.ExactValidArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        switch args[0] {
        case "bash":
            cmd.Root().GenBashCompletion(os.Stdout)
        case "zsh":
            cmd.Root().GenZshCompletion(os.Stdout)
        case "fish":
            cmd.Root().GenFishCompletion(os.Stdout, true)
        case "powershell":
            cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
        }
    },
}
```

## Key Benefits Demonstrated

1. **Multi-word Completion**: `strigoi probe north <TAB>` works naturally
2. **Context Awareness**: ValidArgsFunction provides context-specific completions
3. **Built-in Help**: Cobra generates help automatically
4. **Color Preservation**: Custom templates maintain our visual design
5. **Shell Support**: Bash, zsh, fish, PowerShell out of the box

## Testing the Migration

```bash
# Build with Cobra
go build -o strigoi-cobra ./cmd/strigoi

# Generate completions
./strigoi-cobra completion bash > strigoi-completion.bash
source strigoi-completion.bash

# Test multi-word completion
./strigoi-cobra probe n<TAB>      # Completes to "north"
./strigoi-cobra probe north <TAB>  # Shows target options
./strigoi-cobra stream t<TAB>      # Completes to "tap"
./strigoi-cobra stream tap <TAB>   # Shows process list
```

This approach gives us professional-grade completion while maintaining our unique design philosophy.