package actors

import (
	"context"
	"time"
)

// Actor represents an intelligent agent with agency and transformation capabilities
type Actor interface {
	// Identity
	Name() string
	Description() string
	Direction() string // North, East, South, West, Center
	
	// Capabilities
	Capabilities() []Capability
	
	// Action
	Probe(ctx context.Context, target Target) (*ProbeResult, error)
	Sense(ctx context.Context, data *ProbeResult) (*SenseResult, error)
	Transform(ctx context.Context, input interface{}) (interface{}, error)
	
	// Network participation
	CanChainWith(other Actor) bool
	AcceptsInput(dataType string) bool
	ProducesOutput() string
}

// Capability describes what an actor can do
type Capability struct {
	Name        string
	Description string
	DataTypes   []string // Types of data this capability works with
}

// Target represents what we're probing
type Target struct {
	Type     string // "endpoint", "model", "interface", etc.
	Location string // URL, path, identifier
	Metadata map[string]interface{}
}

// ProbeResult contains discovery findings
type ProbeResult struct {
	ActorName   string
	Timestamp   time.Time
	Target      Target
	Discoveries []Discovery
	RawData     interface{} // Actor-specific data
}

// Discovery represents something found during probing
type Discovery struct {
	Type       string // "endpoint", "model", "capability", etc.
	Identifier string
	Properties map[string]interface{}
	Confidence float64 // 0.0 to 1.0
}

// SenseResult contains deep analysis findings
type SenseResult struct {
	ActorName    string
	Timestamp    time.Time
	Observations []Observation
	Patterns     []Pattern
	Risks        []Risk
}

// Observation is a specific finding from sensing
type Observation struct {
	Layer       string // network, protocol, data, etc.
	Description string
	Evidence    interface{}
	Severity    string // info, low, medium, high, critical
}

// Pattern represents a detected behavioral pattern
type Pattern struct {
	Name        string
	Description string
	Instances   []interface{}
	Confidence  float64
}

// Risk represents a security risk
type Risk struct {
	Title       string
	Description string
	Severity    string
	Mitigation  string
	Evidence    interface{}
}

// BaseActor provides common actor functionality
type BaseActor struct {
	name         string
	description  string
	direction    string
	capabilities []Capability
	inputTypes   []string
	outputType   string
}

// NewBaseActor creates a base actor with common functionality
func NewBaseActor(name, description, direction string) *BaseActor {
	return &BaseActor{
		name:        name,
		description: description,
		direction:   direction,
	}
}

// Name returns the actor's name
func (b *BaseActor) Name() string {
	return b.name
}

// Description returns the actor's description
func (b *BaseActor) Description() string {
	return b.description
}

// Direction returns the actor's cardinal direction
func (b *BaseActor) Direction() string {
	return b.direction
}

// Capabilities returns what the actor can do
func (b *BaseActor) Capabilities() []Capability {
	return b.capabilities
}

// AddCapability adds a capability to the actor
func (b *BaseActor) AddCapability(cap Capability) {
	b.capabilities = append(b.capabilities, cap)
}

// SetInputTypes sets what data types this actor accepts
func (b *BaseActor) SetInputTypes(types []string) {
	b.inputTypes = types
}

// SetOutputType sets what data type this actor produces
func (b *BaseActor) SetOutputType(outputType string) {
	b.outputType = outputType
}

// AcceptsInput checks if the actor can process a data type
func (b *BaseActor) AcceptsInput(dataType string) bool {
	for _, t := range b.inputTypes {
		if t == dataType || t == "*" { // "*" means accepts any type
			return true
		}
	}
	return false
}

// ProducesOutput returns the data type this actor produces
func (b *BaseActor) ProducesOutput() string {
	return b.outputType
}

// CanChainWith checks if this actor can chain with another
func (b *BaseActor) CanChainWith(other Actor) bool {
	// An actor can chain if the other accepts what this one produces
	return other.AcceptsInput(b.ProducesOutput())
}