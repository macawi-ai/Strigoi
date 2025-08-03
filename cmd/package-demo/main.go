package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/macawi-ai/strigoi/internal/core"
	"github.com/macawi-ai/strigoi/internal/packages"
)

func main() {
	var (
		baseDir      = flag.String("base-dir", "./protocols/packages", "Base directory for protocol packages")
		updatePort   = flag.Int("update-port", 8888, "Port for update service")
		checkUpdates = flag.Bool("check-updates", false, "Check for package updates")
		startServer  = flag.Bool("server", false, "Start update server")
	)
	
	flag.Parse()
	
	// Initialize logger
	logger, err := core.NewLogger("info", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	
	if *startServer {
		// Start update server
		startUpdateServer(*updatePort, logger)
		return
	}
	
	// Create package loader
	factory := packages.NewDynamicModuleFactory(logger)
	loader := packages.NewPackageLoader(*baseDir, factory, logger)
	
	// Print banner
	printBanner()
	
	// Load packages
	fmt.Println("📦 Loading protocol packages...")
	if err := loader.LoadPackages(); err != nil {
		logger.Error("Failed to load packages: %v", err)
		os.Exit(1)
	}
	
	// List loaded packages
	fmt.Println("\n📋 Loaded Packages:")
	listPackages(loader)
	
	// Check for updates if requested
	if *checkUpdates {
		fmt.Println("\n🔄 Checking for updates...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := loader.CheckForUpdates(ctx); err != nil {
			logger.Error("Failed to check updates: %v", err)
		}
		
		// List packages again to show updates
		fmt.Println("\n📋 Packages after update check:")
		listPackages(loader)
	}
	
	// Generate modules
	fmt.Println("\n🔧 Generating modules from packages...")
	modules, err := loader.GenerateModules()
	if err != nil {
		logger.Error("Failed to generate modules: %v", err)
		os.Exit(1)
	}
	
	// Display generated modules
	fmt.Printf("\n✅ Generated %d modules:\n", len(modules))
	for _, module := range modules {
		info := module.Info()
		color.New(color.FgGreen).Printf("  • %s", module.Name())
		fmt.Printf(" (v%s) - %s\n", info.Version, module.Description())
	}
	
	// Show package intelligence
	fmt.Println("\n🧠 Protocol Intelligence:")
	showIntelligence(loader)
	
	// Simulate loading into framework
	fmt.Println("\n🚀 Simulating framework integration...")
	simulateFrameworkLoad(modules, logger)
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║                  Strigoi Package Loader                   ║
║         MSF-Style Protocol Package Management             ║
╚═══════════════════════════════════════════════════════════╝
`
	color.New(color.FgCyan, color.Bold).Println(banner)
}

func listPackages(loader *packages.PackageLoader) {
	pkgs := loader.ListPackages()
	
	for _, pkg := range pkgs {
		fmt.Printf("\n  📦 %s v%s\n", 
			color.New(color.FgYellow, color.Bold).Sprint(pkg.Header.ProtocolIdentity.Name),
			pkg.Header.ProtocolIdentity.Version)
		fmt.Printf("     Package Version: %s\n", pkg.Header.StrigoiMetadata.PackageVersion)
		fmt.Printf("     Last Updated: %s\n", pkg.Header.StrigoiMetadata.LastUpdated.Format("2006-01-02"))
		fmt.Printf("     Test Coverage: %.1f%%\n", pkg.Header.SecurityAssessment.TestCoverage)
		fmt.Printf("     Vulnerabilities: %d (Critical: %d)\n", 
			pkg.Header.SecurityAssessment.VulnerabilityCount,
			pkg.Header.SecurityAssessment.CriticalFindings)
		fmt.Printf("     Modules: %d\n", len(pkg.Payload.TestModules))
		
		// Show update info if available
		if pkg.Payload.UpdateConfiguration.UpdateSource != "" {
			fmt.Printf("     Update Source: %s\n", pkg.Payload.UpdateConfiguration.UpdateSource)
			fmt.Printf("     Update Frequency: %s\n", pkg.Payload.UpdateConfiguration.UpdateFrequency)
		}
	}
}

func showIntelligence(loader *packages.PackageLoader) {
	if pkg, exists := loader.GetPackageByName("Model Context Protocol"); exists {
		intel := pkg.Payload.ProtocolIntelligence
		
		// Show known implementations
		if len(intel.KnownImplementations) > 0 {
			fmt.Println("\n  🏢 Known Implementations:")
			for _, impl := range intel.KnownImplementations {
				fmt.Printf("    • %s by %s\n", 
					color.New(color.FgBlue).Sprint(impl.Name),
					impl.Vendor)
			}
		}
		
		// Show vulnerabilities
		if len(intel.CommonVulnerabilities) > 0 {
			fmt.Println("\n  ⚠️  Known Vulnerabilities:")
			for _, vuln := range intel.CommonVulnerabilities {
				severityColor := getSeverityColor(vuln.Severity)
				fmt.Printf("    • %s [%s]: %s\n",
					vuln.CVE,
					severityColor.Sprint(vuln.Severity),
					vuln.Description)
			}
		}
		
		// Show attack chains
		if len(intel.AttackChains) > 0 {
			fmt.Println("\n  ⛓️  Attack Chains:")
			for _, chain := range intel.AttackChains {
				fmt.Printf("    • %s (%s complexity, %s impact)\n",
					color.New(color.FgMagenta).Sprint(chain.Name),
					chain.Complexity,
					getSeverityColor(chain.Impact).Sprint(chain.Impact))
			}
		}
	}
}

func getSeverityColor(severity string) *color.Color {
	switch severity {
	case "critical":
		return color.New(color.FgRed, color.Bold)
	case "high":
		return color.New(color.FgRed)
	case "medium":
		return color.New(color.FgYellow)
	case "low":
		return color.New(color.FgGreen)
	default:
		return color.New(color.FgWhite)
	}
}

func simulateFrameworkLoad(modules []core.Module, logger core.Logger) {
	// Create a minimal framework
	config := core.DefaultConfig()
	framework, err := core.NewFramework(config, logger)
	if err != nil {
		logger.Error("Failed to create framework: %v", err)
		return
	}
	
	// Load modules
	loaded := 0
	for _, module := range modules {
		if err := framework.LoadModule(module); err != nil {
			logger.Error("Failed to load module %s: %v", module.Name(), err)
		} else {
			loaded++
		}
	}
	
	fmt.Printf("\n✅ Successfully loaded %d/%d modules into framework\n", loaded, len(modules))
}

func startUpdateServer(port int, logger core.Logger) {
	fmt.Printf("🌐 Starting update server on port %d...\n", port)
	
	service := packages.NewUpdateService(port)
	if err := service.Start(); err != nil {
		logger.Error("Failed to start update service: %v", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Update server running at http://localhost:%d\n", port)
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("  • /protocols/mcp/latest.apms.yaml - MCP protocol updates")
	fmt.Println("  • /protocols/mcp/intelligence.json - Fresh threat intelligence")
	fmt.Println("  • /protocols/catalog.json - Protocol catalog")
	fmt.Println("\nPress Ctrl+C to stop...")
	
	// Wait forever
	select {}
}