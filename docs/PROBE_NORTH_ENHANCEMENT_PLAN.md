# Probe North Enhancement Plan

Based on comprehensive analysis with Gemini, this document outlines the enhancement roadmap for probe/north's endpoint discovery capabilities.

## Current State

- Basic hardcoded endpoint list
- Simple HTTP method testing (GET, HEAD, OPTIONS)
- v2 output formatting implemented ✅
- Basic security detection (admin/config endpoints)

## Enhancement Phases

### Phase 1: Core Discovery Improvements (Priority: High)

#### 1.1 Wordlist Integration
- **SecLists Integration**: Integrate popular endpoint wordlists
- **Custom Wordlist Support**: Allow users to provide custom wordlists via `--wordlist` flag
- **Smart Wordlist Selection**: Different wordlists for different contexts (API, web app, etc.)

```go
// Example usage
./strigoi probe north https://target.com --wordlist api-endpoints.txt
./strigoi probe north https://target.com --wordlist-preset api-common
```

#### 1.2 Response Analysis Engine
- **Error Message Mining**: Extract information from verbose error messages
- **Status Code Intelligence**: Learn from 403s, 405s, and other "interesting" responses
- **Content-Type Detection**: Identify API types (REST, GraphQL, SOAP)
- **Header Analysis**: Extract framework/technology fingerprints

#### 1.3 Smart Path Construction
- **Pattern Recognition**: Learn from discovered endpoints to predict others
- **Version Iteration**: Automatically test /v1/, /v2/, /api/v1/, etc.
- **Parameter Discovery**: Extract and reuse discovered parameters

### Phase 2: API-Specific Discovery (Priority: High)

#### 2.1 REST API Discovery
- **Resource Enumeration**: /users → /users/{id}, /users/me, /users/list
- **CRUD Pattern Detection**: Identify standard CRUD endpoints
- **Hypermedia Analysis**: Follow HATEOAS links

#### 2.2 GraphQL Support
- **Introspection Queries**: Automatic schema discovery
- **Field Enumeration**: Discover available queries and mutations
- **Endpoint Detection**: Find GraphQL endpoints beyond /graphql

#### 2.3 API Documentation Discovery
- **Swagger/OpenAPI**: Find and parse API definitions
- **Schema Validation**: Compare actual vs documented endpoints
- **Hidden Endpoint Detection**: Find undocumented endpoints

### Phase 3: Security-Focused Features (Priority: Critical)

#### 3.1 High-Value Target Detection
```
Authentication: /login, /auth, /oauth, /token
Admin Access: /admin, /dashboard, /console
Data Export: /export, /backup, /dump
Debug Info: /debug, /trace, /metrics
Secrets: /.env, /config, /.git
```

#### 3.2 Misconfiguration Detection
- **Exposed Secrets**: API keys in responses, .env files
- **Debug Endpoints**: Detect development/debug endpoints in production
- **CORS Misconfigurations**: Overly permissive CORS policies

#### 3.3 Shadow API Discovery
- **JavaScript Analysis**: Extract API calls from client-side code
- **Traffic Analysis**: Passive endpoint discovery from logs
- **Version Skew Detection**: Find old API versions still running

### Phase 4: Advanced Techniques (Priority: Medium)

#### 4.1 Intelligent Fuzzing
- **Context-Aware Fuzzing**: Use discovered patterns to guide fuzzing
- **Response-Based Mutation**: Adapt based on API responses
- **Stateful Discovery**: Handle multi-step authentication flows

#### 4.2 Rate Limit Handling
- **Detection**: Identify rate limiting mechanisms
- **Adaptive Delays**: Automatically adjust request rates
- **Distribution**: Support for distributed discovery

#### 4.3 Technology-Specific Discovery
- **Framework Detection**: Laravel, Django, Express.js patterns
- **CMS Detection**: WordPress, Drupal API endpoints
- **Cloud Provider Patterns**: AWS, Azure, GCP specific endpoints

## Implementation Priorities

### Immediate (for v2.1)
1. Basic wordlist support with SecLists integration
2. Enhanced response analysis for better findings
3. API documentation endpoint discovery

### Near-term (for v2.2)
1. GraphQL introspection support
2. Smart path construction based on patterns
3. Security-focused endpoint prioritization

### Long-term (for v3.0)
1. JavaScript analysis for client-side API discovery
2. Machine learning for pattern recognition
3. Full API schema reconstruction

## Technical Considerations

### Performance
- Concurrent request handling with proper throttling
- Efficient wordlist processing
- Response caching to avoid duplicate requests

### Accuracy
- False positive reduction through response validation
- Confidence scoring for discovered endpoints
- Verification modes (passive vs active)

### Integration
- Export to other tools (Burp, OWASP ZAP)
- Import from traffic captures (HAR files)
- API for programmatic access

## Security Considerations

- **Responsible Disclosure**: Include warnings about testing only authorized targets
- **Rate Limiting**: Respect target's rate limits by default
- **Stealth Mode**: Options for low-profile discovery
- **Audit Trail**: Log all discovery activities for compliance

## Metrics for Success

1. **Coverage**: % of actual endpoints discovered
2. **Accuracy**: False positive rate < 5%
3. **Speed**: Scan completion time for typical targets
4. **Security Value**: Critical findings discovered
5. **Usability**: Time to first meaningful result

## Example Enhanced Output

```
════════════════ API & Endpoint Discovery ═════════════════
Target: https://api.example.com
Time: 2025-08-03 20:30:00
Duration: 2m 34s

▼ Summary
  Status: success
  Total Security Findings: 3 (includes misconfigurations and exposed endpoints)
  Severity Breakdown:
    ● Critical: 1
    ● High: 2
  Recommendations:
    → Immediately secure exposed .env file at /.env
    → Disable GraphQL introspection in production
    → Implement authentication on /admin endpoints

▼ Results
  ► Discovery Stats
    Total Endpoints Discovered: 147
    API Type: REST with GraphQL endpoint
    Documentation: OpenAPI 3.0 at /openapi.json
    Response Status Breakdown:
      2xx Success     : 89 endpoints
      3xx Redirect    : 12 endpoints
      4xx Auth Required: 43 endpoints
      5xx Server Error: 3 endpoints
    Security Concerns: 3 critical exposures found

  ► High-Risk Findings
    ⚠ CRITICAL: Exposed environment file
      GET /.env (text/plain)
      Contains: Database credentials, API keys
    
    ⚠ HIGH: GraphQL introspection enabled
      POST /graphql
      Risk: Full schema enumeration possible
    
    ⚠ HIGH: Unprotected admin panel
      GET /admin/dashboard (text/html)
      Status: 200 OK (No authentication required)

  ► API Documentation
    • OpenAPI 3.0: /openapi.json
    • GraphQL Schema: /graphql (introspection enabled)
    • Developer Portal: /docs

  ► Discovered Resources
    /api/v1/users     [CRUD endpoints found]
    /api/v1/products  [CRUD endpoints found]
    /api/v1/orders    [Partial - GET only]
    /api/v2/beta      [Newer version detected]
    ... and 18 more resources

────────────────────────────────────────────────────────────
Completed in 2m 34s | Requests: 1,247 | Rate: 8.1 req/s
```

This enhancement plan provides a clear roadmap for making probe/north a powerful, modern API discovery tool while maintaining the clean v2 output format we've established.