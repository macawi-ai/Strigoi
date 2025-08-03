package stream

import (
	"crypto/sha256"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// FilterStats tracks filter performance
type FilterStats struct {
	Matched        uint64
	Processed      uint64
	AvgLatency     time.Duration
	LastMatch      time.Time
	HealthScore    float64
}

// BaseFilter provides common filter functionality
type BaseFilter struct {
	name     string
	priority Priority
	stats    FilterStats
	mu       sync.RWMutex
}

// GetName returns the filter name
func (bf *BaseFilter) GetName() string {
	return bf.name
}

// GetPriority returns the filter priority
func (bf *BaseFilter) GetPriority() Priority {
	return bf.priority
}

// updateStats updates filter statistics
func (bf *BaseFilter) updateStats(matched bool, latency time.Duration) {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	
	atomic.AddUint64(&bf.stats.Processed, 1)
	if matched {
		atomic.AddUint64(&bf.stats.Matched, 1)
		bf.stats.LastMatch = time.Now()
	}
	
	// Update average latency
	if bf.stats.AvgLatency == 0 {
		bf.stats.AvgLatency = latency
	} else {
		bf.stats.AvgLatency = (bf.stats.AvgLatency + latency) / 2
	}
}

// RegexFilter performs pattern matching using pre-compiled regex
type RegexFilter struct {
	BaseFilter
	patterns []*regexp.Regexp
	category string
}

// NewRegexFilter creates a filter with pre-compiled patterns
func NewRegexFilter(name, category string, patterns []string, priority Priority) (*RegexFilter, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %s: %w", pattern, err)
		}
		compiled = append(compiled, re)
	}
	
	return &RegexFilter{
		BaseFilter: BaseFilter{
			name:     name,
			priority: priority,
		},
		patterns: compiled,
		category: category,
	}, nil
}

// Match checks if data matches any pattern
func (rf *RegexFilter) Match(data []byte) bool {
	start := time.Now()
	defer func() {
		rf.updateStats(false, time.Since(start)) // Updated in loop if matched
	}()
	
	for _, pattern := range rf.patterns {
		if pattern.Match(data) {
			rf.updateStats(true, time.Since(start))
			return true
		}
	}
	return false
}

// KeywordFilter performs fast string matching without regex
type KeywordFilter struct {
	BaseFilter
	keywords map[string]bool
	caseSensitive bool
}

// NewKeywordFilter creates a filter for exact string matching
func NewKeywordFilter(name string, keywords []string, caseSensitive bool, priority Priority) *KeywordFilter {
	keywordMap := make(map[string]bool, len(keywords))
	for _, keyword := range keywords {
		if !caseSensitive {
			keyword = strings.ToLower(keyword)
		}
		keywordMap[keyword] = true
	}
	
	return &KeywordFilter{
		BaseFilter: BaseFilter{
			name:     name,
			priority: priority,
		},
		keywords:      keywordMap,
		caseSensitive: caseSensitive,
	}
}

// Match checks if data contains any keywords
func (kf *KeywordFilter) Match(data []byte) bool {
	start := time.Now()
	defer func() {
		kf.updateStats(false, time.Since(start))
	}()
	
	text := string(data)
	if !kf.caseSensitive {
		text = strings.ToLower(text)
	}
	
	for keyword := range kf.keywords {
		if strings.Contains(text, keyword) {
			kf.updateStats(true, time.Since(start))
			return true
		}
	}
	return false
}

// RateLimitFilter prevents flooding attacks
type RateLimitFilter struct {
	BaseFilter
	tokens       map[string]*tokenBucket
	tokensPerSec int
	burstSize    int
	mu           sync.RWMutex
}

// tokenBucket implements token bucket algorithm
type tokenBucket struct {
	tokens    float64
	lastCheck time.Time
	mu        sync.Mutex
}

// NewRateLimitFilter creates a filter that limits data rate per source
func NewRateLimitFilter(name string, tokensPerSec, burstSize int, priority Priority) *RateLimitFilter {
	return &RateLimitFilter{
		BaseFilter: BaseFilter{
			name:     name,
			priority: priority,
		},
		tokens:       make(map[string]*tokenBucket),
		tokensPerSec: tokensPerSec,
		burstSize:    burstSize,
	}
}

// Match checks if source is within rate limit
func (rf *RateLimitFilter) Match(data []byte) bool {
	start := time.Now()
	
	// Extract source from data (simplified - real impl would parse metadata)
	hash := sha256.Sum256(data[:min(len(data), 32)])
	source := fmt.Sprintf("%x", hash[:8])
	
	rf.mu.Lock()
	bucket, exists := rf.tokens[source]
	if !exists {
		bucket = &tokenBucket{
			tokens:    float64(rf.burstSize),
			lastCheck: time.Now(),
		}
		rf.tokens[source] = bucket
	}
	rf.mu.Unlock()
	
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	
	// Refill tokens
	elapsed := time.Since(bucket.lastCheck).Seconds()
	bucket.tokens = math.Min(float64(rf.burstSize), bucket.tokens+elapsed*float64(rf.tokensPerSec))
	bucket.lastCheck = time.Now()
	
	// Check if we have tokens
	if bucket.tokens >= 1 {
		bucket.tokens--
		rf.updateStats(true, time.Since(start))
		return true
	}
	
	rf.updateStats(false, time.Since(start))
	return false
}

// EntropyFilter detects encrypted or compressed data
type EntropyFilter struct {
	BaseFilter
	threshold float64
}

// NewEntropyFilter creates a filter that checks data entropy
func NewEntropyFilter(name string, threshold float64, priority Priority) *EntropyFilter {
	return &EntropyFilter{
		BaseFilter: BaseFilter{
			name:     name,
			priority: priority,
		},
		threshold: threshold,
	}
}

// Match checks if data entropy exceeds threshold
func (ef *EntropyFilter) Match(data []byte) bool {
	start := time.Now()
	defer func() {
		ef.updateStats(false, time.Since(start))
	}()
	
	entropy := calculateEntropy(data)
	if entropy > ef.threshold {
		ef.updateStats(true, time.Since(start))
		return true
	}
	return false
}

// calculateEntropy computes Shannon entropy
func calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}
	
	freq := make(map[byte]int)
	for _, b := range data {
		freq[b]++
	}
	
	entropy := 0.0
	dataLen := float64(len(data))
	for _, count := range freq {
		if count > 0 {
			prob := float64(count) / dataLen
			entropy -= prob * math.Log2(prob)
		}
	}
	
	return entropy
}

// LengthFilter quickly rejects oversized data
type LengthFilter struct {
	BaseFilter
	maxLength int
}

// NewLengthFilter creates a filter that checks data length
func NewLengthFilter(name string, maxLength int, priority Priority) *LengthFilter {
	return &LengthFilter{
		BaseFilter: BaseFilter{
			name:     name,
			priority: priority,
		},
		maxLength: maxLength,
	}
}

// Match checks if data length is within limit
func (lf *LengthFilter) Match(data []byte) bool {
	start := time.Now()
	matched := len(data) <= lf.maxLength
	lf.updateStats(matched, time.Since(start))
	return matched
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}