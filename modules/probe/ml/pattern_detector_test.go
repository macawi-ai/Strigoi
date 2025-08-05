package ml

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPatternDetector(t *testing.T) {
	config := DetectorConfig{
		ModelType:           "hybrid",
		SupervisedThreshold: 0.7,
		AnomalyThreshold:    0.8,
		EnableLLM:           false, // Disable for testing
		BatchSize:           10,
		WindowSize:          1 * time.Minute,
		MaxFeatures:         50,
	}

	detector, err := NewPatternDetector(config)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}
	defer detector.Close()

	// Test single event analysis
	t.Run("SingleEventAnalysis", func(t *testing.T) {
		event := &SecurityEvent{
			ID:          "test-001",
			Timestamp:   time.Now(),
			Type:        "network",
			Source:      "192.168.1.100",
			Destination: "10.0.0.1",
			Protocol:    "tcp",
			Payload:     []byte("GET /admin HTTP/1.1\r\nHost: target.com\r\n"),
		}

		result, err := detector.Analyze(context.Background(), event)
		if err != nil {
			t.Fatalf("Analysis failed: %v", err)
		}

		if result.EventID != event.ID {
			t.Errorf("Expected event ID %s, got %s", event.ID, result.EventID)
		}

		if result.ThreatScore < 0 || result.ThreatScore > 1 {
			t.Errorf("Invalid threat score: %f", result.ThreatScore)
		}
	})

	// Test batch analysis
	t.Run("BatchAnalysis", func(t *testing.T) {
		events := generateTestEvents(5)

		results, err := detector.AnalyzeBatch(context.Background(), events)
		if err != nil {
			t.Fatalf("Batch analysis failed: %v", err)
		}

		if len(results) != len(events) {
			t.Errorf("Expected %d results, got %d", len(events), len(results))
		}

		for i, result := range results {
			if result.EventID != events[i].ID {
				t.Errorf("Result %d has wrong event ID", i)
			}
		}
	})

	// Test pattern detection
	t.Run("PatternDetection", func(t *testing.T) {
		// Generate burst pattern
		burstEvents := make([]*SecurityEvent, 10)
		baseTime := time.Now()

		for i := 0; i < 10; i++ {
			burstEvents[i] = &SecurityEvent{
				ID:          fmt.Sprintf("burst-%d", i),
				Timestamp:   baseTime.Add(time.Duration(i) * 100 * time.Millisecond),
				Type:        "network",
				Source:      "192.168.1.100",
				Destination: "10.0.0.1",
				Protocol:    "tcp",
				Payload:     []byte("scan"),
			}
		}

		results, err := detector.AnalyzeBatch(context.Background(), burstEvents)
		if err != nil {
			t.Fatalf("Pattern analysis failed: %v", err)
		}

		// Check if burst pattern was detected
		patternFound := false
		for _, result := range results {
			for _, pattern := range result.Patterns {
				if pattern.Type == "temporal_burst" {
					patternFound = true
					break
				}
			}
		}

		if !patternFound {
			t.Error("Expected burst pattern to be detected")
		}
	})

	// Test metrics
	t.Run("Metrics", func(t *testing.T) {
		metrics := detector.GetMetrics()

		if metrics.TotalAnalyzed == 0 {
			t.Error("Expected non-zero total analyzed count")
		}

		if metrics.ThreatRate < 0 || metrics.ThreatRate > 1 {
			t.Errorf("Invalid threat rate: %f", metrics.ThreatRate)
		}
	})
}

func TestFeatureExtractor(t *testing.T) {
	extractor := NewFeatureExtractor(50)

	t.Run("BasicFeatures", func(t *testing.T) {
		event := &SecurityEvent{
			ID:          "test-001",
			Timestamp:   time.Now(),
			Type:        "network",
			Source:      "192.168.1.100",
			Destination: "10.0.0.1",
			Protocol:    "tcp",
			Payload:     []byte("test payload"),
		}

		features, err := extractor.Extract(event)
		if err != nil {
			t.Fatalf("Feature extraction failed: %v", err)
		}

		if len(features) == 0 {
			t.Error("Expected non-empty features")
		}

		if len(features) > 50 {
			t.Errorf("Expected max 50 features, got %d", len(features))
		}

		// Check normalization
		for i, f := range features {
			if f < 0 || f > 1 {
				t.Errorf("Feature %d not normalized: %f", i, f)
			}
		}
	})

	t.Run("PayloadFeatures", func(t *testing.T) {
		// Test with suspicious payload
		suspiciousPayload := []byte{
			0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, // NOP sled
			0x31, 0xc0, // xor eax, eax
			0xff, 0xe4, // jmp esp
		}

		event := &SecurityEvent{
			ID:        "test-002",
			Timestamp: time.Now(),
			Type:      "network",
			Payload:   suspiciousPayload,
		}

		features, err := extractor.Extract(event)
		if err != nil {
			t.Fatalf("Feature extraction failed: %v", err)
		}

		// Should detect suspicious patterns
		// Last feature is suspicious pattern count
		if features[len(features)-1] == 0 {
			t.Error("Expected suspicious patterns to be detected")
		}
	})

	t.Run("TemporalFeatures", func(t *testing.T) {
		// Add historical events
		for i := 0; i < 5; i++ {
			event := &SecurityEvent{
				ID:        fmt.Sprintf("hist-%d", i),
				Timestamp: time.Now().Add(-time.Duration(i) * time.Second),
				Type:      "network",
				Source:    "192.168.1.100",
			}
			features, _ := extractor.Extract(event)
			_ = features // History is updated
		}

		// Extract features with history
		newEvent := &SecurityEvent{
			ID:        "new-001",
			Timestamp: time.Now(),
			Type:      "network",
			Source:    "192.168.1.100",
		}

		features, err := extractor.Extract(newEvent)
		if err != nil {
			t.Fatalf("Feature extraction failed: %v", err)
		}

		// Should have temporal features from history
		if len(features) == 0 {
			t.Error("Expected features with history")
		}
	})
}

func TestModels(t *testing.T) {
	t.Run("RandomForest", func(t *testing.T) {
		rf := NewRandomForestClassifier()

		// Create training data
		features := [][]float64{
			{0.1, 0.2, 0.3, 0.4},
			{0.8, 0.7, 0.6, 0.5},
			{0.2, 0.3, 0.4, 0.5},
			{0.9, 0.8, 0.7, 0.6},
			{0.3, 0.4, 0.5, 0.6},
		}

		labels := [][]string{
			{"benign"},
			{"malware"},
			{"benign"},
			{"malware"},
			{"scanning"},
		}

		// Train model
		err := rf.Train(features, labels)
		if err != nil {
			t.Fatalf("Training failed: %v", err)
		}

		// Test classification
		testFeatures := []float64{0.85, 0.75, 0.65, 0.55}
		classifications, err := rf.Classify(testFeatures)
		if err != nil {
			t.Fatalf("Classification failed: %v", err)
		}

		if len(classifications) == 0 {
			t.Error("Expected classifications")
		}

		// Check probabilities sum approximately to 1
		totalProb := 0.0
		for _, class := range classifications {
			totalProb += class.Probability
		}

		// Allow some tolerance due to thresholding
		if totalProb < 0.9 || totalProb > 1.1 {
			t.Errorf("Probabilities sum to %f, expected ~1.0", totalProb)
		}
	})

	t.Run("IsolationForest", func(t *testing.T) {
		iforest := NewIsolationForest(0.7)

		// Create normal data
		normalData := [][]float64{
			{0.1, 0.2, 0.3},
			{0.2, 0.3, 0.4},
			{0.15, 0.25, 0.35},
			{0.18, 0.28, 0.38},
			{0.12, 0.22, 0.32},
		}

		// Train model
		err := iforest.Update(normalData)
		if err != nil {
			t.Fatalf("Training failed: %v", err)
		}

		// Test normal point
		normalPoint := []float64{0.16, 0.26, 0.36}
		score, err := iforest.DetectAnomaly(normalPoint)
		if err != nil {
			t.Fatalf("Anomaly detection failed: %v", err)
		}

		if score > 0.7 {
			t.Errorf("Normal point scored too high: %f", score)
		}

		// Test anomalous point
		anomalousPoint := []float64{0.9, 0.9, 0.9}
		score, err = iforest.DetectAnomaly(anomalousPoint)
		if err != nil {
			t.Fatalf("Anomaly detection failed: %v", err)
		}

		if score < 0.5 {
			t.Errorf("Anomalous point scored too low: %f", score)
		}
	})
}

func TestLLMAccelerator(t *testing.T) {
	t.Run("LocalLLM", func(t *testing.T) {
		llm, err := NewLocalLLMAccelerator("test-model")
		if err != nil {
			t.Fatalf("Failed to create LLM: %v", err)
		}
		defer llm.Close()

		event := &SecurityEvent{
			ID:          "test-001",
			Type:        "network",
			Source:      "192.168.1.100",
			Destination: "10.0.0.1",
			Protocol:    "tcp",
		}

		result := &DetectionResult{
			EventID:     event.ID,
			ThreatScore: 0.85,
			Anomalous:   true,
			Classifications: []Classification{
				{Category: "malware", Probability: 0.9},
			},
		}

		explanation, confidence, err := llm.AnalyzeEvent(context.Background(), event, result)
		if err != nil {
			t.Fatalf("LLM analysis failed: %v", err)
		}

		if explanation == "" {
			t.Error("Expected non-empty explanation")
		}

		if confidence < 0 || confidence > 1 {
			t.Errorf("Invalid confidence: %f", confidence)
		}

		// Check explanation contains key elements
		if !strings.Contains(explanation, "Threat Assessment") {
			t.Error("Explanation missing threat assessment")
		}

		if !strings.Contains(explanation, "Recommended Actions") {
			t.Error("Explanation missing recommendations")
		}
	})
}

// Helper functions

func generateTestEvents(count int) []*SecurityEvent {
	events := make([]*SecurityEvent, count)
	protocols := []string{"tcp", "udp", "http", "https"}
	types := []string{"network", "file", "process"}

	for i := 0; i < count; i++ {
		events[i] = &SecurityEvent{
			ID:          fmt.Sprintf("test-%03d", i),
			Timestamp:   time.Now().Add(-time.Duration(count-i) * time.Second),
			Type:        types[i%len(types)],
			Source:      fmt.Sprintf("192.168.1.%d", 100+i),
			Destination: fmt.Sprintf("10.0.0.%d", 1+i),
			Protocol:    protocols[i%len(protocols)],
			Payload:     []byte(fmt.Sprintf("test payload %d", i)),
		}
	}

	return events
}

func BenchmarkFeatureExtraction(b *testing.B) {
	extractor := NewFeatureExtractor(50)
	event := &SecurityEvent{
		ID:          "bench-001",
		Timestamp:   time.Now(),
		Type:        "network",
		Source:      "192.168.1.100",
		Destination: "10.0.0.1",
		Protocol:    "tcp",
		Payload:     make([]byte, 1024),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractor.Extract(event)
	}
}

func BenchmarkPatternDetection(b *testing.B) {
	config := DetectorConfig{
		ModelType:   "unsupervised",
		EnableLLM:   false,
		MaxFeatures: 50,
	}

	detector, _ := NewPatternDetector(config)
	defer detector.Close()

	event := &SecurityEvent{
		ID:          "bench-001",
		Timestamp:   time.Now(),
		Type:        "network",
		Source:      "192.168.1.100",
		Destination: "10.0.0.1",
		Protocol:    "tcp",
		Payload:     make([]byte, 256),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.Analyze(ctx, event)
	}
}
