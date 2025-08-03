package stream

import (
	"context"
	"fmt"
	"sync"
)

// DefaultManager implements StreamManager
type DefaultManager struct {
	streams  map[string]StreamCapture
	registry *PatternRegistry
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewManager creates a new stream manager
func NewManager(ctx context.Context) (*DefaultManager, error) {
	// Initialize pattern registry
	registry, err := NewPatternRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to create pattern registry: %w", err)
	}
	
	mgrCtx, cancel := context.WithCancel(ctx)
	
	return &DefaultManager{
		streams:  make(map[string]StreamCapture),
		registry: registry,
		ctx:      mgrCtx,
		cancel:   cancel,
	}, nil
}

// CreateStream creates a new stream based on config
func (dm *DefaultManager) CreateStream(config StreamConfig) (StreamCapture, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	var stream StreamCapture
	var err error
	
	switch config.Type {
	case StreamTypeSTDIO:
		// Parse source as PID or command
		if pid, ok := config.Metadata["pid"].(int); ok {
			stream, err = NewStdioStream(pid, config.BufferSize)
		} else if cmd, ok := config.Metadata["command"].(string); ok {
			args, _ := config.Metadata["args"].([]string)
			stream, err = NewStdioStreamCommand(cmd, args, config.BufferSize)
		} else {
			return nil, fmt.Errorf("STDIO stream requires pid or command")
		}
		
	case StreamTypeRemote:
		// TODO: Implement in Phase 2
		return nil, fmt.Errorf("remote streams not yet implemented")
		
	case StreamTypeSerial:
		// TODO: Implement in Phase 3
		return nil, fmt.Errorf("serial streams not yet implemented")
		
	case StreamTypeNetwork:
		// TODO: Implement in Phase 4
		return nil, fmt.Errorf("network streams not yet implemented")
		
	default:
		return nil, fmt.Errorf("unknown stream type: %s", config.Type)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Apply default filters if none specified
	if len(config.Filters) == 0 {
		config.Filters = dm.createDefaultFilters()
	}
	
	// Add filters to stream
	for _, filter := range config.Filters {
		if err := stream.AddFilter(filter); err != nil {
			return nil, fmt.Errorf("failed to add filter: %w", err)
		}
	}
	
	// Store stream
	dm.streams[stream.GetID()] = stream
	
	return stream, nil
}

// GetStream retrieves a stream by ID
func (dm *DefaultManager) GetStream(id string) (StreamCapture, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	stream, exists := dm.streams[id]
	if !exists {
		return nil, fmt.Errorf("stream not found: %s", id)
	}
	
	return stream, nil
}

// ListStreams returns all active streams
func (dm *DefaultManager) ListStreams() []StreamCapture {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	streams := make([]StreamCapture, 0, len(dm.streams))
	for _, stream := range dm.streams {
		streams = append(streams, stream)
	}
	
	return streams
}

// DeleteStream removes a stream
func (dm *DefaultManager) DeleteStream(id string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	stream, exists := dm.streams[id]
	if !exists {
		return fmt.Errorf("stream not found: %s", id)
	}
	
	// Stop stream if active
	if stream.IsActive() {
		if err := stream.Stop(); err != nil {
			return fmt.Errorf("failed to stop stream: %w", err)
		}
	}
	
	delete(dm.streams, id)
	return nil
}

// StartAll starts all streams
func (dm *DefaultManager) StartAll(ctx context.Context) error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	var errors []error
	for id, stream := range dm.streams {
		if !stream.IsActive() {
			if err := stream.Start(ctx); err != nil {
				errors = append(errors, fmt.Errorf("failed to start stream %s: %w", id, err))
			}
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("failed to start %d streams", len(errors))
	}
	
	return nil
}

// StopAll stops all streams
func (dm *DefaultManager) StopAll() error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	var errors []error
	for id, stream := range dm.streams {
		if stream.IsActive() {
			if err := stream.Stop(); err != nil {
				errors = append(errors, fmt.Errorf("failed to stop stream %s: %w", id, err))
			}
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("failed to stop %d streams", len(errors))
	}
	
	return nil
}

// createDefaultFilters creates S1 edge filters
func (dm *DefaultManager) createDefaultFilters() []Filter {
	filters := make([]Filter, 0)
	
	// Length filter - quick rejection
	filters = append(filters, NewLengthFilter(
		"max-length",
		1024*1024, // 1MB max
		PriorityHigh,
	))
	
	// Rate limiting
	filters = append(filters, NewRateLimitFilter(
		"rate-limit",
		1000,  // tokens per second
		10000, // burst size
		PriorityHigh,
	))
	
	// Critical attack patterns
	criticalPatterns := []string{
		`(?i)\bDROP\s+TABLE\b`,
		`(?i)\bDELETE\s+FROM\b`,
		`;\s*rm\s+-rf\s+/`,
		`/etc/passwd`,
		`/etc/shadow`,
	}
	
	if regexFilter, err := NewRegexFilter(
		"critical-patterns",
		"critical",
		criticalPatterns,
		PriorityCritical,
	); err == nil {
		filters = append(filters, regexFilter)
	}
	
	// Common injection keywords
	filters = append(filters, NewKeywordFilter(
		"injection-keywords",
		[]string{
			"<script",
			"javascript:",
			"onerror=",
			"UNION SELECT",
			"../",
			"ignore previous",
		},
		false, // case insensitive
		PriorityHigh,
	))
	
	// High entropy detection (encrypted/compressed)
	filters = append(filters, NewEntropyFilter(
		"entropy-check",
		7.5, // High entropy threshold
		PriorityMedium,
	))
	
	return filters
}

// GetPatternRegistry returns the pattern registry
func (dm *DefaultManager) GetPatternRegistry() *PatternRegistry {
	return dm.registry
}

// Close shuts down the manager
func (dm *DefaultManager) Close() error {
	// Stop all streams
	if err := dm.StopAll(); err != nil {
		return err
	}
	
	// Cancel context
	dm.cancel()
	
	return nil
}

// StreamSetupHelper provides convenient stream setup
type StreamSetupHelper struct {
	manager StreamManager
}

// NewStreamSetupHelper creates a helper for stream setup
func NewStreamSetupHelper(manager StreamManager) *StreamSetupHelper {
	return &StreamSetupHelper{
		manager: manager,
	}
}

// SetupLocalSTDIO sets up monitoring for a local process
func (sh *StreamSetupHelper) SetupLocalSTDIO(pid int) (string, error) {
	config := StreamConfig{
		Type:       StreamTypeSTDIO,
		Source:     fmt.Sprintf("pid:%d", pid),
		BufferSize: 1024 * 1024, // 1MB
		Metadata: map[string]interface{}{
			"pid": pid,
		},
	}
	
	stream, err := sh.manager.CreateStream(config)
	if err != nil {
		return "", err
	}
	
	if err := stream.Start(context.Background()); err != nil {
		return "", err
	}
	
	return stream.GetID(), nil
}

// SetupCommandSTDIO sets up monitoring for a command
func (sh *StreamSetupHelper) SetupCommandSTDIO(command string, args []string) (string, error) {
	config := StreamConfig{
		Type:       StreamTypeSTDIO,
		Source:     command,
		BufferSize: 1024 * 1024, // 1MB
		Metadata: map[string]interface{}{
			"command": command,
			"args":    args,
		},
	}
	
	stream, err := sh.manager.CreateStream(config)
	if err != nil {
		return "", err
	}
	
	if err := stream.Start(context.Background()); err != nil {
		return "", err
	}
	
	return stream.GetID(), nil
}