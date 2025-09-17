package probe

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/macawi-ai/strigoi/pkg/modules"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

func init() {
	modules.RegisterBuiltin("probe/west", NewWestModule)
}

// WestModule analyzes authentication and access control mechanisms.
type WestModule struct {
	modules.BaseModule

	// Core detection components
	authEndpoints   []AuthEndpoint
	sessionPatterns []SessionPattern
	authzMatrix     map[string]AccessRequirement
	vulnerabilities []AuthVulnerability

	// Security and operational components
	config         WestConfig
	rateLimiter    *rate.Limiter
	circuitBreaker *CircuitBreaker
	tlsClient      *http.Client

	// Execution control
	ctx       context.Context
	cancel    context.CancelFunc
	semaphore chan struct{}
	mu        sync.Mutex
}

// WestConfig holds configuration for the west probe.
type WestConfig struct {
	RateLimit       float64       `json:"rate_limit"`
	RequestTimeout  time.Duration `json:"request_timeout"`
	MaxConcurrency  int           `json:"max_concurrency"`
	DryRun          bool          `json:"dry_run"`
	FollowRedirects bool          `json:"follow_redirects"`
	MaxRedirects    int           `json:"max_redirects"`
	AllowPrivate    bool          `json:"allow_private"`
}

// AuthEndpoint represents an authentication endpoint.
type AuthEndpoint struct {
	URL             string              `json:"url"`
	Method          string              `json:"method"`
	AuthType        string              `json:"auth_type"`
	RequiresAuth    bool                `json:"requires_auth"`
	MFAEnabled      bool                `json:"mfa_enabled"`
	Headers         map[string]string   `json:"headers,omitempty"`
	Vulnerabilities []AuthVulnerability `json:"vulnerabilities,omitempty"`
}

// SessionPattern represents a session management pattern.
type SessionPattern struct {
	Type       string   `json:"type"`
	Pattern    string   `json:"pattern"`
	Secure     bool     `json:"secure"`
	HTTPOnly   bool     `json:"httponly"`
	SameSite   string   `json:"samesite"`
	Weaknesses []string `json:"weaknesses,omitempty"`
}

// AccessRequirement defines access control requirements.
type AccessRequirement struct {
	Resource   string   `json:"resource"`
	Methods    []string `json:"methods"`
	Roles      []string `json:"roles"`
	Conditions []string `json:"conditions,omitempty"`
}

// AuthVulnerability represents an authentication/authorization vulnerability.
type AuthVulnerability struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Evidence    Evidence  `json:"evidence"`
	Remediation string    `json:"remediation"`
	References  []string  `json:"references,omitempty"`
	Confidence  float64   `json:"confidence"`
	Timestamp   time.Time `json:"timestamp"`
}

// Evidence contains proof of a vulnerability.
type Evidence struct {
	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
	Details  string `json:"details,omitempty"`
}

// CircuitBreaker implements a simple circuit breaker pattern.
type CircuitBreaker struct {
	mu          sync.Mutex
	failures    int
	threshold   int
	lastFailure time.Time
	timeout     time.Duration
	state       string // "closed", "open", "half-open"
}

// NewWestModule creates a new authentication probe module.
func NewWestModule() modules.Module {
	return &WestModule{
		BaseModule: modules.BaseModule{
			ModuleName:        "probe/west",
			ModuleDescription: "Authentication and access control analysis",
			ModuleType:        modules.ProbeModule,
			ModuleOptions: map[string]*modules.ModuleOption{
				"target": {
					Name:        "target",
					Description: "Target URL or domain to analyze",
					Required:    true,
					Type:        "string",
				},
				"timeout": {
					Name:        "timeout",
					Description: "Request timeout in seconds",
					Required:    false,
					Type:        "int",
					Default:     30,
				},
				"rate_limit": {
					Name:        "rate_limit",
					Description: "Requests per second",
					Required:    false,
					Type:        "float",
					Default:     10.0,
				},
				"dry_run": {
					Name:        "dry_run",
					Description: "Perform passive analysis only",
					Required:    false,
					Type:        "bool",
					Default:     false,
				},
				"max_concurrent": {
					Name:        "max_concurrent",
					Description: "Maximum concurrent requests",
					Required:    false,
					Type:        "int",
					Default:     5,
				},
				"allow_private": {
					Name:        "allow_private",
					Description: "Allow scanning private/local addresses",
					Required:    false,
					Type:        "bool",
					Default:     false,
				},
			},
		},
		authEndpoints:   []AuthEndpoint{},
		sessionPatterns: []SessionPattern{},
		authzMatrix:     make(map[string]AccessRequirement),
		vulnerabilities: []AuthVulnerability{},
	}
}

// Check verifies the module can run.
func (m *WestModule) Check() bool {
	// Verify we can make HTTPS requests
	return true
}

// Configure sets up the module before execution.
func (m *WestModule) Configure() error {
	// Parse configuration from options
	timeout := 30
	if t, ok := m.ModuleOptions["timeout"]; ok && t.Value != nil {
		if tInt, ok := t.Value.(int); ok {
			timeout = tInt
		}
	} else if t, ok := m.ModuleOptions["timeout"]; ok && t.Default != nil {
		if tInt, ok := t.Default.(int); ok {
			timeout = tInt
		}
	}

	rateLimit := 10.0
	if r, ok := m.ModuleOptions["rate_limit"]; ok && r.Value != nil {
		if rFloat, ok := r.Value.(float64); ok {
			rateLimit = rFloat
		}
	} else if r, ok := m.ModuleOptions["rate_limit"]; ok && r.Default != nil {
		if rFloat, ok := r.Default.(float64); ok {
			rateLimit = rFloat
		}
	}

	maxConcurrent := 5
	if mc, ok := m.ModuleOptions["max_concurrent"]; ok && mc.Value != nil {
		if mcInt, ok := mc.Value.(int); ok {
			maxConcurrent = mcInt
		}
	} else if mc, ok := m.ModuleOptions["max_concurrent"]; ok && mc.Default != nil {
		if mcInt, ok := mc.Default.(int); ok {
			maxConcurrent = mcInt
		}
	}

	dryRun := false
	if dr, ok := m.ModuleOptions["dry_run"]; ok && dr.Value != nil {
		if drBool, ok := dr.Value.(bool); ok {
			dryRun = drBool
		}
	}

	allowPrivate := false
	if ap, ok := m.ModuleOptions["allow_private"]; ok && ap.Value != nil {
		switch v := ap.Value.(type) {
		case bool:
			allowPrivate = v
		case string:
			if parsed, err := strconv.ParseBool(v); err == nil {
				allowPrivate = parsed
			}
		}
	} else if ap, ok := m.ModuleOptions["allow_private"]; ok && ap.Default != nil {
		if apBool, ok := ap.Default.(bool); ok {
			allowPrivate = apBool
		}
	}

	m.config = WestConfig{
		RateLimit:       rateLimit,
		RequestTimeout:  time.Duration(timeout) * time.Second,
		MaxConcurrency:  maxConcurrent,
		DryRun:          dryRun,
		FollowRedirects: false, // Don't follow redirects during auth testing
		MaxRedirects:    5,
		AllowPrivate:    allowPrivate,
	}

	// Initialize rate limiter
	m.rateLimiter = rate.NewLimiter(rate.Limit(rateLimit), int(rateLimit))

	// Initialize circuit breaker
	m.circuitBreaker = &CircuitBreaker{
		threshold: 5,
		timeout:   30 * time.Second,
		state:     "closed",
	}

	// Initialize semaphore for concurrency control
	m.semaphore = make(chan struct{}, maxConcurrent)

	// Configure secure TLS client
	m.configureTLSClient()

	return nil
}

// configureTLSClient sets up a secure HTTP client.
func (m *WestModule) configureTLSClient() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			},
			InsecureSkipVerify: false,
		},
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	}

	// Don't follow redirects automatically
	checkRedirect := func(_ *http.Request, via []*http.Request) error {
		if len(via) >= m.config.MaxRedirects {
			return fmt.Errorf("too many redirects")
		}
		if !m.config.FollowRedirects {
			return http.ErrUseLastResponse
		}
		return nil
	}

	m.tlsClient = &http.Client{
		Transport:     tr,
		Timeout:       m.config.RequestTimeout,
		CheckRedirect: checkRedirect,
	}
}

// Run executes the authentication analysis.
func (m *WestModule) Run() (*modules.ModuleResult, error) {
	// Configure module
	if err := m.Configure(); err != nil {
		return &modules.ModuleResult{
			Module: m.Name(),
			Status: "failed",
			Error:  err.Error(),
		}, err
	}

	// Validate options
	if err := m.ValidateOptions(); err != nil {
		return &modules.ModuleResult{
			Module: m.Name(),
			Status: "failed",
			Error:  err.Error(),
		}, err
	}

	// Get target
	target, ok := m.ModuleOptions["target"]
	if !ok || target.Value == nil {
		return &modules.ModuleResult{
			Module: m.Name(),
			Status: "failed",
			Error:  "target not set",
		}, fmt.Errorf("target not set")
	}
	targetStr := fmt.Sprintf("%v", target.Value)

	// Validate target URL
	if err := m.validateTarget(targetStr); err != nil {
		return &modules.ModuleResult{
			Module: m.Name(),
			Status: "failed",
			Error:  err.Error(),
		}, err
	}

	// Initialize context
	m.ctx, m.cancel = context.WithCancel(context.Background())
	defer m.cancel()

	// Create error group for concurrent operations
	g, ctx := errgroup.WithContext(m.ctx)

	startTime := time.Now()

	// Phase 1: Passive detection
	g.Go(func() error {
		return m.passiveDetection(ctx, targetStr)
	})

	// Phase 2: Active probing (if not dry run)
	if !m.config.DryRun {
		g.Go(func() error {
			return m.activeProbing(ctx, targetStr)
		})
	}

	// Phase 3: Pattern analysis
	g.Go(func() error {
		return m.patternAnalysis(ctx, targetStr)
	})

	// Wait for all phases to complete
	if err := g.Wait(); err != nil {
		return &modules.ModuleResult{
			Module:    m.Name(),
			Status:    "failed",
			StartTime: startTime,
			EndTime:   time.Now(),
			Error:     err.Error(),
			Data:      m.getResults(),
		}, err
	}

	// Aggregate results
	results := m.getResults()

	return &modules.ModuleResult{
		Module:    m.Name(),
		Status:    "completed",
		StartTime: startTime,
		EndTime:   time.Now(),
		Data: map[string]interface{}{
			"target":          targetStr,
			"auth_endpoints":  results["auth_endpoints"],
			"session_info":    results["session_info"],
			"access_control":  results["access_control"],
			"vulnerabilities": results["vulnerabilities"],
			"statistics":      results["statistics"],
			"recommendations": m.generateRecommendations(),
		},
	}, nil
}

// validateTarget ensures the target URL is valid and safe.
func (m *WestModule) validateTarget(target string) error {
	// Ensure target has a scheme
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}

	parsedURL, err := url.ParseRequestURI(target)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	// Only allow HTTP/HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s", parsedURL.Scheme)
	}

	// Validate hostname
	if parsedURL.Hostname() == "" {
		return fmt.Errorf("missing hostname in URL")
	}

	// Check for local/private addresses (security measure)
	if !m.config.AllowPrivate && m.isPrivateAddress(parsedURL.Hostname()) {
		return fmt.Errorf("scanning private/local addresses is not allowed")
	}

	return nil
}

// isPrivateAddress checks if a hostname refers to a private IP.
func (m *WestModule) isPrivateAddress(hostname string) bool {
	// First check if it's an IP address
	ip := net.ParseIP(hostname)
	if ip != nil {
		// Check if it's a private IP
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return true
		}
	}

	// Check common private hostnames
	privateHostnames := []string{
		"localhost",
		"localhost.localdomain",
	}

	for _, private := range privateHostnames {
		if strings.EqualFold(hostname, private) {
			return true
		}
	}

	return false
}

// passiveDetection performs non-intrusive analysis.
func (m *WestModule) passiveDetection(ctx context.Context, target string) error {
	// Ensure target has scheme
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}

	// Parse base URL
	baseURL, err := url.Parse(target)
	if err != nil {
		return err
	}

	// Discover common auth endpoints
	authPaths := []string{
		"/login", "/signin", "/auth", "/authenticate",
		"/api/login", "/api/auth", "/api/v1/auth",
		"/oauth/authorize", "/oauth/token",
		"/.well-known/openid-configuration",
		"/saml/login", "/saml/metadata",
		"/wp-login.php", "/admin", "/administrator",
	}

	for _, path := range authPaths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			checkURL := *baseURL
			checkURL.Path = path
			endpoint := m.checkAuthEndpoint(ctx, checkURL.String())
			if endpoint != nil {
				m.mu.Lock()
				m.authEndpoints = append(m.authEndpoints, *endpoint)
				m.mu.Unlock()
			}
		}
	}

	return nil
}

// checkAuthEndpoint checks if an endpoint exists and its auth type.
func (m *WestModule) checkAuthEndpoint(ctx context.Context, fullURL string) *AuthEndpoint {
	// Rate limiting
	if err := m.rateLimiter.Wait(ctx); err != nil {
		return nil
	}

	// Circuit breaker check
	if !m.circuitBreaker.Allow() {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Strigoi/1.0)")

	resp, err := m.tlsClient.Do(req)
	if err != nil {
		m.circuitBreaker.RecordFailure()
		return nil
	}
	defer resp.Body.Close()

	m.circuitBreaker.RecordSuccess()

	// Check if endpoint exists
	if resp.StatusCode == 404 {
		return nil
	}

	endpoint := &AuthEndpoint{
		URL:     fullURL,
		Method:  "GET",
		Headers: make(map[string]string),
	}

	// Analyze response headers and body to determine auth type
	m.analyzeAuthType(endpoint, resp)

	return endpoint
}

// analyzeAuthType determines the authentication type from response.
func (m *WestModule) analyzeAuthType(endpoint *AuthEndpoint, resp *http.Response) {
	// Check for authentication headers
	if resp.Header.Get("WWW-Authenticate") != "" {
		authHeader := resp.Header.Get("WWW-Authenticate")
		if strings.HasPrefix(authHeader, "Basic") {
			endpoint.AuthType = "Basic Auth"
		} else if strings.HasPrefix(authHeader, "Bearer") {
			endpoint.AuthType = "Bearer Token"
		} else if strings.HasPrefix(authHeader, "Digest") {
			endpoint.AuthType = "Digest Auth"
		}
		endpoint.RequiresAuth = true
	}

	// Check for common auth indicators
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		endpoint.RequiresAuth = true
	}

	// Check for OAuth indicators
	if strings.Contains(endpoint.URL, "oauth") || strings.Contains(endpoint.URL, "authorize") {
		endpoint.AuthType = "OAuth"
	}

	// Check for SAML indicators
	if strings.Contains(endpoint.URL, "saml") {
		endpoint.AuthType = "SAML"
	}

	// Store security headers
	securityHeaders := []string{
		"Strict-Transport-Security",
		"Content-Security-Policy",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
	}

	for _, header := range securityHeaders {
		if value := resp.Header.Get(header); value != "" {
			endpoint.Headers[header] = value
		}
	}
}

// activeProbing performs more intrusive tests.
func (m *WestModule) activeProbing(ctx context.Context, _ string) error {
	// Only probe discovered endpoints
	m.mu.Lock()
	endpoints := make([]AuthEndpoint, len(m.authEndpoints))
	copy(endpoints, m.authEndpoints)
	m.mu.Unlock()

	for i := range endpoints {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Test each endpoint for vulnerabilities
			vulns := m.testEndpointSecurity(ctx, &endpoints[i])
			if len(vulns) > 0 {
				endpoints[i].Vulnerabilities = vulns

				m.mu.Lock()
				m.vulnerabilities = append(m.vulnerabilities, vulns...)
				m.mu.Unlock()
			}
		}
	}

	// Update endpoints with vulnerability info
	m.mu.Lock()
	m.authEndpoints = endpoints
	m.mu.Unlock()

	return nil
}

// testEndpointSecurity tests an endpoint for common vulnerabilities.
func (m *WestModule) testEndpointSecurity(ctx context.Context, endpoint *AuthEndpoint) []AuthVulnerability {
	var vulns []AuthVulnerability

	// Test for missing security headers
	if _, ok := endpoint.Headers["Strict-Transport-Security"]; !ok {
		vulns = append(vulns, AuthVulnerability{
			ID:          fmt.Sprintf("WEST-%d", time.Now().Unix()),
			Name:        "Missing HSTS Header",
			Severity:    "Medium",
			Category:    "Transport Security",
			Description: fmt.Sprintf("Endpoint %s lacks HSTS header", endpoint.URL),
			Remediation: "Add Strict-Transport-Security header with appropriate max-age",
			References:  []string{"OWASP-TRANSPORT"},
			Confidence:  1.0,
			Timestamp:   time.Now(),
		})
	}

	// Test for authentication bypass attempts (carefully)
	if endpoint.RequiresAuth {
		// Test common bypass techniques
		bypassTests := []struct {
			name     string
			modifier func(*http.Request)
		}{
			{
				name: "X-Forwarded-For bypass",
				modifier: func(req *http.Request) {
					req.Header.Set("X-Forwarded-For", "127.0.0.1")
				},
			},
			{
				name: "X-Real-IP bypass",
				modifier: func(req *http.Request) {
					req.Header.Set("X-Real-IP", "127.0.0.1")
				},
			},
			{
				name: "X-Originating-IP bypass",
				modifier: func(req *http.Request) {
					req.Header.Set("X-Originating-IP", "127.0.0.1")
				},
			},
			{
				name: "Authorization header manipulation",
				modifier: func(req *http.Request) {
					req.Header.Set("Authorization", "Bearer null")
				},
			},
		}

		for _, test := range bypassTests {
			if vuln := m.testAuthBypass(ctx, endpoint, test.name, test.modifier); vuln != nil {
				vulns = append(vulns, *vuln)
			}
		}
	}

	// Test for CSRF protection
	if endpoint.Method == "POST" || endpoint.Method == "PUT" || endpoint.Method == "DELETE" {
		if vuln := m.testCSRFProtection(ctx, endpoint); vuln != nil {
			vulns = append(vulns, *vuln)
		}
	}

	// Test for session fixation
	if vuln := m.testSessionFixation(ctx, endpoint); vuln != nil {
		vulns = append(vulns, *vuln)
	}

	return vulns
}

// testAuthBypass tests for authentication bypass vulnerabilities.
func (m *WestModule) testAuthBypass(ctx context.Context, endpoint *AuthEndpoint, testName string, modifier func(*http.Request)) *AuthVulnerability {
	// Rate limiting
	if err := m.rateLimiter.Wait(ctx); err != nil {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, endpoint.Method, endpoint.URL, nil)
	if err != nil {
		return nil
	}

	// Apply bypass technique
	modifier(req)

	resp, err := m.tlsClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	// Check if bypass was successful
	if resp.StatusCode == 200 || resp.StatusCode == 302 {
		return &AuthVulnerability{
			ID:          fmt.Sprintf("WEST-%d", time.Now().Unix()),
			Name:        fmt.Sprintf("Authentication Bypass - %s", testName),
			Severity:    "Critical",
			Category:    "Authentication",
			Description: fmt.Sprintf("Authentication can be bypassed using %s technique", testName),
			Evidence: Evidence{
				Request:  fmt.Sprintf("%s %s", req.Method, req.URL.String()),
				Response: fmt.Sprintf("Status: %d", resp.StatusCode),
			},
			Remediation: "Implement proper authentication checks that cannot be bypassed with header manipulation",
			References:  []string{"CWE-290", "OWASP-AUTH"},
			Confidence:  0.8,
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// testCSRFProtection tests for CSRF vulnerabilities.
func (m *WestModule) testCSRFProtection(ctx context.Context, endpoint *AuthEndpoint) *AuthVulnerability {
	// Rate limiting
	if err := m.rateLimiter.Wait(ctx); err != nil {
		return nil
	}

	// Test without CSRF token
	req, err := http.NewRequestWithContext(ctx, endpoint.Method, endpoint.URL, nil)
	if err != nil {
		return nil
	}

	// Remove common CSRF headers
	req.Header.Del("X-CSRF-Token")
	req.Header.Del("X-XSRF-Token")
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Referer", "https://evil.com")

	resp, err := m.tlsClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	// If request succeeds without CSRF token, it's vulnerable
	if resp.StatusCode == 200 || resp.StatusCode == 302 {
		return &AuthVulnerability{
			ID:          fmt.Sprintf("WEST-%d", time.Now().Unix()),
			Name:        "Missing CSRF Protection",
			Severity:    "High",
			Category:    "Session Management",
			Description: fmt.Sprintf("Endpoint %s accepts state-changing requests without CSRF protection", endpoint.URL),
			Evidence: Evidence{
				Request:  fmt.Sprintf("%s request from evil.com origin", endpoint.Method),
				Response: fmt.Sprintf("Status: %d (request accepted)", resp.StatusCode),
			},
			Remediation: "Implement CSRF tokens for all state-changing operations",
			References:  []string{"CWE-352", "OWASP-CSRF"},
			Confidence:  0.9,
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// testSessionFixation tests for session fixation vulnerabilities.
func (m *WestModule) testSessionFixation(ctx context.Context, endpoint *AuthEndpoint) *AuthVulnerability {
	// Rate limiting
	if err := m.rateLimiter.Wait(ctx); err != nil {
		return nil
	}

	// First request to get initial session
	req1, err := http.NewRequestWithContext(ctx, "GET", endpoint.URL, nil)
	if err != nil {
		return nil
	}

	resp1, err := m.tlsClient.Do(req1)
	if err != nil {
		return nil
	}
	defer resp1.Body.Close()

	// Check if session cookie was set
	var sessionCookie *http.Cookie
	for _, cookie := range resp1.Cookies() {
		if strings.Contains(strings.ToLower(cookie.Name), "sess") ||
			strings.Contains(strings.ToLower(cookie.Name), "sid") {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		return nil // No session cookie found
	}

	// Try to use the same session ID after "authentication"
	req2, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, nil)
	if err != nil {
		return nil
	}
	req2.AddCookie(sessionCookie)

	resp2, err := m.tlsClient.Do(req2)
	if err != nil {
		return nil
	}
	defer resp2.Body.Close()

	// Check if the same session ID is still valid
	for _, cookie := range resp2.Cookies() {
		if cookie.Name == sessionCookie.Name && cookie.Value == sessionCookie.Value {
			return &AuthVulnerability{
				ID:          fmt.Sprintf("WEST-%d", time.Now().Unix()),
				Name:        "Session Fixation Vulnerability",
				Severity:    "High",
				Category:    "Session Management",
				Description: "Session ID remains unchanged after authentication, allowing session fixation attacks",
				Evidence: Evidence{
					Details: fmt.Sprintf("Session cookie '%s' not regenerated after authentication", sessionCookie.Name),
				},
				Remediation: "Regenerate session IDs after successful authentication",
				References:  []string{"CWE-384", "OWASP-SESSION"},
				Confidence:  0.7,
				Timestamp:   time.Now(),
			}
		}
	}

	return nil
}

// patternAnalysis analyzes authentication patterns.
func (m *WestModule) patternAnalysis(ctx context.Context, target string) error {
	// Analyze session management patterns
	m.analyzeSessionManagement(ctx, target)

	// Build access control matrix
	m.buildAccessControlMatrix()

	return nil
}

// analyzeSessionManagement checks session security.
func (m *WestModule) analyzeSessionManagement(ctx context.Context, target string) {
	// Ensure target has scheme
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}

	// Test root path for cookies
	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return
	}

	resp, err := m.tlsClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Analyze cookies
	for _, cookie := range resp.Cookies() {
		sameSiteStr := ""
		switch cookie.SameSite {
		case http.SameSiteLaxMode:
			sameSiteStr = "Lax"
		case http.SameSiteStrictMode:
			sameSiteStr = "Strict"
		case http.SameSiteNoneMode:
			sameSiteStr = "None"
		default:
			sameSiteStr = "Default"
		}

		pattern := SessionPattern{
			Type:     "Cookie",
			Pattern:  cookie.Name,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HttpOnly,
			SameSite: sameSiteStr,
		}

		// Check for weaknesses
		if !cookie.Secure {
			pattern.Weaknesses = append(pattern.Weaknesses, "Missing Secure flag")
		}
		if !cookie.HttpOnly {
			pattern.Weaknesses = append(pattern.Weaknesses, "Missing HttpOnly flag")
		}
		if cookie.SameSite == http.SameSiteNoneMode {
			pattern.Weaknesses = append(pattern.Weaknesses, "SameSite=None may allow CSRF")
		}

		m.mu.Lock()
		m.sessionPatterns = append(m.sessionPatterns, pattern)
		m.mu.Unlock()

		// Create vulnerability if weaknesses found
		if len(pattern.Weaknesses) > 0 {
			vuln := AuthVulnerability{
				ID:          fmt.Sprintf("WEST-%d", time.Now().Unix()),
				Name:        "Insecure Session Cookie Configuration",
				Severity:    "Medium",
				Category:    "Session Management",
				Description: fmt.Sprintf("Cookie '%s' has security weaknesses: %s", cookie.Name, strings.Join(pattern.Weaknesses, ", ")),
				Remediation: "Set Secure and HttpOnly flags on all session cookies",
				References:  []string{"CWE-614", "CWE-1004"},
				Confidence:  1.0,
				Timestamp:   time.Now(),
			}

			m.mu.Lock()
			m.vulnerabilities = append(m.vulnerabilities, vuln)
			m.mu.Unlock()
		}
	}
}

// buildAccessControlMatrix creates a map of resources and required permissions.
func (m *WestModule) buildAccessControlMatrix() {
	// This is a simplified version - in practice, this would be more comprehensive
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, endpoint := range m.authEndpoints {
		if endpoint.RequiresAuth {
			req := AccessRequirement{
				Resource: endpoint.URL,
				Methods:  []string{endpoint.Method},
			}

			// Infer roles based on path patterns
			if strings.Contains(endpoint.URL, "admin") {
				req.Roles = []string{"admin", "administrator"}
			} else if strings.Contains(endpoint.URL, "api") {
				req.Roles = []string{"api_user", "developer"}
			} else {
				req.Roles = []string{"authenticated_user"}
			}

			m.authzMatrix[endpoint.URL] = req
		}
	}
}

// getResults aggregates all findings.
func (m *WestModule) getResults() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Calculate statistics
	stats := map[string]interface{}{
		"endpoints_discovered":  len(m.authEndpoints),
		"auth_endpoints":        countAuthEndpoints(m.authEndpoints),
		"session_patterns":      len(m.sessionPatterns),
		"vulnerabilities_found": len(m.vulnerabilities),
		"critical_vulns":        countBySeverity(m.vulnerabilities, "Critical"),
		"high_vulns":            countBySeverity(m.vulnerabilities, "High"),
		"medium_vulns":          countBySeverity(m.vulnerabilities, "Medium"),
		"low_vulns":             countBySeverity(m.vulnerabilities, "Low"),
	}

	return map[string]interface{}{
		"auth_endpoints":  m.authEndpoints,
		"session_info":    m.sessionPatterns,
		"access_control":  m.authzMatrix,
		"vulnerabilities": m.vulnerabilities,
		"statistics":      stats,
	}
}

// generateRecommendations creates actionable security recommendations.
func (m *WestModule) generateRecommendations() map[string][]string {
	recommendations := map[string][]string{
		"immediate":  {},
		"short_term": {},
		"long_term":  {},
	}

	// Analyze vulnerabilities for recommendations
	criticalCount := countBySeverity(m.vulnerabilities, "Critical")
	highCount := countBySeverity(m.vulnerabilities, "High")

	if criticalCount > 0 {
		recommendations["immediate"] = append(recommendations["immediate"],
			"Fix critical authentication bypass vulnerabilities immediately",
			"Review and strengthen authentication mechanisms",
		)
	}

	if highCount > 0 {
		recommendations["short_term"] = append(recommendations["short_term"],
			"Implement proper security headers on all endpoints",
			"Enable MFA for sensitive operations",
		)
	}

	// Check for missing security features
	hasHTTPS := false
	for _, endpoint := range m.authEndpoints {
		if strings.HasPrefix(endpoint.URL, "https://") {
			hasHTTPS = true
			break
		}
	}

	if !hasHTTPS {
		recommendations["immediate"] = append(recommendations["immediate"],
			"Implement HTTPS for all authentication endpoints",
		)
	}

	// Long-term recommendations
	recommendations["long_term"] = append(recommendations["long_term"],
		"Implement comprehensive security monitoring",
		"Consider adopting OAuth 2.0/OIDC for standardized authentication",
		"Implement zero-trust architecture principles",
	)

	return recommendations
}

// CircuitBreaker methods.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case "open":
		// Check if timeout has passed
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = "half-open"
			cb.failures = 0
			return true
		}
		return false

	case "half-open":
		// Allow one request to test
		return true

	default: // closed
		return true
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "half-open" {
		cb.state = "closed"
	}
	cb.failures = 0
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

// Helper functions.
func countAuthEndpoints(endpoints []AuthEndpoint) int {
	count := 0
	for _, ep := range endpoints {
		if ep.RequiresAuth {
			count++
		}
	}
	return count
}

func countBySeverity(vulns []AuthVulnerability, severity string) int {
	count := 0
	for _, v := range vulns {
		if v.Severity == severity {
			count++
		}
	}
	return count
}

// Info returns module information.
func (m *WestModule) Info() *modules.ModuleInfo {
	return &modules.ModuleInfo{
		Name:        m.ModuleName,
		Description: m.ModuleDescription,
		Author:      "Strigoi Security",
		Version:     "1.0.0",
		Tags:        []string{"authentication", "authorization", "access-control", "security"},
		References: []string{
			"OWASP Authentication Cheat Sheet",
			"OWASP Session Management Cheat Sheet",
			"CWE-287: Improper Authentication",
			"CWE-285: Improper Authorization",
		},
	}
}

// JSON export methods.
func (m *WestModule) ExportJSON() ([]byte, error) {
	results := m.getResults()
	return json.MarshalIndent(results, "", "  ")
}
