package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// System manages comprehensive error and performance telemetry
type System struct {
	config       Config
	collectors   map[string]MetricCollector
	errorTracker *ErrorTracker
	perfTracker  *PerformanceTracker
	alertManager *AlertManager
	httpServer   *http.Server
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// Config configures the telemetry system
type Config struct {
	// HTTP endpoint configuration
	ListenAddress string
	MetricsPath   string

	// Collection intervals
	CollectionInterval time.Duration
	RetentionPeriod    time.Duration

	// Alert configuration
	EnableAlerts    bool
	AlertWebhookURL string
	AlertThresholds AlertThresholds

	// Export configuration
	EnablePrometheus bool
	EnableGrafana    bool
	GrafanaAPIKey    string
	GrafanaURL       string
}

// AlertThresholds defines thresholds for alerts
type AlertThresholds struct {
	ErrorRateThreshold   float64
	LatencyP99Threshold  time.Duration
	MemoryUsageThreshold uint64
	CPUUsageThreshold    float64
	QueueDepthThreshold  int
	ConsecutiveFailures  int
}

// MetricCollector interface for different metric types
type MetricCollector interface {
	Collect() map[string]interface{}
	Export() string
	Reset()
}

// NewSystem creates a new telemetry system
func NewSystem(config Config) (*System, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ts := &System{
		config:       config,
		collectors:   make(map[string]MetricCollector),
		errorTracker: NewErrorTracker(),
		perfTracker:  NewPerformanceTracker(),
		alertManager: NewAlertManager(config.AlertWebhookURL, config.AlertThresholds),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Initialize collectors
	ts.collectors["errors"] = ts.errorTracker
	ts.collectors["performance"] = ts.perfTracker
	ts.collectors["system"] = NewSystemCollector()
	ts.collectors["application"] = NewApplicationCollector()

	// Setup HTTP server
	ts.setupHTTPServer()

	// Start collection loop
	ts.wg.Add(1)
	go ts.collectionLoop()

	// Start alert monitoring
	if config.EnableAlerts {
		ts.wg.Add(1)
		go ts.alertLoop()
	}

	return ts, nil
}

// setupHTTPServer sets up HTTP endpoints
func (ts *System) setupHTTPServer() {
	mux := http.NewServeMux()

	// Prometheus metrics endpoint
	if ts.config.EnablePrometheus {
		mux.HandleFunc(ts.config.MetricsPath, ts.handlePrometheusMetrics)
	}

	// Health endpoint
	mux.HandleFunc("/health", ts.handleHealth)

	// Detailed metrics endpoint
	mux.HandleFunc("/metrics/detailed", ts.handleDetailedMetrics)

	// Alert status endpoint
	mux.HandleFunc("/alerts", ts.handleAlerts)

	// Grafana dashboard endpoint
	if ts.config.EnableGrafana {
		mux.HandleFunc("/grafana/dashboard", ts.handleGrafanaDashboard)
	}

	ts.httpServer = &http.Server{
		Addr:         ts.config.ListenAddress,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := ts.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// TODO: Add proper error logging
			_ = err
		}
	}()
}

// handlePrometheusMetrics serves Prometheus format metrics
func (ts *System) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	for name, collector := range ts.collectors {
		output := collector.Export()
		fmt.Fprintf(w, "# Collector: %s\n%s\n", name, output)
	}
}

// handleDetailedMetrics serves detailed JSON metrics
func (ts *System) handleDetailedMetrics(w http.ResponseWriter, r *http.Request) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	metrics := make(map[string]interface{})

	for name, collector := range ts.collectors {
		metrics[name] = collector.Collect()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
	}
}

// handleHealth serves health status
func (ts *System) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := ts.GetHealth()

	w.Header().Set("Content-Type", "application/json")
	if health.Healthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, "Failed to encode health status", http.StatusInternalServerError)
	}
}

// handleAlerts serves alert status
func (ts *System) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := ts.alertManager.GetActiveAlerts()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(alerts); err != nil {
		http.Error(w, "Failed to encode alerts", http.StatusInternalServerError)
	}
}

// handleGrafanaDashboard serves Grafana dashboard configuration
func (ts *System) handleGrafanaDashboard(w http.ResponseWriter, r *http.Request) {
	dashboard := ts.generateGrafanaDashboard()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// collectionLoop periodically collects metrics
func (ts *System) collectionLoop() {
	defer ts.wg.Done()

	ticker := time.NewTicker(ts.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.collectMetrics()

		case <-ts.ctx.Done():
			return
		}
	}
}

// collectMetrics collects metrics from all sources
func (ts *System) collectMetrics() {
	// Collect metrics is handled by individual collectors
	// This could aggregate or process collected data
}

// alertLoop monitors metrics for alert conditions
func (ts *System) alertLoop() {
	defer ts.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.checkAlertConditions()

		case <-ts.ctx.Done():
			return
		}
	}
}

// checkAlertConditions checks for alert conditions
func (ts *System) checkAlertConditions() {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// Check error rate
	errorMetrics := ts.errorTracker.Collect()
	if errorRate, ok := errorMetrics["error_rate"].(float64); ok {
		if errorRate > ts.config.AlertThresholds.ErrorRateThreshold {
			ts.alertManager.TriggerAlert(Alert{
				Type:     AlertTypeErrorRate,
				Severity: SeverityCritical,
				Message:  fmt.Sprintf("Error rate %.2f%% exceeds threshold", errorRate*100),
				Value:    errorRate,
			})
		}
	}

	// Check performance
	perfMetrics := ts.perfTracker.Collect()
	if p99, ok := perfMetrics["latency_p99"].(time.Duration); ok {
		if p99 > ts.config.AlertThresholds.LatencyP99Threshold {
			ts.alertManager.TriggerAlert(Alert{
				Type:     AlertTypeLatency,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("P99 latency %v exceeds threshold", p99),
				Value:    float64(p99.Milliseconds()),
			})
		}
	}

	// Check system resources
	sysMetrics := ts.collectors["system"].Collect()
	if memUsage, ok := sysMetrics["memory_usage"].(uint64); ok {
		if memUsage > ts.config.AlertThresholds.MemoryUsageThreshold {
			ts.alertManager.TriggerAlert(Alert{
				Type:     AlertTypeMemory,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Memory usage %d MB exceeds threshold", memUsage/1024/1024),
				Value:    float64(memUsage),
			})
		}
	}
}

// RecordError records an error
func (ts *System) RecordError(err error, context string, severity ErrorSeverity) {
	ts.errorTracker.RecordError(err, context, severity)
}

// RecordLatency records operation latency
func (ts *System) RecordLatency(operation string, duration time.Duration) {
	ts.perfTracker.RecordLatency(operation, duration)
}

// RecordThroughput records throughput metric
func (ts *System) RecordThroughput(metric string, value float64) {
	ts.perfTracker.RecordThroughput(metric, value)
}

// GetHealth returns system health status
func (ts *System) GetHealth() HealthStatus {
	errorMetrics := ts.errorTracker.Collect()
	perfMetrics := ts.perfTracker.Collect()

	errorRate := float64(0)
	if rate, ok := errorMetrics["error_rate"].(float64); ok {
		errorRate = rate
	}

	avgLatency := time.Duration(0)
	if latency, ok := perfMetrics["avg_latency"].(time.Duration); ok {
		avgLatency = latency
	}

	healthy := errorRate < 0.1 && avgLatency < 1*time.Second

	return HealthStatus{
		Healthy:        healthy,
		ErrorRate:      errorRate,
		AverageLatency: avgLatency,
		ActiveAlerts:   len(ts.alertManager.GetActiveAlerts()),
		LastCheck:      time.Now(),
	}
}

// generateGrafanaDashboard generates Grafana dashboard JSON
func (ts *System) generateGrafanaDashboard() map[string]interface{} {
	return map[string]interface{}{
		"dashboard": map[string]interface{}{
			"title": "Strigoi Telemetry Dashboard",
			"panels": []interface{}{
				map[string]interface{}{
					"title": "Error Rate",
					"type":  "graph",
					"targets": []map[string]string{
						{"expr": "rate(strigoi_errors_total[5m])"},
					},
				},
				map[string]interface{}{
					"title": "Latency Percentiles",
					"type":  "graph",
					"targets": []map[string]string{
						{"expr": "strigoi_latency_p50"},
						{"expr": "strigoi_latency_p95"},
						{"expr": "strigoi_latency_p99"},
					},
				},
				map[string]interface{}{
					"title": "Throughput",
					"type":  "graph",
					"targets": []map[string]string{
						{"expr": "rate(strigoi_events_processed_total[5m])"},
					},
				},
				map[string]interface{}{
					"title": "Memory Usage",
					"type":  "graph",
					"targets": []map[string]string{
						{"expr": "strigoi_memory_usage_bytes"},
					},
				},
				map[string]interface{}{
					"title": "Active Alerts",
					"type":  "stat",
					"targets": []map[string]string{
						{"expr": "strigoi_alerts_active"},
					},
				},
			},
		},
	}
}

// Shutdown gracefully shuts down the telemetry system
func (ts *System) Shutdown(timeout time.Duration) error {
	ts.cancel()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := ts.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP server shutdown failed: %w", err)
	}

	// Wait for workers
	done := make(chan struct{})
	go func() {
		ts.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Clean shutdown
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout exceeded")
	}

	return nil
}

// ErrorTracker tracks errors and error rates
type ErrorTracker struct {
	errors      map[string]*ErrorMetric
	totalErrors uint64
	windowSize  time.Duration
	mu          sync.RWMutex
}

// ErrorMetric tracks errors for a specific context
type ErrorMetric struct {
	Count        uint64
	LastError    error
	LastOccurred time.Time
	Severities   map[ErrorSeverity]uint64
}

// ErrorSeverity defines error severity levels
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// NewErrorTracker creates an error tracker
func NewErrorTracker() *ErrorTracker {
	return &ErrorTracker{
		errors:     make(map[string]*ErrorMetric),
		windowSize: 5 * time.Minute,
	}
}

// RecordError records an error occurrence
func (et *ErrorTracker) RecordError(err error, context string, severity ErrorSeverity) {
	et.mu.Lock()
	defer et.mu.Unlock()

	atomic.AddUint64(&et.totalErrors, 1)

	metric, exists := et.errors[context]
	if !exists {
		metric = &ErrorMetric{
			Severities: make(map[ErrorSeverity]uint64),
		}
		et.errors[context] = metric
	}

	atomic.AddUint64(&metric.Count, 1)
	metric.LastError = err
	metric.LastOccurred = time.Now()
	metric.Severities[severity]++
}

// Collect returns error metrics
func (et *ErrorTracker) Collect() map[string]interface{} {
	et.mu.RLock()
	defer et.mu.RUnlock()

	totalErrors := atomic.LoadUint64(&et.totalErrors)

	// Calculate error rate (simplified - in production use sliding window)
	errorRate := float64(0)
	if totalErrors > 0 {
		// This is simplified - real implementation would track time windows
		errorRate = float64(totalErrors) / 1000.0 // Assume 1000 operations
	}

	// Count by severity
	severityCounts := make(map[string]uint64)
	for _, metric := range et.errors {
		for sev, count := range metric.Severities {
			key := fmt.Sprintf("severity_%d", sev)
			severityCounts[key] += count
		}
	}

	// Top error contexts
	topErrors := make([]map[string]interface{}, 0)
	for context, metric := range et.errors {
		topErrors = append(topErrors, map[string]interface{}{
			"context":       context,
			"count":         atomic.LoadUint64(&metric.Count),
			"last_occurred": metric.LastOccurred,
		})
	}

	return map[string]interface{}{
		"total_errors":    totalErrors,
		"error_rate":      errorRate,
		"severity_counts": severityCounts,
		"top_errors":      topErrors,
	}
}

// Export returns Prometheus format metrics
func (et *ErrorTracker) Export() string {
	metrics := et.Collect()

	output := ""

	// Total errors
	output += "# HELP strigoi_errors_total Total number of errors\n"
	output += "# TYPE strigoi_errors_total counter\n"
	output += fmt.Sprintf("strigoi_errors_total %d\n\n", metrics["total_errors"])

	// Error rate
	output += "# HELP strigoi_error_rate Current error rate\n"
	output += "# TYPE strigoi_error_rate gauge\n"
	output += fmt.Sprintf("strigoi_error_rate %f\n\n", metrics["error_rate"])

	// Errors by context
	et.mu.RLock()
	defer et.mu.RUnlock()

	output += "# HELP strigoi_errors_by_context Errors by context\n"
	output += "# TYPE strigoi_errors_by_context counter\n"
	for context, metric := range et.errors {
		output += fmt.Sprintf("strigoi_errors_by_context{context=\"%s\"} %d\n", context, metric.Count)
	}

	return output
}

// Reset resets error metrics
func (et *ErrorTracker) Reset() {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.errors = make(map[string]*ErrorMetric)
	atomic.StoreUint64(&et.totalErrors, 0)
}

// PerformanceTracker tracks performance metrics
type PerformanceTracker struct {
	latencies   map[string]*LatencyMetric
	throughputs map[string]*ThroughputMetric
	mu          sync.RWMutex
}

// LatencyMetric tracks latency for an operation
type LatencyMetric struct {
	samples  []time.Duration
	capacity int
	position int
	count    uint64
	sum      uint64 // in nanoseconds
}

// ThroughputMetric tracks throughput
type ThroughputMetric struct {
	value     float64
	timestamp time.Time
}

// NewPerformanceTracker creates a performance tracker
func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		latencies:   make(map[string]*LatencyMetric),
		throughputs: make(map[string]*ThroughputMetric),
	}
}

// RecordLatency records operation latency
func (pt *PerformanceTracker) RecordLatency(operation string, duration time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	metric, exists := pt.latencies[operation]
	if !exists {
		metric = &LatencyMetric{
			samples:  make([]time.Duration, 1000),
			capacity: 1000,
		}
		pt.latencies[operation] = metric
	}

	metric.samples[metric.position] = duration
	metric.position = (metric.position + 1) % metric.capacity
	atomic.AddUint64(&metric.count, 1)
	atomic.AddUint64(&metric.sum, uint64(duration.Nanoseconds()))
}

// RecordThroughput records throughput metric
func (pt *PerformanceTracker) RecordThroughput(metric string, value float64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.throughputs[metric] = &ThroughputMetric{
		value:     value,
		timestamp: time.Now(),
	}
}

// Collect returns performance metrics
func (pt *PerformanceTracker) Collect() map[string]interface{} {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	// Calculate latency percentiles
	var allLatencies []time.Duration
	for _, metric := range pt.latencies {
		count := int(atomic.LoadUint64(&metric.count))
		if count > metric.capacity {
			count = metric.capacity
		}
		for i := 0; i < count; i++ {
			allLatencies = append(allLatencies, metric.samples[i])
		}
	}

	p50 := calculatePercentile(allLatencies, 50)
	p95 := calculatePercentile(allLatencies, 95)
	p99 := calculatePercentile(allLatencies, 99)

	// Calculate average latency
	totalSum := uint64(0)
	totalCount := uint64(0)
	for _, metric := range pt.latencies {
		totalSum += atomic.LoadUint64(&metric.sum)
		totalCount += atomic.LoadUint64(&metric.count)
	}

	avgLatency := time.Duration(0)
	if totalCount > 0 {
		avgLatency = time.Duration(totalSum / totalCount)
	}

	// Collect throughputs
	throughputs := make(map[string]float64)
	for name, metric := range pt.throughputs {
		throughputs[name] = metric.value
	}

	return map[string]interface{}{
		"avg_latency": avgLatency,
		"latency_p50": p50,
		"latency_p95": p95,
		"latency_p99": p99,
		"throughputs": throughputs,
	}
}

// Export returns Prometheus format metrics
func (pt *PerformanceTracker) Export() string {
	metrics := pt.Collect()

	output := ""

	// Latency metrics
	output += "# HELP strigoi_latency_seconds Operation latency in seconds\n"
	output += "# TYPE strigoi_latency_seconds summary\n"

	if p50, ok := metrics["latency_p50"].(time.Duration); ok {
		output += fmt.Sprintf("strigoi_latency_seconds{quantile=\"0.5\"} %f\n", p50.Seconds())
	}
	if p95, ok := metrics["latency_p95"].(time.Duration); ok {
		output += fmt.Sprintf("strigoi_latency_seconds{quantile=\"0.95\"} %f\n", p95.Seconds())
	}
	if p99, ok := metrics["latency_p99"].(time.Duration); ok {
		output += fmt.Sprintf("strigoi_latency_seconds{quantile=\"0.99\"} %f\n", p99.Seconds())
	}

	// Throughput metrics
	if throughputs, ok := metrics["throughputs"].(map[string]float64); ok {
		for name, value := range throughputs {
			output += fmt.Sprintf("\n# HELP strigoi_%s Current %s\n", name, name)
			output += fmt.Sprintf("# TYPE strigoi_%s gauge\n", name)
			output += fmt.Sprintf("strigoi_%s %f\n", name, value)
		}
	}

	return output
}

// Reset resets performance metrics
func (pt *PerformanceTracker) Reset() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.latencies = make(map[string]*LatencyMetric)
	pt.throughputs = make(map[string]*ThroughputMetric)
}

// calculatePercentile calculates percentile from duration slice
func calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Sort durations
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)-1) * percentile / 100.0)
	return sorted[index]
}

// SystemCollector collects system metrics
type SystemCollector struct {
	lastCPU    time.Time
	lastMemory time.Time
}

// NewSystemCollector creates a system collector
func NewSystemCollector() *SystemCollector {
	return &SystemCollector{
		lastCPU:    time.Now(),
		lastMemory: time.Now(),
	}
}

// Collect returns system metrics
func (sc *SystemCollector) Collect() map[string]interface{} {
	// Simplified - in production use actual system metrics
	return map[string]interface{}{
		"cpu_usage":    0.45,                           // 45%
		"memory_usage": uint64(4 * 1024 * 1024 * 1024), // 4GB
		"disk_usage":   0.60,                           // 60%
		"network_rx":   uint64(1000000),                // 1MB/s
		"network_tx":   uint64(500000),                 // 500KB/s
		"open_files":   150,
		"goroutines":   42,
	}
}

// Export returns Prometheus format metrics
func (sc *SystemCollector) Export() string {
	metrics := sc.Collect()

	output := ""

	output += "# HELP strigoi_cpu_usage CPU usage percentage\n"
	output += "# TYPE strigoi_cpu_usage gauge\n"
	output += fmt.Sprintf("strigoi_cpu_usage %f\n\n", metrics["cpu_usage"])

	output += "# HELP strigoi_memory_usage_bytes Memory usage in bytes\n"
	output += "# TYPE strigoi_memory_usage_bytes gauge\n"
	output += fmt.Sprintf("strigoi_memory_usage_bytes %d\n\n", metrics["memory_usage"])

	output += "# HELP strigoi_goroutines Number of goroutines\n"
	output += "# TYPE strigoi_goroutines gauge\n"
	output += fmt.Sprintf("strigoi_goroutines %d\n", metrics["goroutines"])

	return output
}

// Reset is a no-op for system collector
func (sc *SystemCollector) Reset() {}

// ApplicationCollector collects application-specific metrics
type ApplicationCollector struct {
	counters map[string]uint64
	gauges   map[string]float64
	mu       sync.RWMutex
}

// NewApplicationCollector creates an application collector
func NewApplicationCollector() *ApplicationCollector {
	return &ApplicationCollector{
		counters: make(map[string]uint64),
		gauges:   make(map[string]float64),
	}
}

// IncrementCounter increments a counter
func (ac *ApplicationCollector) IncrementCounter(name string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.counters[name]++
}

// SetGauge sets a gauge value
func (ac *ApplicationCollector) SetGauge(name string, value float64) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.gauges[name] = value
}

// Collect returns application metrics
func (ac *ApplicationCollector) Collect() map[string]interface{} {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	counters := make(map[string]uint64)
	for k, v := range ac.counters {
		counters[k] = v
	}

	gauges := make(map[string]float64)
	for k, v := range ac.gauges {
		gauges[k] = v
	}

	return map[string]interface{}{
		"counters": counters,
		"gauges":   gauges,
	}
}

// Export returns Prometheus format metrics
func (ac *ApplicationCollector) Export() string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	output := ""

	// Export counters
	for name, value := range ac.counters {
		output += fmt.Sprintf("# HELP strigoi_%s Application counter %s\n", name, name)
		output += fmt.Sprintf("# TYPE strigoi_%s counter\n", name)
		output += fmt.Sprintf("strigoi_%s %d\n\n", name, value)
	}

	// Export gauges
	for name, value := range ac.gauges {
		output += fmt.Sprintf("# HELP strigoi_%s Application gauge %s\n", name, name)
		output += fmt.Sprintf("# TYPE strigoi_%s gauge\n", name)
		output += fmt.Sprintf("strigoi_%s %f\n\n", name, value)
	}

	return output
}

// Reset resets application metrics
func (ac *ApplicationCollector) Reset() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.counters = make(map[string]uint64)
	ac.gauges = make(map[string]float64)
}

// AlertManager manages alerts
type AlertManager struct {
	webhookURL      string
	thresholds      AlertThresholds
	activeAlerts    map[string]*Alert
	alertHistory    []Alert
	consecutiveFail map[string]int
	mu              sync.RWMutex
}

// Alert represents an alert
type Alert struct {
	ID        string
	Type      AlertType
	Severity  ErrorSeverity
	Message   string
	Value     float64
	Triggered time.Time
	Resolved  *time.Time
}

// AlertType defines alert types
type AlertType string

const (
	AlertTypeErrorRate AlertType = "error_rate"
	AlertTypeLatency   AlertType = "latency"
	AlertTypeMemory    AlertType = "memory"
	AlertTypeCPU       AlertType = "cpu"
	AlertTypeCustom    AlertType = "custom"
)

// NewAlertManager creates an alert manager
func NewAlertManager(webhookURL string, thresholds AlertThresholds) *AlertManager {
	return &AlertManager{
		webhookURL:      webhookURL,
		thresholds:      thresholds,
		activeAlerts:    make(map[string]*Alert),
		alertHistory:    make([]Alert, 0),
		consecutiveFail: make(map[string]int),
	}
}

// TriggerAlert triggers a new alert
func (am *AlertManager) TriggerAlert(alert Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert.ID = fmt.Sprintf("%s_%d", alert.Type, time.Now().Unix())
	alert.Triggered = time.Now()

	// Check if similar alert already active
	for _, active := range am.activeAlerts {
		if active.Type == alert.Type && active.Message == alert.Message {
			// Update existing alert
			active.Value = alert.Value
			return
		}
	}

	// New alert
	am.activeAlerts[alert.ID] = &alert
	am.alertHistory = append(am.alertHistory, alert)

	// Send webhook notification
	if am.webhookURL != "" {
		go am.sendWebhook(alert)
	}
}

// ResolveAlert resolves an alert
func (am *AlertManager) ResolveAlert(alertID string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if alert, exists := am.activeAlerts[alertID]; exists {
		now := time.Now()
		alert.Resolved = &now
		delete(am.activeAlerts, alertID)
	}
}

// GetActiveAlerts returns active alerts
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]Alert, 0, len(am.activeAlerts))
	for _, alert := range am.activeAlerts {
		alerts = append(alerts, *alert)
	}

	return alerts
}

// sendWebhook sends alert webhook
func (am *AlertManager) sendWebhook(alert Alert) {
	// Implement webhook sending logic
	payload := map[string]interface{}{
		"alert":     alert,
		"timestamp": time.Now(),
	}

	data, _ := json.Marshal(payload)

	// Send HTTP POST to webhook URL
	if _, err := http.Post(am.webhookURL, "application/json", bytes.NewReader(data)); err != nil {
		// Log webhook failure but don't block alert processing
		log.Printf("Failed to send webhook: %v", err)
	}
}

// HealthStatus represents system health
type HealthStatus struct {
	Healthy        bool
	ErrorRate      float64
	AverageLatency time.Duration
	ActiveAlerts   int
	LastCheck      time.Time
}
