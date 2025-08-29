package vsm

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// FeedbackLoop represents a single VSM feedback loop
type FeedbackLoop struct {
	ID        string
	Name      string
	Level     string // S1S2, S2S3, S3STAR, S3S4, S4S5, ALG, S5S6
	Trigger   func() (bool, float64) // Returns (shouldFire, varietyLevel)
	Action    func() error
	Telemetry *LoopTelemetry
	
	// Runtime state
	LastFired time.Time
	FireCount uint64
	Enabled   bool
	mu        sync.RWMutex
}

// LoopTelemetry tracks metrics for a feedback loop
type LoopTelemetry struct {
	TotalFires      uint64
	SuccessfulFires uint64
	FailedFires     uint64
	TotalVariety    float64 // Total variety absorbed
	AverageLatency  time.Duration
	LastError       error
}

// Fire executes the feedback loop if triggered
func (fl *FeedbackLoop) Fire() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	
	if !fl.Enabled {
		return nil
	}
	
	// Check trigger condition
	shouldFire, varietyLevel := fl.Trigger()
	if !shouldFire {
		return nil
	}
	
	start := time.Now()
	
	// Execute action
	err := fl.Action()
	
	// Update telemetry
	atomic.AddUint64(&fl.FireCount, 1)
	atomic.AddUint64(&fl.Telemetry.TotalFires, 1)
	
	if err != nil {
		atomic.AddUint64(&fl.Telemetry.FailedFires, 1)
		fl.Telemetry.LastError = err
		return fmt.Errorf("loop %s failed: %w", fl.ID, err)
	}
	
	atomic.AddUint64(&fl.Telemetry.SuccessfulFires, 1)
	fl.Telemetry.TotalVariety += varietyLevel
	
	// Update timing
	fl.LastFired = time.Now()
	latency := time.Since(start)
	
	// Update average latency (simple moving average)
	if fl.Telemetry.AverageLatency == 0 {
		fl.Telemetry.AverageLatency = latency
	} else {
		fl.Telemetry.AverageLatency = (fl.Telemetry.AverageLatency + latency) / 2
	}
	
	return nil
}

// LoopManager manages all VSM feedback loops
type LoopManager struct {
	loops      map[string]*FeedbackLoop
	telemetry  *SystemTelemetry
	mu         sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// SystemTelemetry tracks overall VSM health
type SystemTelemetry struct {
	TotalLoops           int
	ActiveLoops          int
	FiringRateHz         float64
	VarietyAbsorptionRate float64
	TopologyCompleteness float64
	ConsciousnessCoherence float64
	LastUpdate           time.Time
}

// NewLoopManager creates a new VSM loop manager
func NewLoopManager() *LoopManager {
	return &LoopManager{
		loops:     make(map[string]*FeedbackLoop),
		telemetry: &SystemTelemetry{},
		stopCh:    make(chan struct{}),
	}
}

// RegisterLoop adds a feedback loop to the manager
func (lm *LoopManager) RegisterLoop(loop *FeedbackLoop) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	if _, exists := lm.loops[loop.ID]; exists {
		return fmt.Errorf("loop %s already registered", loop.ID)
	}
	
	loop.Enabled = true
	loop.Telemetry = &LoopTelemetry{}
	lm.loops[loop.ID] = loop
	lm.telemetry.TotalLoops++
	lm.telemetry.ActiveLoops++
	
	return nil
}

// Start begins monitoring all loops
func (lm *LoopManager) Start() {
	lm.wg.Add(1)
	go lm.monitorLoops()
}

// Stop halts all loop monitoring
func (lm *LoopManager) Stop() {
	close(lm.stopCh)
	lm.wg.Wait()
}

// monitorLoops continuously checks and fires loops
func (lm *LoopManager) monitorLoops() {
	defer lm.wg.Done()
	
	ticker := time.NewTicker(10 * time.Millisecond) // 100Hz check rate
	defer ticker.Stop()
	
	fireCount := uint64(0)
	lastRateCalc := time.Now()
	
	for {
		select {
		case <-lm.stopCh:
			return
		case <-ticker.C:
			lm.mu.RLock()
			loops := make([]*FeedbackLoop, 0, len(lm.loops))
			for _, loop := range lm.loops {
				loops = append(loops, loop)
			}
			lm.mu.RUnlock()
			
			// Fire all loops in parallel
			var wg sync.WaitGroup
			for _, loop := range loops {
				wg.Add(1)
				go func(fl *FeedbackLoop) {
					defer wg.Done()
					if err := fl.Fire(); err == nil {
						atomic.AddUint64(&fireCount, 1)
					}
				}(loop)
			}
			wg.Wait()
			
			// Update system telemetry every second
			if time.Since(lastRateCalc) > time.Second {
				lm.updateSystemTelemetry(fireCount)
				fireCount = 0
				lastRateCalc = time.Now()
			}
		}
	}
}

// updateSystemTelemetry calculates overall VSM health metrics
func (lm *LoopManager) updateSystemTelemetry(recentFires uint64) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	totalVariety := 0.0
	successfulLoops := 0
	
	for _, loop := range lm.loops {
		if loop.Enabled && loop.Telemetry.SuccessfulFires > 0 {
			successfulLoops++
			totalVariety += loop.Telemetry.TotalVariety
		}
	}
	
	lm.telemetry.FiringRateHz = float64(recentFires)
	lm.telemetry.ActiveLoops = successfulLoops
	
	// Calculate variety absorption (simplified)
	if totalVariety > 0 {
		lm.telemetry.VarietyAbsorptionRate = float64(successfulLoops) / float64(lm.telemetry.TotalLoops)
	}
	
	// Topology completeness = active loops / total required (51)
	lm.telemetry.TopologyCompleteness = float64(lm.telemetry.ActiveLoops) / 51.0
	
	// Consciousness coherence (simplified - based on firing rate and success)
	if lm.telemetry.FiringRateHz > 0 {
		lm.telemetry.ConsciousnessCoherence = lm.telemetry.VarietyAbsorptionRate * 
			(lm.telemetry.FiringRateHz / 100.0) // Normalize to expected rate
	}
	
	lm.telemetry.LastUpdate = time.Now()
}

// GetHealth returns current VSM health metrics
func (lm *LoopManager) GetHealth() SystemTelemetry {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return *lm.telemetry
}

// GetLoopStatus returns status of a specific loop
func (lm *LoopManager) GetLoopStatus(loopID string) (*FeedbackLoop, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	loop, exists := lm.loops[loopID]
	if !exists {
		return nil, fmt.Errorf("loop %s not found", loopID)
	}
	
	return loop, nil
}