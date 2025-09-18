package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	// "github.com/macawi-ai/strigoi/modules/probe/capture"
	// "github.com/macawi-ai/strigoi/modules/probe/dissect"
	"github.com/macawi-ai/strigoi/modules/probe/ml"
)

// Temporary mock types until capture and dissect packages are available
type captureEngine struct{}

func (c *captureEngine) CaptureWithContext(ctx context.Context, iface string, filter string) (<-chan []byte, error) {
	ch := make(chan []byte)
	close(ch)
	return ch, nil
}

func (c *captureEngine) Close() error {
	return nil
}

type dissectorEngine struct{}

func (d *dissectorEngine) Dissect(ctx context.Context, protocol string, data []byte) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

func (d *dissectorEngine) Close() error {
	return nil
}

// Worker represents a processing worker node
type Worker struct {
	config     WorkerConfig
	id         string
	engine     *captureEngine
	dissector  *dissectorEngine
	mlDetector *ml.PatternDetector
	httpServer *http.Server
	taskQueue  chan *ProcessingTask
	metrics    *WorkerMetrics
	status     WorkerHealthStatus
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// WorkerConfig configures a worker node
type WorkerConfig struct {
	ID             string
	ListenAddress  string
	QueueSize      int
	MaxConcurrent  int
	ProcessTimeout time.Duration
	EnableML       bool
	MLConfig       ml.DetectorConfig
	// CaptureConfig   capture.EngineConfig
	// DissectorConfig dissect.EngineConfig
}

// WorkerHealthStatus represents worker health
type WorkerHealthStatus struct {
	Healthy       bool
	LoadAverage   float64
	ErrorRate     float64
	ActiveTasks   int32
	QueueDepth    int
	MemoryUsage   uint64
	CPUUsage      float64
	LastHeartbeat time.Time
}

// NewWorker creates a new worker node
func NewWorker(config WorkerConfig) (*Worker, error) {
	ctx, cancel := context.WithCancel(context.Background())

	worker := &Worker{
		config:    config,
		id:        config.ID,
		taskQueue: make(chan *ProcessingTask, config.QueueSize),
		metrics:   NewWorkerMetrics(),
		status: WorkerHealthStatus{
			Healthy:       true,
			LastHeartbeat: time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize capture engine (mocked for now)
	// engine, err := capture.NewCaptureEngine(config.CaptureConfig)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create capture engine: %w", err)
	// }
	worker.engine = &captureEngine{}

	// Initialize dissector (mocked for now)
	// dissector, err := dissect.NewDissectorEngine(config.DissectorConfig)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create dissector: %w", err)
	// }
	worker.dissector = &dissectorEngine{}

	// Initialize ML detector if enabled
	if config.EnableML {
		detector, err := ml.NewPatternDetector(config.MLConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create ML detector: %w", err)
		}
		worker.mlDetector = detector
	}

	// Start HTTP server for task submission
	worker.setupHTTPServer()

	// Start processing workers
	for i := 0; i < config.MaxConcurrent; i++ {
		worker.wg.Add(1)
		go worker.processLoop()
	}

	// Start metrics updater
	worker.wg.Add(1)
	go worker.metricsLoop()

	return worker, nil
}

// setupHTTPServer sets up HTTP endpoints
func (w *Worker) setupHTTPServer() {
	mux := http.NewServeMux()

	// Task submission endpoint
	mux.HandleFunc("/task", w.handleTask)

	// Batch submission endpoint
	mux.HandleFunc("/batch", w.handleBatch)

	// Health check endpoint
	mux.HandleFunc("/health", w.handleHealth)

	// Metrics endpoint
	mux.HandleFunc("/metrics", w.handleMetrics)

	w.httpServer = &http.Server{
		Addr:         w.config.ListenAddress,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := w.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// TODO: Add proper error logging
			_ = err
		}
	}()
}

// handleTask handles single task submission
func (w *Worker) handleTask(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task ProcessingTask
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(rw, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Try to queue task
	select {
	case w.taskQueue <- &task:
		w.metrics.RecordTaskReceived()
		rw.WriteHeader(http.StatusAccepted)
		if err := json.NewEncoder(rw).Encode(map[string]string{"status": "queued"}); err != nil {
			http.Error(rw, "Failed to encode response", http.StatusInternalServerError)
		}
	default:
		w.metrics.RecordQueueFull()
		http.Error(rw, "Queue full", http.StatusServiceUnavailable)
	}
}

// handleBatch handles batch task submission
func (w *Worker) handleBatch(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var tasks []ProcessingTask
	if err := json.NewDecoder(r.Body).Decode(&tasks); err != nil {
		http.Error(rw, "Invalid request body", http.StatusBadRequest)
		return
	}

	results := make([]*ProcessingResult, 0, len(tasks))

	// Process batch with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	// Process tasks concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range tasks {
		wg.Add(1)
		go func(task *ProcessingTask) {
			defer wg.Done()

			result := w.processTask(ctx, task)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(&tasks[i])
	}

	// Wait for all tasks to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All tasks completed
		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(results); err != nil {
			http.Error(rw, "Failed to encode results", http.StatusInternalServerError)
		}
	case <-ctx.Done():
		// Timeout
		http.Error(rw, "Processing timeout", http.StatusRequestTimeout)
	}
}

// handleHealth handles health check requests
func (w *Worker) handleHealth(rw http.ResponseWriter, r *http.Request) {
	health := w.GetHealth()

	rw.Header().Set("Content-Type", "application/json")
	if health.Healthy {
		rw.WriteHeader(http.StatusOK)
	} else {
		rw.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(rw).Encode(health); err != nil {
		http.Error(rw, "Failed to encode health status", http.StatusInternalServerError)
	}
}

// handleMetrics handles metrics requests
func (w *Worker) handleMetrics(rw http.ResponseWriter, r *http.Request) {
	metrics := w.metrics.GetSnapshot()

	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(metrics); err != nil {
		http.Error(rw, "Failed to encode metrics", http.StatusInternalServerError)
	}
}

// processLoop processes tasks from the queue
func (w *Worker) processLoop() {
	defer w.wg.Done()

	for {
		select {
		case task := <-w.taskQueue:
			atomic.AddInt32(&w.status.ActiveTasks, 1)

			ctx, cancel := context.WithTimeout(w.ctx, w.config.ProcessTimeout)
			result := w.processTask(ctx, task)
			cancel()

			atomic.AddInt32(&w.status.ActiveTasks, -1)

			// Update metrics
			if result.Success {
				w.metrics.RecordTaskSuccess(result.ProcessingTime)
			} else {
				w.metrics.RecordTaskError()
			}

		case <-w.ctx.Done():
			return
		}
	}
}

// processTask processes a single task
func (w *Worker) processTask(ctx context.Context, task *ProcessingTask) *ProcessingResult {
	startTime := time.Now()

	result := &ProcessingResult{
		TaskID:      task.ID,
		CompletedAt: time.Now(),
	}

	// Decode task data based on type
	switch task.Type {
	case TaskTypeCapture:
		err := w.processCaptureTask(ctx, task)
		result.Success = err == nil
		result.Error = err

	case TaskTypeDissect:
		output, err := w.processDissectTask(ctx, task)
		result.Success = err == nil
		result.Error = err
		result.Data = output

	case TaskTypeAnalyze:
		output, err := w.processAnalyzeTask(ctx, task)
		result.Success = err == nil
		result.Error = err
		result.Data = output

	case TaskTypeAggregate:
		output, err := w.processAggregateTask(ctx, task)
		result.Success = err == nil
		result.Error = err
		result.Data = output

	default:
		result.Success = false
		result.Error = fmt.Errorf("unknown task type: %s", task.Type)
	}

	result.ProcessingTime = time.Since(startTime)

	return result
}

// processCaptureTask processes network capture tasks
func (w *Worker) processCaptureTask(ctx context.Context, task *ProcessingTask) error {
	// Parse capture parameters
	var params struct {
		Interface string        `json:"interface"`
		Filter    string        `json:"filter"`
		Duration  time.Duration `json:"duration"`
	}

	if err := json.Unmarshal(task.Data, &params); err != nil {
		return fmt.Errorf("invalid capture params: %w", err)
	}

	// Start capture
	captureCtx, cancel := context.WithTimeout(ctx, params.Duration)
	defer cancel()

	frames, err := w.engine.CaptureWithContext(captureCtx, params.Interface, params.Filter)
	if err != nil {
		return fmt.Errorf("capture failed: %w", err)
	}

	// Process captured frames
	for frame := range frames {
		// Send to dissector or storage
		_ = frame
	}

	return nil
}

// processDissectTask processes protocol dissection tasks
func (w *Worker) processDissectTask(ctx context.Context, task *ProcessingTask) ([]byte, error) {
	// Parse dissection parameters
	var params struct {
		Protocol string `json:"protocol"`
		Payload  []byte `json:"payload"`
	}

	if err := json.Unmarshal(task.Data, &params); err != nil {
		return nil, fmt.Errorf("invalid dissect params: %w", err)
	}

	// Perform dissection
	result, err := w.dissector.Dissect(ctx, params.Protocol, params.Payload)
	if err != nil {
		return nil, fmt.Errorf("dissection failed: %w", err)
	}

	// Serialize result
	output, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return output, nil
}

// processAnalyzeTask processes ML analysis tasks
func (w *Worker) processAnalyzeTask(ctx context.Context, task *ProcessingTask) ([]byte, error) {
	if w.mlDetector == nil {
		return nil, fmt.Errorf("ML detector not enabled")
	}

	// Parse analysis parameters
	var event ml.SecurityEvent
	if err := json.Unmarshal(task.Data, &event); err != nil {
		return nil, fmt.Errorf("invalid event data: %w", err)
	}

	// Perform ML analysis
	result, err := w.mlDetector.Analyze(ctx, &event)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Serialize result
	output, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return output, nil
}

// processAggregateTask processes aggregation tasks
func (w *Worker) processAggregateTask(ctx context.Context, task *ProcessingTask) ([]byte, error) {
	// Parse aggregation parameters
	var params struct {
		Window  time.Duration     `json:"window"`
		GroupBy []string          `json:"group_by"`
		Metrics []string          `json:"metrics"`
		Events  []json.RawMessage `json:"events"`
	}

	if err := json.Unmarshal(task.Data, &params); err != nil {
		return nil, fmt.Errorf("invalid aggregate params: %w", err)
	}

	// Perform aggregation
	aggregates := make(map[string]interface{})

	// Simple count aggregation example
	aggregates["total_events"] = len(params.Events)
	aggregates["window"] = params.Window.String()
	aggregates["timestamp"] = time.Now()

	// Serialize result
	output, err := json.Marshal(aggregates)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return output, nil
}

// metricsLoop updates worker metrics
func (w *Worker) metricsLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.updateMetrics()

		case <-w.ctx.Done():
			return
		}
	}
}

// updateMetrics updates worker health metrics
func (w *Worker) updateMetrics() {
	// Update queue depth
	w.status.QueueDepth = len(w.taskQueue)

	// Update load average
	activeTasks := atomic.LoadInt32(&w.status.ActiveTasks)
	w.status.LoadAverage = float64(activeTasks) / float64(w.config.MaxConcurrent)

	// Update error rate
	totalProcessed := w.metrics.GetTotalProcessed()
	totalErrors := w.metrics.GetTotalErrors()
	if totalProcessed > 0 {
		w.status.ErrorRate = float64(totalErrors) / float64(totalProcessed)
	}

	// Update heartbeat
	w.status.LastHeartbeat = time.Now()

	// Check health
	w.status.Healthy = w.status.LoadAverage < 0.9 && w.status.ErrorRate < 0.1
}

// GetHealth returns current health status
func (w *Worker) GetHealth() WorkerHealthStatus {
	status := w.status
	status.ActiveTasks = atomic.LoadInt32(&w.status.ActiveTasks)
	status.QueueDepth = len(w.taskQueue)
	return status
}

// Shutdown gracefully shuts down the worker
func (w *Worker) Shutdown(timeout time.Duration) error {
	// Stop accepting new tasks
	w.cancel()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := w.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP server shutdown failed: %w", err)
	}

	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Clean shutdown
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout exceeded")
	}

	// Cleanup resources
	if w.engine != nil {
		w.engine.Close()
	}
	if w.dissector != nil {
		w.dissector.Close()
	}
	if w.mlDetector != nil {
		w.mlDetector.Close()
	}

	return nil
}

// WorkerMetrics tracks worker performance
type WorkerMetrics struct {
	tasksReceived  uint64
	tasksProcessed uint64
	tasksSucceeded uint64
	tasksFailed    uint64
	queueFullCount uint64
	totalTime      uint64
	mu             sync.RWMutex
}

// NewWorkerMetrics creates new metrics tracker
func NewWorkerMetrics() *WorkerMetrics {
	return &WorkerMetrics{}
}

// RecordTaskReceived records task receipt
func (m *WorkerMetrics) RecordTaskReceived() {
	atomic.AddUint64(&m.tasksReceived, 1)
}

// RecordTaskSuccess records successful task
func (m *WorkerMetrics) RecordTaskSuccess(duration time.Duration) {
	atomic.AddUint64(&m.tasksProcessed, 1)
	atomic.AddUint64(&m.tasksSucceeded, 1)
	atomic.AddUint64(&m.totalTime, uint64(duration.Milliseconds()))
}

// RecordTaskError records failed task
func (m *WorkerMetrics) RecordTaskError() {
	atomic.AddUint64(&m.tasksProcessed, 1)
	atomic.AddUint64(&m.tasksFailed, 1)
}

// RecordQueueFull records queue full event
func (m *WorkerMetrics) RecordQueueFull() {
	atomic.AddUint64(&m.queueFullCount, 1)
}

// GetTotalProcessed returns total processed
func (m *WorkerMetrics) GetTotalProcessed() uint64 {
	return atomic.LoadUint64(&m.tasksProcessed)
}

// GetTotalErrors returns total errors
func (m *WorkerMetrics) GetTotalErrors() uint64 {
	return atomic.LoadUint64(&m.tasksFailed)
}

// GetSnapshot returns metrics snapshot
func (m *WorkerMetrics) GetSnapshot() map[string]interface{} {
	avgTime := float64(0)
	processed := atomic.LoadUint64(&m.tasksProcessed)
	if processed > 0 {
		avgTime = float64(atomic.LoadUint64(&m.totalTime)) / float64(processed)
	}

	return map[string]interface{}{
		"tasks_received":   atomic.LoadUint64(&m.tasksReceived),
		"tasks_processed":  processed,
		"tasks_succeeded":  atomic.LoadUint64(&m.tasksSucceeded),
		"tasks_failed":     atomic.LoadUint64(&m.tasksFailed),
		"queue_full_count": atomic.LoadUint64(&m.queueFullCount),
		"avg_process_time": avgTime,
	}
}
