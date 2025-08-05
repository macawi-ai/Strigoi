package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// PatternDetector provides ML-based pattern detection for security events
type PatternDetector struct {
	config            DetectorConfig
	supervisedModel   SupervisedModel
	unsupervisedModel UnsupervisedModel
	llmAccelerator    LLMAccelerator
	featureExtractor  *FeatureExtractor
	eventBuffer       *EventBuffer
	metrics           *DetectorMetrics
	mu                sync.RWMutex
}

// DetectorConfig configures the pattern detection system
type DetectorConfig struct {
	// Model configuration
	ModelType           string  // "hybrid", "supervised", "unsupervised"
	SupervisedThreshold float64 // Confidence threshold for supervised model
	AnomalyThreshold    float64 // Threshold for anomaly detection

	// LLM acceleration
	EnableLLM    bool
	LLMProvider  string // "openai", "anthropic", "local"
	LLMModel     string // Model identifier
	LLMBatchSize int

	// Processing configuration
	BatchSize      int
	WindowSize     time.Duration
	UpdateInterval time.Duration

	// Feature extraction
	MaxFeatures  int
	FeatureTypes []string // "statistical", "temporal", "behavioral"
}

// SecurityEvent represents an event to analyze
type SecurityEvent struct {
	ID          string
	Timestamp   time.Time
	Type        string
	Source      string
	Destination string
	Protocol    string
	Payload     []byte
	Metadata    map[string]interface{}
	Features    []float64
	Labels      []string
}

// DetectionResult represents the analysis outcome
type DetectionResult struct {
	EventID         string
	Timestamp       time.Time
	ThreatScore     float64
	Anomalous       bool
	Classifications []Classification
	Patterns        []Pattern
	Explanation     string
	Confidence      float64
}

// Classification represents a threat classification
type Classification struct {
	Category    string
	Probability float64
	Evidence    []string
}

// Pattern represents a detected pattern
type Pattern struct {
	ID         string
	Type       string
	Frequency  int
	Confidence float64
	Related    []string
}

// NewPatternDetector creates a new ML pattern detector
func NewPatternDetector(config DetectorConfig) (*PatternDetector, error) {
	detector := &PatternDetector{
		config:           config,
		eventBuffer:      NewEventBuffer(config.BatchSize),
		metrics:          NewDetectorMetrics(),
		featureExtractor: NewFeatureExtractor(config.MaxFeatures),
	}

	// Initialize models based on configuration
	if config.ModelType == "supervised" || config.ModelType == "hybrid" {
		detector.supervisedModel = NewRandomForestClassifier()
	}

	if config.ModelType == "unsupervised" || config.ModelType == "hybrid" {
		detector.unsupervisedModel = NewIsolationForest(config.AnomalyThreshold)
	}

	// Initialize LLM if enabled
	if config.EnableLLM {
		llm, err := NewLLMAccelerator(config.LLMProvider, config.LLMModel)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize LLM: %w", err)
		}
		detector.llmAccelerator = llm
	}

	return detector, nil
}

// Analyze processes a single event
func (d *PatternDetector) Analyze(ctx context.Context, event *SecurityEvent) (*DetectionResult, error) {
	// Extract features if not already present
	if len(event.Features) == 0 {
		features, err := d.featureExtractor.Extract(event)
		if err != nil {
			return nil, fmt.Errorf("feature extraction failed: %w", err)
		}
		event.Features = features
	}

	result := &DetectionResult{
		EventID:   event.ID,
		Timestamp: time.Now(),
	}

	// Run supervised classification if available
	if d.supervisedModel != nil {
		classifications, err := d.supervisedModel.Classify(event.Features)
		if err == nil {
			result.Classifications = classifications
			result.ThreatScore = d.calculateThreatScore(classifications)
		}
	}

	// Run anomaly detection if available
	if d.unsupervisedModel != nil {
		anomalyScore, err := d.unsupervisedModel.DetectAnomaly(event.Features)
		if err == nil {
			result.Anomalous = anomalyScore > d.config.AnomalyThreshold
			if result.Anomalous {
				result.ThreatScore = math.Max(result.ThreatScore, anomalyScore)
			}
		}
	}

	// Detect patterns in event stream
	patterns := d.detectPatterns(event)
	result.Patterns = patterns

	// Use LLM for enhanced analysis if enabled
	if d.llmAccelerator != nil && (result.ThreatScore > 0.7 || result.Anomalous) {
		explanation, confidence, err := d.llmAccelerator.AnalyzeEvent(ctx, event, result)
		if err == nil {
			result.Explanation = explanation
			result.Confidence = confidence
		}
	}

	// Update metrics
	d.metrics.RecordAnalysis(result)

	return result, nil
}

// AnalyzeBatch processes multiple events efficiently
func (d *PatternDetector) AnalyzeBatch(ctx context.Context, events []*SecurityEvent) ([]*DetectionResult, error) {
	// Extract features for all events
	featureMatrix := make([][]float64, len(events))
	for i, event := range events {
		if len(event.Features) == 0 {
			features, err := d.featureExtractor.Extract(event)
			if err != nil {
				continue
			}
			event.Features = features
		}
		featureMatrix[i] = event.Features
	}

	results := make([]*DetectionResult, len(events))

	// Batch process with supervised model
	if d.supervisedModel != nil {
		batchClassifications, err := d.supervisedModel.ClassifyBatch(featureMatrix)
		if err == nil {
			for i, classifications := range batchClassifications {
				results[i] = &DetectionResult{
					EventID:         events[i].ID,
					Timestamp:       time.Now(),
					Classifications: classifications,
					ThreatScore:     d.calculateThreatScore(classifications),
				}
			}
		}
	}

	// Batch anomaly detection
	if d.unsupervisedModel != nil {
		anomalyScores, err := d.unsupervisedModel.DetectAnomalyBatch(featureMatrix)
		if err == nil {
			for i, score := range anomalyScores {
				if results[i] == nil {
					results[i] = &DetectionResult{
						EventID:   events[i].ID,
						Timestamp: time.Now(),
					}
				}
				results[i].Anomalous = score > d.config.AnomalyThreshold
				if results[i].Anomalous {
					results[i].ThreatScore = math.Max(results[i].ThreatScore, score)
				}
			}
		}
	}

	// Pattern detection across batch
	d.detectBatchPatterns(events, results)

	// LLM batch analysis for high-risk events
	if d.llmAccelerator != nil {
		highRiskIndices := []int{}
		for i, result := range results {
			if result != nil && (result.ThreatScore > 0.7 || result.Anomalous) {
				highRiskIndices = append(highRiskIndices, i)
			}
		}

		if len(highRiskIndices) > 0 {
			highRiskEvents := make([]*SecurityEvent, len(highRiskIndices))
			highRiskResults := make([]*DetectionResult, len(highRiskIndices))
			for i, idx := range highRiskIndices {
				highRiskEvents[i] = events[idx]
				highRiskResults[i] = results[idx]
			}

			explanations, err := d.llmAccelerator.AnalyzeBatch(ctx, highRiskEvents, highRiskResults)
			if err == nil {
				for i, idx := range highRiskIndices {
					results[idx].Explanation = explanations[i].Explanation
					results[idx].Confidence = explanations[i].Confidence
				}
			}
		}
	}

	// Update metrics
	for _, result := range results {
		if result != nil {
			d.metrics.RecordAnalysis(result)
		}
	}

	return results, nil
}

// Train updates the models with new labeled data
func (d *PatternDetector) Train(ctx context.Context, events []*SecurityEvent, labels [][]string) error {
	if d.supervisedModel == nil {
		return fmt.Errorf("supervised model not initialized")
	}

	// Extract features
	featureMatrix := make([][]float64, len(events))
	for i, event := range events {
		features, err := d.featureExtractor.Extract(event)
		if err != nil {
			return fmt.Errorf("feature extraction failed for event %s: %w", event.ID, err)
		}
		featureMatrix[i] = features
	}

	// Train supervised model
	if err := d.supervisedModel.Train(featureMatrix, labels); err != nil {
		return fmt.Errorf("supervised model training failed: %w", err)
	}

	// Update unsupervised model if in hybrid mode
	if d.config.ModelType == "hybrid" && d.unsupervisedModel != nil {
		if err := d.unsupervisedModel.Update(featureMatrix); err != nil {
			return fmt.Errorf("unsupervised model update failed: %w", err)
		}
	}

	// Fine-tune LLM if supported
	if d.llmAccelerator != nil && d.llmAccelerator.SupportsFineTuning() {
		if err := d.llmAccelerator.FineTune(ctx, events, labels); err != nil {
			return fmt.Errorf("LLM fine-tuning failed: %w", err)
		}
	}

	d.metrics.RecordTraining(len(events))

	return nil
}

// detectPatterns identifies patterns in the event stream
func (d *PatternDetector) detectPatterns(event *SecurityEvent) []Pattern {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Add event to buffer
	d.eventBuffer.Add(event)

	// Get recent events for pattern analysis
	recentEvents := d.eventBuffer.GetRecent(d.config.WindowSize)

	patterns := []Pattern{}

	// Temporal patterns
	if temporal := d.detectTemporalPatterns(recentEvents); temporal != nil {
		patterns = append(patterns, temporal...)
	}

	// Behavioral patterns
	if behavioral := d.detectBehavioralPatterns(recentEvents); behavioral != nil {
		patterns = append(patterns, behavioral...)
	}

	// Sequence patterns
	if sequence := d.detectSequencePatterns(recentEvents); sequence != nil {
		patterns = append(patterns, sequence...)
	}

	return patterns
}

// detectBatchPatterns identifies patterns across a batch of events
func (d *PatternDetector) detectBatchPatterns(events []*SecurityEvent, results []*DetectionResult) {
	// Group events by source
	sourceGroups := make(map[string][]*SecurityEvent)
	for _, event := range events {
		sourceGroups[event.Source] = append(sourceGroups[event.Source], event)
	}

	// Detect coordinated patterns
	for source, group := range sourceGroups {
		if len(group) > 5 {
			// Potential scanning or brute force
			pattern := Pattern{
				ID:         fmt.Sprintf("coord_%s_%d", source, time.Now().Unix()),
				Type:       "coordinated_activity",
				Frequency:  len(group),
				Confidence: float64(len(group)) / float64(len(events)),
			}

			// Add pattern to relevant results
			for i, event := range events {
				if event.Source == source && results[i] != nil {
					results[i].Patterns = append(results[i].Patterns, pattern)
				}
			}
		}
	}
}

// detectTemporalPatterns identifies time-based patterns
func (d *PatternDetector) detectTemporalPatterns(events []*SecurityEvent) []Pattern {
	if len(events) < 3 {
		return nil
	}

	// Sort by timestamp
	sorted := make([]*SecurityEvent, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	patterns := []Pattern{}

	// Check for burst patterns
	bursts := d.detectBursts(sorted)
	for _, burst := range bursts {
		patterns = append(patterns, Pattern{
			ID:         fmt.Sprintf("burst_%d", burst.Start.Unix()),
			Type:       "temporal_burst",
			Frequency:  burst.Count,
			Confidence: burst.Intensity,
		})
	}

	// Check for periodic patterns
	if periodic := d.detectPeriodic(sorted); periodic != nil {
		patterns = append(patterns, *periodic)
	}

	return patterns
}

// detectBehavioralPatterns identifies behavioral anomalies
func (d *PatternDetector) detectBehavioralPatterns(events []*SecurityEvent) []Pattern {
	// Group by behavior type
	behaviors := make(map[string][]*SecurityEvent)
	for _, event := range events {
		key := fmt.Sprintf("%s_%s_%s", event.Type, event.Protocol, event.Destination)
		behaviors[key] = append(behaviors[key], event)
	}

	patterns := []Pattern{}

	// Look for unusual behaviors
	for behavior, group := range behaviors {
		if len(group) > len(events)/4 {
			// Dominant behavior pattern
			patterns = append(patterns, Pattern{
				ID:         fmt.Sprintf("behavior_%s_%d", behavior, time.Now().Unix()),
				Type:       "dominant_behavior",
				Frequency:  len(group),
				Confidence: float64(len(group)) / float64(len(events)),
			})
		}
	}

	return patterns
}

// detectSequencePatterns identifies sequential patterns
func (d *PatternDetector) detectSequencePatterns(events []*SecurityEvent) []Pattern {
	if len(events) < 3 {
		return nil
	}

	// Build sequence representation
	sequences := make([]string, len(events))
	for i, event := range events {
		sequences[i] = fmt.Sprintf("%s:%s", event.Type, event.Protocol)
	}

	// Find repeated subsequences
	patterns := []Pattern{}
	subsequences := d.findRepeatedSubsequences(sequences, 3)

	for seq, count := range subsequences {
		if count > 2 {
			patterns = append(patterns, Pattern{
				ID:         fmt.Sprintf("seq_%s_%d", seq, time.Now().Unix()),
				Type:       "sequence_pattern",
				Frequency:  count,
				Confidence: float64(count) / float64(len(events)),
			})
		}
	}

	return patterns
}

// calculateThreatScore computes overall threat score from classifications
func (d *PatternDetector) calculateThreatScore(classifications []Classification) float64 {
	if len(classifications) == 0 {
		return 0.0
	}

	maxScore := 0.0
	weightedSum := 0.0
	totalWeight := 0.0

	threatWeights := map[string]float64{
		"malware":      1.0,
		"intrusion":    0.9,
		"ddos":         0.8,
		"scanning":     0.7,
		"bruteforce":   0.7,
		"exfiltration": 0.9,
		"cryptomining": 0.6,
		"phishing":     0.8,
		"c2":           0.95,
	}

	for _, class := range classifications {
		weight, ok := threatWeights[class.Category]
		if !ok {
			weight = 0.5 // Default weight for unknown categories
		}

		score := class.Probability * weight
		maxScore = math.Max(maxScore, score)
		weightedSum += score
		totalWeight += weight
	}

	// Combine max and average
	avgScore := 0.0
	if totalWeight > 0 {
		avgScore = weightedSum / totalWeight
	}

	return 0.7*maxScore + 0.3*avgScore
}

// Burst represents a burst of activity
type Burst struct {
	Start     time.Time
	End       time.Time
	Count     int
	Intensity float64
}

// detectBursts identifies burst patterns in events
func (d *PatternDetector) detectBursts(events []*SecurityEvent) []Burst {
	if len(events) < 3 {
		return nil
	}

	bursts := []Burst{}
	threshold := 3.0 // Events per second threshold

	i := 0
	for i < len(events)-1 {
		j := i + 1
		burstCount := 1

		// Look for rapid succession of events
		for j < len(events) {
			gap := events[j].Timestamp.Sub(events[j-1].Timestamp).Seconds()
			rate := 1.0 / gap

			if rate > threshold {
				burstCount++
				j++
			} else {
				break
			}
		}

		if burstCount >= 3 {
			duration := events[j-1].Timestamp.Sub(events[i].Timestamp).Seconds()
			intensity := float64(burstCount) / math.Max(duration, 1.0)

			bursts = append(bursts, Burst{
				Start:     events[i].Timestamp,
				End:       events[j-1].Timestamp,
				Count:     burstCount,
				Intensity: intensity,
			})

			i = j
		} else {
			i++
		}
	}

	return bursts
}

// detectPeriodic identifies periodic patterns
func (d *PatternDetector) detectPeriodic(events []*SecurityEvent) *Pattern {
	if len(events) < 5 {
		return nil
	}

	// Calculate inter-arrival times
	intervals := make([]float64, len(events)-1)
	for i := 1; i < len(events); i++ {
		intervals[i-1] = events[i].Timestamp.Sub(events[i-1].Timestamp).Seconds()
	}

	// Check for regularity
	mean := 0.0
	for _, interval := range intervals {
		mean += interval
	}
	mean /= float64(len(intervals))

	variance := 0.0
	for _, interval := range intervals {
		variance += math.Pow(interval-mean, 2)
	}
	variance /= float64(len(intervals))

	cv := math.Sqrt(variance) / mean // Coefficient of variation

	if cv < 0.3 { // Low variation indicates periodic pattern
		return &Pattern{
			ID:         fmt.Sprintf("periodic_%d", time.Now().Unix()),
			Type:       "periodic_pattern",
			Frequency:  len(events),
			Confidence: 1.0 - cv,
			Related:    []string{fmt.Sprintf("period: %.2fs", mean)},
		}
	}

	return nil
}

// findRepeatedSubsequences finds repeated patterns in sequences
func (d *PatternDetector) findRepeatedSubsequences(sequences []string, minLength int) map[string]int {
	counts := make(map[string]int)

	for length := minLength; length <= len(sequences)/2; length++ {
		for i := 0; i <= len(sequences)-length; i++ {
			subseq := ""
			for j := 0; j < length; j++ {
				if j > 0 {
					subseq += ","
				}
				subseq += sequences[i+j]
			}
			counts[subseq]++
		}
	}

	// Filter out non-repeated sequences
	repeated := make(map[string]int)
	for seq, count := range counts {
		if count > 1 {
			repeated[seq] = count
		}
	}

	return repeated
}

// GetMetrics returns current detector metrics
func (d *PatternDetector) GetMetrics() DetectorMetricsSnapshot {
	return d.metrics.Snapshot()
}

// Close cleanly shuts down the pattern detector
func (d *PatternDetector) Close() error {
	if d.llmAccelerator != nil {
		return d.llmAccelerator.Close()
	}
	return nil
}

// EventBuffer manages a sliding window of events
type EventBuffer struct {
	events   []*SecurityEvent
	capacity int
	mu       sync.RWMutex
}

// NewEventBuffer creates a new event buffer
func NewEventBuffer(capacity int) *EventBuffer {
	return &EventBuffer{
		events:   make([]*SecurityEvent, 0, capacity),
		capacity: capacity,
	}
}

// Add adds an event to the buffer
func (b *EventBuffer) Add(event *SecurityEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.events = append(b.events, event)
	if len(b.events) > b.capacity {
		b.events = b.events[1:]
	}
}

// GetRecent returns events within the specified time window
func (b *EventBuffer) GetRecent(window time.Duration) []*SecurityEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	recent := []*SecurityEvent{}

	for _, event := range b.events {
		if event.Timestamp.After(cutoff) {
			recent = append(recent, event)
		}
	}

	return recent
}

// DetectorMetrics tracks pattern detector performance
type DetectorMetrics struct {
	totalAnalyzed  int64
	threats        int64
	anomalies      int64
	patterns       int64
	trainingEvents int64
	lastTraining   time.Time
	processingTime time.Duration
	mu             sync.RWMutex
}

// NewDetectorMetrics creates new metrics tracker
func NewDetectorMetrics() *DetectorMetrics {
	return &DetectorMetrics{}
}

// RecordAnalysis records analysis metrics
func (m *DetectorMetrics) RecordAnalysis(result *DetectionResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalAnalyzed++
	if result.ThreatScore > 0.7 {
		m.threats++
	}
	if result.Anomalous {
		m.anomalies++
	}
	m.patterns += int64(len(result.Patterns))
}

// RecordTraining records training metrics
func (m *DetectorMetrics) RecordTraining(eventCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.trainingEvents += int64(eventCount)
	m.lastTraining = time.Now()
}

// Snapshot returns current metrics
func (m *DetectorMetrics) Snapshot() DetectorMetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return DetectorMetricsSnapshot{
		TotalAnalyzed:  m.totalAnalyzed,
		Threats:        m.threats,
		Anomalies:      m.anomalies,
		Patterns:       m.patterns,
		TrainingEvents: m.trainingEvents,
		LastTraining:   m.lastTraining,
		ThreatRate:     float64(m.threats) / float64(m.totalAnalyzed),
		AnomalyRate:    float64(m.anomalies) / float64(m.totalAnalyzed),
	}
}

// DetectorMetricsSnapshot represents a point-in-time metrics snapshot
type DetectorMetricsSnapshot struct {
	TotalAnalyzed  int64
	Threats        int64
	Anomalies      int64
	Patterns       int64
	TrainingEvents int64
	LastTraining   time.Time
	ThreatRate     float64
	AnomalyRate    float64
}

// MarshalJSON implements json.Marshaler
func (s DetectorMetricsSnapshot) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"total_analyzed":  s.TotalAnalyzed,
		"threats":         s.Threats,
		"anomalies":       s.Anomalies,
		"patterns":        s.Patterns,
		"training_events": s.TrainingEvents,
		"last_training":   s.LastTraining,
		"threat_rate":     s.ThreatRate,
		"anomaly_rate":    s.AnomalyRate,
	})
}
