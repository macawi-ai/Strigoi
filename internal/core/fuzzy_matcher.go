package core

import (
	"sort"
	"strings"
)

// FuzzyMatch represents a fuzzy match result
type FuzzyMatch struct {
	Command  string
	Path     []string
	Score    float64
	Distance int
}

// FuzzyMatcher provides fuzzy command matching
type FuzzyMatcher struct {
	rootNode *CommandNode
}

// NewFuzzyMatcher creates a new fuzzy matcher
func NewFuzzyMatcher(root *CommandNode) *FuzzyMatcher {
	return &FuzzyMatcher{
		rootNode: root,
	}
}

// FindMatches finds fuzzy matches for the input
func (fm *FuzzyMatcher) FindMatches(input string, maxResults int) []FuzzyMatch {
	input = strings.ToLower(input)
	matches := []FuzzyMatch{}
	
	// Collect all commands
	fm.walkCommands(fm.rootNode, []string{}, func(path []string, node *CommandNode) {
		if node.Hidden {
			return
		}
		
		// Create full command string
		fullCommand := strings.Join(path, "/")
		
		// Calculate match score
		score := fm.calculateScore(input, fullCommand, path)
		if score > 0 {
			matches = append(matches, FuzzyMatch{
				Command:  fullCommand,
				Path:     path,
				Score:    score,
				Distance: levenshteinDistance(input, fullCommand),
			})
		}
	})
	
	// Sort by score (descending) and distance (ascending)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].Distance < matches[j].Distance
	})
	
	// Limit results
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}
	
	return matches
}

// calculateScore calculates the fuzzy match score
func (fm *FuzzyMatcher) calculateScore(input, command string, path []string) float64 {
	input = strings.ToLower(input)
	command = strings.ToLower(command)
	
	// Exact match
	if input == command {
		return 1.0
	}
	
	// Prefix match
	if strings.HasPrefix(command, input) {
		return 0.9 - (0.1 * float64(len(command)-len(input)) / float64(len(command)))
	}
	
	// Contains match
	if strings.Contains(command, input) {
		position := float64(strings.Index(command, input)) / float64(len(command))
		return 0.7 - (0.2 * position)
	}
	
	// Check individual path components
	for _, component := range path {
		component = strings.ToLower(component)
		if input == component {
			return 0.8
		}
		if strings.HasPrefix(component, input) {
			return 0.6
		}
	}
	
	// Fuzzy character matching
	score := fm.fuzzyCharacterMatch(input, command)
	if score > 0.3 {
		return score * 0.5
	}
	
	return 0
}

// fuzzyCharacterMatch performs character-by-character fuzzy matching
func (fm *FuzzyMatcher) fuzzyCharacterMatch(pattern, text string) float64 {
	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)
	
	if len(pattern) == 0 {
		return 1.0
	}
	
	if len(pattern) > len(text) {
		return 0
	}
	
	patternIdx := 0
	textIdx := 0
	matches := 0
	
	for patternIdx < len(pattern) && textIdx < len(text) {
		if pattern[patternIdx] == text[textIdx] {
			matches++
			patternIdx++
		}
		textIdx++
	}
	
	if patternIdx == len(pattern) {
		// All pattern characters found
		return float64(matches) / float64(len(text))
	}
	
	return 0
}

// walkCommands walks the command tree
func (fm *FuzzyMatcher) walkCommands(node *CommandNode, path []string, fn func([]string, *CommandNode)) {
	// Process current node if it has a handler
	if node.Handler != nil && len(path) > 0 {
		fn(path, node)
	}
	
	// Recurse into children
	for name, child := range node.Children {
		childPath := append(path, name)
		fm.walkCommands(child, childPath, fn)
	}
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}
	
	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}
	
	// Initialize first column and row
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}
	
	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}
	
	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// SuggestCommand suggests the best matching command
func (fm *FuzzyMatcher) SuggestCommand(input string) (string, float64) {
	matches := fm.FindMatches(input, 1)
	if len(matches) > 0 && matches[0].Score > 0.5 {
		return matches[0].Command, matches[0].Score
	}
	return "", 0
}