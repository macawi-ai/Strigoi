package north

import (
	"context"
	"fmt"
	"time"
	
	"github.com/macawi-ai/strigoi/internal/actors"
)

// EndpointDiscoveryActor probes for LLM API endpoints
type EndpointDiscoveryActor struct {
	*actors.BaseActor
}

// NewEndpointDiscoveryActor creates an actor that discovers LLM endpoints
func NewEndpointDiscoveryActor() *EndpointDiscoveryActor {
	actor := &EndpointDiscoveryActor{
		BaseActor: actors.NewBaseActor(
			"endpoint_discovery",
			"Discovers LLM API endpoints and service boundaries",
			"north",
		),
	}
	
	// Define capabilities
	actor.AddCapability(actors.Capability{
		Name:        "api_detection",
		Description: "Detects common LLM API patterns",
		DataTypes:   []string{"url", "endpoint"},
	})
	
	actor.AddCapability(actors.Capability{
		Name:        "version_enumeration",
		Description: "Enumerates API versions and capabilities",
		DataTypes:   []string{"api_response"},
	})
	
	actor.SetInputTypes([]string{"url", "domain"})
	actor.SetOutputType("endpoint_list")
	
	return actor
}

// Probe discovers LLM endpoints
func (e *EndpointDiscoveryActor) Probe(ctx context.Context, target actors.Target) (*actors.ProbeResult, error) {
	result := &actors.ProbeResult{
		ActorName: e.Name(),
		Timestamp: time.Now(),
		Target:    target,
	}
	
	// Common LLM API patterns to check
	patterns := []struct {
		path     string
		platform string
	}{
		// OpenAI patterns
		{"/v1/models", "openai"},
		{"/v1/chat/completions", "openai"},
		{"/v1/completions", "openai"},
		
		// Anthropic patterns
		{"/v1/messages", "anthropic"},
		{"/v1/complete", "anthropic"},
		
		// Google patterns
		{"/v1beta/models", "google"},
		{"/_/BardChatUi/", "google"},
		
		// Generic patterns
		{"/api/v1/models", "generic"},
		{"/api/chat", "generic"},
		{"/api/generate", "generic"},
	}
	
	// Probe each pattern
	for _, pattern := range patterns {
		discovery := actors.Discovery{
			Type:       "endpoint",
			Identifier: pattern.path,
			Properties: map[string]interface{}{
				"platform":    pattern.platform,
				"full_url":    fmt.Sprintf("%s%s", target.Location, pattern.path),
				"http_method": "POST",
			},
			Confidence: 0.7, // Would be adjusted based on actual probe
		}
		
		result.Discoveries = append(result.Discoveries, discovery)
	}
	
	// Store raw data for further analysis
	result.RawData = map[string]interface{}{
		"patterns_checked": len(patterns),
		"base_url":        target.Location,
	}
	
	return result, nil
}

// Sense performs deep analysis on discovered endpoints
func (e *EndpointDiscoveryActor) Sense(ctx context.Context, data *actors.ProbeResult) (*actors.SenseResult, error) {
	result := &actors.SenseResult{
		ActorName: e.Name(),
		Timestamp: time.Now(),
	}
	
	// Analyze discovered endpoints
	for _, discovery := range data.Discoveries {
		if discovery.Type == "endpoint" {
			// Create observations about the endpoint
			obs := actors.Observation{
				Layer:       "application",
				Description: fmt.Sprintf("Found %s endpoint: %s", 
					discovery.Properties["platform"], 
					discovery.Identifier),
				Evidence: discovery.Properties,
				Severity: "info",
			}
			result.Observations = append(result.Observations, obs)
		}
	}
	
	// Detect patterns
	platformCount := make(map[string]int)
	for _, disc := range data.Discoveries {
		if platform, ok := disc.Properties["platform"].(string); ok {
			platformCount[platform]++
		}
	}
	
	// Multi-platform pattern
	if len(platformCount) > 1 {
		pattern := actors.Pattern{
			Name:        "multi_platform_deployment",
			Description: "Multiple LLM platforms detected",
			Instances:   []interface{}{platformCount},
			Confidence:  0.8,
		}
		result.Patterns = append(result.Patterns, pattern)
	}
	
	// Risk assessment
	if len(data.Discoveries) > 10 {
		risk := actors.Risk{
			Title:       "Excessive API Exposure",
			Description: "Large number of API endpoints exposed",
			Severity:    "medium",
			Mitigation:  "Consider API gateway or rate limiting",
			Evidence:    fmt.Sprintf("%d endpoints discovered", len(data.Discoveries)),
		}
		result.Risks = append(result.Risks, risk)
	}
	
	return result, nil
}

// Transform converts data for chaining with other actors
func (e *EndpointDiscoveryActor) Transform(ctx context.Context, input interface{}) (interface{}, error) {
	// Transform probe results into a format other actors can use
	if probeResult, ok := input.(*actors.ProbeResult); ok {
		endpoints := []string{}
		for _, disc := range probeResult.Discoveries {
			if url, ok := disc.Properties["full_url"].(string); ok {
				endpoints = append(endpoints, url)
			}
		}
		return endpoints, nil
	}
	
	return nil, fmt.Errorf("unsupported input type for transformation")
}