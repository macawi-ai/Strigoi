package core

import (
	"fmt"
	"time"
)

// LicenseType represents the type of license
type LicenseType string

const (
	LicenseCommunity     LicenseType = "community"
	LicenseCommunityPlus LicenseType = "community_plus"
	LicenseCommercial    LicenseType = "commercial"
	LicenseEnterprise    LicenseType = "enterprise"
	LicenseTrial         LicenseType = "trial"
)

// License represents a Strigoi license
type License struct {
	Type           LicenseType    `json:"type"`
	ID             string         `json:"id"`
	IssuedTo       string         `json:"issued_to"`
	IssuedAt       time.Time      `json:"issued_at"`
	ExpiresAt      time.Time      `json:"expires_at"`
	Features       LicenseFeatures `json:"features"`
	IntelSharing   IntelConfig    `json:"intel_sharing"`
}

// LicenseFeatures defines what features are available
type LicenseFeatures struct {
	MaxStreams        int  `json:"max_streams"`
	MarketplaceAccess bool `json:"marketplace_access"`
	EnhancedModules   bool `json:"enhanced_modules"`
	DirectSupport     bool `json:"direct_support"`
	CustomIntegration bool `json:"custom_integration"`
	PriorityUpdates   bool `json:"priority_updates"`
}

// IntelConfig defines intelligence sharing configuration
type IntelConfig struct {
	Required        bool     `json:"required"`
	ShareAttacks    bool     `json:"share_attacks"`
	ShareVulns      bool     `json:"share_vulns"`
	ShareConfigs    bool     `json:"share_configs"`
	AnonymizeLevel  string   `json:"anonymize_level"`
	ContribMultiplier float64 `json:"contrib_multiplier"`
}

// GetDefaultLicense returns the default Community license
func GetDefaultLicense() *License {
	return &License{
		Type:      LicenseCommunity,
		ID:        "community-default",
		IssuedTo:  "Community User",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().AddDate(100, 0, 0), // Effectively never expires
		Features: LicenseFeatures{
			MaxStreams:        3,
			MarketplaceAccess: true, // Requires intel sharing
			EnhancedModules:   false,
			DirectSupport:     false,
			CustomIntegration: false,
			PriorityUpdates:   false,
		},
		IntelSharing: IntelConfig{
			Required:          true,
			ShareAttacks:      true,
			ShareVulns:        true,
			ShareConfigs:      true,
			AnonymizeLevel:    "standard",
			ContribMultiplier: 1.0,
		},
	}
}

// GetLicenseInfo returns a formatted string describing the license
func (l *License) GetLicenseInfo() string {
	var info string
	
	switch l.Type {
	case LicenseCommunity:
		info = fmt.Sprintf(`
License Type: Community (Free)
Status: Active
Intel Sharing: REQUIRED

Features:
  ✓ Up to %d concurrent streams
  ✓ Marketplace access (with intel sharing)
  ✓ Community support
  
Intelligence Contribution:
  - Attack patterns: %s
  - Vulnerabilities: %s
  - Configurations: %s
  - Anonymization: %s level
  
Note: This license requires sharing anonymized threat
intelligence to maintain marketplace access.
`, l.Features.MaxStreams, 
			boolToStatus(l.IntelSharing.ShareAttacks),
			boolToStatus(l.IntelSharing.ShareVulns),
			boolToStatus(l.IntelSharing.ShareConfigs),
			l.IntelSharing.AnonymizeLevel)
			
	case LicenseCommunityPlus:
		info = `
License Type: Community+ ($20/month)
Status: Not Implemented Yet

Features (Coming Soon):
  ✓ All Community features
  ✓ 2x-5x contribution multipliers
  ✓ Researcher badges & recognition
  ✓ Early access to experimental modules
  ✓ Direct communication with Strigoi team
  ✓ Student discount available ($10/month)

To upgrade: Visit https://strigoi.io/community-plus
`
		
	case LicenseCommercial:
		info = `
License Type: Commercial ($5,000/year)
Status: Not Implemented Yet

Features (Coming Soon):
  ✓ Unlimited streams
  ✓ Full marketplace access
  ✓ NO mandatory intel sharing
  ✓ Priority support
  ✓ Volume discounts available

To purchase: Contact sales@strigoi.io
`
		
	case LicenseEnterprise:
		info = `
License Type: Enterprise (Custom Pricing)
Status: Not Implemented Yet

Features (Coming Soon):
  ✓ All Commercial features
  ✓ Custom integrations
  ✓ On-premise deployment options
  ✓ SLA guarantees
  ✓ Dedicated support team

To inquire: Contact enterprise@strigoi.io
`
	}
	
	return info
}

// ShowLicenseOptions displays all available license tiers
func ShowLicenseOptions() string {
	return `
Strigoi License Options - "Pay with Money or Pay with Intelligence"

1. Community (Free)
   - Requires sharing anonymized threat intelligence
   - Full access to marketplace with active sharing
   - Up to 3 concurrent streams
   - Community support
   
2. Community+ ($20/month) [Coming Soon]
   - For independent security researchers
   - Enhanced contribution rewards (2x-5x multipliers)
   - Researcher badges and recognition
   - Student discount: $10/month
   - Early access to experimental modules
   
3. Commercial ($5,000/year) [Coming Soon]
   - No mandatory intelligence sharing
   - Unlimited streams
   - Priority support
   - Volume discounts: $2,500 (2-5), $1,000 (6-10), $500 (11+)
   
4. Enterprise (Custom) [Coming Soon]
   - All Commercial features
   - Custom integrations
   - On-premise options
   - SLA guarantees

Current version: Community License with mandatory intel sharing
For more info: https://strigoi.io/licensing
`
}

func boolToStatus(b bool) string {
	if b {
		return "Enabled"
	}
	return "Disabled"
}