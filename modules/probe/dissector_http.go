package probe

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// HTTPDissector parses HTTP protocol data from streams.
type HTTPDissector struct {
	// Patterns for detecting HTTP traffic
	requestPattern  *regexp.Regexp
	responsePattern *regexp.Regexp

	// Sensitive data patterns
	authPattern    *regexp.Regexp
	apiKeyPattern  *regexp.Regexp
	tokenPattern   *regexp.Regexp
	cookiePattern  *regexp.Regexp
	sessionPattern *regexp.Regexp
}

// NewHTTPDissector creates a new HTTP protocol dissector.
func NewHTTPDissector() *HTTPDissector {
	return &HTTPDissector{
		// HTTP request line: METHOD /path HTTP/1.x
		requestPattern: regexp.MustCompile(`^(GET|POST|PUT|DELETE|HEAD|OPTIONS|PATCH|CONNECT|TRACE)\s+\S+\s+HTTP/\d\.\d`),
		// HTTP response line: HTTP/1.x STATUS MESSAGE
		responsePattern: regexp.MustCompile(`^HTTP/\d\.\d\s+\d{3}`),

		// Sensitive data patterns
		authPattern:    regexp.MustCompile(`(?i)(authorization|www-authenticate):\s*(.+)`),
		apiKeyPattern:  regexp.MustCompile(`(?i)(api[_-]?key|apikey|x-api-key)[:=]\s*([A-Za-z0-9_\-]{20,})`),
		tokenPattern:   regexp.MustCompile(`(?i)(token|bearer|jwt)[:=]\s*([A-Za-z0-9_\-\.]{20,})`),
		cookiePattern:  regexp.MustCompile(`(?i)(cookie|set-cookie):\s*(.+)`),
		sessionPattern: regexp.MustCompile(`(?i)(session[_-]?id|sess[_-]?id|phpsessid|jsessionid)[:=]\s*([A-Za-z0-9_\-]{16,})`),
	}
}

// Identify checks if the data contains HTTP protocol traffic.
func (d *HTTPDissector) Identify(data []byte) (bool, float64) {
	if len(data) < 10 {
		return false, 0
	}

	// Convert to string for pattern matching
	dataStr := string(data[:min(1024, len(data))]) // Check first 1KB
	lines := strings.Split(dataStr, "\n")

	// Debug logging disabled for production

	// Check for HTTP request
	if len(lines) > 0 && d.requestPattern.MatchString(lines[0]) {
		return true, 0.9
	}

	// Check for HTTP response
	if len(lines) > 0 && d.responsePattern.MatchString(lines[0]) {
		return true, 0.9
	}

	// Check for HTTP-like headers
	httpHeaderCount := 0
	for _, line := range lines {
		if strings.Contains(line, ": ") && !strings.HasPrefix(line, " ") {
			httpHeaderCount++
		}
	}

	// If we see multiple header-like lines, it might be HTTP
	if httpHeaderCount >= 3 {
		return true, 0.7
	}

	return false, 0
}

// Dissect parses HTTP data into a structured frame.
func (d *HTTPDissector) Dissect(data []byte) (*Frame, error) {
	frame := &Frame{
		Protocol: "HTTP",
		Fields:   make(map[string]interface{}),
		Raw:      data,
	}

	// Try to parse as HTTP request first
	if req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(data))); err == nil {
		frame.Fields["type"] = "request"
		frame.Fields["method"] = req.Method
		frame.Fields["url"] = req.URL.String()
		frame.Fields["host"] = req.Host
		frame.Fields["headers"] = d.headerMapToDict(req.Header)

		// Parse body if present
		if req.Body != nil {
			bodyData, _ := io.ReadAll(req.Body)
			req.Body.Close()
			frame.Fields["body"] = string(bodyData)
			frame.Fields["body_size"] = len(bodyData)
		}

		return frame, nil
	}

	// Try to parse as HTTP response
	if resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(data)), nil); err == nil {
		frame.Fields["type"] = "response"
		frame.Fields["status_code"] = resp.StatusCode
		frame.Fields["status"] = resp.Status
		frame.Fields["headers"] = d.headerMapToDict(resp.Header)

		// Parse body if present
		if resp.Body != nil {
			bodyData, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			frame.Fields["body"] = string(bodyData)
			frame.Fields["body_size"] = len(bodyData)
		}

		return frame, nil
	}

	// If standard parsing fails, try manual parsing for partial data
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])

		// Check if it's a request line
		if d.requestPattern.MatchString(firstLine) {
			parts := strings.Fields(firstLine)
			if len(parts) >= 3 {
				frame.Fields["type"] = "request"
				frame.Fields["method"] = parts[0]
				frame.Fields["url"] = parts[1]
				frame.Fields["version"] = parts[2]
			}
		} else if d.responsePattern.MatchString(firstLine) {
			parts := strings.Fields(firstLine)
			if len(parts) >= 2 {
				frame.Fields["type"] = "response"
				frame.Fields["version"] = parts[0]
				frame.Fields["status_code"] = parts[1]
				if len(parts) > 2 {
					frame.Fields["status_text"] = strings.Join(parts[2:], " ")
				}
			}
		}

		// Parse headers manually
		headers := make(map[string]string)
		for i := 1; i < len(lines); i++ {
			line := lines[i]
			if line == "" {
				// End of headers
				break
			}
			if idx := strings.Index(line, ":"); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				value := strings.TrimSpace(line[idx+1:])
				headers[key] = value
			}
		}
		if len(headers) > 0 {
			frame.Fields["headers"] = headers
		}
	}

	return frame, nil
}

// FindVulnerabilities analyzes an HTTP frame for security issues.
func (d *HTTPDissector) FindVulnerabilities(frame *Frame) []StreamVulnerability {
	var vulns []StreamVulnerability

	// Check URL for sensitive data
	if urlStr, ok := frame.Fields["url"].(string); ok {
		vulns = append(vulns, d.checkURLForSecrets(urlStr)...)
	}

	// Check headers
	if headers, ok := frame.Fields["headers"].(map[string]string); ok {
		vulns = append(vulns, d.checkHeadersForSecrets(headers)...)
	} else if headers, ok := frame.Fields["headers"].(map[string][]string); ok {
		// Convert to simple map for checking
		simpleHeaders := make(map[string]string)
		for k, v := range headers {
			if len(v) > 0 {
				simpleHeaders[k] = strings.Join(v, "; ")
			}
		}
		vulns = append(vulns, d.checkHeadersForSecrets(simpleHeaders)...)
	}

	// Check body
	if body, ok := frame.Fields["body"].(string); ok {
		vulns = append(vulns, d.checkBodyForSecrets(body)...)
	}

	// Add protocol context to all vulnerabilities
	for i := range vulns {
		if method, ok := frame.Fields["method"].(string); ok {
			vulns[i].Context = fmt.Sprintf("HTTP %s request", method)
		} else if status, ok := frame.Fields["status_code"].(int); ok {
			vulns[i].Context = fmt.Sprintf("HTTP %d response", status)
		}
	}

	return vulns
}

// checkURLForSecrets looks for sensitive data in URLs.
func (d *HTTPDissector) checkURLForSecrets(urlStr string) []StreamVulnerability {
	var vulns []StreamVulnerability

	// Parse URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return vulns
	}

	// Check query parameters
	params := u.Query()
	for key, values := range params {
		keyLower := strings.ToLower(key)

		// Check for API keys in query params
		if strings.Contains(keyLower, "api") && strings.Contains(keyLower, "key") ||
			strings.Contains(keyLower, "apikey") {
			for _, value := range values {
				if len(value) >= 20 {
					vulns = append(vulns, StreamVulnerability{
						Type:       "credential",
						Subtype:    "api_key_in_url",
						Severity:   "critical",
						Confidence: 0.9,
						Evidence:   fmt.Sprintf("%s=%s", key, maskValue(value)),
						Location:   "URL query parameter",
						Context:    fmt.Sprintf("API key exposed in URL query parameter '%s'", key),
					})
				}
			}
		}

		// Check for tokens
		if strings.Contains(keyLower, "token") || strings.Contains(keyLower, "auth") {
			for _, value := range values {
				if len(value) >= 20 {
					vulns = append(vulns, StreamVulnerability{
						Type:       "credential",
						Subtype:    "token_in_url",
						Severity:   "critical",
						Confidence: 0.9,
						Evidence:   fmt.Sprintf("%s=%s", key, maskValue(value)),
						Location:   "URL query parameter",
						Context:    fmt.Sprintf("Authentication token exposed in URL query parameter '%s'", key),
					})
				}
			}
		}

		// Check for passwords (should never be in URLs!)
		if strings.Contains(keyLower, "pass") || keyLower == "pwd" {
			for range values {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "password_in_url",
					Severity:   "critical",
					Confidence: 0.95,
					Evidence:   fmt.Sprintf("%s=***", key),
					Location:   "URL query parameter",
					Context:    "Password exposed in URL query parameter",
				})
			}
		}
	}

	// Check for credentials in URL path (e.g., /api/v1/users/admin:password@host)
	if u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			vulns = append(vulns, StreamVulnerability{
				Type:       "credential",
				Subtype:    "basic_auth_in_url",
				Severity:   "critical",
				Confidence: 1.0,
				Evidence:   fmt.Sprintf("%s:***@%s", u.User.Username(), u.Host),
				Location:   "URL",
				Context:    "Basic authentication credentials exposed in URL",
			})
		}
	}

	return vulns
}

// checkHeadersForSecrets looks for sensitive data in HTTP headers.
func (d *HTTPDissector) checkHeadersForSecrets(headers map[string]string) []StreamVulnerability {
	var vulns []StreamVulnerability

	for name, value := range headers {
		nameLower := strings.ToLower(name)

		// Authorization header
		if nameLower == "authorization" {
			authType := ""
			if strings.HasPrefix(value, "Bearer ") {
				authType = "bearer_token"
			} else if strings.HasPrefix(value, "Basic ") {
				authType = "basic_auth"
			} else if strings.HasPrefix(value, "API-Key ") {
				authType = "api_key"
			} else {
				authType = "auth_token"
			}

			vulns = append(vulns, StreamVulnerability{
				Type:       "credential",
				Subtype:    authType,
				Severity:   "high",
				Confidence: 1.0,
				Evidence:   fmt.Sprintf("%s: %s", name, maskAuthValue(value)),
				Location:   "HTTP header",
				Context:    fmt.Sprintf("Authentication credentials in %s header", name),
			})
		}

		// API Key headers
		if strings.Contains(nameLower, "api") && strings.Contains(nameLower, "key") ||
			nameLower == "x-api-key" || nameLower == "apikey" {
			vulns = append(vulns, StreamVulnerability{
				Type:       "credential",
				Subtype:    "api_key",
				Severity:   "high",
				Confidence: 0.95,
				Evidence:   fmt.Sprintf("%s: %s", name, maskValue(value)),
				Location:   "HTTP header",
				Context:    fmt.Sprintf("API key in %s header", name),
			})
		}

		// Session cookies
		if nameLower == "cookie" || nameLower == "set-cookie" {
			// Parse cookie string
			cookies := parseCookies(value)
			for cookieName, cookieValue := range cookies {
				cookieNameLower := strings.ToLower(cookieName)
				if strings.Contains(cookieNameLower, "session") ||
					strings.Contains(cookieNameLower, "sess") ||
					cookieNameLower == "phpsessid" ||
					cookieNameLower == "jsessionid" {
					vulns = append(vulns, StreamVulnerability{
						Type:       "credential",
						Subtype:    "session_cookie",
						Severity:   "high",
						Confidence: 0.9,
						Evidence:   fmt.Sprintf("%s=%s", cookieName, maskValue(cookieValue)),
						Location:   "HTTP cookie",
						Context:    fmt.Sprintf("Session identifier in cookie '%s'", cookieName),
					})
				}
			}
		}

		// Custom authentication headers
		if strings.Contains(nameLower, "token") || strings.Contains(nameLower, "auth") {
			if len(value) >= 20 && !strings.Contains(nameLower, "content") {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "custom_auth_header",
					Severity:   "medium",
					Confidence: 0.8,
					Evidence:   fmt.Sprintf("%s: %s", name, maskValue(value)),
					Location:   "HTTP header",
					Context:    fmt.Sprintf("Potential authentication data in %s header", name),
				})
			}
		}
	}

	return vulns
}

// checkBodyForSecrets looks for sensitive data in HTTP body.
func (d *HTTPDissector) checkBodyForSecrets(body string) []StreamVulnerability {
	var vulns []StreamVulnerability

	// Check for API keys in body
	if matches := d.apiKeyPattern.FindAllStringSubmatch(body, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 3 {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "api_key_in_body",
					Severity:   "high",
					Confidence: 0.85,
					Evidence:   fmt.Sprintf("%s: %s", match[1], maskValue(match[2])),
					Location:   "HTTP body",
					Context:    "API key found in HTTP body",
				})
			}
		}
	}

	// Check for tokens in body
	if matches := d.tokenPattern.FindAllStringSubmatch(body, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 3 {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "token_in_body",
					Severity:   "high",
					Confidence: 0.85,
					Evidence:   fmt.Sprintf("%s: %s", match[1], maskValue(match[2])),
					Location:   "HTTP body",
					Context:    "Authentication token found in HTTP body",
				})
			}
		}
	}

	// Check for session IDs in body
	if matches := d.sessionPattern.FindAllStringSubmatch(body, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 3 {
				vulns = append(vulns, StreamVulnerability{
					Type:       "credential",
					Subtype:    "session_in_body",
					Severity:   "medium",
					Confidence: 0.8,
					Evidence:   fmt.Sprintf("%s: %s", match[1], maskValue(match[2])),
					Location:   "HTTP body",
					Context:    "Session identifier found in HTTP body",
				})
			}
		}
	}

	// Check for passwords in JSON/form data
	passwordPatterns := []string{
		`"password"\s*:\s*"([^"]+)"`,
		`'password'\s*:\s*'([^']+)'`,
		`password=([^&\s]+)`,
		`"pass"\s*:\s*"([^"]+)"`,
		`"pwd"\s*:\s*"([^"]+)"`,
	}

	for _, pattern := range passwordPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindAllStringSubmatch(body, -1); matches != nil {
			for _, match := range matches {
				if len(match) >= 2 {
					vulns = append(vulns, StreamVulnerability{
						Type:       "credential",
						Subtype:    "password_in_body",
						Severity:   "critical",
						Confidence: 0.9,
						Evidence:   "password: ***",
						Location:   "HTTP body",
						Context:    "Password found in HTTP body",
					})
				}
			}
		}
	}

	return vulns
}

// Helper functions

func (d *HTTPDissector) headerMapToDict(headers http.Header) map[string]string {
	dict := make(map[string]string)
	for name, values := range headers {
		dict[name] = strings.Join(values, "; ")
	}
	return dict
}

func maskValue(value string) string {
	if len(value) <= 8 {
		return "***"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

func maskAuthValue(value string) string {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) == 2 {
		return parts[0] + " " + maskValue(parts[1])
	}
	return maskValue(value)
}

func parseCookies(cookieStr string) map[string]string {
	cookies := make(map[string]string)
	parts := strings.Split(cookieStr, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, "="); idx > 0 {
			name := strings.TrimSpace(part[:idx])
			value := strings.TrimSpace(part[idx+1:])
			cookies[name] = value
		}
	}
	return cookies
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetSessionID extracts session identifier from HTTP frame.
func (d *HTTPDissector) GetSessionID(frame *Frame) (string, error) {
	headers, ok := frame.Fields["headers"].(map[string]string)
	if !ok {
		return "", fmt.Errorf("no headers in frame")
	}

	// Check cookies first
	if cookie := headers["cookie"]; cookie != "" {
		cookies := parseCookies(cookie)
		// Common session cookie names
		sessionNames := []string{"JSESSIONID", "PHPSESSID", "session_id", "sid",
			"sessionid", "connect.sid", "ASP.NET_SessionId", "_session_id",
			"ci_session", "laravel_session"}

		for _, name := range sessionNames {
			if val, ok := cookies[name]; ok && val != "" {
				return fmt.Sprintf("http_cookie_%s", val), nil
			}
		}

		// Check for any cookie with "session" in the name
		for name, val := range cookies {
			if strings.Contains(strings.ToLower(name), "session") && val != "" {
				return fmt.Sprintf("http_cookie_%s", val), nil
			}
		}
	}

	// Check for session in URL parameters
	if urlStr, ok := frame.Fields["url"].(string); ok {
		if idx := strings.Index(urlStr, "?"); idx >= 0 {
			params := urlStr[idx+1:]
			values, _ := url.ParseQuery(params)

			sessionParams := []string{"session_id", "sid", "sessionid", "s"}
			for _, param := range sessionParams {
				if val := values.Get(param); val != "" {
					return fmt.Sprintf("http_url_%s", val), nil
				}
			}
		}
	}

	// Use authorization header as fallback for API sessions
	if auth := headers["authorization"]; auth != "" {
		// Create a stable session ID from auth header
		hash := sha256.Sum256([]byte(auth))
		return fmt.Sprintf("http_auth_%x", hash[:8]), nil
	}

	// Check custom headers that might contain session info
	sessionHeaders := []string{"x-session-id", "x-auth-token", "x-request-id"}
	for _, hdr := range sessionHeaders {
		if val := headers[hdr]; val != "" {
			return fmt.Sprintf("http_header_%s", val), nil
		}
	}

	return "", fmt.Errorf("no session identifier found")
}
