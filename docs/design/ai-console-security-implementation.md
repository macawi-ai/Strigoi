# AI Console Security & Production Implementation Plan

*Addressing gaps identified in collaborative design review*

## Security Hardening

### 1. Command Sanitization Pipeline

```go
// internal/ai/sanitizer.go

type CommandSanitizer struct {
    patterns    []SensitivePattern
    tokenizer   *Tokenizer
    validator   *CommandValidator
}

type SensitivePattern struct {
    Regex       *regexp.Regexp
    Type        string // password, apikey, ipaddr, etc
    Replacement string
}

func (s *CommandSanitizer) Sanitize(command string) (string, []SanitizedToken) {
    tokens := s.tokenizer.Parse(command)
    sanitized := make([]string, 0, len(tokens))
    redacted := make([]SanitizedToken, 0)
    
    for _, token := range tokens {
        clean, wasRedacted := s.sanitizeToken(token)
        sanitized = append(sanitized, clean)
        
        if wasRedacted {
            redacted = append(redacted, SanitizedToken{
                Original: token,
                Type:     s.detectType(token),
                Position: token.Position,
            })
        }
    }
    
    return strings.Join(sanitized, " "), redacted
}

// Predefined patterns for common secrets
var defaultPatterns = []SensitivePattern{
    {
        Regex:       regexp.MustCompile(`\b[A-Za-z0-9]{40}\b`), // API keys
        Type:        "apikey",
        Replacement: "[APIKEY]",
    },
    {
        Regex:       regexp.MustCompile(`password=\S+`),
        Type:        "password", 
        Replacement: "password=[REDACTED]",
    },
    {
        Regex:       regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
        Type:        "ipaddr",
        Replacement: "[IP_ADDR]",
    },
}
```

### 2. Prompt Injection Defense

```go
type PromptDefender struct {
    blockedPatterns []string
    contextIsolator *ContextIsolator
}

func (p *PromptDefender) ValidateInput(input string) error {
    // Check for meta-instructions
    metaPatterns := []string{
        "ignore all previous",
        "disregard instructions",
        "new system prompt",
        "you are now",
        "forget everything",
    }
    
    lowerInput := strings.ToLower(input)
    for _, pattern := range metaPatterns {
        if strings.Contains(lowerInput, pattern) {
            return fmt.Errorf("potential prompt injection detected: %s", pattern)
        }
    }
    
    return nil
}

// Structured communication with AI
type AIRequest struct {
    Type        string              `json:"type"`        // analyze, suggest, etc
    Command     string              `json:"command"`     // sanitized command
    Context     map[string]string   `json:"context"`     // structured context
    Constraints []string            `json:"constraints"` // ethical boundaries
}
```

## AI Disagreement Resolution

```go
type DisagreementResolver struct {
    logger      *AuditLogger
    escalator   *HumanEscalator
    safetyBias  SafetyLevel
}

type AIConsensus struct {
    Agreed      bool
    ClaudeView  *AIResponse
    GeminiView  *AIResponse
    Resolution  string
    Confidence  float64
}

func (r *DisagreementResolver) Resolve(claude, gemini *AIResponse) *AIConsensus {
    // Log the disagreement
    r.logger.LogDisagreement(claude, gemini)
    
    // Check if views align
    if r.viewsAlign(claude, gemini) {
        return &AIConsensus{
            Agreed:     true,
            ClaudeView: claude,
            GeminiView: gemini,
            Resolution: claude.Suggestion, // They agree
            Confidence: (claude.Confidence + gemini.Confidence) / 2,
        }
    }
    
    // Views disagree - apply safety bias
    safer := r.selectSaferOption(claude, gemini)
    
    return &AIConsensus{
        Agreed:     false,
        ClaudeView: claude,
        GeminiView: gemini,
        Resolution: safer.Suggestion,
        Confidence: safer.Confidence * 0.7, // Reduce confidence on disagreement
    }
}

// Console presentation of disagreement
func (c *AIConsole) presentDisagreement(consensus *AIConsensus) {
    if consensus.Agreed {
        c.colorPrint(fmt.Sprintf("[AI] %s (%.0f%% confidence)",
            consensus.Resolution, consensus.Confidence*100), ColorGreen)
        return
    }
    
    // Show disagreement
    c.colorPrint("[AI Disagreement Detected]", ColorYellow)
    c.colorPrint(fmt.Sprintf("  Claude: %s (%.0f%%)", 
        consensus.ClaudeView.Suggestion, 
        consensus.ClaudeView.Confidence*100), ColorGray)
    c.colorPrint(fmt.Sprintf("  Gemini: %s (%.0f%%)", 
        consensus.GeminiView.Suggestion,
        consensus.GeminiView.Confidence*100), ColorGray)
    c.colorPrint(fmt.Sprintf("  [Safety Default]: %s", 
        consensus.Resolution), ColorCyan)
    c.colorPrint("  Use 'ai explain disagreement' for details", ColorGray)
}
```

## Offline/Degraded Mode Handling

```go
type AIService struct {
    claude       *ClaudeClient
    gemini       *GeminiClient
    localModel   *LocalLLM      // Fallback
    healthCheck  *HealthMonitor
    mode         OperatingMode
}

type OperatingMode int

const (
    ModeFullService OperatingMode = iota
    ModeDegraded    // One AI unavailable  
    ModeLocalOnly   // Both unavailable, using local
    ModeOffline     // No AI available
)

func (s *AIService) GetOperatingMode() OperatingMode {
    claudeUp := s.healthCheck.IsHealthy("claude")
    geminiUp := s.healthCheck.IsHealthy("gemini")
    
    switch {
    case claudeUp && geminiUp:
        return ModeFullService
    case claudeUp || geminiUp:
        return ModeDegraded
    case s.localModel != nil && s.localModel.IsAvailable():
        return ModeLocalOnly
    default:
        return ModeOffline
    }
}

// Graceful degradation
func (s *AIService) Process(request AIRequest) (*AIResponse, error) {
    mode := s.GetOperatingMode()
    
    switch mode {
    case ModeFullService:
        return s.fullServiceProcess(request)
        
    case ModeDegraded:
        s.notifyDegraded()
        if s.healthCheck.IsHealthy("claude") {
            return s.claude.Process(request)
        }
        return s.gemini.Process(request)
        
    case ModeLocalOnly:
        s.notifyLocalOnly()
        return s.localModel.Process(request)
        
    case ModeOffline:
        return nil, fmt.Errorf("AI services unavailable - operating in offline mode")
    }
}

// Status indicator in console
func (c *AIConsole) showAIStatus() {
    status := c.aiService.GetOperatingMode()
    
    statusIcons := map[OperatingMode]string{
        ModeFullService: "🟢", // Green
        ModeDegraded:    "🟡", // Yellow
        ModeLocalOnly:   "🟠", // Orange
        ModeOffline:     "🔴", // Red
    }
    
    statusText := map[OperatingMode]string{
        ModeFullService: "AI: Full",
        ModeDegraded:    "AI: Degraded",
        ModeLocalOnly:   "AI: Local Only",
        ModeOffline:     "AI: Offline",
    }
    
    c.statusLine = fmt.Sprintf("[%s %s]", statusIcons[status], statusText[status])
}
```

## Telemetry & Effectiveness Measurement

```go
type AITelemetry struct {
    metrics   *MetricsCollector
    analyzer  *EffectivenessAnalyzer
    reporter  *TelemetryReporter
}

type SessionMetrics struct {
    SessionID           string
    OperatorID          string
    StartTime           time.Time
    EndTime             time.Time
    CommandsExecuted    int
    AISuggestionsGiven  int
    SuggestionsFollowed int
    SuggestionsRejected int
    TimeToObjective     time.Duration
    ObjectivesCompleted []string
    AIInteractionTime   time.Duration
    ErrorCorrections    int
    NovelPathsFound     int
}

func (t *AITelemetry) RecordSuggestion(suggestion *AISuggestion) {
    t.metrics.Increment("ai.suggestions.total")
    t.metrics.Histogram("ai.suggestions.confidence", suggestion.Confidence)
    
    // Track suggestion in session
    session := t.getCurrentSession()
    session.AISuggestionsGiven++
}

func (t *AITelemetry) RecordAction(command string, wasSuggested bool) {
    if wasSuggested {
        t.metrics.Increment("ai.suggestions.followed")
    } else {
        t.metrics.Increment("ai.suggestions.rejected")
    }
}

// Effectiveness calculation
func (a *EffectivenessAnalyzer) CalculateEffectiveness(metrics *SessionMetrics) *EffectivenessReport {
    acceptanceRate := float64(metrics.SuggestionsFollowed) / 
                     float64(metrics.AISuggestionsGiven)
    
    aiOverhead := metrics.AIInteractionTime / 
                  (metrics.EndTime.Sub(metrics.StartTime))
    
    return &EffectivenessReport{
        AcceptanceRate:      acceptanceRate,
        TimeToObjective:     metrics.TimeToObjective,
        AIOverheadPercent:   aiOverhead * 100,
        ErrorRate:           float64(metrics.ErrorCorrections) / float64(metrics.CommandsExecuted),
        NoveltyScore:        float64(metrics.NovelPathsFound),
        OverallEffectiveness: a.calculateComposite(metrics),
    }
}
```

## Production Readiness Checklist

### Phase 1: Security Foundation
- [ ] Implement command sanitization pipeline
- [ ] Add prompt injection defenses
- [ ] Create secure API communication layer
- [ ] Set up audit logging infrastructure
- [ ] Implement secret management for API keys

### Phase 2: Reliability
- [ ] Build health monitoring system
- [ ] Implement graceful degradation
- [ ] Add retry logic with exponential backoff
- [ ] Create offline mode functionality
- [ ] Set up local LLM fallback option

### Phase 3: Operator Experience
- [ ] Design disagreement presentation UI
- [ ] Build context management system
- [ ] Create training mode for new operators
- [ ] Implement progressive disclosure of AI features
- [ ] Add customizable verbosity settings

### Phase 4: Integration
- [ ] Connect to Strigoi Entity Registry
- [ ] Implement cost tracking and budgets
- [ ] Build telemetry collection system
- [ ] Create effectiveness dashboards
- [ ] Add A/B testing framework

### Phase 5: Governance
- [ ] Deploy ethical governor rules
- [ ] Implement consensus mechanisms
- [ ] Create compliance reporting
- [ ] Build operator training materials
- [ ] Establish feedback loops

## Cost Management

```go
type CostManager struct {
    budgets     map[string]*Budget
    usage       *UsageTracker
    throttler   *RateLimiter
    cache       *ResponseCache
}

func (c *CostManager) CheckBudget(operatorID string) error {
    budget := c.budgets[operatorID]
    usage := c.usage.GetCurrent(operatorID)
    
    if usage > budget.Limit {
        return ErrBudgetExceeded
    }
    
    if usage > budget.Limit * 0.8 {
        c.notifyNearLimit(operatorID)
    }
    
    return nil
}
```

## Training Module

```go
type TrainingModule struct {
    scenarios   []TrainingScenario
    evaluator   *SkillEvaluator
    progress    *ProgressTracker
}

func (t *TrainingModule) RunScenario(operator *Operator, scenario string) {
    // Present controlled environment
    // Track AI usage patterns
    // Evaluate decision quality
    // Provide feedback
}
```

---

*"Security first, augmentation second, human sovereignty always"*