package distributed

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestCoordinator(t *testing.T) {
	config := CoordinatorConfig{
		MinNodes:            1,
		MaxNodes:            5,
		NodeTimeout:         30 * time.Second,
		HealthCheckInterval: 5 * time.Second,
		QueueSize:           1000,
		BatchSize:           10,
		BatchTimeout:        100 * time.Millisecond,
		MaxRetries:          3,
		PartitionStrategy:   "hash",
		LoadBalanceStrategy: "least-loaded",
		EnableFailover:      true,
		FailoverTimeout:     5 * time.Second,
	}

	coord, err := NewCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coord.Shutdown(5 * time.Second)

	t.Run("NodeRegistration", func(t *testing.T) {
		// Register nodes
		for i := 1; i <= 3; i++ {
			nodeID := fmt.Sprintf("node-%d", i)
			err := coord.RegisterNode(nodeID, fmt.Sprintf("localhost:808%d", i), 10)
			if err != nil {
				t.Errorf("Failed to register node %s: %v", nodeID, err)
			}
		}

		// Verify nodes registered
		status := coord.GetStatus()
		if len(status.Nodes) != 3 {
			t.Errorf("Expected 3 nodes, got %d", len(status.Nodes))
		}
	})

	t.Run("TaskSubmission", func(t *testing.T) {
		task := &ProcessingTask{
			ID:           "test-task-001",
			Type:         TaskTypeAnalyze,
			Priority:     1,
			Data:         []byte(`{"test": "data"}`),
			CreatedAt:    time.Now(),
			PartitionKey: "test-key",
		}

		err := coord.Submit(task)
		if err != nil {
			t.Errorf("Failed to submit task: %v", err)
		}

		// Check metrics
		metrics := coord.metrics.GetSnapshot()
		if metrics.TasksSubmitted != 1 {
			t.Errorf("Expected 1 task submitted, got %d", metrics.TasksSubmitted)
		}
	})

	t.Run("BatchSubmission", func(t *testing.T) {
		tasks := make([]*ProcessingTask, 5)
		for i := 0; i < 5; i++ {
			tasks[i] = &ProcessingTask{
				ID:           fmt.Sprintf("batch-task-%03d", i),
				Type:         TaskTypeCapture,
				Priority:     1,
				Data:         []byte(fmt.Sprintf(`{"index": %d}`, i)),
				CreatedAt:    time.Now(),
				PartitionKey: fmt.Sprintf("key-%d", i),
			}
		}

		err := coord.SubmitBatch(tasks)
		if err != nil {
			t.Errorf("Failed to submit batch: %v", err)
		}
	})

	t.Run("NodeUnregistration", func(t *testing.T) {
		err := coord.UnregisterNode("node-3")
		if err != nil {
			t.Errorf("Failed to unregister node: %v", err)
		}

		status := coord.GetStatus()
		if len(status.Nodes) != 2 {
			t.Errorf("Expected 2 nodes after unregistration, got %d", len(status.Nodes))
		}
	})
}

func TestPartitioners(t *testing.T) {
	nodes := []*WorkerNode{
		{ID: "node-1", Capacity: 10},
		{ID: "node-2", Capacity: 10},
		{ID: "node-3", Capacity: 10},
	}

	t.Run("HashPartitioner", func(t *testing.T) {
		partitioner := NewHashPartitioner()

		// Same key should always go to same node
		key := "test-key"
		node1 := partitioner.GetNode(key, nodes)
		node2 := partitioner.GetNode(key, nodes)

		if node1 != node2 {
			t.Errorf("Hash partitioner not consistent: %s != %s", node1, node2)
		}
	})

	t.Run("ConsistentHashPartitioner", func(t *testing.T) {
		partitioner := NewConsistentHashPartitioner(100)

		// Add nodes
		for _, node := range nodes {
			partitioner.AddNode(node.ID, 1)
		}

		// Test consistency
		key := "test-key"
		node1 := partitioner.GetNode(key, nodes)

		// Remove a different node
		partitioner.RemoveNode("node-2")
		remainingNodes := []*WorkerNode{nodes[0], nodes[2]}

		node2 := partitioner.GetNode(key, remainingNodes)

		// Key should still map to same node if that node is available
		if node1 == "node-2" {
			// Original node was removed, mapping should change
			if node2 == "node-2" {
				t.Error("Removed node still selected")
			}
		} else {
			// Original node still available, mapping should stay same
			if node1 != node2 {
				t.Errorf("Consistent hash changed unnecessarily: %s != %s", node1, node2)
			}
		}
	})

	t.Run("RoundRobinPartitioner", func(t *testing.T) {
		partitioner := NewRoundRobinPartitioner()

		// Should cycle through nodes
		selections := make(map[string]int)
		for i := 0; i < 30; i++ {
			node := partitioner.GetNode(fmt.Sprintf("key-%d", i), nodes)
			selections[node]++
		}

		// Each node should get roughly equal share
		for _, node := range nodes {
			count := selections[node.ID]
			if count < 8 || count > 12 {
				t.Errorf("Round robin not balanced: node %s got %d", node.ID, count)
			}
		}
	})

	t.Run("WeightedPartitioner", func(t *testing.T) {
		partitioner := NewWeightedPartitioner()

		// Set different weights
		partitioner.AddNode("node-1", 1)
		partitioner.AddNode("node-2", 2)
		partitioner.AddNode("node-3", 3)

		// Test distribution
		selections := make(map[string]int)
		for i := 0; i < 600; i++ {
			node := partitioner.GetNode(fmt.Sprintf("key-%d", i), nodes)
			selections[node]++
		}

		// Node-3 should get roughly 3x node-1's load
		ratio := float64(selections["node-3"]) / float64(selections["node-1"])
		if ratio < 2.5 || ratio > 3.5 {
			t.Errorf("Weighted distribution incorrect: ratio %.2f", ratio)
		}
	})
}

func TestLoadBalancers(t *testing.T) {
	nodes := []*WorkerNode{
		{ID: "node-1", Capacity: 10, ActiveTasks: 2},
		{ID: "node-2", Capacity: 10, ActiveTasks: 5},
		{ID: "node-3", Capacity: 10, ActiveTasks: 8},
	}

	t.Run("LeastLoadedBalancer", func(t *testing.T) {
		balancer := NewLeastLoadedBalancer()

		// Should select node with least load
		selected := balancer.SelectNode("", nodes)
		if selected != "node-1" {
			t.Errorf("Expected node-1 (least loaded), got %s", selected)
		}

		// Preferred node should be selected if not overloaded
		selected = balancer.SelectNode("node-2", nodes)
		if selected != "node-2" {
			t.Errorf("Expected preferred node-2, got %s", selected)
		}
	})

	t.Run("WeightedResponseTimeBalancer", func(t *testing.T) {
		balancer := NewWeightedResponseTimeBalancer()

		// Add some metrics
		balancer.UpdateMetrics("node-1", 10*time.Millisecond, true)
		balancer.UpdateMetrics("node-1", 15*time.Millisecond, true)
		balancer.UpdateMetrics("node-2", 50*time.Millisecond, true)
		balancer.UpdateMetrics("node-2", 100*time.Millisecond, false) // failure
		balancer.UpdateMetrics("node-3", 5*time.Millisecond, true)

		// Node-3 should be preferred (fastest, no failures)
		selected := balancer.SelectNode("", nodes)
		if selected != "node-3" && selected != "node-1" {
			t.Errorf("Expected fast node, got %s", selected)
		}
	})

	t.Run("PowerOfTwoBalancer", func(t *testing.T) {
		random := &mockRandom{values: []int{0, 2, 1, 2}}
		balancer := NewPowerOfTwoBalancer(random)

		// Should select less loaded of two random nodes
		selected := balancer.SelectNode("", nodes)
		// Random selected node-1 (index 0) and node-3 (index 2)
		// node-1 has less load, should be selected
		if selected != "node-1" {
			t.Errorf("Expected node-1 (less loaded), got %s", selected)
		}
	})
}

func TestWorker(t *testing.T) {
	config := WorkerConfig{
		ID:             "test-worker",
		ListenAddress:  "localhost:8090",
		QueueSize:      100,
		MaxConcurrent:  5,
		ProcessTimeout: 30 * time.Second,
		EnableML:       false,
	}

	// Use mock capture and dissect configs
	// config.CaptureConfig.Interfaces = []string{"lo"}
	// config.DissectorConfig.MaxGoroutines = 2

	worker, err := NewWorker(config)
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}
	defer worker.Shutdown(5 * time.Second)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	t.Run("HealthCheck", func(t *testing.T) {
		health := worker.GetHealth()
		if !health.Healthy {
			t.Error("Worker should be healthy initially")
		}
		if health.LoadAverage != 0 {
			t.Errorf("Expected 0 load average, got %f", health.LoadAverage)
		}
	})

	t.Run("ProcessTask", func(t *testing.T) {
		task := &ProcessingTask{
			ID:   "test-001",
			Type: TaskTypeAggregate,
			Data: []byte(`{
				"window": "5m",
				"group_by": ["source"],
				"metrics": ["count"],
				"events": []
			}`),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result := worker.processTask(ctx, task)
		if !result.Success {
			t.Errorf("Task processing failed: %v", result.Error)
		}
		if result.TaskID != task.ID {
			t.Errorf("Result task ID mismatch: %s != %s", result.TaskID, task.ID)
		}
	})
}

func TestDistributedProcessing(t *testing.T) {
	// This test simulates a full distributed processing scenario
	// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel()

	// Create coordinator
	coordConfig := CoordinatorConfig{
		MinNodes:            2,
		MaxNodes:            5,
		QueueSize:           1000,
		BatchSize:           20,
		BatchTimeout:        200 * time.Millisecond,
		PartitionStrategy:   "consistent-hash",
		LoadBalanceStrategy: "weighted-response",
	}

	coord, err := NewCoordinator(coordConfig)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coord.Shutdown(5 * time.Second)

	// Create mock workers
	mockClients := make([]*MockWorkerClient, 3)
	for i := 0; i < 3; i++ {
		nodeID := fmt.Sprintf("worker-%d", i)
		client := NewMockWorkerClient()
		mockClients[i] = client

		// Register with coordinator (mock registration)
		coord.nodes[nodeID] = &WorkerNode{
			ID:            nodeID,
			Address:       fmt.Sprintf("mock:%d", i),
			Status:        NodeStatusHealthy,
			Capacity:      20,
			ResponseTimes: NewResponseTimeTracker(100),
			client:        client,
		}

		// Update partitioner
		if ch, ok := coord.partitioner.(*ConsistentHashPartitioner); ok {
			ch.AddNode(nodeID, 1)
		}
	}

	// Submit many tasks
	var wg sync.WaitGroup
	numTasks := 100

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			task := &ProcessingTask{
				ID:           fmt.Sprintf("task-%04d", idx),
				Type:         TaskTypeAnalyze,
				Priority:     rand.Intn(3) + 1,
				Data:         []byte(fmt.Sprintf(`{"index": %d}`, idx)),
				CreatedAt:    time.Now(),
				PartitionKey: fmt.Sprintf("key-%d", idx%10),
			}

			if err := coord.Submit(task); err != nil {
				t.Logf("Failed to submit task %s: %v", task.ID, err)
			}
		}(i)
	}

	// Wait for submission
	wg.Wait()

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Check metrics
	metrics := coord.metrics.GetSnapshot()
	t.Logf("Processing metrics: submitted=%d, completed=%d, failed=%d",
		metrics.TasksSubmitted, metrics.TasksCompleted, metrics.TasksFailed)

	if metrics.TasksSubmitted != uint64(numTasks) {
		t.Errorf("Expected %d tasks submitted, got %d", numTasks, metrics.TasksSubmitted)
	}

	// Check distribution across nodes
	status := coord.GetStatus()
	for nodeID, nodeStatus := range status.Nodes {
		t.Logf("Node %s: processed=%d, errors=%d",
			nodeID, nodeStatus.ProcessedCount, nodeStatus.ErrorCount)
	}
}

func TestMetrics(t *testing.T) {
	metrics := NewCoordinatorMetrics()

	t.Run("TaskMetrics", func(t *testing.T) {
		metrics.RecordTaskSubmitted(TaskTypeAnalyze)
		metrics.RecordTaskCompleted(true, 50*time.Millisecond)
		metrics.RecordTaskCompleted(false, 100*time.Millisecond)
		metrics.RecordTaskRetry()

		snapshot := metrics.GetSnapshot()
		if snapshot.TasksSubmitted != 1 {
			t.Errorf("Expected 1 task submitted, got %d", snapshot.TasksSubmitted)
		}
		if snapshot.TasksCompleted != 1 {
			t.Errorf("Expected 1 task completed, got %d", snapshot.TasksCompleted)
		}
		if snapshot.TasksFailed != 1 {
			t.Errorf("Expected 1 task failed, got %d", snapshot.TasksFailed)
		}
		if snapshot.SuccessRate != 0.5 {
			t.Errorf("Expected 0.5 success rate, got %f", snapshot.SuccessRate)
		}
	})

	t.Run("NodeMetrics", func(t *testing.T) {
		metrics.RecordNodeRegistered("node-1")
		metrics.RecordNodeError("node-1")
		metrics.RecordNodeError("node-1")
		metrics.RecordHealthCheckFailure("node-1")

		snapshot := metrics.GetSnapshot()
		if snapshot.NodesRegistered != 1 {
			t.Errorf("Expected 1 node registered, got %d", snapshot.NodesRegistered)
		}
		if snapshot.NodeErrors["node-1"] != 3 { // 2 errors + 1 from health check
			t.Errorf("Expected 3 node errors, got %d", snapshot.NodeErrors["node-1"])
		}
		if snapshot.HealthCheckFailures != 1 {
			t.Errorf("Expected 1 health check failure, got %d", snapshot.HealthCheckFailures)
		}
	})

	t.Run("PrometheusExport", func(t *testing.T) {
		exporter := NewPrometheusExporter(metrics)
		output := exporter.Export()

		// Check for key metrics in output
		expectedMetrics := []string{
			"strigoi_tasks_submitted_total",
			"strigoi_tasks_completed_total",
			"strigoi_tasks_failed_total",
			"strigoi_task_success_rate",
			"strigoi_nodes_registered_total",
		}

		for _, metric := range expectedMetrics {
			if !contains(output, metric) {
				t.Errorf("Expected metric %s not found in Prometheus output", metric)
			}
		}
	})
}

func BenchmarkCoordinator(b *testing.B) {
	config := CoordinatorConfig{
		QueueSize:           10000,
		BatchSize:           50,
		BatchTimeout:        50 * time.Millisecond,
		PartitionStrategy:   "hash",
		LoadBalanceStrategy: "round-robin",
	}

	coord, _ := NewCoordinator(config)
	defer coord.Shutdown(5 * time.Second)

	// Add mock nodes
	for i := 0; i < 5; i++ {
		nodeID := fmt.Sprintf("bench-node-%d", i)
		coord.nodes[nodeID] = &WorkerNode{
			ID:            nodeID,
			Capacity:      100,
			ResponseTimes: NewResponseTimeTracker(100),
			client:        NewMockWorkerClient(),
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			task := &ProcessingTask{
				ID:           fmt.Sprintf("bench-%d", i),
				Type:         TaskTypeAnalyze,
				Data:         []byte(`{"test": "data"}`),
				PartitionKey: fmt.Sprintf("key-%d", i),
			}
			coord.Submit(task)
			i++
		}
	})
}

// Helper types

type mockRandom struct {
	values []int
	index  int
}

func (m *mockRandom) Intn(n int) int {
	if m.index >= len(m.values) {
		return 0
	}
	val := m.values[m.index] % n
	m.index++
	return val
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
