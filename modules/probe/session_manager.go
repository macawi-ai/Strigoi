package probe

import (
	"fmt"
	"sync"
	"time"
)

// Session represents a collection of related frames in a protocol session.
type Session struct {
	ID         string
	Protocol   string
	Frames     []*Frame
	StartTime  time.Time
	LastActive time.Time
	State      map[string]interface{} // For tracking session state

	// Network metadata
	SourceIP   string
	DestIP     string
	SourcePort int
	DestPort   int

	// Session relationships
	RelatedSessionIDs []string

	// Vulnerabilities found in this session
	Vulnerabilities []SessionVulnerability

	// Mutex for thread-safe operations
	mu sync.Mutex
}

// SessionVulnerability represents a vulnerability that spans multiple frames.
type SessionVulnerability struct {
	Type        string   // e.g., "session_hijacking", "token_reuse", "session_fixation"
	Severity    string   // critical, high, medium, low
	Confidence  float64  // 0.0 to 1.0
	Evidence    string   // What was found
	FrameIDs    []string // Which frames are involved
	Timestamp   time.Time
	Description string // Detailed description
}

// SessionManager manages active sessions across all protocols.
type SessionManager struct {
	sessions              sync.Map // map[string]*Session
	timeout               time.Duration
	cleanupTick           time.Duration
	stopCleanup           chan bool
	sessionCompleted      chan *Session
	vulnerabilityCheckers []SessionVulnerabilityChecker
	mu                    sync.RWMutex // Protects vulnerabilityCheckers
}

// SessionVulnerabilityChecker checks for session-level vulnerabilities.
type SessionVulnerabilityChecker interface {
	CheckSession(session *Session) []SessionVulnerability
}

// NewSessionManager creates a new session manager.
func NewSessionManager(timeout, cleanupInterval time.Duration) *SessionManager {
	sm := &SessionManager{
		timeout:               timeout,
		cleanupTick:           cleanupInterval,
		stopCleanup:           make(chan bool),
		sessionCompleted:      make(chan *Session, 1000), // Larger buffer for load testing
		vulnerabilityCheckers: []SessionVulnerabilityChecker{},
	}

	// Start cleanup goroutine
	go sm.cleanupExpiredSessions()

	return sm
}

// AddFrame adds a frame to a session, creating the session if necessary.
func (sm *SessionManager) AddFrame(sessionID string, protocol string, frame *Frame) error {
	if sessionID == "" {
		return fmt.Errorf("empty session ID")
	}

	now := time.Now()

	// Load or create session
	value, loaded := sm.sessions.LoadOrStore(sessionID, &Session{
		ID:                sessionID,
		Protocol:          protocol,
		Frames:            []*Frame{},
		StartTime:         now,
		LastActive:        now,
		State:             make(map[string]interface{}),
		RelatedSessionIDs: []string{},
		Vulnerabilities:   []SessionVulnerability{},
	})

	session := value.(*Session)

	// Thread-safe update of session
	if loaded {
		// Session exists, need to safely append frame
		session.mu.Lock()
		session.Frames = append(session.Frames, frame)
		session.LastActive = now
		session.mu.Unlock()
	} else {
		// New session, we own it
		session.Frames = append(session.Frames, frame)

		// Extract network metadata from frame if available
		sm.extractNetworkMetadata(session, frame)
	}

	// Check if session is complete (protocol-specific logic)
	if sm.isSessionComplete(session) {
		sm.CompleteSession(sessionID)
	}

	return nil
}

// GetSession retrieves a session by ID.
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	value, exists := sm.sessions.Load(sessionID)
	if !exists {
		return nil, false
	}
	return value.(*Session), true
}

// GetActiveSessions returns all active sessions.
func (sm *SessionManager) GetActiveSessions() []*Session {
	var sessions []*Session
	sm.sessions.Range(func(_, value interface{}) bool {
		sessions = append(sessions, value.(*Session))
		return true
	})
	return sessions
}

// CompleteSession marks a session as complete and triggers vulnerability checks.
func (sm *SessionManager) CompleteSession(sessionID string) {
	value, loaded := sm.sessions.LoadAndDelete(sessionID)
	if loaded {
		session := value.(*Session)

		// Run vulnerability checks
		sm.checkSessionVulnerabilities(session)

		// Send to completion channel
		select {
		case sm.sessionCompleted <- session:
		default:
			// Channel full, log warning
			fmt.Printf("Warning: Session completion channel full for session %s\n", sessionID)
		}
	}
}

// RegisterVulnerabilityChecker adds a vulnerability checker.
func (sm *SessionManager) RegisterVulnerabilityChecker(checker SessionVulnerabilityChecker) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.vulnerabilityCheckers = append(sm.vulnerabilityCheckers, checker)
}

// GetCompletedSessionChannel returns the channel for completed sessions.
func (sm *SessionManager) GetCompletedSessionChannel() <-chan *Session {
	return sm.sessionCompleted
}

// Stop gracefully shuts down the session manager.
func (sm *SessionManager) Stop() {
	close(sm.stopCleanup)
	close(sm.sessionCompleted)
}

// cleanupExpiredSessions runs periodically to remove expired sessions.
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(sm.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			var expiredIDs []string

			// Find expired sessions
			sm.sessions.Range(func(key, value interface{}) bool {
				session := value.(*Session)
				if now.Sub(session.LastActive) > sm.timeout {
					expiredIDs = append(expiredIDs, key.(string))
				}
				return true
			})

			// Complete expired sessions
			for _, id := range expiredIDs {
				sm.CompleteSession(id)
			}

		case <-sm.stopCleanup:
			return
		}
	}
}

// checkSessionVulnerabilities runs all registered vulnerability checkers.
func (sm *SessionManager) checkSessionVulnerabilities(session *Session) {
	sm.mu.RLock()
	checkers := sm.vulnerabilityCheckers
	sm.mu.RUnlock()

	for _, checker := range checkers {
		vulns := checker.CheckSession(session)
		session.Vulnerabilities = append(session.Vulnerabilities, vulns...)
	}
}

// extractNetworkMetadata extracts network information from the first frame.
func (sm *SessionManager) extractNetworkMetadata(session *Session, frame *Frame) {
	// Extract IPs and ports from frame fields if available
	if srcIP, ok := frame.Fields["source_ip"].(string); ok {
		session.SourceIP = srcIP
	}
	if dstIP, ok := frame.Fields["dest_ip"].(string); ok {
		session.DestIP = dstIP
	}
	if srcPort, ok := frame.Fields["source_port"].(int); ok {
		session.SourcePort = srcPort
	}
	if dstPort, ok := frame.Fields["dest_port"].(int); ok {
		session.DestPort = dstPort
	}
}

// isSessionComplete checks if a session is complete based on protocol-specific logic.
func (sm *SessionManager) isSessionComplete(session *Session) bool {
	switch session.Protocol {
	case "HTTP":
		// HTTP session is complete when we have a request and response
		hasRequest := false
		hasResponse := false
		for _, frame := range session.Frames {
			if frameType, ok := frame.Fields["type"].(string); ok {
				if frameType == "request" {
					hasRequest = true
				} else if frameType == "response" {
					hasResponse = true
				}
			}
		}
		return hasRequest && hasResponse

	case "WebSocket":
		// WebSocket session is complete when we see a close frame
		for _, frame := range session.Frames {
			if opcode, ok := frame.Fields["opcode"].(string); ok {
				if opcode == "close" {
					return true
				}
			}
		}
		return false

	case "gRPC":
		// gRPC session is complete when we see end-of-stream
		for _, frame := range session.Frames {
			if frameType, ok := frame.Fields["frame_type"].(string); ok {
				if frameType == "RST_STREAM" || frameType == "GOAWAY" {
					return true
				}
			}
		}
		return false

	default:
		return false
	}
}
