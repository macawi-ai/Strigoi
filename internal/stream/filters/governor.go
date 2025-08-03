package filters

import (
	"context"
	"sync"
	"time"
)

// AdaptiveGovernor provides self-regulation for filters
type AdaptiveGovernor struct {
	name            string
	healthThreshold float64
	errorWindow     time.Duration
	recoveryDelay   time.Duration
	
	// State tracking
	errors        []time.Time
	lastHealthy   time.Time
	recoveryCount int
	currentHealth float64
	mu            sync.RWMutex
	
	// Learning parameters
	adaptiveRate  float64
	baseline      *ExponentialSmoothing
}

// NewAdaptiveGovernor creates a new adaptive filter governor
func NewAdaptiveGovernor(name string) *AdaptiveGovernor {
	return &AdaptiveGovernor{
		name:            name,
		healthThreshold: 0.95, // 95% health required
		errorWindow:     time.Minute,
		recoveryDelay:   time.Second * 5,
		errors:          make([]time.Time, 0),
		lastHealthy:     time.Now(),
		currentHealth:   1.0,
		adaptiveRate:    0.1,
		baseline:        NewExponentialSmoothing(0.1),
	}
}

// ShouldProcess determines if filter should process data
func (g *AdaptiveGovernor) ShouldProcess(ctx context.Context) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	// Check context cancellation
	select {
	case <-ctx.Done():
		return false
	default:
	}
	
	// If unhealthy and in recovery period, skip
	if g.currentHealth < g.healthThreshold {
		if time.Since(g.lastHealthy) < g.recoveryDelay {
			return false
		}
	}
	
	return true
}

// OnResult updates governor based on filter result
func (g *AdaptiveGovernor) OnResult(result FilterResult, latency time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	// Update baseline latency
	g.baseline.Update(float64(latency.Nanoseconds()))
	
	// Check if latency is anomalous
	if g.isAnomalous(latency) {
		g.recordError()
	} else {
		g.recordSuccess()
	}
	
	// Update health score
	g.updateHealth()
}

// GetHealth returns current health status
func (g *AdaptiveGovernor) GetHealth() HealthStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	return HealthStatus{
		Healthy:       g.currentHealth >= g.healthThreshold,
		ErrorRate:     g.calculateErrorRate(),
		Latency:       time.Duration(g.baseline.Value()),
		LastHealthy:   g.lastHealthy,
		RecoveryCount: g.recoveryCount,
	}
}

// isAnomalous checks if latency is outside normal bounds
func (g *AdaptiveGovernor) isAnomalous(latency time.Duration) bool {
	baseline := g.baseline.Value()
	threshold := baseline * 3 // 3x baseline is anomalous
	
	return float64(latency.Nanoseconds()) > threshold
}

// recordError adds an error timestamp
func (g *AdaptiveGovernor) recordError() {
	now := time.Now()
	g.errors = append(g.errors, now)
	
	// Clean old errors outside window
	cutoff := now.Add(-g.errorWindow)
	newErrors := make([]time.Time, 0)
	for _, t := range g.errors {
		if t.After(cutoff) {
			newErrors = append(newErrors, t)
		}
	}
	g.errors = newErrors
}

// recordSuccess updates last healthy time
func (g *AdaptiveGovernor) recordSuccess() {
	g.lastHealthy = time.Now()
}

// updateHealth recalculates health score
func (g *AdaptiveGovernor) updateHealth() {
	errorRate := g.calculateErrorRate()
	
	// Exponential decay for health score
	targetHealth := 1.0 - errorRate
	g.currentHealth = g.currentHealth*(1-g.adaptiveRate) + targetHealth*g.adaptiveRate
	
	// Track recoveries
	if g.currentHealth >= g.healthThreshold && errorRate < 0.05 {
		if time.Since(g.lastHealthy) > g.recoveryDelay {
			g.recoveryCount++
		}
	}
}

// calculateErrorRate returns error rate in window
func (g *AdaptiveGovernor) calculateErrorRate() float64 {
	if len(g.errors) == 0 {
		return 0.0
	}
	
	// Estimate request rate (simplified)
	windowSeconds := g.errorWindow.Seconds()
	estimatedRequests := windowSeconds * 100 // Assume 100 RPS baseline
	
	return float64(len(g.errors)) / estimatedRequests
}

// ExponentialSmoothing provides adaptive baseline tracking
type ExponentialSmoothing struct {
	alpha float64
	value float64
	init  bool
}

// NewExponentialSmoothing creates a new smoother
func NewExponentialSmoothing(alpha float64) *ExponentialSmoothing {
	return &ExponentialSmoothing{
		alpha: alpha,
		init:  false,
	}
}

// Update adds a new observation
func (s *ExponentialSmoothing) Update(observation float64) {
	if !s.init {
		s.value = observation
		s.init = true
		return
	}
	
	s.value = s.alpha*observation + (1-s.alpha)*s.value
}

// Value returns current smoothed value
func (s *ExponentialSmoothing) Value() float64 {
	return s.value
}

// CircuitBreaker provides fail-fast protection
type CircuitBreaker struct {
	name          string
	maxFailures   int
	resetTimeout  time.Duration
	halfOpenLimit int
	
	state        CircuitState
	failures     int
	lastFailTime time.Time
	successCount int
	mu           sync.RWMutex
}

// CircuitState represents circuit breaker states
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:          name,
		maxFailures:   maxFailures,
		resetTimeout:  resetTimeout,
		halfOpenLimit: maxFailures / 2,
		state:         StateClosed,
	}
}

// Call executes function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	switch cb.state {
	case StateOpen:
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
		} else {
			return ErrCircuitOpen
		}
		
	case StateHalfOpen:
		// Allow limited requests
		if cb.successCount >= cb.halfOpenLimit {
			cb.state = StateClosed
			cb.failures = 0
		}
	}
	
	// Execute function
	err := fn()
	
	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()
		
		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
		}
		return err
	}
	
	// Success
	if cb.state == StateHalfOpen {
		cb.successCount++
	}
	
	return nil
}

// GetState returns current circuit state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Custom errors
var (
	ErrCircuitOpen = &CircuitError{msg: "circuit breaker is open"}
)

// CircuitError represents a circuit breaker error
type CircuitError struct {
	msg string
}

func (e *CircuitError) Error() string {
	return e.msg
}