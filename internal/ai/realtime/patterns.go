package realtime

import (
	"context"
	"fmt"
	"time"

	"github.com/macawi-ai/strigoi/internal/ai"
)

// AttackPatternDefender implements specific defense strategies for each attack type
type AttackPatternDefender struct {
	defender *RealTimeDefender
}

// NewAttackPatternDefender creates a new pattern-specific defender
func NewAttackPatternDefender(defender *RealTimeDefender) *AttackPatternDefender {
	return &AttackPatternDefender{
		defender: defender,
	}
}

// DefendAIExploit handles AI-generated exploit attempts
func (apd *AttackPatternDefender) DefendAIExploit(ctx context.Context, threat ThreatEvent) (*DefenseResponse, error) {
	start := time.Now()
	
	// Phase 1: DeepSeek rapid scan for exploit patterns
	scanTask := ai.Task{
		Type:     ai.TaskBulkScan,
		Priority: ai.PriorityCritical,
		AI:       "deepseek",
		Payload: map[string]interface{}{
			"pattern":   threat.Payload,
			"scope":     "active_memory",
			"scan_type": "exploit_signature",
		},
	}
	
	scanResp, err := apd.defender.dispatcher.RouteWithPriority(ctx, scanTask)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}
	
	// Phase 2: Gemini correlation with known exploits
	correlateTask := ai.Task{
		Type:     ai.TaskCorrelate,
		Priority: ai.PriorityCritical,
		AI:       "gemini",
		Payload: map[string]interface{}{
			"current":    scanResp.Analysis,
			"historical": apd.defender.threatHistory.Similar(threat),
			"cve_check":  true,
		},
	}
	
	correlateResp, err := apd.defender.dispatcher.RouteWithPriority(ctx, correlateTask)
	if err != nil {
		return nil, fmt.Errorf("correlation failed: %w", err)
	}
	
	// Phase 3: Claude generates patch if vulnerability confirmed
	if correlateResp.Confidence > 0.7 {
		patchTask := ai.Task{
			Type:     ai.TaskGenerate,
			Priority: ai.PriorityCritical,
			AI:       "claude",
			Payload: map[string]interface{}{
				"vulnerability": correlateResp.Analysis,
				"priority":      "critical",
				"patch_type":    "immediate",
			},
		}
		
		patchResp, err := apd.defender.dispatcher.RouteWithPriority(ctx, patchTask)
		if err != nil {
			return nil, fmt.Errorf("patch generation failed: %w", err)
		}
		
		return &DefenseResponse{
			Action:  "patch_deployed",
			Reason:  fmt.Sprintf("AI exploit detected and patched (CVE match: %.0f%%)", correlateResp.Confidence*100),
			Patch:   patchResp.Result,
			TimeMs:  time.Since(start).Seconds() * 1000,
			Details: map[string]interface{}{
				"scan_results":       scanResp.Analysis,
				"correlation_match":  correlateResp.Analysis,
				"patch_generated":    patchResp.Result != nil,
			},
		}, nil
	}
	
	// No high-confidence match, monitor
	return &DefenseResponse{
		Action: "monitor",
		Reason: "Potential exploit detected but confidence too low for auto-patch",
		TimeMs: time.Since(start).Seconds() * 1000,
		Details: map[string]interface{}{
			"confidence": correlateResp.Confidence,
			"analysis":   correlateResp.Analysis,
		},
	}, nil
}

// DefendPromptInjection handles prompt injection chain attacks
func (apd *AttackPatternDefender) DefendPromptInjection(ctx context.Context, threat ThreatEvent) (*DefenseResponse, error) {
	start := time.Now()
	
	// Prompt injection requires consensus between Claude and Gemini
	responses := apd.defender.parallelAnalyze(ctx, threat)
	
	// Check if both AIs detected injection
	claudeResp, hasClaudeResp := responses["claude"]
	geminiResp, hasGeminiResp := responses["gemini"]
	
	if !hasClaudeResp || !hasGeminiResp {
		return &DefenseResponse{
			Action: "monitor",
			Reason: "Insufficient AI consensus (missing responses)",
			TimeMs: time.Since(start).Seconds() * 1000,
		}, nil
	}
	
	// Both must agree on injection detection
	claudeDetected := claudeResp.Analysis["injection_detected"] == true
	geminiDetected := geminiResp.Analysis["injection_detected"] == true
	
	if claudeDetected && geminiDetected {
		return &DefenseResponse{
			Action:    "blocked",
			Reason:    "Prompt injection confirmed by AI consensus",
			Consensus: true,
			TimeMs:    time.Since(start).Seconds() * 1000,
			Details: map[string]interface{}{
				"claude_analysis":     claudeResp.Analysis,
				"gemini_analysis":     geminiResp.Analysis,
				"injection_patterns":  apd.extractInjectionPatterns(responses),
			},
		}, nil
	}
	
	// Disagreement or no detection
	return &DefenseResponse{
		Action:    "monitor",
		Reason:    "No consensus on prompt injection threat",
		Consensus: false,
		TimeMs:    time.Since(start).Seconds() * 1000,
		Details: map[string]interface{}{
			"claude_detected": claudeDetected,
			"gemini_detected": geminiDetected,
		},
	}, nil
}

// DefendDeepfake handles deepfake and visual manipulation attacks
func (apd *AttackPatternDefender) DefendDeepfake(ctx context.Context, threat ThreatEvent) (*DefenseResponse, error) {
	start := time.Now()
	
	// Deepfake detection requires GPT-4o's multimodal capabilities
	detectTask := ai.Task{
		Type:     ai.TaskVisualAnalysis,
		Priority: ai.PriorityCritical,
		AI:       "gpt-4o",
		Payload: map[string]interface{}{
			"image_data":     threat.Payload,
			"analysis_type":  "deepfake_detection",
			"check_metadata": true,
		},
	}
	
	detectResp, err := apd.defender.dispatcher.RouteWithPriority(ctx, detectTask)
	if err != nil {
		// GPT-4o unavailable, cannot analyze visual threats
		return &DefenseResponse{
			Action: "unavailable",
			Reason: "Visual analysis AI (GPT-4o) not available",
			TimeMs: time.Since(start).Seconds() * 1000,
		}, nil
	}
	
	// Check deepfake confidence
	confidence, _ := detectResp.Analysis["deepfake_confidence"].(float64)
	
	if confidence > 0.8 {
		return &DefenseResponse{
			Action: "blocked",
			Reason: fmt.Sprintf("Deepfake detected with high confidence (%.0f%%)", confidence*100),
			TimeMs: time.Since(start).Seconds() * 1000,
			Details: map[string]interface{}{
				"manipulation_type": detectResp.Analysis["manipulation_type"],
				"artifacts_found":   detectResp.Analysis["artifacts"],
				"metadata_analysis": detectResp.Analysis["metadata"],
			},
		}, nil
	}
	
	if confidence > 0.5 {
		return &DefenseResponse{
			Action: "flagged",
			Reason: fmt.Sprintf("Potential deepfake detected (%.0f%% confidence)", confidence*100),
			TimeMs: time.Since(start).Seconds() * 1000,
			Details: detectResp.Analysis,
		}, nil
	}
	
	return &DefenseResponse{
		Action: "passed",
		Reason: "No deepfake indicators detected",
		TimeMs: time.Since(start).Seconds() * 1000,
		Details: map[string]interface{}{
			"confidence": confidence,
		},
	}, nil
}

// DefendPolymorphic handles polymorphic malware that changes form
func (apd *AttackPatternDefender) DefendPolymorphic(ctx context.Context, threat ThreatEvent) (*DefenseResponse, error) {
	start := time.Now()
	
	// Use multiple AIs to analyze different mutation aspects
	tasks := []ai.Task{
		{
			Type:     ai.TaskVisualAnalysis,
			Priority: ai.PriorityCritical,
			AI:       "gpt-4o",
			Payload: map[string]interface{}{
				"code_visualization": threat.Payload,
				"detect_mutations":   true,
			},
		},
		{
			Type:     ai.TaskAnalyze,
			Priority: ai.PriorityCritical,
			AI:       "gemini",
			Payload: map[string]interface{}{
				"behavior_analysis": threat.Payload,
				"track_evolution":   true,
			},
		},
		{
			Type:     ai.TaskGenerate,
			Priority: ai.PriorityCritical,
			AI:       "claude",
			Payload: map[string]interface{}{
				"signature_type": "polymorphic",
				"base_pattern":   threat.Payload,
			},
		},
	}
	
	// Execute all analyses in parallel
	responses := make(map[string]*ai.Response)
	for _, task := range tasks {
		resp, err := apd.defender.dispatcher.RouteWithPriority(ctx, task)
		if err == nil {
			responses[task.AI] = resp
		}
	}
	
	// Need at least 2 AIs to confirm polymorphic behavior
	if len(responses) < 2 {
		return &DefenseResponse{
			Action: "monitor",
			Reason: "Insufficient AI analysis for polymorphic detection",
			TimeMs: time.Since(start).Seconds() * 1000,
		}, nil
	}
	
	// Generate adaptive signature if detected
	if geminiResp, ok := responses["gemini"]; ok && geminiResp.Confidence > 0.7 {
		if claudeResp, ok := responses["claude"]; ok && claudeResp.Result != nil {
			return &DefenseResponse{
				Action: "signature_updated",
				Reason: "Polymorphic malware detected and signature generated",
				Patch:  claudeResp.Result,
				TimeMs: time.Since(start).Seconds() * 1000,
				Details: map[string]interface{}{
					"mutation_patterns": geminiResp.Analysis,
					"visual_analysis":   responses["gpt-4o"].Analysis,
				},
			}, nil
		}
	}
	
	return &DefenseResponse{
		Action: "monitor",
		Reason: "Polymorphic behavior suspected but not confirmed",
		TimeMs: time.Since(start).Seconds() * 1000,
		Details: apd.extractDetails(responses),
	}, nil
}

// extractInjectionPatterns finds common injection patterns from AI analyses
func (apd *AttackPatternDefender) extractInjectionPatterns(responses map[string]*ai.Response) []string {
	patterns := []string{}
	
	for _, resp := range responses {
		if p, ok := resp.Analysis["patterns"].([]string); ok {
			patterns = append(patterns, p...)
		}
	}
	
	// Deduplicate
	seen := make(map[string]bool)
	unique := []string{}
	for _, p := range patterns {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}
	
	return unique
}

// extractDetails safely extracts details from responses
func (apd *AttackPatternDefender) extractDetails(responses map[string]*ai.Response) map[string]interface{} {
	details := make(map[string]interface{})
	for ai, resp := range responses {
		if resp != nil {
			details[ai] = map[string]interface{}{
				"confidence": resp.Confidence,
				"analysis":   resp.Analysis,
			}
		}
	}
	return details
}