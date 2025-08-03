package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/macawi-ai/strigoi/internal/core"
)

var (
	version = "0.2.0-debug"
	build   = "dev"
)

func main() {
	fmt.Println("=== Strigoi Debug Version Starting ===")
	fmt.Printf("Version: %s (build: %s)\n", version, build)
	fmt.Printf("Time: %s\n\n", time.Now().Format("15:04:05.000"))
	
	var showVersion = flag.Bool("version", false, "Show version information")
	var testOnly = flag.Bool("test", false, "Run component tests only (no console)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Strigoi v%s (build: %s)\n", version, build)
		os.Exit(0)
	}

	// Get deployment paths
	fmt.Printf("[%s] Getting deployment paths...\n", timestamp())
	paths := core.GetPaths()
	fmt.Printf("  Config: %s\n", paths.Config)
	fmt.Printf("  Logs: %s\n", paths.Logs)
	fmt.Printf("  Protocols: %s\n", paths.Protocols)
	
	// Ensure directories exist
	fmt.Printf("\n[%s] Ensuring directories exist...\n", timestamp())
	if err := paths.EnsureDirectories(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create directories: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Directories created/verified")

	// Load configuration
	fmt.Printf("\n[%s] Loading configuration...\n", timestamp())
	config := core.DefaultConfig()
	fmt.Printf("  Log Level: %s\n", config.LogLevel)
	fmt.Printf("  Check on Start: %v\n", config.CheckOnStart)
	
	// Initialize logger
	fmt.Printf("\n[%s] Initializing logger...\n", timestamp())
	logger, err := core.NewLogger(config.LogLevel, config.LogFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Logger initialized")

	// Initialize framework without package loader
	fmt.Printf("\n[%s] Initializing framework (no package loader)...\n", timestamp())
	framework, err := core.NewFramework(config, logger)
	if err != nil {
		fmt.Printf("  ✗ Framework initialization failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Framework initialized")

	// Test stream manager
	fmt.Printf("\n[%s] Testing stream manager...\n", timestamp())
	streamMgr := framework.GetStreamManager()
	if streamMgr == nil {
		fmt.Println("  ✗ Stream manager is nil!")
	} else {
		fmt.Println("  ✓ Stream manager available")
		streams := streamMgr.ListStreams()
		fmt.Printf("  Active streams: %d\n", len(streams))
	}

	// Console will be created during Start()
	fmt.Printf("\n[%s] Console will be initialized during Start()...\n", timestamp())

	if *testOnly {
		fmt.Printf("\n[%s] Test mode complete. Exiting.\n", timestamp())
		os.Exit(0)
	}

	// Start console
	fmt.Printf("\n[%s] Starting console (this is where it might hang)...\n", timestamp())
	fmt.Println("If you see this message but nothing after, the console Start() is blocking.")
	
	if err := framework.Start(); err != nil {
		fmt.Printf("Console error: %v\n", err)
		os.Exit(1)
	}

	// Shutdown
	fmt.Printf("\n[%s] Shutting down...\n", timestamp())
	framework.Shutdown()
}

func timestamp() string {
	return time.Now().Format("15:04:05.000")
}