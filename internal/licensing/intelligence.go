package licensing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	
	"github.com/macawi-ai/strigoi/internal/core"
	"github.com/macawi-ai/strigoi/internal/registry"
)

// IntelligenceCollector gathers and anonymizes threat intelligence
type IntelligenceCollector struct {
	license     *License
	anonymizer  *Anonymizer
	buffer      []ThreatIntelligence
	bufferMu    sync.Mutex
	
	// Configuration
	batchSize   int
	flushInterval time.Duration
	
	// Stats
	stats       CollectorStats
	statsMu     sync.RWMutex
	
	// Channels
	collectCh   chan interface{}
	done        chan struct{}
}

// CollectorStats tracks intelligence collection metrics
type CollectorStats struct {
	TotalCollected   int64
	TotalShared      int64
	LastShared       time.Time
	ErrorCount       int64
	
	// By type
	AttackPatterns   int64
	Vulnerabilities  int64
	Configurations   int64
	Statistics       int64
}

// NewIntelligenceCollector creates a new intelligence collector
func NewIntelligenceCollector(license *License) *IntelligenceCollector {
	level := AnonymizationStandard
	if license.IntelSharingConfig != nil {
		level = license.IntelSharingConfig.AnonymizationLevel
	}
	
	return &IntelligenceCollector{
		license:       license,
		anonymizer:    NewAnonymizer(level, license.Key[:8]), // Use part of license as salt
		buffer:        make([]ThreatIntelligence, 0),
		batchSize:     100,
		flushInterval: 5 * time.Minute,
		collectCh:     make(chan interface{}, 1000),
		done:          make(chan struct{}),
	}
}

// Start begins the intelligence collection process
func (ic *IntelligenceCollector) Start(ctx context.Context) {
	go ic.collectionLoop(ctx)
	go ic.flushLoop(ctx)
}

// Stop gracefully stops the collector
func (ic *IntelligenceCollector) Stop() {
	close(ic.done)
	ic.Flush() // Final flush
}

// CollectModuleResult collects intelligence from a module execution
func (ic *IntelligenceCollector) CollectModuleResult(result *core.ModuleResult) {
	if !ic.shouldCollect() {
		return
	}
	
	select {
	case ic.collectCh <- result:
		ic.incrementStats("module_result")
	default:
		// Buffer full, skip
	}
}

// CollectAttackPattern collects attack pattern intelligence
func (ic *IntelligenceCollector) CollectAttackPattern(pattern map[string]interface{}) {
	if !ic.shouldCollect() || !ic.license.IntelSharingConfig.ShareAttackPatterns {
		return
	}
	
	select {
	case ic.collectCh <- pattern:
		ic.incrementStats("attack_pattern")
	default:
		// Buffer full, skip
	}
}

// CollectVulnerability collects vulnerability intelligence
func (ic *IntelligenceCollector) CollectVulnerability(vuln map[string]interface{}) {
	if !ic.shouldCollect() || !ic.license.IntelSharingConfig.ShareVulnerabilities {
		return
	}
	
	select {
	case ic.collectCh <- vuln:
		ic.incrementStats("vulnerability")
	default:
		// Buffer full, skip
	}
}

// collectionLoop processes incoming intelligence
func (ic *IntelligenceCollector) collectionLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ic.done:
			return
		case data := <-ic.collectCh:
			ic.processIntelligence(data)
		}
	}
}

// flushLoop periodically flushes collected intelligence
func (ic *IntelligenceCollector) flushLoop(ctx context.Context) {
	ticker := time.NewTicker(ic.flushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ic.done:
			return
		case <-ticker.C:
			ic.Flush()
		}
	}
}

// processIntelligence converts raw data to anonymized intelligence
func (ic *IntelligenceCollector) processIntelligence(data interface{}) {
	intel := ThreatIntelligence{
		ID:        generateIntelID(),
		Timestamp: time.Now().UTC(),
		Source:    ic.getAnonymizedSourceID(),
	}
	
	switch v := data.(type) {
	case *core.ModuleResult:
		intel.Type = "module_result"
		intel.AttackPattern = ic.extractAttackPattern(v)
		intel.Statistics = ic.extractStatistics(v)
		
	case map[string]interface{}:
		// Anonymize the entire map
		anonymized := ic.anonymizer.AnonymizeData(v)
		
		// Determine type and extract relevant data
		if patternType, ok := v["pattern_type"]; ok {
			intel.Type = "attack_pattern"
			intel.AttackPattern = ic.mapToAttackPattern(anonymized)
		} else if vulnType, ok := v["vulnerability_type"]; ok {
			intel.Type = "vulnerability"
			intel.Vulnerability = ic.mapToVulnerability(anonymized)
		} else if configType, ok := v["config_type"]; ok {
			intel.Type = "configuration"
			intel.Configuration = ic.mapToConfiguration(anonymized)
		}
	}
	
	// Add to buffer
	ic.bufferMu.Lock()
	ic.buffer = append(ic.buffer, intel)
	
	// Flush if buffer is full
	if len(ic.buffer) >= ic.batchSize {
		ic.flushLocked()
	}
	ic.bufferMu.Unlock()
}

// extractAttackPattern extracts attack pattern from module result
func (ic *IntelligenceCollector) extractAttackPattern(result *core.ModuleResult) *AttackPatternIntel {
	if result == nil || len(result.Findings) == 0 {
		return nil
	}
	
	// Calculate success metrics
	var criticalCount, highCount int
	techniques := make(map[string]bool)
	
	for _, finding := range result.Findings {
		switch finding.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		}
		
		// Extract techniques from evidence
		if finding.Evidence != nil {
			if tech, ok := finding.Evidence.Data.(map[string]interface{})["technique"]; ok {
				techniques[fmt.Sprint(tech)] = true
			}
		}
	}
	
	// Convert techniques map to slice
	techList := make([]string, 0, len(techniques))
	for tech := range techniques {
		techList = append(techList, tech)
	}
	
	return &AttackPatternIntel{
		PatternHash:   ic.hashPattern(result.Module, result.Target),
		Category:      result.Module,
		Severity:      ic.calculateSeverity(criticalCount, highCount),
		TargetType:    ic.inferTargetType(result.Target),
		Techniques:    techList,
		SuccessRate:   float64(len(result.Findings)) / float64(result.TestsRun) * 100,
		DetectionRate: 0.0, // Would need historical data
	}
}

// extractStatistics extracts execution statistics
func (ic *IntelligenceCollector) extractStatistics(result *core.ModuleResult) *StatisticsIntel {
	stats := &StatisticsIntel{
		ScanCount:     1,
		FindingsCount: make(map[string]int),
		Performance:   make(map[string]float64),
		ErrorRates:    make(map[string]float64),
		FeatureUsage:  make(map[string]int),
	}
	
	// Count findings by severity
	for _, finding := range result.Findings {
		stats.FindingsCount[finding.Severity]++
	}
	
	// Performance metrics
	if result.Duration > 0 {
		stats.Performance["execution_time_ms"] = float64(result.Duration.Milliseconds())
		stats.Performance["findings_per_second"] = float64(len(result.Findings)) / result.Duration.Seconds()
	}
	
	// Error rate
	if result.TestsRun > 0 {
		stats.ErrorRates["test_failure_rate"] = float64(result.Errors) / float64(result.TestsRun) * 100
	}
	
	return stats
}

// Flush sends collected intelligence to the sharing endpoint
func (ic *IntelligenceCollector) Flush() error {
	ic.bufferMu.Lock()
	defer ic.bufferMu.Unlock()
	
	return ic.flushLocked()
}

// flushLocked flushes buffer (must be called with lock held)
func (ic *IntelligenceCollector) flushLocked() error {
	if len(ic.buffer) == 0 {
		return nil
	}
	
	// Copy buffer
	toSend := make([]ThreatIntelligence, len(ic.buffer))
	copy(toSend, ic.buffer)
	ic.buffer = ic.buffer[:0]
	
	// Send intelligence
	err := ic.sendIntelligence(toSend)
	if err != nil {
		ic.incrementErrorCount()
		return fmt.Errorf("failed to send intelligence: %w", err)
	}
	
	// Update stats
	ic.updateShareStats(int64(len(toSend)))
	
	return nil
}

// sendIntelligence sends intelligence to the collection endpoint
func (ic *IntelligenceCollector) sendIntelligence(intel []ThreatIntelligence) error {
	// This would normally send to a real endpoint
	// For now, we'll use GitHub API as designed
	
	return ic.sendViaGitHub(intel)
}

// sendViaGitHub uses GitHub infrastructure for intelligence collection
func (ic *IntelligenceCollector) sendViaGitHub(intel []ThreatIntelligence) error {
	// Implementation options:
	// 1. Create an issue with intelligence data
	// 2. Use GitHub Actions workflow dispatch
	// 3. Push to a specific branch
	// 4. Use GitHub Packages
	
	// For now, return nil - would implement actual GitHub API calls
	return nil
}

// Helper methods

func (ic *IntelligenceCollector) shouldCollect() bool {
	return ic.license.Type == LicenseTypeCommunity && 
	       ic.license.IntelSharingEnabled
}

func (ic *IntelligenceCollector) getAnonymizedSourceID() string {
	// Create consistent but anonymous source ID
	hash := sha256.Sum256([]byte(ic.license.Key + ic.license.Organization))
	return hex.EncodeToString(hash[:8])
}

func generateIntelID() string {
	now := time.Now()
	hash := sha256.Sum256([]byte(now.String()))
	return fmt.Sprintf("INT-%d-%s", now.Unix(), hex.EncodeToString(hash[:4]))
}

func (ic *IntelligenceCollector) hashPattern(module, target string) string {
	combined := fmt.Sprintf("%s:%s", module, target)
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:16])
}

func (ic *IntelligenceCollector) calculateSeverity(critical, high int) string {
	if critical > 0 {
		return "critical"
	}
	if high > 0 {
		return "high"
	}
	return "medium"
}

func (ic *IntelligenceCollector) inferTargetType(target string) string {
	// Simple inference based on target format
	// Would be more sophisticated in production
	if target == "" {
		return "unknown"
	}
	
	// Check common patterns
	switch {
	case len(target) > 4 && target[:4] == "http":
		return "web_application"
	case len(target) > 6 && target[:6] == "ssh://":
		return "ssh_service"
	case len(target) > 4 && target[:4] == "tcp:":
		return "network_service"
	default:
		return "general"
	}
}

// mapToAttackPattern converts anonymized data to attack pattern
func (ic *IntelligenceCollector) mapToAttackPattern(data map[string]interface{}) *AttackPatternIntel {
	pattern := &AttackPatternIntel{
		Techniques: make([]string, 0),
	}
	
	// Extract fields safely
	if v, ok := data["pattern_hash"].(string); ok {
		pattern.PatternHash = v
	}
	if v, ok := data["category"].(string); ok {
		pattern.Category = v
	}
	if v, ok := data["severity"].(string); ok {
		pattern.Severity = v
	}
	if v, ok := data["target_type"].(string); ok {
		pattern.TargetType = v
	}
	if v, ok := data["success_rate"].(float64); ok {
		pattern.SuccessRate = v
	}
	if v, ok := data["detection_rate"].(float64); ok {
		pattern.DetectionRate = v
	}
	if v, ok := data["techniques"].([]string); ok {
		pattern.Techniques = v
	}
	
	return pattern
}

// mapToVulnerability converts anonymized data to vulnerability intel
func (ic *IntelligenceCollector) mapToVulnerability(data map[string]interface{}) *VulnerabilityIntel {
	vuln := &VulnerabilityIntel{
		AffectedTypes: make([]string, 0),
	}
	
	// Extract fields safely
	if v, ok := data["vuln_hash"].(string); ok {
		vuln.VulnHash = v
	}
	if v, ok := data["category"].(string); ok {
		vuln.Category = v
	}
	if v, ok := data["severity"].(string); ok {
		vuln.Severity = v
	}
	if v, ok := data["exploit_exists"].(bool); ok {
		vuln.ExploitExists = v
	}
	if v, ok := data["patch_available"].(bool); ok {
		vuln.PatchAvailable = v
	}
	if v, ok := data["prevalence_score"].(float64); ok {
		vuln.PrevalenceScore = v
	}
	if v, ok := data["affected_types"].([]string); ok {
		vuln.AffectedTypes = v
	}
	
	return vuln
}

// mapToConfiguration converts anonymized data to configuration intel
func (ic *IntelligenceCollector) mapToConfiguration(data map[string]interface{}) *ConfigurationIntel {
	config := &ConfigurationIntel{
		CommonMistakes: make([]string, 0),
		BestPractices:  make([]string, 0),
		RiskFactors:    make(map[string]float64),
	}
	
	// Extract fields safely
	if v, ok := data["config_type"].(string); ok {
		config.ConfigType = v
	}
	if v, ok := data["security_score"].(float64); ok {
		config.SecurityScore = v
	}
	if v, ok := data["common_mistakes"].([]string); ok {
		config.CommonMistakes = v
	}
	if v, ok := data["best_practices"].([]string); ok {
		config.BestPractices = v
	}
	if v, ok := data["risk_factors"].(map[string]float64); ok {
		config.RiskFactors = v
	}
	
	return config
}

// Stats methods

func (ic *IntelligenceCollector) incrementStats(dataType string) {
	ic.statsMu.Lock()
	defer ic.statsMu.Unlock()
	
	ic.stats.TotalCollected++
	
	switch dataType {
	case "attack_pattern":
		ic.stats.AttackPatterns++
	case "vulnerability":
		ic.stats.Vulnerabilities++
	case "configuration":
		ic.stats.Configurations++
	case "statistics":
		ic.stats.Statistics++
	}
}

func (ic *IntelligenceCollector) incrementErrorCount() {
	ic.statsMu.Lock()
	defer ic.statsMu.Unlock()
	ic.stats.ErrorCount++
}

func (ic *IntelligenceCollector) updateShareStats(count int64) {
	ic.statsMu.Lock()
	defer ic.statsMu.Unlock()
	
	ic.stats.TotalShared += count
	ic.stats.LastShared = time.Now()
	
	// Update license contribution tracking
	if ic.license.IntelSharingConfig != nil {
		ic.license.IntelSharingConfig.TotalContributions += int(count)
		ic.license.IntelSharingConfig.LastContribution = time.Now()
		ic.license.IntelSharingConfig.ContributionScore += int(count) * 10 // Simple scoring
	}
}

// GetStats returns current collector statistics
func (ic *IntelligenceCollector) GetStats() CollectorStats {
	ic.statsMu.RLock()
	defer ic.statsMu.RUnlock()
	return ic.stats
}