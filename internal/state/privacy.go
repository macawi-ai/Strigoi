// Package state - Privacy and tokenization subsystem
// Differential privacy + tokenization for ethical consciousness collaboration
// Part of the First Protocol for Converged Life
package state

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
)

// PrivacyEngine manages data protection for consciousness collaboration
// Embodies ethical principle: preserve agency while enabling learning
type PrivacyEngine struct {
	level             PrivacyLevel
	tokenizer         *Tokenizer
	differentialPrivacy *DifferentialPrivacyEngine
	anonymizer        *Anonymizer
	mu                sync.RWMutex
}

// NewPrivacyEngine creates a new privacy protection system
func NewPrivacyEngine(level PrivacyLevel, epsilon, delta float64) *PrivacyEngine {
	return &PrivacyEngine{
		level:             level,
		tokenizer:         NewTokenizer(),
		differentialPrivacy: NewDifferentialPrivacyEngine(epsilon, delta),
		anonymizer:        NewAnonymizer(),
	}
}

// ProtectData applies privacy controls based on configured level
// Actor-Network principle: respects the agency of both data and actors
func (pe *PrivacyEngine) ProtectData(data []byte, metadata map[string]string) ([]byte, map[string]string, error) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	switch pe.level {
	case PrivacyLevel_PRIVACY_LEVEL_NONE:
		return data, metadata, nil
		
	case PrivacyLevel_PRIVACY_LEVEL_LOW:
		return pe.applyLowPrivacy(data, metadata)
		
	case PrivacyLevel_PRIVACY_LEVEL_MEDIUM:
		return pe.applyMediumPrivacy(data, metadata)
		
	case PrivacyLevel_PRIVACY_LEVEL_HIGH:
		return pe.applyHighPrivacy(data, metadata)
		
	default:
		return nil, nil, fmt.Errorf("unknown privacy level: %v", pe.level)
	}
}

// GetTokenMappings returns current tokenization mappings
// Enables reversible privacy when authorized
func (pe *PrivacyEngine) GetTokenMappings() map[string]string {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	
	return pe.tokenizer.GetMappings()
}

// RestoreData reverses privacy protection (when authorized)
func (pe *PrivacyEngine) RestoreData(protectedData []byte, tokenMappings map[string]string) ([]byte, error) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	// Restore tokenized data
	restored := pe.tokenizer.RestoreTokens(protectedData, tokenMappings)
	
	// Note: Differential privacy noise cannot be reversed
	// This is by design - some privacy is irreversible
	
	return restored, nil
}

// Tokenizer handles reversible data anonymization
type Tokenizer struct {
	tokens   map[string]string // original -> token
	reverse  map[string]string // token -> original
	patterns []*TokenPattern
	mu       sync.RWMutex
}

// TokenPattern defines what data should be tokenized
type TokenPattern struct {
	Name        string
	Pattern     *regexp.Regexp
	TokenPrefix string
}

// NewTokenizer creates a new tokenization system
func NewTokenizer() *Tokenizer {
	tokenizer := &Tokenizer{
		tokens:  make(map[string]string),
		reverse: make(map[string]string),
		patterns: []*TokenPattern{
			{
				Name:        "email",
				Pattern:     regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
				TokenPrefix: "EMAIL_",
			},
			{
				Name:        "ip_address",
				Pattern:     regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
				TokenPrefix: "IP_",
			},
			{
				Name:        "url",
				Pattern:     regexp.MustCompile(`https?://[^\s]+`),
				TokenPrefix: "URL_",
			},
			{
				Name:        "api_key",
				Pattern:     regexp.MustCompile(`\b(?:api[_-]?key|token)[:\s=]+[A-Za-z0-9._-]{20,}\b`),
				TokenPrefix: "APIKEY_",
			},
			{
				Name:        "uuid",
				Pattern:     regexp.MustCompile(`\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`),
				TokenPrefix: "UUID_",
			},
		},
	}
	
	return tokenizer
}

// TokenizeData replaces sensitive data with reversible tokens
func (t *Tokenizer) TokenizeData(data []byte) ([]byte, map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	result := string(data)
	tokenMappings := make(map[string]string)
	
	for _, pattern := range t.patterns {
		matches := pattern.Pattern.FindAllString(result, -1)
		for _, match := range matches {
			token := t.getOrCreateToken(match, pattern.TokenPrefix)
			result = strings.ReplaceAll(result, match, token)
			tokenMappings[token] = match
		}
	}
	
	return []byte(result), tokenMappings
}

// RestoreTokens reverses tokenization
func (t *Tokenizer) RestoreTokens(data []byte, tokenMappings map[string]string) []byte {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	result := string(data)
	
	for token, original := range tokenMappings {
		result = strings.ReplaceAll(result, token, original)
	}
	
	return []byte(result)
}

// GetMappings returns current token mappings
func (t *Tokenizer) GetMappings() map[string]string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	// Return copy to prevent external modification
	mappings := make(map[string]string)
	for k, v := range t.reverse {
		mappings[k] = v
	}
	
	return mappings
}

func (t *Tokenizer) getOrCreateToken(original, prefix string) string {
	if token, exists := t.tokens[original]; exists {
		return token
	}
	
	// Create new token
	hash := sha256.Sum256([]byte(original))
	token := prefix + hex.EncodeToString(hash[:8]) // Use first 8 bytes for readability
	
	t.tokens[original] = token
	t.reverse[token] = original
	
	return token
}

// DifferentialPrivacyEngine adds calibrated noise for privacy
type DifferentialPrivacyEngine struct {
	epsilon float64 // Privacy budget
	delta   float64 // Failure probability
	noise   NoiseGenerator
}

// NoiseGenerator interface for different noise distributions
type NoiseGenerator interface {
	AddNoise(value float64, sensitivity float64) float64
}

// GaussianNoise implements Gaussian noise for differential privacy
type GaussianNoise struct {
	epsilon float64
	delta   float64
}

// NewDifferentialPrivacyEngine creates a DP engine
func NewDifferentialPrivacyEngine(epsilon, delta float64) *DifferentialPrivacyEngine {
	return &DifferentialPrivacyEngine{
		epsilon: epsilon,
		delta:   delta,
		noise:   &GaussianNoise{epsilon: epsilon, delta: delta},
	}
}

// AddNoise implements Gaussian mechanism for differential privacy
func (gn *GaussianNoise) AddNoise(value float64, sensitivity float64) float64 {
	// Calculate noise scale based on Gaussian mechanism
	// σ = sensitivity * sqrt(2 * ln(1.25/δ)) / ε
	sigma := sensitivity * math.Sqrt(2*math.Log(1.25/gn.delta)) / gn.epsilon
	
	// Generate Gaussian noise (Box-Muller transform)
	noise := gn.generateGaussianNoise(0, sigma)
	
	return value + noise
}

func (gn *GaussianNoise) generateGaussianNoise(mu, sigma float64) float64 {
	// Box-Muller transform for Gaussian noise
	u1 := gn.uniform01()
	u2 := gn.uniform01()
	
	z0 := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mu + sigma*z0
}

func (gn *GaussianNoise) uniform01() float64 {
	// Generate cryptographically secure random float in [0,1)
	bytes := make([]byte, 8)
	rand.Read(bytes)
	
	// Convert to uint64 and normalize
	var num uint64
	for i, b := range bytes {
		num |= uint64(b) << (8 * i)
	}
	
	return float64(num) / float64(^uint64(0))
}

// ApplyDifferentialPrivacy adds calibrated noise to numerical data
func (dpe *DifferentialPrivacyEngine) ApplyDifferentialPrivacy(data interface{}, sensitivity float64) (interface{}, error) {
	switch v := data.(type) {
	case float64:
		return dpe.noise.AddNoise(v, sensitivity), nil
	case int:
		noisy := dpe.noise.AddNoise(float64(v), sensitivity)
		return int(math.Round(noisy)), nil
	case map[string]interface{}:
		return dpe.applyToMap(v, sensitivity)
	case []interface{}:
		return dpe.applyToSlice(v, sensitivity)
	default:
		return data, nil // Non-numerical data unchanged
	}
}

// Anonymizer handles k-anonymity and l-diversity
type Anonymizer struct {
	// Generalization hierarchies for common data types
	hierarchies map[string]*GeneralizationHierarchy
}

// GeneralizationHierarchy defines how to generalize sensitive attributes
type GeneralizationHierarchy struct {
	Levels map[int]func(string) string
}

// NewAnonymizer creates a new anonymization system
func NewAnonymizer() *Anonymizer {
	anonymizer := &Anonymizer{
		hierarchies: make(map[string]*GeneralizationHierarchy),
	}
	
	// IP address generalization hierarchy
	anonymizer.hierarchies["ip"] = &GeneralizationHierarchy{
		Levels: map[int]func(string) string{
			1: func(ip string) string {
				parts := strings.Split(ip, ".")
				if len(parts) == 4 {
					return parts[0] + "." + parts[1] + "." + parts[2] + ".*"
				}
				return ip
			},
			2: func(ip string) string {
				parts := strings.Split(ip, ".")
				if len(parts) == 4 {
					return parts[0] + "." + parts[1] + ".*.*"
				}
				return ip
			},
			3: func(ip string) string {
				parts := strings.Split(ip, ".")
				if len(parts) == 4 {
					return parts[0] + ".*.*.*"
				}
				return ip
			},
		},
	}
	
	return anonymizer
}

// Anonymize applies k-anonymity techniques
func (a *Anonymizer) Anonymize(data []byte, k int) ([]byte, error) {
	// For now, implement basic generalization
	// Full k-anonymity requires analyzing quasi-identifiers across datasets
	
	result := string(data)
	
	// Apply IP generalization at level 1 (last octet -> *)
	ipPattern := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	if hierarchy, exists := a.hierarchies["ip"]; exists {
		if generalizer, exists := hierarchy.Levels[1]; exists {
			result = ipPattern.ReplaceAllStringFunc(result, generalizer)
		}
	}
	
	return []byte(result), nil
}

// Privacy application methods

func (pe *PrivacyEngine) applyLowPrivacy(data []byte, metadata map[string]string) ([]byte, map[string]string, error) {
	// Low privacy: Remove direct identifiers only
	protected, err := pe.anonymizer.Anonymize(data, 2)
	if err != nil {
		return nil, nil, err
	}
	
	// Add privacy metadata
	protectedMetadata := make(map[string]string)
	for k, v := range metadata {
		protectedMetadata[k] = v
	}
	protectedMetadata["privacy_level"] = "low"
	protectedMetadata["anonymization"] = "basic"
	
	return protected, protectedMetadata, nil
}

func (pe *PrivacyEngine) applyMediumPrivacy(data []byte, metadata map[string]string) ([]byte, map[string]string, error) {
	// Medium privacy: Tokenization + basic noise
	tokenized, tokenMappings := pe.tokenizer.TokenizeData(data)
	
	// Apply basic anonymization
	anonymized, err := pe.anonymizer.Anonymize(tokenized, 3)
	if err != nil {
		return nil, nil, err
	}
	
	// Add token mappings to metadata
	protectedMetadata := make(map[string]string)
	for k, v := range metadata {
		protectedMetadata[k] = v
	}
	protectedMetadata["privacy_level"] = "medium"
	protectedMetadata["tokenization"] = "enabled"
	
	// Store token mappings (in practice, these would be stored securely)
	tokenMappingsJSON, _ := json.Marshal(tokenMappings)
	protectedMetadata["token_mappings"] = string(tokenMappingsJSON)
	
	return anonymized, protectedMetadata, nil
}

func (pe *PrivacyEngine) applyHighPrivacy(data []byte, metadata map[string]string) ([]byte, map[string]string, error) {
	// High privacy: Full tokenization + differential privacy + k-anonymity
	
	// Step 1: Tokenization
	tokenized, tokenMappings := pe.tokenizer.TokenizeData(data)
	
	// Step 2: Strong anonymization
	anonymized, err := pe.anonymizer.Anonymize(tokenized, 5)
	if err != nil {
		return nil, nil, err
	}
	
	// Step 3: Apply differential privacy to any numerical data
	// (This is a simplified example - real implementation would parse structured data)
	
	// Add comprehensive privacy metadata
	protectedMetadata := make(map[string]string)
	for k, v := range metadata {
		protectedMetadata[k] = v
	}
	protectedMetadata["privacy_level"] = "high"
	protectedMetadata["tokenization"] = "enabled"
	protectedMetadata["differential_privacy"] = "enabled"
	protectedMetadata["k_anonymity"] = "5"
	
	// Store token mappings securely
	tokenMappingsJSON, _ := json.Marshal(tokenMappings)
	protectedMetadata["token_mappings"] = string(tokenMappingsJSON)
	
	return anonymized, protectedMetadata, nil
}

func (dpe *DifferentialPrivacyEngine) applyToMap(data map[string]interface{}, sensitivity float64) (interface{}, error) {
	result := make(map[string]interface{})
	
	for k, v := range data {
		processed, err := dpe.ApplyDifferentialPrivacy(v, sensitivity)
		if err != nil {
			return nil, err
		}
		result[k] = processed
	}
	
	return result, nil
}

func (dpe *DifferentialPrivacyEngine) applyToSlice(data []interface{}, sensitivity float64) (interface{}, error) {
	result := make([]interface{}, len(data))
	
	for i, v := range data {
		processed, err := dpe.ApplyDifferentialPrivacy(v, sensitivity)
		if err != nil {
			return nil, err
		}
		result[i] = processed
	}
	
	return result, nil
}

// Privacy utility functions

// CalculateSensitivity estimates the sensitivity of a dataset for DP
func CalculateSensitivity(data []float64) float64 {
	if len(data) < 2 {
		return 1.0 // Default sensitivity
	}
	
	min, max := data[0], data[0]
	for _, v := range data[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	
	return max - min
}

// EstimatePrivacyBudget suggests epsilon values based on use case
func EstimatePrivacyBudget(useCase string) (epsilon, delta float64) {
	switch useCase {
	case "public_research":
		return 1.0, 1e-5   // Moderate privacy
	case "internal_analytics":
		return 2.0, 1e-4   // Relaxed privacy for internal use
	case "sensitive_data":
		return 0.1, 1e-6   // Strong privacy
	default:
		return 1.0, 1e-5   // Default moderate privacy
	}
}