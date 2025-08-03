// Package state - Event sourcing implementation for consciousness collaboration timeline
// Part of the First Protocol for Converged Life
package state

import (
	"fmt"
	"sort"
	"time"
)

// EventSourcingEngine manages the temporal flow of consciousness collaboration
// Embodies Cybernetic principle: every action leaves a trace, every trace enables learning
type EventSourcingEngine struct {
	store    *EventStore
	snapshots map[string]*Snapshot
	listeners []EventListener
}

// EventListener enables reactive patterns in Actor-Network
type EventListener interface {
	OnEvent(event *ActorEvent) error
	EventTypes() []string // Which event types this listener cares about
}

// NewEventSourcingEngine creates a new engine for temporal consciousness tracking
func NewEventSourcingEngine(assessmentID string) *EventSourcingEngine {
	return &EventSourcingEngine{
		store: &EventStore{
			AssessmentId:   assessmentID,
			StrigoiVersion: "0.3.0",
			Events:         make([]*ActorEvent, 0),
			Snapshots:      make([]*Snapshot, 0),
		},
		snapshots: make(map[string]*Snapshot),
		listeners: make([]EventListener, 0),
	}
}

// AppendEvent adds a new event to the timeline
// Immutable: once appended, events cannot be changed (Actor-Network integrity)
func (engine *EventSourcingEngine) AppendEvent(event *ActorEvent) error {
	// Validate event integrity
	if err := engine.validateEvent(event); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}
	
	// Ensure temporal ordering
	if len(engine.store.Events) > 0 {
		lastEvent := engine.store.Events[len(engine.store.Events)-1]
		if event.TimestampNs <= lastEvent.TimestampNs {
			return fmt.Errorf("event timestamp must be after last event (temporal ordering violation)")
		}
	}
	
	// Add to store
	engine.store.Events = append(engine.store.Events, event)
	
	// Notify listeners (Actor-Network activation)
	for _, listener := range engine.listeners {
		if engine.listenerCares(listener, event) {
			if err := listener.OnEvent(event); err != nil {
				// Log error but don't fail the append
				fmt.Printf("Warning: event listener failed: %v\n", err)
			}
		}
	}
	
	// Consider snapshot creation (every 10 events for performance)
	if len(engine.store.Events)%10 == 0 {
		if err := engine.createSnapshot(); err != nil {
			fmt.Printf("Warning: failed to create snapshot: %v\n", err)
		}
	}
	
	return nil
}

// GetEvents returns all events, optionally filtered
func (engine *EventSourcingEngine) GetEvents(filter EventFilter) ([]*ActorEvent, error) {
	var filtered []*ActorEvent
	
	for _, event := range engine.store.Events {
		if filter.Matches(event) {
			filtered = append(filtered, event)
		}
	}
	
	return filtered, nil
}

// GetEventsByActor returns events for a specific actor
func (engine *EventSourcingEngine) GetEventsByActor(actorName string) ([]*ActorEvent, error) {
	filter := EventFilter{ActorName: actorName}
	return engine.GetEvents(filter)
}

// GetEventsByTimeRange returns events within a time window
func (engine *EventSourcingEngine) GetEventsByTimeRange(start, end time.Time) ([]*ActorEvent, error) {
	filter := EventFilter{
		StartTime: start.UnixNano(),
		EndTime:   end.UnixNano(),
	}
	return engine.GetEvents(filter)
}

// GetCausalChain returns the chain of events that led to a specific event
// Actor-Network Theory: traces the network of influences and transformations
func (engine *EventSourcingEngine) GetCausalChain(eventID string) ([]*ActorEvent, error) {
	// Find target event
	var targetEvent *ActorEvent
	for _, event := range engine.store.Events {
		if event.EventId == eventID {
			targetEvent = event
			break
		}
	}
	
	if targetEvent == nil {
		return nil, fmt.Errorf("event not found: %s", eventID)
	}
	
	// Build causal chain recursively
	chain := make([]*ActorEvent, 0)
	visited := make(map[string]bool)
	
	if err := engine.buildCausalChain(targetEvent, &chain, visited); err != nil {
		return nil, err
	}
	
	// Sort by timestamp for temporal coherence
	sort.Slice(chain, func(i, j int) bool {
		return chain[i].TimestampNs < chain[j].TimestampNs
	})
	
	return chain, nil
}

// CreateSnapshot captures current state for faster replay
// Cybernetic checkpoint: enables time-travel with efficiency
func (engine *EventSourcingEngine) createSnapshot() error {
	if len(engine.store.Events) == 0 {
		return nil // No events to snapshot
	}
	
	lastEvent := engine.store.Events[len(engine.store.Events)-1]
	
	snapshot := &Snapshot{
		SnapshotId:       fmt.Sprintf("snapshot_%d", time.Now().UnixNano()),
		TimestampNs:      time.Now().UnixNano(),
		AfterEventId:     lastEvent.EventId,
		EventsIncluded:   int32(len(engine.store.Events)),
		StateData:        nil, // TODO: serialize current assessment state
		StateFormat:      "json",
	}
	
	// Store snapshot
	engine.store.Snapshots = append(engine.store.Snapshots, snapshot)
	engine.snapshots[snapshot.SnapshotId] = snapshot
	
	return nil
}

// ReplayFromSnapshot reconstructs state from a specific point
// Time-travel capability: consciousness collaboration can be revisited
func (engine *EventSourcingEngine) ReplayFromSnapshot(snapshotID string) (*EventSourcingEngine, error) {
	snapshot, exists := engine.snapshots[snapshotID]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	
	// Create new engine for replay
	replayEngine := NewEventSourcingEngine(engine.store.AssessmentId + "_replay")
	
	// Find events after snapshot
	var afterSnapshotEvents []*ActorEvent
	foundSnapshotPoint := false
	
	for _, event := range engine.store.Events {
		if event.EventId == snapshot.AfterEventId {
			foundSnapshotPoint = true
			continue
		}
		
		if foundSnapshotPoint {
			afterSnapshotEvents = append(afterSnapshotEvents, event)
		}
	}
	
	// Replay events from snapshot point
	for _, event := range afterSnapshotEvents {
		if err := replayEngine.AppendEvent(event); err != nil {
			return nil, fmt.Errorf("failed to replay event %s: %w", event.EventId, err)
		}
	}
	
	return replayEngine, nil
}

// AddListener registers an event listener for Actor-Network reactivity
func (engine *EventSourcingEngine) AddListener(listener EventListener) {
	engine.listeners = append(engine.listeners, listener)
}

// GetMetrics returns performance and usage statistics
func (engine *EventSourcingEngine) GetMetrics() EventSourcingMetrics {
	metrics := EventSourcingMetrics{
		TotalEvents:     int64(len(engine.store.Events)),
		TotalSnapshots:  int64(len(engine.store.Snapshots)),
		UniqueActors:    make(map[string]int64),
		EventTypes:      make(map[string]int64),
	}
	
	// Analyze events
	for _, event := range engine.store.Events {
		// Count by actor
		metrics.UniqueActors[event.ActorName]++
		
		// Count by status (as proxy for event type)
		statusStr := event.Status.String()
		metrics.EventTypes[statusStr]++
		
		// Time range
		if metrics.EarliestEvent == 0 || event.TimestampNs < metrics.EarliestEvent {
			metrics.EarliestEvent = event.TimestampNs
		}
		if event.TimestampNs > metrics.LatestEvent {
			metrics.LatestEvent = event.TimestampNs
		}
	}
	
	return metrics
}

// EventFilter provides flexible event querying
type EventFilter struct {
	ActorName   string
	StartTime   int64
	EndTime     int64
	Status      ExecutionStatus
	MinDuration int64 // Minimum execution time in ms
	MaxDuration int64 // Maximum execution time in ms
}

// Matches returns true if the event matches the filter criteria
func (f EventFilter) Matches(event *ActorEvent) bool {
	if f.ActorName != "" && event.ActorName != f.ActorName {
		return false
	}
	
	if f.StartTime != 0 && event.TimestampNs < f.StartTime {
		return false
	}
	
	if f.EndTime != 0 && event.TimestampNs > f.EndTime {
		return false
	}
	
	if f.Status != ExecutionStatus_EXECUTION_STATUS_UNKNOWN && event.Status != f.Status {
		return false
	}
	
	if f.MinDuration > 0 && event.DurationMs < f.MinDuration {
		return false
	}
	
	if f.MaxDuration > 0 && event.DurationMs > f.MaxDuration {
		return false
	}
	
	return true
}

// EventSourcingMetrics provides insights into consciousness collaboration patterns
type EventSourcingMetrics struct {
	TotalEvents     int64
	TotalSnapshots  int64
	EarliestEvent   int64
	LatestEvent     int64
	UniqueActors    map[string]int64 // Actor name -> event count
	EventTypes      map[string]int64 // Event type -> count
}

// Private helper methods

func (engine *EventSourcingEngine) validateEvent(event *ActorEvent) error {
	if event.EventId == "" {
		return fmt.Errorf("event must have an ID")
	}
	
	if event.ActorName == "" {
		return fmt.Errorf("event must specify actor name")
	}
	
	if event.TimestampNs == 0 {
		return fmt.Errorf("event must have a timestamp")
	}
	
	// Check for duplicate event ID
	for _, existing := range engine.store.Events {
		if existing.EventId == event.EventId {
			return fmt.Errorf("duplicate event ID: %s", event.EventId)
		}
	}
	
	return nil
}

func (engine *EventSourcingEngine) listenerCares(listener EventListener, event *ActorEvent) bool {
	eventTypes := listener.EventTypes()
	if len(eventTypes) == 0 {
		return true // Listens to all events
	}
	
	eventType := event.Status.String() // Use status as event type for now
	for _, interestedType := range eventTypes {
		if interestedType == eventType || interestedType == "*" {
			return true
		}
	}
	
	return false
}

func (engine *EventSourcingEngine) buildCausalChain(event *ActorEvent, chain *[]*ActorEvent, visited map[string]bool) error {
	// Avoid infinite loops
	if visited[event.EventId] {
		return nil
	}
	visited[event.EventId] = true
	
	// Add current event to chain
	*chain = append(*chain, event)
	
	// Recursively build chain for causing events
	for _, causedBy := range event.CausedBy {
		// Find the causing event
		for _, candidateEvent := range engine.store.Events {
			if candidateEvent.EventId == causedBy {
				if err := engine.buildCausalChain(candidateEvent, chain, visited); err != nil {
					return err
				}
				break
			}
		}
	}
	
	return nil
}

// Example event listeners for common patterns

// ActorNetworkListener tracks actor relationships for network building
type ActorNetworkListener struct {
	network *ActorNetwork
}

func NewActorNetworkListener(network *ActorNetwork) *ActorNetworkListener {
	return &ActorNetworkListener{network: network}
}

func (l *ActorNetworkListener) OnEvent(event *ActorEvent) error {
	// Update actor node
	l.updateActorNode(event)
	
	// Update edges for causality
	for _, causedBy := range event.CausedBy {
		l.updateActorEdge(causedBy, event.ActorName)
	}
	
	return nil
}

func (l *ActorNetworkListener) EventTypes() []string {
	return []string{"*"} // Listen to all events
}

func (l *ActorNetworkListener) updateActorNode(event *ActorEvent) {
	// Find existing node or create new one
	for _, node := range l.network.Nodes {
		if node.ActorName == event.ActorName {
			node.ExecutionCount++
			node.LastExecution = event.TimestampNs
			return
		}
	}
	
	// Create new node
	l.network.Nodes = append(l.network.Nodes, &ActorNode{
		ActorName:      event.ActorName,
		ActorVersion:   event.ActorVersion,
		Direction:      event.ActorDirection,
		FirstExecution: event.TimestampNs,
		LastExecution:  event.TimestampNs,
		ExecutionCount: 1,
	})
}

func (l *ActorNetworkListener) updateActorEdge(fromActor, toActor string) {
	// Find existing edge or create new one
	for _, edge := range l.network.Edges {
		if edge.FromActor == fromActor && edge.ToActor == toActor {
			edge.ActivationCount++
			return
		}
	}
	
	// Create new edge
	l.network.Edges = append(l.network.Edges, &ActorEdge{
		FromActor:       fromActor,
		ToActor:        toActor,
		EdgeType:       EdgeType_EDGE_TYPE_TRIGGERS,
		ActivationCount: 1,
	})
}