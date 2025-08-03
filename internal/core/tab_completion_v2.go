package core

import (
	"github.com/chzyer/readline"
)

// BuildCompleterFromTree builds a readline completer from command tree
func BuildCompleterFromTree(root *CommandNode) *readline.PrefixCompleter {
	return buildNodeCompleter(root)
}

// buildNodeCompleter recursively builds completers from a command node
func buildNodeCompleter(node *CommandNode) *readline.PrefixCompleter {
	items := []readline.PrefixCompleterInterface{}
	
	// Add child commands
	for name, child := range node.Children {
		if child.Hidden {
			continue
		}
		
		if len(child.Children) > 0 {
			// Has subcommands - build recursively
			subItems := []readline.PrefixCompleterInterface{}
			for subName, subChild := range child.Children {
				if !subChild.Hidden {
					subItems = append(subItems, buildLeafCompleter(subChild, subName))
				}
			}
			items = append(items, readline.PcItem(name+"/", subItems...))
		} else {
			// Leaf command
			items = append(items, buildLeafCompleter(child, name))
		}
	}
	
	// Add global commands (including navigation)
	items = append(items,
		readline.PcItem("help"),
		readline.PcItem("exit"),
		readline.PcItem("clear"),
		readline.PcItem("alias"),
		readline.PcItem("cd"),
		readline.PcItem("ls"),
		readline.PcItem("pwd"),
		readline.PcItem("jobs"),
		readline.PcItem("support"),
		readline.PcItem("report"),
		readline.PcItem("respond"),
	)
	
	return readline.NewPrefixCompleter(items...)
}

// buildLeafCompleter builds completer for a leaf command
func buildLeafCompleter(node *CommandNode, name string) readline.PrefixCompleterInterface {
	flagItems := []readline.PrefixCompleterInterface{}
	
	// Add flags
	for _, flag := range node.Flags {
		if flag.Name != "" {
			flagItems = append(flagItems, readline.PcItem("--"+flag.Name))
		}
		if flag.Short != "" {
			flagItems = append(flagItems, readline.PcItem("-"+flag.Short))
		}
	}
	
	return readline.PcItem(name, flagItems...)
}