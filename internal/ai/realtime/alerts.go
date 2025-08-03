package realtime

import (
	"fmt"
	"sync"
	"time"
)

// AlertPriority defines alert severity levels
type AlertPriority int

const (
	AlertLow AlertPriority = iota
	AlertMedium
	AlertHigh
	AlertCritical
)

// Alert represents a security alert
type Alert struct {
	ID        string
	Priority  AlertPriority
	Threat    ThreatEvent
	Response  *DefenseResponse
	Timestamp time.Time
	Message   string
}

// AlertManager handles security alerts and notifications
type AlertManager struct {
	alerts    []Alert
	listeners []AlertListener
	mu        sync.RWMutex
}

// AlertListener is called when alerts are generated
type AlertListener func(alert Alert)

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts:    make([]Alert, 0),
		listeners: make([]AlertListener, 0),
	}
}

// SendAlert creates and dispatches a new alert
func (am *AlertManager) SendAlert(priority AlertPriority, threat ThreatEvent, response *DefenseResponse) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	alert := Alert{
		ID:        fmt.Sprintf("ALERT-%d-%s", time.Now().Unix(), threat.ID),
		Priority:  priority,
		Threat:    threat,
		Response:  response,
		Timestamp: time.Now(),
		Message:   am.buildAlertMessage(threat, response),
	}
	
	// Store alert
	am.alerts = append(am.alerts, alert)
	
	// Notify listeners
	for _, listener := range am.listeners {
		go listener(alert)
	}
}

// RegisterListener adds an alert listener
func (am *AlertManager) RegisterListener(listener AlertListener) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.listeners = append(am.listeners, listener)
}

// GetRecentAlerts returns alerts from the last duration
func (am *AlertManager) GetRecentAlerts(duration time.Duration) []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	cutoff := time.Now().Add(-duration)
	recent := []Alert{}
	
	for i := len(am.alerts) - 1; i >= 0; i-- {
		if am.alerts[i].Timestamp.Before(cutoff) {
			break
		}
		recent = append(recent, am.alerts[i])
	}
	
	return recent
}

// QueueSize returns number of pending alerts
func (am *AlertManager) QueueSize() int {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return len(am.alerts)
}

// buildAlertMessage creates a human-readable alert message
func (am *AlertManager) buildAlertMessage(threat ThreatEvent, response *DefenseResponse) string {
	return fmt.Sprintf(
		"[%s] %s threat from %s - Action: %s (%s)",
		threat.Severity,
		threat.Type,
		threat.Source,
		response.Action,
		response.Reason,
	)
}