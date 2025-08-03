package realtime

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// ResponseCache stores recent threat responses for fast lookup
type ResponseCache struct {
	cache map[string]*cacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

type cacheEntry struct {
	response  *DefenseResponse
	timestamp time.Time
}

// NewResponseCache creates a new response cache
func NewResponseCache() *ResponseCache {
	rc := &ResponseCache{
		cache: make(map[string]*cacheEntry),
		ttl:   5 * time.Minute, // Short TTL for real-time threats
	}
	
	// Start cleanup goroutine
	go rc.cleanup()
	
	return rc
}

// Get retrieves a cached response if available and not expired
func (rc *ResponseCache) Get(threat ThreatEvent) *DefenseResponse {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	key := rc.threatKey(threat)
	entry, exists := rc.cache[key]
	if !exists {
		return nil
	}
	
	// Check if expired
	if time.Since(entry.timestamp) > rc.ttl {
		return nil
	}
	
	// Return copy to prevent mutation
	resp := *entry.response
	return &resp
}

// Set stores a response in the cache
func (rc *ResponseCache) Set(threat ThreatEvent, response *DefenseResponse) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	key := rc.threatKey(threat)
	rc.cache[key] = &cacheEntry{
		response:  response,
		timestamp: time.Now(),
	}
}

// Size returns the number of cached entries
func (rc *ResponseCache) Size() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.cache)
}

// threatKey generates a unique key for a threat
func (rc *ResponseCache) threatKey(threat ThreatEvent) string {
	h := sha256.New()
	h.Write([]byte(threat.Type))
	h.Write([]byte(threat.Source))
	// Add payload hash if it's a string
	if payload, ok := threat.Payload.(string); ok {
		h.Write([]byte(payload))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// cleanup periodically removes expired entries
func (rc *ResponseCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		for key, entry := range rc.cache {
			if now.Sub(entry.timestamp) > rc.ttl {
				delete(rc.cache, key)
			}
		}
		rc.mu.Unlock()
	}
}