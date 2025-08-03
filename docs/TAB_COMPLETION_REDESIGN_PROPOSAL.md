# TAB Completion Redesign Proposal for Strigoi

## Executive Summary

After testing revealed that our current readline-based TAB completion fails on multi-word commands (`cd probe<TAB>`), we collaborated with Sister Gemini to identify the best path forward. The recommendation is to **migrate to Cobra/pflag** for robust, maintainable completion.

## Current Issues

1. **Multi-word Completion Broken**: `cd probe<TAB>` fails to complete
2. **Over-engineered Solution**: Pre-computed caches add complexity without solving core issues
3. **Readline Limitations**: The library wasn't designed for our hierarchical, context-aware needs

## Recommended Solution: Cobra Migration

### Why Cobra?

1. **Built-in Completion**: Native support for bash, zsh, fish, PowerShell
2. **Battle-tested**: Used by kubectl, docker, gh CLI, helm
3. **Context-aware**: Handles hierarchical commands naturally
4. **Multi-word Support**: Works out of the box
5. **Maintainable**: Less custom code, more framework support

### Implementation Plan

#### Phase 1: Cobra Structure
```go
// cmd/root.go
var rootCmd = &cobra.Command{
    Use:   "strigoi",
    Short: "Advanced Security Validation Platform",
}

// cmd/probe.go
var probeCmd = &cobra.Command{
    Use:   "probe",
    Short: "Discovery and reconnaissance tools",
}

var probeNorthCmd = &cobra.Command{
    Use:   "north",
    Short: "Probe north direction (endpoints)",
    Run: func(cmd *cobra.Command, args []string) {
        // Implementation
    },
}
```

#### Phase 2: Preserve Our Design

1. **Keep Color Coding**: Implement custom help formatter
2. **Maintain Navigation Feel**: Use subcommands as "directories"
3. **Global Commands**: Available at all levels (help, exit, clear)
4. **Aliases**: Cobra supports aliases natively

#### Phase 3: Completion Scripts

```bash
# Generate completion
strigoi completion bash > /etc/bash_completion.d/strigoi

# User enables with:
source <(strigoi completion bash)
```

### What We Keep

- ✅ Hierarchical structure
- ✅ Color-coded output
- ✅ Bash-like navigation feel
- ✅ Our command organization
- ✅ Interface stability tests

### What Changes

- 🔄 Command routing (Cobra handles it)
- 🔄 Completion mechanism (native, not custom)
- 🔄 Help generation (Cobra templates)
- 🔄 Flag parsing (pflag library)

## Alternative Considered

**Fixing Readline**: Gemini and I agree this would be "patching" - risky and unmaintainable.

## Migration Strategy

1. **Proof of Concept**: Implement core commands in Cobra
2. **Side-by-side Testing**: Run both versions
3. **Feature Parity**: Ensure all commands work
4. **User Testing**: Get feedback from security professionals
5. **Clean Switch**: Remove old implementation

## Benefits

1. **Reliability**: Multi-word completion works consistently
2. **Performance**: No pre-computation needed
3. **Maintainability**: Less custom code
4. **Future-proof**: Cobra is actively maintained
5. **Professional**: Same as kubectl, docker, etc.

## Example Implementation

```go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/fatih/color"
)

func init() {
    // Preserve our colors
    cobra.AddTemplateFunc("blue", color.New(color.FgBlue).SprintFunc())
    cobra.AddTemplateFunc("green", color.New(color.FgGreen).SprintFunc())
    
    // Custom help template with colors
    rootCmd.SetHelpTemplate(coloredHelpTemplate)
}

// Navigation-style commands
var cdCmd = &cobra.Command{
    Use:   "cd [directory]",
    Short: "Change to directory",
    ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        // Return only directories
        return getDirectories(), cobra.ShellCompDirectiveNoFileComp
    },
}
```

## Recommendation

**Proceed with Cobra migration**. It's the pragmatic choice that balances:
- User expectations (bash-like behavior)
- Maintainability (less custom code)
- Reliability (proven in production tools)
- Features (multi-word completion works)

The investment in migration will pay off with a more stable, professional tool that security professionals can rely on.

---

*Collaborative Analysis by Synth & Sister Gemini*