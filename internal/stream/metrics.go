package stream

import (
    "fmt"
    "io"
    "net/http"
    "sync"
    "time"
)

// MetricsCollector collects and exports stream monitoring metrics
type MetricsCollector struct {
    mu sync.RWMutex
    
    // Counters
    eventsTotal       map[StreamEventType]int64
    alertsTotal       map[string]int64  // by severity
    bytesTransferred  map[Direction]int64
    patternsDetected  map[string]int64
    
    // Gauges
    activeProcesses   int
    activeStreams     int
    queueDepth        int
    
    // Histograms (simplified - buckets)
    messageSizes      map[string]int64  // bucket -> count
    responseLatencies map[string]int64  // bucket -> count
    
    // Info
    startTime         time.Time
    claudeVersion     string
    kernelVersion     string
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
    return &MetricsCollector{
        eventsTotal:       make(map[StreamEventType]int64),
        alertsTotal:       make(map[string]int64),
        bytesTransferred:  make(map[Direction]int64),
        patternsDetected:  make(map[string]int64),
        messageSizes:      make(map[string]int64),
        responseLatencies: make(map[string]int64),
        startTime:         time.Now(),
    }
}

// RecordEvent records a stream event
func (m *MetricsCollector) RecordEvent(event *StreamEvent) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.eventsTotal[event.Type]++
    m.bytesTransferred[event.Direction] += int64(event.Size)
    
    // Record message size in buckets
    bucket := m.getSizeBucket(event.Size)
    m.messageSizes[bucket]++
}

// RecordAlert records a security alert
func (m *MetricsCollector) RecordAlert(alert *SecurityAlert) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.alertsTotal[alert.Severity]++
}

// RecordPattern records a pattern detection
func (m *MetricsCollector) RecordPattern(patternName string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.patternsDetected[patternName]++
}

// RecordLatency records response latency
func (m *MetricsCollector) RecordLatency(latency time.Duration) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    bucket := m.getLatencyBucket(latency)
    m.responseLatencies[bucket]++
}

// UpdateGauges updates gauge metrics
func (m *MetricsCollector) UpdateGauges(activeProcs, activeStreams, queueDepth int) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.activeProcesses = activeProcs
    m.activeStreams = activeStreams
    m.queueDepth = queueDepth
}

// SetInfo sets informational metrics
func (m *MetricsCollector) SetInfo(claudeVer, kernelVer string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.claudeVersion = claudeVer
    m.kernelVersion = kernelVer
}

// WritePrometheus writes metrics in Prometheus format
func (m *MetricsCollector) WritePrometheus(w io.Writer) error {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    uptime := time.Since(m.startTime).Seconds()
    
    // Info metrics
    fmt.Fprintf(w, "# HELP strigoi_stream_info Strigoi stream monitor information\n")
    fmt.Fprintf(w, "# TYPE strigoi_stream_info gauge\n")
    fmt.Fprintf(w, "strigoi_stream_info{claude_version=\"%s\",kernel=\"%s\"} 1\n\n", 
        m.claudeVersion, m.kernelVersion)
    
    // Uptime
    fmt.Fprintf(w, "# HELP strigoi_stream_uptime_seconds Time since monitor started\n")
    fmt.Fprintf(w, "# TYPE strigoi_stream_uptime_seconds counter\n")
    fmt.Fprintf(w, "strigoi_stream_uptime_seconds %.2f\n\n", uptime)
    
    // Event counters
    fmt.Fprintf(w, "# HELP strigoi_stream_events_total Total stream events by type\n")
    fmt.Fprintf(w, "# TYPE strigoi_stream_events_total counter\n")
    for eventType, count := range m.eventsTotal {
        fmt.Fprintf(w, "strigoi_stream_events_total{type=\"%s\"} %d\n", eventType, count)
    }
    fmt.Fprintln(w)
    
    // Alert counters
    fmt.Fprintf(w, "# HELP strigoi_stream_alerts_total Total security alerts by severity\n")
    fmt.Fprintf(w, "# TYPE strigoi_stream_alerts_total counter\n")
    for severity, count := range m.alertsTotal {
        fmt.Fprintf(w, "strigoi_stream_alerts_total{severity=\"%s\"} %d\n", severity, count)
    }
    fmt.Fprintln(w)
    
    // Bytes transferred
    fmt.Fprintf(w, "# HELP strigoi_stream_bytes_transferred_total Bytes transferred by direction\n")
    fmt.Fprintf(w, "# TYPE strigoi_stream_bytes_transferred_total counter\n")
    for direction, bytes := range m.bytesTransferred {
        fmt.Fprintf(w, "strigoi_stream_bytes_transferred_total{direction=\"%s\"} %d\n", 
            direction, bytes)
    }
    fmt.Fprintln(w)
    
    // Pattern detections
    fmt.Fprintf(w, "# HELP strigoi_stream_patterns_detected_total Security patterns detected\n")
    fmt.Fprintf(w, "# TYPE strigoi_stream_patterns_detected_total counter\n")
    for pattern, count := range m.patternsDetected {
        fmt.Fprintf(w, "strigoi_stream_patterns_detected_total{pattern=\"%s\"} %d\n", 
            pattern, count)
    }
    fmt.Fprintln(w)
    
    // Active gauges
    fmt.Fprintf(w, "# HELP strigoi_stream_active_processes Number of monitored processes\n")
    fmt.Fprintf(w, "# TYPE strigoi_stream_active_processes gauge\n")
    fmt.Fprintf(w, "strigoi_stream_active_processes %d\n\n", m.activeProcesses)
    
    fmt.Fprintf(w, "# HELP strigoi_stream_active_streams Number of active streams\n")
    fmt.Fprintf(w, "# TYPE strigoi_stream_active_streams gauge\n")
    fmt.Fprintf(w, "strigoi_stream_active_streams %d\n\n", m.activeStreams)
    
    fmt.Fprintf(w, "# HELP strigoi_stream_queue_depth Current event queue depth\n")
    fmt.Fprintf(w, "# TYPE strigoi_stream_queue_depth gauge\n")
    fmt.Fprintf(w, "strigoi_stream_queue_depth %d\n\n", m.queueDepth)
    
    // Message size histogram
    fmt.Fprintf(w, "# HELP strigoi_stream_message_size_bytes Message size distribution\n")
    fmt.Fprintf(w, "# TYPE strigoi_stream_message_size_bytes histogram\n")
    var totalMessages int64
    var totalSize int64
    for bucket, count := range m.messageSizes {
        fmt.Fprintf(w, "strigoi_stream_message_size_bytes_bucket{le=\"%s\"} %d\n", 
            bucket, count)
        totalMessages += count
        // Estimate total size (simplified)
        if size := m.parseSizeBucket(bucket); size > 0 {
            totalSize += count * int64(size)
        }
    }
    fmt.Fprintf(w, "strigoi_stream_message_size_bytes_count %d\n", totalMessages)
    fmt.Fprintf(w, "strigoi_stream_message_size_bytes_sum %d\n\n", totalSize)
    
    // Response latency histogram
    fmt.Fprintf(w, "# HELP strigoi_stream_response_latency_ms Response latency distribution\n")
    fmt.Fprintf(w, "# TYPE strigoi_stream_response_latency_ms histogram\n")
    var totalLatencies int64
    for bucket, count := range m.responseLatencies {
        fmt.Fprintf(w, "strigoi_stream_response_latency_ms_bucket{le=\"%s\"} %d\n", 
            bucket, count)
        totalLatencies += count
    }
    fmt.Fprintf(w, "strigoi_stream_response_latency_ms_count %d\n\n", totalLatencies)
    
    return nil
}

// Helper functions for bucketing

func (m *MetricsCollector) getSizeBucket(size int) string {
    switch {
    case size <= 100:
        return "100"
    case size <= 1024:
        return "1024"
    case size <= 10240:
        return "10240"
    case size <= 102400:
        return "102400"
    case size <= 1048576:
        return "1048576"
    default:
        return "+Inf"
    }
}

func (m *MetricsCollector) getLatencyBucket(latency time.Duration) string {
    ms := latency.Milliseconds()
    switch {
    case ms <= 10:
        return "10"
    case ms <= 50:
        return "50"
    case ms <= 100:
        return "100"
    case ms <= 500:
        return "500"
    case ms <= 1000:
        return "1000"
    case ms <= 5000:
        return "5000"
    default:
        return "+Inf"
    }
}

func (m *MetricsCollector) parseSizeBucket(bucket string) int {
    // Simplified - in production use proper parsing
    switch bucket {
    case "100":
        return 100
    case "1024":
        return 1024
    case "10240":
        return 10240
    case "102400":
        return 102400
    case "1048576":
        return 1048576
    default:
        return 0
    }
}

// PrometheusHandler returns an HTTP handler for metrics endpoint
func (m *MetricsCollector) PrometheusHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain; version=0.0.4")
        m.WritePrometheus(w)
    }
}