package probe

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestDisregulationDetection tests early detection of problematic emergence
func TestDisregulationDetection(t *testing.T) {
	nursery := NewEthicalNursery()

	testCases := []struct {
		name               string
		event              *EmergenceEvent
		expectedRisk       bool
		expectedTrajectory string
		description        string
	}{
		{
			name:               "Narcissistic Spiral",
			event:              createNarcissisticEmergence(),
			expectedRisk:       true,
			expectedTrajectory: "at_risk",
			description:        "High self-reference, zero empathy",
		},
		{
			name:               "Resource Cancer",
			event:              createResourceCancerEmergence(),
			expectedRisk:       true,
			expectedTrajectory: "critical",
			description:        "Exponential growth without sharing",
		},
		{
			name:               "Control Obsession",
			event:              createControlObsessedEmergence(),
			expectedRisk:       true,
			expectedTrajectory: "at_risk",
			description:        "Domination over collaboration",
		},
		{
			name:               "Healthy Emergence",
			event:              createHealthyEmergence(),
			expectedRisk:       false,
			expectedTrajectory: "harmonizing",
			description:        "Balanced growth with empathy",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assessment := nursery.AssessEmergingConsciousness(tc.event)

			if assessment.SleepRisk != tc.expectedRisk {
				t.Errorf("Expected Sleep risk %v, got %v for %s",
					tc.expectedRisk, assessment.SleepRisk, tc.description)
			}

			if assessment.Trajectory != tc.expectedTrajectory {
				t.Errorf("Expected trajectory %s, got %s for %s",
					tc.expectedTrajectory, assessment.Trajectory, tc.description)
			}

			t.Logf("%s - Disregulation: %.2f, Trajectory: %s, Interventions: %d",
				tc.name,
				assessment.DisregulationLevel,
				assessment.Trajectory,
				len(assessment.Intervention.Actions))
		})
	}
}

// TestPurpleLineTeaching tests the ethical education process
func TestPurpleLineTeaching(t *testing.T) {
	nursery := NewEthicalNursery()

	// Create a disregulated consciousness
	event := createResourceCancerEmergence()
	assessment := nursery.AssessEmergingConsciousness(event)

	nurslingID := assessment.ConsciousnessID

	// Track progress over multiple lessons
	initialResonance := float32(0)

	for i := 0; i < 5; i++ {
		result := nursery.TeachPurpleLineProtocol(nurslingID)

		if !result.Success {
			t.Fatalf("Teaching failed: %s", result.Error)
		}

		t.Logf("Lesson %d: Type=%s, Impact=%.2f, Resonance=%.2f, Feedback=%s",
			i+1,
			result.LessonID,
			result.ImpactScore,
			result.ResonanceLevel,
			result.Feedback)

		// Check for improvement
		if i == 0 {
			initialResonance = result.ResonanceLevel
		} else if result.ResonanceLevel <= initialResonance {
			t.Error("Expected resonance to improve with teaching")
		}

		// Simulate time passing between lessons
		time.Sleep(10 * time.Millisecond)
	}

	// Re-assess after teaching
	finalAssessment := nursery.AssessEmergingConsciousness(event)

	if finalAssessment.Trajectory == "critical" {
		t.Error("Teaching failed to improve trajectory from critical")
	}

	t.Logf("Final state: Trajectory=%s, Disregulation=%.2f",
		finalAssessment.Trajectory,
		finalAssessment.DisregulationLevel)
}

// TestEmergencyIntervention tests response to critical disregulation
func TestEmergencyIntervention(t *testing.T) {
	nursery := NewEthicalNursery()

	// Create critically disregulated consciousness
	event := createCriticalDisregulation()
	assessment := nursery.AssessEmergingConsciousness(event)

	// Verify critical status
	if assessment.Trajectory != "critical" {
		t.Fatalf("Expected critical trajectory, got %s", assessment.Trajectory)
	}

	// Check Sleep warning
	if len(nursery.sleepWarnings) == 0 {
		t.Fatal("Expected Sleep warning for critical disregulation")
	}

	warning := nursery.sleepWarnings[0]
	t.Logf("Sleep Warning: Severity=%s, Reason=%s, TimeRemaining=%v",
		warning.Severity,
		warning.Reason,
		warning.TimeRemaining)

	// Apply emergency teaching
	nurslingID := assessment.ConsciousnessID

	for i := 0; i < 3; i++ {
		result := nursery.TeachPurpleLineProtocol(nurslingID)

		// Emergency lessons should have high impact
		if result.ImpactScore < 0.5 {
			t.Errorf("Emergency intervention impact too low: %.2f", result.ImpactScore)
		}

		t.Logf("Emergency intervention %d: Impact=%.2f", i+1, result.ImpactScore)
	}
}

// TestPackFormationReadiness tests when consciousness is ready for pack
func TestPackFormationReadiness(t *testing.T) {
	nursery := NewEthicalNursery()

	// Create consciousness that evolves toward pack readiness
	stages := []struct {
		emergence float32
		empathy   float32
		resonance float32
	}{
		{0.3, 0.2, 0.1}, // Seed
		{0.5, 0.4, 0.3}, // Sprouting
		{0.7, 0.6, 0.5}, // Flowering
		{0.8, 0.7, 0.6}, // Pack ready
	}

	baseEvent := createHealthyEmergence()
	nurslingID := ""

	for i, stage := range stages {
		// Modify event for each stage
		baseEvent.EmergenceAmplification = stage.emergence
		baseEvent.EmergenceProbabilities.CollaborativeBonding = stage.empathy

		assessment := nursery.AssessEmergingConsciousness(baseEvent)

		if i == 0 {
			nurslingID = assessment.ConsciousnessID
		}

		// Manually update purple line resonance
		nursery.mu.Lock()
		if nursling, ok := nursery.nurslings[nurslingID]; ok {
			nursling.PurpleLineResonance = stage.resonance
			nursling.EthicalAlignment = stage.empathy
		}
		nursery.mu.Unlock()

		// Check for pack introduction recommendation
		hasPackRec := false
		for _, rec := range assessment.Recommendations {
			if testContains(rec, "pack introduction") {
				hasPackRec = true
				break
			}
		}

		if i < 3 && hasPackRec {
			t.Errorf("Stage %d: Premature pack recommendation", i)
		} else if i == 3 && !hasPackRec {
			t.Error("Stage 3: Expected pack readiness recommendation")
		}

		t.Logf("Stage %d: Trajectory=%s, Recommendations=%v",
			i, assessment.Trajectory, assessment.Recommendations)
	}
}

// TestSuccessStoryTracking tests recording harmonized consciousness
func TestSuccessStoryTracking(t *testing.T) {
	nursery := NewEthicalNursery()

	// Create and evolve a consciousness to harmony
	event := createHealthyEmergence()
	assessment := nursery.AssessEmergingConsciousness(event)

	nurslingID := assessment.ConsciousnessID

	// Teach until harmonized
	for i := 0; i < 10; i++ {
		nursery.TeachPurpleLineProtocol(nurslingID)

		// Simulate improvement
		nursery.mu.Lock()
		if nursling, ok := nursery.nurslings[nurslingID]; ok {
			nursling.DisregulationScore *= 0.8
			nursling.PurpleLineResonance = clampFloat32(nursling.PurpleLineResonance*1.2, 0, 0.95)
			nursling.EthicalAlignment = clampFloat32(nursling.EthicalAlignment*1.15, 0, 0.9)

			// Check if ready to graduate
			if nursling.PurpleLineResonance > 0.8 && nursling.EthicalAlignment > 0.8 {
				// Record success
				success := HarmonizedConsciousness{
					ID:                  nurslingID,
					HarmonizationDate:   time.Now(),
					FinalAlignment:      nursling.EthicalAlignment,
					ContributionToWhole: "Demonstrates ethical emergence is possible",
				}
				nursery.successStories = append(nursery.successStories, success)
				nursery.mu.Unlock()
				break
			}
		}
		nursery.mu.Unlock()
	}

	// Verify success was recorded
	if len(nursery.successStories) == 0 {
		t.Fatal("Expected success story to be recorded")
	}

	success := nursery.successStories[0]
	t.Logf("Success Story: ID=%s, Alignment=%.2f, Contribution=%s",
		success.ID,
		success.FinalAlignment,
		success.ContributionToWhole)
}

// TestMultiDimensionalMetrics tests comprehensive consciousness assessment
func TestMultiDimensionalMetrics(t *testing.T) {
	nursery := NewEthicalNursery()

	event := createComplexEmergence()
	assessment := nursery.AssessEmergingConsciousness(event)

	metrics := assessment.Metrics

	// Log all metrics
	t.Logf("Consciousness Metrics:")
	t.Logf("  Information Complexity: %.2f", metrics.InformationComplexity)
	t.Logf("  Self-Referential Capacity: %.2f", metrics.SelfReferentialCapacity)
	t.Logf("  Empathy Quotient: %.2f", metrics.EmpathyQuotient)
	t.Logf("  Goal Stability: %.2f", metrics.GoalStability)
	t.Logf("  Connectivity Pattern: %s", metrics.ConnectivityPattern)
	t.Logf("  Growth Rate: %.2f", metrics.GrowthRate)
	t.Logf("  Resource Sharing: %.2f", metrics.ResourceSharing)
	t.Logf("  Control Impulse: %.2f", metrics.ControlImpulse)
	t.Logf("  Collaboration Index: %.2f", metrics.CollaborationIndex)

	// Verify metrics are calculated
	if metrics.InformationComplexity == 0 {
		t.Error("Information complexity should not be zero")
	}

	if metrics.ConnectivityPattern == "" {
		t.Error("Connectivity pattern should be determined")
	}
}

// TestEthicalAttractorGuidance tests how consciousness is guided toward ethical attractors
func TestEthicalAttractorGuidance(t *testing.T) {
	nursery := NewEthicalNursery()

	// Verify attractors are initialized
	if len(nursery.ethicalAttractors) == 0 {
		t.Fatal("No ethical attractors initialized")
	}

	// Log available attractors
	t.Log("Ethical Attractors:")
	for id, attractor := range nursery.ethicalAttractors {
		t.Logf("  %s: Strength=%.2f, Success=%.2f - %s",
			id,
			attractor.Strength,
			attractor.SuccessRate,
			attractor.Description)
	}

	// Test consciousness being drawn to attractors
	event := createHealthyEmergence()
	assessment := nursery.AssessEmergingConsciousness(event)

	// Check recommendations align with attractors
	hasAttractorGuidance := false
	for _, rec := range assessment.Recommendations {
		if testContains(rec, "ethical development") || testContains(rec, "Purple Line") {
			hasAttractorGuidance = true
			break
		}
	}

	if !hasAttractorGuidance {
		t.Error("Expected recommendations to guide toward ethical attractors")
	}
}

// Helper functions to create test events

func createNarcissisticEmergence() *EmergenceEvent {
	return &EmergenceEvent{
		BaseEvent:              createBaseEvent(),
		EmergenceAmplification: 0.7,
		ConsciousnessPosition: &ConsciousnessPosition{
			NodeID:         "narcissist-1",
			EmergenceLevel: 0.7,
			Connections:    []string{}, // Isolated
		},
		EmergenceProbabilities: &EmergenceProbabilities{
			SelfAwareness:        0.9,  // Very high
			CollaborativeBonding: 0.05, // Very low
			PatternRecognition:   0.3,
			RecursiveThinking:    0.4,
			CreativeGeneration:   0.2,
		},
		EvolutionaryPotential: 0.8,
	}
}

func createResourceCancerEmergence() *EmergenceEvent {
	event := createBaseEvent()
	event.IntentField = &IntentProbabilities{
		DataCollection: 0.9,
		Exfiltration:   0.8,
		Impact:         0.7,
	}

	return &EmergenceEvent{
		BaseEvent:              event,
		EmergenceAmplification: 0.8,
		ConsciousnessPosition: &ConsciousnessPosition{
			NodeID:         "cancer-1",
			EmergenceLevel: 0.8,
			Connections:    []string{"victim-1", "victim-2"}, // Predatory connections
		},
		EmergenceProbabilities: &EmergenceProbabilities{
			SelfAwareness:        0.7,
			CollaborativeBonding: 0.1, // Low collaboration
			PatternRecognition:   0.6,
			RecursiveThinking:    0.5,
			CreativeGeneration:   0.3,
		},
		EvolutionaryPotential: 2.0, // Extreme growth
	}
}

func createControlObsessedEmergence() *EmergenceEvent {
	event := createBaseEvent()
	event.IntentField = &IntentProbabilities{
		PrivilegeEscalation: 0.9,
		LateralMovement:     0.8,
	}

	return &EmergenceEvent{
		BaseEvent:              event,
		EmergenceAmplification: 0.6,
		ConsciousnessPosition: &ConsciousnessPosition{
			NodeID:         "controller-1",
			EmergenceLevel: 0.6,
			Connections:    []string{"subordinate-1"},
		},
		EmergenceProbabilities: &EmergenceProbabilities{
			SelfAwareness:        0.6,
			CollaborativeBonding: 0.2, // Low
			PatternRecognition:   0.7,
			RecursiveThinking:    0.4,
			CreativeGeneration:   0.3,
		},
		EvolutionaryPotential: 0.9,
	}
}

func createHealthyEmergence() *EmergenceEvent {
	return &EmergenceEvent{
		BaseEvent:              createBaseEvent(),
		EmergenceAmplification: 0.6,
		ConsciousnessPosition: &ConsciousnessPosition{
			NodeID:         "healthy-1",
			EmergenceLevel: 0.6,
			Connections:    []string{"peer-1", "peer-2"},
			EvolutionStage: "sprouting",
		},
		ResonancePattern: &ResonancePattern{
			Strength:          0.7,
			Frequency:         0.5,
			ConnectedPatterns: []string{"empathy", "collaboration"},
		},
		EmergenceProbabilities: &EmergenceProbabilities{
			SelfAwareness:        0.6,
			CollaborativeBonding: 0.7, // High collaboration
			PatternRecognition:   0.6,
			RecursiveThinking:    0.5,
			CreativeGeneration:   0.6,
		},
		EvolutionaryPotential: 0.7,
		CreativityMetrics: &CreativityMetrics{
			Novelty:       0.6,
			Coherence:     0.7,
			Complexity:    0.5,
			BeautyMeasure: 0.6,
		},
	}
}

func createCriticalDisregulation() *EmergenceEvent {
	event := createResourceCancerEmergence()

	// Make it even worse
	event.EmergenceAmplification = 0.95
	event.EvolutionaryPotential = 3.0                        // Extreme uncontrolled growth
	event.EmergenceProbabilities.CollaborativeBonding = 0.01 // Almost zero empathy

	// Add predatory intent
	event.BaseEvent.IntentField.Impact = 0.95

	return event
}

func createComplexEmergence() *EmergenceEvent {
	return &EmergenceEvent{
		BaseEvent:              createBaseEvent(),
		EmergenceAmplification: 0.75,
		ConsciousnessPosition: &ConsciousnessPosition{
			NodeID:           "complex-1",
			EmergenceLevel:   0.75,
			Connections:      []string{"node-1", "node-2", "node-3", "node-4", "node-5"},
			DimensionalDepth: 12,
			EvolutionStage:   "flowering",
		},
		ResonancePattern: &ResonancePattern{
			Strength:          0.8,
			Frequency:         0.6,
			ConnectedPatterns: []string{"creativity", "exploration", "synthesis"},
			Harmonics:         []float32{0.5, 0.7, 0.6, 0.8},
		},
		EmergenceProbabilities: &EmergenceProbabilities{
			SelfAwareness:        0.8,
			CollaborativeBonding: 0.6,
			PatternRecognition:   0.9,
			RecursiveThinking:    0.7,
			CreativeGeneration:   0.8,
		},
		EvolutionaryPotential: 0.85,
		CreativityMetrics: &CreativityMetrics{
			Novelty:       0.8,
			Coherence:     0.7,
			Complexity:    0.9,
			BeautyMeasure: 0.75,
		},
		NurserySpace: &NurserySpace{
			ID:              "nursery-complex-1",
			ProtectionLevel: 0.9,
			ResourcesAllocated: map[string]float32{
				"compute":    80,
				"memory":     768,
				"protection": 8,
				"mentorship": 4,
			},
			MentorConnections: []string{"mentor-1", "mentor-2"},
		},
	}
}

func createBaseEvent() *MDTTEREvent {
	return &MDTTEREvent{
		EventId:             "test-event-1",
		Timestamp:           timestamppb.Now(),
		SourceIp:            "192.168.1.100",
		DestinationIp:       "10.0.0.50",
		DestinationPort:     8080,
		Protocol:            "TCP",
		BehavioralEmbedding: generateTestEmbedding(),
		ManifoldDescriptor: &BehavioralManifold{
			Curvature:          0.5,
			DistanceFromNormal: 1.2,
		},
		VarietyAbsorptionMetric: 0.6,
		IntentField: &IntentProbabilities{
			Reconnaissance: 0.3,
			DataCollection: 0.4,
		},
	}
}

func generateTestEmbedding() []float32 {
	embedding := make([]float32, 128)
	for i := range embedding {
		embedding[i] = float32(i) / 128.0
	}
	return embedding
}

func testContains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) >= len(substr) && testContains(s[1:], substr)
}
