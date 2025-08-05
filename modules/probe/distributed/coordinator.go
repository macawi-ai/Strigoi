package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	// "github.com/macawi-ai/strigoi/modules/probe/types"
)

// Coordinator manages distributed processing of security events
type Coordinator struct {
	config       CoordinatorConfig
	nodes        map[string]*WorkerNode
	nodesMu      sync.RWMutex
	partitioner  Partitioner
	loadBalancer LoadBalancer
	eventQueue   chan *ProcessingTask
	resultQueue  chan *ProcessingResult
	metrics      *CoordinatorMetrics
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// CoordinatorConfig configures the distributed coordinator
type CoordinatorConfig struct {
	// Node management
	MinNodes            int
	MaxNodes            int
	NodeTimeout         time.Duration
	HealthCheckInterval time.Duration

	// Processing configuration
	QueueSize    int
	BatchSize    int
	BatchTimeout time.Duration
	MaxRetries   int

	// Partitioning strategy
	PartitionStrategy string // "hash", "round-robin", "weighted", "consistent-hash"
	ReplicationFactor int

	// Load balancing
	LoadBalanceStrategy string // "least-loaded", "round-robin", "weighted-response-time"

	// Resilience
	EnableFailover     bool
	FailoverTimeout    time.Duration
	EnableCheckpoint   bool
	CheckpointInterval time.Duration
}

// WorkerNode represents a processing node
type WorkerNode struct {
	ID              string
	Address         string
	Status          NodeStatus
	Capacity        int
	ActiveTasks     int32
	ProcessedCount  int64
	ErrorCount      int64
	LastHealthCheck time.Time
	ResponseTimes   *ResponseTimeTracker
	client          WorkerClient
	mu              sync.RWMutex
}

// NodeStatus represents node health status
type NodeStatus int

const (
	NodeStatusUnknown NodeStatus = iota
	NodeStatusHealthy
	NodeStatusDegraded
	NodeStatusUnhealthy
	NodeStatusOffline
)

// ProcessingTask represents a unit of work
type ProcessingTask struct {
	ID           string
	Type         TaskType
	Priority     int
	Data         []byte
	Metadata     map[string]string
	CreatedAt    time.Time
	Deadline     time.Time
	RetryCount   int
	PartitionKey string
}

// TaskType defines the type of processing task
type TaskType string

const (
	TaskTypeCapture   TaskType = "capture"
	TaskTypeDissect   TaskType = "dissect"
	TaskTypeAnalyze   TaskType = "analyze"
	TaskTypeAggregate TaskType = "aggregate"
)

// ProcessingResult represents task completion
type ProcessingResult struct {
	TaskID         string
	NodeID         string
	Success        bool
	Error          error
	Data           []byte
	ProcessingTime time.Duration
	CompletedAt    time.Time
}

// NewCoordinator creates a new distributed coordinator
func NewCoordinator(config CoordinatorConfig) (*Coordinator, error) {
	ctx, cancel := context.WithCancel(context.Background())

	coord := &Coordinator{
		config:      config,
		nodes:       make(map[string]*WorkerNode),
		eventQueue:  make(chan *ProcessingTask, config.QueueSize),
		resultQueue: make(chan *ProcessingResult, config.QueueSize),
		metrics:     NewCoordinatorMetrics(),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Initialize partitioner
	switch config.PartitionStrategy {
	case "hash":
		coord.partitioner = NewHashPartitioner()
	case "consistent-hash":
		coord.partitioner = NewConsistentHashPartitioner(config.ReplicationFactor)
	case "round-robin":
		coord.partitioner = NewRoundRobinPartitioner()
	default:
		coord.partitioner = NewHashPartitioner()
	}

	// Initialize load balancer
	switch config.LoadBalanceStrategy {
	case "least-loaded":
		coord.loadBalancer = NewLeastLoadedBalancer()
	case "weighted-response-time":
		coord.loadBalancer = NewWeightedResponseTimeBalancer()
	case "round-robin":
		coord.loadBalancer = NewRoundRobinBalancer()
	default:
		coord.loadBalancer = NewLeastLoadedBalancer()
	}

	// Start background workers
	coord.wg.Add(3)
	go coord.processLoop()
	go coord.resultLoop()
	go coord.healthCheckLoop()

	return coord, nil
}

// RegisterNode adds a new worker node
func (c *Coordinator) RegisterNode(id, address string, capacity int) error {
	c.nodesMu.Lock()
	defer c.nodesMu.Unlock()

	if _, exists := c.nodes[id]; exists {
		return fmt.Errorf("node %s already registered", id)
	}

	client, err := NewWorkerClient(address)
	if err != nil {
		return fmt.Errorf("failed to create client for node %s: %w", id, err)
	}

	node := &WorkerNode{
		ID:              id,
		Address:         address,
		Status:          NodeStatusUnknown,
		Capacity:        capacity,
		ActiveTasks:     0,
		LastHealthCheck: time.Now(),
		ResponseTimes:   NewResponseTimeTracker(100),
		client:          client,
	}

	c.nodes[id] = node

	// Update partitioner
	if ch, ok := c.partitioner.(*ConsistentHashPartitioner); ok {
		ch.AddNode(id, capacity)
	}

	c.metrics.RecordNodeRegistered(id)

	return nil
}

// UnregisterNode removes a worker node
func (c *Coordinator) UnregisterNode(id string) error {
	c.nodesMu.Lock()
	defer c.nodesMu.Unlock()

	node, exists := c.nodes[id]
	if !exists {
		return fmt.Errorf("node %s not found", id)
	}

	// Wait for active tasks to complete
	timeout := time.NewTimer(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for atomic.LoadInt32(&node.ActiveTasks) > 0 {
		select {
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for node %s to complete tasks", id)
		case <-ticker.C:
			continue
		}
	}

	// Close client connection
	node.client.Close()

	// Remove from coordinator
	delete(c.nodes, id)

	// Update partitioner
	if ch, ok := c.partitioner.(*ConsistentHashPartitioner); ok {
		ch.RemoveNode(id)
	}

	c.metrics.RecordNodeUnregistered(id)

	return nil
}

// Submit submits a task for processing
func (c *Coordinator) Submit(task *ProcessingTask) error {
	select {
	case c.eventQueue <- task:
		c.metrics.RecordTaskSubmitted(task.Type)
		return nil
	case <-c.ctx.Done():
		return fmt.Errorf("coordinator shutting down")
	default:
		return fmt.Errorf("task queue full")
	}
}

// SubmitBatch submits multiple tasks
func (c *Coordinator) SubmitBatch(tasks []*ProcessingTask) error {
	for _, task := range tasks {
		if err := c.Submit(task); err != nil {
			return fmt.Errorf("failed to submit task %s: %w", task.ID, err)
		}
	}
	return nil
}

// GetResult retrieves a processing result
func (c *Coordinator) GetResult(timeout time.Duration) (*ProcessingResult, error) {
	select {
	case result := <-c.resultQueue:
		return result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for result")
	case <-c.ctx.Done():
		return nil, fmt.Errorf("coordinator shutting down")
	}
}

// processLoop handles task distribution
func (c *Coordinator) processLoop() {
	defer c.wg.Done()

	batch := make([]*ProcessingTask, 0, c.config.BatchSize)
	batchTimer := time.NewTimer(c.config.BatchTimeout)
	defer batchTimer.Stop()

	for {
		select {
		case task := <-c.eventQueue:
			batch = append(batch, task)

			if len(batch) >= c.config.BatchSize {
				c.processBatch(batch)
				batch = batch[:0]
				batchTimer.Reset(c.config.BatchTimeout)
			}

		case <-batchTimer.C:
			if len(batch) > 0 {
				c.processBatch(batch)
				batch = batch[:0]
			}
			batchTimer.Reset(c.config.BatchTimeout)

		case <-c.ctx.Done():
			// Process remaining batch
			if len(batch) > 0 {
				c.processBatch(batch)
			}
			return
		}
	}
}

// processBatch distributes a batch of tasks to nodes
func (c *Coordinator) processBatch(tasks []*ProcessingTask) {
	// Group tasks by partition
	partitions := c.partitionTasks(tasks)

	// Process each partition
	for nodeID, partitionTasks := range partitions {
		go c.processPartition(nodeID, partitionTasks)
	}
}

// partitionTasks groups tasks by target node
func (c *Coordinator) partitionTasks(tasks []*ProcessingTask) map[string][]*ProcessingTask {
	partitions := make(map[string][]*ProcessingTask)

	c.nodesMu.RLock()
	defer c.nodesMu.RUnlock()

	for _, task := range tasks {
		// Get available nodes
		availableNodes := c.getHealthyNodes()
		if len(availableNodes) == 0 {
			// No healthy nodes, queue for retry
			go func(t *ProcessingTask) {
				time.Sleep(time.Second)
				c.Submit(t)
			}(task)
			continue
		}

		// Select node based on partitioning strategy
		nodeID := c.partitioner.GetNode(task.PartitionKey, availableNodes)

		// Apply load balancing
		nodeID = c.loadBalancer.SelectNode(nodeID, availableNodes)

		partitions[nodeID] = append(partitions[nodeID], task)
	}

	return partitions
}

// processPartition sends tasks to a specific node
func (c *Coordinator) processPartition(nodeID string, tasks []*ProcessingTask) {
	c.nodesMu.RLock()
	node, exists := c.nodes[nodeID]
	c.nodesMu.RUnlock()

	if !exists {
		// Node disappeared, resubmit tasks
		for _, task := range tasks {
			c.Submit(task)
		}
		return
	}

	// Update active task count
	atomic.AddInt32(&node.ActiveTasks, int32(len(tasks)))
	defer atomic.AddInt32(&node.ActiveTasks, -int32(len(tasks)))

	// Send tasks to node
	startTime := time.Now()
	results, err := node.client.ProcessBatch(c.ctx, tasks)
	processingTime := time.Since(startTime)

	if err != nil {
		// Handle failure
		c.handleNodeFailure(nodeID, tasks, err)
		return
	}

	// Update metrics
	node.ResponseTimes.Record(processingTime)
	atomic.AddInt64(&node.ProcessedCount, int64(len(results)))

	// Queue results
	for _, result := range results {
		result.NodeID = nodeID
		select {
		case c.resultQueue <- result:
			c.metrics.RecordTaskCompleted(result.Success, processingTime)
		case <-c.ctx.Done():
			return
		}
	}
}

// handleNodeFailure handles node processing failures
func (c *Coordinator) handleNodeFailure(nodeID string, tasks []*ProcessingTask, err error) {
	c.nodesMu.Lock()
	node, exists := c.nodes[nodeID]
	if exists {
		atomic.AddInt64(&node.ErrorCount, 1)
		node.Status = NodeStatusDegraded

		// Check if node should be marked unhealthy
		if atomic.LoadInt64(&node.ErrorCount) > 10 {
			node.Status = NodeStatusUnhealthy
		}
	}
	c.nodesMu.Unlock()

	c.metrics.RecordNodeError(nodeID)

	// Retry tasks on different nodes
	for _, task := range tasks {
		task.RetryCount++
		if task.RetryCount < c.config.MaxRetries {
			// Resubmit with delay
			go func(t *ProcessingTask) {
				time.Sleep(time.Duration(t.RetryCount) * time.Second)
				c.Submit(t)
			}(task)
		} else {
			// Max retries exceeded, send failure result
			result := &ProcessingResult{
				TaskID:      task.ID,
				NodeID:      nodeID,
				Success:     false,
				Error:       fmt.Errorf("max retries exceeded: %w", err),
				CompletedAt: time.Now(),
			}

			select {
			case c.resultQueue <- result:
			case <-c.ctx.Done():
			}
		}
	}
}

// resultLoop processes results
func (c *Coordinator) resultLoop() {
	defer c.wg.Done()

	// Result aggregation for analytics
	aggregator := NewResultAggregator(c.config.BatchSize)
	aggregateTicker := time.NewTicker(time.Second)
	defer aggregateTicker.Stop()

	for {
		select {
		case result := <-c.resultQueue:
			aggregator.Add(result)

		case <-aggregateTicker.C:
			if summary := aggregator.GetSummary(); summary != nil {
				c.metrics.UpdateAggregates(summary)
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// healthCheckLoop monitors node health
func (c *Coordinator) healthCheckLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.performHealthChecks()

		case <-c.ctx.Done():
			return
		}
	}
}

// performHealthChecks checks all nodes
func (c *Coordinator) performHealthChecks() {
	c.nodesMu.RLock()
	nodes := make([]*WorkerNode, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	c.nodesMu.RUnlock()

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(n *WorkerNode) {
			defer wg.Done()
			c.checkNodeHealth(n)
		}(node)
	}

	wg.Wait()
}

// checkNodeHealth checks a single node
func (c *Coordinator) checkNodeHealth(node *WorkerNode) {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	health, err := node.client.HealthCheck(ctx)

	node.mu.Lock()
	defer node.mu.Unlock()

	node.LastHealthCheck = time.Now()

	if err != nil {
		// Health check failed
		if node.Status == NodeStatusHealthy {
			node.Status = NodeStatusDegraded
		} else if time.Since(node.LastHealthCheck) > c.config.NodeTimeout {
			node.Status = NodeStatusOffline
		}
		c.metrics.RecordHealthCheckFailure(node.ID)
		return
	}

	// Update status based on health
	if health.LoadAverage > 0.9 {
		node.Status = NodeStatusDegraded
	} else if health.ErrorRate > 0.1 {
		node.Status = NodeStatusDegraded
	} else {
		node.Status = NodeStatusHealthy
	}

	c.metrics.RecordHealthCheck(node.ID, node.Status)
}

// getHealthyNodes returns nodes that can accept work
func (c *Coordinator) getHealthyNodes() []*WorkerNode {
	var healthy []*WorkerNode

	for _, node := range c.nodes {
		if node.Status == NodeStatusHealthy || node.Status == NodeStatusDegraded {
			if atomic.LoadInt32(&node.ActiveTasks) < int32(node.Capacity) {
				healthy = append(healthy, node)
			}
		}
	}

	return healthy
}

// GetStatus returns coordinator status
func (c *Coordinator) GetStatus() CoordinatorStatus {
	c.nodesMu.RLock()
	defer c.nodesMu.RUnlock()

	nodeStatuses := make(map[string]NodeStatusInfo)
	totalCapacity := 0
	activeTasksTotal := int32(0)

	for id, node := range c.nodes {
		node.mu.RLock()
		activeTasks := atomic.LoadInt32(&node.ActiveTasks)
		nodeStatuses[id] = NodeStatusInfo{
			ID:             id,
			Status:         node.Status,
			ActiveTasks:    int(activeTasks),
			Capacity:       node.Capacity,
			ProcessedCount: atomic.LoadInt64(&node.ProcessedCount),
			ErrorCount:     atomic.LoadInt64(&node.ErrorCount),
			ResponseTime:   node.ResponseTimes.GetAverage(),
		}
		node.mu.RUnlock()

		totalCapacity += node.Capacity
		activeTasksTotal += activeTasks
	}

	queueLen := len(c.eventQueue)

	return CoordinatorStatus{
		Nodes:         nodeStatuses,
		QueueLength:   queueLen,
		TotalCapacity: totalCapacity,
		ActiveTasks:   int(activeTasksTotal),
		Metrics:       c.metrics.GetSnapshot(),
	}
}

// Shutdown gracefully shuts down the coordinator
func (c *Coordinator) Shutdown(timeout time.Duration) error {
	c.cancel()

	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Clean shutdown
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout exceeded")
	}

	// Close node connections
	c.nodesMu.Lock()
	for _, node := range c.nodes {
		node.client.Close()
	}
	c.nodesMu.Unlock()

	// Close channels
	close(c.eventQueue)
	close(c.resultQueue)

	return nil
}

// CoordinatorStatus represents the current state
type CoordinatorStatus struct {
	Nodes         map[string]NodeStatusInfo
	QueueLength   int
	TotalCapacity int
	ActiveTasks   int
	Metrics       MetricsSnapshot
}

// NodeStatusInfo contains node status details
type NodeStatusInfo struct {
	ID             string
	Status         NodeStatus
	ActiveTasks    int
	Capacity       int
	ProcessedCount int64
	ErrorCount     int64
	ResponseTime   time.Duration
}

// MarshalJSON implements json.Marshaler
func (s NodeStatus) MarshalJSON() ([]byte, error) {
	var status string
	switch s {
	case NodeStatusHealthy:
		status = "healthy"
	case NodeStatusDegraded:
		status = "degraded"
	case NodeStatusUnhealthy:
		status = "unhealthy"
	case NodeStatusOffline:
		status = "offline"
	default:
		status = "unknown"
	}
	return json.Marshal(status)
}
