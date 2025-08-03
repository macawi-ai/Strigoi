package core

import "time"

// Module is the interface that all Strigoi modules must implement
type Module interface {
	Name() string
	Description() string
	Type() ModuleType
	Options() map[string]*ModuleOption
	SetOption(name, value string) error
	ValidateOptions() error
	Run() (*ModuleResult, error)
	Check() bool
	Info() *ModuleInfo
}

// ModuleType represents the type of module
type ModuleType string

const (
	ModuleTypeAttack    ModuleType = "attack"
	ModuleTypeScanner   ModuleType = "scanner"
	ModuleTypeDiscovery ModuleType = "discovery"
	ModuleTypeExploit   ModuleType = "exploit"
	ModuleTypePayload   ModuleType = "payload"
	ModuleTypePost      ModuleType = "post"
	ModuleTypeAuxiliary ModuleType = "auxiliary"
	NetworkScanning     ModuleType = "network"
)

// ModuleOption represents a configurable option for a module
type ModuleOption struct {
	Name        string      `json:"name"`
	Value       interface{} `json:"value"`
	Required    bool        `json:"required"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
}

// ModuleResult represents the result of a module execution
type ModuleResult struct {
	Success  bool                   `json:"success"`
	Findings []SecurityFinding      `json:"findings"`
	Summary  *FindingSummary        `json:"summary"`
	Duration time.Duration          `json:"duration"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SecurityFinding represents a security vulnerability or issue
type SecurityFinding struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    Severity               `json:"severity"`
	CVSSScore   float64                `json:"cvss_score,omitempty"`
	Evidence    []Evidence             `json:"evidence,omitempty"`
	Remediation *Remediation           `json:"remediation,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Severity levels for findings
type Severity string

const (
	Critical Severity = "critical"
	High     Severity = "high"
	Medium   Severity = "medium"
	Low      Severity = "low"
	Info     Severity = "info"
)

// FindingSummary provides a summary of findings by severity
type FindingSummary struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
}

// Evidence represents proof of a finding
type Evidence struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// Remediation provides guidance on fixing a finding
type Remediation struct {
	Description string   `json:"description"`
	References  []string `json:"references,omitempty"`
}

// ModuleInfo contains metadata about a module
type ModuleInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	References  []string `json:"references,omitempty"`
	Targets     []string `json:"targets,omitempty"`
}

// Session represents an active module session
type Session struct {
	ID        string                 `json:"id"`
	Module    string                 `json:"module"`
	Target    string                 `json:"target"`
	Status    string                 `json:"status"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time,omitempty"`
	Options   map[string]interface{} `json:"options"`
}

// Job represents a background job
type Job struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"`
	Module   string    `json:"module"`
	Status   string    `json:"status"`
	Progress int       `json:"progress"`
	Started  time.Time `json:"started"`
}

// Config represents framework configuration
type Config struct {
	LogLevel     string `json:"log_level"`
	LogFile      string `json:"log_file"`
	CheckOnStart bool   `json:"check_on_start"`
	UseConsoleV2 bool   `json:"use_console_v2"`
}


// PackageLoader interface for loading protocol packages
type PackageLoader interface {
	LoadPackages() error
	GenerateModules() ([]Module, error)
}