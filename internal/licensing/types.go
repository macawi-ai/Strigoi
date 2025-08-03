package licensing

import (
	"time"
)

// LicenseType represents the type of license
type LicenseType string

const (
	LicenseTypeCommercial     LicenseType = "commercial"
	LicenseTypeCommunity      LicenseType = "community"
	LicenseTypeCommunityPlus  LicenseType = "community_plus"
	LicenseTypeTrial         LicenseType = "trial"
	LicenseTypeEnterprise    LicenseType = "enterprise"
)

// License represents a Strigoi license
type License struct {
	// Core fields
	ID           string      `json:"id"`
	Type         LicenseType `json:"type"`
	Key          string      `json:"key"`
	Organization string      `json:"organization"`
	Email        string      `json:"email"`
	
	// Validity
	IssuedAt    time.Time  `json:"issued_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	LastValidated time.Time `json:"last_validated,omitempty"`
	
	// Capabilities
	MaxInstances    int      `json:"max_instances"`
	AllowedFeatures []string `json:"allowed_features"`
	
	// Intelligence sharing settings
	IntelSharingEnabled  bool                `json:"intel_sharing_enabled"`
	IntelSharingConfig   *IntelSharingConfig `json:"intel_sharing_config,omitempty"`
	
	// Compliance
	ComplianceMode      bool     `json:"compliance_mode"`
	CompliancePolicies  []string `json:"compliance_policies,omitempty"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	
	// Community+ specific fields
	ResearcherVerification *ResearcherVerification `json:"researcher_verification,omitempty"`
}

// IntelSharingConfig defines intelligence sharing parameters
type IntelSharingConfig struct {
	// What to share
	ShareAttackPatterns   bool `json:"share_attack_patterns"`
	ShareVulnerabilities  bool `json:"share_vulnerabilities"`
	ShareConfigurations   bool `json:"share_configurations"`
	ShareStatistics       bool `json:"share_statistics"`
	
	// Anonymization settings
	AnonymizationLevel    AnonymizationLevel `json:"anonymization_level"`
	TokenizationEnabled   bool              `json:"tokenization_enabled"`
	ReversibleTokens      bool              `json:"reversible_tokens"`
	
	// Contribution tracking
	ContributionScore     int               `json:"contribution_score"`
	LastContribution      time.Time         `json:"last_contribution,omitempty"`
	TotalContributions    int               `json:"total_contributions"`
	
	// Marketplace access
	MarketplaceAccessLevel MarketplaceAccess `json:"marketplace_access_level"`
}

// AnonymizationLevel defines how deeply to anonymize data
type AnonymizationLevel string

const (
	AnonymizationMinimal  AnonymizationLevel = "minimal"   // Only direct PII
	AnonymizationStandard AnonymizationLevel = "standard"  // PII + internal identifiers
	AnonymizationStrict   AnonymizationLevel = "strict"    // Everything identifiable
	AnonymizationParanoid AnonymizationLevel = "paranoid"  // Maximum scrubbing
)

// MarketplaceAccess defines marketplace access levels
type MarketplaceAccess string

const (
	MarketplaceNone      MarketplaceAccess = "none"
	MarketplaceBasic     MarketplaceAccess = "basic"      // Community contributions
	MarketplaceStandard  MarketplaceAccess = "standard"   // Regular updates
	MarketplaceEnhanced  MarketplaceAccess = "enhanced"   // Community+ researchers
	MarketplacePremium   MarketplaceAccess = "premium"    // Early access
	MarketplaceUnlimited MarketplaceAccess = "unlimited"  // Everything
)

// ValidationResponse from license server
type ValidationResponse struct {
	Valid       bool        `json:"valid"`
	License     *License    `json:"license,omitempty"`
	Message     string      `json:"message,omitempty"`
	NextCheck   time.Time   `json:"next_check"`
	UpdateToken string      `json:"update_token,omitempty"`
}

// ThreatIntelligence represents shared threat data
type ThreatIntelligence struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Timestamp     time.Time              `json:"timestamp"`
	Source        string                 `json:"source"` // Anonymized instance ID
	
	// Attack pattern data
	AttackPattern *AttackPatternIntel    `json:"attack_pattern,omitempty"`
	
	// Vulnerability data
	Vulnerability *VulnerabilityIntel    `json:"vulnerability,omitempty"`
	
	// Configuration intel
	Configuration *ConfigurationIntel    `json:"configuration,omitempty"`
	
	// Statistics
	Statistics    *StatisticsIntel       `json:"statistics,omitempty"`
	
	// Metadata
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AttackPatternIntel contains anonymized attack pattern information
type AttackPatternIntel struct {
	PatternHash    string   `json:"pattern_hash"`
	Category       string   `json:"category"`
	Severity       string   `json:"severity"`
	TargetType     string   `json:"target_type"`
	Techniques     []string `json:"techniques"`
	Indicators     []string `json:"indicators"` // Hashed/tokenized
	SuccessRate    float64  `json:"success_rate"`
	DetectionRate  float64  `json:"detection_rate"`
}

// VulnerabilityIntel contains anonymized vulnerability data
type VulnerabilityIntel struct {
	VulnHash       string   `json:"vuln_hash"`
	Category       string   `json:"category"`
	Severity       string   `json:"severity"`
	AffectedTypes  []string `json:"affected_types"`
	ExploitExists  bool     `json:"exploit_exists"`
	PatchAvailable bool     `json:"patch_available"`
	PrevalenceScore float64 `json:"prevalence_score"`
}

// ConfigurationIntel contains anonymized configuration insights
type ConfigurationIntel struct {
	ConfigType     string                 `json:"config_type"`
	SecurityScore  float64                `json:"security_score"`
	CommonMistakes []string               `json:"common_mistakes"`
	BestPractices  []string               `json:"best_practices"`
	RiskFactors    map[string]float64     `json:"risk_factors"`
}

// StatisticsIntel contains usage and performance statistics
type StatisticsIntel struct {
	ScanCount      int                    `json:"scan_count"`
	FindingsCount  map[string]int         `json:"findings_count"` // By severity
	Performance    map[string]float64     `json:"performance"`    // Timing stats
	ErrorRates     map[string]float64     `json:"error_rates"`
	FeatureUsage   map[string]int         `json:"feature_usage"`
}

// CompliancePolicy defines data handling requirements
type CompliancePolicy struct {
	Name           string   `json:"name"`
	Regulations    []string `json:"regulations"` // GDPR, HIPAA, etc.
	MustScrub      []string `json:"must_scrub"`  // Data types to remove
	RetentionDays  int      `json:"retention_days"`
	GeoRestrictions []string `json:"geo_restrictions"`
}

// ResearcherVerification contains verification details for Community+ researchers
type ResearcherVerification struct {
	// Verification method
	VerificationMethod string    `json:"verification_method"` // github, academic, linkedin
	VerificationID     string    `json:"verification_id"`     // GitHub username, email domain, etc.
	VerifiedAt         time.Time `json:"verified_at"`
	LastReverified     time.Time `json:"last_reverified,omitempty"`
	
	// Researcher profile
	GitHubProfile      string    `json:"github_profile,omitempty"`
	AcademicEmail      string    `json:"academic_email,omitempty"`
	LinkedInProfile    string    `json:"linkedin_profile,omitempty"`
	ResearchAreas      []string  `json:"research_areas,omitempty"`
	
	// Recognition
	BadgeLevel         string    `json:"badge_level"` // bronze, silver, gold, platinum
	MonthlyHighlight   bool      `json:"monthly_highlight"`
	SpecialRecognition []string  `json:"special_recognition,omitempty"`
	
	// Enhanced contribution tracking
	ContributionMultiplier float64 `json:"contribution_multiplier"` // 2x-5x for researchers
	NovelFindings          int     `json:"novel_findings"`
	ZeroDayDiscoveries     int     `json:"zero_day_discoveries"`
	PublishedResearch      []string `json:"published_research,omitempty"`
}

// ResearcherContribution represents enhanced contribution tracking for Community+
type ResearcherContribution struct {
	ContributionID   string    `json:"contribution_id"`
	ResearcherID     string    `json:"researcher_id"`
	Timestamp        time.Time `json:"timestamp"`
	Type             string    `json:"type"` // novel_attack, zero_day, research_integration, etc.
	
	// Enhanced scoring
	BasePoints       int       `json:"base_points"`
	Multiplier       float64   `json:"multiplier"`
	FinalPoints      int       `json:"final_points"`
	
	// Recognition
	Featured         bool      `json:"featured"`
	Description      string    `json:"description"`
	Impact           string    `json:"impact"` // low, medium, high, critical
}