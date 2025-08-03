package realtime

import (
	"sync"
	"time"
)

// ThreatHistory tracks historical threat data for correlation
type ThreatHistory struct {
	threats      []ThreatRecord
	responseTimes []float64
	mu           sync.RWMutex
	maxRecords   int
}

// ThreatRecord stores threat with response metrics
type ThreatRecord struct {
	Threat       ThreatEvent
	ResponseTime float64
	Timestamp    time.Time
}

// NewThreatHistory creates a new threat history tracker
func NewThreatHistory() *ThreatHistory {
	return &ThreatHistory{
		threats:       make([]ThreatRecord, 0),
		responseTimes: make([]float64, 0),
		maxRecords:    10000, // Keep last 10k threats
	}
}

// Record adds a threat to history
func (th *ThreatHistory) Record(threat ThreatEvent) {
	th.mu.Lock()
	defer th.mu.Unlock()
	
	record := ThreatRecord{
		Threat:    threat,
		Timestamp: time.Now(),
	}
	
	th.threats = append(th.threats, record)
	
	// Trim if too large
	if len(th.threats) > th.maxRecords {
		th.threats = th.threats[len(th.threats)-th.maxRecords:]
	}
}

// RecordResponse updates response time for a threat
func (th *ThreatHistory) RecordResponse(threatID string, responseTime float64) {
	th.mu.Lock()
	defer th.mu.Unlock()
	
	// Find and update the threat record
	for i := len(th.threats) - 1; i >= 0; i-- {
		if th.threats[i].Threat.ID == threatID {
			th.threats[i].ResponseTime = responseTime
			th.responseTimes = append(th.responseTimes, responseTime)
			
			// Keep only last 1000 response times
			if len(th.responseTimes) > 1000 {
				th.responseTimes = th.responseTimes[len(th.responseTimes)-1000:]
			}
			break
		}
	}
}

// Similar finds threats similar to the given one
func (th *ThreatHistory) Similar(threat ThreatEvent) []ThreatRecord {
	th.mu.RLock()
	defer th.mu.RUnlock()
	
	similar := []ThreatRecord{}
	for _, record := range th.threats {
		if record.Threat.Type == threat.Type {
			similar = append(similar, record)
		}
	}
	
	// Return last 10 similar threats
	if len(similar) > 10 {
		return similar[len(similar)-10:]
	}
	return similar
}

// GetContext returns historical context for a threat
func (th *ThreatHistory) GetContext(threat ThreatEvent) map[string]interface{} {
	th.mu.RLock()
	defer th.mu.RUnlock()
	
	// Count threats by type in last hour
	hourAgo := time.Now().Add(-1 * time.Hour)
	typeCounts := make(map[ThreatType]int)
	recentThreats := []ThreatRecord{}
	
	for i := len(th.threats) - 1; i >= 0; i-- {
		if th.threats[i].Timestamp.Before(hourAgo) {
			break
		}
		typeCounts[th.threats[i].Threat.Type]++
		if th.threats[i].Threat.Type == threat.Type {
			recentThreats = append(recentThreats, th.threats[i])
		}
	}
	
	return map[string]interface{}{
		"threat_counts_1h": typeCounts,
		"similar_recent":   recentThreats,
		"total_today":      th.CountToday(),
	}
}

// CountToday returns number of threats detected today
func (th *ThreatHistory) CountToday() int {
	th.mu.RLock()
	defer th.mu.RUnlock()
	
	today := time.Now().Truncate(24 * time.Hour)
	count := 0
	
	for i := len(th.threats) - 1; i >= 0; i-- {
		if th.threats[i].Timestamp.Before(today) {
			break
		}
		count++
	}
	
	return count
}

// AverageResponseTime calculates average response time
func (th *ThreatHistory) AverageResponseTime() float64 {
	th.mu.RLock()
	defer th.mu.RUnlock()
	
	if len(th.responseTimes) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, rt := range th.responseTimes {
		sum += rt
	}
	
	return sum / float64(len(th.responseTimes))
}