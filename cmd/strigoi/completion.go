package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for Strigoi.

To load completions:

Bash:
  $ source <(strigoi completion bash)

  # To load completions for each session, execute once:
  Linux:
    $ strigoi completion bash > /etc/bash_completion.d/strigoi
  macOS:
    $ strigoi completion bash > $(brew --prefix)/etc/bash_completion.d/strigoi

Zsh:
  $ source <(strigoi completion zsh)

  # To load completions for each session, execute once:
  $ strigoi completion zsh > "${fpath[1]}/_strigoi"

Fish:
  $ strigoi completion fish | source

  # To load completions for each session, execute once:
  $ strigoi completion fish > ~/.config/fish/completions/strigoi.fish

PowerShell:
  PS> strigoi completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> strigoi completion powershell > strigoi.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		switch args[0] {
		case "bash":
			err = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			err = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			err = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			err = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating completion: %v\n", err)
			os.Exit(1)
		}
	},
}
