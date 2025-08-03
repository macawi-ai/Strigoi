package stream

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// Governor provides self-regulating behavior for stream components
type Governor interface {
	// Assess current health and performance
	Assess() HealthStatus
	
	// Regulate adjusts behavior based on conditions
	Regulate() error
	
	// Learn from past behavior
	Learn(feedback Feedback) error
	
	// GetMetrics returns governor metrics
	GetMetrics() GovernorMetrics
}

// HealthStatus represents component health
type HealthStatus struct {
	Score       float64 // 0.0 to 1.0
	Status      string  // healthy, degraded, critical
	LastChecked time.Time
	Issues      []string
}

// Feedback provides learning input
type Feedback struct {
	Type      string // positive, negative, neutral
	Metric    string
	Value     float64
	Timestamp time.Time
}

// GovernorMetrics tracks governor performance
type GovernorMetrics struct {
	Regulations   uint64
	LearningCycles uint64
	HealthScore    float64
	LastRegulation time.Time
}

// AdaptiveGovernor regulates filter behavior
type AdaptiveGovernor struct {
	name          string
	component     interface{}
	metrics       GovernorMetrics
	healthHistory []HealthStatus
	mu            sync.RWMutex
	
	// Thresholds
	healthThreshold   float64
	latencyThreshold  time.Duration
	errorRateLimit   float64
	
	// Learning parameters
	learningRate      float64
	adaptationFactor  float64
}

// NewAdaptiveGovernor creates a governor for filter adaptation
func NewAdaptiveGovernor(name string, component interface{}) *AdaptiveGovernor {
	return &AdaptiveGovernor{
		name:              name,
		component:         component,
		healthHistory:     make([]HealthStatus, 0, 100),
		healthThreshold:   0.7,
		latencyThreshold:  100 * time.Microsecond,
		errorRateLimit:   0.05, // 5% error rate max
		learningRate:      0.1,
		adaptationFactor:  1.0,
	}
}

// Assess evaluates current component health
func (ag *AdaptiveGovernor) Assess() HealthStatus {
	ag.mu.RLock()
	defer ag.mu.RUnlock()
	
	status := HealthStatus{
		LastChecked: time.Now(),
		Issues:      make([]string, 0),
	}
	
	// Check component-specific health
	switch v := ag.component.(type) {
	case *BaseFilter:
		stats := v.stats
		
		// Check error rate
		if stats.Processed > 0 {
			errorRate := float64(stats.Matched) / float64(stats.Processed)
			if errorRate > ag.errorRateLimit {
				status.Issues = append(status.Issues, "High error rate")
			}
		}
		
		// Check latency
		if stats.AvgLatency > ag.latencyThreshold {
			status.Issues = append(status.Issues, "High latency")
		}
		
		// Calculate health score
		status.Score = ag.calculateHealthScore(stats)
	}
	
	// Determine status
	if status.Score >= 0.8 {
		status.Status = "healthy"
	} else if status.Score >= 0.5 {
		status.Status = "degraded"
	} else {
		status.Status = "critical"
	}
	
	// Store in history
	ag.mu.RUnlock()
	ag.mu.Lock()
	ag.healthHistory = append(ag.healthHistory, status)
	if len(ag.healthHistory) > 100 {
		ag.healthHistory = ag.healthHistory[1:]
	}
	ag.mu.Unlock()
	ag.mu.RLock()
	
	return status
}

// Regulate adjusts component behavior
func (ag *AdaptiveGovernor) Regulate() error {
	status := ag.Assess()
	
	ag.mu.Lock()
	defer ag.mu.Unlock()
	
	ag.metrics.Regulations++
	ag.metrics.LastRegulation = time.Now()
	ag.metrics.HealthScore = status.Score
	
	// Apply regulations based on health
	if status.Score < ag.healthThreshold {
		// Reduce load or adjust parameters
		ag.adaptationFactor *= 0.9 // Slow down
	} else if status.Score > 0.9 {
		// Can handle more load
		ag.adaptationFactor *= 1.1 // Speed up
	}
	
	// Keep adaptation factor in reasonable bounds
	ag.adaptationFactor = math.Max(0.5, math.Min(2.0, ag.adaptationFactor))
	
	return nil
}

// Learn adjusts governor parameters based on feedback
func (ag *AdaptiveGovernor) Learn(feedback Feedback) error {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	
	ag.metrics.LearningCycles++
	
	// Simple learning algorithm
	switch feedback.Type {
	case "positive":
		// Reinforce current behavior
		ag.learningRate *= 1.05
	case "negative":
		// Adjust behavior
		ag.learningRate *= 0.95
		if feedback.Metric == "latency" {
			ag.latencyThreshold = time.Duration(float64(ag.latencyThreshold) * 1.1) // Be more tolerant
		}
	}
	
	return nil
}

// GetMetrics returns governor performance metrics
func (ag *AdaptiveGovernor) GetMetrics() GovernorMetrics {
	ag.mu.RLock()
	defer ag.mu.RUnlock()
	return ag.metrics
}

// calculateHealthScore computes health from filter stats
func (ag *AdaptiveGovernor) calculateHealthScore(stats FilterStats) float64 {
	score := 1.0
	
	// Penalize high latency
	if stats.AvgLatency > ag.latencyThreshold {
		latencyRatio := float64(ag.latencyThreshold) / float64(stats.AvgLatency)
		score *= latencyRatio
	}
	
	// Reward processing efficiency
	if stats.Processed > 0 {
		efficiency := 1.0 - (float64(stats.Matched) / float64(stats.Processed))
		score *= efficiency
	}
	
	// Consider recency
	if time.Since(stats.LastMatch) > time.Minute {
		score *= 0.9 // Slight penalty for inactivity
	}
	
	return math.Max(0.0, math.Min(1.0, score))
}

// CircuitBreaker prevents cascade failures
type CircuitBreaker struct {
	name            string
	failureThreshold int
	recoveryTimeout  time.Duration
	failureCount     int
	lastFailure      time.Time
	state            string // closed, open, half-open
	mu               sync.RWMutex
}

// NewCircuitBreaker creates a circuit breaker governor
func NewCircuitBreaker(name string, threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		failureThreshold: threshold,
		recoveryTimeout:  timeout,
		state:            "closed",
	}
}

// Call executes function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	// Check state
	switch cb.state {
	case "open":
		// Check if we should try half-open
		if time.Since(cb.lastFailure) > cb.recoveryTimeout {
			cb.state = "half-open"
			cb.failureCount = 0
		} else {
			return ErrCircuitOpen
		}
	}
	
	// Try the call
	err := fn()
	if err != nil {
		cb.failureCount++
		cb.lastFailure = time.Now()
		
		if cb.failureCount >= cb.failureThreshold {
			cb.state = "open"
		}
		return err
	}
	
	// Success
	if cb.state == "half-open" {
		cb.state = "closed"
	}
	cb.failureCount = 0
	
	return nil
}

// GetState returns current circuit breaker state
func (cb *CircuitBreaker) GetState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// ExponentialSmoothing tracks baseline for anomaly detection
type ExponentialSmoothing struct {
	alpha    float64 // Smoothing factor
	baseline float64
	variance float64
	count    uint64
	mu       sync.RWMutex
}

// NewExponentialSmoothing creates baseline tracker
func NewExponentialSmoothing(alpha float64) *ExponentialSmoothing {
	return &ExponentialSmoothing{
		alpha: alpha,
	}
}

// Update adds new observation
func (es *ExponentialSmoothing) Update(value float64) {
	es.mu.Lock()
	defer es.mu.Unlock()
	
	if es.count == 0 {
		es.baseline = value
		es.variance = 0
	} else {
		// Update baseline
		oldBaseline := es.baseline
		es.baseline = es.alpha*value + (1-es.alpha)*es.baseline
		
		// Update variance estimate
		diff := value - oldBaseline
		es.variance = es.alpha*diff*diff + (1-es.alpha)*es.variance
	}
	
	es.count++
}

// IsAnomaly checks if value is anomalous
func (es *ExponentialSmoothing) IsAnomaly(value float64, threshold float64) bool {
	es.mu.RLock()
	defer es.mu.RUnlock()
	
	if es.count < 10 {
		return false // Not enough data
	}
	
	stdDev := math.Sqrt(es.variance)
	deviation := math.Abs(value - es.baseline)
	
	return deviation > threshold*stdDev
}

// GetBaseline returns current baseline
func (es *ExponentialSmoothing) GetBaseline() float64 {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.baseline
}

// Custom errors
var (
	ErrCircuitOpen = fmt.Errorf("circuit breaker is open")
)