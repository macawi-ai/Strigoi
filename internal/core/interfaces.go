package core

import (
	"context"
	"io"
	"time"
)

// Logger interface for framework logging
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Success(format string, args ...interface{})
	Fatal(format string, args ...interface{})
}

// PolicyEngine interface for policy evaluation
type PolicyEngine interface {
	CheckPermissions(module Module, target Target) error
	FilterFindings(findings []SecurityFinding, target Target) []SecurityFinding
	EvaluateRisk(findings []SecurityFinding) RiskAssessment
}

// Reporter interface for report generation
type Reporter interface {
	Generate(results []*ModuleResult, format ReportFormat) ([]byte, error)
	GenerateStream(results []*ModuleResult, format ReportFormat, writer io.Writer) error
}

// ReportFormat represents output format options
type ReportFormat string

const (
	ReportFormatText     ReportFormat = "text"
	ReportFormatJSON     ReportFormat = "json"
	ReportFormatHTML     ReportFormat = "html"
	ReportFormatMarkdown ReportFormat = "markdown"
	ReportFormatSARIF    ReportFormat = "sarif"
	ReportFormatCSV      ReportFormat = "csv"
)

// RiskAssessment represents policy engine risk evaluation
type RiskAssessment struct {
	Score          int    `json:"score"`
	Level          string `json:"level"`
	Recommendation string `json:"recommendation"`
	Justification  string `json:"justification,omitempty"`
}

// Target represents a security assessment target
type Target struct {
	URL         string                 `json:"url,omitempty"`
	Host        string                 `json:"host,omitempty"`
	Port        int                    `json:"port,omitempty"`
	Protocol    string                 `json:"protocol,omitempty"`
	Path        string                 `json:"path,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Scanner interface for modules that perform scanning
type Scanner interface {
	Module
	Scan(ctx context.Context, target Target) (*ScanResult, error)
}

// ScanResult contains scanning results
type ScanResult struct {
	Target      Target            `json:"target"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Status      string            `json:"status"`
	Findings    []SecurityFinding `json:"findings,omitempty"`
	RawData     interface{}       `json:"raw_data,omitempty"`
	Error       string            `json:"error,omitempty"`
}