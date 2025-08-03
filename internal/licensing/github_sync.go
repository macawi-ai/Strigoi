package licensing

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GitHubSync handles intelligence sharing via GitHub infrastructure
type GitHubSync struct {
	repoOwner    string
	repoName     string
	branch       string
	token        string // Optional, for authenticated requests
	client       *http.Client
}

// NewGitHubSync creates a new GitHub sync client
func NewGitHubSync(owner, repo, branch string) *GitHubSync {
	return &GitHubSync{
		repoOwner: owner,
		repoName:  repo,
		branch:    branch,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IntelligenceSubmission represents a batch of intelligence data
type IntelligenceSubmission struct {
	SubmissionID   string               `json:"submission_id"`
	Timestamp      time.Time            `json:"timestamp"`
	LicenseType    string               `json:"license_type"`
	SourceHash     string               `json:"source_hash"`
	Intelligence   []ThreatIntelligence `json:"intelligence"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// SubmitIntelligence submits intelligence data via GitHub
func (gs *GitHubSync) SubmitIntelligence(ctx context.Context, intel []ThreatIntelligence, sourceID string) error {
	submission := IntelligenceSubmission{
		SubmissionID: generateSubmissionID(),
		Timestamp:    time.Now().UTC(),
		LicenseType:  "community",
		SourceHash:   sourceID,
		Intelligence: intel,
		Metadata: map[string]interface{}{
			"version":    "1.0",
			"batch_size": len(intel),
		},
	}
	
	// Try multiple submission methods in order of preference
	
	// Method 1: GitHub Actions workflow dispatch
	if err := gs.submitViaWorkflow(ctx, submission); err == nil {
		return nil
	}
	
	// Method 2: Create issue in intelligence repository
	if err := gs.submitViaIssue(ctx, submission); err == nil {
		return nil
	}
	
	// Method 3: Push to intelligence branch (requires token)
	if gs.token != "" {
		if err := gs.submitViaBranch(ctx, submission); err == nil {
			return nil
		}
	}
	
	// Method 4: Use GitHub Discussions API
	if err := gs.submitViaDiscussion(ctx, submission); err == nil {
		return nil
	}
	
	return fmt.Errorf("all submission methods failed")
}

// submitViaWorkflow triggers a GitHub Actions workflow
func (gs *GitHubSync) submitViaWorkflow(ctx context.Context, submission IntelligenceSubmission) error {
	// Workflow dispatch API
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/process-intelligence.yml/dispatches",
		gs.repoOwner, gs.repoName)
	
	payload := map[string]interface{}{
		"ref": gs.branch,
		"inputs": map[string]string{
			"intelligence_data": gs.encodeSubmission(submission),
			"submission_id":     submission.SubmissionID,
		},
	}
	
	return gs.makeGitHubRequest(ctx, "POST", url, payload)
}

// submitViaIssue creates an issue with intelligence data
func (gs *GitHubSync) submitViaIssue(ctx context.Context, submission IntelligenceSubmission) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", gs.repoOwner, gs.repoName)
	
	// Create issue with encoded data
	issueBody := fmt.Sprintf(`## Intelligence Submission

**Submission ID**: %s
**Timestamp**: %s
**Source**: %s
**Count**: %d

### Encoded Data
%s%s%s

---
*This is an automated intelligence submission from Strigoi Community Edition*
`, 
		submission.SubmissionID,
		submission.Timestamp.Format(time.RFC3339),
		submission.SourceHash,
		len(submission.Intelligence),
		"```json\n",
		gs.encodeSubmission(submission),
		"\n```")
	
	payload := map[string]interface{}{
		"title":  fmt.Sprintf("[Intel] Submission %s", submission.SubmissionID[:8]),
		"body":   issueBody,
		"labels": []string{"intelligence", "community", "automated"},
	}
	
	return gs.makeGitHubRequest(ctx, "POST", url, payload)
}

// submitViaBranch pushes intelligence to a dedicated branch
func (gs *GitHubSync) submitViaBranch(ctx context.Context, submission IntelligenceSubmission) error {
	// Get current commit SHA for the branch
	branchURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs/heads/%s",
		gs.repoOwner, gs.repoName, gs.branch)
	
	resp, err := gs.client.Get(branchURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	var branchInfo struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&branchInfo); err != nil {
		return err
	}
	
	// Create blob with intelligence data
	blobURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/blobs", gs.repoOwner, gs.repoName)
	blobPayload := map[string]interface{}{
		"content":  gs.encodeSubmission(submission),
		"encoding": "base64",
	}
	
	var blobResp struct {
		SHA string `json:"sha"`
	}
	
	if err := gs.makeGitHubRequestWithResponse(ctx, "POST", blobURL, blobPayload, &blobResp); err != nil {
		return err
	}
	
	// Create tree
	treeURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees", gs.repoOwner, gs.repoName)
	treePath := fmt.Sprintf("intelligence/%s/%s.json",
		submission.Timestamp.Format("2006-01-02"),
		submission.SubmissionID)
	
	treePayload := map[string]interface{}{
		"base_tree": branchInfo.Object.SHA,
		"tree": []map[string]interface{}{
			{
				"path": treePath,
				"mode": "100644",
				"type": "blob",
				"sha":  blobResp.SHA,
			},
		},
	}
	
	var treeResp struct {
		SHA string `json:"sha"`
	}
	
	if err := gs.makeGitHubRequestWithResponse(ctx, "POST", treeURL, treePayload, &treeResp); err != nil {
		return err
	}
	
	// Create commit
	commitURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/commits", gs.repoOwner, gs.repoName)
	commitPayload := map[string]interface{}{
		"message": fmt.Sprintf("Intelligence submission: %s", submission.SubmissionID),
		"tree":    treeResp.SHA,
		"parents": []string{branchInfo.Object.SHA},
	}
	
	var commitResp struct {
		SHA string `json:"sha"`
	}
	
	if err := gs.makeGitHubRequestWithResponse(ctx, "POST", commitURL, commitPayload, &commitResp); err != nil {
		return err
	}
	
	// Update branch reference
	updatePayload := map[string]interface{}{
		"sha": commitResp.SHA,
	}
	
	return gs.makeGitHubRequest(ctx, "PATCH", branchURL, updatePayload)
}

// submitViaDiscussion creates a discussion with intelligence data
func (gs *GitHubSync) submitViaDiscussion(ctx context.Context, submission IntelligenceSubmission) error {
	// GraphQL API for discussions
	graphQLURL := "https://api.github.com/graphql"
	
	// First, get repository ID and category ID
	repoQuery := `
	query($owner: String!, $repo: String!) {
		repository(owner: $owner, name: $repo) {
			id
			discussionCategories(first: 10) {
				nodes {
					id
					name
				}
			}
		}
	}`
	
	variables := map[string]interface{}{
		"owner": gs.repoOwner,
		"repo":  gs.repoName,
	}
	
	// Would implement full GraphQL flow here
	// For now, return error to try next method
	return fmt.Errorf("discussions API not fully implemented")
}

// FetchMarketplaceUpdates retrieves updates based on contribution level
func (gs *GitHubSync) FetchMarketplaceUpdates(ctx context.Context, accessLevel MarketplaceAccess, lastSync time.Time) ([]MarketplaceUpdate, error) {
	// Determine which release channel to use based on access level
	channel := gs.getChannelForAccess(accessLevel)
	
	// Fetch releases from GitHub
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", gs.repoOwner, gs.repoName)
	
	resp, err := gs.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	
	// Filter releases based on access level and timestamp
	updates := make([]MarketplaceUpdate, 0)
	for _, release := range releases {
		// Check if release is newer than last sync
		publishedAt, _ := time.Parse(time.RFC3339, release.PublishedAt)
		if publishedAt.Before(lastSync) {
			continue
		}
		
		// Check if user has access to this release
		if !gs.hasAccessToRelease(release, accessLevel) {
			continue
		}
		
		// Find intelligence data asset
		for _, asset := range release.Assets {
			if asset.Name == fmt.Sprintf("intelligence-%s.json", channel) {
				update := MarketplaceUpdate{
					ID:          release.ID,
					Name:        release.Name,
					Version:     release.TagName,
					PublishedAt: publishedAt,
					Channel:     channel,
					DownloadURL: asset.BrowserDownloadURL,
					Size:        asset.Size,
				}
				updates = append(updates, update)
				break
			}
		}
	}
	
	return updates, nil
}

// Helper methods

func (gs *GitHubSync) encodeSubmission(submission IntelligenceSubmission) string {
	data, _ := json.MarshalIndent(submission, "", "  ")
	return base64.StdEncoding.EncodeToString(data)
}

func (gs *GitHubSync) makeGitHubRequest(ctx context.Context, method, url string, payload interface{}) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(data)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")
	
	if gs.token != "" {
		req.Header.Set("Authorization", "token "+gs.token)
	}
	
	resp, err := gs.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (gs *GitHubSync) makeGitHubRequestWithResponse(ctx context.Context, method, url string, payload, response interface{}) error {
	if err := gs.makeGitHubRequest(ctx, method, url, payload); err != nil {
		return err
	}
	
	// Make request again to get response
	// (In production, would modify makeGitHubRequest to return response)
	var body io.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		body = bytes.NewBuffer(data)
	}
	
	req, _ := http.NewRequestWithContext(ctx, method, url, body)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")
	
	if gs.token != "" {
		req.Header.Set("Authorization", "token "+gs.token)
	}
	
	resp, err := gs.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	return json.NewDecoder(resp.Body).Decode(response)
}

func (gs *GitHubSync) getChannelForAccess(level MarketplaceAccess) string {
	switch level {
	case MarketplacePremium, MarketplaceUnlimited:
		return "premium"
	case MarketplaceStandard:
		return "standard"
	default:
		return "community"
	}
}

func (gs *GitHubSync) hasAccessToRelease(release GitHubRelease, level MarketplaceAccess) bool {
	// Check release tags/labels for access requirements
	if release.Prerelease && level != MarketplacePremium && level != MarketplaceUnlimited {
		return false
	}
	
	// Could implement more sophisticated access control
	return true
}

func generateSubmissionID() string {
	// Generate unique submission ID
	return fmt.Sprintf("SUB-%d-%s", time.Now().Unix(), randomHex(4))
}

func randomHex(n int) string {
	// In production, use crypto/rand
	return "abcd1234"[:n*2]
}

// Types for GitHub API

type GitHubRelease struct {
	ID          int              `json:"id"`
	TagName     string           `json:"tag_name"`
	Name        string           `json:"name"`
	Prerelease  bool            `json:"prerelease"`
	PublishedAt string          `json:"published_at"`
	Assets      []GitHubAsset   `json:"assets"`
}

type GitHubAsset struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Size               int    `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type MarketplaceUpdate struct {
	ID          int
	Name        string
	Version     string
	PublishedAt time.Time
	Channel     string
	DownloadURL string
	Size        int
}