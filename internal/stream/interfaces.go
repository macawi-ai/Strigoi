package stream

import (
	"context"
	"time"
)

// StreamType defines the type of data stream
type StreamType string

const (
	StreamTypeSTDIO   StreamType = "stdio"   // Local process I/O
	StreamTypeRemote  StreamType = "remote"  // Remote agent streams
	StreamTypeSerial  StreamType = "serial"  // Serial port streams
	StreamTypeNetwork StreamType = "network" // Network protocol streams
)

// Priority levels for stream processing
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// StreamData represents data captured from a stream
type StreamData struct {
	ID        string
	Type      StreamType
	Source    string
	Timestamp time.Time
	Data      []byte
	Metadata  map[string]interface{}
}

// StreamHandler processes stream data
type StreamHandler interface {
	OnData(data StreamData) error
	GetID() string
	GetPriority() Priority
}

// Filter applies filtering rules to stream data
type Filter interface {
	Match(data []byte) bool
	GetName() string
	GetPriority() Priority
}

// StreamCapture manages stream data capture
type StreamCapture interface {
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
	GetID() string
	GetType() StreamType
	
	// Subscription
	Subscribe(handler StreamHandler) error
	Unsubscribe(handlerID string) error
	
	// Filtering
	AddFilter(filter Filter) error
	RemoveFilter(name string) error
	
	// Status
	IsActive() bool
	GetStats() StreamStats
}

// StreamStats provides stream statistics
type StreamStats struct {
	BytesProcessed uint64
	EventsCount    uint64
	StartTime      time.Time
	LastEventTime  time.Time
	ErrorCount     uint64
}

// StreamManager manages multiple streams
type StreamManager interface {
	// Stream lifecycle
	CreateStream(config StreamConfig) (StreamCapture, error)
	GetStream(id string) (StreamCapture, error)
	ListStreams() []StreamCapture
	DeleteStream(id string) error
	
	// Global operations
	StartAll(ctx context.Context) error
	StopAll() error
}

// StreamConfig configures a new stream
type StreamConfig struct {
	Type     StreamType
	Source   string // PID for STDIO, address for remote, device for serial
	BufferSize int
	Filters  []Filter
	Metadata map[string]interface{}
}

// RingBuffer provides efficient circular buffer for streams
type RingBuffer interface {
	Write(data []byte) (int, error)
	Read(p []byte) (int, error)
	ReadAt(p []byte, offset int64) (int, error)
	Size() int
	Capacity() int
	Reset()
}

// ProcessingStage represents a stage in the hierarchical pipeline
type ProcessingStage string

const (
	StageS1Edge    ProcessingStage = "s1_edge"    // Microsecond filtering
	StageS2Shallow ProcessingStage = "s2_shallow" // Millisecond analysis
	StageS3Deep    ProcessingStage = "s3_deep"    // Second-level analysis
)

// ProcessingPipeline manages hierarchical stream processing
type ProcessingPipeline interface {
	// Pipeline stages
	AddStage(stage ProcessingStage, processor StageProcessor) error
	RemoveStage(stage ProcessingStage) error
	
	// Processing
	Process(ctx context.Context, data StreamData) (*ProcessingResult, error)
	
	// Metrics
	GetStageMetrics(stage ProcessingStage) StageMetrics
}

// StageProcessor processes data at a specific pipeline stage
type StageProcessor interface {
	Process(ctx context.Context, data StreamData) (*StageResult, error)
	GetCapabilities() []string
	GetStage() ProcessingStage
}

// StageResult contains results from a processing stage
type StageResult struct {
	Stage      ProcessingStage
	Passed     bool // Whether to continue to next stage
	Confidence float64
	Findings   []Finding
	Metrics    StageMetrics
}

// Finding represents a security finding
type Finding struct {
	Type       string
	Severity   string
	Confidence float64
	Details    map[string]interface{}
}

// ProcessingResult aggregates all stage results
type ProcessingResult struct {
	StreamID     string
	Timestamp    time.Time
	StageResults []StageResult
	FinalAction  Action
	TotalLatency time.Duration
}

// Action represents the response action
type Action struct {
	Type     string // block, alert, allow, redirect
	Details  map[string]interface{}
	Executor string // Which component should execute
}

// StageMetrics tracks performance metrics
type StageMetrics struct {
	ProcessedCount uint64
	AvgLatency     time.Duration
	MaxLatency     time.Duration
	ErrorCount     uint64
	LastProcessed  time.Time
}