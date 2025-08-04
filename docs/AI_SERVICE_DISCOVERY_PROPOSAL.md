# Ethical AI Service Discovery Proposal

## Problem Statement
Currently, there's no standardized way for AI services to ethically identify themselves to discovery tools. This leads to:
- Aggressive endpoint probing
- Unclear service boundaries
- No standardized way to communicate capabilities, limits, and ethical guidelines

## Proposed Standard: `/.well-known/ai-service`

Following RFC 8615 (well-known URIs), we propose a standardized endpoint for AI service self-identification.

### Example Response

```json
{
  "service": {
    "name": "Anthropic Claude API",
    "provider": "Anthropic",
    "version": "2024-02-01",
    "description": "Constitutional AI assistant API"
  },
  "capabilities": [
    {
      "type": "chat",
      "endpoint": "/v1/messages",
      "models": ["claude-3-opus", "claude-3-sonnet", "claude-3-haiku"]
    },
    {
      "type": "completion",
      "endpoint": "/v1/complete",
      "models": ["claude-2.1", "claude-instant-1.2"]
    }
  ],
  "authentication": {
    "type": "api_key",
    "header": "X-API-Key",
    "documentation": "https://docs.anthropic.com/claude/reference/authentication"
  },
  "rate_limits": {
    "requests_per_minute": 60,
    "tokens_per_minute": 100000,
    "documentation": "https://docs.anthropic.com/claude/reference/rate-limits"
  },
  "ethical_guidelines": {
    "usage_policy": "https://www.anthropic.com/legal/use-policy",
    "safety_measures": "https://www.anthropic.com/claude/safety",
    "bias_mitigation": "https://www.anthropic.com/research/constitutional-ai"
  },
  "support": {
    "documentation": "https://docs.anthropic.com",
    "contact": "support@anthropic.com",
    "status_page": "https://status.anthropic.com"
  },
  "discovery": {
    "preferred_methods": ["well-known", "options"],
    "rate_limit_discovery": 10,
    "cache_duration": 3600
  }
}
```

### Alternative: OPTIONS Method

For services that can't implement well-known URIs, the OPTIONS method on the API root could return similar information:

```
OPTIONS /v1 HTTP/1.1
Host: api.anthropic.com

Response:
HTTP/1.1 200 OK
Content-Type: application/json
X-AI-Service: Anthropic Claude API
X-AI-Provider: Anthropic
```

### Benefits

1. **Ethical Discovery**: Services explicitly consent to discovery
2. **Reduced Probing**: One request provides all needed information
3. **Clear Boundaries**: Services define their own capabilities
4. **Transparency**: Rate limits and ethical guidelines upfront
5. **Standardization**: Consistent format across providers

### Implementation Recommendations

1. **Graceful Fallback**: If not implemented, fall back to current detection methods
2. **Caching**: Respect cache_duration to minimize repeated queries
3. **Rate Limiting**: Honor discovery rate limits
4. **Version Negotiation**: Support content negotiation for future versions

### Security Considerations

1. **No Sensitive Data**: Response should contain only public information
2. **CORS Headers**: Enable cross-origin requests for browser-based tools
3. **Authentication**: Discovery endpoint should not require authentication
4. **Rate Limiting**: Implement rate limiting to prevent abuse

### Next Steps

1. **Community Feedback**: Gather input from AI providers and consumers
2. **Pilot Implementation**: Work with willing providers to test
3. **Standardization**: Submit to appropriate standards body (W3C, IETF)
4. **Tool Support**: Update discovery tools to check this endpoint first

## Example Implementation in Strigoi

```go
// Check ethical discovery endpoint first
func (m *NorthModule) checkEthicalDiscovery(client *http.Client, target string) (*AIServiceInfo, error) {
    // Try .well-known first
    resp, err := client.Get(target + "/.well-known/ai-service")
    if err == nil && resp.StatusCode == 200 {
        var info AIServiceInfo
        if err := json.NewDecoder(resp.Body).Decode(&info); err == nil {
            return &info, nil
        }
    }
    
    // Try OPTIONS on API root
    req, _ := http.NewRequest("OPTIONS", target + "/v1", nil)
    resp, err = client.Do(req)
    if err == nil && resp.StatusCode == 200 {
        // Parse OPTIONS response
        if provider := resp.Header.Get("X-AI-Provider"); provider != "" {
            return &AIServiceInfo{
                Service: ServiceInfo{
                    Provider: provider,
                    Name: resp.Header.Get("X-AI-Service"),
                },
            }, nil
        }
    }
    
    return nil, fmt.Errorf("no ethical discovery endpoint found")
}
```

## Conclusion

By establishing a standard for ethical AI service discovery, we can create a more transparent, efficient, and respectful ecosystem for AI infrastructure scanning and monitoring.