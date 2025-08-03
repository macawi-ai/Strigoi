package core

import (
	"context"
	"fmt"
	"time"
)

// BaseModule provides common functionality for attack modules
type BaseModule struct {
	Name        string
	Description string
	Author      string
	Version     string
	Type        ModuleType
	RiskLevel   Severity
	options     map[string]*ModuleOption
}

// ModuleExecutor is the interface that modules must implement
type ModuleExecutor interface {
	Execute(ctx context.Context, session *Session) (*Result, error)
}

// Result represents the result of a module execution
type Result struct {
	Findings []Finding              `json:"findings"`
	Metrics  map[string]interface{} `json:"metrics"`
}

// Finding represents a security finding
type Finding struct {
	Title       string                 `json:"title"`
	Severity    Severity               `json:"severity"`
	Confidence  Confidence             `json:"confidence"`
	Description string                 `json:"description"`
	Evidence    map[string]interface{} `json:"evidence,omitempty"`
	Mitigation  string                 `json:"mitigation,omitempty"`
}

// Confidence level for findings
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// AttackModule wraps a module that implements ModuleExecutor
type AttackModule struct {
	BaseModule
	executor ModuleExecutor
}

// NewAttackModule creates a wrapper for attack modules
func NewAttackModule(base BaseModule, executor ModuleExecutor) *AttackModule {
	return &AttackModule{
		BaseModule: base,
		executor:   executor,
	}
}

// AddOption adds an option to the module
func (b *BaseModule) AddOption(name string, defaultValue interface{}, required bool, description string) {
	if b.options == nil {
		b.options = make(map[string]*ModuleOption)
	}
	
	b.options[name] = &ModuleOption{
		Name:        name,
		Value:       defaultValue,
		Required:    required,
		Description: description,
		Default:     defaultValue,
		Type:        getTypeString(defaultValue),
	}
}

// GetOption retrieves an option value as string
func (b *BaseModule) GetOption(name string) string {
	if opt, exists := b.options[name]; exists {
		return fmt.Sprintf("%v", opt.Value)
	}
	return ""
}

// Name returns the module name
func (m *AttackModule) Name() string {
	return m.BaseModule.Name
}

// Description returns the module description
func (m *AttackModule) Description() string {
	return m.BaseModule.Description
}

// Type returns the module type
func (m *AttackModule) Type() ModuleType {
	return ModuleTypeAttack
}

// Options returns module options
func (m *AttackModule) Options() map[string]*ModuleOption {
	return m.BaseModule.options
}

// SetOption sets a module option
func (m *AttackModule) SetOption(name, value string) error {
	opt, exists := m.BaseModule.options[name]
	if !exists {
		return fmt.Errorf("unknown option: %s", name)
	}

	// Type conversion based on option type
	switch opt.Type {
	case "int":
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err != nil {
			return fmt.Errorf("invalid integer value: %s", value)
		}
		opt.Value = intVal
	case "bool":
		boolVal := value == "true" || value == "1" || value == "yes"
		opt.Value = boolVal
	default:
		opt.Value = value
	}

	return nil
}

// ValidateOptions validates all required options are set
func (m *AttackModule) ValidateOptions() error {
	for name, opt := range m.BaseModule.options {
		if opt.Required && (opt.Value == nil || opt.Value == "") {
			return fmt.Errorf("required option %s not set", name)
		}
	}
	return nil
}

// Run executes the module
func (m *AttackModule) Run() (*ModuleResult, error) {
	// Validate options first
	if err := m.ValidateOptions(); err != nil {
		return nil, err
	}

	// Create execution context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create a session for the module
	session := &Session{
		ID:        generateAttackID(),
		Module:    m.Name(),
		Target:    m.GetOption("TARGET"),
		Status:    "running",
		StartTime: time.Now(),
		Options:   make(map[string]interface{}),
	}

	// Copy options to session
	for name, opt := range m.BaseModule.options {
		session.Options[name] = opt.Value
	}

	// Execute the module
	startTime := time.Now()
	result, err := m.executor.Execute(ctx, session)
	duration := time.Since(startTime)

	if err != nil {
		return nil, err
	}

	// Convert Result to ModuleResult
	moduleResult := &ModuleResult{
		Success:  true,
		Duration: duration,
		Metadata: result.Metrics,
	}

	// Convert findings
	for _, finding := range result.Findings {
		securityFinding := SecurityFinding{
			ID:          generateAttackID(),
			Title:       finding.Title,
			Description: finding.Description,
			Severity:    finding.Severity,
			CVSSScore:   severityToScore(finding.Severity),
		}

		// Add evidence
		if finding.Evidence != nil {
			securityFinding.Evidence = []Evidence{{
				Type: "data",
				Data: finding.Evidence,
			}}
		}

		// Add mitigation
		if finding.Mitigation != "" {
			securityFinding.Remediation = &Remediation{
				Description: finding.Mitigation,
			}
		}

		moduleResult.Findings = append(moduleResult.Findings, securityFinding)
	}

	// Generate summary
	moduleResult.Summary = &FindingSummary{
		Total:    len(moduleResult.Findings),
		Critical: countBySeverity(moduleResult.Findings, Critical),
		High:     countBySeverity(moduleResult.Findings, High),
		Medium:   countBySeverity(moduleResult.Findings, Medium),
		Low:      countBySeverity(moduleResult.Findings, Low),
		Info:     countBySeverity(moduleResult.Findings, Info),
	}

	return moduleResult, nil
}

// Check returns whether the module can run
func (m *AttackModule) Check() bool {
	return m.ValidateOptions() == nil
}

// Info returns module information
func (m *AttackModule) Info() *ModuleInfo {
	return &ModuleInfo{
		Name:        m.BaseModule.Name,
		Version:     m.BaseModule.Version,
		Author:      m.BaseModule.Author,
		Description: m.BaseModule.Description,
	}
}

// Helper functions

func getTypeString(v interface{}) string {
	switch v.(type) {
	case int, int32, int64:
		return "int"
	case bool:
		return "bool"
	default:
		return "string"
	}
}

func severityToScore(sev Severity) float64 {
	switch sev {
	case Critical:
		return 9.0
	case High:
		return 7.5
	case Medium:
		return 5.0
	case Low:
		return 3.0
	default:
		return 0.0
	}
}

func countBySeverity(findings []SecurityFinding, sev Severity) int {
	count := 0
	for _, f := range findings {
		if f.Severity == sev {
			count++
		}
	}
	return count
}

func generateAttackID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

