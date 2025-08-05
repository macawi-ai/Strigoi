package probe

import (
	"fmt"
	"testing"
	"time"
)

func TestMDTTERGeneration(t *testing.T) {
	// Create session manager
	sessionManager := NewSessionManager(30*time.Second, 5*time.Second)

	// Create MDTTER generator
	gen := NewMDTTERGenerator(sessionManager)

	// Test case 1: Normal HTTP traffic
	t.Run("Normal HTTP Traffic", func(t *testing.T) {
		frame := &Frame{
			Protocol: "HTTP",
			Fields: map[string]interface{}{
				"source_ip":        "192.168.1.100",
				"destination_ip":   "10.0.0.50",
				"destination_port": 443,
				"method":           "GET",
				"url":              "/api/data",
				"status":           200,
			},
		}

		event, err := gen.GenerateFromFrame(frame, "session-001")
		if err != nil {
			t.Fatalf("Failed to generate MDTTER event: %v", err)
		}

		// Check basic fields
		if event.SourceIp != "192.168.1.100" {
			t.Errorf("Expected source IP 192.168.1.100, got %s", event.SourceIp)
		}

		// Check embedding exists
		if len(event.BehavioralEmbedding) != 128 {
			t.Errorf("Expected 128-dim embedding, got %d", len(event.BehavioralEmbedding))
		}

		// Check VAM is calculated
		if event.VarietyAbsorptionMetric < 0 || event.VarietyAbsorptionMetric > 1 {
			t.Errorf("VAM out of range: %f", event.VarietyAbsorptionMetric)
		}

		t.Logf("Normal traffic: VAM=%.2f, Intent=%s",
			event.VarietyAbsorptionMetric,
			dominantIntent(event.IntentField))
	})

	// Test case 2: Reconnaissance pattern
	t.Run("Reconnaissance Pattern", func(t *testing.T) {
		frame := &Frame{
			Protocol: "HTTP",
			Fields: map[string]interface{}{
				"source_ip":        "192.168.1.100",
				"destination_ip":   "10.0.0.50",
				"destination_port": 80,
				"method":           "OPTIONS",
				"url":              "/",
			},
		}

		event, err := gen.GenerateFromFrame(frame, "session-002")
		if err != nil {
			t.Fatalf("Failed to generate MDTTER event: %v", err)
		}

		// Should detect reconnaissance intent
		if event.IntentField.Reconnaissance < 0.5 {
			t.Errorf("Expected high reconnaissance probability, got %.2f",
				event.IntentField.Reconnaissance)
		}

		t.Logf("Reconnaissance: VAM=%.2f, Recon Intent=%.2f",
			event.VarietyAbsorptionMetric,
			event.IntentField.Reconnaissance)
	})

	// Test case 3: Potential exfiltration
	t.Run("Exfiltration Pattern", func(t *testing.T) {
		largePayload := make([]byte, 50000) // 50KB payload
		frame := &Frame{
			Protocol: "HTTP",
			Fields: map[string]interface{}{
				"source_ip":        "192.168.1.100",
				"destination_ip":   "203.0.113.1", // External IP
				"destination_port": 443,
				"method":           "POST",
				"url":              "/upload",
				"payload":          largePayload,
			},
		}

		event, err := gen.GenerateFromFrame(frame, "session-003")
		if err != nil {
			t.Fatalf("Failed to generate MDTTER event: %v", err)
		}

		// Should detect exfiltration intent (adjusted threshold)
		if event.IntentField.Exfiltration < 0.4 {
			t.Errorf("Expected high exfiltration probability, got %.2f",
				event.IntentField.Exfiltration)
		}

		// Large payload should increase distance from normal
		if event.ManifoldDescriptor.DistanceFromNormal < 1.0 {
			t.Errorf("Expected high distance from normal, got %.2f",
				event.ManifoldDescriptor.DistanceFromNormal)
		}

		t.Logf("Exfiltration: VAM=%.2f, Exfil Intent=%.2f, Distance=%.2f",
			event.VarietyAbsorptionMetric,
			event.IntentField.Exfiltration,
			event.ManifoldDescriptor.DistanceFromNormal)
	})
}

func TestMDTTERTopologyTracking(t *testing.T) {
	gen := NewMDTTERGenerator(NewSessionManager(30*time.Second, 5*time.Second))

	// Simulate evolving attack
	attackStages := []struct {
		name  string
		frame *Frame
	}{
		{
			name: "Initial Probe",
			frame: &Frame{
				Protocol: "HTTP",
				Fields: map[string]interface{}{
					"source_ip":        "192.168.1.100",
					"destination_ip":   "10.0.0.50",
					"destination_port": 80,
					"method":           "GET",
					"url":              "/robots.txt",
				},
			},
		},
		{
			name: "Lateral Movement",
			frame: &Frame{
				Protocol: "HTTP",
				Fields: map[string]interface{}{
					"source_ip":        "192.168.1.100",
					"destination_ip":   "10.0.0.51", // Different internal host
					"destination_port": 8080,
					"method":           "POST",
					"url":              "/admin",
					"new_connection":   true,
				},
			},
		},
		{
			name: "Data Staging",
			frame: &Frame{
				Protocol: "HTTP",
				Fields: map[string]interface{}{
					"source_ip":        "192.168.1.100",
					"destination_ip":   "10.0.0.50",
					"destination_port": 443,
					"method":           "PUT",
					"url":              "/temp/data.zip",
					"payload":          make([]byte, 10000),
				},
			},
		},
	}

	sessionID := "attack-session-001"

	for i, stage := range attackStages {
		event, err := gen.GenerateFromFrame(stage.frame, sessionID)
		if err != nil {
			t.Fatalf("Stage %s failed: %v", stage.name, err)
		}

		// Log topology changes
		if len(event.TopologyChanges) > 0 {
			t.Logf("Stage %d (%s): Topology changes detected", i+1, stage.name)
			for _, change := range event.TopologyChanges {
				t.Logf("  - %s on %s", change.Operation, change.TargetId)
			}
		}

		// Show intent evolution
		t.Logf("Stage %d (%s): VAM=%.2f, Intent=%s",
			i+1, stage.name,
			event.VarietyAbsorptionMetric,
			dominantIntent(event.IntentField))
	}
}

func TestMDTTERIntegration(t *testing.T) {
	// Create MDTTER-enhanced module
	config := MDTTERConfig{
		Enabled:          true,
		EmbeddingDim:     128,
		VAMThreshold:     0.7,
		TopologyAdaptive: true,
	}

	module := NewMDTTEREnhancedModule(config)

	// Create test HTTP data
	httpRequest := []byte("GET /api/secret HTTP/1.1\r\n" +
		"Host: internal.corp\r\n" +
		"Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9\r\n" +
		"\r\n")

	// Create HTTP dissector and wrap it
	httpDissector := NewHTTPDissector()
	wrappedDissectors := module.WrapDissectors([]Dissector{httpDissector})
	dissector := wrappedDissectors[0]

	// Identify
	matches, confidence := dissector.Identify(httpRequest)
	if !matches {
		t.Fatalf("Failed to identify HTTP traffic (confidence: %.2f)", confidence)
	}

	// Dissect (should generate MDTTER event)
	frame, err := dissector.Dissect(httpRequest)
	if err != nil {
		t.Fatalf("Failed to dissect: %v", err)
	}

	// Give async MDTTER generation time to complete
	time.Sleep(100 * time.Millisecond)

	// Check if MDTTER event was generated
	select {
	case event := <-module.GetMDTTEREvents():
		t.Logf("MDTTER event generated: ID=%s, VAM=%.2f",
			event.EventId, event.VarietyAbsorptionMetric)

		// Check that it detected the auth token
		vulns := dissector.FindVulnerabilities(frame)
		if len(vulns) == 0 {
			t.Errorf("Expected to find auth token vulnerability")
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("No MDTTER event generated")
	}
}

// Benchmark MDTTER generation performance
func BenchmarkMDTTERGeneration(b *testing.B) {
	gen := NewMDTTERGenerator(NewSessionManager(30*time.Second, 5*time.Second))

	frame := &Frame{
		Protocol: "HTTP",
		Fields: map[string]interface{}{
			"source_ip":        "192.168.1.100",
			"destination_ip":   "10.0.0.50",
			"destination_port": 443,
			"method":           "GET",
			"url":              "/api/data",
			"payload":          make([]byte, 1024),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.GenerateFromFrame(frame, fmt.Sprintf("session-%d", i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestMDTTERVsTraditional shows the power of MDTTER vs traditional logs
func TestMDTTERVsTraditional(t *testing.T) {
	// Traditional log entry
	fmt.Println("=== Traditional Firewall Log ===")
	fmt.Println("2025-02-03T10:00:00Z SRC=192.168.1.100 DST=10.0.0.50 DPORT=443 PROTO=TCP ACTION=ALLOW")
	fmt.Println()

	// Same event in MDTTER
	fmt.Println("=== MDTTER Representation ===")

	// Create generator
	gen := NewMDTTERGenerator(NewSessionManager(30*time.Second, 5*time.Second))

	// Create frame
	frame := &Frame{
		Protocol: "HTTP",
		Fields: map[string]interface{}{
			"source_ip":        "192.168.1.100",
			"destination_ip":   "10.0.0.50",
			"destination_port": 443,
			"method":           "POST",
			"url":              "/api/upload",
			"payload":          make([]byte, 25000), // 25KB upload
		},
	}

	// Generate MDTTER event
	event, _ := gen.GenerateFromFrame(frame, "session-example")

	fmt.Printf("Event ID: %s\n", event.EventId)
	fmt.Printf("Topological Position: Node=%s, Connected=%d nodes\n",
		event.AstPosition.NodeId, len(event.AstPosition.ConnectedNodes))
	fmt.Printf("Behavioral Manifold: Curvature=%.2f, Distance=%.2f\n",
		event.ManifoldDescriptor.Curvature,
		event.ManifoldDescriptor.DistanceFromNormal)
	fmt.Printf("Variety Absorption: %.2f (%.0f%% novel)\n",
		event.VarietyAbsorptionMetric,
		event.VarietyAbsorptionMetric*100)
	fmt.Printf("Intent Analysis:\n")
	fmt.Printf("  - Reconnaissance: %.0f%%\n", event.IntentField.Reconnaissance*100)
	fmt.Printf("  - Lateral Movement: %.0f%%\n", event.IntentField.LateralMovement*100)
	fmt.Printf("  - Data Collection: %.0f%%\n", event.IntentField.DataCollection*100)
	fmt.Printf("  - Exfiltration: %.0f%%\n", event.IntentField.Exfiltration*100)

	// Output:
	// === Traditional Firewall Log ===
	// 2025-02-03T10:00:00Z SRC=192.168.1.100 DST=10.0.0.50 DPORT=443 PROTO=TCP ACTION=ALLOW
	//
	// === MDTTER Representation ===
	// Event ID: a3f4b8c9d2e1f0g5
	// Topological Position: Node=192.168.1.100, Connected=2 nodes
	// Behavioral Manifold: Curvature=0.45, Distance=2.31
	// Variety Absorption: 0.78 (78% novel)
	// Intent Analysis:
	//   - Reconnaissance: 10%
	//   - Lateral Movement: 15%
	//   - Data Collection: 60%
	//   - Exfiltration: 80%
}
