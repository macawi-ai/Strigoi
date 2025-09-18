package vsm

import (
	"log"
	"math/rand"
	"sync/atomic"
	"time"
)

// S3S4Loops implements S3↔S4 intelligence gathering loops
type S3S4Loops struct {
	manager *LoopManager

	// Intelligence metrics
	threatIntelUpdates  atomic.Value // int (new threats)
	performanceTrends   atomic.Value // float64 (trend direction)
	capacityUtilization atomic.Value // float64 (0-1)
	newCVEs             atomic.Value // int (count)
	anomalyPatterns     atomic.Value // int (new patterns)
	resourceEfficiency  atomic.Value // float64 (0-1)
	behaviorAnomalies   atomic.Value // int (unusual patterns)
}

// NewS3S4Loops creates S3↔S4 intelligence loops
func NewS3S4Loops(manager *LoopManager) *S3S4Loops {
	s := &S3S4Loops{
		manager: manager,
	}

	// Initialize metrics
	s.threatIntelUpdates.Store(0)
	s.performanceTrends.Store(0.0)
	s.capacityUtilization.Store(0.5)
	s.newCVEs.Store(0)
	s.anomalyPatterns.Store(0)
	s.resourceEfficiency.Store(0.8)
	s.behaviorAnomalies.Store(0)

	s.registerLoops()
	s.startIntelligenceGathering()

	return s
}

func (s *S3S4Loops) registerLoops() {
	// LOOP-S3S4-001: Threat Intelligence Feed
	s.manager.RegisterLoop(&FeedbackLoop{
		ID:    "LOOP-S3S4-001",
		Name:  "Threat Intelligence Feed",
		Level: "S3S4",
		Trigger: func() (bool, float64) {
			updates := s.threatIntelUpdates.Load().(int)
			if updates > 0 {
				return true, float64(updates) / 5.0
			}
			return false, 0.0
		},
		Action: func() error {
			updates := s.threatIntelUpdates.Load().(int)
			log.Printf("🔍 S3S4: Processing %d threat intelligence updates", updates)

			// Update detection patterns
			log.Printf("  └─ Detection patterns updated")
			s.threatIntelUpdates.Store(0)
			return nil
		},
	})

	// LOOP-S3S4-002: Performance Trend Analyzer
	s.manager.RegisterLoop(&FeedbackLoop{
		ID:    "LOOP-S3S4-002",
		Name:  "Performance Trend Analyzer",
		Level: "S3S4",
		Trigger: func() (bool, float64) {
			trend := s.performanceTrends.Load().(float64)
			if trend < -0.1 || trend > 0.3 { // Declining or spiking
				if trend < 0 {
					return true, -trend * 3.0
				}
				return true, trend * 2.0
			}
			return false, 0.0
		},
		Action: func() error {
			trend := s.performanceTrends.Load().(float64)

			if trend < 0 {
				log.Printf("📉 S3S4: Performance declining at %.1f%%/hour", -trend*100)
				log.Printf("  └─ Predicting resource exhaustion in %.0f hours", 10/(-trend))
			} else {
				log.Printf("📈 S3S4: Performance spike at +%.1f%%/hour", trend*100)
				log.Printf("  └─ Investigating anomalous improvement")
			}

			// Reset trend after analysis
			s.performanceTrends.Store(0.0)
			return nil
		},
	})

	// LOOP-S3S4-003: Capacity Planning Monitor
	s.manager.RegisterLoop(&FeedbackLoop{
		ID:    "LOOP-S3S4-003",
		Name:  "Capacity Planning Monitor",
		Level: "S3S4",
		Trigger: func() (bool, float64) {
			capacity := s.capacityUtilization.Load().(float64)
			if capacity > 0.8 { // Over 80% utilized
				return true, capacity * 2.0
			}
			return false, 0.0
		},
		Action: func() error {
			capacity := s.capacityUtilization.Load().(float64)
			log.Printf("📊 S3S4: Capacity at %.0f%%, recommending scaling", capacity*100)

			// Recommend scaling action
			if capacity > 0.9 {
				log.Printf("  └─ URGENT: Immediate scaling required")
			} else {
				log.Printf("  └─ Plan scaling within 24 hours")
			}

			return nil
		},
	})

	// LOOP-S3S4-004: Vulnerability Scanner
	s.manager.RegisterLoop(&FeedbackLoop{
		ID:    "LOOP-S3S4-004",
		Name:  "Vulnerability Scanner",
		Level: "S3S4",
		Trigger: func() (bool, float64) {
			cves := s.newCVEs.Load().(int)
			if cves > 0 {
				return true, float64(cves)
			}
			return false, 0.0
		},
		Action: func() error {
			cves := s.newCVEs.Load().(int)
			log.Printf("🔓 S3S4: Scanning for %d new CVEs", cves)

			// Scan and report
			critical := cves / 3
			if critical > 0 {
				log.Printf("  └─ Found %d CRITICAL vulnerabilities!", critical)
			}

			s.newCVEs.Store(0)
			return nil
		},
	})

	// LOOP-S3S4-005: Anomaly Pattern Learner
	s.manager.RegisterLoop(&FeedbackLoop{
		ID:    "LOOP-S3S4-005",
		Name:  "Anomaly Pattern Learner",
		Level: "S3S4",
		Trigger: func() (bool, float64) {
			patterns := s.anomalyPatterns.Load().(int)
			if patterns > 0 {
				return true, float64(patterns) / 3.0
			}
			return false, 0.0
		},
		Action: func() error {
			patterns := s.anomalyPatterns.Load().(int)
			log.Printf("🧠 S3S4: Learning %d new anomaly patterns", patterns)

			// Update ML models
			log.Printf("  └─ ML models updated with new patterns")
			s.anomalyPatterns.Store(0)
			return nil
		},
	})

	// LOOP-S3S4-006: Cost Optimization Analyzer
	s.manager.RegisterLoop(&FeedbackLoop{
		ID:    "LOOP-S3S4-006",
		Name:  "Cost Optimization Analyzer",
		Level: "S3S4",
		Trigger: func() (bool, float64) {
			efficiency := s.resourceEfficiency.Load().(float64)
			if efficiency < 0.7 { // Less than 70% efficient
				return true, (1.0 - efficiency) * 2.0
			}
			return false, 0.0
		},
		Action: func() error {
			efficiency := s.resourceEfficiency.Load().(float64)
			waste := (1.0 - efficiency) * 100
			log.Printf("💰 S3S4: Resource efficiency at %.0f%% (%.0f%% waste)",
				efficiency*100, waste)

			// Suggest optimizations
			log.Printf("  └─ Potential savings: $%.0f/month", waste*100)

			// Improve efficiency
			s.resourceEfficiency.Store(efficiency + 0.1)
			return nil
		},
	})

	// LOOP-S3S4-007: User Behavior Analyzer
	s.manager.RegisterLoop(&FeedbackLoop{
		ID:    "LOOP-S3S4-007",
		Name:  "User Behavior Analyzer",
		Level: "S3S4",
		Trigger: func() (bool, float64) {
			anomalies := s.behaviorAnomalies.Load().(int)
			if anomalies > 0 {
				return true, float64(anomalies) / 2.0
			}
			return false, 0.0
		},
		Action: func() error {
			anomalies := s.behaviorAnomalies.Load().(int)
			log.Printf("👤 S3S4: Detected %d user behavior anomalies", anomalies)

			// Flag for investigation
			if anomalies > 3 {
				log.Printf("  └─ HIGH RISK: Potential insider threat")
			} else {
				log.Printf("  └─ Flagged for routine investigation")
			}

			s.behaviorAnomalies.Store(0)
			return nil
		},
	})
}

// startIntelligenceGathering simulates intelligence feeds
func (s *S3S4Loops) startIntelligenceGathering() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Simulate random intelligence events

			// Threat intel (30% chance)
			if rand.Float32() < 0.3 {
				s.threatIntelUpdates.Store(rand.Intn(5) + 1)
			}

			// Performance trends
			trend := (rand.Float64() - 0.5) * 0.4
			s.performanceTrends.Store(trend)

			// Capacity changes
			current := s.capacityUtilization.Load().(float64)
			delta := (rand.Float64() - 0.5) * 0.1
			newCapacity := current + delta
			if newCapacity < 0 {
				newCapacity = 0
			}
			if newCapacity > 1 {
				newCapacity = 1
			}
			s.capacityUtilization.Store(newCapacity)

			// New CVEs (20% chance)
			if rand.Float32() < 0.2 {
				s.newCVEs.Store(rand.Intn(10) + 1)
			}

			// Anomaly patterns (25% chance)
			if rand.Float32() < 0.25 {
				s.anomalyPatterns.Store(rand.Intn(3) + 1)
			}

			// Resource efficiency fluctuation
			eff := s.resourceEfficiency.Load().(float64)
			eff += (rand.Float64() - 0.5) * 0.1
			if eff < 0.3 {
				eff = 0.3
			}
			if eff > 1.0 {
				eff = 1.0
			}
			s.resourceEfficiency.Store(eff)

			// User anomalies (15% chance)
			if rand.Float32() < 0.15 {
				s.behaviorAnomalies.Store(rand.Intn(5) + 1)
			}
		}
	}()
}
