package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LLMAccelerator provides LLM-based analysis acceleration
type LLMAccelerator interface {
	AnalyzeEvent(ctx context.Context, event *SecurityEvent, result *DetectionResult) (string, float64, error)
	AnalyzeBatch(ctx context.Context, events []*SecurityEvent, results []*DetectionResult) ([]LLMAnalysis, error)
	SupportsFineTuning() bool
	FineTune(ctx context.Context, events []*SecurityEvent, labels [][]string) error
	Close() error
}

// LLMAnalysis represents LLM analysis output
type LLMAnalysis struct {
	EventID     string
	Explanation string
	Confidence  float64
	Insights    []string
	Indicators  []string
}

// BaseLLMAccelerator provides common LLM functionality
type BaseLLMAccelerator struct {
	provider   string
	model      string
	apiKey     string
	endpoint   string
	httpClient *http.Client
	cache      *LLMCache
	mu         sync.RWMutex
}

// NewLLMAccelerator creates appropriate LLM accelerator
func NewLLMAccelerator(provider, model string) (LLMAccelerator, error) {
	switch provider {
	case "openai":
		return NewOpenAIAccelerator(model)
	case "anthropic":
		return NewAnthropicAccelerator(model)
	case "local":
		return NewLocalLLMAccelerator(model)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}
}

// OpenAIAccelerator implements OpenAI-based acceleration
type OpenAIAccelerator struct {
	BaseLLMAccelerator
	systemPrompt string
}

// NewOpenAIAccelerator creates OpenAI accelerator
func NewOpenAIAccelerator(model string) (*OpenAIAccelerator, error) {
	if model == "" {
		model = "gpt-4"
	}

	acc := &OpenAIAccelerator{
		BaseLLMAccelerator: BaseLLMAccelerator{
			provider:   "openai",
			model:      model,
			endpoint:   "https://api.openai.com/v1/chat/completions",
			httpClient: &http.Client{Timeout: 30 * time.Second},
			cache:      NewLLMCache(1000),
		},
		systemPrompt: `You are a security analyst expert system. Analyze the provided security event data and detection results. 
Provide a concise explanation of the threat, its potential impact, and key indicators. 
Focus on actionable insights and avoid speculation.`,
	}

	// Note: API key should be set via environment variable
	// acc.apiKey = os.Getenv("OPENAI_API_KEY")

	return acc, nil
}

// AnalyzeEvent analyzes a single event with OpenAI
func (o *OpenAIAccelerator) AnalyzeEvent(ctx context.Context, event *SecurityEvent, result *DetectionResult) (string, float64, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s_%f", event.ID, result.ThreatScore)
	if cached, ok := o.cache.Get(cacheKey); ok {
		return cached.Explanation, cached.Confidence, nil
	}

	// Prepare prompt
	prompt := o.formatEventPrompt(event, result)

	// Create request
	reqBody := map[string]interface{}{
		"model": o.model,
		"messages": []map[string]string{
			{"role": "system", "content": o.systemPrompt},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"max_tokens":  500,
	}

	// Make API call
	response, err := o.callAPI(ctx, reqBody)
	if err != nil {
		return "", 0, err
	}

	// Parse response
	explanation, confidence := o.parseResponse(response)

	// Cache result
	o.cache.Set(cacheKey, LLMAnalysis{
		EventID:     event.ID,
		Explanation: explanation,
		Confidence:  confidence,
	})

	return explanation, confidence, nil
}

// AnalyzeBatch analyzes multiple events
func (o *OpenAIAccelerator) AnalyzeBatch(ctx context.Context, events []*SecurityEvent, results []*DetectionResult) ([]LLMAnalysis, error) {
	analyses := make([]LLMAnalysis, len(events))

	// Process in batches to respect rate limits
	batchSize := 5
	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}

		// Process batch concurrently
		var wg sync.WaitGroup
		for j := i; j < end; j++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				explanation, confidence, err := o.AnalyzeEvent(ctx, events[idx], results[idx])
				if err == nil {
					analyses[idx] = LLMAnalysis{
						EventID:     events[idx].ID,
						Explanation: explanation,
						Confidence:  confidence,
					}
				}
			}(j)
		}

		wg.Wait()

		// Rate limiting
		if end < len(events) {
			time.Sleep(time.Second)
		}
	}

	return analyses, nil
}

// SupportsFineTuning indicates if fine-tuning is supported
func (o *OpenAIAccelerator) SupportsFineTuning() bool {
	return true
}

// FineTune performs model fine-tuning
func (o *OpenAIAccelerator) FineTune(ctx context.Context, events []*SecurityEvent, labels [][]string) error {
	// OpenAI fine-tuning would involve:
	// 1. Preparing training data in JSONL format
	// 2. Uploading to OpenAI
	// 3. Creating fine-tuning job
	// 4. Monitoring progress

	// This is a placeholder - actual implementation would be more complex
	return fmt.Errorf("fine-tuning not implemented in this demo")
}

// Close closes the accelerator
func (o *OpenAIAccelerator) Close() error {
	return nil
}

// formatEventPrompt formats event data for LLM
func (o *OpenAIAccelerator) formatEventPrompt(event *SecurityEvent, result *DetectionResult) string {
	var prompt strings.Builder

	prompt.WriteString("Analyze this security event:\n\n")

	// Event details
	prompt.WriteString(fmt.Sprintf("Event Type: %s\n", event.Type))
	prompt.WriteString(fmt.Sprintf("Protocol: %s\n", event.Protocol))
	prompt.WriteString(fmt.Sprintf("Source: %s\n", event.Source))
	prompt.WriteString(fmt.Sprintf("Destination: %s\n", event.Destination))
	prompt.WriteString(fmt.Sprintf("Timestamp: %s\n", event.Timestamp.Format(time.RFC3339)))

	// Detection results
	prompt.WriteString(fmt.Sprintf("\nThreat Score: %.2f\n", result.ThreatScore))
	prompt.WriteString(fmt.Sprintf("Anomalous: %v\n", result.Anomalous))

	if len(result.Classifications) > 0 {
		prompt.WriteString("\nClassifications:\n")
		for _, class := range result.Classifications {
			prompt.WriteString(fmt.Sprintf("- %s (%.2f probability)\n", class.Category, class.Probability))
		}
	}

	if len(result.Patterns) > 0 {
		prompt.WriteString("\nDetected Patterns:\n")
		for _, pattern := range result.Patterns {
			prompt.WriteString(fmt.Sprintf("- %s: %s (frequency: %d)\n", pattern.Type, pattern.ID, pattern.Frequency))
		}
	}

	// Payload snippet if available
	if len(event.Payload) > 0 {
		snippet := string(event.Payload)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		prompt.WriteString(fmt.Sprintf("\nPayload snippet: %s\n", snippet))
	}

	prompt.WriteString("\nProvide a security analysis including:\n")
	prompt.WriteString("1. Threat assessment\n")
	prompt.WriteString("2. Potential attack vector\n")
	prompt.WriteString("3. Recommended actions\n")
	prompt.WriteString("4. Key indicators to monitor\n")

	return prompt.String()
}

// callAPI makes the actual API call
func (o *OpenAIAccelerator) callAPI(ctx context.Context, reqBody map[string]interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// parseResponse extracts explanation and confidence
func (o *OpenAIAccelerator) parseResponse(response map[string]interface{}) (string, float64) {
	explanation := ""
	confidence := 0.0

	// Extract message content
	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					explanation = content
				}
			}

			// Extract confidence from finish_reason or other metadata
			if finishReason, ok := choice["finish_reason"].(string); ok {
				if finishReason == "stop" {
					confidence = 0.9
				} else {
					confidence = 0.7
				}
			}
		}
	}

	// Adjust confidence based on response quality
	if len(explanation) > 100 && strings.Contains(explanation, "Threat assessment") {
		confidence = math.Min(confidence+0.1, 1.0)
	}

	return explanation, confidence
}

// LocalLLMAccelerator implements local model acceleration
type LocalLLMAccelerator struct {
	BaseLLMAccelerator
	modelPath string
	// In production, this would interface with local model
}

// NewLocalLLMAccelerator creates local LLM accelerator
func NewLocalLLMAccelerator(model string) (*LocalLLMAccelerator, error) {
	return &LocalLLMAccelerator{
		BaseLLMAccelerator: BaseLLMAccelerator{
			provider: "local",
			model:    model,
			cache:    NewLLMCache(1000),
		},
		modelPath: fmt.Sprintf("/models/%s", model),
	}, nil
}

// AnalyzeEvent analyzes event with local model
func (l *LocalLLMAccelerator) AnalyzeEvent(ctx context.Context, event *SecurityEvent, result *DetectionResult) (string, float64, error) {
	// Simplified local analysis
	explanation := l.generateLocalExplanation(event, result)
	confidence := l.calculateLocalConfidence(result)

	return explanation, confidence, nil
}

// AnalyzeBatch analyzes batch with local model
func (l *LocalLLMAccelerator) AnalyzeBatch(ctx context.Context, events []*SecurityEvent, results []*DetectionResult) ([]LLMAnalysis, error) {
	analyses := make([]LLMAnalysis, len(events))

	for i := range events {
		explanation, confidence, err := l.AnalyzeEvent(ctx, events[i], results[i])
		if err == nil {
			analyses[i] = LLMAnalysis{
				EventID:     events[i].ID,
				Explanation: explanation,
				Confidence:  confidence,
			}
		}
	}

	return analyses, nil
}

// generateLocalExplanation creates rule-based explanation
func (l *LocalLLMAccelerator) generateLocalExplanation(event *SecurityEvent, result *DetectionResult) string {
	var explanation strings.Builder

	// Threat assessment
	threatLevel := "Low"
	if result.ThreatScore > 0.8 {
		threatLevel = "Critical"
	} else if result.ThreatScore > 0.6 {
		threatLevel = "High"
	} else if result.ThreatScore > 0.4 {
		threatLevel = "Medium"
	}

	explanation.WriteString(fmt.Sprintf("Threat Assessment: %s risk event detected.\n\n", threatLevel))

	// Classification-based analysis
	if len(result.Classifications) > 0 {
		topClass := result.Classifications[0]
		explanation.WriteString(fmt.Sprintf("Primary Classification: %s (%.0f%% confidence)\n",
			topClass.Category, topClass.Probability*100))

		// Category-specific insights
		switch topClass.Category {
		case "malware":
			explanation.WriteString("Potential malware activity detected. ")
			explanation.WriteString("Monitor for file modifications, network callbacks, and process injection.\n")
		case "intrusion":
			explanation.WriteString("Intrusion attempt identified. ")
			explanation.WriteString("Check for unauthorized access, privilege escalation, and lateral movement.\n")
		case "ddos":
			explanation.WriteString("DDoS pattern recognized. ")
			explanation.WriteString("Implement rate limiting and consider blocking source IPs.\n")
		case "scanning":
			explanation.WriteString("Network scanning detected. ")
			explanation.WriteString("May indicate reconnaissance phase of attack.\n")
		}
	}

	// Anomaly analysis
	if result.Anomalous {
		explanation.WriteString("\nAnomaly Detection: Unusual behavior compared to baseline.\n")
	}

	// Pattern analysis
	if len(result.Patterns) > 0 {
		explanation.WriteString("\nPattern Analysis:\n")
		for _, pattern := range result.Patterns {
			switch pattern.Type {
			case "temporal_burst":
				explanation.WriteString("- Burst activity detected, possible automated attack\n")
			case "periodic_pattern":
				explanation.WriteString("- Periodic behavior suggests bot or scheduled activity\n")
			case "coordinated_activity":
				explanation.WriteString("- Coordinated activity from multiple sources\n")
			}
		}
	}

	// Recommendations
	explanation.WriteString("\nRecommended Actions:\n")
	if result.ThreatScore > 0.7 {
		explanation.WriteString("1. Immediate investigation required\n")
		explanation.WriteString("2. Consider blocking source IP\n")
		explanation.WriteString("3. Check for related events\n")
		explanation.WriteString("4. Preserve evidence for analysis\n")
	} else if result.ThreatScore > 0.4 {
		explanation.WriteString("1. Monitor for escalation\n")
		explanation.WriteString("2. Review security policies\n")
		explanation.WriteString("3. Update detection rules\n")
	} else {
		explanation.WriteString("1. Log for future correlation\n")
		explanation.WriteString("2. Update baseline if legitimate\n")
	}

	return explanation.String()
}

// calculateLocalConfidence estimates confidence
func (l *LocalLLMAccelerator) calculateLocalConfidence(result *DetectionResult) float64 {
	confidence := 0.5

	// Adjust based on detection strength
	if result.ThreatScore > 0.8 {
		confidence += 0.3
	} else if result.ThreatScore > 0.6 {
		confidence += 0.2
	}

	// Adjust based on classification confidence
	if len(result.Classifications) > 0 && result.Classifications[0].Probability > 0.8 {
		confidence += 0.1
	}

	// Adjust based on pattern detection
	if len(result.Patterns) > 2 {
		confidence += 0.1
	}

	return math.Min(confidence, 0.95)
}

// SupportsFineTuning indicates if fine-tuning is supported
func (l *LocalLLMAccelerator) SupportsFineTuning() bool {
	return false
}

// FineTune is not supported for local models in this implementation
func (l *LocalLLMAccelerator) FineTune(ctx context.Context, events []*SecurityEvent, labels [][]string) error {
	return fmt.Errorf("fine-tuning not supported for local models")
}

// Close closes the accelerator
func (l *LocalLLMAccelerator) Close() error {
	return nil
}

// AnthropicAccelerator implements Anthropic Claude acceleration
type AnthropicAccelerator struct {
	BaseLLMAccelerator
}

// NewAnthropicAccelerator creates Anthropic accelerator
func NewAnthropicAccelerator(model string) (*AnthropicAccelerator, error) {
	if model == "" {
		model = "claude-2"
	}

	return &AnthropicAccelerator{
		BaseLLMAccelerator: BaseLLMAccelerator{
			provider:   "anthropic",
			model:      model,
			endpoint:   "https://api.anthropic.com/v1/complete",
			httpClient: &http.Client{Timeout: 30 * time.Second},
			cache:      NewLLMCache(1000),
		},
	}, nil
}

// Implement similar methods as OpenAI but with Anthropic API format
func (a *AnthropicAccelerator) AnalyzeEvent(ctx context.Context, event *SecurityEvent, result *DetectionResult) (string, float64, error) {
	// Implementation would follow Anthropic's API format
	return "", 0, fmt.Errorf("Anthropic integration not implemented in this demo")
}

func (a *AnthropicAccelerator) AnalyzeBatch(ctx context.Context, events []*SecurityEvent, results []*DetectionResult) ([]LLMAnalysis, error) {
	return nil, fmt.Errorf("Anthropic integration not implemented in this demo")
}

func (a *AnthropicAccelerator) SupportsFineTuning() bool {
	return false
}

func (a *AnthropicAccelerator) FineTune(ctx context.Context, events []*SecurityEvent, labels [][]string) error {
	return fmt.Errorf("fine-tuning not supported")
}

func (a *AnthropicAccelerator) Close() error {
	return nil
}

// LLMCache provides simple caching for LLM responses
type LLMCache struct {
	cache    map[string]LLMAnalysis
	capacity int
	mu       sync.RWMutex
}

// NewLLMCache creates a new LLM cache
func NewLLMCache(capacity int) *LLMCache {
	return &LLMCache{
		cache:    make(map[string]LLMAnalysis),
		capacity: capacity,
	}
}

// Get retrieves from cache
func (c *LLMCache) Get(key string) (LLMAnalysis, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	analysis, ok := c.cache[key]
	return analysis, ok
}

// Set adds to cache
func (c *LLMCache) Set(key string, analysis LLMAnalysis) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction if at capacity
	if len(c.cache) >= c.capacity {
		// Remove random entry
		for k := range c.cache {
			delete(c.cache, k)
			break
		}
	}

	c.cache[key] = analysis
}
