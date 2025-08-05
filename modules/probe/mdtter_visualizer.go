package probe

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// MDTTERVisualizer creates visual representations of multi-dimensional attack data
type MDTTERVisualizer struct {
	events     []*MDTTEREvent
	trajectory []TrajectoryPoint
}

// TrajectoryPoint represents a point in the attack trajectory
type TrajectoryPoint struct {
	Time        time.Time
	Position    []float32 // Position in manifold
	Velocity    []float32 // Tangent vector (rate of change)
	VAM         float32   // Distance from normal
	Intent      string    // Dominant intent
	ATTACKPhase string    // MITRE ATT&CK category
	Curvature   float32   // Local manifold curvature
}

// NewMDTTERVisualizer creates a new visualizer
func NewMDTTERVisualizer() *MDTTERVisualizer {
	return &MDTTERVisualizer{
		events:     make([]*MDTTEREvent, 0),
		trajectory: make([]TrajectoryPoint, 0),
	}
}

// AddEvent adds an MDTTER event to the visualization
func (v *MDTTERVisualizer) AddEvent(event *MDTTEREvent) {
	v.events = append(v.events, event)

	// Extract trajectory point
	point := TrajectoryPoint{
		Time:        event.Timestamp.AsTime(),
		Position:    event.BehavioralEmbedding[:3], // First 3 dims for visualization
		Velocity:    event.ManifoldDescriptor.TangentVector,
		VAM:         event.VarietyAbsorptionMetric,
		Intent:      dominantIntent(event.IntentField),
		ATTACKPhase: mapIntentToATTACK(event.IntentField),
		Curvature:   event.ManifoldDescriptor.Curvature,
	}

	v.trajectory = append(v.trajectory, point)
}

// GenerateTrajectoryVisualization creates ASCII art of the attack trajectory
func (v *MDTTERVisualizer) GenerateTrajectoryVisualization() string {
	if len(v.trajectory) == 0 {
		return "No trajectory data"
	}

	var output strings.Builder

	// Header
	output.WriteString("\n🌌 ATTACK TRAJECTORY THROUGH BEHAVIORAL MANIFOLD 🌌\n")
	output.WriteString("═══════════════════════════════════════════════════\n\n")

	// Time-based progression
	for i, point := range v.trajectory {
		// Visual representation of VAM (threat level)
		vamBar := generateBar(point.VAM, 20, "█", "░")

		// Show progression
		output.WriteString(fmt.Sprintf("[%s] ", point.Time.Format("15:04:05")))

		// Show ATT&CK phase with color codes
		phase := formatATTACKPhase(point.ATTACKPhase)
		output.WriteString(fmt.Sprintf("%-20s ", phase))

		// VAM visualization
		output.WriteString(fmt.Sprintf("VAM [%s] %.2f ", vamBar, point.VAM))

		// Show if crossing VAM threshold
		if point.VAM > 0.7 {
			output.WriteString("⚠️  DEFENSIVE TRIGGER!")
		}

		output.WriteString("\n")

		// Show velocity/direction change
		if i > 0 {
			velocityChange := calculateVelocityChange(
				v.trajectory[i-1].Velocity,
				point.Velocity,
			)
			if velocityChange > 0.5 {
				output.WriteString(fmt.Sprintf("     ↳ Direction change: %.1f° ", velocityChange*180/math.Pi))
				output.WriteString("🔄 BEHAVIOR SHIFT DETECTED\n")
			}
		}

		// Show curvature spikes
		if point.Curvature > 0.5 {
			output.WriteString(fmt.Sprintf("     ↳ High curvature: %.2f ", point.Curvature))
			output.WriteString("📈 COMPLEX MANEUVER\n")
		}
	}

	return output.String()
}

// GenerateATTACKTransitionMap shows movement between MITRE ATT&CK categories
func (v *MDTTERVisualizer) GenerateATTACKTransitionMap() string {
	if len(v.trajectory) < 2 {
		return "Insufficient data for transition analysis"
	}

	var output strings.Builder

	output.WriteString("\n🗺️  MITRE ATT&CK TRANSITION MAP 🗺️\n")
	output.WriteString("════════════════════════════════\n\n")

	// Track transitions
	transitions := make(map[string]int)
	var lastPhase string

	for _, point := range v.trajectory {
		if lastPhase != "" && lastPhase != point.ATTACKPhase {
			key := fmt.Sprintf("%s → %s", lastPhase, point.ATTACKPhase)
			transitions[key]++
		}
		lastPhase = point.ATTACKPhase
	}

	// Display transitions
	for transition, count := range transitions {
		output.WriteString(fmt.Sprintf("  %s", transition))
		if count > 1 {
			output.WriteString(fmt.Sprintf(" (×%d)", count))
		}
		output.WriteString("\n")
	}

	return output.String()
}

// GenerateComparativeAnalysis compares this attack to normal behavior
func (v *MDTTERVisualizer) GenerateComparativeAnalysis() string {
	if len(v.events) == 0 {
		return "No data for analysis"
	}

	var output strings.Builder

	output.WriteString("\n📊 DIMENSIONAL ANALYSIS COMPARISON 📊\n")
	output.WriteString("════════════════════════════════════\n\n")

	// Legacy SIEM view
	output.WriteString("LEGACY SIEM SEES:\n")
	output.WriteString("─────────────────\n")
	for _, event := range v.events {
		output.WriteString(fmt.Sprintf("  %s | %s → %s:%d | %s\n",
			event.Timestamp.AsTime().Format("15:04:05"),
			event.SourceIp,
			event.DestinationIp,
			event.DestinationPort,
			event.Protocol,
		))
	}

	output.WriteString("\nMDTTER SEES:\n")
	output.WriteString("────────────\n")

	// Calculate manifold statistics
	var totalCurvature float32
	var maxVAM float32
	intentCounts := make(map[string]int)

	for _, event := range v.events {
		totalCurvature += event.ManifoldDescriptor.Curvature
		if event.VarietyAbsorptionMetric > maxVAM {
			maxVAM = event.VarietyAbsorptionMetric
		}
		intent := dominantIntent(event.IntentField)
		intentCounts[intent]++
	}

	avgCurvature := totalCurvature / float32(len(v.events))

	output.WriteString(fmt.Sprintf("  • Behavioral Dimensions: 128\n"))
	output.WriteString(fmt.Sprintf("  • Trajectory Points: %d\n", len(v.trajectory)))
	output.WriteString(fmt.Sprintf("  • Max Novelty (VAM): %.2f\n", maxVAM))
	output.WriteString(fmt.Sprintf("  • Avg Curvature: %.2f\n", avgCurvature))
	output.WriteString(fmt.Sprintf("  • Topology Changes: %d\n", countTopologyChanges(v.events)))

	output.WriteString("\n  Intent Evolution:\n")
	for intent, count := range intentCounts {
		bar := generateBar(float32(count)/float32(len(v.events)), 10, "▓", "░")
		output.WriteString(fmt.Sprintf("    %-15s [%s] %d events\n", intent, bar, count))
	}

	return output.String()
}

// GenerateLiveUpdate creates a real-time update visualization
func (v *MDTTERVisualizer) GenerateLiveUpdate(event *MDTTEREvent) string {
	var output strings.Builder

	// Compact live view
	vamIndicator := "🟢"
	if event.VarietyAbsorptionMetric > 0.7 {
		vamIndicator = "🔴"
	} else if event.VarietyAbsorptionMetric > 0.4 {
		vamIndicator = "🟡"
	}

	intent := dominantIntent(event.IntentField)

	output.WriteString(fmt.Sprintf("%s %s | VAM: %.2f | %s | %s → %s | ",
		vamIndicator,
		event.Timestamp.AsTime().Format("15:04:05"),
		event.VarietyAbsorptionMetric,
		intent,
		event.SourceIp,
		event.DestinationIp,
	))

	// Show topology impact
	if len(event.TopologyChanges) > 0 {
		output.WriteString("🔄 TOPOLOGY SHIFT | ")
	}

	// Show if defensive trigger
	if event.VarietyAbsorptionMetric > 0.7 {
		output.WriteString("⚡ DEFENSIVE MORPH TRIGGERED")
	}

	return output.String()
}

// Helper functions

func generateBar(value float32, width int, filled, empty string) string {
	fillCount := int(value * float32(width))
	if fillCount > width {
		fillCount = width
	}
	return strings.Repeat(filled, fillCount) + strings.Repeat(empty, width-fillCount)
}

func formatATTACKPhase(phase string) string {
	// Add visual indicators for different phases
	indicators := map[string]string{
		"reconnaissance":       "🔍 Reconnaissance",
		"initial_access":       "🚪 Initial Access",
		"lateral_movement":     "↔️  Lateral Movement",
		"privilege_escalation": "⬆️  Privilege Esc",
		"data_collection":      "📦 Data Collection",
		"exfiltration":         "📤 Exfiltration",
		"impact":               "💥 Impact",
	}

	if formatted, ok := indicators[phase]; ok {
		return formatted
	}
	return phase
}

func mapIntentToATTACK(intent *IntentProbabilities) string {
	// Map to MITRE ATT&CK phases based on highest probability
	maxProb := float32(0)
	maxPhase := "unknown"

	phases := map[string]float32{
		"reconnaissance":       intent.Reconnaissance,
		"initial_access":       intent.InitialAccess,
		"lateral_movement":     intent.LateralMovement,
		"privilege_escalation": intent.PrivilegeEscalation,
		"data_collection":      intent.DataCollection,
		"exfiltration":         intent.Exfiltration,
		"impact":               intent.Impact,
	}

	for phase, prob := range phases {
		if prob > maxProb {
			maxProb = prob
			maxPhase = phase
		}
	}

	return maxPhase
}

func calculateVelocityChange(v1, v2 []float32) float32 {
	if len(v1) == 0 || len(v2) == 0 {
		return 0
	}

	// Calculate angle between velocity vectors
	var dotProduct float32
	var mag1, mag2 float32

	for i := 0; i < len(v1) && i < len(v2); i++ {
		dotProduct += v1[i] * v2[i]
		mag1 += v1[i] * v1[i]
		mag2 += v2[i] * v2[i]
	}

	if mag1 == 0 || mag2 == 0 {
		return 0
	}

	cosAngle := dotProduct / (float32(math.Sqrt(float64(mag1))) * float32(math.Sqrt(float64(mag2))))
	return float32(math.Acos(float64(cosAngle)))
}

func countTopologyChanges(events []*MDTTEREvent) int {
	count := 0
	for _, event := range events {
		count += len(event.TopologyChanges)
	}
	return count
}

// ExportForD3JS exports trajectory data for 3D visualization
func (v *MDTTERVisualizer) ExportForD3JS() ([]byte, error) {
	data := map[string]interface{}{
		"nodes":      make([]map[string]interface{}, 0),
		"links":      make([]map[string]interface{}, 0),
		"trajectory": make([]map[string]interface{}, 0),
	}

	// Export trajectory points
	for i, point := range v.trajectory {
		trajPoint := map[string]interface{}{
			"id":        i,
			"time":      point.Time.Unix(),
			"x":         point.Position[0],
			"y":         point.Position[1],
			"z":         point.Position[2],
			"vam":       point.VAM,
			"intent":    point.Intent,
			"phase":     point.ATTACKPhase,
			"curvature": point.Curvature,
		}
		data["trajectory"] = append(data["trajectory"].([]map[string]interface{}), trajPoint)
	}

	// Add VAM boundary surface points
	// This would generate points representing the VAM=0.7 threshold surface

	return json.MarshalIndent(data, "", "  ")
}
