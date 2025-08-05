package distributed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WorkerClient interface for worker communication
type WorkerClient interface {
	ProcessBatch(ctx context.Context, tasks []*ProcessingTask) ([]*ProcessingResult, error)
	HealthCheck(ctx context.Context) (*HealthCheckResponse, error)
	Close() error
}

// HTTPWorkerClient implements WorkerClient using HTTP
type HTTPWorkerClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewWorkerClient creates a new HTTP worker client
func NewWorkerClient(address string) (WorkerClient, error) {
	return &HTTPWorkerClient{
		baseURL: fmt.Sprintf("http://%s", address),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}, nil
}

// ProcessBatch sends a batch of tasks to worker
func (c *HTTPWorkerClient) ProcessBatch(ctx context.Context, tasks []*ProcessingTask) ([]*ProcessingResult, error) {
	// Serialize tasks
	data, err := json.Marshal(tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tasks: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/batch", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("worker returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse results
	var results []*ProcessingResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode results: %w", err)
	}

	return results, nil
}

// HealthCheck performs a health check on the worker
func (c *HTTPWorkerClient) HealthCheck(ctx context.Context) (*HealthCheckResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	var health WorkerHealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health status: %w", err)
	}

	return &HealthCheckResponse{
		Healthy:      health.Healthy,
		LoadAverage:  health.LoadAverage,
		ErrorRate:    health.ErrorRate,
		ActiveTasks:  int(health.ActiveTasks),
		QueueDepth:   health.QueueDepth,
		MemoryUsage:  health.MemoryUsage,
		CPUUsage:     health.CPUUsage,
		ResponseTime: time.Since(health.LastHeartbeat),
	}, nil
}

// Close closes the client
func (c *HTTPWorkerClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// HealthCheckResponse contains health check results
type HealthCheckResponse struct {
	Healthy      bool
	LoadAverage  float64
	ErrorRate    float64
	ActiveTasks  int
	QueueDepth   int
	MemoryUsage  uint64
	CPUUsage     float64
	ResponseTime time.Duration
}

// GRPCWorkerClient implements WorkerClient using gRPC
type GRPCWorkerClient struct {
	address string
	// In production, this would use actual gRPC client
}

// NewGRPCWorkerClient creates a gRPC worker client
func NewGRPCWorkerClient(address string) (WorkerClient, error) {
	return &GRPCWorkerClient{
		address: address,
	}, nil
}

// ProcessBatch sends tasks via gRPC
func (c *GRPCWorkerClient) ProcessBatch(ctx context.Context, tasks []*ProcessingTask) ([]*ProcessingResult, error) {
	// gRPC implementation would go here
	return nil, fmt.Errorf("gRPC not implemented in this demo")
}

// HealthCheck performs health check via gRPC
func (c *GRPCWorkerClient) HealthCheck(ctx context.Context) (*HealthCheckResponse, error) {
	// gRPC implementation would go here
	return nil, fmt.Errorf("gRPC not implemented in this demo")
}

// Close closes gRPC connection
func (c *GRPCWorkerClient) Close() error {
	// Close gRPC connection
	return nil
}

// MockWorkerClient for testing
type MockWorkerClient struct {
	processFunc func(ctx context.Context, tasks []*ProcessingTask) ([]*ProcessingResult, error)
	healthFunc  func(ctx context.Context) (*HealthCheckResponse, error)
	closed      bool
}

// NewMockWorkerClient creates a mock client
func NewMockWorkerClient() *MockWorkerClient {
	return &MockWorkerClient{
		processFunc: func(ctx context.Context, tasks []*ProcessingTask) ([]*ProcessingResult, error) {
			// Default mock implementation
			results := make([]*ProcessingResult, len(tasks))
			for i, task := range tasks {
				results[i] = &ProcessingResult{
					TaskID:         task.ID,
					Success:        true,
					ProcessingTime: 10 * time.Millisecond,
					CompletedAt:    time.Now(),
				}
			}
			return results, nil
		},
		healthFunc: func(ctx context.Context) (*HealthCheckResponse, error) {
			return &HealthCheckResponse{
				Healthy:     true,
				LoadAverage: 0.5,
				ErrorRate:   0.01,
				ActiveTasks: 2,
				QueueDepth:  10,
			}, nil
		},
	}
}

// ProcessBatch mocks batch processing
func (c *MockWorkerClient) ProcessBatch(ctx context.Context, tasks []*ProcessingTask) ([]*ProcessingResult, error) {
	if c.closed {
		return nil, fmt.Errorf("client closed")
	}
	return c.processFunc(ctx, tasks)
}

// HealthCheck mocks health check
func (c *MockWorkerClient) HealthCheck(ctx context.Context) (*HealthCheckResponse, error) {
	if c.closed {
		return nil, fmt.Errorf("client closed")
	}
	return c.healthFunc(ctx)
}

// Close closes the mock client
func (c *MockWorkerClient) Close() error {
	c.closed = true
	return nil
}

// SetProcessFunc sets custom process function
func (c *MockWorkerClient) SetProcessFunc(f func(ctx context.Context, tasks []*ProcessingTask) ([]*ProcessingResult, error)) {
	c.processFunc = f
}

// SetHealthFunc sets custom health function
func (c *MockWorkerClient) SetHealthFunc(f func(ctx context.Context) (*HealthCheckResponse, error)) {
	c.healthFunc = f
}
