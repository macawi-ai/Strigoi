package probe

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/pkg/modules"
)

func init() {
	// Register the module factory
	modules.RegisterBuiltin("probe/north", NewNorthModule)
}

// NorthModule implements AI/LLM infrastructure discovery.
type NorthModule struct {
	modules.BaseModule
	target       string
	timeout      int
	userAgent    string
	aiPreset     string
	includeLocal bool
	delay        int // milliseconds between requests
	discovered   []EndpointResponse
	providers    map[string]*AIProvider
}

// NewNorthModule creates a new probe north module instance.
func NewNorthModule() modules.Module {
	m := &NorthModule{
		BaseModule: modules.BaseModule{
			ModuleName:        "probe/north",
			ModuleDescription: "AI/LLM infrastructure discovery and enumeration",
			ModuleType:        modules.ProbeModule,
			ModuleOptions: map[string]*modules.ModuleOption{
				"target": {
					Name:        "target",
					Description: "Target URL or hostname",
					Required:    true,
					Type:        "string",
					Default:     "",
				},
				"timeout": {
					Name:        "timeout",
					Description: "Request timeout in seconds",
					Required:    false,
					Type:        "int",
					Default:     "10",
				},
				"user-agent": {
					Name:        "user-agent",
					Description: "User-Agent header to use",
					Required:    false,
					Type:        "string",
					Default:     "Strigoi/0.5.0",
				},
				"ai-preset": {
					Name:        "ai-preset",
					Description: "AI endpoint preset (basic, comprehensive, local)",
					Required:    false,
					Type:        "string",
					Default:     "basic",
				},
				"include-local": {
					Name:        "include-local",
					Description: "Include local model server ports",
					Required:    false,
					Type:        "bool",
					Default:     "false",
				},
				"delay": {
					Name:        "delay",
					Description: "Delay between requests in milliseconds",
					Required:    false,
					Type:        "int",
					Default:     "100",
				},
			},
		},
		timeout:   10,
		userAgent: "Strigoi/0.5.0",
		aiPreset:  "basic",
		delay:     100,
		providers: GetAIProviders(),
	}
	return m
}

// SetOption sets a module option.
func (m *NorthModule) SetOption(name, value string) error {
	switch name {
	case "target":
		m.target = value
	case "timeout":
		timeout, err := modules.ParseInt(value)
		if err != nil {
			return fmt.Errorf("invalid timeout value: %v", err)
		}
		m.timeout = timeout
	case "user-agent":
		m.userAgent = value
	case "ai-preset":
		if value != "basic" && value != "comprehensive" && value != "local" {
			return fmt.Errorf("invalid ai-preset: %s (must be basic, comprehensive, or local)", value)
		}
		m.aiPreset = value
	case "include-local":
		include, err := modules.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid include-local value: %v", err)
		}
		m.includeLocal = include
	case "delay":
		delay, err := modules.ParseInt(value)
		if err != nil {
			return fmt.Errorf("invalid delay value: %v", err)
		}
		m.delay = delay
	default:
		return m.BaseModule.SetOption(name, value)
	}
	return nil
}

// ValidateOptions validates all required options are set.
func (m *NorthModule) ValidateOptions() error {
	if m.target == "" {
		return fmt.Errorf("target is required")
	}

	// Ensure target has a scheme
	if !strings.HasPrefix(m.target, "http://") && !strings.HasPrefix(m.target, "https://") {
		m.target = "https://" + m.target
	}

	// Validate URL
	_, err := url.Parse(m.target)
	if err != nil {
		return fmt.Errorf("invalid target URL: %v", err)
	}

	return nil
}

// Run executes the module.
func (m *NorthModule) Run() (*modules.ModuleResult, error) {
	if err := m.ValidateOptions(); err != nil {
		return nil, err
	}

	result := &modules.ModuleResult{
		Module:    m.Name(),
		StartTime: time.Now(),
		Status:    "running",
		Data:      make(map[string]interface{}),
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(m.timeout) * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			// Don't follow redirects automatically
			return http.ErrUseLastResponse
		},
	}

	// Try ethical discovery first
	if provider, confidence := m.ethicalDiscoveryProbe(client); provider != "" {
		// Store the discovered provider with high confidence
		result.Data["ethical_discovery"] = map[string]interface{}{
			"provider":   provider,
			"confidence": confidence,
			"method":     "ethical_probe",
		}
	}

	// Get AI-focused endpoints based on preset
	endpoints := m.getEndpointsByPreset()

	m.discovered = []EndpointResponse{}

	// Probe main target with AI endpoints
	for _, endpoint := range endpoints {
		targetURL := strings.TrimRight(m.target, "/") + endpoint

		// Try GET and POST methods (most AI endpoints use these)
		methods := []string{"GET", "POST"}

		for _, method := range methods {
			resp, err := m.probeAIEndpoint(client, targetURL, method)
			if err != nil {
				continue
			}

			// Only save meaningful responses
			if resp.StatusCode < 500 && resp.StatusCode != 0 {
				m.discovered = append(m.discovered, *resp)
			}

			// Add delay between requests
			time.Sleep(time.Duration(m.delay) * time.Millisecond)
		}
	}

	// Check local ports if requested
	m.probeLocalPorts(client)

	// Analyze findings
	aiServicesFound := m.analyzeFindings()

	// Convert EndpointResponse to EndpointInfo for compatibility
	endpointInfos := make([]modules.EndpointInfo, 0, len(m.discovered))
	for _, resp := range m.discovered {
		contentType := ""
		if ct := resp.Headers["Content-Type"]; len(ct) > 0 {
			contentType = ct[0]
		}

		info := modules.EndpointInfo{
			Path:        resp.URL,
			Method:      resp.Method,
			StatusCode:  resp.StatusCode,
			ContentType: contentType,
			Headers:     make(map[string]string),
			Timestamp:   time.Now(),
		}

		// Add important headers
		for k, v := range resp.Headers {
			if len(v) > 0 {
				info.Headers[k] = v[0]
			}
		}

		// Add AI provider info as a header for now
		if resp.Provider != "" {
			info.Headers["X-AI-Provider"] = resp.Provider
		}

		// Add security findings as header
		if len(resp.Security) > 0 {
			info.Headers["X-Security-Risk"] = resp.Security[0].Name
		}

		endpointInfos = append(endpointInfos, info)
	}

	result.Status = "completed"
	result.EndTime = time.Now()
	result.Data["discovered"] = endpointInfos
	result.Data["total_endpoints"] = len(endpointInfos)
	result.Data["ai_services"] = aiServicesFound
	result.Data["target"] = m.target

	return result, nil
}

// probeEndpoint probes a single endpoint.
func (m *NorthModule) probeEndpoint(client *http.Client, targetURL, method string) (modules.EndpointInfo, error) {
	req, err := http.NewRequest(method, targetURL, nil)
	if err != nil {
		return modules.EndpointInfo{}, err
	}

	req.Header.Set("User-Agent", m.userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return modules.EndpointInfo{}, err
	}
	defer resp.Body.Close()

	info := modules.EndpointInfo{
		Path:        targetURL,
		Method:      method,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Headers:     make(map[string]string),
		Timestamp:   time.Now(),
	}

	// Collect interesting headers
	interestingHeaders := []string{
		"Server",
		"X-Powered-By",
		"X-Framework",
		"X-Runtime",
		"X-Version",
		"X-API-Version",
		"Allow",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
	}

	for _, header := range interestingHeaders {
		if value := resp.Header.Get(header); value != "" {
			info.Headers[header] = value
		}
	}

	return info, nil
}

// getEndpointsByPreset returns endpoints based on the selected preset.
func (m *NorthModule) getEndpointsByPreset() []string {
	switch m.aiPreset {
	case "comprehensive":
		// Return all AI endpoints plus common variations
		endpoints := GetAIEndpoints()
		// Add more comprehensive patterns
		additions := []string{
			"/v1beta/completions",
			"/v2/completions",
			"/api/v1/completions",
			"/api/v2/completions",
			"/inference/v1/completions",
			"/ml/predict",
			"/api/inference",
			"/.well-known/ai-plugin.json",
			"/api/capabilities",
		}
		return append(endpoints, additions...)

	case "local":
		// Focus on local model server endpoints
		return []string{
			"/api/generate",
			"/api/chat",
			"/api/tags",
			"/api/embeddings",
			"/v1/completions",
			"/v1/chat/completions",
			"/completion",
			"/generate",
			"/health",
			"/models",
			"/api/models",
		}

	default: // "basic"
		return GetAIEndpoints()
	}
}

// probeAIEndpoint probes an endpoint and analyzes for AI characteristics.
func (m *NorthModule) probeAIEndpoint(client *http.Client, targetURL, method string) (*EndpointResponse, error) {
	req, err := http.NewRequest(method, targetURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", m.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return &EndpointResponse{
			URL:    targetURL,
			Method: method,
			Error:  err,
		}, nil
	}
	defer resp.Body.Close()

	// Read response body for analysis (limited to 1MB to prevent DoS)
	bodyReader := io.LimitReader(resp.Body, 1024*1024)
	body, _ := io.ReadAll(bodyReader)

	response := &EndpointResponse{
		URL:        targetURL,
		Method:     method,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
		Security:   []SecurityFinding{},
	}

	// Analyze response for AI provider fingerprints
	m.analyzeAIResponse(response)

	return response, nil
}

// analyzeAIResponse analyzes response to identify AI provider and security issues.
func (m *NorthModule) analyzeAIResponse(resp *EndpointResponse) {
	// Skip error responses
	if resp.StatusCode >= 500 || resp.Error != nil {
		return
	}

	// Check each provider's patterns
	for providerName, provider := range m.providers {
		// Check URL patterns
		for _, pattern := range provider.Patterns {
			if pattern.MatchString(resp.URL) {
				resp.Provider = providerName
				break
			}
		}

		// Check response fields if we have a body
		if len(resp.Body) > 0 && resp.StatusCode == 200 {
			bodyStr := string(resp.Body)
			matchCount := 0

			for _, field := range provider.ResponseFields {
				if strings.Contains(bodyStr, field) {
					matchCount++
				}
			}

			// If we match multiple fields, likely this provider
			if matchCount >= 2 {
				resp.Provider = providerName
			}
		}

		// Run security checks
		for _, check := range provider.SecurityChecks {
			if check.CheckFunc(resp) {
				resp.Security = append(resp.Security, SecurityFinding{
					Severity:    check.Severity,
					Name:        check.Name,
					Description: check.Description,
					Evidence:    resp.URL,
				})
			}
		}
	}
}

// probeLocalPorts checks common local model server ports.
func (m *NorthModule) probeLocalPorts(client *http.Client) {
	if !m.includeLocal {
		return
	}

	// Parse base URL to get host
	u, err := url.Parse(m.target)
	if err != nil {
		return
	}

	host := u.Hostname()
	if host == "" {
		host = m.target
	}

	// Check common local model ports
	for _, port := range GetLocalModelPorts() {
		baseURL := fmt.Sprintf("http://%s:%d", host, port)

		// Try a few common endpoints on each port
		testEndpoints := []string{"/health", "/api/tags", "/models", "/v1/models", "/"}

		for _, endpoint := range testEndpoints {
			targetURL := baseURL + endpoint

			resp, err := m.probeAIEndpoint(client, targetURL, "GET")
			if err == nil && resp.StatusCode > 0 && resp.StatusCode < 500 {
				m.discovered = append(m.discovered, *resp)

				// Add delay between requests
				time.Sleep(time.Duration(m.delay) * time.Millisecond)
			}
		}
	}
}

// analyzeFindings processes discovered endpoints to identify AI services.
func (m *NorthModule) analyzeFindings() map[string]interface{} {
	aiServices := make(map[string]interface{})

	// Group by provider
	providerCount := make(map[string]int)
	securityFindings := []SecurityFinding{}

	for _, resp := range m.discovered {
		if resp.Provider != "" {
			providerCount[resp.Provider]++
		}

		// Collect all security findings
		securityFindings = append(securityFindings, resp.Security...)

		// Try to extract model info from successful responses
		if resp.StatusCode == 200 && len(resp.Body) > 0 {
			// Try to parse as JSON and look for model info
			var jsonResp map[string]interface{}
			if err := json.Unmarshal(resp.Body, &jsonResp); err == nil {
				// Look for model field
				if model, ok := jsonResp["model"].(string); ok {
					resp.ModelInfo = model
				} else if models, ok := jsonResp["models"].([]interface{}); ok && len(models) > 0 {
					// For model listing endpoints
					resp.ModelInfo = fmt.Sprintf("%d models available", len(models))
				}
			}
		}
	}

	aiServices["providers_detected"] = providerCount
	aiServices["total_providers"] = len(providerCount)
	aiServices["security_findings"] = securityFindings
	aiServices["total_security_findings"] = len(securityFindings)

	return aiServices
}

// Check validates the module can run.
func (m *NorthModule) Check() bool {
	return true
}

// Info returns module information.
func (m *NorthModule) Info() *modules.ModuleInfo {
	return &modules.ModuleInfo{
		Name:        m.Name(),
		Description: m.Description(),
		Author:      "Macawi",
		Version:     "0.5.0",
		Type:        m.Type(),
		Options:     m.Options(),
		References: []string{
			"https://github.com/macawi-ai/strigoi",
		},
	}
}

// GetDiscoveredEndpoints returns the discovered endpoints.
func (m *NorthModule) GetDiscoveredEndpoints() []modules.EndpointInfo {
	// Convert EndpointResponse to EndpointInfo
	endpoints := make([]modules.EndpointInfo, 0, len(m.discovered))
	for _, resp := range m.discovered {
		contentType := ""
		if ct := resp.Headers["Content-Type"]; len(ct) > 0 {
			contentType = ct[0]
		}

		info := modules.EndpointInfo{
			Path:        resp.URL,
			Method:      resp.Method,
			StatusCode:  resp.StatusCode,
			ContentType: contentType,
			Headers:     make(map[string]string),
			Timestamp:   time.Now(),
		}

		// Add important headers
		for k, v := range resp.Headers {
			if len(v) > 0 {
				info.Headers[k] = v[0]
			}
		}

		endpoints = append(endpoints, info)
	}
	return endpoints
}

// ToJSON converts discovered endpoints to JSON.
func (m *NorthModule) ToJSON() (string, error) {
	data, err := json.MarshalIndent(m.discovered, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ethicalDiscoveryProbe attempts to identify the AI provider using polite, ethical methods.
func (m *NorthModule) ethicalDiscoveryProbe(client *http.Client) (string, float64) {
	// Common AI API endpoints that require authentication
	probeEndpoints := []struct {
		path     string
		method   string
		provider string
	}{
		{"/v1/models", "GET", "openai"},
		{"/v1/messages", "POST", "anthropic"},
		{"/v1/chat/completions", "POST", "openai"},
		{"/v1/complete", "POST", "anthropic"},
		{"/generate", "POST", "cohere"},
		{"/api/tags", "GET", "ollama"},
	}

	for _, probe := range probeEndpoints {
		targetURL := strings.TrimRight(m.target, "/") + probe.path

		req, err := http.NewRequest(probe.method, targetURL, nil)
		if err != nil {
			continue
		}

		// Set ethical discovery headers
		req.Header.Set("User-Agent", m.userAgent+" (Ethical Discovery Probe)")
		req.Header.Set("Authorization", "Bearer ETHICAL_DISCOVERY_PROBE")
		req.Header.Set("X-API-Key", "ETHICAL_DISCOVERY_PROBE")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		// Read the error response
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))

		// Analyze error response for provider fingerprints
		if provider, confidence := m.analyzeErrorResponse(resp.StatusCode, resp.Header, body); provider != "" {
			return provider, confidence
		}
	}

	return "", 0.0
}

// analyzeErrorResponse identifies providers from error responses.
func (m *NorthModule) analyzeErrorResponse(statusCode int, headers http.Header, body []byte) (string, float64) {
	bodyStr := string(body)

	// OpenAI detection
	if strings.Contains(bodyStr, "platform.openai.com") ||
		strings.Contains(bodyStr, "invalid_api_key") && strings.Contains(bodyStr, "Incorrect API key provided") {
		return "openai", 0.9
	}

	// Anthropic detection
	if strings.Contains(bodyStr, "anthropic") ||
		strings.Contains(bodyStr, "claude") ||
		headers.Get("X-Anthropic-Request-ID") != "" ||
		(strings.Contains(bodyStr, "authentication_error") && strings.Contains(bodyStr, "Invalid bearer token")) {
		return "anthropic", 0.9
	}

	// Google/Vertex AI detection
	if strings.Contains(bodyStr, "googleapis.com") ||
		strings.Contains(bodyStr, "Google Cloud") ||
		headers.Get("X-Goog-Api-Client") != "" {
		return "google", 0.9
	}

	// Cohere detection
	if strings.Contains(bodyStr, "cohere") ||
		headers.Get("X-Cohere-Request-ID") != "" {
		return "cohere", 0.9
	}

	// Ollama detection (usually no auth, so 200 response)
	if statusCode == 200 && strings.Contains(bodyStr, "models") &&
		(strings.Contains(bodyStr, "llama") || strings.Contains(bodyStr, "mistral")) {
		return "ollama", 0.8
	}

	// Hugging Face detection
	if strings.Contains(bodyStr, "huggingface") ||
		strings.Contains(bodyStr, "Model") && strings.Contains(bodyStr, "is currently loading") {
		return "huggingface", 0.85
	}

	// Generic API key error (lower confidence)
	if statusCode == 401 || statusCode == 403 {
		if strings.Contains(bodyStr, "API key") || strings.Contains(bodyStr, "api key") {
			return "unknown_ai_service", 0.3
		}
	}

	return "", 0.0
}
