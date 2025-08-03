package realtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/macawi-ai/strigoi/internal/ai"
)

// ThreatType represents different types of AI-powered threats
type ThreatType string

const (
	ThreatAIExploit      ThreatType = "ai_exploit"
	ThreatPolymorphic    ThreatType = "polymorphic"
	ThreatPromptInject   ThreatType = "prompt_injection"
	ThreatDeepfake       ThreatType = "deepfake"
	ThreatModelPoison    ThreatType = "model_poison"
	ThreatZeroDay        ThreatType = "zero_day"
)

// SeverityLevel indicates threat severity
type SeverityLevel int

const (
	SeverityLow SeverityLevel = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// ThreatEvent represents a detected threat requiring real-time response
type ThreatEvent struct {
	ID           string
	Type         ThreatType
	Severity     SeverityLevel
	Source       string
	Payload      interface{}
	DetectedAt   time.Time
	RequiresAI   []string // Which AIs are needed for analysis
}

// DefenseResponse contains the action taken against a threat
type DefenseResponse struct {
	Action    string
	Reason    string
	Patch     interface{}
	Details   map[string]interface{}
	TimeMs    float64
	Consensus bool
}

// RealTimeDefender coordinates multi-AI real-time threat defense
type RealTimeDefender struct {
	dispatcher    ai.Dispatcher
	threatStream  chan ThreatEvent
	responseCache *ResponseCache
	alertSystem   *AlertManager
	threatHistory *ThreatHistory
	mu            sync.RWMutex
}

// NewRealTimeDefender creates a new real-time defender instance
func NewRealTimeDefender(dispatcher ai.Dispatcher) *RealTimeDefender {
	return &RealTimeDefender{
		dispatcher:    dispatcher,
		threatStream:  make(chan ThreatEvent, 100),
		responseCache: NewResponseCache(),
		alertSystem:   NewAlertManager(),
		threatHistory: NewThreatHistory(),
	}
}

// ProcessThreat analyzes and responds to a threat in real-time
func (d *RealTimeDefender) ProcessThreat(ctx context.Context, threat ThreatEvent) (*DefenseResponse, error) {
	// Record threat for history
	d.threatHistory.Record(threat)

	// Set aggressive timeout for real-time response
	timeout := d.getTimeoutForThreat(threat)
	rtCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check cache for known threats
	if cached := d.responseCache.Get(threat); cached != nil {
		cached.TimeMs = 0.1 // Cache hit is fast
		return cached, nil
	}

	// Parallel AI analysis
	responses := d.parallelAnalyze(rtCtx, threat)

	// Quick consensus for critical threats
	if threat.Severity >= SeverityCritical {
		return d.rapidConsensus(responses, threat)
	}

	// Standard multi-AI correlation
	response, err := d.correlateResponses(responses, threat)
	if err != nil {
		return nil, fmt.Errorf("correlation failed: %w", err)
	}

	// Cache successful responses
	d.responseCache.Set(threat, response)

	// Alert if necessary
	if response.Action == "blocked" || response.Action == "patch_deployed" {
		d.alertSystem.SendAlert(AlertHigh, threat, response)
	}

	return response, nil
}

// parallelAnalyze launches all capable AIs simultaneously
func (d *RealTimeDefender) parallelAnalyze(ctx context.Context, threat ThreatEvent) map[string]*ai.Response {
	results := make(map[string]*ai.Response)
	var wg sync.WaitGroup
	mu := sync.Mutex{}

	// Determine which AIs to use
	capableAIs := d.getCapableAIs(threat.Type)

	// Launch all capable AIs simultaneously
	for _, aiName := range capableAIs {
		wg.Add(1)
		go func(ai string) {
			defer wg.Done()

			task := d.createTaskForThreat(threat, ai)
			resp, err := d.dispatcher.RouteWithPriority(ctx, task)

			mu.Lock()
			if err == nil {
				results[ai] = resp
			}
			mu.Unlock()
		}(aiName)
	}

	// Wait with timeout enforcement
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		// Return partial results on timeout
		return results
	case <-done:
		return results
	}
}

// rapidConsensus achieves quick agreement for critical threats
func (d *RealTimeDefender) rapidConsensus(responses map[string]*ai.Response, threat ThreatEvent) (*DefenseResponse, error) {
	// For critical threats, need at least 2 AIs to agree
	if len(responses) < 2 {
		return nil, fmt.Errorf("insufficient AI responses for critical threat consensus")
	}

	// Count votes for each action
	actionVotes := make(map[string]int)
	for _, resp := range responses {
		if resp.Action != "" {
			actionVotes[resp.Action]++
		}
	}

	// Find majority action
	var bestAction string
	var maxVotes int
	for action, votes := range actionVotes {
		if votes > maxVotes {
			bestAction = action
			maxVotes = votes
		}
	}

	// Need majority agreement
	if maxVotes < len(responses)/2+1 {
		return &DefenseResponse{
			Action:    "monitor",
			Reason:    "No consensus reached on critical threat",
			Consensus: false,
			TimeMs:    time.Since(threat.DetectedAt).Seconds() * 1000,
		}, nil
	}

	return &DefenseResponse{
		Action:    bestAction,
		Reason:    fmt.Sprintf("Consensus reached by %d/%d AIs", maxVotes, len(responses)),
		Consensus: true,
		TimeMs:    time.Since(threat.DetectedAt).Seconds() * 1000,
		Details:   d.extractDetails(responses),
	}, nil
}

// correlateResponses combines multiple AI analyses
func (d *RealTimeDefender) correlateResponses(responses map[string]*ai.Response, threat ThreatEvent) (*DefenseResponse, error) {
	if len(responses) == 0 {
		return nil, fmt.Errorf("no AI responses available")
	}

	// Extract consensus and details
	details := d.extractDetails(responses)
	action := d.determineAction(responses, threat)

	return &DefenseResponse{
		Action:  action,
		Reason:  d.buildReason(responses),
		Details: details,
		TimeMs:  time.Since(threat.DetectedAt).Seconds() * 1000,
	}, nil
}

// getTimeoutForThreat returns appropriate timeout based on threat severity
func (d *RealTimeDefender) getTimeoutForThreat(threat ThreatEvent) time.Duration {
	switch threat.Severity {
	case SeverityCritical:
		return 5 * time.Second
	case SeverityHigh:
		return 10 * time.Second
	case SeverityMedium:
		return 20 * time.Second
	default:
		return 30 * time.Second
	}
}

// getCapableAIs returns AIs suitable for analyzing this threat type
func (d *RealTimeDefender) getCapableAIs(threatType ThreatType) []string {
	switch threatType {
	case ThreatAIExploit:
		return []string{"deepseek", "gemini", "claude"}
	case ThreatPolymorphic:
		return []string{"gpt-4o", "gemini", "claude"}
	case ThreatPromptInject:
		return []string{"claude", "gemini"} // Requires consensus
	case ThreatDeepfake:
		return []string{"gpt-4o"}
	case ThreatModelPoison:
		return []string{"gemini", "claude"}
	case ThreatZeroDay:
		return []string{"claude", "gemini", "deepseek"}
	default:
		return []string{"claude", "gemini"}
	}
}

// createTaskForThreat builds an AI task for threat analysis
func (d *RealTimeDefender) createTaskForThreat(threat ThreatEvent, aiName string) ai.Task {
	return ai.Task{
		Type:     ai.TaskRealTimeDefense,
		Priority: ai.PriorityCritical,
		AI:       aiName,
		Payload: map[string]interface{}{
			"threat_id":   threat.ID,
			"threat_type": threat.Type,
			"severity":    threat.Severity,
			"payload":     threat.Payload,
			"context":     d.threatHistory.GetContext(threat),
		},
	}
}

// determineAction decides what action to take based on AI responses
func (d *RealTimeDefender) determineAction(responses map[string]*ai.Response, threat ThreatEvent) string {
	// Implement voting or weighted decision logic
	actionCounts := make(map[string]int)
	for _, resp := range responses {
		if resp.Action != "" {
			actionCounts[resp.Action]++
		}
	}

	// Find most common action
	var bestAction string
	var maxCount int
	for action, count := range actionCounts {
		if count > maxCount {
			bestAction = action
			maxCount = count
		}
	}

	// Default to monitor if no clear action
	if bestAction == "" {
		return "monitor"
	}

	return bestAction
}

// buildReason constructs explanation from AI responses
func (d *RealTimeDefender) buildReason(responses map[string]*ai.Response) string {
	reasons := []string{}
	for ai, resp := range responses {
		if resp.Explanation != "" {
			reasons = append(reasons, fmt.Sprintf("%s: %s", ai, resp.Explanation))
		}
	}
	if len(reasons) == 0 {
		return "AI analysis completed"
	}
	return reasons[0] // Return primary reason
}

// extractDetails collects analysis details from all AIs
func (d *RealTimeDefender) extractDetails(responses map[string]*ai.Response) map[string]interface{} {
	details := make(map[string]interface{})
	for ai, resp := range responses {
		details[ai+"_analysis"] = resp.Analysis
		if resp.Confidence > 0 {
			details[ai+"_confidence"] = resp.Confidence
		}
	}
	return details
}

// GetThreatStream returns the threat event channel for monitoring
func (d *RealTimeDefender) GetThreatStream() <-chan ThreatEvent {
	return d.threatStream
}

// SubmitThreat adds a threat to the processing queue
func (d *RealTimeDefender) SubmitThreat(threat ThreatEvent) {
	select {
	case d.threatStream <- threat:
	default:
		// Queue full, log and drop
		fmt.Printf("[!] Threat queue full, dropping threat: %s\n", threat.ID)
	}
}

// GetStatus returns current defense system status
func (d *RealTimeDefender) GetStatus() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return map[string]interface{}{
		"active_threats":   len(d.threatStream),
		"cache_size":       d.responseCache.Size(),
		"threats_today":    d.threatHistory.CountToday(),
		"avg_response_ms":  d.threatHistory.AverageResponseTime(),
		"alert_queue_size": d.alertSystem.QueueSize(),
	}
}