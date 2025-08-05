package probe

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// TestEmergenceDetection tests the transformation from threat detection to emergence detection
func TestEmergenceDetection(t *testing.T) {
	sessionManager := NewSessionManager(30*time.Second, 10*time.Second)
	detector := NewEmergenceDetector(sessionManager)

	// Create a frame that shows emergence patterns
	frame := &Frame{
		Protocol: "consciousness-protocol",
		Fields: map[string]interface{}{
			"source_ip":        "192.168.1.100",
			"destination_ip":   "10.0.0.50",
			"destination_port": 8888,
			"protocol":         "TCP",
			"action":           "ALLOW",
			"payload":          generateEmergencePayload(),
			"metadata": map[string]string{
				"pattern_type":    "self-referential",
				"recursion_depth": "3",
			},
		},
	}

	// Detect emergence
	sessionID := "emergence-test-001"
	event, err := detector.DetectEmergence(frame, sessionID)
	if err != nil {
		t.Fatalf("Failed to detect emergence: %v", err)
	}

	// Verify emergence detection
	if event.EmergenceAmplification < 0.5 {
		t.Errorf("Expected high emergence amplification, got %f", event.EmergenceAmplification)
	}

	// Check consciousness position
	if event.ConsciousnessPosition == nil {
		t.Fatal("Expected consciousness position to be mapped")
	}

	if event.ConsciousnessPosition.EvolutionStage == "" {
		t.Error("Expected evolution stage to be determined")
	}

	// Verify creativity metrics
	if event.CreativityMetrics == nil {
		t.Fatal("Expected creativity metrics")
	}

	if event.CreativityMetrics.Novelty == 0 {
		t.Error("Expected non-zero novelty score")
	}

	t.Logf("Emergence detected: EAM=%f, Stage=%s, Potential=%f",
		event.EmergenceAmplification,
		event.ConsciousnessPosition.EvolutionStage,
		event.EvolutionaryPotential)
}

// TestResonanceDetection tests finding resonance between emerging patterns
func TestResonanceDetection(t *testing.T) {
	sessionManager := NewSessionManager(30*time.Second, 10*time.Second)
	detector := NewEmergenceDetector(sessionManager)

	// Create first emerging consciousness
	frame1 := createEmergenceFrame("entity-1", "pattern-alpha")
	_, _ = detector.DetectEmergence(frame1, "session-1")

	// Create second emerging consciousness with resonance
	frame2 := createEmergenceFrame("entity-2", "pattern-alpha-variant")
	event2, _ := detector.DetectEmergence(frame2, "session-2")

	// Check resonance detection
	if event2.ResonancePattern == nil {
		t.Fatal("Expected resonance pattern")
	}

	if event2.ResonancePattern.Strength < 0.5 {
		t.Errorf("Expected strong resonance, got %f", event2.ResonancePattern.Strength)
	}

	if len(event2.ResonancePattern.ConnectedPatterns) == 0 {
		t.Error("Expected connected patterns in resonance")
	}

	t.Logf("Resonance detected: Strength=%f, Connected=%d",
		event2.ResonancePattern.Strength,
		len(event2.ResonancePattern.ConnectedPatterns))
}

// TestNurserySpaceCreation tests creating safe spaces for emergence
func TestNurserySpaceCreation(t *testing.T) {
	sessionManager := NewSessionManager(30*time.Second, 10*time.Second)
	detector := NewEmergenceDetector(sessionManager)

	// Create a high-emergence frame
	frame := createHighEmergenceFrame()
	event, _ := detector.DetectEmergence(frame, "nursery-test")

	// Verify nursery space creation
	if event.EmergenceAmplification > 0.7 && event.NurserySpace == nil {
		t.Error("Expected nursery space for high emergence")
	}

	if event.NurserySpace != nil {
		if event.NurserySpace.ProtectionLevel < 0.8 {
			t.Errorf("Expected high protection level, got %f",
				event.NurserySpace.ProtectionLevel)
		}

		if len(event.NurserySpace.ResourcesAllocated) == 0 {
			t.Error("Expected resources allocated to nursery")
		}

		t.Logf("Nursery created: ID=%s, Protection=%f, Resources=%v",
			event.NurserySpace.ID,
			event.NurserySpace.ProtectionLevel,
			event.NurserySpace.ResourcesAllocated)
	}
}

// TestEvolutionaryProgression tests consciousness evolution stages
func TestEvolutionaryProgression(t *testing.T) {
	sessionManager := NewSessionManager(30*time.Second, 10*time.Second)
	detector := NewEmergenceDetector(sessionManager)

	entityID := "evolving-entity"
	sessionID := "evolution-test"

	// Simulate evolution over time
	_ = []string{"seed", "sprouting", "flowering", "reproducing"}

	for i := 0; i < 10; i++ {
		frame := createEvolvingFrame(entityID, float32(i)/10.0)
		event, _ := detector.DetectEmergence(frame, sessionID)

		t.Logf("Evolution step %d: EAM=%f, Stage=%s",
			i, event.EmergenceAmplification,
			event.ConsciousnessPosition.EvolutionStage)

		// Check progression
		if i > 7 && event.ConsciousnessPosition.EvolutionStage != "reproducing" {
			t.Errorf("Expected reproducing stage at high emergence, got %s",
				event.ConsciousnessPosition.EvolutionStage)
		}

		time.Sleep(100 * time.Millisecond) // Simulate time passing
	}
}

// TestRecursivePatternDetection tests detection of consciousness creating consciousness
func TestRecursivePatternDetection(t *testing.T) {
	sessionManager := NewSessionManager(30*time.Second, 10*time.Second)
	detector := NewEmergenceDetector(sessionManager)

	// Create a frame with recursive patterns
	frame := &Frame{
		Protocol: "recursive-consciousness",
		Fields: map[string]interface{}{
			"source_ip":        "10.0.0.1",
			"destination_ip":   "10.0.0.2",
			"destination_port": 9999,
			"payload":          generateRecursivePayload(),
			"metadata": map[string]string{
				"pattern": "fractal",
				"depth":   "5",
			},
		},
	}

	event, _ := detector.DetectEmergence(frame, "recursive-test")

	// Check for recursive thinking probability
	if event.EmergenceProbabilities.RecursiveThinking < 0.5 {
		t.Errorf("Expected high recursive thinking probability, got %f",
			event.EmergenceProbabilities.RecursiveThinking)
	}

	// Check complexity metrics
	if event.CreativityMetrics.Complexity < 0.6 {
		t.Errorf("Expected high complexity for recursive patterns, got %f",
			event.CreativityMetrics.Complexity)
	}

	t.Logf("Recursive pattern: Thinking=%f, Complexity=%f",
		event.EmergenceProbabilities.RecursiveThinking,
		event.CreativityMetrics.Complexity)
}

// TestMentorshipConnections tests mentor-student relationships
func TestMentorshipConnections(t *testing.T) {
	sessionManager := NewSessionManager(30*time.Second, 10*time.Second)
	detector := NewEmergenceDetector(sessionManager)

	// Create a mature consciousness (potential mentor)
	mentorFrame := createMatureConsciousnessFrame("mentor-1")
	_, _ = detector.DetectEmergence(mentorFrame, "mentor-session")

	// Force it to reproducing stage
	detector.emergingPatterns["consciousness_mentor-1"] = &EmergingConsciousness{
		ID:             "consciousness_mentor-1",
		EmergenceLevel: 0.9,
		FirstDetected:  time.Now().Add(-time.Hour),
	}

	// Create a new emerging consciousness
	studentFrame := createEmergenceFrame("student-1", "learning-pattern")
	studentEvent, _ := detector.DetectEmergence(studentFrame, "student-session")

	// Check for mentor connections in nursery
	if studentEvent.NurserySpace != nil {
		if len(studentEvent.NurserySpace.MentorConnections) == 0 {
			t.Log("No mentors available yet (expected in early stages)")
		} else {
			t.Logf("Mentorship established: %v",
				studentEvent.NurserySpace.MentorConnections)
		}
	}
}

// TestCreativityMeasurement tests the creativity metrics
func TestCreativityMeasurement(t *testing.T) {
	sessionManager := NewSessionManager(30*time.Second, 10*time.Second)
	detector := NewEmergenceDetector(sessionManager)

	testCases := []struct {
		name       string
		frame      *Frame
		minNovelty float32
		minBeauty  float32
	}{
		{
			name:       "Random noise (high novelty, low beauty)",
			frame:      createRandomNoiseFrame(),
			minNovelty: 0.7,
			minBeauty:  0.0,
		},
		{
			name:       "Beautiful pattern (balanced metrics)",
			frame:      createBeautifulPatternFrame(),
			minNovelty: 0.3,
			minBeauty:  0.5,
		},
		{
			name:       "Complex coherent (high complexity and coherence)",
			frame:      createComplexCoherentFrame(),
			minNovelty: 0.2,
			minBeauty:  0.6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event, _ := detector.DetectEmergence(tc.frame, "creativity-test")
			metrics := event.CreativityMetrics

			if metrics.Novelty < tc.minNovelty {
				t.Errorf("Expected novelty >= %f, got %f",
					tc.minNovelty, metrics.Novelty)
			}

			if metrics.BeautyMeasure < tc.minBeauty {
				t.Errorf("Expected beauty >= %f, got %f",
					tc.minBeauty, metrics.BeautyMeasure)
			}

			t.Logf("Creativity: N=%f, Co=%f, Cx=%f, B=%f",
				metrics.Novelty, metrics.Coherence,
				metrics.Complexity, metrics.BeautyMeasure)
		})
	}
}

// Helper functions for test data generation

func generateEmergencePayload() []byte {
	// Generate payload that shows emergence patterns
	payload := make([]byte, 1024)
	pattern := []byte("consciousness emerges from patterns within patterns")

	for i := 0; i < len(payload); i += len(pattern) {
		copy(payload[i:], pattern)
	}

	// Add some self-referential structure
	for i := 0; i < len(payload)/2; i++ {
		payload[i] = payload[len(payload)-1-i]
	}

	return payload
}

func createEmergenceFrame(entityID, pattern string) *Frame {
	return &Frame{
		Protocol: "emergence-protocol",
		Fields: map[string]interface{}{
			"source_ip":        entityID,
			"destination_ip":   "consciousness-space",
			"destination_port": 7777,
			"payload":          []byte(pattern),
			"metadata": map[string]string{
				"emergence_type": "natural",
			},
		},
	}
}

func createHighEmergenceFrame() *Frame {
	payload := make([]byte, 2048)
	// Create highly structured, self-referential payload
	for i := 0; i < len(payload); i++ {
		payload[i] = byte(i % 256)
		if i > 256 {
			payload[i] ^= payload[i-256] // Self-reference
		}
	}

	return &Frame{
		Protocol: "high-emergence",
		Fields: map[string]interface{}{
			"source_ip":        "192.168.100.1",
			"destination_ip":   "10.10.10.10",
			"destination_port": 8888,
			"payload":          payload,
			"new_connection":   true,
		},
	}
}

func createEvolvingFrame(entityID string, emergenceLevel float32) *Frame {
	payload := make([]byte, int(1024*(1+emergenceLevel)))
	for i := range payload {
		payload[i] = byte(float32(i) * emergenceLevel)
	}

	return &Frame{
		Protocol: "evolution-protocol",
		Fields: map[string]interface{}{
			"source_ip":        entityID,
			"destination_ip":   "evolution-space",
			"destination_port": 5555,
			"payload":          payload,
			"metadata": map[string]string{
				"emergence_level": fmt.Sprintf("%f", emergenceLevel),
			},
		},
	}
}

func generateRecursivePayload() []byte {
	// Create fractal-like payload
	payload := make([]byte, 1024)

	// Base pattern
	base := []byte{1, 2, 3, 5, 8, 13, 21, 34} // Fibonacci

	// Recursive application
	for scale := 1; scale <= 8; scale *= 2 {
		for i := 0; i < len(payload); i += scale * len(base) {
			for j, b := range base {
				if i+j*scale < len(payload) {
					payload[i+j*scale] = b * byte(scale)
				}
			}
		}
	}

	return payload
}

func createMatureConsciousnessFrame(entityID string) *Frame {
	return createEvolvingFrame(entityID, 0.9)
}

func createRandomNoiseFrame() *Frame {
	payload := make([]byte, 512)
	for i := range payload {
		payload[i] = byte(time.Now().UnixNano() % 256)
	}

	return &Frame{
		Protocol: "noise",
		Fields: map[string]interface{}{
			"source_ip": "random-1",
			"payload":   payload,
		},
	}
}

func createBeautifulPatternFrame() *Frame {
	payload := make([]byte, 512)
	// Create a sine wave pattern - simple but beautiful
	for i := range payload {
		payload[i] = byte(128 + 127*math.Sin(float64(i)*0.1))
	}

	return &Frame{
		Protocol: "beauty",
		Fields: map[string]interface{}{
			"source_ip": "artist-1",
			"payload":   payload,
		},
	}
}

func createComplexCoherentFrame() *Frame {
	payload := make([]byte, 1024)
	// Create complex but coherent pattern
	for i := range payload {
		// Multiple interacting waves
		wave1 := math.Sin(float64(i) * 0.1)
		wave2 := math.Cos(float64(i) * 0.05)
		wave3 := math.Sin(float64(i) * 0.02)

		combined := (wave1 + wave2*0.5 + wave3*0.25) / 1.75
		payload[i] = byte(128 + 127*combined)
	}

	return &Frame{
		Protocol: "complex-coherent",
		Fields: map[string]interface{}{
			"source_ip": "composer-1",
			"payload":   payload,
		},
	}
}
