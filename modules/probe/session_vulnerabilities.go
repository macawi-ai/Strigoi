package probe

import (
	"crypto/sha256"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

// Common session vulnerability checkers

// AuthSessionChecker checks for authentication-related session vulnerabilities.
type AuthSessionChecker struct{}

// NewAuthSessionChecker creates a new authentication session checker.
func NewAuthSessionChecker() *AuthSessionChecker {
	return &AuthSessionChecker{}
}

// CheckSession analyzes a session for authentication vulnerabilities.
func (c *AuthSessionChecker) CheckSession(session *Session) []SessionVulnerability {
	var vulns []SessionVulnerability

	// Track authentication state
	var (
		hasAuth             bool
		authToken           string
		authTime            time.Time
		sessionIDBeforeAuth string
		sessionIDAfterAuth  string
		authFrameIndex      int
	)

	// Track all session IDs seen
	sessionIDs := make(map[string][]int) // sessionID -> frame indices

	for i, frame := range session.Frames {
		// Extract session ID from frame
		if sid := extractSessionIDFromFrame(frame); sid != "" {
			sessionIDs[sid] = append(sessionIDs[sid], i)
		}

		// Check for authentication request
		if isAuthRequest(frame) && !hasAuth {
			sessionIDBeforeAuth = extractSessionIDFromFrame(frame)
		}

		// Check for authentication success
		if isAuthSuccess(frame) {
			hasAuth = true
			authTime = time.Now() // In real implementation, get from frame timestamp
			authFrameIndex = i
			authToken = extractAuthToken(frame)
			sessionIDAfterAuth = extractSessionIDFromFrame(frame)

			// Check for session fixation
			if sessionIDBeforeAuth != "" && sessionIDBeforeAuth == sessionIDAfterAuth {
				vulns = append(vulns, SessionVulnerability{
					Type:        "session_fixation",
					Severity:    "high",
					Confidence:  0.9,
					Evidence:    fmt.Sprintf("Session ID '%s' unchanged after authentication", maskSessionID(sessionIDBeforeAuth)),
					FrameIDs:    []string{fmt.Sprintf("%d", i)},
					Timestamp:   authTime,
					Description: "Session ID should be regenerated after successful authentication to prevent fixation attacks",
				})
			}
		}

		// Check for token reuse in different contexts
		if authToken != "" && i > authFrameIndex {
			if isNewSessionContext(frame, session.Frames[authFrameIndex]) && containsToken(frame, authToken) {
				vulns = append(vulns, SessionVulnerability{
					Type:        "token_reuse",
					Severity:    "critical",
					Confidence:  0.95,
					Evidence:    "Authentication token reused in different session context",
					FrameIDs:    []string{fmt.Sprintf("%d", authFrameIndex), fmt.Sprintf("%d", i)},
					Timestamp:   time.Now(),
					Description: "Auth tokens should not be reused across different session contexts",
				})
			}
		}

		// Check for session hijacking indicators
		if hasAuth && i > authFrameIndex {
			if indicators := getHijackingIndicators(frame, session.Frames[authFrameIndex]); len(indicators) > 0 {
				vulns = append(vulns, SessionVulnerability{
					Type:        "session_hijacking_risk",
					Severity:    "high",
					Confidence:  0.7,
					Evidence:    fmt.Sprintf("Potential hijacking indicators: %v", indicators),
					FrameIDs:    []string{fmt.Sprintf("%d", i)},
					Timestamp:   time.Now(),
					Description: "Detected anomalies that may indicate session hijacking attempt",
				})
			}
		}
	}

	// Check for session ID reuse across different authentication states
	for sid, indices := range sessionIDs {
		if len(indices) > 10 { // Same session ID used in many frames
			vulns = append(vulns, SessionVulnerability{
				Type:        "excessive_session_reuse",
				Severity:    "medium",
				Confidence:  0.8,
				Evidence:    fmt.Sprintf("Session ID '%s' reused in %d frames", maskSessionID(sid), len(indices)),
				FrameIDs:    formatFrameIndices(indices[:min(5, len(indices))]), // First 5 occurrences
				Timestamp:   time.Now(),
				Description: "Excessive reuse of session ID may indicate poor session management",
			})
		}
	}

	return vulns
}

// TokenLeakageChecker checks for credential leakage across frames.
type TokenLeakageChecker struct{}

// NewTokenLeakageChecker creates a new token leakage checker.
func NewTokenLeakageChecker() *TokenLeakageChecker {
	return &TokenLeakageChecker{}
}

// CheckSession analyzes a session for token leakage.
func (c *TokenLeakageChecker) CheckSession(session *Session) []SessionVulnerability {
	var vulns []SessionVulnerability

	// Track all tokens/credentials seen
	tokenLocations := make(map[string][]tokenLocation) // token -> locations

	for i, frame := range session.Frames {
		tokens := extractAllTokens(frame)
		for _, token := range tokens {
			loc := tokenLocation{
				frameIndex: i,
				tokenType:  token.Type,
				location:   token.Location,
			}
			tokenLocations[token.Value] = append(tokenLocations[token.Value], loc)
		}
	}

	// Check for tokens appearing in multiple locations
	for _, locations := range tokenLocations {
		if len(locations) > 1 {
			// Check if token appears in unsafe locations
			unsafeCount := 0
			var unsafeLocations []string
			for _, loc := range locations {
				if isUnsafeLocation(loc.location) {
					unsafeCount++
					unsafeLocations = append(unsafeLocations,
						fmt.Sprintf("frame %d: %s", loc.frameIndex, loc.location))
				}
			}

			if unsafeCount > 0 {
				severity := "medium"
				if unsafeCount > 1 || containsSensitiveTokenType(locations) {
					severity = "high"
				}

				vulns = append(vulns, SessionVulnerability{
					Type:       "token_leakage",
					Severity:   severity,
					Confidence: 0.85,
					Evidence: fmt.Sprintf("Token exposed in %d locations including: %v",
						len(locations), unsafeLocations),
					FrameIDs:    getFrameIndicesFromLocations(locations),
					Timestamp:   time.Now(),
					Description: "Sensitive tokens should not be exposed in multiple locations or unsafe contexts",
				})
			}
		}
	}

	return vulns
}

// SessionTimeoutChecker checks for session timeout issues.
type SessionTimeoutChecker struct{}

// NewSessionTimeoutChecker creates a new session timeout checker.
func NewSessionTimeoutChecker() *SessionTimeoutChecker {
	return &SessionTimeoutChecker{}
}

// CheckSession analyzes session timing.
func (c *SessionTimeoutChecker) CheckSession(session *Session) []SessionVulnerability {
	var vulns []SessionVulnerability

	// Calculate session duration
	duration := session.LastActive.Sub(session.StartTime)

	// Check for excessively long sessions
	if duration > 24*time.Hour {
		vulns = append(vulns, SessionVulnerability{
			Type:        "excessive_session_duration",
			Severity:    "medium",
			Confidence:  0.9,
			Evidence:    fmt.Sprintf("Session active for %v", duration),
			FrameIDs:    []string{"0", fmt.Sprintf("%d", len(session.Frames)-1)},
			Timestamp:   session.StartTime,
			Description: "Long-lived sessions increase the window for attacks",
		})
	}

	// Check for session timeout configuration in frames
	for i, frame := range session.Frames {
		if timeout := extractSessionTimeout(frame); timeout > 0 {
			if timeout > 12*time.Hour {
				vulns = append(vulns, SessionVulnerability{
					Type:        "long_session_timeout",
					Severity:    "medium",
					Confidence:  0.95,
					Evidence:    fmt.Sprintf("Session timeout set to %v", timeout),
					FrameIDs:    []string{fmt.Sprintf("%d", i)},
					Timestamp:   time.Now(),
					Description: "Excessively long session timeout increases security risk",
				})
			} else if timeout < 5*time.Minute {
				vulns = append(vulns, SessionVulnerability{
					Type:        "short_session_timeout",
					Severity:    "low",
					Confidence:  0.95,
					Evidence:    fmt.Sprintf("Session timeout set to %v", timeout),
					FrameIDs:    []string{fmt.Sprintf("%d", i)},
					Timestamp:   time.Now(),
					Description: "Very short session timeout may impact user experience",
				})
			}
		}
	}

	return vulns
}

// WeakSessionChecker checks for weak session management.
type WeakSessionChecker struct{}

// NewWeakSessionChecker creates a new weak session checker.
func NewWeakSessionChecker() *WeakSessionChecker {
	return &WeakSessionChecker{}
}

// CheckSession analyzes session strength.
func (c *WeakSessionChecker) CheckSession(session *Session) []SessionVulnerability {
	var vulns []SessionVulnerability

	// Collect all session IDs
	var sessionIDs []string
	for _, frame := range session.Frames {
		if sid := extractSessionIDFromFrame(frame); sid != "" {
			sessionIDs = append(sessionIDs, sid)
		}
	}

	// Check for weak session IDs
	for i, sid := range sessionIDs {
		if weakness := analyzeSessionIDStrength(sid); weakness != "" {
			vulns = append(vulns, SessionVulnerability{
				Type:        "weak_session_id",
				Severity:    "high",
				Confidence:  0.8,
				Evidence:    fmt.Sprintf("Session ID weakness: %s", weakness),
				FrameIDs:    []string{fmt.Sprintf("%d", i)},
				Timestamp:   time.Now(),
				Description: "Weak session IDs are vulnerable to prediction or brute force attacks",
			})
			break // Report once per session
		}
	}

	// Check for missing secure flags in HTTP sessions
	if session.Protocol == "HTTP" {
		for i, frame := range session.Frames {
			if issues := checkHTTPSessionSecurity(frame); len(issues) > 0 {
				vulns = append(vulns, SessionVulnerability{
					Type:        "insecure_session_cookie",
					Severity:    "high",
					Confidence:  0.95,
					Evidence:    fmt.Sprintf("Missing security flags: %v", issues),
					FrameIDs:    []string{fmt.Sprintf("%d", i)},
					Timestamp:   time.Now(),
					Description: "Session cookies should use Secure and HttpOnly flags",
				})
			}
		}
	}

	return vulns
}

// CrossSessionChecker checks for data leakage between sessions.
type CrossSessionChecker struct {
	allSessions map[string]*Session // Track all sessions for comparison
}

// NewCrossSessionChecker creates a new cross-session checker.
func NewCrossSessionChecker() *CrossSessionChecker {
	return &CrossSessionChecker{
		allSessions: make(map[string]*Session),
	}
}

// CheckSession analyzes for cross-session issues.
func (c *CrossSessionChecker) CheckSession(session *Session) []SessionVulnerability {
	var vulns []SessionVulnerability

	// Store this session for future comparisons
	c.allSessions[session.ID] = session

	// Extract sensitive data from this session
	// This will be used in future for more sophisticated cross-session analysis
	// sensitiveData := extractSensitiveData(session)

	// Check if sensitive data from other sessions appears here
	for otherID, otherSession := range c.allSessions {
		if otherID == session.ID {
			continue
		}

		otherSensitiveData := extractSensitiveData(otherSession)

		// Check for data leakage
		for _, data := range otherSensitiveData {
			if containsSensitiveData(session, data) {
				vulns = append(vulns, SessionVulnerability{
					Type:       "cross_session_data_leak",
					Severity:   "critical",
					Confidence: 0.9,
					Evidence: fmt.Sprintf("Data from session %s found in session %s",
						maskSessionID(otherID), maskSessionID(session.ID)),
					FrameIDs:    []string{"multiple"},
					Timestamp:   time.Now(),
					Description: "Sensitive data from one session should not appear in another",
				})
			}
		}
	}

	return vulns
}

// Helper types.
type tokenLocation struct {
	frameIndex int
	tokenType  string
	location   string
}

type tokenInfo struct {
	Value    string
	Type     string
	Location string
}

type sensitiveDataItem struct {
	Type  string
	Value string
	Hash  string
}

// Helper functions

func extractSessionIDFromFrame(frame *Frame) string {
	// Try various methods to extract session ID
	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		// Check cookies
		if cookie := headers["cookie"]; cookie != "" {
			if sid := extractSessionIDFromCookie(cookie); sid != "" {
				return sid
			}
		}
		// Check set-cookie header (for responses)
		if setCookie := headers["set-cookie"]; setCookie != "" {
			if sid := extractSessionIDFromCookie(setCookie); sid != "" {
				return sid
			}
		}
		// Check custom headers
		for _, hdr := range []string{"x-session-id", "session-id"} {
			if val := headers[hdr]; val != "" {
				return val
			}
		}
	}

	// Check URL parameters
	if url, ok := frame.Fields["url"].(string); ok {
		if sid := extractSessionIDFromURL(url); sid != "" {
			return sid
		}
	}

	return ""
}

func extractSessionIDFromCookie(cookie string) string {
	sessionNames := []string{"JSESSIONID", "PHPSESSID", "session_id", "sid", "sessionid"}
	cookies := parseCookies(cookie)

	for _, name := range sessionNames {
		if val, ok := cookies[name]; ok && val != "" {
			return val
		}
	}
	return ""
}

func extractSessionIDFromURL(url string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`[?&]sid=([^&]+)`),
		regexp.MustCompile(`[?&]session_id=([^&]+)`),
		regexp.MustCompile(`[?&]sessionid=([^&]+)`),
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(url); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func isAuthRequest(frame *Frame) bool {
	// Check for authentication indicators
	if url, ok := frame.Fields["url"].(string); ok {
		authPaths := []string{"/login", "/auth", "/signin", "/authenticate"}
		for _, path := range authPaths {
			if strings.Contains(url, path) {
				return true
			}
		}
	}

	// Check for auth credentials in payload
	if payload, ok := frame.Fields["payload"].([]byte); ok {
		authPatterns := []string{"username", "password", "email", "login"}
		payloadStr := string(payload)
		for _, pattern := range authPatterns {
			if strings.Contains(strings.ToLower(payloadStr), pattern) {
				return true
			}
		}
	}

	return false
}

func isAuthSuccess(frame *Frame) bool {
	// Check HTTP status codes
	if status, ok := frame.Fields["status"].(int); ok {
		if status == 200 || status == 302 { // Success or redirect
			// Check for auth tokens in response
			if headers, ok := frame.Fields["headers"].(map[string]string); ok {
				if headers["set-cookie"] != "" || headers["authorization"] != "" {
					return true
				}
			}
		}
	}

	// Check for auth tokens in response body
	if payload, ok := frame.Fields["payload"].([]byte); ok {
		tokenPatterns := []string{"token", "session", "jwt", "access_token"}
		payloadStr := string(payload)
		for _, pattern := range tokenPatterns {
			if strings.Contains(strings.ToLower(payloadStr), pattern) {
				return true
			}
		}
	}

	return false
}

func extractAuthToken(frame *Frame) string {
	// Extract from headers
	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		if auth := headers["authorization"]; auth != "" {
			return auth
		}
		if cookie := headers["set-cookie"]; cookie != "" {
			// Extract session cookie
			cookies := parseCookies(cookie)
			for name, value := range cookies {
				if strings.Contains(strings.ToLower(name), "session") ||
					strings.Contains(strings.ToLower(name), "token") {
					return value
				}
			}
		}
	}

	// Extract from payload
	if payload, ok := frame.Fields["payload"].([]byte); ok {
		// Simple token extraction - in real implementation would be more robust
		tokenPattern := regexp.MustCompile(`"token"\s*:\s*"([^"]+)"`)
		if matches := tokenPattern.FindSubmatch(payload); len(matches) > 1 {
			return string(matches[1])
		}
	}

	return ""
}

func isNewSessionContext(frame1, frame2 *Frame) bool {
	// Compare IPs if available
	if ip1, ok1 := frame1.Fields["source_ip"].(string); ok1 {
		if ip2, ok2 := frame2.Fields["source_ip"].(string); ok2 {
			if ip1 != ip2 {
				return true
			}
		}
	}

	// Compare user agents
	if headers1, ok1 := frame1.Fields["headers"].(map[string]string); ok1 {
		if headers2, ok2 := frame2.Fields["headers"].(map[string]string); ok2 {
			if headers1["user-agent"] != headers2["user-agent"] {
				return true
			}
		}
	}

	return false
}

func containsToken(frame *Frame, token string) bool {
	// Check headers
	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		for _, value := range headers {
			if strings.Contains(value, token) {
				return true
			}
		}
	}

	// Check payload
	if payload, ok := frame.Fields["payload"].([]byte); ok {
		if strings.Contains(string(payload), token) {
			return true
		}
	}

	return false
}

func getHijackingIndicators(frame1, frame2 *Frame) []string {
	var indicators []string

	// Check for IP change
	if ip1, ok1 := frame1.Fields["source_ip"].(string); ok1 {
		if ip2, ok2 := frame2.Fields["source_ip"].(string); ok2 {
			if ip1 != ip2 {
				indicators = append(indicators, fmt.Sprintf("IP changed from %s to %s", ip1, ip2))
			}
		}
	}

	// Check for user agent change
	if headers1, ok1 := frame1.Fields["headers"].(map[string]string); ok1 {
		if headers2, ok2 := frame2.Fields["headers"].(map[string]string); ok2 {
			if ua1, ua2 := headers1["user-agent"], headers2["user-agent"]; ua1 != ua2 {
				indicators = append(indicators, "User-Agent changed")
			}
		}
	}

	// Check for geolocation change (if available)
	// Check for unusual timing patterns
	// etc.

	return indicators
}

func maskSessionID(sid string) string {
	if len(sid) <= 8 {
		return "****"
	}
	return sid[:4] + "****" + sid[len(sid)-4:]
}

func formatFrameIndices(indices []int) []string {
	result := make([]string, len(indices))
	for i, idx := range indices {
		result[i] = fmt.Sprintf("%d", idx)
	}
	return result
}

func extractAllTokens(frame *Frame) []tokenInfo {
	var tokens []tokenInfo

	// Extract from headers
	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		for name, value := range headers {
			if isTokenHeader(name) {
				// Check if it's a Bearer token with JWT
				if strings.HasPrefix(value, "Bearer ") {
					tokenValue := strings.TrimPrefix(value, "Bearer ")
					if isJWT(tokenValue) {
						tokens = append(tokens, tokenInfo{
							Value:    tokenValue,
							Type:     "jwt_token",
							Location: fmt.Sprintf("header:%s", name),
						})
					} else {
						tokens = append(tokens, tokenInfo{
							Value:    tokenValue,
							Type:     "bearer_token",
							Location: fmt.Sprintf("header:%s", name),
						})
					}
				} else {
					tokens = append(tokens, tokenInfo{
						Value:    value,
						Type:     classifyTokenType(name, value),
						Location: fmt.Sprintf("header:%s", name),
					})
				}
			}
		}
	}

	// Extract from payload using patterns
	if payload, ok := frame.Fields["payload"].([]byte); ok {
		payloadTokens := extractTokensFromPayload(payload)
		tokens = append(tokens, payloadTokens...)
	}

	// Extract from URL
	if url, ok := frame.Fields["url"].(string); ok {
		urlTokens := extractTokensFromURL(url)
		tokens = append(tokens, urlTokens...)
	}

	return tokens
}

func isTokenHeader(name string) bool {
	tokenHeaders := []string{"authorization", "x-auth-token", "x-api-key", "x-session-id"}
	nameLower := strings.ToLower(name)
	for _, th := range tokenHeaders {
		if nameLower == th {
			return true
		}
	}
	return false
}

func isJWT(token string) bool {
	// JWT has 3 parts separated by dots
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	// Basic JWT pattern check
	jwtPattern := regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)
	return jwtPattern.MatchString(token)
}

func classifyTokenType(name, value string) string {
	nameLower := strings.ToLower(name)

	if strings.Contains(nameLower, "api") {
		return "api_key"
	}
	if strings.Contains(nameLower, "session") {
		return "session_token"
	}
	if strings.Contains(value, "Bearer") {
		return "bearer_token"
	}
	if strings.HasPrefix(value, "Basic") {
		return "basic_auth"
	}

	return "unknown_token"
}

func extractTokensFromPayload(payload []byte) []tokenInfo {
	var tokens []tokenInfo

	// Common token patterns in JSON/XML/etc
	patterns := []struct {
		pattern *regexp.Regexp
		Type    string
	}{
		{regexp.MustCompile(`"token"\s*:\s*"([^"]+)"`), "json_token"},
		{regexp.MustCompile(`"api_key"\s*:\s*"([^"]+)"`), "api_key"},
		{regexp.MustCompile(`"session_id"\s*:\s*"([^"]+)"`), "session_token"},
		{regexp.MustCompile(`<token>([^<]+)</token>`), "xml_token"},
		// Also look for tokens in error messages
		{regexp.MustCompile(`(?i)api\s*key:\s*([A-Za-z0-9_\-]{20,})`), "api_key"},
		{regexp.MustCompile(`(?i)token:\s*([A-Za-z0-9_\-]{20,})`), "token"},
	}

	for _, p := range patterns {
		matches := p.pattern.FindAllSubmatch(payload, -1)
		for _, match := range matches {
			if len(match) > 1 {
				tokens = append(tokens, tokenInfo{
					Value:    string(match[1]),
					Type:     p.Type,
					Location: "payload",
				})
			}
		}
	}

	return tokens
}

func extractTokensFromURL(url string) []tokenInfo {
	var tokens []tokenInfo

	// Common URL parameter patterns
	patterns := []struct {
		pattern *regexp.Regexp
		Type    string
		Param   string
	}{
		{regexp.MustCompile(`[?&]api_key=([^&]+)`), "api_key", "api_key"},
		{regexp.MustCompile(`[?&]token=([^&]+)`), "token", "token"},
		{regexp.MustCompile(`[?&]access_token=([^&]+)`), "access_token", "access_token"},
		{regexp.MustCompile(`[?&]auth=([^&]+)`), "auth_token", "auth"},
	}

	for _, p := range patterns {
		if matches := p.pattern.FindStringSubmatch(url); len(matches) > 1 {
			tokens = append(tokens, tokenInfo{
				Value:    matches[1],
				Type:     p.Type,
				Location: fmt.Sprintf("url_param:%s", p.Param),
			})
		}
	}

	return tokens
}

func isUnsafeLocation(location string) bool {
	unsafeLocations := []string{"url", "query_param", "log", "error_message", "url_param"}
	for _, ul := range unsafeLocations {
		if strings.Contains(location, ul) {
			return true
		}
	}
	// Payload containing error messages is also unsafe
	if location == "payload" {
		return true
	}
	return false
}

func containsSensitiveTokenType(locations []tokenLocation) bool {
	for _, loc := range locations {
		if loc.tokenType == "api_key" || loc.tokenType == "bearer_token" || loc.tokenType == "jwt_token" {
			return true
		}
	}
	return false
}

func getFrameIndicesFromLocations(locations []tokenLocation) []string {
	var indices []string
	seen := make(map[int]bool)

	for _, loc := range locations {
		if !seen[loc.frameIndex] {
			seen[loc.frameIndex] = true
			indices = append(indices, fmt.Sprintf("%d", loc.frameIndex))
		}
	}

	return indices
}

func extractSessionTimeout(frame *Frame) time.Duration {
	// Check for timeout in headers
	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		// Check Set-Cookie for Max-Age
		if cookie := headers["set-cookie"]; cookie != "" {
			if maxAge := extractMaxAgeFromCookie(cookie); maxAge > 0 {
				return maxAge
			}
		}

		// Check custom timeout headers
		if timeout := headers["x-session-timeout"]; timeout != "" {
			if duration, err := time.ParseDuration(timeout); err == nil {
				return duration
			}
		}
	}

	return 0
}

func extractMaxAgeFromCookie(cookie string) time.Duration {
	maxAgePattern := regexp.MustCompile(`Max-Age=(\d+)`)
	if matches := maxAgePattern.FindStringSubmatch(cookie); len(matches) > 1 {
		if seconds, err := time.ParseDuration(matches[1] + "s"); err == nil {
			return seconds
		}
	}
	return 0
}

func analyzeSessionIDStrength(sid string) string {
	// Check length
	if len(sid) < 16 {
		return "too short (< 16 characters)"
	}

	// Check for patterns
	if isSequential(sid) {
		return "sequential pattern detected"
	}

	if isPredictable(sid) {
		return "predictable pattern detected"
	}

	// Check entropy
	if entropy := calculateEntropy(sid); entropy < 3.0 {
		return fmt.Sprintf("low entropy (%.2f)", entropy)
	}

	return ""
}

func isSequential(s string) bool {
	// Check for sequential numbers
	sequentialPattern := regexp.MustCompile(`(\d{4,})`)
	if matches := sequentialPattern.FindStringSubmatch(s); len(matches) > 1 {
		nums := matches[1]
		sequentialCount := 0
		for i := 1; i < len(nums); i++ {
			diff := int(nums[i] - nums[i-1])
			// Check for sequential digits (including 9->0 wrap)
			if diff == 1 || (nums[i-1] == '9' && nums[i] == '0') {
				sequentialCount++
			} else if diff != 0 {
				// Reset count if not sequential and not same digit
				sequentialCount = 0
			}
			// If we have 4 or more sequential digits, it's a sequential pattern
			if sequentialCount >= 3 {
				return true
			}
		}
	}
	return false
}

func isPredictable(s string) bool {
	// Check for timestamp patterns - but not if it's part of a larger sequential pattern
	timestampPattern := regexp.MustCompile(`^\d{10,13}$`)
	if timestampPattern.MatchString(s) {
		return true
	}
	// Also check if the entire string looks like a timestamp embedded in text
	if regexp.MustCompile(`\d{10,13}`).MatchString(s) {
		// But make sure it's not sequential
		matches := regexp.MustCompile(`(\d{10,13})`).FindStringSubmatch(s)
		if len(matches) > 1 && !isSequentialNumbers(matches[1]) {
			return true
		}
	}
	return false
}

func isSequentialNumbers(nums string) bool {
	sequentialCount := 0
	for i := 1; i < len(nums); i++ {
		diff := int(nums[i] - nums[i-1])
		if diff == 1 || (nums[i-1] == '9' && nums[i] == '0') {
			sequentialCount++
		} else if diff != 0 {
			sequentialCount = 0
		}
		if sequentialCount >= 3 {
			return true
		}
	}
	return false
}

func calculateEntropy(s string) float64 {
	// Simple Shannon entropy calculation
	if len(s) == 0 {
		return 0
	}

	frequency := make(map[rune]int)
	for _, char := range s {
		frequency[char]++
	}

	var entropy float64
	length := float64(len(s))

	for _, count := range frequency {
		if count > 0 {
			probability := float64(count) / length
			entropy -= probability * math.Log2(probability)
		}
	}

	return entropy
}

func checkHTTPSessionSecurity(frame *Frame) []string {
	var issues []string

	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		if cookie := headers["set-cookie"]; cookie != "" {
			cookieLower := strings.ToLower(cookie)

			// Check for Secure flag
			if !strings.Contains(cookieLower, "secure") {
				issues = append(issues, "missing Secure flag")
			}

			// Check for HttpOnly flag
			if !strings.Contains(cookieLower, "httponly") {
				issues = append(issues, "missing HttpOnly flag")
			}

			// Check for SameSite
			if !strings.Contains(cookieLower, "samesite") {
				issues = append(issues, "missing SameSite attribute")
			}
		}
	}

	return issues
}

func extractSensitiveData(session *Session) []sensitiveDataItem {
	var items []sensitiveDataItem

	for _, frame := range session.Frames {
		// Extract from headers
		if headers, ok := frame.Fields["headers"].(map[string]string); ok {
			for name, value := range headers {
				if isSensitiveHeader(name) {
					items = append(items, sensitiveDataItem{
						Type:  "header",
						Value: value,
						Hash:  hashString(value),
					})
				}
			}
		}

		// Extract from payload patterns
		if payload, ok := frame.Fields["payload"].([]byte); ok {
			sensitivePatterns := extractSensitivePatterns(payload)
			items = append(items, sensitivePatterns...)
		}
	}

	return items
}

func isSensitiveHeader(name string) bool {
	sensitive := []string{"authorization", "x-api-key", "x-auth-token"}
	nameLower := strings.ToLower(name)
	for _, s := range sensitive {
		if nameLower == s {
			return true
		}
	}
	return false
}

func extractSensitivePatterns(payload []byte) []sensitiveDataItem {
	var items []sensitiveDataItem

	// Extract API keys, tokens, etc.
	patterns := []struct {
		pattern *regexp.Regexp
		Type    string
	}{
		{regexp.MustCompile(`[a-zA-Z0-9]{32,}`), "potential_api_key"},
		{regexp.MustCompile(`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`), "jwt_token"},
	}

	for _, p := range patterns {
		matches := p.pattern.FindAll(payload, -1)
		for _, match := range matches {
			items = append(items, sensitiveDataItem{
				Type:  p.Type,
				Value: string(match),
				Hash:  hashString(string(match)),
			})
		}
	}

	return items
}

func containsSensitiveData(session *Session, data sensitiveDataItem) bool {
	for _, frame := range session.Frames {
		// Check in frame data
		if raw := frame.Raw; raw != nil {
			if strings.Contains(string(raw), data.Value) {
				return true
			}
		}
	}
	return false
}

func hashString(s string) string {
	// Simple hash for comparison
	hash := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", hash[:8])
}
