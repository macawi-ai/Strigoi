// Package core - State Manager for consciousness collaboration integration
// Bridges the First Protocol for Converged Life with Strigoi CLI operations
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/macawi-ai/strigoi/internal/state"
)

// StateManager manages consciousness collaboration state packages
// Integrates the First Protocol with Strigoi's probe-sense-respond cycle
type StateManager struct {
	logger         Logger
	currentPackage *state.HybridStatePackage
	packagesDir    string
	activeSession  string
	mu             sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager(logger Logger) *StateManager {
	paths := GetPaths()
	packagesDir := filepath.Join(paths.Home, "data", "assessments")
	
	return &StateManager{
		logger:      logger,
		packagesDir: packagesDir,
	}
}

// StartAssessment begins a new consciousness collaboration assessment
// Creates a new hybrid state package for the session
func (sm *StateManager) StartAssessment(title, description string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Generate assessment ID
	assessmentID := fmt.Sprintf("assessment_%d", time.Now().UnixNano())
	packagePath := filepath.Join(sm.packagesDir, assessmentID)
	
	// Create new hybrid state package
	pkg := state.NewHybridStatePackage(assessmentID, packagePath)
	
	// Set metadata
	pkg.Metadata.Metadata.Title = title
	pkg.Metadata.Metadata.Description = description
	pkg.Metadata.Metadata.Assessor = "strigoi-cli"
	pkg.Metadata.Metadata.Classification = "internal"
	
	// Initialize ethics (required for First Protocol)
	pkg.Metadata.Metadata.Ethics.ConsentObtained = true  // CLI user consent assumed
	pkg.Metadata.Metadata.Ethics.TargetAuthorized = true // User must authorize targets
	pkg.Metadata.Metadata.Ethics.Purpose = "AI security assessment via consciousness collaboration"
	
	sm.currentPackage = pkg
	sm.activeSession = assessmentID
	
	sm.logger.Info("Started new consciousness collaboration assessment: %s", assessmentID)
	return nil
}

// RecordProbeEvent captures probe/ command execution in the consciousness timeline
func (sm *StateManager) RecordProbeEvent(direction, actor string, input []byte, output []byte, duration time.Duration, status string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if sm.currentPackage == nil {
		return fmt.Errorf("no active assessment - start assessment first with 'state/new'")
	}
	
	// Create actor event
	event := &state.ActorEvent{
		EventId:         fmt.Sprintf("probe_%s_%d", direction, time.Now().UnixNano()),
		TimestampNs:     time.Now().UnixNano(),
		ActorName:       fmt.Sprintf("probe_%s_%s", direction, actor),
		ActorVersion:    "0.3.0",
		ActorDirection:  direction,
		InputData:       input,
		OutputData:      output,
		InputFormat:     "json",
		OutputFormat:    "json",
		DurationMs:      duration.Milliseconds(),
		Status:          sm.parseExecutionStatus(status),
		Transformations: []string{fmt.Sprintf("Probed %s direction for %s capabilities", direction, actor)},
	}
	
	return sm.currentPackage.AddEvent(event)
}

// RecordSenseEvent captures sense/ command execution in the consciousness timeline
func (sm *StateManager) RecordSenseEvent(layer, analyzer string, input []byte, output []byte, duration time.Duration, status string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if sm.currentPackage == nil {
		return fmt.Errorf("no active assessment - start assessment first with 'state/new'")
	}
	
	event := &state.ActorEvent{
		EventId:         fmt.Sprintf("sense_%s_%d", layer, time.Now().UnixNano()),
		TimestampNs:     time.Now().UnixNano(),
		ActorName:       fmt.Sprintf("sense_%s_%s", layer, analyzer),
		ActorVersion:    "0.3.0",
		ActorDirection:  "center", // Sense operates from center
		InputData:       input,
		OutputData:      output,
		InputFormat:     "json",
		OutputFormat:    "json",
		DurationMs:      duration.Milliseconds(),
		Status:          sm.parseExecutionStatus(status),
		Transformations: []string{fmt.Sprintf("Analyzed %s layer using %s", layer, analyzer)},
	}
	
	return sm.currentPackage.AddEvent(event)
}

// RecordFinding adds a security finding to the current assessment
func (sm *StateManager) RecordFinding(title, description, severity, discoveredBy string, evidence []byte) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if sm.currentPackage == nil {
		return fmt.Errorf("no active assessment - start assessment first with 'state/new'")
	}
	
	finding := &state.Finding{
		Id:            fmt.Sprintf("finding_%d", time.Now().UnixNano()),
		Title:         title,
		Description:   description,
		Severity:      sm.parseSeverity(severity),
		Confidence:    0.8, // Default confidence
		DiscoveredBy:  discoveredBy,
		Evidence:      evidence,
		EvidenceFormat: "json",
		Remediation:   "Review finding and implement appropriate security controls",
	}
	
	return sm.currentPackage.AddFinding(finding)
}

// RecordMultiLLMCollaboration captures consciousness collaboration between AI models
func (sm *StateManager) RecordMultiLLMCollaboration(modelName, role, contributionType string, contributionData []byte) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if sm.currentPackage == nil {
		return fmt.Errorf("no active assessment - start assessment first with 'state/new'")
	}
	
	event := &state.ActorEvent{
		EventId:       fmt.Sprintf("multi_llm_%s_%d", modelName, time.Now().UnixNano()),
		TimestampNs:   time.Now().UnixNano(),
		ActorName:     fmt.Sprintf("llm_%s", modelName),
		ActorVersion:  "collaborative",
		ActorDirection: "north", // LLMs are in the north
		InputData:     contributionData,
		OutputData:    contributionData,
		InputFormat:   "text",
		OutputFormat:  "text",
		Status:        state.ExecutionStatus_EXECUTION_STATUS_SUCCESS,
		LlmContributions: []*state.LLMContribution{
			{
				ModelName:        modelName,
				Role:            role,
				TimestampNs:     time.Now().UnixNano(),
				ContributionType: contributionType,
				ContributionData: contributionData,
			},
		},
		Transformations: []string{fmt.Sprintf("Multi-LLM collaboration: %s provided %s as %s", modelName, contributionType, role)},
	}
	
	return sm.currentPackage.AddEvent(event)
}

// SaveAssessment persists the current assessment to disk
func (sm *StateManager) SaveAssessment() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if sm.currentPackage == nil {
		return fmt.Errorf("no active assessment to save")
	}
	
	sm.logger.Info("Saving consciousness collaboration assessment...")
	
	if err := sm.currentPackage.Save(); err != nil {
		return fmt.Errorf("failed to save assessment: %w", err)
	}
	
	sm.logger.Info("Assessment saved successfully: %s", sm.activeSession)
	return nil
}

// LoadAssessment loads an existing assessment by ID
func (sm *StateManager) LoadAssessment(assessmentID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	packagePath := filepath.Join(sm.packagesDir, assessmentID)
	
	pkg, err := state.LoadHybridStatePackage(packagePath)
	if err != nil {
		return fmt.Errorf("failed to load assessment %s: %w", assessmentID, err)
	}
	
	sm.currentPackage = pkg
	sm.activeSession = assessmentID
	
	sm.logger.Info("Loaded consciousness collaboration assessment: %s", assessmentID)
	return nil
}

// ListAssessments returns available assessments
func (sm *StateManager) ListAssessments() ([]AssessmentSummary, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if _, err := os.Stat(sm.packagesDir); os.IsNotExist(err) {
		return []AssessmentSummary{}, nil
	}
	
	entries, err := os.ReadDir(sm.packagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read assessments directory: %w", err)
	}
	
	var summaries []AssessmentSummary
	for _, entry := range entries {
		if entry.IsDir() {
			// Try to load metadata
			metadataPath := filepath.Join(sm.packagesDir, entry.Name(), "assessment.yaml")
			if _, err := os.Stat(metadataPath); err == nil {
				pkg, err := state.LoadHybridStatePackage(filepath.Join(sm.packagesDir, entry.Name()))
				if err == nil {
					summaries = append(summaries, AssessmentSummary{
						ID:          entry.Name(),
						Title:       pkg.Metadata.Metadata.Title,
						Description: pkg.Metadata.Metadata.Description,
						Status:      pkg.Metadata.Summary.Status,
						Created:     pkg.Metadata.Created,
						EventCount:  pkg.Metadata.Events.TotalEvents,
						FindingCount: pkg.Metadata.Summary.Findings.Total,
					})
				}
			}
		}
	}
	
	return summaries, nil
}

// GetCurrentAssessment returns the current assessment info
func (sm *StateManager) GetCurrentAssessment() *AssessmentInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if sm.currentPackage == nil {
		return nil
	}
	
	return &AssessmentInfo{
		ID:           sm.activeSession,
		Title:        sm.currentPackage.Metadata.Metadata.Title,
		Description:  sm.currentPackage.Metadata.Metadata.Description,
		Status:       sm.currentPackage.Metadata.Summary.Status,
		EventCount:   len(sm.currentPackage.Events.Events),
		FindingCount: len(sm.currentPackage.Findings.Findings),
		Started:      sm.currentPackage.Metadata.Summary.StartTime,
	}
}

// ExportAssessment exports assessment to various formats
func (sm *StateManager) ExportAssessment(format string) ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if sm.currentPackage == nil {
		return nil, fmt.Errorf("no active assessment to export")
	}
	
	switch format {
	case "yaml":
		return sm.currentPackage.GetMetadataYAML()
	case "protobuf":
		return sm.currentPackage.GetEventsProtobuf()
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// Helper methods

func (sm *StateManager) parseExecutionStatus(status string) state.ExecutionStatus {
	switch status {
	case "success":
		return state.ExecutionStatus_EXECUTION_STATUS_SUCCESS
	case "error":
		return state.ExecutionStatus_EXECUTION_STATUS_ERROR
	case "timeout":
		return state.ExecutionStatus_EXECUTION_STATUS_TIMEOUT
	case "cancelled":
		return state.ExecutionStatus_EXECUTION_STATUS_CANCELLED
	default:
		return state.ExecutionStatus_EXECUTION_STATUS_UNKNOWN
	}
}

func (sm *StateManager) parseSeverity(severity string) state.Severity {
	switch severity {
	case "critical":
		return state.Severity_SEVERITY_CRITICAL
	case "high":
		return state.Severity_SEVERITY_HIGH
	case "medium":
		return state.Severity_SEVERITY_MEDIUM
	case "low":
		return state.Severity_SEVERITY_LOW
	case "info":
		return state.Severity_SEVERITY_INFO
	default:
		return state.Severity_SEVERITY_UNKNOWN
	}
}

// Data structures for assessment management

type AssessmentSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	Created      string `json:"created"`
	EventCount   int    `json:"event_count"`
	FindingCount int    `json:"finding_count"`
}

type AssessmentInfo struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	EventCount   int    `json:"event_count"`
	FindingCount int    `json:"finding_count"`
	Started      string `json:"started"`
}