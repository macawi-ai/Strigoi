package filters

import (
	"context"
	"math"
	"regexp"
	"sync"
	"time"
	
	"golang.org/x/time/rate"
)

// RegexFilter performs pattern matching with pre-compiled regex
type RegexFilter struct {
	BaseFilter
	pattern  *regexp.Regexp
	category string
}

// NewRegexFilter creates a new regex filter
func NewRegexFilter(name, pattern, category string) (*RegexFilter, error) {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	
	return &RegexFilter{
		BaseFilter: BaseFilter{
			name:     name,
			priority: stream.PriorityHigh,
			enabled:  true,
		},
		pattern:  compiled,
		category: category,
	}, nil
}

// Match checks if data matches the pattern
func (f *RegexFilter) Match(data []byte) bool {
	start := time.Now()
	matched := f.pattern.Match(data)
	f.UpdateStats(matched, time.Since(start))
	return matched
}

// KeywordFilter performs fast string matching
type KeywordFilter struct {
	BaseFilter
	keywords map[string]bool
	minLen   int
}

// NewKeywordFilter creates a new keyword filter
func NewKeywordFilter(name string, keywords []string) *KeywordFilter {
	kw := make(map[string]bool)
	minLen := math.MaxInt
	
	for _, k := range keywords {
		kw[k] = true
		if len(k) < minLen {
			minLen = len(k)
		}
	}
	
	return &KeywordFilter{
		BaseFilter: BaseFilter{
			name:     name,
			priority: stream.PriorityMedium,
			enabled:  true,
		},
		keywords: kw,
		minLen:   minLen,
	}
}

// Match performs keyword matching
func (f *KeywordFilter) Match(data []byte) bool {
	if len(data) < f.minLen {
		return false
	}
	
	start := time.Now()
	dataStr := string(data)
	
	for keyword := range f.keywords {
		if contains(dataStr, keyword) {
			f.UpdateStats(true, time.Since(start))
			return true
		}
	}
	
	f.UpdateStats(false, time.Since(start))
	return false
}

// RateLimitFilter prevents flooding with per-source tracking
type RateLimitFilter struct {
	StatefulFilter
	limiters sync.Map // source -> *rate.Limiter
	rps      int      // requests per second
	burst    int
}

// NewRateLimitFilter creates a new rate limit filter
func NewRateLimitFilter(name string, rps, burst int) *RateLimitFilter {
	return &RateLimitFilter{
		StatefulFilter: StatefulFilter{
			BaseFilter: BaseFilter{
				name:     name,
				priority: stream.PriorityCritical,
				enabled:  true,
			},
		},
		rps:   rps,
		burst: burst,
	}
}

// Match checks rate limit for source
func (f *RateLimitFilter) Match(data []byte) bool {
	// Extract source from data (implementation depends on protocol)
	source := extractSource(data)
	
	limiterI, _ := f.limiters.LoadOrStore(source, rate.NewLimiter(rate.Limit(f.rps), f.burst))
	limiter := limiterI.(*rate.Limiter)
	
	allowed := limiter.Allow()
	f.UpdateStats(!allowed, 0) // Track blocks, not allows
	
	return !allowed // Return true if rate limit exceeded
}

// EntropyFilter detects high-entropy data (encryption/compression)
type EntropyFilter struct {
	BaseFilter
	threshold float64
}

// NewEntropyFilter creates a new entropy filter
func NewEntropyFilter(name string, threshold float64) *EntropyFilter {
	return &EntropyFilter{
		BaseFilter: BaseFilter{
			name:     name,
			priority: stream.PriorityLow,
			enabled:  true,
		},
		threshold: threshold,
	}
}

// Match calculates Shannon entropy
func (f *EntropyFilter) Match(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	
	start := time.Now()
	entropy := calculateEntropy(data)
	matched := entropy > f.threshold
	
	f.UpdateStats(matched, time.Since(start))
	return matched
}

// LengthFilter quickly rejects oversized data
type LengthFilter struct {
	BaseFilter
	maxLength int
}

// NewLengthFilter creates a new length filter
func NewLengthFilter(name string, maxLength int) *LengthFilter {
	return &LengthFilter{
		BaseFilter: BaseFilter{
			name:     name,
			priority: stream.PriorityCritical, // Run first
			enabled:  true,
		},
		maxLength: maxLength,
	}
}

// Match checks data length
func (f *LengthFilter) Match(data []byte) bool {
	matched := len(data) > f.maxLength
	f.UpdateStats(matched, 0) // Negligible latency
	return matched
}

// S1EdgeProcessor combines all edge filters
type S1EdgeProcessor struct {
	filters    []stream.Filter
	registry   *PatternRegistry
	mu         sync.RWMutex
}

// NewS1EdgeProcessor creates the S1 edge processing stage
func NewS1EdgeProcessor() (*S1EdgeProcessor, error) {
	processor := &S1EdgeProcessor{
		filters:  make([]stream.Filter, 0),
		registry: NewPatternRegistry(),
	}
	
	// Bootstrap critical patterns
	if err := processor.bootstrapPatterns(); err != nil {
		return nil, err
	}
	
	// Initialize default filters
	if err := processor.initializeFilters(); err != nil {
		return nil, err
	}
	
	return processor, nil
}

// bootstrapPatterns loads all critical patterns
func (p *S1EdgeProcessor) bootstrapPatterns() error {
	for _, pattern := range CriticalPatterns {
		if err := p.registry.Register(pattern.Name, pattern.Regex); err != nil {
			return err
		}
	}
	return nil
}

// initializeFilters sets up default S1 filters
func (p *S1EdgeProcessor) initializeFilters() error {
	// Length filter first (fastest)
	p.filters = append(p.filters, NewLengthFilter("max_length", 1024*1024)) // 1MB
	
	// Rate limiting
	p.filters = append(p.filters, NewRateLimitFilter("rate_limit", 100, 1000))
	
	// Critical regex patterns
	for _, pattern := range GetPatternsBySeverity("critical") {
		compiled, _ := p.registry.Get(pattern.Name)
		filter := &RegexFilter{
			BaseFilter: BaseFilter{
				name:     pattern.Name,
				priority: stream.PriorityCritical,
				enabled:  true,
			},
			pattern:  compiled,
			category: pattern.Category,
		}
		p.filters = append(p.filters, filter)
	}
	
	// Entropy detection
	p.filters = append(p.filters, NewEntropyFilter("high_entropy", 7.0))
	
	return nil
}

// Process runs all S1 edge filters
func (p *S1EdgeProcessor) Process(ctx context.Context, data stream.StreamData) (*stream.StageResult, error) {
	result := &stream.StageResult{
		Stage:      stream.StageS1Edge,
		Passed:     true,
		Confidence: 0.0,
		Findings:   make([]stream.Finding, 0),
	}
	
	start := time.Now()
	
	// Run filters in priority order
	for _, filter := range p.filters {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if filter.Match(data.Data) {
				result.Findings = append(result.Findings, stream.Finding{
					Type:       filter.GetName(),
					Severity:   "high", // S1 matches are always high priority
					Confidence: 0.9,
					Details: map[string]interface{}{
						"filter": filter.GetName(),
						"stage":  "s1_edge",
					},
				})
				
				// Critical filters block immediately
				if filter.GetPriority() == stream.PriorityCritical {
					result.Passed = false
					break
				}
			}
		}
	}
	
	result.Metrics = stream.StageMetrics{
		ProcessedCount: 1,
		AvgLatency:     time.Since(start),
		LastProcessed:  time.Now(),
	}
	
	return result, nil
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func extractSource(data []byte) string {
	// TODO: Implement source extraction based on protocol
	return "default"
}

func calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}
	
	// Count byte frequencies
	freq := make(map[byte]int)
	for _, b := range data {
		freq[b]++
	}
	
	// Calculate Shannon entropy
	var entropy float64
	dataLen := float64(len(data))
	
	for _, count := range freq {
		if count > 0 {
			p := float64(count) / dataLen
			entropy -= p * math.Log2(p)
		}
	}
	
	return entropy
}