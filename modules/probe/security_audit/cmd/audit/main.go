package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/macawi-ai/strigoi/modules/probe/security_audit"
)

func main() {
	var (
		sourcePath   = flag.String("path", ".", "Path to scan")
		outputPath   = flag.String("output", "", "Output file (default: stdout)")
		outputFormat = flag.String("format", "markdown", "Output format: json, markdown, html")

		// Scan options
		enableCode    = flag.Bool("code", true, "Enable code security scanning")
		enableDeps    = flag.Bool("deps", true, "Enable dependency scanning")
		enableConfig  = flag.Bool("config", true, "Enable configuration scanning")
		enableRuntime = flag.Bool("runtime", false, "Enable runtime scanning (requires tests)")
		enableNetwork = flag.Bool("network", false, "Enable network scanning")
		enableAll     = flag.Bool("all", false, "Enable all scanners")

		// Compliance standards
		compliance = flag.String("compliance", "", "Compliance standards to check (comma-separated): PCI-DSS,OWASP,CIS")

		// Thresholds
		maxCritical = flag.Int("max-critical", 0, "Maximum critical issues allowed (0 = no limit)")
		maxHigh     = flag.Int("max-high", 0, "Maximum high issues allowed (0 = no limit)")

		// Other options
		exclude = flag.String("exclude", "", "Paths to exclude (comma-separated)")
		verbose = flag.Bool("verbose", false, "Verbose output")
		help    = flag.Bool("help", false, "Show help")
	)

	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Build configuration
	config := security_audit.AuditConfig{
		SourcePaths:       []string{*sourcePath},
		OutputPath:        *outputPath,
		OutputFormat:      *outputFormat,
		EnableCodeScan:    *enableCode || *enableAll,
		EnableDepsScan:    *enableDeps || *enableAll,
		EnableConfigScan:  *enableConfig || *enableAll,
		EnableRuntimeScan: *enableRuntime || *enableAll,
		EnableNetworkScan: *enableNetwork || *enableAll,
		MaxCriticalIssues: *maxCritical,
		MaxHighIssues:     *maxHigh,
	}

	// Parse exclude paths
	if *exclude != "" {
		config.ExcludePaths = strings.Split(*exclude, ",")
	}

	// Create audit framework
	framework := security_audit.NewAuditFramework(config)

	// Add compliance scanners if requested
	if *compliance != "" {
		standards := strings.Split(*compliance, ",")
		complianceScanner := security_audit.NewComplianceScanner(standards)
		framework.AddScanner(complianceScanner)
	}

	// Run audit
	if *verbose {
		fmt.Printf("Starting security audit of %s...\n", *sourcePath)
		fmt.Printf("Scanners enabled:\n")
		if config.EnableCodeScan {
			fmt.Println("  - Code security")
		}
		if config.EnableDepsScan {
			fmt.Println("  - Dependencies")
		}
		if config.EnableConfigScan {
			fmt.Println("  - Configuration")
		}
		if config.EnableRuntimeScan {
			fmt.Println("  - Runtime analysis")
		}
		if config.EnableNetworkScan {
			fmt.Println("  - Network security")
		}
		fmt.Println()
	}

	results, err := framework.RunAudit()
	if err != nil {
		// Check if error is due to threshold
		if strings.Contains(err.Error(), "exceed threshold") {
			// Still generated results, just failed threshold
			if results != nil {
				printSummary(results)
			}
			fmt.Fprintf(os.Stderr, "\n❌ Audit failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Error running audit: %v\n", err)
		os.Exit(1)
	}

	// Print summary if verbose
	if *verbose && *outputPath != "" {
		printSummary(results)
	}

	// Success
	if results.Summary.CriticalIssues == 0 && results.Summary.HighIssues == 0 {
		fmt.Println("\n✅ Security audit passed!")
	} else {
		fmt.Printf("\n⚠️  Security audit completed with issues\n")
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Strigoi Security Audit Tool

Usage: audit [options]

Options:
  -path string      Path to scan (default ".")
  -output string    Output file (default: stdout)
  -format string    Output format: json, markdown, html (default "markdown")
  
Scanners:
  -code             Enable code security scanning (default true)
  -deps             Enable dependency scanning (default true)
  -config           Enable configuration scanning (default true)
  -runtime          Enable runtime scanning - requires tests (default false)
  -network          Enable network scanning (default false)
  -all              Enable all scanners
  
Compliance:
  -compliance       Compliance standards to check: PCI-DSS,OWASP,CIS
  
Thresholds:
  -max-critical     Maximum critical issues allowed (0 = no limit)
  -max-high         Maximum high issues allowed (0 = no limit)
  
Other:
  -exclude          Paths to exclude (comma-separated)
  -verbose          Verbose output
  -help             Show this help

Examples:
  # Basic audit
  audit -path ./src
  
  # Full audit with all scanners
  audit -path . -all -output report.md
  
  # CI/CD mode with thresholds
  audit -path . -max-critical 0 -max-high 5
  
  # Compliance check
  audit -path . -compliance OWASP,PCI-DSS
  
  # JSON output for automation
  audit -path . -format json -output audit.json`)
}

func printSummary(results *security_audit.AuditResults) {
	fmt.Println("\n=== Audit Summary ===")
	fmt.Printf("Security Score: %.1f/100\n", results.Metrics.SecurityScore)
	fmt.Printf("Total Issues:   %d\n", results.Summary.TotalIssues)
	fmt.Printf("  Critical:     %d\n", results.Summary.CriticalIssues)
	fmt.Printf("  High:         %d\n", results.Summary.HighIssues)
	fmt.Printf("  Medium:       %d\n", results.Summary.MediumIssues)
	fmt.Printf("  Low:          %d\n", results.Summary.LowIssues)
	fmt.Printf("  Info:         %d\n", results.Summary.InfoIssues)
	fmt.Printf("\nScan Duration:  %s\n", results.Summary.TimeElapsed)
}
