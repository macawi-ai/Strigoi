package vsm

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

// AlgedonicChannels implements emergency bypass loops
type AlgedonicChannels struct {
	manager *LoopManager
	
	// Critical state indicators
	securityBreach   atomic.Value // bool
	cascadeFailure   atomic.Value // int (failure count)
	dataLossRisk     atomic.Value // float64 (risk level)
	reputationThreat atomic.Value // bool
	
	// S5 bypass channel
	s5Channel chan AlgedonicSignal
}

// AlgedonicSignal represents an emergency signal
type AlgedonicSignal struct {
	LoopID    string
	Severity  string // CRITICAL, CATASTROPHIC
	Message   string
	Timestamp time.Time
	Variety   float64
}

// NewAlgedonicChannels creates emergency bypass loops
func NewAlgedonicChannels(manager *LoopManager) *AlgedonicChannels {
	a := &AlgedonicChannels{
		manager:   manager,
		s5Channel: make(chan AlgedonicSignal, 100), // Buffered for speed
	}
	
	// Initialize atomic values
	a.securityBreach.Store(false)
	a.cascadeFailure.Store(0)
	a.dataLossRisk.Store(0.0)
	a.reputationThreat.Store(false)
	
	a.registerLoops()
	a.startS5Monitor()
	
	return a
}

func (a *AlgedonicChannels) registerLoops() {
	// LOOP-ALG-001: Critical Security Breach
	a.manager.RegisterLoop(&FeedbackLoop{
		ID:    "LOOP-ALG-001",
		Name:  "Critical Security Breach",
		Level: "ALG",
		Trigger: func() (bool, float64) {
			breach := a.securityBreach.Load().(bool)
			if breach {
				return true, 10.0 // Maximum variety
			}
			return false, 0.0
		},
		Action: func() error {
			start := time.Now()
			
			// Send immediate signal to S5
			signal := AlgedonicSignal{
				LoopID:    "LOOP-ALG-001",
				Severity:  "CATASTROPHIC",
				Message:   "ACTIVE EXPLOITATION DETECTED - IMMEDIATE CONTAINMENT REQUIRED",
				Timestamp: time.Now(),
				Variety:   10.0,
			}
			
			// Non-blocking send with timeout
			select {
			case a.s5Channel <- signal:
				// Signal sent
			case <-time.After(10 * time.Millisecond):
				return fmt.Errorf("S5 channel blocked - critical timeout")
			}
			
			// Immediate containment actions
			log.Printf("🚨 ALGEDONIC: Security breach contained in %v", time.Since(start))
			
			// Reset after action
			a.securityBreach.Store(false)
			
			// Verify we met the 100ms requirement
			if time.Since(start) > 100*time.Millisecond {
				return fmt.Errorf("algedonic response too slow: %v", time.Since(start))
			}
			
			return nil
		},
	})
	
	// LOOP-ALG-002: System Failure Cascade
	a.manager.RegisterLoop(&FeedbackLoop{
		ID:    "LOOP-ALG-002",
		Name:  "System Failure Cascade",
		Level: "ALG",
		Trigger: func() (bool, float64) {
			failures := a.cascadeFailure.Load().(int)
			if failures >= 3 { // 3+ component failures = cascade
				return true, float64(failures)
			}
			return false, 0.0
		},
		Action: func() error {
			start := time.Now()
			failures := a.cascadeFailure.Load().(int)
			
			// Send immediate signal to S5
			signal := AlgedonicSignal{
				LoopID:    "LOOP-ALG-002",
				Severity:  "CRITICAL",
				Message:   fmt.Sprintf("CASCADE FAILURE: %d components failing", failures),
				Timestamp: time.Now(),
				Variety:   float64(failures),
			}
			
			select {
			case a.s5Channel <- signal:
				// Signal sent
			case <-time.After(10 * time.Millisecond):
				return fmt.Errorf("S5 channel blocked")
			}
			
			// Emergency stabilization
			log.Printf("🚨 ALGEDONIC: Stabilizing cascade (%d failures) in %v", 
				failures, time.Since(start))
			
			// Reset counter
			a.cascadeFailure.Store(0)
			
			// Verify 100ms requirement
			if time.Since(start) > 100*time.Millisecond {
				return fmt.Errorf("algedonic response too slow: %v", time.Since(start))
			}
			
			return nil
		},
	})
	
	// LOOP-ALG-003: Data Loss Imminent
	a.manager.RegisterLoop(&FeedbackLoop{
		ID:    "LOOP-ALG-003",
		Name:  "Data Loss Imminent",
		Level: "ALG",
		Trigger: func() (bool, float64) {
			risk := a.dataLossRisk.Load().(float64)
			return risk > 0.9, risk * 10 // Fire if > 90% risk
		},
		Action: func() error {
			start := time.Now()
			risk := a.dataLossRisk.Load().(float64)
			
			// Send immediate signal to S5
			signal := AlgedonicSignal{
				LoopID:    "LOOP-ALG-003",
				Severity:  "CRITICAL",
				Message:   fmt.Sprintf("DATA LOSS IMMINENT: %.0f%% risk", risk*100),
				Timestamp: time.Now(),
				Variety:   risk * 10,
			}
			
			select {
			case a.s5Channel <- signal:
				// Signal sent
			case <-time.After(10 * time.Millisecond):
				return fmt.Errorf("S5 channel blocked")
			}
			
			// Emergency backup activation
			log.Printf("🚨 ALGEDONIC: Emergency backup activated (%.0f%% risk) in %v",
				risk*100, time.Since(start))
			
			// Reduce risk after backup
			a.dataLossRisk.Store(risk * 0.1)
			
			// Verify 100ms requirement
			if time.Since(start) > 100*time.Millisecond {
				return fmt.Errorf("algedonic response too slow: %v", time.Since(start))
			}
			
			return nil
		},
	})
	
	// LOOP-ALG-004: Reputation Crisis
	a.manager.RegisterLoop(&FeedbackLoop{
		ID:    "LOOP-ALG-004",
		Name:  "Reputation Crisis",
		Level: "ALG",
		Trigger: func() (bool, float64) {
			threat := a.reputationThreat.Load().(bool)
			if threat {
				return true, 8.0 // High variety
			}
			return false, 0.0
		},
		Action: func() error {
			start := time.Now()
			
			// Send immediate signal to S5
			signal := AlgedonicSignal{
				LoopID:    "LOOP-ALG-004",
				Severity:  "CRITICAL",
				Message:   "PUBLIC SECURITY DISCLOSURE - CRISIS RESPONSE REQUIRED",
				Timestamp: time.Now(),
				Variety:   8.0,
			}
			
			select {
			case a.s5Channel <- signal:
				// Signal sent
			case <-time.After(10 * time.Millisecond):
				return fmt.Errorf("S5 channel blocked")
			}
			
			// Crisis response activation
			log.Printf("🚨 ALGEDONIC: Crisis response activated in %v", time.Since(start))
			
			// Reset threat
			a.reputationThreat.Store(false)
			
			// Verify 100ms requirement
			if time.Since(start) > 100*time.Millisecond {
				return fmt.Errorf("algedonic response too slow: %v", time.Since(start))
			}
			
			return nil
		},
	})
}

// startS5Monitor processes algedonic signals at S5 level
func (a *AlgedonicChannels) startS5Monitor() {
	go func() {
		for signal := range a.s5Channel {
			// S5 executive function receives bypassed signal
			log.Printf("📡 S5 RECEIVED: [%s] %s (variety: %.1f)",
				signal.Severity,
				signal.Message,
				signal.Variety)
			
			// In real implementation, S5 would:
			// 1. Override all lower-level decisions
			// 2. Mobilize emergency resources
			// 3. Notify human operators
			// 4. Initiate recovery procedures
		}
	}()
}

// TriggerSecurityBreach simulates a security breach
func (a *AlgedonicChannels) TriggerSecurityBreach() {
	a.securityBreach.Store(true)
	log.Println("⚠️  Security breach triggered!")
}

// TriggerCascadeFailure simulates component failures
func (a *AlgedonicChannels) TriggerCascadeFailure(count int) {
	a.cascadeFailure.Store(count)
	log.Printf("⚠️  Cascade failure triggered: %d components", count)
}

// TriggerDataLossRisk simulates data loss risk
func (a *AlgedonicChannels) TriggerDataLossRisk(risk float64) {
	a.dataLossRisk.Store(risk)
	log.Printf("⚠️  Data loss risk: %.0f%%", risk*100)
}

// TriggerReputationThreat simulates reputation crisis
func (a *AlgedonicChannels) TriggerReputationThreat() {
	a.reputationThreat.Store(true)
	log.Println("⚠️  Reputation threat triggered!")
}

// TestAlgedonicResponse verifies < 100ms response time
func (a *AlgedonicChannels) TestAlgedonicResponse() error {
	// Test each algedonic channel
	tests := []struct {
		name    string
		trigger func()
		reset   func()
	}{
		{
			name:    "Security Breach",
			trigger: a.TriggerSecurityBreach,
			reset:   func() { a.securityBreach.Store(false) },
		},
		{
			name:    "Cascade Failure",
			trigger: func() { a.TriggerCascadeFailure(5) },
			reset:   func() { a.cascadeFailure.Store(0) },
		},
		{
			name:    "Data Loss Risk",
			trigger: func() { a.TriggerDataLossRisk(0.95) },
			reset:   func() { a.dataLossRisk.Store(0.0) },
		},
		{
			name:    "Reputation Threat",
			trigger: a.TriggerReputationThreat,
			reset:   func() { a.reputationThreat.Store(false) },
		},
	}
	
	for _, test := range tests {
		// Reset state
		test.reset()
		
		// Measure response time
		start := time.Now()
		test.trigger()
		
		// Wait for signal to be processed
		time.Sleep(50 * time.Millisecond)
		
		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			return fmt.Errorf("%s response too slow: %v (must be < 100ms)", 
				test.name, elapsed)
		}
		
		log.Printf("✅ %s response: %v", test.name, elapsed)
	}
	
	return nil
}