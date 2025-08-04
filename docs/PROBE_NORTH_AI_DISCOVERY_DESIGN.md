# Probe North: AI/LLM Infrastructure Discovery Design

## Core Concept Shift
**North = Intelligence Layer** - Discovering AI, LLM, and ML infrastructure

## Discovery Categories

### 1. Cloud LLM APIs
```yaml
providers:
  openai:
    endpoints: ["/v1/completions", "/v1/chat/completions", "/v1/embeddings", "/v1/models"]
    headers: ["Authorization: Bearer", "OpenAI-Organization"]
    fingerprints: ["model", "choices", "usage", "tokens"]
  
  anthropic:
    endpoints: ["/v1/complete", "/v1/messages"]
    headers: ["X-API-Key", "anthropic-version"]
    fingerprints: ["completion", "stop_reason", "model"]
  
  google:
    endpoints: ["/v1/projects/*/locations/*/models/*:generateContent"]
    headers: ["Authorization", "x-goog-api-key"]
    fingerprints: ["candidates", "safetyRatings", "citationMetadata"]
  
  cohere:
    endpoints: ["/generate", "/embed", "/classify"]
    headers: ["Authorization: Bearer", "Cohere-Version"]
    fingerprints: ["generations", "likelihoods", "token_likelihoods"]
```

### 2. Local Model Servers
```yaml
local_servers:
  ollama:
    ports: [11434]
    endpoints: ["/api/generate", "/api/chat", "/api/tags", "/api/pull"]
    fingerprints: ["model", "prompt", "stream", "context"]
  
  llama_cpp:
    ports: [8080]
    endpoints: ["/completion", "/v1/chat/completions", "/health"]
    fingerprints: ["content", "tokens_predicted", "tokens_evaluated"]
  
  vllm:
    ports: [8000]
    endpoints: ["/v1/completions", "/v1/models", "/health"]
    fingerprints: ["model", "best_of", "presence_penalty"]
  
  text_generation_inference:
    ports: [3000]
    endpoints: ["/generate", "/generate_stream", "/info"]
    fingerprints: ["generated_text", "details", "parameters"]
```

### 3. Vector Databases
```yaml
vector_stores:
  pinecone:
    endpoints: ["/vectors/upsert", "/query", "/describe_index_stats"]
    headers: ["Api-Key"]
    fingerprints: ["matches", "namespace", "vectors"]
  
  weaviate:
    ports: [8080]
    endpoints: ["/v1/graphql", "/v1/schema", "/v1/objects"]
    fingerprints: ["Get", "Aggregate", "Explore"]
  
  chroma:
    ports: [8000]
    endpoints: ["/api/v1/collections", "/api/v1/heartbeat"]
    fingerprints: ["ids", "embeddings", "metadatas", "documents"]
  
  qdrant:
    ports: [6333]
    endpoints: ["/collections", "/points/search", "/cluster/status"]
    fingerprints: ["points", "vectors", "payload"]
```

### 4. Model Serving Platforms
```yaml
ml_platforms:
  tensorflow_serving:
    ports: [8501, 8500]
    endpoints: ["/v1/models/*/versions/*:predict"]
    fingerprints: ["predictions", "model_spec", "signature_name"]
  
  triton:
    ports: [8000, 8001]
    endpoints: ["/v2/models/*/infer", "/v2/health/ready"]
    fingerprints: ["model_name", "model_version", "outputs"]
  
  mlflow:
    ports: [5000]
    endpoints: ["/invocations", "/ping", "/version"]
    fingerprints: ["predictions", "model_uri", "run_id"]
```

## Advanced Discovery Techniques

### 1. AI-Specific Fingerprinting
```python
class AIFingerprinter:
    def identify_llm_by_response(self, response):
        """Identify LLM provider by response characteristics"""
        patterns = {
            "openai": ["choices", "usage", "model", "created"],
            "anthropic": ["completion", "stop_reason", "stop", "log_id"],
            "google": ["candidates", "promptFeedback", "safetyRatings"],
            "cohere": ["generations", "prompt", "likelihood"],
            "local_ollama": ["response", "context", "total_duration"],
        }
        
    def detect_model_capabilities(self, endpoint):
        """Probe for model capabilities"""
        capability_tests = {
            "context_window": self._test_context_limits,
            "multimodal": self._test_image_support,
            "streaming": self._test_streaming_support,
            "function_calling": self._test_function_calls,
        }
```

### 2. Traffic Pattern Analysis
```yaml
ai_traffic_patterns:
  llm_requests:
    - large_json_payloads: true
    - streaming_responses: common
    - token_counting_headers: true
    - rate_limit_headers: ["x-ratelimit-*", "retry-after"]
  
  embedding_requests:
    - batch_processing: true
    - high_dimension_vectors: true
    - consistent_payload_size: false
  
  model_inference:
    - binary_protocols: possible
    - tensor_formats: true
    - batch_predictions: common
```

### 3. Cost and Quota Detection
```python
def detect_pricing_tier(headers, response):
    """Identify pricing tier from API responses"""
    indicators = {
        "rate_limits": extract_rate_limit_headers(headers),
        "quota_remaining": headers.get("x-api-quota-remaining"),
        "model_access": detect_available_models(response),
        "feature_flags": extract_feature_availability(response)
    }
```

### 4. Model Capability Discovery
```yaml
capability_probes:
  context_testing:
    - send_increasing_token_counts
    - detect_truncation_or_errors
    - identify_max_context
  
  multimodal_testing:
    - attempt_image_upload
    - test_audio_transcription
    - check_video_support
  
  feature_detection:
    - function_calling_support
    - json_mode_availability
    - streaming_capability
    - fine_tuning_access
```

## Implementation Strategy

### Phase 1: Core AI Service Detection
1. Implement fingerprinting for major LLM providers
2. Add local model server detection (Ollama, llama.cpp)
3. Create response pattern matching

### Phase 2: Advanced Discovery
1. Traffic pattern analysis
2. Model capability probing
3. Authentication mechanism detection

### Phase 3: Intelligence Gathering
1. Model version identification
2. Cost tier detection
3. Rate limit enumeration
4. Feature availability mapping

### Phase 4: Specialized Techniques
1. Side-channel analysis for model inference
2. GraphQL introspection for vector DBs
3. Streaming protocol detection

## Security Considerations

### Ethical Boundaries
- Only probe publicly accessible endpoints
- Respect rate limits and quotas
- No exploitation of discovered vulnerabilities
- Clear documentation of findings only

### Detection Avoidance
- Implement request throttling
- Randomize probe patterns
- Use legitimate-looking queries
- Rotate user agents appropriately

## Output Example
```
════════════════ AI & LLM Infrastructure Discovery ═════════════════
Target: api.example.com
Time: 2025-08-03 21:00:00
Duration: 45s

▼ Summary
  Status: success
  AI Services Found: 3
  Security Concerns: 1 (exposed API key in header)
  
▼ Discovered AI Infrastructure
  ► OpenAI Compatible API
    Endpoint: https://api.example.com/v1/chat/completions
    Model: gpt-4-turbo-preview
    Context Window: 128k tokens
    Features: Function calling, JSON mode
    Rate Limits: 500 RPM, 10k TPD
    
  ► Local Ollama Server
    Endpoint: http://api.example.com:11434
    Models Available: llama2, mistral, codellama
    Status: Public access (NO AUTH)
    ⚠ Security Risk: Unauthenticated access
    
  ► Pinecone Vector Database
    Endpoint: https://api.example.com/vectors
    Index: product-embeddings
    Dimensions: 1536
    Vectors: ~2.3M
    
▼ Model Capabilities
  Text Generation: ✓ (Multiple providers)
  Embeddings: ✓ (OpenAI, local)
  Image Understanding: ✓ (GPT-4V detected)
  Function Calling: ✓
  Streaming: ✓
  
▼ Security Findings
  ⚠ HIGH: Ollama server exposed without authentication
    Recommendation: Implement API key or network restrictions

────────────────────────────────────────────────────────────
Completed in 45s | AI Services: 3 | Models: 5 | Vectors: 2.3M
```

This design transforms probe/north into a specialized AI infrastructure discovery tool, making it incredibly valuable for understanding an organization's AI attack surface and integration points.