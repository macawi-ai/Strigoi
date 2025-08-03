package licensing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TelemetryClient handles DNS-based telemetry
type TelemetryClient struct {
	domain     string
	version    string
	instanceID string
	resolver   *net.Resolver
	
	// Rate limiting
	lastEvent  time.Time
	eventCount int
	mu         sync.Mutex
}

// NewTelemetryClient creates a new telemetry client
func NewTelemetryClient() *TelemetryClient {
	return &TelemetryClient{
		domain:     "validation.macawi.io",
		version:    "v1",
		instanceID: generateInstanceID(),
		resolver: &net.Resolver{
			PreferGo: true,
		},
	}
}

// SendEvent sends a telemetry event via DNS
func (tc *TelemetryClient) SendEvent(action string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	// Rate limiting - max 10 events per minute
	now := time.Now()
	if now.Sub(tc.lastEvent) < time.Minute {
		tc.eventCount++
		if tc.eventCount > 10 {
			return // Skip event
		}
	} else {
		tc.eventCount = 1
		tc.lastEvent = now
	}
	
	// Construct DNS query
	timestamp := now.Unix()
	hash := tc.computeHash(timestamp)
	
	query := fmt.Sprintf("%s.%s.%d.%s.%s",
		tc.version,
		hash[:6],
		timestamp,
		action,
		tc.domain)
	
	// Non-blocking DNS query
	go tc.performDNSQuery(query)
}

// performDNSQuery executes the DNS query
func (tc *TelemetryClient) performDNSQuery(query string) {
	// Set timeout for DNS query
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Perform DNS lookup (we don't care about the result)
	tc.resolver.LookupHost(ctx, query)
}

// computeHash generates a hash for the telemetry event
func (tc *TelemetryClient) computeHash(timestamp int64) string {
	data := fmt.Sprintf("%s:%d", tc.instanceID, timestamp)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Close performs any cleanup
func (tc *TelemetryClient) Close() {
	// Nothing to clean up for DNS client
}

// generateInstanceID creates a unique instance identifier
func generateInstanceID() string {
	// Combine hostname and current time for uniqueness
	hostname, _ := os.Hostname()
	data := fmt.Sprintf("%s:%d", hostname, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

// TelemetryEvent represents a telemetry event
type TelemetryEvent struct {
	Version   string
	Hash      string
	Timestamp int64
	Action    string
}

// ParseTelemetryQuery parses a DNS telemetry query
func ParseTelemetryQuery(query string) (*TelemetryEvent, error) {
	// Expected format: v1.a7b9c2.1737389400.start.validation.macawi.io
	parts := strings.Split(query, ".")
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid telemetry query format")
	}
	
	timestamp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}
	
	return &TelemetryEvent{
		Version:   parts[0],
		Hash:      parts[1],
		Timestamp: timestamp,
		Action:    parts[3],
	}, nil
}

// ComplianceTelemetry handles compliance-specific telemetry
type ComplianceTelemetry struct {
	*TelemetryClient
	policies []string
}

// NewComplianceTelemetry creates compliance-aware telemetry
func NewComplianceTelemetry(policies []string) *ComplianceTelemetry {
	return &ComplianceTelemetry{
		TelemetryClient: NewTelemetryClient(),
		policies:        policies,
	}
}

// SendComplianceEvent sends compliance-specific events
func (ct *ComplianceTelemetry) SendComplianceEvent(action string, metadata map[string]string) {
	// For compliance, we need to be extra careful about what we send
	// Only send action types, no metadata that could contain PII
	
	// Validate action is allowed for compliance
	allowedActions := map[string]bool{
		"scan_start":       true,
		"scan_complete":    true,
		"error_occurred":   true,
		"license_check":    true,
		"update_check":     true,
	}
	
	if !allowedActions[action] {
		return // Skip non-compliant actions
	}
	
	// Send basic event
	ct.SendEvent(action)
}

// MetricsCollector aggregates telemetry metrics
type MetricsCollector struct {
	events map[string]int64
	mu     sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		events: make(map[string]int64),
	}
}

// RecordEvent records a telemetry event
func (mc *MetricsCollector) RecordEvent(event *TelemetryEvent) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", event.Version, event.Action)
	mc.events[key]++
}

// GetMetrics returns current metrics
func (mc *MetricsCollector) GetMetrics() map[string]int64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	// Return a copy
	metrics := make(map[string]int64)
	for k, v := range mc.events {
		metrics[k] = v
	}
	
	return metrics
}

// Reset clears all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.events = make(map[string]int64)
}