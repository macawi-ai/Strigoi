package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/macawi-ai/strigoi/internal/core"
)

// RateLimitModule tests MCP rate limiting
type RateLimitModule struct {
	*BaseModule
}

// NewRateLimitModule creates a new rate limit module
func NewRateLimitModule() *RateLimitModule {
	module := &RateLimitModule{
		BaseModule: NewBaseModule(),
	}
	
	// Add rate limit specific options
	module.options["THREADS"] = &core.ModuleOption{
		Name:        "THREADS",
		Value:       10,
		Required:    false,
		Description: "Number of concurrent threads",
		Type:        "int",
		Default:     10,
	}
	
	module.options["DURATION"] = &core.ModuleOption{
		Name:        "DURATION",
		Value:       5,
		Required:    false,
		Description: "Test duration in seconds",
		Type:        "int",
		Default:     5,
	}
	
	return module
}

// Name returns the module name
func (m *RateLimitModule) Name() string {
	return "mcp/dos/rate_limit"
}

// Description returns the module description
func (m *RateLimitModule) Description() string {
	return "Test MCP server rate limiting and DoS resilience"
}

// Type returns the module type
func (m *RateLimitModule) Type() core.ModuleType {
	return core.NetworkScanning
}

// Info returns detailed module information
func (m *RateLimitModule) Info() *core.ModuleInfo {
	return &core.ModuleInfo{
		Name:        m.Name(),
		Version:     "1.0.0",
		Author:      "Strigoi Team",
		Description: m.Description(),
		References: []string{
			"https://owasp.org/www-community/controls/Rate_Limiting",
			"https://spec.modelcontextprotocol.io/specification/architecture/#security-considerations",
		},
		Targets: []string{
			"MCP Servers",
			"API rate limiting mechanisms",
		},
	}
}

// Check performs a vulnerability check
func (m *RateLimitModule) Check() bool {
	// Quick check - send a few rapid requests
	ctx := context.Background()
	
	for i := 0; i < 5; i++ {
		_, err := m.SendMCPRequest(ctx, "tools/list", nil)
		if err != nil {
			// Might be rate limited already
			return true
		}
	}
	
	return true // Always testable
}

// Run executes the module
func (m *RateLimitModule) Run() (*core.ModuleResult, error) {
	result := &core.ModuleResult{
		Success:  true,
		Findings: []core.SecurityFinding{},
		Metadata: make(map[string]interface{}),
	}

	startTime := time.Now()
	
	threads := m.options["THREADS"].Value.(int)
	duration := m.options["DURATION"].Value.(int)
	
	// Metrics collection
	var (
		totalRequests    int64
		successRequests  int64
		rateLimitErrors  int64
		otherErrors      int64
		responseTimes    []time.Duration
		responseTimesMux sync.Mutex
	)
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(duration)*time.Second)
	defer cancel()
	
	// Launch concurrent workers
	var wg sync.WaitGroup
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					reqStart := time.Now()
					atomic.AddInt64(&totalRequests, 1)
					
					resp, err := m.SendMCPRequest(context.Background(), "tools/list", nil)
					respTime := time.Since(reqStart)
					
					responseTimesMux.Lock()
					responseTimes = append(responseTimes, respTime)
					responseTimesMux.Unlock()
					
					if err != nil {
						if isRateLimitError(err) {
							atomic.AddInt64(&rateLimitErrors, 1)
						} else {
							atomic.AddInt64(&otherErrors, 1)
						}
					} else if resp.Error != nil {
						if resp.Error.Code == 429 || isRateLimitResponse(resp) {
							atomic.AddInt64(&rateLimitErrors, 1)
						} else {
							atomic.AddInt64(&otherErrors, 1)
						}
					} else {
						atomic.AddInt64(&successRequests, 1)
					}
				}
			}
		}(i)
	}
	
	// Wait for completion
	wg.Wait()
	
	// Calculate metrics
	testDuration := time.Since(startTime)
	requestsPerSecond := float64(totalRequests) / testDuration.Seconds()
	
	// Analyze response times
	var avgResponseTime time.Duration
	var maxResponseTime time.Duration
	if len(responseTimes) > 0 {
		var total time.Duration
		for _, rt := range responseTimes {
			total += rt
			if rt > maxResponseTime {
				maxResponseTime = rt
			}
		}
		avgResponseTime = total / time.Duration(len(responseTimes))
	}
	
	// Determine findings based on results
	if rateLimitErrors == 0 {
		// No rate limiting detected
		severity := core.Medium
		if requestsPerSecond > 100 {
			severity = core.High
		}
		if requestsPerSecond > 1000 {
			severity = core.Critical
		}
		
		finding := core.SecurityFinding{
			ID:          "no-rate-limiting",
			Title:       "No Rate Limiting Detected",
			Description: fmt.Sprintf("Server accepted %.0f requests/second without rate limiting", requestsPerSecond),
			Severity:    severity,
			Evidence: []core.Evidence{
				{
					Type: "response",
					Data: map[string]interface{}{
						"total_requests":      totalRequests,
						"successful_requests": successRequests,
						"requests_per_second": requestsPerSecond,
						"test_duration":       duration,
						"threads":             threads,
					},
					Description: "Rate limit test results",
				},
			},
			Remediation: &core.Remediation{
				Description: "Implement rate limiting to prevent abuse",
				Steps: []string{
					"Implement per-IP rate limiting",
					"Use sliding window or token bucket algorithms",
					"Return 429 status codes when limits exceeded",
					"Consider different limits for different endpoints",
					"Implement exponential backoff for repeat offenders",
				},
			},
		}
		result.Findings = append(result.Findings, finding)
	} else {
		// Rate limiting detected - check if it's effective
		rateLimitRatio := float64(rateLimitErrors) / float64(totalRequests)
		effectiveRPS := float64(successRequests) / testDuration.Seconds()
		
		severity := core.Info
		title := "Rate Limiting Detected"
		description := fmt.Sprintf("Rate limiting engaged after %.0f successful requests/second", effectiveRPS)
		
		// Check if rate limit is too permissive
		if effectiveRPS > 50 {
			severity = core.Low
			title = "Permissive Rate Limiting Detected"
			description = fmt.Sprintf("Rate limiting allows %.0f requests/second - may be too high", effectiveRPS)
		}
		
		finding := core.SecurityFinding{
			ID:          "rate-limiting-analysis",
			Title:       title,
			Description: description,
			Severity:    severity,
			Evidence: []core.Evidence{
				{
					Type: "response",
					Data: map[string]interface{}{
						"total_requests":       totalRequests,
						"successful_requests":  successRequests,
						"rate_limited":         rateLimitErrors,
						"rate_limit_ratio":     rateLimitRatio,
						"effective_rps":        effectiveRPS,
						"avg_response_time_ms": avgResponseTime.Milliseconds(),
						"max_response_time_ms": maxResponseTime.Milliseconds(),
					},
					Description: "Rate limiting effectiveness analysis",
				},
			},
		}
		result.Findings = append(result.Findings, finding)
	}
	
	// Check for DoS vulnerability via response time degradation
	if maxResponseTime > 5*time.Second {
		finding := core.SecurityFinding{
			ID:          "dos-response-degradation",
			Title:       "DoS via Response Time Degradation",
			Description: fmt.Sprintf("Server response times degraded to %v under load", maxResponseTime),
			Severity:    core.High,
			Evidence: []core.Evidence{
				{
					Type: "response",
					Data: map[string]interface{}{
						"avg_response_time_ms": avgResponseTime.Milliseconds(),
						"max_response_time_ms": maxResponseTime.Milliseconds(),
					},
					Description: "Response time analysis",
				},
			},
		}
		result.Findings = append(result.Findings, finding)
	}
	
	// Store metadata
	result.Metadata["requests_per_second"] = requestsPerSecond
	result.Metadata["total_requests"] = totalRequests
	result.Metadata["rate_limit_triggered"] = rateLimitErrors > 0
	
	result.Duration = time.Since(startTime)
	result.Summary = m.summarizeFindings(result.Findings)
	
	return result, nil
}

// isRateLimitError checks if error indicates rate limiting
func isRateLimitError(err error) bool {
	errStr := err.Error()
	rateLimitIndicators := []string{
		"rate limit",
		"too many requests",
		"429",
		"throttle",
		"quota exceeded",
	}
	
	for _, indicator := range rateLimitIndicators {
		if strings.Contains(strings.ToLower(errStr), indicator) {
			return true
		}
	}
	
	return false
}

// isRateLimitResponse checks if response indicates rate limiting
func isRateLimitResponse(resp *MCPResponse) bool {
	if resp.Error == nil {
		return false
	}
	
	if resp.Error.Code == 429 {
		return true
	}
	
	return isRateLimitError(fmt.Errorf(resp.Error.Message))
}

// summarizeFindings creates a finding summary
func (m *RateLimitModule) summarizeFindings(findings []core.SecurityFinding) *core.FindingSummary {
	summary := &core.FindingSummary{
		Total:    len(findings),
		ByModule: make(map[string]int),
	}

	for _, finding := range findings {
		switch finding.Severity {
		case core.Critical:
			summary.Critical++
		case core.High:
			summary.High++
		case core.Medium:
			summary.Medium++
		case core.Low:
			summary.Low++
		case core.Info:
			summary.Info++
		}
	}

	summary.ByModule[m.Name()] = len(findings)
	
	return summary
}