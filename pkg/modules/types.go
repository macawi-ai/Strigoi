package modules

import (
	"time"
)

// ModuleType represents the type of module.
type ModuleType string

const (
	// ProbeModule is for discovery and reconnaissance.
	ProbeModule ModuleType = "probe"
	// StreamModule is for STDIO monitoring.
	StreamModule ModuleType = "stream"
	// SenseModule is for passive monitoring.
	SenseModule ModuleType = "sense"
	// ExploitModule is for vulnerability exploitation.
	ExploitModule ModuleType = "exploit"
)

// Module is the interface all modules must implement.
type Module interface {
	// Core identification
	Name() string
	Description() string
	Type() ModuleType

	// Options management
	Options() map[string]*ModuleOption
	SetOption(name, value string) error
	ValidateOptions() error

	// Execution
	Run() (*ModuleResult, error)
	Check() bool

	// Metadata
	Info() *ModuleInfo
}

// ModuleOption represents a configurable option for a module.
type ModuleOption struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Type        string      `json:"type"` // string, int, bool, list
	Default     interface{} `json:"default,omitempty"`
	Value       interface{} `json:"value,omitempty"`
}

// ModuleResult represents the result of running a module.
type ModuleResult struct {
	Module    string                 `json:"module"`
	Status    string                 `json:"status"` // running, completed, failed
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Error     string                 `json:"error,omitempty"`
}

// ModuleInfo contains metadata about a module.
type ModuleInfo struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Author      string                   `json:"author"`
	Version     string                   `json:"version"`
	Type        ModuleType               `json:"type"`
	Options     map[string]*ModuleOption `json:"options"`
	References  []string                 `json:"references,omitempty"`
	Tags        []string                 `json:"tags,omitempty"`
}
