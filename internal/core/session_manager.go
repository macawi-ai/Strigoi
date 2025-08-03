package core

import (
	"fmt"
	"sync"
	"time"
	"crypto/rand"
	"encoding/hex"
)

// SessionManager manages active sessions and jobs
type SessionManager struct {
	sessions      map[string]*Session
	jobs          map[string]*Job
	results       []*ModuleResult
	currentModule Module
	mu            sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		jobs:     make(map[string]*Job),
		results:  make([]*ModuleResult, 0),
	}
}

// CreateSession creates a new session
func (sm *SessionManager) CreateSession(module string, target string, options map[string]interface{}) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session := &Session{
		ID:        generateID(),
		Module:    module,
		Target:    target,
		Status:    "active",
		StartTime: time.Now(),
		Options:   options,
	}
	
	sm.sessions[session.ID] = session
	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(id string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	session, exists := sm.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	
	return session, nil
}

// GetSessions returns all sessions
func (sm *SessionManager) GetSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}
	
	return sessions
}

// CloseSession closes a session
func (sm *SessionManager) CloseSession(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session, exists := sm.sessions[id]
	if !exists {
		return fmt.Errorf("session not found: %s", id)
	}
	
	session.EndTime = time.Now()
	session.Status = "closed"
	
	return nil
}

// CreateJob creates a new background job
func (sm *SessionManager) CreateJob(jobType string, module string) (*Job, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	job := &Job{
		ID:        generateID(),
		Type:      jobType,
		Module:    module,
		Status:    "running",
		Progress:  0,
		Started:   time.Now(),
	}
	
	sm.jobs[job.ID] = job
	return job, nil
}

// GetJobs returns all jobs
func (sm *SessionManager) GetJobs() []*Job {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	jobs := make([]*Job, 0, len(sm.jobs))
	for _, job := range sm.jobs {
		jobs = append(jobs, job)
	}
	
	return jobs
}

// UpdateJobProgress updates job progress
func (sm *SessionManager) UpdateJobProgress(id string, progress int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	job, exists := sm.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}
	
	job.Progress = progress
	if progress >= 100 {
		job.Status = "completed"
		// Job completed
	}
	
	return nil
}

// SetCurrentModule sets the currently active module
func (sm *SessionManager) SetCurrentModule(module Module) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.currentModule = module
}

// GetCurrentModule returns the currently active module
func (sm *SessionManager) GetCurrentModule() Module {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentModule
}

// ClearCurrentModule clears the currently active module
func (sm *SessionManager) ClearCurrentModule() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.currentModule = nil
}

// AddResult stores a module execution result
func (sm *SessionManager) AddResult(result *ModuleResult) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.results = append(sm.results, result)
}

// GetResults returns all stored results
func (sm *SessionManager) GetResults() []*ModuleResult {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	results := make([]*ModuleResult, len(sm.results))
	copy(results, sm.results)
	return results
}

// generateID generates a random session/job ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}