package stream

import (
    "sync"
)

// MemoryOutput stores events and alerts in memory
type MemoryOutput struct {
    mu     sync.Mutex
    events []*StreamEvent
    alerts []*SecurityAlert
}

// NewMemoryOutput creates a new memory output writer
func NewMemoryOutput() *MemoryOutput {
    return &MemoryOutput{
        events: make([]*StreamEvent, 0),
        alerts: make([]*SecurityAlert, 0),
    }
}

// WriteEvent stores an event in memory
func (m *MemoryOutput) WriteEvent(event *StreamEvent) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.events = append(m.events, event)
    return nil
}

// WriteAlert stores an alert in memory
func (m *MemoryOutput) WriteAlert(alert *SecurityAlert) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.alerts = append(m.alerts, alert)
    return nil
}

// Close is a no-op for memory output
func (m *MemoryOutput) Close() error {
    return nil
}

// GetEvents returns all stored events
func (m *MemoryOutput) GetEvents() []*StreamEvent {
    m.mu.Lock()
    defer m.mu.Unlock()
    result := make([]*StreamEvent, len(m.events))
    copy(result, m.events)
    return result
}

// GetAlerts returns all stored alerts
func (m *MemoryOutput) GetAlerts() []*SecurityAlert {
    m.mu.Lock()
    defer m.mu.Unlock()
    result := make([]*SecurityAlert, len(m.alerts))
    copy(result, m.alerts)
    return result
}

// Clear removes all stored events and alerts
func (m *MemoryOutput) Clear() {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.events = m.events[:0]
    m.alerts = m.alerts[:0]
}