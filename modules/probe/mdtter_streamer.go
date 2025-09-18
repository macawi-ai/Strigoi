package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// MDTTERStreamer handles streaming MDTTER events to various endpoints
type MDTTERStreamer struct {
	config      StreamerConfig
	translators map[string]*SIEMTranslator
	endpoints   map[string]StreamEndpoint

	// Channels
	input    <-chan *MDTTEREvent
	shutdown chan struct{}

	// Metrics to show what legacy systems are missing
	metricsmu      sync.RWMutex
	eventCount     int64
	dimensionsLost int64

	wg sync.WaitGroup
}

// StreamerConfig configures the MDTTER streamer
type StreamerConfig struct {
	// Full-dimension endpoints (the future)
	MDTTEREndpoints []string

	// Legacy endpoints (backwards compatibility)
	LegacyEndpoints map[string]LegacyEndpointConfig

	// Performance tuning
	BatchSize     int
	FlushInterval time.Duration
	RetryAttempts int
}

// LegacyEndpointConfig defines a legacy SIEM endpoint
type LegacyEndpointConfig struct {
	Type     SIEMFormat
	URL      string
	APIKey   string
	Index    string // For Elastic/Splunk
	Facility string // For syslog
}

// StreamEndpoint represents a destination for events
type StreamEndpoint interface {
	Send(events []interface{}) error
	Close() error
}

// NewMDTTERStreamer creates a new streamer
func NewMDTTERStreamer(config StreamerConfig, input <-chan *MDTTEREvent) *MDTTERStreamer {
	s := &MDTTERStreamer{
		config:      config,
		translators: make(map[string]*SIEMTranslator),
		endpoints:   make(map[string]StreamEndpoint),
		input:       input,
		shutdown:    make(chan struct{}),
	}

	// Initialize translators for each legacy endpoint
	for name, cfg := range config.LegacyEndpoints {
		s.translators[name] = NewSIEMTranslator(cfg.Type)
		// Initialize appropriate endpoint (Kafka, HTTP, etc.)
		s.endpoints[name] = s.createEndpoint(name, cfg)
	}

	return s
}

// Start begins streaming events
func (s *MDTTERStreamer) Start(ctx context.Context) error {
	// Start workers for each endpoint
	for name := range s.endpoints {
		s.wg.Add(1)
		go s.streamWorker(ctx, name)
	}

	// Start metrics reporter
	s.wg.Add(1)
	go s.metricsReporter(ctx)

	return nil
}

// streamWorker handles streaming to a specific endpoint
func (s *MDTTERStreamer) streamWorker(ctx context.Context, endpointName string) {
	defer s.wg.Done()

	endpoint := s.endpoints[endpointName]
	translator := s.translators[endpointName]

	batch := make([]interface{}, 0, s.config.BatchSize)
	ticker := time.NewTicker(s.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Flush remaining events
			if len(batch) > 0 {
				s.sendBatch(endpoint, batch)
			}
			return

		case event := <-s.input:
			// Translate to legacy format
			translated, err := translator.TranslateToLegacy(event)
			if err != nil {
				log.Printf("Translation error for %s: %v", endpointName, err)
				continue
			}

			batch = append(batch, translated)

			// Track metrics
			s.updateMetrics(translator)

			// Send if batch is full
			if len(batch) >= s.config.BatchSize {
				s.sendBatch(endpoint, batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			// Periodic flush
			if len(batch) > 0 {
				s.sendBatch(endpoint, batch)
				batch = batch[:0]
			}
		}
	}
}

// sendBatch sends a batch of events to an endpoint
func (s *MDTTERStreamer) sendBatch(endpoint StreamEndpoint, batch []interface{}) {
	attempts := 0
	for attempts < s.config.RetryAttempts {
		err := endpoint.Send(batch)
		if err == nil {
			return
		}

		attempts++
		log.Printf("Send failed (attempt %d/%d): %v", attempts, s.config.RetryAttempts, err)

		// Exponential backoff
		time.Sleep(time.Duration(attempts) * time.Second)
	}

	log.Printf("Failed to send batch after %d attempts", s.config.RetryAttempts)
}

// updateMetrics tracks what we're losing in translation
func (s *MDTTERStreamer) updateMetrics(translator *SIEMTranslator) {
	s.metricsmu.Lock()
	defer s.metricsmu.Unlock()

	s.eventCount++

	// Calculate dimensions lost
	lost := translator.GetLostDimensions()
	for _, v := range lost {
		if dims, ok := v.(int); ok {
			s.dimensionsLost += int64(dims)
		}
	}
}

// metricsReporter periodically reports what legacy systems are missing
func (s *MDTTERStreamer) metricsReporter(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reportMetrics()
		}
	}
}

// reportMetrics logs what dimensional richness is being lost
func (s *MDTTERStreamer) reportMetrics() {
	s.metricsmu.RLock()
	defer s.metricsmu.RUnlock()

	if s.eventCount == 0 {
		return
	}

	avgDimensionsLost := s.dimensionsLost / s.eventCount

	log.Printf(
		"MDTTER Streaming Metrics: %d events translated, ~%d dimensions lost per event. "+
			"Legacy systems are seeing only shadows of the true security topology.",
		s.eventCount,
		avgDimensionsLost,
	)
}

// createEndpoint creates the appropriate endpoint for a configuration
func (s *MDTTERStreamer) createEndpoint(name string, cfg LegacyEndpointConfig) StreamEndpoint {
	switch cfg.Type {
	case FormatSplunk:
		return NewSplunkHECEndpoint(cfg.URL, cfg.APIKey, cfg.Index)
	case FormatElastic:
		return NewElasticEndpoint(cfg.URL, cfg.APIKey, cfg.Index)
	default:
		// For now, return a mock endpoint
		return &MockEndpoint{name: name}
	}
}

// MockEndpoint for testing
type MockEndpoint struct {
	name string
}

func (m *MockEndpoint) Send(events []interface{}) error {
	log.Printf("MockEndpoint %s: Would send %d events", m.name, len(events))
	return nil
}

func (m *MockEndpoint) Close() error {
	return nil
}

// SplunkHECEndpoint sends to Splunk HTTP Event Collector
type SplunkHECEndpoint struct {
	url    string
	token  string
	index  string
	client HTTPClient
}

// HTTPClient interface for testing
type HTTPClient interface {
	Post(url string, contentType string, body []byte) error
}

func NewSplunkHECEndpoint(url, token, index string) *SplunkHECEndpoint {
	return &SplunkHECEndpoint{
		url:   url,
		token: token,
		index: index,
		// client would be initialized with real HTTP client
	}
}

func (s *SplunkHECEndpoint) Send(events []interface{}) error {
	// Build Splunk HEC payload
	payload := make([]map[string]interface{}, len(events))
	for i, event := range events {
		if hecEvent, ok := event.(map[string]interface{}); ok {
			// Add index if specified
			if s.index != "" && hecEvent["event"] != nil {
				hecEvent["index"] = s.index
			}
			payload[i] = hecEvent
		}
	}

	// Convert to JSON
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// In real implementation, would POST to Splunk
	log.Printf("Would send %d events (%d bytes) to Splunk HEC", len(events), len(data))
	return nil
}

func (s *SplunkHECEndpoint) Close() error {
	return nil
}

// ElasticEndpoint sends to Elasticsearch
type ElasticEndpoint struct {
	url    string
	apiKey string
	index  string
}

func NewElasticEndpoint(url, apiKey, index string) *ElasticEndpoint {
	return &ElasticEndpoint{
		url:    url,
		apiKey: apiKey,
		index:  index,
	}
}

func (e *ElasticEndpoint) Send(events []interface{}) error {
	// Build bulk request
	var bulkBody []byte

	for _, event := range events {
		// Add bulk metadata
		meta := map[string]interface{}{
			"index": map[string]string{
				"_index": e.index,
			},
		}

		metaJSON, _ := json.Marshal(meta)
		eventJSON, _ := json.Marshal(event)

		bulkBody = append(bulkBody, metaJSON...)
		bulkBody = append(bulkBody, '\n')
		bulkBody = append(bulkBody, eventJSON...)
		bulkBody = append(bulkBody, '\n')
	}

	// In real implementation, would POST to Elasticsearch
	log.Printf("Would send %d events to Elasticsearch", len(events))
	return nil
}

func (e *ElasticEndpoint) Close() error {
	return nil
}

// Example usage showing the evolution path
func ExampleMDTTEREvolution() {
	// Configuration showing both legacy and future endpoints
	config := StreamerConfig{
		// The future - full MDTTER endpoints
		MDTTEREndpoints: []string{
			"kafka://mdtter-cluster:9092/mdtter-events",
			"grpc://topology-engine:50051",
		},

		// Legacy compatibility - speaking their language
		LegacyEndpoints: map[string]LegacyEndpointConfig{
			"splunk-prod": {
				Type:   FormatSplunk,
				URL:    "https://splunk.company.com:8088/services/collector",
				APIKey: "xxx",
				Index:  "security",
			},
			"elastic-soc": {
				Type:   FormatElastic,
				URL:    "https://elastic.company.com:9200",
				APIKey: "yyy",
				Index:  "security-events-2025.02",
			},
		},

		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		RetryAttempts: 3,
	}

	// Create channels
	mdtterEvents := make(chan *MDTTEREvent, 1000)

	// Create streamer
	streamer := NewMDTTERStreamer(config, mdtterEvents)

	// Start streaming
	ctx := context.Background()
	if err := streamer.Start(ctx); err != nil {
		log.Printf("Failed to start MDTTER streamer: %v", err)
	}

	log.Println("MDTTER Streamer: Feeding legacy systems their comfort food while we hunt in higher dimensions")
}
