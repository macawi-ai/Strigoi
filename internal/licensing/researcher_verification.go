package licensing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ResearcherVerifier handles verification of Community+ researchers
type ResearcherVerifier struct {
	githubToken string
	httpClient  *http.Client
}

// NewResearcherVerifier creates a new researcher verifier
func NewResearcherVerifier(githubToken string) *ResearcherVerifier {
	return &ResearcherVerifier{
		githubToken: githubToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// VerifyResearcher verifies a researcher's credentials
func (v *ResearcherVerifier) VerifyResearcher(ctx context.Context, method string, identifier string) (*ResearcherVerification, error) {
	switch method {
	case "github":
		return v.verifyGitHub(ctx, identifier)
	case "academic":
		return v.verifyAcademic(identifier)
	case "linkedin":
		return v.verifyLinkedIn(identifier)
	default:
		return nil, fmt.Errorf("unsupported verification method: %s", method)
	}
}

// verifyGitHub verifies a researcher through their GitHub profile
func (v *ResearcherVerifier) verifyGitHub(ctx context.Context, username string) (*ResearcherVerification, error) {
	// Check GitHub profile
	profile, err := v.fetchGitHubProfile(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub profile: %w", err)
	}
	
	// Check for security-related repositories
	repos, err := v.fetchGitHubRepos(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}
	
	// Count security-related repos
	securityRepos := 0
	researchAreas := make(map[string]bool)
	
	for _, repo := range repos {
		if v.isSecurityRelated(repo) {
			securityRepos++
			areas := v.extractResearchAreas(repo)
			for _, area := range areas {
				researchAreas[area] = true
			}
		}
	}
	
	// Require at least 2 security-related repos
	if securityRepos < 2 {
		return nil, fmt.Errorf("insufficient security-related repositories (found %d, need 2+)", securityRepos)
	}
	
	// Build researcher verification
	areas := make([]string, 0, len(researchAreas))
	for area := range researchAreas {
		areas = append(areas, area)
	}
	
	return &ResearcherVerification{
		VerificationMethod: "github",
		VerificationID:     username,
		VerifiedAt:         time.Now(),
		GitHubProfile:      fmt.Sprintf("https://github.com/%s", username),
		ResearchAreas:      areas,
		BadgeLevel:         "bronze", // Start at bronze
		ContributionMultiplier: 2.0,  // 2x multiplier for GitHub-verified researchers
	}, nil
}

// verifyAcademic verifies a researcher through academic email
func (v *ResearcherVerifier) verifyAcademic(email string) (*ResearcherVerification, error) {
	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.edu$`)
	if !emailRegex.MatchString(email) {
		return nil, fmt.Errorf("invalid academic email format (must end with .edu)")
	}
	
	// Extract institution
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid email format")
	}
	
	domain := parts[1]
	
	// Check against known academic institutions
	if !v.isKnownAcademicDomain(domain) {
		return nil, fmt.Errorf("unrecognized academic institution: %s", domain)
	}
	
	return &ResearcherVerification{
		VerificationMethod: "academic",
		VerificationID:     domain,
		VerifiedAt:         time.Now(),
		AcademicEmail:      email,
		ResearchAreas:      []string{"academic_research"},
		BadgeLevel:         "bronze",
		ContributionMultiplier: 2.5,  // 2.5x multiplier for academic researchers
	}, nil
}

// verifyLinkedIn verifies a researcher through LinkedIn profile
func (v *ResearcherVerifier) verifyLinkedIn(profileURL string) (*ResearcherVerification, error) {
	// Basic LinkedIn URL validation
	linkedInRegex := regexp.MustCompile(`^https?://(?:www\.)?linkedin\.com/in/[\w-]+/?$`)
	if !linkedInRegex.MatchString(profileURL) {
		return nil, fmt.Errorf("invalid LinkedIn profile URL")
	}
	
	// Note: Actual LinkedIn verification would require OAuth or API access
	// For now, we'll do basic validation and manual review
	
	return &ResearcherVerification{
		VerificationMethod: "linkedin",
		VerificationID:     profileURL,
		VerifiedAt:         time.Now(),
		LinkedInProfile:    profileURL,
		ResearchAreas:      []string{"security_research"},
		BadgeLevel:         "bronze",
		ContributionMultiplier: 2.0,  // 2x multiplier for LinkedIn-verified researchers
	}, nil
}

// Helper methods

func (v *ResearcherVerifier) fetchGitHubProfile(ctx context.Context, username string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", username)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	if v.githubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", v.githubToken))
	}
	
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	
	var profile map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}
	
	return profile, nil
}

func (v *ResearcherVerifier) fetchGitHubRepos(ctx context.Context, username string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100", username)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	if v.githubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", v.githubToken))
	}
	
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	
	var repos []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}
	
	return repos, nil
}

func (v *ResearcherVerifier) isSecurityRelated(repo map[string]interface{}) bool {
	// Check repository name and description for security keywords
	securityKeywords := []string{
		"security", "vulnerability", "exploit", "pentest", "penetration",
		"ctf", "capture-the-flag", "hack", "audit", "scanner", "fuzzer",
		"malware", "reverse", "forensic", "crypto", "authentication",
		"authorization", "firewall", "ids", "ips", "siem", "threat",
		"incident", "response", "osint", "recon", "enumeration",
	}
	
	name, _ := repo["name"].(string)
	description, _ := repo["description"].(string)
	topics, _ := repo["topics"].([]interface{})
	
	// Check name and description
	combined := strings.ToLower(name + " " + description)
	for _, keyword := range securityKeywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}
	
	// Check topics
	for _, topic := range topics {
		topicStr, _ := topic.(string)
		for _, keyword := range securityKeywords {
			if strings.Contains(strings.ToLower(topicStr), keyword) {
				return true
			}
		}
	}
	
	return false
}

func (v *ResearcherVerifier) extractResearchAreas(repo map[string]interface{}) []string {
	areas := make(map[string]bool)
	
	// Map keywords to research areas
	areaMapping := map[string]string{
		"web":           "web_security",
		"network":       "network_security",
		"mobile":        "mobile_security",
		"cloud":         "cloud_security",
		"crypto":        "cryptography",
		"malware":       "malware_analysis",
		"forensic":      "digital_forensics",
		"reverse":       "reverse_engineering",
		"osint":         "osint",
		"ai":            "ai_security",
		"iot":           "iot_security",
		"blockchain":    "blockchain_security",
		"container":     "container_security",
		"kubernetes":    "kubernetes_security",
	}
	
	name, _ := repo["name"].(string)
	description, _ := repo["description"].(string)
	combined := strings.ToLower(name + " " + description)
	
	for keyword, area := range areaMapping {
		if strings.Contains(combined, keyword) {
			areas[area] = true
		}
	}
	
	result := make([]string, 0, len(areas))
	for area := range areas {
		result = append(result, area)
	}
	
	return result
}

func (v *ResearcherVerifier) isKnownAcademicDomain(domain string) bool {
	// This would ideally check against a comprehensive list
	// For now, we'll accept any .edu domain
	return strings.HasSuffix(domain, ".edu")
}

// UpdateResearcherBadge updates a researcher's badge level based on contributions
func UpdateResearcherBadge(verification *ResearcherVerification, contributions int, novelFindings int) {
	// Badge levels: bronze -> silver -> gold -> platinum
	
	if contributions >= 1000 && novelFindings >= 10 {
		verification.BadgeLevel = "platinum"
		verification.ContributionMultiplier = 5.0
	} else if contributions >= 500 && novelFindings >= 5 {
		verification.BadgeLevel = "gold"
		verification.ContributionMultiplier = 4.0
	} else if contributions >= 100 && novelFindings >= 2 {
		verification.BadgeLevel = "silver"
		verification.ContributionMultiplier = 3.0
	}
	// Bronze is the default starting level
}