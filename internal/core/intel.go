package core

import (
	"fmt"
	"time"
)

// IntelReport represents threat intelligence to be shared
type IntelReport struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Type         string    `json:"type"` // attack, vulnerability, configuration
	Severity     string    `json:"severity"`
	Anonymized   bool      `json:"anonymized"`
	DataPoints   int       `json:"data_points"`
	Contributors int       `json:"contributors"`
}

// IntelManager handles intelligence collection and sharing
type IntelManager struct {
	license      *License
	reports      []IntelReport
	lastSync     time.Time
	pointsEarned int
}

// NewIntelManager creates a new intelligence manager
func NewIntelManager(license *License) *IntelManager {
	return &IntelManager{
		license:  license,
		reports:  make([]IntelReport, 0),
		lastSync: time.Now(),
	}
}

// GetIntelStatus returns current intelligence sharing status
func (im *IntelManager) GetIntelStatus() string {
	if !im.license.IntelSharing.Required {
		return "Intelligence sharing: Optional (Commercial License)"
	}
	
	status := fmt.Sprintf(`
Intelligence Sharing Status (Community License)
==============================================

Last Sync: %s
Reports Pending: %d
Points Earned: %d
Contribution Level: %s

Next Actions:
- Reports will be anonymized and shared on next marketplace sync
- Your contributions unlock continued marketplace access
- Higher quality intel earns more points

Privacy Protection:
- All internal IPs, hostnames, and PII are automatically removed
- Attack patterns are generalized to protect your infrastructure
- You maintain full control over what gets shared

`, im.lastSync.Format("2006-01-02 15:04:05"),
		len(im.reports),
		im.pointsEarned,
		im.getContributionLevel())
		
	return status
}

// getContributionLevel returns the user's contribution tier
func (im *IntelManager) getContributionLevel() string {
	switch {
	case im.pointsEarned >= 10000:
		return "Platinum Contributor"
	case im.pointsEarned >= 5000:
		return "Gold Contributor"
	case im.pointsEarned >= 1000:
		return "Silver Contributor"
	case im.pointsEarned >= 100:
		return "Bronze Contributor"
	default:
		return "New Contributor"
	}
}

// SimulateIntelCollection simulates collecting threat intelligence
func (im *IntelManager) SimulateIntelCollection() string {
	// This is a stub - real implementation would collect from streams
	newReports := []IntelReport{
		{
			ID:           fmt.Sprintf("intel-%d", time.Now().Unix()),
			Timestamp:    time.Now(),
			Type:         "attack",
			Severity:     "high",
			Anonymized:   true,
			DataPoints:   5,
			Contributors: 1,
		},
		{
			ID:           fmt.Sprintf("intel-%d-2", time.Now().Unix()),
			Timestamp:    time.Now(),
			Type:         "vulnerability",
			Severity:     "medium",
			Anonymized:   true,
			DataPoints:   3,
			Contributors: 1,
		},
	}
	
	im.reports = append(im.reports, newReports...)
	im.pointsEarned += 80 // Simulate earning points
	
	return fmt.Sprintf(`
Intelligence Collection Simulation
==================================

Collected 2 new intelligence reports:
- 1 high-severity attack pattern
- 1 medium-severity vulnerability

All data has been anonymized according to privacy standards.
Ready for sharing on next marketplace sync.

Points earned: +80
Total points: %d
`, im.pointsEarned)
}