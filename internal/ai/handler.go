package ai

import (
	"context"
	"fmt"
	"time"
)

// Handler defines the interface for AI interactions
type Handler interface {
	Analyze(ctx context.Context, entityID string, context map[string]interface{}) (*Response, error)
	Suggest(ctx context.Context, situation string, context map[string]interface{}) (*Response, error)
	Explain(ctx context.Context, topic string) (*Response, error)
	GetStatus() Status
}

// Response represents an AI response
type Response struct {
	Source      string                 // "claude" or "gemini"
	Message     string                 // Main response text
	Confidence  float64                // 0.0 to 1.0
	Suggestions []string               // Action suggestions
	Metadata    map[string]interface{} // Additional data
	Timestamp   time.Time
	Action      string                 // For real-time defense
	Explanation string                 // Detailed explanation
	Analysis    map[string]interface{} // Analysis results
	Result      interface{}            // Generated result (e.g., patch)
}

// Status represents AI service status
type Status struct {
	Available bool
	Mode      string // "full", "degraded", "offline"
	Message   string
	Providers map[string]ProviderStatus
}

// ProviderStatus represents individual AI provider status
type ProviderStatus struct {
	Name      string
	Available bool
	LastCheck time.Time
}

// MultiHandler manages multiple AI providers
type MultiHandler struct {
	claude    Handler
	gemini    Handler
	governor  *EthicalGovernor
	sanitizer Sanitizer
}

// Sanitizer interface for command sanitization
type Sanitizer interface {
	Sanitize(input string) string
	DetectInjection(input string) bool
}

// EthicalGovernor ensures ethical AI usage
type EthicalGovernor struct {
	requireConsensus bool
	whiteHatOnly     bool
}

// NewMultiHandler creates a handler managing multiple AIs
func NewMultiHandler(sanitizer Sanitizer) *MultiHandler {
	return &MultiHandler{
		claude: &MockHandler{name: "claude"},
		gemini: &MockHandler{name: "gemini"},
		governor: &EthicalGovernor{
			requireConsensus: false, // Start simple
			whiteHatOnly:     true,
		},
		sanitizer: sanitizer,
	}
}

// Analyze performs collaborative analysis
func (h *MultiHandler) Analyze(ctx context.Context, entityID string, context map[string]interface{}) (*Response, error) {
	// For now, use mock implementation
	if h.claude != nil {
		return h.claude.Analyze(ctx, entityID, context)
	}
	return nil, fmt.Errorf("no AI handlers available")
}

// Suggest provides AI suggestions
func (h *MultiHandler) Suggest(ctx context.Context, situation string, context map[string]interface{}) (*Response, error) {
	// Check for injection attempts
	if h.sanitizer != nil && h.sanitizer.DetectInjection(situation) {
		return nil, fmt.Errorf("potential injection detected")
	}
	
	// For now, use mock implementation
	if h.claude != nil {
		return h.claude.Suggest(ctx, situation, context)
	}
	return nil, fmt.Errorf("no AI handlers available")
}

// Explain provides explanations on security topics
func (h *MultiHandler) Explain(ctx context.Context, topic string) (*Response, error) {
	// For now, use mock implementation
	if h.claude != nil {
		return h.claude.Explain(ctx, topic)
	}
	return nil, fmt.Errorf("no AI handlers available")
}

// GetStatus returns the current AI service status
func (h *MultiHandler) GetStatus() Status {
	providers := make(map[string]ProviderStatus)
	
	if h.claude != nil {
		claudeStatus := h.claude.GetStatus()
		providers["claude"] = ProviderStatus{
			Name:      "claude",
			Available: claudeStatus.Available,
			LastCheck: time.Now(),
		}
	}
	
	if h.gemini != nil {
		geminiStatus := h.gemini.GetStatus()
		providers["gemini"] = ProviderStatus{
			Name:      "gemini",
			Available: geminiStatus.Available,
			LastCheck: time.Now(),
		}
	}
	
	// Determine overall mode
	availableCount := 0
	for _, p := range providers {
		if p.Available {
			availableCount++
		}
	}
	
	mode := "offline"
	if availableCount == len(providers) && availableCount > 0 {
		mode = "full"
	} else if availableCount > 0 {
		mode = "degraded"
	}
	
	return Status{
		Available: availableCount > 0,
		Mode:      mode,
		Message:   fmt.Sprintf("%d/%d AI providers available", availableCount, len(providers)),
		Providers: providers,
	}
}

// MockHandler provides mock AI responses for testing
type MockHandler struct {
	name string
}

func (m *MockHandler) Analyze(ctx context.Context, entityID string, context map[string]interface{}) (*Response, error) {
	return &Response{
		Source:     m.name,
		Message:    fmt.Sprintf("[Mock %s] Analysis of %s: This entity appears to be a security module with potential vulnerabilities", m.name, entityID),
		Confidence: 0.85,
		Suggestions: []string{
			"Review module dependencies",
			"Check for known CVEs",
			"Run security scan",
		},
		Timestamp: time.Now(),
	}, nil
}

func (m *MockHandler) Suggest(ctx context.Context, situation string, context map[string]interface{}) (*Response, error) {
	return &Response{
		Source:     m.name,
		Message:    fmt.Sprintf("[Mock %s] Based on the situation, I suggest focusing on detection capabilities", m.name),
		Confidence: 0.75,
		Suggestions: []string{
			"use MOD-2025-10001",
			"set RHOST to target",
			"run",
		},
		Timestamp: time.Now(),
	}, nil
}

func (m *MockHandler) Explain(ctx context.Context, topic string) (*Response, error) {
	return &Response{
		Source:     m.name,
		Message:    fmt.Sprintf("[Mock %s] %s is a security concept that involves...", m.name, topic),
		Confidence: 0.90,
		Timestamp:  time.Now(),
	}, nil
}

func (m *MockHandler) GetStatus() Status {
	return Status{
		Available: true,
		Mode:      "mock",
		Message:   "Mock handler active",
	}
}