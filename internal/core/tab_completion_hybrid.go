package core

import (
	"sort"
	"strings"
	
	"github.com/chzyer/readline"
)

// CompleterCache stores pre-computed completers for each directory
type CompleterCache struct {
	completers map[string]*readline.PrefixCompleter
	globalCmds []string
	aliases    map[string]string
}

// NewCompleterCache creates and initializes the completer cache
func NewCompleterCache(root *CommandNode, aliasManager *AliasManager) *CompleterCache {
	cache := &CompleterCache{
		completers: make(map[string]*readline.PrefixCompleter),
		globalCmds: []string{"cd", "ls", "pwd", "exit", "help", "clear", "alias", "jobs"},
		aliases:    aliasManager.GetAllAliases(),
	}
	
	// Pre-compute completions for all directories
	cache.buildCompletionsRecursive(root, []string{})
	
	return cache
}

// buildCompletionsRecursive pre-computes completions for each directory node
func (cc *CompleterCache) buildCompletionsRecursive(node *CommandNode, path []string) {
	// Determine if this node is a directory
	isDirectory := len(node.Children) > 0 || node.Handler == nil
	node.IsDirectory = isDirectory
	
	// Build completion items for this node
	items := []readline.PrefixCompleterInterface{}
	
	// Add child nodes
	var childNames []string
	for name, child := range node.Children {
		if !child.Hidden {
			childNames = append(childNames, name)
		}
	}
	sort.Strings(childNames)
	
	for _, name := range childNames {
		child := node.Children[name]
		childIsDir := len(child.Children) > 0 || child.Handler == nil
		
		if childIsDir {
			// Directory - add with trailing slash
			items = append(items, readline.PcItem(name+"/"))
			// Recursively build completions for subdirectory
			newPath := append(append([]string{}, path...), name)
			cc.buildCompletionsRecursive(child, newPath)
		} else {
			// Command - add without slash
			items = append(items, readline.PcItem(name))
		}
	}
	
	// Add global commands
	for _, cmd := range cc.globalCmds {
		items = append(items, readline.PcItem(cmd))
	}
	
	// Store completer for this path
	pathStr := "/" + strings.Join(path, "/")
	if pathStr == "/" {
		pathStr = ""
	}
	cc.completers[pathStr] = readline.NewPrefixCompleter(items...)
}

// GetCompleter returns a context-aware completer function
func (cc *CompleterCache) GetCompleter(navigator *ContextNavigator) readline.AutoCompleter {
	return &contextAwareCompleter{
		cache:     cc,
		navigator: navigator,
	}
}

// contextAwareCompleter implements readline.AutoCompleter with context awareness
type contextAwareCompleter struct {
	cache     *CompleterCache
	navigator *ContextNavigator
}

// Do implements the readline.AutoCompleter interface
func (cac *contextAwareCompleter) Do(line []rune, pos int) ([][]rune, int) {
	// Get current path from navigator
	currentPath := cac.navigator.GetPathString()
	
	// Get the completer for current directory
	completer, exists := cac.cache.completers[currentPath]
	if !exists {
		// Fallback to root completer
		completer = cac.cache.completers[""]
	}
	
	// Get base completions
	completions, length := completer.Do(line, pos)
	
	// Special handling for cd command - only show directories
	lineStr := string(line)
	if strings.HasPrefix(lineStr, "cd ") {
		filtered := [][]rune{}
		for _, comp := range completions {
			compStr := string(comp)
			if strings.HasSuffix(compStr, "/") {
				filtered = append(filtered, comp)
			}
		}
		completions = filtered
	}
	
	// Apply alias expansion where needed
	expanded := [][]rune{}
	for _, comp := range completions {
		compStr := string(comp)
		// Check if this completion is an alias
		if expansion, isAlias := cac.cache.aliases[strings.TrimSuffix(compStr, "/")]; isAlias {
			// Add both the alias and its expansion
			expanded = append(expanded, comp)
			if !containsRune(completions, []rune(expansion)) {
				expanded = append(expanded, []rune(expansion))
			}
		} else {
			expanded = append(expanded, comp)
		}
	}
	
	return expanded, length
}

// Helper function to check if slice contains string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// Helper function to check if slice contains rune slice
func containsRune(slice [][]rune, target []rune) bool {
	targetStr := string(target)
	for _, s := range slice {
		if string(s) == targetStr {
			return true
		}
	}
	return false
}

// BuildHybridCompleter creates the new hybrid completer system
func BuildHybridCompleter(root *CommandNode, navigator *ContextNavigator, aliasManager *AliasManager) readline.AutoCompleter {
	cache := NewCompleterCache(root, aliasManager)
	return cache.GetCompleter(navigator)
}