package distributed

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CoordinatorMetrics tracks coordinator performance
type CoordinatorMetrics struct {
	// Task metrics
	tasksSubmitted uint64
	tasksCompleted uint64
	tasksFailed    uint64
	tasksRetried   uint64

	// Node metrics
	nodesRegistered   uint64
	nodesUnregistered uint64
	nodeErrors        map[string]uint64
	nodeErrorsMu      sync.RWMutex

	// Performance metrics
	totalProcessingTime uint64 // in microseconds
	healthCheckFailures uint64

	// Aggregation metrics
	aggregationResults map[string]*AggregateMetric
	aggregationMu      sync.RWMutex
}

// AggregateMetric represents aggregated metrics
type AggregateMetric struct {
	Count      uint64
	Sum        float64
	Min        float64
	Max        float64
	LastUpdate time.Time
}

// NewCoordinatorMetrics creates new metrics tracker
func NewCoordinatorMetrics() *CoordinatorMetrics {
	return &CoordinatorMetrics{
		nodeErrors:         make(map[string]uint64),
		aggregationResults: make(map[string]*AggregateMetric),
	}
}

// RecordTaskSubmitted records task submission
func (m *CoordinatorMetrics) RecordTaskSubmitted(taskType TaskType) {
	atomic.AddUint64(&m.tasksSubmitted, 1)
}

// RecordTaskCompleted records task completion
func (m *CoordinatorMetrics) RecordTaskCompleted(success bool, duration time.Duration) {
	if success {
		atomic.AddUint64(&m.tasksCompleted, 1)
	} else {
		atomic.AddUint64(&m.tasksFailed, 1)
	}
	atomic.AddUint64(&m.totalProcessingTime, uint64(duration.Microseconds()))
}

// RecordTaskRetry records task retry
func (m *CoordinatorMetrics) RecordTaskRetry() {
	atomic.AddUint64(&m.tasksRetried, 1)
}

// RecordNodeRegistered records node registration
func (m *CoordinatorMetrics) RecordNodeRegistered(nodeID string) {
	atomic.AddUint64(&m.nodesRegistered, 1)
}

// RecordNodeUnregistered records node removal
func (m *CoordinatorMetrics) RecordNodeUnregistered(nodeID string) {
	atomic.AddUint64(&m.nodesUnregistered, 1)
}

// RecordNodeError records node error
func (m *CoordinatorMetrics) RecordNodeError(nodeID string) {
	m.nodeErrorsMu.Lock()
	defer m.nodeErrorsMu.Unlock()

	if _, exists := m.nodeErrors[nodeID]; !exists {
		m.nodeErrors[nodeID] = 0
	}
	m.nodeErrors[nodeID]++
}

// RecordHealthCheck records health check result
func (m *CoordinatorMetrics) RecordHealthCheck(nodeID string, status NodeStatus) {
	if status == NodeStatusOffline || status == NodeStatusUnhealthy {
		atomic.AddUint64(&m.healthCheckFailures, 1)
	}
}

// RecordHealthCheckFailure records health check failure
func (m *CoordinatorMetrics) RecordHealthCheckFailure(nodeID string) {
	atomic.AddUint64(&m.healthCheckFailures, 1)
	m.RecordNodeError(nodeID)
}

// UpdateAggregates updates aggregated metrics
func (m *CoordinatorMetrics) UpdateAggregates(summary *ResultSummary) {
	m.aggregationMu.Lock()
	defer m.aggregationMu.Unlock()

	for key, value := range summary.Metrics {
		metric, exists := m.aggregationResults[key]
		if !exists {
			metric = &AggregateMetric{
				Min: value,
				Max: value,
			}
			m.aggregationResults[key] = metric
		}

		metric.Count++
		metric.Sum += value
		if value < metric.Min {
			metric.Min = value
		}
		if value > metric.Max {
			metric.Max = value
		}
		metric.LastUpdate = time.Now()
	}
}

// GetSnapshot returns metrics snapshot
func (m *CoordinatorMetrics) GetSnapshot() MetricsSnapshot {
	m.nodeErrorsMu.RLock()
	nodeErrorsCopy := make(map[string]uint64)
	for k, v := range m.nodeErrors {
		nodeErrorsCopy[k] = v
	}
	m.nodeErrorsMu.RUnlock()

	m.aggregationMu.RLock()
	aggregatesCopy := make(map[string]AggregateSnapshot)
	for k, v := range m.aggregationResults {
		avg := float64(0)
		if v.Count > 0 {
			avg = v.Sum / float64(v.Count)
		}
		aggregatesCopy[k] = AggregateSnapshot{
			Count:   v.Count,
			Average: avg,
			Min:     v.Min,
			Max:     v.Max,
		}
	}
	m.aggregationMu.RUnlock()

	completed := atomic.LoadUint64(&m.tasksCompleted)
	failed := atomic.LoadUint64(&m.tasksFailed)
	total := completed + failed

	successRate := float64(0)
	if total > 0 {
		successRate = float64(completed) / float64(total)
	}

	avgProcessingTime := float64(0)
	if completed > 0 {
		avgProcessingTime = float64(atomic.LoadUint64(&m.totalProcessingTime)) / float64(completed) / 1000.0 // Convert to ms
	}

	return MetricsSnapshot{
		TasksSubmitted:      atomic.LoadUint64(&m.tasksSubmitted),
		TasksCompleted:      completed,
		TasksFailed:         failed,
		TasksRetried:        atomic.LoadUint64(&m.tasksRetried),
		SuccessRate:         successRate,
		AvgProcessingTime:   avgProcessingTime,
		NodesRegistered:     atomic.LoadUint64(&m.nodesRegistered),
		NodesUnregistered:   atomic.LoadUint64(&m.nodesUnregistered),
		NodeErrors:          nodeErrorsCopy,
		HealthCheckFailures: atomic.LoadUint64(&m.healthCheckFailures),
		Aggregates:          aggregatesCopy,
		Timestamp:           time.Now(),
	}
}

// MetricsSnapshot represents a point-in-time metrics snapshot
type MetricsSnapshot struct {
	// Task metrics
	TasksSubmitted    uint64
	TasksCompleted    uint64
	TasksFailed       uint64
	TasksRetried      uint64
	SuccessRate       float64
	AvgProcessingTime float64 // in milliseconds

	// Node metrics
	NodesRegistered     uint64
	NodesUnregistered   uint64
	NodeErrors          map[string]uint64
	HealthCheckFailures uint64

	// Aggregates
	Aggregates map[string]AggregateSnapshot

	// Metadata
	Timestamp time.Time
}

// AggregateSnapshot represents aggregated metric snapshot
type AggregateSnapshot struct {
	Count   uint64
	Average float64
	Min     float64
	Max     float64
}

// ResultAggregator aggregates processing results
type ResultAggregator struct {
	results  []*ProcessingResult
	capacity int
	position int
	mu       sync.Mutex
}

// NewResultAggregator creates a result aggregator
func NewResultAggregator(capacity int) *ResultAggregator {
	return &ResultAggregator{
		results:  make([]*ProcessingResult, capacity),
		capacity: capacity,
	}
}

// Add adds a result to the aggregator
func (a *ResultAggregator) Add(result *ProcessingResult) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.results[a.position] = result
	a.position = (a.position + 1) % a.capacity
}

// GetSummary returns aggregated summary
func (a *ResultAggregator) GetSummary() *ResultSummary {
	a.mu.Lock()
	defer a.mu.Unlock()

	summary := &ResultSummary{
		TotalResults: 0,
		SuccessCount: 0,
		FailureCount: 0,
		Metrics:      make(map[string]float64),
		Timestamp:    time.Now(),
	}

	totalTime := time.Duration(0)

	for _, result := range a.results {
		if result == nil {
			continue
		}

		summary.TotalResults++

		if result.Success {
			summary.SuccessCount++
		} else {
			summary.FailureCount++
		}

		totalTime += result.ProcessingTime
	}

	if summary.TotalResults > 0 {
		summary.Metrics["avg_processing_time"] = float64(totalTime.Milliseconds()) / float64(summary.TotalResults)
		summary.Metrics["success_rate"] = float64(summary.SuccessCount) / float64(summary.TotalResults)
	}

	return summary
}

// ResultSummary represents aggregated result summary
type ResultSummary struct {
	TotalResults int
	SuccessCount int
	FailureCount int
	Metrics      map[string]float64
	Timestamp    time.Time
}

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	metrics *CoordinatorMetrics
}

// NewPrometheusExporter creates a Prometheus exporter
func NewPrometheusExporter(metrics *CoordinatorMetrics) *PrometheusExporter {
	return &PrometheusExporter{
		metrics: metrics,
	}
}

// Export returns metrics in Prometheus format
func (e *PrometheusExporter) Export() string {
	snapshot := e.metrics.GetSnapshot()

	var output string

	// Task metrics
	output += fmt.Sprintf("# HELP strigoi_tasks_submitted_total Total tasks submitted\n")
	output += fmt.Sprintf("# TYPE strigoi_tasks_submitted_total counter\n")
	output += fmt.Sprintf("strigoi_tasks_submitted_total %d\n\n", snapshot.TasksSubmitted)

	output += fmt.Sprintf("# HELP strigoi_tasks_completed_total Total tasks completed\n")
	output += fmt.Sprintf("# TYPE strigoi_tasks_completed_total counter\n")
	output += fmt.Sprintf("strigoi_tasks_completed_total %d\n\n", snapshot.TasksCompleted)

	output += fmt.Sprintf("# HELP strigoi_tasks_failed_total Total tasks failed\n")
	output += fmt.Sprintf("# TYPE strigoi_tasks_failed_total counter\n")
	output += fmt.Sprintf("strigoi_tasks_failed_total %d\n\n", snapshot.TasksFailed)

	output += fmt.Sprintf("# HELP strigoi_tasks_retried_total Total tasks retried\n")
	output += fmt.Sprintf("# TYPE strigoi_tasks_retried_total counter\n")
	output += fmt.Sprintf("strigoi_tasks_retried_total %d\n\n", snapshot.TasksRetried)

	output += fmt.Sprintf("# HELP strigoi_task_success_rate Task success rate\n")
	output += fmt.Sprintf("# TYPE strigoi_task_success_rate gauge\n")
	output += fmt.Sprintf("strigoi_task_success_rate %f\n\n", snapshot.SuccessRate)

	output += fmt.Sprintf("# HELP strigoi_task_processing_time_ms Average task processing time in milliseconds\n")
	output += fmt.Sprintf("# TYPE strigoi_task_processing_time_ms gauge\n")
	output += fmt.Sprintf("strigoi_task_processing_time_ms %f\n\n", snapshot.AvgProcessingTime)

	// Node metrics
	output += fmt.Sprintf("# HELP strigoi_nodes_registered_total Total nodes registered\n")
	output += fmt.Sprintf("# TYPE strigoi_nodes_registered_total counter\n")
	output += fmt.Sprintf("strigoi_nodes_registered_total %d\n\n", snapshot.NodesRegistered)

	output += fmt.Sprintf("# HELP strigoi_nodes_unregistered_total Total nodes unregistered\n")
	output += fmt.Sprintf("# TYPE strigoi_nodes_unregistered_total counter\n")
	output += fmt.Sprintf("strigoi_nodes_unregistered_total %d\n\n", snapshot.NodesUnregistered)

	output += fmt.Sprintf("# HELP strigoi_health_check_failures_total Total health check failures\n")
	output += fmt.Sprintf("# TYPE strigoi_health_check_failures_total counter\n")
	output += fmt.Sprintf("strigoi_health_check_failures_total %d\n\n", snapshot.HealthCheckFailures)

	// Node errors by node
	if len(snapshot.NodeErrors) > 0 {
		output += fmt.Sprintf("# HELP strigoi_node_errors_total Total errors by node\n")
		output += fmt.Sprintf("# TYPE strigoi_node_errors_total counter\n")
		for nodeID, count := range snapshot.NodeErrors {
			output += fmt.Sprintf("strigoi_node_errors_total{node=\"%s\"} %d\n", nodeID, count)
		}
		output += "\n"
	}

	// Aggregated metrics
	for name, agg := range snapshot.Aggregates {
		metricName := fmt.Sprintf("strigoi_aggregate_%s", name)

		output += fmt.Sprintf("# HELP %s_count Count of %s\n", metricName, name)
		output += fmt.Sprintf("# TYPE %s_count counter\n", metricName)
		output += fmt.Sprintf("%s_count %d\n", metricName, agg.Count)

		output += fmt.Sprintf("# HELP %s_avg Average of %s\n", metricName, name)
		output += fmt.Sprintf("# TYPE %s_avg gauge\n", metricName)
		output += fmt.Sprintf("%s_avg %f\n", metricName, agg.Average)

		output += fmt.Sprintf("# HELP %s_min Minimum of %s\n", metricName, name)
		output += fmt.Sprintf("# TYPE %s_min gauge\n", metricName)
		output += fmt.Sprintf("%s_min %f\n", metricName, agg.Min)

		output += fmt.Sprintf("# HELP %s_max Maximum of %s\n", metricName, name)
		output += fmt.Sprintf("# TYPE %s_max gauge\n", metricName)
		output += fmt.Sprintf("%s_max %f\n\n", metricName, agg.Max)
	}

	return output
}

// GrafanaDashboard generates a Grafana dashboard JSON
func GenerateGrafanaDashboard() string {
	dashboard := `{
  "dashboard": {
    "title": "Strigoi Distributed Processing",
    "panels": [
      {
        "title": "Task Processing Rate",
        "targets": [
          {"expr": "rate(strigoi_tasks_completed_total[5m])"},
          {"expr": "rate(strigoi_tasks_failed_total[5m])"}
        ],
        "type": "graph"
      },
      {
        "title": "Task Success Rate",
        "targets": [
          {"expr": "strigoi_task_success_rate"}
        ],
        "type": "gauge"
      },
      {
        "title": "Average Processing Time",
        "targets": [
          {"expr": "strigoi_task_processing_time_ms"}
        ],
        "type": "graph"
      },
      {
        "title": "Node Health",
        "targets": [
          {"expr": "strigoi_nodes_registered_total - strigoi_nodes_unregistered_total"}
        ],
        "type": "stat"
      },
      {
        "title": "Node Errors",
        "targets": [
          {"expr": "rate(strigoi_node_errors_total[5m])"}
        ],
        "type": "graph"
      },
      {
        "title": "Health Check Failures",
        "targets": [
          {"expr": "rate(strigoi_health_check_failures_total[5m])"}
        ],
        "type": "graph"
      }
    ]
  }
}`
	return dashboard
}
