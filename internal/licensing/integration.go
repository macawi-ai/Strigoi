package licensing

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
	
	"github.com/macawi-ai/strigoi/internal/core"
)

// Integration provides licensing integration with Strigoi framework
type Integration struct {
	validator *Validator
	framework *core.Framework
}

// NewIntegration creates a new licensing integration
func NewIntegration(config *core.Config) (*Integration, error) {
	// Determine cache path
	cachePath := filepath.Join(config.Paths.DataPath, "license")
	
	// Create validator
	validator := NewValidator(cachePath)
	
	return &Integration{
		validator: validator,
	}, nil
}

// Initialize validates license and sets up components
func (i *Integration) Initialize(ctx context.Context, licenseKey string) error {
	// Validate license
	license, err := i.validator.ValidateLicense(ctx, licenseKey)
	if err != nil {
		return fmt.Errorf("license validation failed: %w", err)
	}
	
	// Display license info
	i.displayLicenseInfo(license)
	
	// Configure framework based on license
	if err := i.configureFramework(license); err != nil {
		return fmt.Errorf("failed to configure framework: %w", err)
	}
	
	return nil
}

// SetFramework sets the framework reference
func (i *Integration) SetFramework(fw *core.Framework) {
	i.framework = fw
}

// CollectModuleResult collects intelligence from module results
func (i *Integration) CollectModuleResult(result *core.ModuleResult) {
	if collector := i.validator.GetIntelligenceCollector(); collector != nil {
		collector.CollectModuleResult(result)
	}
}

// CollectAttackPattern collects attack pattern intelligence
func (i *Integration) CollectAttackPattern(pattern map[string]interface{}) {
	if collector := i.validator.GetIntelligenceCollector(); collector != nil {
		collector.CollectAttackPattern(pattern)
	}
}

// SyncMarketplace synchronizes with the module marketplace
func (i *Integration) SyncMarketplace(ctx context.Context) error {
	return i.validator.SyncMarketplace(ctx)
}

// GetLicense returns the current license
func (i *Integration) GetLicense() *License {
	return i.validator.license
}

// GetCollectorStats returns intelligence collection statistics
func (i *Integration) GetCollectorStats() *CollectorStats {
	if collector := i.validator.GetIntelligenceCollector(); collector != nil {
		stats := collector.GetStats()
		return &stats
	}
	return nil
}

// Stop gracefully stops the integration
func (i *Integration) Stop() {
	i.validator.Stop()
}

// Helper methods

func (i *Integration) displayLicenseInfo(license *License) {
	fmt.Println("\n╔════════════════════════════════════════╗")
	fmt.Println("║        LICENSE INFORMATION             ║")
	fmt.Println("╠════════════════════════════════════════╣")
	fmt.Printf("║ Type:         %-24s ║\n", license.Type)
	fmt.Printf("║ Organization: %-24s ║\n", truncate(license.Organization, 24))
	fmt.Printf("║ Expires:      %-24s ║\n", license.ExpiresAt.Format("2006-01-02"))
	
	if license.Type == LicenseTypeCommunity {
		fmt.Println("║                                        ║")
		fmt.Println("║ Intelligence Sharing: ENABLED          ║")
		if license.IntelSharingConfig != nil {
			fmt.Printf("║ Anonymization: %-23s ║\n", license.IntelSharingConfig.AnonymizationLevel)
			fmt.Printf("║ Contributions: %-23d ║\n", license.IntelSharingConfig.TotalContributions)
			fmt.Printf("║ Access Level:  %-23s ║\n", license.IntelSharingConfig.MarketplaceAccessLevel)
		}
	}
	
	fmt.Println("╚════════════════════════════════════════╝\n")
}

func (i *Integration) configureFramework(license *License) error {
	if i.framework == nil {
		return nil // Framework not set yet
	}
	
	// Configure based on license type
	switch license.Type {
	case LicenseTypeCommercial, LicenseTypeEnterprise:
		// Full features enabled
		i.framework.Config.SafeMode = false
		
	case LicenseTypeCommunity:
		// Enable intelligence collection
		if license.IntelSharingEnabled {
			i.enableIntelligenceCollection()
		}
		
	case LicenseTypeTrial:
		// Limited features
		i.framework.Config.SafeMode = true
	}
	
	// Apply compliance policies
	if license.ComplianceMode {
		i.applyCompliancePolicies(license.CompliancePolicies)
	}
	
	return nil
}

func (i *Integration) enableIntelligenceCollection() {
	// Hook into framework events for intelligence collection
	// This would integrate with the framework's event system
}

func (i *Integration) applyCompliancePolicies(policies []string) {
	// Configure framework with compliance requirements
	for _, policy := range policies {
		switch policy {
		case "GDPR":
			// Apply GDPR-specific configurations
		case "HIPAA":
			// Apply HIPAA-specific configurations
		case "PCI-DSS":
			// Apply PCI-DSS-specific configurations
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// LicenseMiddleware provides HTTP middleware for license validation
func LicenseMiddleware(integration *Integration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if license is valid
			license := integration.GetLicense()
			if license == nil {
				http.Error(w, "No valid license", http.StatusUnauthorized)
				return
			}
			
			// Check expiration
			if time.Now().After(license.ExpiresAt) {
				http.Error(w, "License expired", http.StatusUnauthorized)
				return
			}
			
			// Add license info to context
			ctx := context.WithValue(r.Context(), "license", license)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Example usage in main application
func ExampleUsage() {
	// Create framework config
	config := &core.Config{
		Paths: &core.Paths{
			DataPath: "/var/lib/strigoi",
		},
	}
	
	// Create licensing integration
	licensing, err := NewIntegration(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create licensing: %v\n", err)
		os.Exit(1)
	}
	
	// Get license key from environment or config
	licenseKey := os.Getenv("STRIGOI_LICENSE_KEY")
	if licenseKey == "" {
		licenseKey = "TRIAL-DEMO-KEY" // Default trial
	}
	
	// Initialize licensing
	ctx := context.Background()
	if err := licensing.Initialize(ctx, licenseKey); err != nil {
		fmt.Fprintf(os.Stderr, "License validation failed: %v\n", err)
		os.Exit(1)
	}
	
	// Create framework
	framework, err := core.NewFramework(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create framework: %v\n", err)
		os.Exit(1)
	}
	
	// Link licensing with framework
	licensing.SetFramework(framework)
	
	// Use framework with licensing integration
	// Module results will be automatically collected for intelligence
	
	// Sync marketplace periodically
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
			if err := licensing.SyncMarketplace(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Marketplace sync failed: %v\n", err)
			}
		}
	}()
	
	// Graceful shutdown
	defer licensing.Stop()
}