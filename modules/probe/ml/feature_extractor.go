package ml

import (
	"bytes"
	"math"
	"sort"
	"strings"
	"time"
)

// FeatureExtractor extracts ML features from security events
type FeatureExtractor struct {
	maxFeatures      int
	statisticalStats bool
	temporalStats    bool
	behavioralStats  bool
	windowSize       time.Duration
	history          *FeatureHistory
}

// NewFeatureExtractor creates a new feature extractor
func NewFeatureExtractor(maxFeatures int) *FeatureExtractor {
	return &FeatureExtractor{
		maxFeatures:      maxFeatures,
		statisticalStats: true,
		temporalStats:    true,
		behavioralStats:  true,
		windowSize:       5 * time.Minute,
		history:          NewFeatureHistory(1000),
	}
}

// Extract extracts features from a security event
func (fe *FeatureExtractor) Extract(event *SecurityEvent) ([]float64, error) {
	features := []float64{}

	// Basic features
	basicFeatures := fe.extractBasicFeatures(event)
	features = append(features, basicFeatures...)

	// Statistical features
	if fe.statisticalStats {
		statFeatures := fe.extractStatisticalFeatures(event)
		features = append(features, statFeatures...)
	}

	// Temporal features
	if fe.temporalStats {
		tempFeatures := fe.extractTemporalFeatures(event)
		features = append(features, tempFeatures...)
	}

	// Behavioral features
	if fe.behavioralStats {
		behavFeatures := fe.extractBehavioralFeatures(event)
		features = append(features, behavFeatures...)
	}

	// Payload features
	payloadFeatures := fe.extractPayloadFeatures(event)
	features = append(features, payloadFeatures...)

	// Normalize and limit features
	features = fe.normalizeFeatures(features)
	if len(features) > fe.maxFeatures {
		features = features[:fe.maxFeatures]
	}

	// Update history
	fe.history.Add(event, features)

	return features, nil
}

// extractBasicFeatures extracts basic event features
func (fe *FeatureExtractor) extractBasicFeatures(event *SecurityEvent) []float64 {
	features := []float64{}

	// Event type encoding (one-hot)
	eventTypes := []string{"network", "file", "process", "registry", "authentication"}
	for _, t := range eventTypes {
		if event.Type == t {
			features = append(features, 1.0)
		} else {
			features = append(features, 0.0)
		}
	}

	// Protocol encoding
	protocolMap := map[string]float64{
		"tcp":   1.0,
		"udp":   2.0,
		"icmp":  3.0,
		"http":  4.0,
		"https": 5.0,
		"dns":   6.0,
		"ssh":   7.0,
		"ftp":   8.0,
	}

	protocolValue := 0.0
	if val, ok := protocolMap[strings.ToLower(event.Protocol)]; ok {
		protocolValue = val
	}
	features = append(features, protocolValue)

	// Time-based features
	hour := float64(event.Timestamp.Hour())
	dayOfWeek := float64(event.Timestamp.Weekday())
	isWeekend := 0.0
	if dayOfWeek == 0 || dayOfWeek == 6 {
		isWeekend = 1.0
	}

	features = append(features, hour/24.0)     // Normalized hour
	features = append(features, dayOfWeek/7.0) // Normalized day
	features = append(features, isWeekend)     // Weekend indicator

	// Payload size
	payloadSize := float64(len(event.Payload))
	features = append(features, math.Log1p(payloadSize)) // Log-normalized size

	return features
}

// extractStatisticalFeatures extracts statistical features
func (fe *FeatureExtractor) extractStatisticalFeatures(event *SecurityEvent) []float64 {
	features := []float64{}

	// Get recent events from history
	recentEvents := fe.history.GetRecentEvents(fe.windowSize)

	// Event frequency
	eventCount := 0
	sourceCount := 0
	destCount := 0

	for _, e := range recentEvents {
		if e.Type == event.Type {
			eventCount++
		}
		if e.Source == event.Source {
			sourceCount++
		}
		if e.Destination == event.Destination {
			destCount++
		}
	}

	totalEvents := len(recentEvents) + 1
	features = append(features, float64(eventCount)/float64(totalEvents))
	features = append(features, float64(sourceCount)/float64(totalEvents))
	features = append(features, float64(destCount)/float64(totalEvents))

	// Entropy calculations
	sourceEntropy := fe.calculateEntropy(recentEvents, "source")
	destEntropy := fe.calculateEntropy(recentEvents, "destination")

	features = append(features, sourceEntropy)
	features = append(features, destEntropy)

	// Variance in payload sizes
	payloadSizes := []float64{}
	for _, e := range recentEvents {
		payloadSizes = append(payloadSizes, float64(len(e.Payload)))
	}

	mean, variance := fe.calculateMeanVariance(payloadSizes)
	features = append(features, math.Log1p(mean))
	features = append(features, math.Log1p(variance))

	return features
}

// extractTemporalFeatures extracts time-based patterns
func (fe *FeatureExtractor) extractTemporalFeatures(event *SecurityEvent) []float64 {
	features := []float64{}

	recentEvents := fe.history.GetRecentEvents(fe.windowSize)

	// Inter-arrival times
	if len(recentEvents) > 0 {
		lastEvent := recentEvents[len(recentEvents)-1]
		timeDiff := event.Timestamp.Sub(lastEvent.Timestamp).Seconds()
		features = append(features, math.Log1p(timeDiff))

		// Calculate rate features
		eventTimes := []time.Time{event.Timestamp}
		for _, e := range recentEvents {
			eventTimes = append(eventTimes, e.Timestamp)
		}

		// Events per minute
		duration := event.Timestamp.Sub(recentEvents[0].Timestamp).Minutes()
		if duration > 0 {
			rate := float64(len(recentEvents)) / duration
			features = append(features, rate)
		} else {
			features = append(features, 0.0)
		}

		// Burst detection
		burstScore := fe.calculateBurstScore(eventTimes)
		features = append(features, burstScore)

		// Periodicity score
		periodicityScore := fe.calculatePeriodicityScore(eventTimes)
		features = append(features, periodicityScore)
	} else {
		// No recent events - use defaults
		features = append(features, 0.0, 0.0, 0.0, 0.0)
	}

	return features
}

// extractBehavioralFeatures extracts behavioral patterns
func (fe *FeatureExtractor) extractBehavioralFeatures(event *SecurityEvent) []float64 {
	features := []float64{}

	recentEvents := fe.history.GetRecentEvents(fe.windowSize)

	// Communication patterns
	uniqueSources := make(map[string]bool)
	uniqueDestinations := make(map[string]bool)
	connectionPairs := make(map[string]bool)

	for _, e := range recentEvents {
		uniqueSources[e.Source] = true
		uniqueDestinations[e.Destination] = true
		connectionPairs[e.Source+"->"+e.Destination] = true
	}

	// Diversity metrics
	sourceDiv := float64(len(uniqueSources)) / float64(len(recentEvents)+1)
	destDiv := float64(len(uniqueDestinations)) / float64(len(recentEvents)+1)
	pairDiv := float64(len(connectionPairs)) / float64(len(recentEvents)+1)

	features = append(features, sourceDiv)
	features = append(features, destDiv)
	features = append(features, pairDiv)

	// Connection patterns
	inDegree := 0
	outDegree := 0

	for _, e := range recentEvents {
		if e.Destination == event.Source {
			inDegree++
		}
		if e.Source == event.Source {
			outDegree++
		}
	}

	features = append(features, float64(inDegree)/float64(len(recentEvents)+1))
	features = append(features, float64(outDegree)/float64(len(recentEvents)+1))

	// Protocol diversity
	protocols := make(map[string]int)
	for _, e := range recentEvents {
		protocols[e.Protocol]++
	}

	protocolEntropy := 0.0
	total := len(recentEvents) + 1
	for _, count := range protocols {
		prob := float64(count) / float64(total)
		if prob > 0 {
			protocolEntropy -= prob * math.Log2(prob)
		}
	}

	features = append(features, protocolEntropy)

	return features
}

// extractPayloadFeatures extracts features from payload content
func (fe *FeatureExtractor) extractPayloadFeatures(event *SecurityEvent) []float64 {
	features := []float64{}

	payload := event.Payload
	payloadLen := float64(len(payload))

	if payloadLen == 0 {
		// Return zero features for empty payload
		return make([]float64, 10)
	}

	// Basic statistics
	features = append(features, payloadLen)

	// Byte frequency analysis
	byteFreq := make([]int, 256)
	for _, b := range payload {
		byteFreq[b]++
	}

	// Entropy
	entropy := 0.0
	for _, count := range byteFreq {
		if count > 0 {
			prob := float64(count) / payloadLen
			entropy -= prob * math.Log2(prob)
		}
	}
	features = append(features, entropy)

	// Character classes
	printable := 0
	alphanumeric := 0
	whitespace := 0

	for _, b := range payload {
		if b >= 32 && b <= 126 {
			printable++
		}
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			alphanumeric++
		}
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			whitespace++
		}
	}

	features = append(features, float64(printable)/payloadLen)
	features = append(features, float64(alphanumeric)/payloadLen)
	features = append(features, float64(whitespace)/payloadLen)

	// N-gram analysis (bi-grams)
	if len(payload) > 1 {
		bigramMap := make(map[uint16]int)
		for i := 0; i < len(payload)-1; i++ {
			bigram := uint16(payload[i])<<8 | uint16(payload[i+1])
			bigramMap[bigram]++
		}

		// Bigram entropy
		bigramEntropy := 0.0
		totalBigrams := float64(len(payload) - 1)
		for _, count := range bigramMap {
			prob := float64(count) / totalBigrams
			bigramEntropy -= prob * math.Log2(prob)
		}
		features = append(features, bigramEntropy)

		// Unique bigram ratio
		uniqueBigramRatio := float64(len(bigramMap)) / totalBigrams
		features = append(features, uniqueBigramRatio)
	} else {
		features = append(features, 0.0, 0.0)
	}

	// Statistical moments
	byteMean := 0.0
	for i, count := range byteFreq {
		byteMean += float64(i) * float64(count)
	}
	byteMean /= payloadLen

	byteVariance := 0.0
	for i, count := range byteFreq {
		if count > 0 {
			diff := float64(i) - byteMean
			byteVariance += diff * diff * float64(count)
		}
	}
	byteVariance /= payloadLen

	features = append(features, byteMean/255.0)                // Normalized mean
	features = append(features, math.Sqrt(byteVariance)/255.0) // Normalized std dev

	// Pattern detection
	suspiciousPatterns := fe.detectSuspiciousPatterns(payload)
	features = append(features, float64(suspiciousPatterns))

	return features
}

// detectSuspiciousPatterns counts suspicious byte patterns
func (fe *FeatureExtractor) detectSuspiciousPatterns(payload []byte) int {
	count := 0

	// NOP sled detection
	nopCount := 0
	for _, b := range payload {
		if b == 0x90 { // x86 NOP
			nopCount++
			if nopCount > 10 {
				count++
				break
			}
		} else {
			nopCount = 0
		}
	}

	// Shell code patterns
	shellPatterns := [][]byte{
		{0x31, 0xc0}, // xor eax, eax
		{0x31, 0xdb}, // xor ebx, ebx
		{0x31, 0xc9}, // xor ecx, ecx
		{0xff, 0xe4}, // jmp esp
		{0xff, 0xd0}, // call eax
	}

	for _, pattern := range shellPatterns {
		if bytes.Contains(payload, pattern) {
			count++
		}
	}

	// Long sequences of same byte (possible encoding/encryption)
	sameByteCount := 1
	for i := 1; i < len(payload); i++ {
		if payload[i] == payload[i-1] {
			sameByteCount++
			if sameByteCount > 50 {
				count++
				break
			}
		} else {
			sameByteCount = 1
		}
	}

	return count
}

// calculateEntropy calculates entropy for a specific field
func (fe *FeatureExtractor) calculateEntropy(events []*SecurityEvent, field string) float64 {
	counts := make(map[string]int)
	total := len(events)

	for _, e := range events {
		var value string
		switch field {
		case "source":
			value = e.Source
		case "destination":
			value = e.Destination
		case "type":
			value = e.Type
		case "protocol":
			value = e.Protocol
		}
		counts[value]++
	}

	entropy := 0.0
	for _, count := range counts {
		prob := float64(count) / float64(total)
		if prob > 0 {
			entropy -= prob * math.Log2(prob)
		}
	}

	return entropy
}

// calculateMeanVariance calculates mean and variance
func (fe *FeatureExtractor) calculateMeanVariance(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return mean, variance
}

// calculateBurstScore calculates burst activity score
func (fe *FeatureExtractor) calculateBurstScore(times []time.Time) float64 {
	if len(times) < 3 {
		return 0.0
	}

	// Sort times
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})

	// Calculate inter-arrival times
	intervals := make([]float64, len(times)-1)
	for i := 1; i < len(times); i++ {
		intervals[i-1] = times[i].Sub(times[i-1]).Seconds()
	}

	// Calculate coefficient of variation
	mean, variance := fe.calculateMeanVariance(intervals)
	if mean == 0 {
		return 0.0
	}

	cv := math.Sqrt(variance) / mean

	// High CV indicates bursty behavior
	return math.Min(cv, 10.0) / 10.0
}

// calculatePeriodicityScore detects periodic patterns
func (fe *FeatureExtractor) calculatePeriodicityScore(times []time.Time) float64 {
	if len(times) < 5 {
		return 0.0
	}

	// Calculate inter-arrival times
	intervals := make([]float64, len(times)-1)
	for i := 1; i < len(times); i++ {
		intervals[i-1] = times[i].Sub(times[i-1]).Seconds()
	}

	// Simple autocorrelation at lag 1
	mean, _ := fe.calculateMeanVariance(intervals)

	autocorr := 0.0
	variance := 0.0

	for i := 0; i < len(intervals)-1; i++ {
		autocorr += (intervals[i] - mean) * (intervals[i+1] - mean)
		variance += (intervals[i] - mean) * (intervals[i] - mean)
	}

	if variance == 0 {
		return 0.0
	}

	// Normalize autocorrelation
	autocorr /= variance

	// High autocorrelation indicates periodicity
	return math.Abs(autocorr)
}

// normalizeFeatures normalizes feature values
func (fe *FeatureExtractor) normalizeFeatures(features []float64) []float64 {
	normalized := make([]float64, len(features))

	for i, value := range features {
		// Handle special values
		if math.IsNaN(value) || math.IsInf(value, 0) {
			normalized[i] = 0.0
			continue
		}

		// Clip extreme values
		if value > 100 {
			value = 100
		} else if value < -100 {
			value = -100
		}

		// Apply sigmoid normalization for bounded output
		normalized[i] = 1.0 / (1.0 + math.Exp(-value/10.0))
	}

	return normalized
}

// FeatureHistory maintains historical event data
type FeatureHistory struct {
	events   []*SecurityEvent
	features [][]float64
	capacity int
	position int
}

// NewFeatureHistory creates a new feature history buffer
func NewFeatureHistory(capacity int) *FeatureHistory {
	return &FeatureHistory{
		events:   make([]*SecurityEvent, capacity),
		features: make([][]float64, capacity),
		capacity: capacity,
		position: 0,
	}
}

// Add adds an event and its features to history
func (h *FeatureHistory) Add(event *SecurityEvent, features []float64) {
	h.events[h.position] = event
	h.features[h.position] = features
	h.position = (h.position + 1) % h.capacity
}

// GetRecentEvents returns events within time window
func (h *FeatureHistory) GetRecentEvents(window time.Duration) []*SecurityEvent {
	cutoff := time.Now().Add(-window)
	recent := []*SecurityEvent{}

	for _, event := range h.events {
		if event != nil && event.Timestamp.After(cutoff) {
			recent = append(recent, event)
		}
	}

	return recent
}
