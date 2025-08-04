package output

import (
	"time"
)

// OutputFormat represents the format for output display.
type OutputFormat string

const (
	FormatPretty   OutputFormat = "pretty"
	FormatJSON     OutputFormat = "json"
	FormatYAML     OutputFormat = "yaml"
	FormatMarkdown OutputFormat = "markdown"
)

// VerbosityLevel controls the amount of detail in output.
type VerbosityLevel int

const (
	VerbosityQuiet VerbosityLevel = iota
	VerbosityNormal
	VerbosityVerbose
	VerbosityDebug
)

// Severity represents the severity level of findings.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// StandardOutput is the common output structure for all modules.
type StandardOutput struct {
	Module       string                 `json:"module" yaml:"module"`
	Target       string                 `json:"target" yaml:"target"`
	Timestamp    time.Time              `json:"timestamp" yaml:"timestamp"`
	Duration     time.Duration          `json:"duration,omitempty" yaml:"duration,omitempty"`
	Summary      *Summary               `json:"summary,omitempty" yaml:"summary,omitempty"`
	Results      map[string]interface{} `json:"results" yaml:"results"`
	DeepAnalysis *DeepAnalysis          `json:"deep_analysis,omitempty" yaml:"deep_analysis,omitempty"`
	Errors       []Error                `json:"errors,omitempty" yaml:"errors,omitempty"`
}

// Summary provides a high-level overview of findings.
type Summary struct {
	TotalFindings   int              `json:"total_findings" yaml:"total_findings"`
	SeverityCounts  map[Severity]int `json:"severity_counts" yaml:"severity_counts"`
	Status          string           `json:"status" yaml:"status"`
	Recommendations []string         `json:"recommendations,omitempty" yaml:"recommendations,omitempty"`
}

// DeepAnalysis contains results from deep analysis mode.
type DeepAnalysis struct {
	Enabled  bool                        `json:"enabled" yaml:"enabled"`
	Duration time.Duration               `json:"duration" yaml:"duration"`
	Sections map[string]*AnalysisSection `json:"sections" yaml:"sections"`
}

// AnalysisSection represents a category of deep analysis findings.
type AnalysisSection struct {
	Title     string         `json:"title" yaml:"title"`
	Items     []AnalysisItem `json:"items" yaml:"items"`
	Summary   string         `json:"summary,omitempty" yaml:"summary,omitempty"`
	ItemCount int            `json:"item_count" yaml:"item_count"`
}

// AnalysisItem represents a single finding or observation.
type AnalysisItem struct {
	ID          string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Severity    Severity               `json:"severity" yaml:"severity"`
	Evidence    string                 `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	Remediation string                 `json:"remediation,omitempty" yaml:"remediation,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Error represents an error that occurred during analysis.
type Error struct {
	Phase   string `json:"phase" yaml:"phase"`
	Message string `json:"message" yaml:"message"`
	Details string `json:"details,omitempty" yaml:"details,omitempty"`
}

// FormatterOptions configures output formatting behavior.
type FormatterOptions struct {
	Format      OutputFormat
	Verbosity   VerbosityLevel
	ColorScheme *ColorScheme
	NoColor     bool
	Filters     []FilterFunc
}

// FilterFunc is a function that filters output items.
type FilterFunc func(item AnalysisItem) bool

// Formatter is the interface for output formatters.
type Formatter interface {
	Format(output StandardOutput, options FormatterOptions) (string, error)
}
