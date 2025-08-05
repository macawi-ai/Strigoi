package probe

import (
	"regexp"
)

// AIProvider represents an AI service provider's characteristics.
type AIProvider struct {
	Name           string
	Patterns       []*regexp.Regexp
	Ports          []int
	Headers        []string
	ResponseFields []string
	SecurityChecks []SecurityCheck
}

// SecurityCheck represents a security validation.
type SecurityCheck struct {
	Name        string
	Severity    string
	CheckFunc   func(response *EndpointResponse) bool
	Description string
}

// EndpointResponse represents the full response from an endpoint.
type EndpointResponse struct {
	URL        string
	Method     string
	StatusCode int
	Headers    map[string][]string
	Body       []byte
	Error      error
	Provider   string
	ModelInfo  string
	Security   []SecurityFinding
}

// SecurityFinding represents a security issue.
type SecurityFinding struct {
	Severity    string
	Name        string
	Description string
	Evidence    string
}

// GetAIProviders returns configured AI service providers.
func GetAIProviders() map[string]*AIProvider {
	providers := make(map[string]*AIProvider)

	// OpenAI and compatible APIs
	providers["openai"] = &AIProvider{
		Name: "OpenAI",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`/v\d+/completions`),
			regexp.MustCompile(`/v\d+/chat/completions`),
			regexp.MustCompile(`/v\d+/embeddings`),
			regexp.MustCompile(`/v\d+/models`),
			regexp.MustCompile(`/v\d+/files`),
			regexp.MustCompile(`/v\d+/fine-tunes`),
		},
		Headers: []string{
			"Authorization",
			"OpenAI-Organization",
			"OpenAI-Version",
		},
		ResponseFields: []string{
			"choices",
			"usage",
			"model",
			"created",
			"object",
		},
		SecurityChecks: []SecurityCheck{
			{
				Name:     "Missing Authentication",
				Severity: "critical",
				CheckFunc: func(resp *EndpointResponse) bool {
					auth := resp.Headers["Authorization"]
					apiKey := resp.Headers["X-Api-Key"]
					return len(auth) == 0 && len(apiKey) == 0 && resp.StatusCode == 200
				},
				Description: "API endpoint accessible without authentication",
			},
		},
	}

	// Anthropic Claude API
	providers["anthropic"] = &AIProvider{
		Name: "Anthropic",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`/v\d+/complete`),
			regexp.MustCompile(`/v\d+/messages`),
			regexp.MustCompile(`/v\d+/organizations`),
		},
		Headers: []string{
			"X-API-Key",
			"anthropic-version",
			"anthropic-beta",
		},
		ResponseFields: []string{
			"completion",
			"stop_reason",
			"model",
			"id",
			"type",
			"role",
			"content",
		},
		SecurityChecks: []SecurityCheck{
			{
				Name:     "API Key in URL",
				Severity: "high",
				CheckFunc: func(resp *EndpointResponse) bool {
					// Check if API key appears in URL parameters
					return regexp.MustCompile(`[?&]api[_-]?key=`).MatchString(resp.URL)
				},
				Description: "API key exposed in URL parameters",
			},
		},
	}

	// Ollama local server
	providers["ollama"] = &AIProvider{
		Name:  "Ollama",
		Ports: []int{11434},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`/api/generate`),
			regexp.MustCompile(`/api/chat`),
			regexp.MustCompile(`/api/tags`),
			regexp.MustCompile(`/api/pull`),
			regexp.MustCompile(`/api/push`),
			regexp.MustCompile(`/api/embeddings`),
		},
		ResponseFields: []string{
			"model",
			"response",
			"context",
			"total_duration",
			"load_duration",
			"eval_duration",
		},
		SecurityChecks: []SecurityCheck{
			{
				Name:     "Unauthenticated Local Access",
				Severity: "high",
				CheckFunc: func(resp *EndpointResponse) bool {
					// Only check for local servers (localhost/127.0.0.1)
					isLocal := regexp.MustCompile(`https?://(localhost|127\.0\.0\.1|::1)`).MatchString(resp.URL)
					if !isLocal {
						return false
					}
					// Ollama by default has no auth
					return resp.StatusCode == 200 && len(resp.Headers["Authorization"]) == 0
				},
				Description: "Local model server exposed without authentication",
			},
			{
				Name:     "Public Network Exposure",
				Severity: "critical",
				CheckFunc: func(resp *EndpointResponse) bool {
					// Only check local servers that are accessible from non-localhost IPs
					// This would require the target to be a non-localhost IP but serving Ollama
					isNonLocalIP := !regexp.MustCompile(`https?://(localhost|127\.0\.0\.1|::1)`).MatchString(resp.URL)
					// Check if it's actually an Ollama server by looking for Ollama-specific endpoints
					isOllamaEndpoint := regexp.MustCompile(`/api/(generate|chat|tags|pull|push|embeddings)`).MatchString(resp.URL)
					return isNonLocalIP && isOllamaEndpoint && resp.StatusCode == 200
				},
				Description: "Local model server accessible from external network",
			},
		},
	}

	// Google Vertex AI / PaLM API
	providers["google"] = &AIProvider{
		Name: "Google AI",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`/v\d+/projects/.*/locations/.*/models/.*:generateContent`),
			regexp.MustCompile(`/v\d+/projects/.*/locations/.*/publishers/.*/models/.*`),
			regexp.MustCompile(`/v\d+/models/.*:generate`),
		},
		Headers: []string{
			"Authorization",
			"x-goog-api-key",
			"x-goog-user-project",
		},
		ResponseFields: []string{
			"candidates",
			"promptFeedback",
			"safetyRatings",
			"citationMetadata",
		},
	}

	// Cohere API
	providers["cohere"] = &AIProvider{
		Name: "Cohere",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`/v\d+/generate`),
			regexp.MustCompile(`/v\d+/embed`),
			regexp.MustCompile(`/v\d+/classify`),
			regexp.MustCompile(`/v\d+/tokenize`),
			regexp.MustCompile(`/v\d+/detokenize`),
		},
		Headers: []string{
			"Authorization",
			"Cohere-Version",
			"X-Client-Name",
		},
		ResponseFields: []string{
			"generations",
			"prompt",
			"likelihood",
			"embeddings",
		},
	}

	// Hugging Face Inference API
	providers["huggingface"] = &AIProvider{
		Name: "Hugging Face",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`/models/.*/predict`),
			regexp.MustCompile(`/models/.*/generate`),
			regexp.MustCompile(`/pipeline/feature-extraction/.*`),
		},
		Headers: []string{
			"Authorization",
			"X-Use-Cache",
			"X-Wait-For-Model",
		},
		ResponseFields: []string{
			"generated_text",
			"score",
			"label",
		},
	}

	return providers
}

// GetAIEndpoints returns a list of common AI endpoints to probe.
func GetAIEndpoints() []string {
	return []string{
		// OpenAI style
		"/v1/completions",
		"/v1/chat/completions",
		"/v1/embeddings",
		"/v1/models",

		// Anthropic
		"/v1/complete",
		"/v1/messages",

		// Google
		"/v1/models",
		"/v1beta/models",

		// Generic
		"/api/generate",
		"/api/chat",
		"/api/v1/generate",
		"/generate",
		"/chat",
		"/complete",
		"/completion",
		"/inference",
		"/predict",

		// Model info
		"/models",
		"/api/models",
		"/api/tags",

		// Health/Status
		"/health",
		"/api/health",
		"/status",

		// Documentation
		"/docs",
		"/api-docs",
		"/openapi.json",
		"/swagger.json",

		// Ethical AI Service Discovery
		"/.well-known/ai-service",
		"/.well-known/ai-plugin.json",
		"/api/info",
		"/api/about",
	}
}

// GetLocalModelPorts returns common ports for local model servers.
func GetLocalModelPorts() []int {
	return []int{
		11434, // Ollama default
		8080,  // llama.cpp default
		8000,  // vLLM, TGI default
		5000,  // LocalAI default
		3000,  // Some web UIs
		7860,  // Gradio default
		8501,  // TensorFlow Serving REST
	}
}
