package distributed

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// LoadBalancer selects the best node for a task
type LoadBalancer interface {
	SelectNode(preferredNode string, availableNodes []*WorkerNode) string
	UpdateMetrics(nodeID string, responseTime time.Duration, success bool)
}

// LeastLoadedBalancer selects the least loaded node
type LeastLoadedBalancer struct {
	mu sync.RWMutex
}

// NewLeastLoadedBalancer creates a least-loaded balancer
func NewLeastLoadedBalancer() *LeastLoadedBalancer {
	return &LeastLoadedBalancer{}
}

// SelectNode selects node with lowest load
func (b *LeastLoadedBalancer) SelectNode(preferredNode string, availableNodes []*WorkerNode) string {
	if len(availableNodes) == 0 {
		return ""
	}

	// Check if preferred node is available and not overloaded
	for _, node := range availableNodes {
		if node.ID == preferredNode {
			load := float64(atomic.LoadInt32(&node.ActiveTasks)) / float64(node.Capacity)
			if load < 0.8 { // 80% threshold
				return preferredNode
			}
			break
		}
	}

	// Find least loaded node
	var selectedNode *WorkerNode
	minLoad := math.MaxFloat64

	for _, node := range availableNodes {
		activeTasks := atomic.LoadInt32(&node.ActiveTasks)
		load := float64(activeTasks) / float64(node.Capacity)

		if load < minLoad {
			minLoad = load
			selectedNode = node
		}
	}

	if selectedNode != nil {
		return selectedNode.ID
	}

	// Fallback to first available
	return availableNodes[0].ID
}

// UpdateMetrics is a no-op for least-loaded balancer
func (b *LeastLoadedBalancer) UpdateMetrics(nodeID string, responseTime time.Duration, success bool) {
}

// WeightedResponseTimeBalancer selects based on response times
type WeightedResponseTimeBalancer struct {
	nodeStats map[string]*NodeStats
	mu        sync.RWMutex
}

// NodeStats tracks node performance statistics
type NodeStats struct {
	TotalRequests    uint64
	TotalTime        uint64 // in microseconds
	FailureCount     uint64
	LastResponseTime time.Duration
	LastUpdate       time.Time
}

// NewWeightedResponseTimeBalancer creates a response-time based balancer
func NewWeightedResponseTimeBalancer() *WeightedResponseTimeBalancer {
	return &WeightedResponseTimeBalancer{
		nodeStats: make(map[string]*NodeStats),
	}
}

// SelectNode selects node based on response times
func (b *WeightedResponseTimeBalancer) SelectNode(preferredNode string, availableNodes []*WorkerNode) string {
	if len(availableNodes) == 0 {
		return ""
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Calculate scores for each node
	type nodeScore struct {
		nodeID string
		score  float64
	}

	scores := make([]nodeScore, 0, len(availableNodes))

	for _, node := range availableNodes {
		score := b.calculateNodeScore(node)
		scores = append(scores, nodeScore{
			nodeID: node.ID,
			score:  score,
		})
	}

	// Select node with best score
	bestScore := math.Inf(-1)
	selectedNode := ""

	for _, ns := range scores {
		if ns.score > bestScore {
			bestScore = ns.score
			selectedNode = ns.nodeID
		}
	}

	if selectedNode != "" {
		return selectedNode
	}

	// Fallback
	return availableNodes[0].ID
}

// calculateNodeScore calculates a score for node selection
func (b *WeightedResponseTimeBalancer) calculateNodeScore(node *WorkerNode) float64 {
	stats, exists := b.nodeStats[node.ID]
	if !exists {
		// New node, give it a chance
		return 1.0
	}

	// Calculate average response time
	avgResponseTime := float64(1.0) // Default 1ms
	if stats.TotalRequests > 0 {
		avgResponseTime = float64(stats.TotalTime) / float64(stats.TotalRequests) / 1000.0 // Convert to ms
	}

	// Calculate failure rate
	failureRate := float64(0)
	if stats.TotalRequests > 0 {
		failureRate = float64(stats.FailureCount) / float64(stats.TotalRequests)
	}

	// Calculate current load
	currentLoad := float64(atomic.LoadInt32(&node.ActiveTasks)) / float64(node.Capacity)

	// Calculate score (higher is better)
	// Penalize high response times, failure rates, and current load
	score := 1.0 / (avgResponseTime * (1.0 + failureRate) * (1.0 + currentLoad))

	// Boost score for recently updated nodes
	if time.Since(stats.LastUpdate) < 30*time.Second {
		score *= 1.1
	}

	return score
}

// UpdateMetrics updates node performance metrics
func (b *WeightedResponseTimeBalancer) UpdateMetrics(nodeID string, responseTime time.Duration, success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	stats, exists := b.nodeStats[nodeID]
	if !exists {
		stats = &NodeStats{}
		b.nodeStats[nodeID] = stats
	}

	atomic.AddUint64(&stats.TotalRequests, 1)
	atomic.AddUint64(&stats.TotalTime, uint64(responseTime.Microseconds()))

	if !success {
		atomic.AddUint64(&stats.FailureCount, 1)
	}

	stats.LastResponseTime = responseTime
	stats.LastUpdate = time.Now()
}

// RoundRobinBalancer distributes requests evenly
type RoundRobinBalancer struct {
	counter uint64
}

// NewRoundRobinBalancer creates a round-robin balancer
func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{}
}

// SelectNode selects next node in sequence
func (b *RoundRobinBalancer) SelectNode(preferredNode string, availableNodes []*WorkerNode) string {
	if len(availableNodes) == 0 {
		return ""
	}

	count := atomic.AddUint64(&b.counter, 1)
	index := (count - 1) % uint64(len(availableNodes))

	return availableNodes[index].ID
}

// UpdateMetrics is a no-op for round-robin
func (b *RoundRobinBalancer) UpdateMetrics(nodeID string, responseTime time.Duration, success bool) {}

// PowerOfTwoBalancer uses power-of-two-choices algorithm
type PowerOfTwoBalancer struct {
	random RandomSource
	mu     sync.RWMutex
}

// RandomSource provides random number generation
type RandomSource interface {
	Intn(n int) int
}

// NewPowerOfTwoBalancer creates a power-of-two balancer
func NewPowerOfTwoBalancer(random RandomSource) *PowerOfTwoBalancer {
	return &PowerOfTwoBalancer{
		random: random,
	}
}

// SelectNode selects best of two random nodes
func (b *PowerOfTwoBalancer) SelectNode(preferredNode string, availableNodes []*WorkerNode) string {
	if len(availableNodes) == 0 {
		return ""
	}

	if len(availableNodes) == 1 {
		return availableNodes[0].ID
	}

	// Select two random nodes
	idx1 := b.random.Intn(len(availableNodes))
	idx2 := b.random.Intn(len(availableNodes))

	// Ensure they're different
	for idx2 == idx1 && len(availableNodes) > 1 {
		idx2 = b.random.Intn(len(availableNodes))
	}

	node1 := availableNodes[idx1]
	node2 := availableNodes[idx2]

	// Compare loads
	load1 := float64(atomic.LoadInt32(&node1.ActiveTasks)) / float64(node1.Capacity)
	load2 := float64(atomic.LoadInt32(&node2.ActiveTasks)) / float64(node2.Capacity)

	if load1 <= load2 {
		return node1.ID
	}
	return node2.ID
}

// UpdateMetrics is a no-op for power-of-two
func (b *PowerOfTwoBalancer) UpdateMetrics(nodeID string, responseTime time.Duration, success bool) {}

// AdaptiveBalancer switches strategies based on conditions
type AdaptiveBalancer struct {
	strategies       map[string]LoadBalancer
	currentStrategy  string
	metricsCollector *AdaptiveMetrics
	mu               sync.RWMutex
}

// AdaptiveMetrics collects system-wide metrics
type AdaptiveMetrics struct {
	totalRequests   uint64
	totalFailures   uint64
	avgResponseTime float64
	loadVariance    float64
	lastEvaluation  time.Time
}

// NewAdaptiveBalancer creates an adaptive balancer
func NewAdaptiveBalancer() *AdaptiveBalancer {
	balancer := &AdaptiveBalancer{
		strategies: make(map[string]LoadBalancer),
		metricsCollector: &AdaptiveMetrics{
			lastEvaluation: time.Now(),
		},
	}

	// Initialize strategies
	balancer.strategies["least-loaded"] = NewLeastLoadedBalancer()
	balancer.strategies["weighted-response"] = NewWeightedResponseTimeBalancer()
	balancer.strategies["round-robin"] = NewRoundRobinBalancer()

	// Start with least-loaded
	balancer.currentStrategy = "least-loaded"

	// Start adaptation loop
	go balancer.adaptationLoop()

	return balancer
}

// SelectNode delegates to current strategy
func (b *AdaptiveBalancer) SelectNode(preferredNode string, availableNodes []*WorkerNode) string {
	b.mu.RLock()
	strategy := b.strategies[b.currentStrategy]
	b.mu.RUnlock()

	return strategy.SelectNode(preferredNode, availableNodes)
}

// UpdateMetrics updates metrics for all strategies
func (b *AdaptiveBalancer) UpdateMetrics(nodeID string, responseTime time.Duration, success bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Update all strategies
	for _, strategy := range b.strategies {
		strategy.UpdateMetrics(nodeID, responseTime, success)
	}

	// Update adaptive metrics
	atomic.AddUint64(&b.metricsCollector.totalRequests, 1)
	if !success {
		atomic.AddUint64(&b.metricsCollector.totalFailures, 1)
	}

	// Update average response time (simplified moving average)
	currentAvg := b.metricsCollector.avgResponseTime
	newAvg := currentAvg*0.95 + float64(responseTime.Milliseconds())*0.05
	b.metricsCollector.avgResponseTime = newAvg
}

// adaptationLoop periodically evaluates and switches strategies
func (b *AdaptiveBalancer) adaptationLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		b.evaluateAndAdapt()
	}
}

// evaluateAndAdapt evaluates current conditions and adapts strategy
func (b *AdaptiveBalancer) evaluateAndAdapt() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Calculate metrics
	failureRate := float64(0)
	totalReqs := atomic.LoadUint64(&b.metricsCollector.totalRequests)
	if totalReqs > 0 {
		failureRate = float64(atomic.LoadUint64(&b.metricsCollector.totalFailures)) / float64(totalReqs)
	}

	avgResponseTime := b.metricsCollector.avgResponseTime

	// Decision logic
	newStrategy := b.currentStrategy

	if failureRate > 0.1 {
		// High failure rate - use round-robin to distribute load
		newStrategy = "round-robin"
	} else if avgResponseTime > 100 {
		// High response times - use weighted response time
		newStrategy = "weighted-response"
	} else {
		// Normal conditions - use least loaded
		newStrategy = "least-loaded"
	}

	if newStrategy != b.currentStrategy {
		b.currentStrategy = newStrategy
		// Log strategy change
	}

	// Reset counters
	atomic.StoreUint64(&b.metricsCollector.totalRequests, 0)
	atomic.StoreUint64(&b.metricsCollector.totalFailures, 0)
	b.metricsCollector.lastEvaluation = time.Now()
}

// HealthAwareBalancer considers node health in selection
type HealthAwareBalancer struct {
	baseBalancer LoadBalancer
	healthScores map[string]float64
	mu           sync.RWMutex
}

// NewHealthAwareBalancer wraps a balancer with health awareness
func NewHealthAwareBalancer(base LoadBalancer) *HealthAwareBalancer {
	return &HealthAwareBalancer{
		baseBalancer: base,
		healthScores: make(map[string]float64),
	}
}

// SelectNode filters unhealthy nodes before selection
func (b *HealthAwareBalancer) SelectNode(preferredNode string, availableNodes []*WorkerNode) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Filter healthy nodes
	healthyNodes := make([]*WorkerNode, 0, len(availableNodes))

	for _, node := range availableNodes {
		// Check node status
		if node.Status == NodeStatusHealthy || node.Status == NodeStatusDegraded {
			// Check health score if available
			score, exists := b.healthScores[node.ID]
			if !exists || score > 0.3 { // 30% health threshold
				healthyNodes = append(healthyNodes, node)
			}
		}
	}

	// If no healthy nodes, use all available (degraded mode)
	if len(healthyNodes) == 0 {
		healthyNodes = availableNodes
	}

	return b.baseBalancer.SelectNode(preferredNode, healthyNodes)
}

// UpdateMetrics updates both base balancer and health scores
func (b *HealthAwareBalancer) UpdateMetrics(nodeID string, responseTime time.Duration, success bool) {
	b.baseBalancer.UpdateMetrics(nodeID, responseTime, success)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Update health score
	currentScore, exists := b.healthScores[nodeID]
	if !exists {
		currentScore = 1.0
	}

	// Adjust score based on success
	if success {
		// Increase health score
		currentScore = math.Min(currentScore*1.05, 1.0)
	} else {
		// Decrease health score
		currentScore = currentScore * 0.9
	}

	b.healthScores[nodeID] = currentScore
}

// ResponseTimeTracker tracks response time statistics
type ResponseTimeTracker struct {
	samples  []time.Duration
	capacity int
	position int
	count    int
	mu       sync.Mutex
}

// NewResponseTimeTracker creates a response time tracker
func NewResponseTimeTracker(capacity int) *ResponseTimeTracker {
	return &ResponseTimeTracker{
		samples:  make([]time.Duration, capacity),
		capacity: capacity,
	}
}

// Record adds a response time sample
func (t *ResponseTimeTracker) Record(duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.samples[t.position] = duration
	t.position = (t.position + 1) % t.capacity

	if t.count < t.capacity {
		t.count++
	}
}

// GetAverage returns average response time
func (t *ResponseTimeTracker) GetAverage() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.count == 0 {
		return 0
	}

	var total time.Duration
	for i := 0; i < t.count; i++ {
		total += t.samples[i]
	}

	return total / time.Duration(t.count)
}

// GetPercentile returns percentile response time
func (t *ResponseTimeTracker) GetPercentile(percentile float64) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.count == 0 {
		return 0
	}

	// Copy and sort samples
	sorted := make([]time.Duration, t.count)
	copy(sorted, t.samples[:t.count])

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate percentile index
	index := int(float64(t.count-1) * percentile / 100.0)
	return sorted[index]
}
