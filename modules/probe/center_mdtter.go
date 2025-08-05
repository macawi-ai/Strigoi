package probe

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// MDTTERConfig holds configuration for MDTTER event generation
type MDTTERConfig struct {
	Enabled           bool
	EmbeddingDim      int
	VAMThreshold      float32
	TopologyAdaptive  bool
	StreamingEndpoint string
}

// MDTTERDissectorWrapper wraps a dissector to generate MDTTER events
type MDTTERDissectorWrapper struct {
	base       Dissector
	mdtterGen  *MDTTERGenerator
	config     MDTTERConfig
	eventsChan chan<- *MDTTEREvent
}

// Identify delegates to base dissector
func (w *MDTTERDissectorWrapper) Identify(data []byte) (bool, float64) {
	return w.base.Identify(data)
}

// Dissect delegates to base and generates MDTTER event
func (w *MDTTERDissectorWrapper) Dissect(data []byte) (*Frame, error) {
	// Get base dissection
	frame, err := w.base.Dissect(data)
	if err != nil {
		return nil, err
	}

	// Generate MDTTER event if enabled
	if w.config.Enabled {
		go w.generateMDTTEREvent(frame)
	}

	return frame, nil
}

// FindVulnerabilities delegates to base dissector
func (w *MDTTERDissectorWrapper) FindVulnerabilities(frame *Frame) []StreamVulnerability {
	vulns := w.base.FindVulnerabilities(frame)

	// Enrich MDTTER event with vulnerability data
	if w.config.Enabled && len(vulns) > 0 {
		go w.enrichMDTTERWithVulns(frame, vulns)
	}

	return vulns
}

// GetSessionID delegates to base dissector
func (w *MDTTERDissectorWrapper) GetSessionID(frame *Frame) (string, error) {
	return w.base.GetSessionID(frame)
}

// generateMDTTEREvent creates an MDTTER event from a frame
func (w *MDTTERDissectorWrapper) generateMDTTEREvent(frame *Frame) {
	// Get session ID
	sessionID, err := w.base.GetSessionID(frame)
	if err != nil {
		sessionID = "unknown"
	}

	// Generate MDTTER event
	event, err := w.mdtterGen.GenerateFromFrame(frame, sessionID)
	if err != nil {
		log.Printf("Failed to generate MDTTER event: %v", err)
		return
	}

	// Send to channel
	select {
	case w.eventsChan <- event:
		// Event sent
	default:
		log.Printf("MDTTER event channel full, dropping event")
	}

	// Log high-novelty events
	if event.VarietyAbsorptionMetric > w.config.VAMThreshold {
		log.Printf("High-novelty event detected: VAM=%.2f, Intent=%s",
			event.VarietyAbsorptionMetric,
			dominantIntent(event.IntentField))
	}
}

// enrichMDTTERWithVulns adds vulnerability context to MDTTER events
func (w *MDTTERDissectorWrapper) enrichMDTTERWithVulns(frame *Frame, vulns []StreamVulnerability) {
	// This would correlate vulnerabilities with topology changes
	// For PoC, we'll log the correlation
	for _, vuln := range vulns {
		if vuln.Severity == "critical" || vuln.Severity == "high" {
			log.Printf("MDTTER: High-severity vulnerability detected: %s (%s)",
				vuln.Type, vuln.Evidence)
		}
	}
}

// Helper function to get dominant intent
func dominantIntent(intents *IntentProbabilities) string {
	maxProb := float32(0)
	maxIntent := "unknown"

	intentsMap := map[string]float32{
		"reconnaissance":       intents.Reconnaissance,
		"initial_access":       intents.InitialAccess,
		"lateral_movement":     intents.LateralMovement,
		"privilege_escalation": intents.PrivilegeEscalation,
		"data_collection":      intents.DataCollection,
		"exfiltration":         intents.Exfiltration,
		"impact":               intents.Impact,
	}

	for intent, prob := range intentsMap {
		if prob > maxProb {
			maxProb = prob
			maxIntent = intent
		}
	}

	return fmt.Sprintf("%s (%.0f%%)", maxIntent, maxProb*100)
}

// WrapDissectorWithMDTTER wraps a dissector to generate MDTTER events
func WrapDissectorWithMDTTER(dissector Dissector, gen *MDTTERGenerator, config MDTTERConfig, eventsChan chan<- *MDTTEREvent) Dissector {
	return &MDTTERDissectorWrapper{
		base:       dissector,
		mdtterGen:  gen,
		config:     config,
		eventsChan: eventsChan,
	}
}

// MDTTEREnhancedModule provides MDTTER capabilities to any module
type MDTTEREnhancedModule struct {
	mdtterGen    *MDTTERGenerator
	mdtterConfig MDTTERConfig
	mdtterEvents chan *MDTTEREvent
	mu           sync.RWMutex
}

// NewMDTTEREnhancedModule creates MDTTER enhancement capabilities
func NewMDTTEREnhancedModule(config MDTTERConfig) *MDTTEREnhancedModule {
	// Create session manager with default values
	sessionManager := NewSessionManager(30*time.Second, 5*time.Second)

	return &MDTTEREnhancedModule{
		mdtterGen:    NewMDTTERGenerator(sessionManager),
		mdtterConfig: config,
		mdtterEvents: make(chan *MDTTEREvent, 1000),
	}
}

// GetMDTTEREvents returns the channel for MDTTER events
func (m *MDTTEREnhancedModule) GetMDTTEREvents() <-chan *MDTTEREvent {
	return m.mdtterEvents
}

// WrapDissectors wraps existing dissectors to generate MDTTER events
func (m *MDTTEREnhancedModule) WrapDissectors(dissectors []Dissector) []Dissector {
	wrapped := make([]Dissector, len(dissectors))

	for i, dissector := range dissectors {
		wrapped[i] = WrapDissectorWithMDTTER(dissector, m.mdtterGen, m.mdtterConfig, m.mdtterEvents)
	}

	return wrapped
}
