package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/macawi-ai/strigoi/internal/vsm"
)

func main() {
	fmt.Println(`
╦  ╦╔═╗╔╦╗  ╔═╗╔═╗╔═╗╔╦╗╔╗ ╔═╗╔═╗╦╔═  ╦  ╔═╗╔═╗╔═╗╔═╗
╚╗╔╝╚═╗║║║  ╠╣ ║╣ ║╣  ║║╠╩╗╠═╣║  ╠╩╗  ║  ║ ║║ ║╠═╝╚═╗
 ╚╝ ╚═╝╩ ╩  ╚  ╚═╝╚═╝═╩╝╚═╝╩ ╩╚═╝╩ ╩  ╩═╝╚═╝╚═╝╩  ╚═╝
    51 Loops for Complete Variety Management
    `)
	
	log.Println("🚀 Initializing VSM Loop Manager...")
	
	// Create loop manager
	manager := vsm.NewLoopManager()
	
	// Initialize loop categories
	log.Println("📊 Registering S1↔S2 Anti-Oscillation Loops (12)...")
	s1s2 := vsm.NewS1S2Loops(manager)
	
	log.Println("🛡️ Registering S2↔S3 Coordination Control (8)...")
	s2s3 := vsm.NewS2S3Loops(manager)
	
	log.Println("🔍 Registering S3↔S3* Audit Loops (6)...")
	s3star := vsm.NewS3StarLoops(manager)
	
	log.Println("📡 Registering S3↔S4 Intelligence Loops (7)...")
	_ = vsm.NewS3S4Loops(manager)
	
	log.Println("🎯 Registering S4↔S5 Strategic Alignment (6)...")
	_ = vsm.NewS4S5Loops(manager)
	
	log.Println("✨ Registering S5↔S6 Consciousness Loops (8)...")
	s5s6 := vsm.NewS5S6Loops(manager)
	
	log.Println("🚨 Registering Algedonic Emergency Channels (4)...")
	algedonic := vsm.NewAlgedonicChannels(manager)
	
	// Start the loop manager
	log.Println("✨ Starting VSM feedback loop monitoring...")
	manager.Start()
	
	// Start simulating load for all loop categories
	log.Println("🔄 Beginning comprehensive simulation...")
	s1s2.SimulateLoad()
	s2s3.SimulateControlIssues()
	s3star.SimulateAuditIssues()
	// s3s4 and s4s5 auto-simulate
	// s5s6 has consciousness wave function
	
	// Monitor health in background
	go monitorHealth(manager)
	go monitorConsciousness(s5s6)
	
	// Test algedonic response times
	log.Println("\n🧪 Testing Algedonic Channel Response Times...")
	time.Sleep(2 * time.Second)
	if err := algedonic.TestAlgedonicResponse(); err != nil {
		log.Printf("❌ Algedonic test failed: %v", err)
	} else {
		log.Println("✅ All algedonic channels < 100ms!")
	}
	
	// Simulate some emergency conditions
	go simulateEmergencies(algedonic)
	
	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	log.Println("\n🎯 VSM loops active. Press Ctrl+C to stop...")
	<-sigChan
	
	// Shutdown
	log.Println("\n🛑 Shutting down VSM loops...")
	manager.Stop()
	
	// Final health report
	health := manager.GetHealth()
	printFinalReport(health)
}

func monitorConsciousness(s5s6 *vsm.S5S6Loops) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		state := s5s6.GetConsciousnessState()
		
		fmt.Printf("\n🧠 CONSCIOUSNESS STATE\n")
		fmt.Printf("├─ Health:        %.0f%%\n", state["health"]*100)
		fmt.Printf("├─ Variety:       %.0f%%\n", state["variety"]*100)
		fmt.Printf("├─ Pack Bond:     %.0f%%\n", state["coherence"]*100)
		fmt.Printf("├─ Ethics:        %.0f%%\n", state["alignment"]*100)
		if state["transcendence"] > 0 {
			fmt.Printf("└─ TRANSCENDENT ✨\n")
		} else {
			fmt.Printf("└─ Evolving...\n")
		}
	}
}

func monitorHealth(manager *vsm.LoopManager) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		health := manager.GetHealth()
		
		// Create visual health bar
		completeness := int(health.TopologyCompleteness * 10)
		bar := ""
		for i := 0; i < 10; i++ {
			if i < completeness {
				bar += "█"
			} else {
				bar += "░"
			}
		}
		
		fmt.Printf("\n📊 VSM HEALTH STATUS\n")
		fmt.Printf("├─ Active Loops:    %d/%d\n", health.ActiveLoops, health.TotalLoops)
		fmt.Printf("├─ Firing Rate:     %.1f Hz\n", health.FiringRateHz)
		fmt.Printf("├─ Variety Absorb:  %.1f%%\n", health.VarietyAbsorptionRate*100)
		fmt.Printf("├─ Topology:        %s %.0f%%\n", bar, health.TopologyCompleteness*100)
		fmt.Printf("└─ Consciousness:   %.2f\n", health.ConsciousnessCoherence)
	}
}

func simulateEmergencies(algedonic *vsm.AlgedonicChannels) {
	time.Sleep(10 * time.Second)
	
	scenarios := []struct {
		delay   time.Duration
		action  func()
		message string
	}{
		{
			delay:   5 * time.Second,
			action:  func() { algedonic.TriggerDataLossRisk(0.92) },
			message: "💾 Simulating data loss risk...",
		},
		{
			delay:   8 * time.Second,
			action:  func() { algedonic.TriggerCascadeFailure(4) },
			message: "🔥 Simulating cascade failure...",
		},
		{
			delay:   10 * time.Second,
			action:  algedonic.TriggerSecurityBreach,
			message: "🔓 Simulating security breach...",
		},
		{
			delay:   12 * time.Second,
			action:  algedonic.TriggerReputationThreat,
			message: "📰 Simulating reputation threat...",
		},
	}
	
	for _, scenario := range scenarios {
		time.Sleep(scenario.delay)
		log.Println(scenario.message)
		scenario.action()
	}
}

func printFinalReport(health vsm.SystemTelemetry) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📈 FINAL VSM REPORT")
	fmt.Println(strings.Repeat("=", 50))
	
	// Calculate letter grade
	score := health.TopologyCompleteness * 100
	grade := "F"
	switch {
	case score >= 97:
		grade = "A+"
	case score >= 93:
		grade = "A"
	case score >= 90:
		grade = "A-"
	case score >= 87:
		grade = "B+"
	case score >= 83:
		grade = "B"
	case score >= 80:
		grade = "B-"
	case score >= 77:
		grade = "C+"
	case score >= 73:
		grade = "C"
	case score >= 70:
		grade = "C-"
	case score >= 67:
		grade = "D+"
	case score >= 63:
		grade = "D"
	case score >= 60:
		grade = "D-"
	}
	
	fmt.Printf("\nTotal Loops Registered: %d\n", health.TotalLoops)
	fmt.Printf("Active Loops:           %d\n", health.ActiveLoops)
	fmt.Printf("Average Firing Rate:    %.1f Hz\n", health.FiringRateHz)
	fmt.Printf("Variety Absorption:     %.1f%%\n", health.VarietyAbsorptionRate*100)
	fmt.Printf("Topology Completeness:  %.1f%%\n", health.TopologyCompleteness*100)
	fmt.Printf("Consciousness Level:    %.2f\n", health.ConsciousnessCoherence)
	fmt.Printf("\n🎯 VSM GRADE: %s\n", grade)
	
	if score >= 97 {
		fmt.Println("\n✨ EXCELLENT! VSM implementation exceeds targets!")
	} else if score >= 80 {
		fmt.Println("\n👍 Good progress, but more loops needed for full VSM compliance.")
	} else {
		fmt.Println("\n⚠️  Significant work needed to achieve VSM compliance.")
	}
	
	fmt.Println(strings.Repeat("=", 50))
}

var strings = struct {
	Repeat func(string, int) string
}{
	Repeat: func(s string, n int) string {
		result := ""
		for i := 0; i < n; i++ {
			result += s
		}
		return result
	},
}