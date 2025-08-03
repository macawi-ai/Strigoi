package licensing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Validator handles license validation and enforcement
type Validator struct {
	license      *License
	collector    *IntelligenceCollector
	githubSync   *GitHubSync
	
	// Caching
	cachePath    string
	cacheTimeout time.Duration
	lastCheck    time.Time
	mu           sync.RWMutex
	
	// Telemetry
	telemetry    *TelemetryClient
	
	// Configuration
	validationURL string
	offlineMode   bool
}

// NewValidator creates a new license validator
func NewValidator(cachePath string) *Validator {
	return &Validator{
		cachePath:     cachePath,
		cacheTimeout:  24 * time.Hour,
		validationURL: "https://api.macawi.io/v1/license/validate",
		telemetry:     NewTelemetryClient(),
		githubSync:    NewGitHubSync("macawi-ai", "strigoi-intelligence", "main"),
	}
}

// ValidateLicense validates a license key
func (v *Validator) ValidateLicense(ctx context.Context, key string) (*License, error) {
	// Send telemetry
	v.telemetry.SendEvent("license_validation_start")
	
	// Check cache first
	if cached, err := v.loadCachedLicense(key); err == nil && v.isCacheValid(cached) {
		v.license = cached
		v.telemetry.SendEvent("license_validation_cached")
		return cached, nil
	}
	
	// Validate online
	license, err := v.validateOnline(ctx, key)
	if err != nil {
		// Fall back to offline validation if available
		if v.offlineMode {
			return v.validateOffline(key)
		}
		v.telemetry.SendEvent("license_validation_failed")
		return nil, fmt.Errorf("license validation failed: %w", err)
	}
	
	// Cache the license
	if err := v.cacheLicense(license); err != nil {
		// Non-fatal error
		fmt.Fprintf(os.Stderr, "Warning: failed to cache license: %v\n", err)
	}
	
	// Initialize components based on license type
	if err := v.initializeLicenseComponents(license); err != nil {
		return nil, fmt.Errorf("failed to initialize license components: %w", err)
	}
	
	v.license = license
	v.telemetry.SendEvent("license_validation_success")
	
	return license, nil
}

// validateOnline performs online license validation
func (v *Validator) validateOnline(ctx context.Context, key string) (*License, error) {
	payload := map[string]interface{}{
		"key":     key,
		"product": "strigoi",
		"version": "1.0.0", // Would get from build info
		"host":    v.getHostInfo(),
	}
	
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", v.validationURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Strigoi/1.0")
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("validation server returned %d: %s", resp.StatusCode, string(body))
	}
	
	var validationResp ValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&validationResp); err != nil {
		return nil, err
	}
	
	if !validationResp.Valid {
		return nil, fmt.Errorf("license is invalid: %s", validationResp.Message)
	}
	
	return validationResp.License, nil
}

// validateOffline performs offline license validation
func (v *Validator) validateOffline(key string) (*License, error) {
	// For offline validation, we check:
	// 1. License format
	// 2. Cached license data
	// 3. Built-in trial licenses
	
	// Check if it's a known trial key
	if v.isTrialKey(key) {
		return v.createTrialLicense(key), nil
	}
	
	// Try to load from cache even if expired
	cached, err := v.loadCachedLicense(key)
	if err == nil {
		// Allow grace period for offline usage
		gracePeriod := 7 * 24 * time.Hour
		if time.Since(cached.LastValidated) < gracePeriod {
			return cached, nil
		}
	}
	
	return nil, fmt.Errorf("offline validation failed: no valid cached license")
}

// initializeLicenseComponents sets up license-specific components
func (v *Validator) initializeLicenseComponents(license *License) error {
	// Initialize intelligence collector for community and community+ licenses
	if (license.Type == LicenseTypeCommunity || license.Type == LicenseTypeCommunityPlus) && license.IntelSharingEnabled {
		v.collector = NewIntelligenceCollector(license)
		v.collector.Start(context.Background())
		
		// Set contribution multiplier for Community+ researchers
		if license.Type == LicenseTypeCommunityPlus && license.ResearcherVerification != nil {
			v.collector.SetContributionMultiplier(license.ResearcherVerification.ContributionMultiplier)
		}
	}
	
	// Set compliance policies
	if license.ComplianceMode && v.collector != nil {
		policies := make([]CompliancePolicy, 0)
		for _, policyName := range license.CompliancePolicies {
			policy := v.getCompliancePolicy(policyName)
			if policy != nil {
				policies = append(policies, *policy)
			}
		}
		v.collector.anonymizer.SetCompliancePolicies(policies)
	}
	
	return nil
}

// GetIntelligenceCollector returns the intelligence collector if available
func (v *Validator) GetIntelligenceCollector() *IntelligenceCollector {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.collector
}

// SyncMarketplace synchronizes with the marketplace based on contribution level
func (v *Validator) SyncMarketplace(ctx context.Context) error {
	if v.license == nil {
		return fmt.Errorf("no valid license")
	}
	
	// Check if user has marketplace access
	if v.license.Type == LicenseTypeCommercial {
		// Commercial licenses have full access
		return v.syncMarketplaceFull(ctx)
	}
	
	if v.license.Type == LicenseTypeCommunityPlus {
		// Community+ researchers get enhanced marketplace access
		return v.syncMarketplaceEnhanced(ctx)
	}
	
	if v.license.Type == LicenseTypeCommunity {
		// Community licenses need to contribute first
		if !v.hasContributed() {
			return fmt.Errorf("marketplace access requires intelligence contribution")
		}
		
		// Sync based on contribution level
		return v.syncMarketplaceLimited(ctx)
	}
	
	return fmt.Errorf("license type does not support marketplace access")
}

// syncMarketplaceFull performs full marketplace sync for commercial licenses
func (v *Validator) syncMarketplaceFull(ctx context.Context) error {
	v.telemetry.SendEvent("marketplace_sync_start")
	
	// Get all available updates
	updates, err := v.githubSync.FetchMarketplaceUpdates(ctx, MarketplaceUnlimited, v.lastCheck)
	if err != nil {
		return fmt.Errorf("failed to fetch updates: %w", err)
	}
	
	// Download and apply updates
	for _, update := range updates {
		if err := v.downloadAndApplyUpdate(ctx, update); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to apply update %s: %v\n", update.Name, err)
		}
	}
	
	v.lastCheck = time.Now()
	v.telemetry.SendEvent("marketplace_sync_complete")
	
	return nil
}

// syncMarketplaceEnhanced performs enhanced sync for Community+ researchers
func (v *Validator) syncMarketplaceEnhanced(ctx context.Context) error {
	v.telemetry.SendEvent("marketplace_sync_enhanced_start")
	
	if v.collector == nil {
		return fmt.Errorf("intelligence collector not initialized")
	}
	
	// Push any pending intelligence with researcher attribution
	if err := v.collector.FlushWithAttribution(); err != nil {
		return fmt.Errorf("failed to submit intelligence: %w", err)
	}
	
	// Community+ gets enhanced marketplace access
	updates, err := v.githubSync.FetchMarketplaceUpdates(ctx, MarketplaceEnhanced, v.lastCheck)
	if err != nil {
		return fmt.Errorf("failed to fetch updates: %w", err)
	}
	
	// Apply updates with researcher priority
	for _, update := range updates {
		if err := v.downloadAndApplyUpdate(ctx, update); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to apply update %s: %v\n", update.Name, err)
		}
	}
	
	// Check for researcher-exclusive content
	if v.license.ResearcherVerification != nil {
		exclusiveUpdates, err := v.githubSync.FetchResearcherExclusive(ctx, v.license.ResearcherVerification.ResearcherID)
		if err == nil {
			for _, update := range exclusiveUpdates {
				v.downloadAndApplyUpdate(ctx, update)
			}
		}
	}
	
	v.lastCheck = time.Now()
	v.telemetry.SendEvent("marketplace_sync_enhanced_complete")
	
	return nil
}

// syncMarketplaceLimited performs limited sync for community licenses
func (v *Validator) syncMarketplaceLimited(ctx context.Context) error {
	if v.collector == nil {
		return fmt.Errorf("intelligence collector not initialized")
	}
	
	// First, push any pending intelligence
	if err := v.collector.Flush(); err != nil {
		return fmt.Errorf("failed to submit intelligence: %w", err)
	}
	
	// Determine access level based on contributions
	accessLevel := v.calculateAccessLevel()
	
	// Fetch updates based on access level
	updates, err := v.githubSync.FetchMarketplaceUpdates(ctx, accessLevel, v.lastCheck)
	if err != nil {
		return fmt.Errorf("failed to fetch updates: %w", err)
	}
	
	// Apply updates
	for _, update := range updates {
		if err := v.downloadAndApplyUpdate(ctx, update); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to apply update %s: %v\n", update.Name, err)
		}
	}
	
	v.lastCheck = time.Now()
	return nil
}

// Helper methods

func (v *Validator) loadCachedLicense(key string) (*License, error) {
	cacheFile := filepath.Join(v.cachePath, v.getCacheFileName(key))
	
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}
	
	var license License
	if err := json.Unmarshal(data, &license); err != nil {
		return nil, err
	}
	
	// Verify key matches
	if license.Key != key {
		return nil, fmt.Errorf("cached license key mismatch")
	}
	
	return &license, nil
}

func (v *Validator) cacheLicense(license *License) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(v.cachePath, 0700); err != nil {
		return err
	}
	
	cacheFile := filepath.Join(v.cachePath, v.getCacheFileName(license.Key))
	
	// Update last validated time
	license.LastValidated = time.Now()
	
	data, err := json.MarshalIndent(license, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(cacheFile, data, 0600)
}

func (v *Validator) getCacheFileName(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("license-%s.json", hex.EncodeToString(hash[:8]))
}

func (v *Validator) isCacheValid(license *License) bool {
	return time.Since(license.LastValidated) < v.cacheTimeout
}

func (v *Validator) getHostInfo() map[string]interface{} {
	hostname, _ := os.Hostname()
	return map[string]interface{}{
		"hostname": hostname,
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"version":  "1.0.0", // Would get from build info
	}
}

func (v *Validator) isTrialKey(key string) bool {
	// Check against known trial key patterns
	trialPrefixes := []string{
		"TRIAL-",
		"DEMO-",
		"EVAL-",
	}
	
	for _, prefix := range trialPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	
	return false
}

func (v *Validator) createTrialLicense(key string) *License {
	return &License{
		ID:           key,
		Type:         LicenseTypeTrial,
		Key:          key,
		Organization: "Trial User",
		Email:        "trial@example.com",
		IssuedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour),
		MaxInstances: 1,
		AllowedFeatures: []string{
			"basic_scanning",
			"reporting",
		},
		IntelSharingEnabled: false,
		ComplianceMode:      true,
		CompliancePolicies:  []string{"GDPR", "CCPA"},
	}
}

func (v *Validator) hasContributed() bool {
	if v.license == nil || v.license.IntelSharingConfig == nil {
		return false
	}
	
	return v.license.IntelSharingConfig.TotalContributions > 0
}

func (v *Validator) calculateAccessLevel() MarketplaceAccess {
	if v.license == nil || v.license.IntelSharingConfig == nil {
		return MarketplaceNone
	}
	
	// Community+ researchers get enhanced access by default
	if v.license.Type == LicenseTypeCommunityPlus {
		return MarketplaceEnhanced
	}
	
	score := v.license.IntelSharingConfig.ContributionScore
	
	switch {
	case score >= 10000:
		return MarketplacePremium
	case score >= 1000:
		return MarketplaceStandard
	case score >= 100:
		return MarketplaceBasic
	default:
		return MarketplaceNone
	}
}

func (v *Validator) downloadAndApplyUpdate(ctx context.Context, update MarketplaceUpdate) error {
	// Download update data
	resp, err := http.Get(update.DownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Parse and apply intelligence data
	var intel []ThreatIntelligence
	if err := json.NewDecoder(resp.Body).Decode(&intel); err != nil {
		return err
	}
	
	// Apply intelligence updates to local database
	// This would integrate with the registry/database
	
	return nil
}

func (v *Validator) getCompliancePolicy(name string) *CompliancePolicy {
	// Return predefined compliance policies
	policies := map[string]CompliancePolicy{
		"GDPR": {
			Name:        "GDPR",
			Regulations: []string{"EU General Data Protection Regulation"},
			MustScrub: []string{
				"email", "ip_address", "name", "address", "phone",
				"biometric_data", "genetic_data", "health_data",
			},
			RetentionDays:   90,
			GeoRestrictions: []string{"EU"},
		},
		"HIPAA": {
			Name:        "HIPAA",
			Regulations: []string{"Health Insurance Portability and Accountability Act"},
			MustScrub: []string{
				"patient_name", "medical_record", "health_plan",
				"certificate_numbers", "device_identifiers", "biometric",
				"full_face_photos", "account_numbers", "vehicle_identifiers",
			},
			RetentionDays:   180,
			GeoRestrictions: []string{"US"},
		},
		"PCI-DSS": {
			Name:        "PCI-DSS",
			Regulations: []string{"Payment Card Industry Data Security Standard"},
			MustScrub: []string{
				"card_number", "cvv", "cvv2", "cvc2", "cid",
				"cardholder_name", "expiry_date", "service_code",
			},
			RetentionDays:   365,
			GeoRestrictions: []string{},
		},
		"CCPA": {
			Name:        "CCPA",
			Regulations: []string{"California Consumer Privacy Act"},
			MustScrub: []string{
				"real_name", "postal_address", "email", "ssn",
				"drivers_license", "passport", "signature", "biometric",
				"browsing_history", "search_history", "geolocation",
			},
			RetentionDays:   365,
			GeoRestrictions: []string{"US-CA"},
		},
	}
	
	if policy, ok := policies[name]; ok {
		return &policy
	}
	
	return nil
}

// Stop gracefully stops the validator and its components
func (v *Validator) Stop() {
	if v.collector != nil {
		v.collector.Stop()
	}
	
	if v.telemetry != nil {
		v.telemetry.Close()
	}
}