package ai

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TaskType represents different AI task types
type TaskType string

const (
	TaskAnalyze          TaskType = "analyze"
	TaskGenerate         TaskType = "generate"
	TaskVisualAnalysis   TaskType = "visual"
	TaskBulkScan         TaskType = "bulk"
	TaskCorrelate        TaskType = "correlate"
	TaskValidate         TaskType = "validate"
	TaskEthical          TaskType = "ethical"
	TaskRealTimeDefense  TaskType = "realtime_defense"
)

// Priority levels for task execution
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// Task represents a work item for AI processing
type Task struct {
	Type     TaskType
	Priority Priority
	AI       string // Specific AI to use (optional)
	Payload  map[string]interface{}
}

// Dispatcher interface for routing tasks to AI providers
type Dispatcher interface {
	Route(ctx context.Context, task Task) (*Response, error)
	RouteWithPriority(ctx context.Context, task Task) (*Response, error)
	GetStatus() Status
}

// BasicDispatcher provides simple task routing
type BasicDispatcher struct {
	handler Handler
	mu      sync.RWMutex
}

// NewBasicDispatcher creates a basic dispatcher with a single handler
func NewBasicDispatcher(handler Handler) *BasicDispatcher {
	return &BasicDispatcher{
		handler: handler,
	}
}

// Route processes a task through available AI handlers
func (d *BasicDispatcher) Route(ctx context.Context, task Task) (*Response, error) {
	return d.RouteWithPriority(ctx, task)
}

// RouteWithPriority processes a task with priority handling
func (d *BasicDispatcher) RouteWithPriority(ctx context.Context, task Task) (*Response, error) {
	d.mu.RLock()
	handler := d.handler
	d.mu.RUnlock()

	if handler == nil {
		return nil, fmt.Errorf("no AI handler available")
	}

	// Route based on task type
	switch task.Type {
	case TaskAnalyze:
		entityID, _ := task.Payload["entity_id"].(string)
		return handler.Analyze(ctx, entityID, task.Payload)
		
	case TaskGenerate:
		// For now, use suggest as a proxy for generation
		situation := fmt.Sprintf("Generate: %v", task.Payload)
		return handler.Suggest(ctx, situation, task.Payload)
		
	case TaskCorrelate:
		// Use analyze for correlation tasks
		entityID := "correlation_task"
		return handler.Analyze(ctx, entityID, task.Payload)
		
	case TaskBulkScan:
		// Use analyze for bulk scanning
		entityID := "bulk_scan"
		return handler.Analyze(ctx, entityID, task.Payload)
		
	case TaskVisualAnalysis:
		// Visual analysis requires GPT-4o, mock for now
		return &Response{
			Source:     "mock",
			Message:    "Visual analysis not yet implemented",
			Confidence: 0.0,
			Timestamp:  time.Now(),
			Analysis: map[string]interface{}{
				"error": "GPT-4o integration pending",
			},
		}, nil
		
	case TaskRealTimeDefense:
		// Process real-time defense tasks
		threatType, _ := task.Payload["threat_type"].(string)
		return &Response{
			Source:     "mock",
			Message:    fmt.Sprintf("Analyzing %s threat", threatType),
			Confidence: 0.85,
			Timestamp:  time.Now(),
			Action:     "monitor",
			Analysis: map[string]interface{}{
				"threat_type":        threatType,
				"injection_detected": false,
			},
		}, nil
		
	default:
		return nil, fmt.Errorf("unknown task type: %s", task.Type)
	}
}

// GetStatus returns dispatcher status
func (d *BasicDispatcher) GetStatus() Status {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	if d.handler != nil {
		return d.handler.GetStatus()
	}
	
	return Status{
		Available: false,
		Mode:      "offline",
		Message:   "No handler configured",
	}
}

// SetHandler updates the AI handler
func (d *BasicDispatcher) SetHandler(handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handler = handler
}