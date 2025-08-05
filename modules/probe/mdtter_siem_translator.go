package probe

import (
	"fmt"
	"time"
)

// SIEMTranslator converts rich MDTTER events to flat formats for legacy systems
type SIEMTranslator struct {
	format SIEMFormat
	// Track what dimensions they can't see yet
	lostDimensions map[string]interface{}
}

type SIEMFormat string

const (
	FormatCEF     SIEMFormat = "cef"     // Common Event Format
	FormatLEEF    SIEMFormat = "leef"    // Log Event Extended Format
	FormatJSON    SIEMFormat = "json"    // Simplified JSON
	FormatSyslog  SIEMFormat = "syslog"  // Traditional syslog
	FormatSplunk  SIEMFormat = "splunk"  // Splunk HEC
	FormatElastic SIEMFormat = "elastic" // Elasticsearch
)

// NewSIEMTranslator creates a translator for legacy systems
func NewSIEMTranslator(format SIEMFormat) *SIEMTranslator {
	return &SIEMTranslator{
		format:         format,
		lostDimensions: make(map[string]interface{}),
	}
}

// TranslateToLegacy converts MDTTER to flat format while preserving what we can
func (t *SIEMTranslator) TranslateToLegacy(event *MDTTEREvent) (interface{}, error) {
	switch t.format {
	case FormatCEF:
		return t.toCEF(event), nil
	case FormatJSON:
		return t.toSimplifiedJSON(event), nil
	case FormatSplunk:
		return t.toSplunkHEC(event), nil
	case FormatElastic:
		return t.toElastic(event), nil
	default:
		return t.toGenericFlat(event), nil
	}
}

// toCEF converts to ArcSight Common Event Format
func (t *SIEMTranslator) toCEF(event *MDTTEREvent) string {
	// CEF:Version|Device Vendor|Device Product|Device Version|Signature ID|Name|Severity|Extension

	// Map VAM to legacy severity (0-10)
	severity := int(event.VarietyAbsorptionMetric * 10)

	// Flatten the rich intent into a simple category
	intent := t.flattenIntent(event.IntentField)

	// Create CEF extension with as much dimensional data as CEF can handle
	extension := fmt.Sprintf(
		"src=%s dst=%s dpt=%d proto=%s act=%s cs1=%.2f cs1Label=VAM cs2=%s cs2Label=Intent cs3=%.2f cs3Label=Curvature cs4=%.2f cs4Label=DistanceFromNormal cs5=%s cs5Label=TopologyNode",
		event.SourceIp,
		event.DestinationIp,
		event.DestinationPort,
		event.Protocol,
		event.Action,
		event.VarietyAbsorptionMetric,
		intent,
		event.ManifoldDescriptor.Curvature,
		event.ManifoldDescriptor.DistanceFromNormal,
		event.AstPosition.NodeId,
	)

	// Track what dimensions we're losing
	t.trackLostDimensions(event)

	return fmt.Sprintf(
		"CEF:0|Strigoi|MDTTER|1.0|MDTTER-%s|%s Behavioral Anomaly|%d|%s",
		event.EventId[:8],
		intent,
		severity,
		extension,
	)
}

// toSimplifiedJSON creates JSON that legacy parsers can handle
func (t *SIEMTranslator) toSimplifiedJSON(event *MDTTEREvent) map[string]interface{} {
	// Flatten the multi-dimensional data into something parseable
	flat := map[string]interface{}{
		"timestamp":        event.Timestamp.AsTime().Format(time.RFC3339),
		"event_id":         event.EventId,
		"source_ip":        event.SourceIp,
		"destination_ip":   event.DestinationIp,
		"destination_port": event.DestinationPort,
		"protocol":         event.Protocol,
		"action":           event.Action,
		"severity":         t.calculateLegacySeverity(event),
		"category":         t.flattenIntent(event.IntentField),
		"risk_score":       event.VarietyAbsorptionMetric * 100,

		// Preserve some dimensional data in custom fields
		"mdtter_vam":       event.VarietyAbsorptionMetric,
		"mdtter_curvature": event.ManifoldDescriptor.Curvature,
		"mdtter_distance":  event.ManifoldDescriptor.DistanceFromNormal,
		"mdtter_topology":  event.AstPosition.NodeId,

		// Intent probabilities as percentages
		"intent_recon":   int(event.IntentField.Reconnaissance * 100),
		"intent_lateral": int(event.IntentField.LateralMovement * 100),
		"intent_exfil":   int(event.IntentField.Exfiltration * 100),

		// Hint at the richer data available
		"mdtter_dimensions": len(event.BehavioralEmbedding),
		"mdtter_enabled":    true,
		"message":           t.generateLegacyMessage(event),
	}

	return flat
}

// toSplunkHEC creates Splunk HTTP Event Collector format
func (t *SIEMTranslator) toSplunkHEC(event *MDTTEREvent) map[string]interface{} {
	return map[string]interface{}{
		"time":       event.Timestamp.AsTime().Unix(),
		"host":       event.SourceIp,
		"source":     "strigoi-mdtter",
		"sourcetype": "mdtter:security",
		"event": map[string]interface{}{
			"mdtter_id":          event.EventId,
			"src_ip":             event.SourceIp,
			"dest_ip":            event.DestinationIp,
			"dest_port":          event.DestinationPort,
			"protocol":           event.Protocol,
			"action":             event.Action,
			"vam_score":          event.VarietyAbsorptionMetric,
			"threat_category":    t.flattenIntent(event.IntentField),
			"behavioral_anomaly": event.ManifoldDescriptor.DistanceFromNormal > 2.0,
			"topology_change":    len(event.TopologyChanges) > 0,

			// Searchable intent fields
			"is_reconnaissance": event.IntentField.Reconnaissance > 0.5,
			"is_lateral":        event.IntentField.LateralMovement > 0.5,
			"is_exfiltration":   event.IntentField.Exfiltration > 0.5,

			// Preserve dimensional hints
			"mdtter_version":  "1.0",
			"full_dimensions": len(event.BehavioralEmbedding),
		},
	}
}

// toElastic creates Elasticsearch-compatible format
func (t *SIEMTranslator) toElastic(event *MDTTEREvent) map[string]interface{} {
	doc := map[string]interface{}{
		"@timestamp":       event.Timestamp.AsTime(),
		"event.id":         event.EventId,
		"event.kind":       "event",
		"event.category":   []string{"network", "threat"},
		"event.type":       []string{t.flattenIntent(event.IntentField)},
		"event.risk_score": event.VarietyAbsorptionMetric * 100,

		// ECS fields
		"source.ip":        event.SourceIp,
		"destination.ip":   event.DestinationIp,
		"destination.port": event.DestinationPort,
		"network.protocol": event.Protocol,

		// MDTTER fields (properly namespaced)
		"mdtter.vam": event.VarietyAbsorptionMetric,
		"mdtter.manifold": map[string]interface{}{
			"curvature":            event.ManifoldDescriptor.Curvature,
			"distance_from_normal": event.ManifoldDescriptor.DistanceFromNormal,
		},
		"mdtter.topology": map[string]interface{}{
			"node_id":         event.AstPosition.NodeId,
			"connected_nodes": len(event.AstPosition.ConnectedNodes),
		},
		"mdtter.intent": map[string]interface{}{
			"reconnaissance":   event.IntentField.Reconnaissance,
			"lateral_movement": event.IntentField.LateralMovement,
			"exfiltration":     event.IntentField.Exfiltration,
			"data_collection":  event.IntentField.DataCollection,
		},

		// Preserve embedding summary
		"mdtter.embedding_dimensions": len(event.BehavioralEmbedding),
		"mdtter.topology_changes":     len(event.TopologyChanges),

		// Legacy-friendly message
		"message": t.generateLegacyMessage(event),
	}

	// Add labels for easy searching
	labels := []string{}
	if event.VarietyAbsorptionMetric > 0.7 {
		labels = append(labels, "high_novelty")
	}
	if event.IntentField.Exfiltration > 0.5 {
		labels = append(labels, "possible_exfiltration")
	}
	if len(event.TopologyChanges) > 0 {
		labels = append(labels, "topology_shift")
	}
	if len(labels) > 0 {
		doc["labels"] = labels
	}

	return doc
}

// Helper functions

func (t *SIEMTranslator) flattenIntent(intent *IntentProbabilities) string {
	// Find dominant intent for legacy systems that need a single category
	maxProb := float32(0)
	maxIntent := "unknown"

	intents := map[string]float32{
		"reconnaissance":       intent.Reconnaissance,
		"lateral_movement":     intent.LateralMovement,
		"exfiltration":         intent.Exfiltration,
		"privilege_escalation": intent.PrivilegeEscalation,
		"data_collection":      intent.DataCollection,
		"initial_access":       intent.InitialAccess,
		"impact":               intent.Impact,
	}

	for name, prob := range intents {
		if prob > maxProb {
			maxProb = prob
			maxIntent = name
		}
	}

	return maxIntent
}

func (t *SIEMTranslator) calculateLegacySeverity(event *MDTTEREvent) string {
	// Map our rich metrics to legacy severity levels
	if event.VarietyAbsorptionMetric > 0.8 {
		return "critical"
	} else if event.VarietyAbsorptionMetric > 0.6 {
		return "high"
	} else if event.VarietyAbsorptionMetric > 0.4 {
		return "medium"
	} else if event.VarietyAbsorptionMetric > 0.2 {
		return "low"
	}
	return "info"
}

func (t *SIEMTranslator) generateLegacyMessage(event *MDTTEREvent) string {
	// Create a human-readable message for legacy systems
	intent := t.flattenIntent(event.IntentField)
	severity := t.calculateLegacySeverity(event)

	return fmt.Sprintf(
		"MDTTER: %s activity detected from %s to %s:%d (VAM: %.2f, Intent: %s, Severity: %s)",
		intent,
		event.SourceIp,
		event.DestinationIp,
		event.DestinationPort,
		event.VarietyAbsorptionMetric,
		intent,
		severity,
	)
}

func (t *SIEMTranslator) trackLostDimensions(event *MDTTEREvent) {
	// Track what dimensional richness is lost in translation
	t.lostDimensions["behavioral_embedding"] = len(event.BehavioralEmbedding)
	t.lostDimensions["topology_connections"] = len(event.AstPosition.ConnectedNodes)
	t.lostDimensions["manifold_dimensions"] = 4 // point, tangent, curvature, distance
	t.lostDimensions["intent_probabilities"] = 7
	t.lostDimensions["topology_changes"] = len(event.TopologyChanges)

	// This data helps us show legacy systems what they're missing
}

func (t *SIEMTranslator) toGenericFlat(event *MDTTEREvent) map[string]interface{} {
	// Ultimate fallback - the flattest possible representation
	return map[string]interface{}{
		"timestamp": event.Timestamp.AsTime().Format(time.RFC3339),
		"source":    event.SourceIp,
		"dest":      event.DestinationIp,
		"port":      event.DestinationPort,
		"protocol":  event.Protocol,
		"action":    event.Action,
		"severity":  t.calculateLegacySeverity(event),
		"category":  t.flattenIntent(event.IntentField),
		"message":   t.generateLegacyMessage(event),

		// At least hint at our capabilities
		"mdtter_enabled":    true,
		"upgrade_available": "Contact Strigoi for multi-dimensional analysis",
	}
}

// GetLostDimensions returns what dimensional data was lost in translation
func (t *SIEMTranslator) GetLostDimensions() map[string]interface{} {
	return t.lostDimensions
}

// BatchTranslate converts multiple MDTTER events for bulk ingestion
func (t *SIEMTranslator) BatchTranslate(events []*MDTTEREvent) ([]interface{}, error) {
	results := make([]interface{}, len(events))
	for i, event := range events {
		translated, err := t.TranslateToLegacy(event)
		if err != nil {
			return nil, fmt.Errorf("failed to translate event %s: %w", event.EventId, err)
		}
		results[i] = translated
	}
	return results, nil
}
