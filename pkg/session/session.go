package session

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/macawi-ai/strigoi/pkg/modules"
)

// Version represents the session format version.
const Version = "1.0"

// Session represents a saved module configuration.
type Session struct {
	Version     string                 `json:"version"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Created     time.Time              `json:"created"`
	Modified    time.Time              `json:"modified"`
	Tags        []string               `json:"tags,omitempty"`
	Module      ModuleConfig           `json:"module"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ModuleConfig stores module-specific configuration.
type ModuleConfig struct {
	Name      string                 `json:"name"`
	Options   map[string]interface{} `json:"options"`
	Sensitive []string               `json:"sensitive,omitempty"` // List of sensitive option keys
}

// NewSession creates a new session from a module.
func NewSession(name, description string, module modules.Module) (*Session, error) {
	if name == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}

	if module == nil {
		return nil, fmt.Errorf("module cannot be nil")
	}

	// Extract module options
	options := make(map[string]interface{})
	for optName, opt := range module.Options() {
		if opt.Value != nil {
			options[optName] = opt.Value
		} else if opt.Default != nil {
			options[optName] = opt.Default
		}
	}

	// Identify sensitive options
	var sensitive []string
	for optName, opt := range module.Options() {
		// Mark as sensitive if it contains certain keywords
		if isSensitiveOption(optName, opt.Description) {
			sensitive = append(sensitive, optName)
		}
	}

	now := time.Now()
	return &Session{
		Version:     Version,
		Name:        name,
		Description: description,
		Created:     now,
		Modified:    now,
		Module: ModuleConfig{
			Name:      module.Name(),
			Options:   options,
			Sensitive: sensitive,
		},
		Metadata: make(map[string]interface{}),
	}, nil
}

// LoadIntoModule loads session configuration into a module.
func (s *Session) LoadIntoModule(module modules.Module) error {
	if module.Name() != s.Module.Name {
		return fmt.Errorf("module name mismatch: session has '%s', but module is '%s'",
			s.Module.Name, module.Name())
	}

	// Set options from session
	for name, value := range s.Module.Options {
		// Convert value to string for SetOption
		strValue := fmt.Sprintf("%v", value)
		if err := module.SetOption(name, strValue); err != nil {
			return fmt.Errorf("failed to set option %s: %w", name, err)
		}
	}

	return nil
}

// Validate checks if the session is valid.
func (s *Session) Validate() error {
	if s.Version != Version {
		return fmt.Errorf("unsupported session version: %s (expected %s)", s.Version, Version)
	}

	if s.Name == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	if s.Module.Name == "" {
		return fmt.Errorf("module name cannot be empty")
	}

	return nil
}

// Clone creates a deep copy of the session.
func (s *Session) Clone() *Session {
	// Marshal and unmarshal to create a deep copy
	data, _ := json.Marshal(s)
	var clone Session
	_ = json.Unmarshal(data, &clone)
	return &clone
}

// AddTag adds a tag to the session.
func (s *Session) AddTag(tag string) {
	// Check if tag already exists
	for _, t := range s.Tags {
		if t == tag {
			return
		}
	}
	s.Tags = append(s.Tags, tag)
	s.Modified = time.Now()
}

// RemoveTag removes a tag from the session.
func (s *Session) RemoveTag(tag string) {
	var tags []string
	for _, t := range s.Tags {
		if t != tag {
			tags = append(tags, t)
		}
	}
	s.Tags = tags
	s.Modified = time.Now()
}

// HasTag checks if the session has a specific tag.
func (s *Session) HasTag(tag string) bool {
	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// SetMetadata sets a metadata value.
func (s *Session) SetMetadata(key string, value interface{}) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}
	s.Metadata[key] = value
	s.Modified = time.Now()
}

// GetMetadata gets a metadata value.
func (s *Session) GetMetadata(key string) (interface{}, bool) {
	if s.Metadata == nil {
		return nil, false
	}
	value, exists := s.Metadata[key]
	return value, exists
}

// isSensitiveOption determines if an option should be considered sensitive.
func isSensitiveOption(name, description string) bool {
	sensitiveKeywords := []string{
		"password", "passwd", "pwd",
		"token", "key", "secret",
		"auth", "credential", "cred",
		"apikey", "api_key", "api-key",
		"private", "certificate", "cert",
	}

	lowName := fmt.Sprintf("%s %s", name, description)
	for _, keyword := range sensitiveKeywords {
		if containsIgnoreCase(lowName, keyword) {
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		containsIgnoreCaseAt(s, substr, 0)
}

// containsIgnoreCaseAt checks if a string contains a substring at any position.
func containsIgnoreCaseAt(s, substr string, start int) bool {
	if start+len(substr) > len(s) {
		return false
	}

	for i := start; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// toLower converts a byte to lowercase.
func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
