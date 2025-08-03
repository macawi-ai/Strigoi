package filters

import (
	"context"
	"regexp"
	"sync"
	"time"
	
	"github.com/macawi-ai/strigoi/internal/stream"
)

// FilterResult contains the outcome of filter processing
type FilterResult struct {
	Matched    bool
	Confidence float64
	Action     string // pass, block, alert
	Metadata   map[string]interface{}
}

// BaseFilter provides common filter functionality
type BaseFilter struct {
	name     string
	priority stream.Priority
	enabled  bool
	stats    FilterStats
	mu       sync.RWMutex
}

// FilterStats tracks filter performance
type FilterStats struct {
	Processed   uint64
	Matched     uint64
	AvgLatency  time.Duration
	LastMatched time.Time
}

// GetName returns filter name
func (f *BaseFilter) GetName() string {
	return f.name
}

// GetPriority returns filter priority
func (f *BaseFilter) GetPriority() stream.Priority {
	return f.priority
}

// UpdateStats updates filter statistics
func (f *BaseFilter) UpdateStats(matched bool, latency time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.stats.Processed++
	if matched {
		f.stats.Matched++
		f.stats.LastMatched = time.Now()
	}
	
	// Update average latency with exponential decay
	if f.stats.AvgLatency == 0 {
		f.stats.AvgLatency = latency
	} else {
		f.stats.AvgLatency = (f.stats.AvgLatency*9 + latency) / 10
	}
}

// StatefulFilter extends BaseFilter with state management
type StatefulFilter struct {
	BaseFilter
	state    interface{}
	stateMu  sync.RWMutex
	governor FilterGovernor
}

// FilterGovernor provides self-regulation for filters
type FilterGovernor interface {
	// ShouldProcess determines if filter should process data
	ShouldProcess(ctx context.Context) bool
	
	// OnResult updates governor based on filter result
	OnResult(result FilterResult, latency time.Duration)
	
	// GetHealth returns filter health status
	GetHealth() HealthStatus
}

// HealthStatus represents filter health
type HealthStatus struct {
	Healthy       bool
	ErrorRate     float64
	Latency       time.Duration
	LastHealthy   time.Time
	RecoveryCount int
}

// PatternRegistry manages compiled patterns
type PatternRegistry struct {
	patterns map[string]*regexp.Regexp
	mu       sync.RWMutex
}

// NewPatternRegistry creates a new pattern registry
func NewPatternRegistry() *PatternRegistry {
	return &PatternRegistry{
		patterns: make(map[string]*regexp.Regexp),
	}
}

// Register compiles and stores a pattern
func (r *PatternRegistry) Register(name, pattern string) error {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	
	r.mu.Lock()
	r.patterns[name] = compiled
	r.mu.Unlock()
	
	return nil
}

// Get returns a compiled pattern
func (r *PatternRegistry) Get(name string) (*regexp.Regexp, bool) {
	r.mu.RLock()
	pattern, ok := r.patterns[name]
	r.mu.RUnlock()
	return pattern, ok
}

// LoadPatterns loads patterns from various sources
func (r *PatternRegistry) LoadPatterns(source string) error {
	// TODO: Implement pattern loading from files/URLs
	return nil
}