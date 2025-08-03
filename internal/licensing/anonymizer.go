package licensing

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Anonymizer handles all data scrubbing and anonymization
type Anonymizer struct {
	level         AnonymizationLevel
	tokenMap      map[string]string
	reverseMap    map[string]string
	tokenCounter  int
	mu            sync.RWMutex
	salt          string
	policies      []CompliancePolicy
}

// NewAnonymizer creates a new anonymizer with specified level
func NewAnonymizer(level AnonymizationLevel, salt string) *Anonymizer {
	return &Anonymizer{
		level:      level,
		tokenMap:   make(map[string]string),
		reverseMap: make(map[string]string),
		salt:       salt,
	}
}

// SetCompliancePolicies sets active compliance policies
func (a *Anonymizer) SetCompliancePolicies(policies []CompliancePolicy) {
	a.policies = policies
}

// AnonymizeData scrubs sensitive information from arbitrary data
func (a *Anonymizer) AnonymizeData(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for key, value := range data {
		// Check if key itself needs anonymization
		anonKey := a.anonymizeKey(key)
		
		// Anonymize the value
		switch v := value.(type) {
		case string:
			result[anonKey] = a.anonymizeString(v)
		case []string:
			result[anonKey] = a.anonymizeStringSlice(v)
		case map[string]interface{}:
			result[anonKey] = a.AnonymizeData(v)
		case []interface{}:
			result[anonKey] = a.anonymizeSlice(v)
		default:
			// For non-string types, apply selective anonymization
			result[anonKey] = a.anonymizeValue(v)
		}
	}
	
	return result
}

// Comprehensive regex patterns for sensitive data
var (
	// Personal Identifiers
	emailRegex        = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phoneRegex        = regexp.MustCompile(`(\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`)
	ssnRegex          = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	
	// Financial Data
	creditCardRegex   = regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`)
	ibanRegex         = regexp.MustCompile(`[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}([A-Z0-9]?){0,16}`)
	
	// Network Identifiers  
	ipv4Regex         = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)
	ipv6Regex         = regexp.MustCompile(`(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`)
	macRegex          = regexp.MustCompile(`([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})`)
	
	// Hostnames and Domains
	internalHostRegex = regexp.MustCompile(`[a-zA-Z0-9-]+\.(local|internal|corp|lan|localdomain|localhost)`)
	
	// IDs and Keys
	uuidRegex         = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	apiKeyRegex       = regexp.MustCompile(`[a-zA-Z0-9_-]{20,}`)
	
	// Health Information (HIPAA)
	mrnRegex          = regexp.MustCompile(`MRN[:\s]?\d{6,}`)
	npiRegex          = regexp.MustCompile(`\b\d{10}\b`) // National Provider ID
	
	// Geographic Data
	gpsRegex          = regexp.MustCompile(`[-+]?([1-8]?\d(\.\d+)?|90(\.0+)?),\s*[-+]?(180(\.0+)?|((1[0-7]\d)|([1-9]?\d))(\.\d+)?)`)
)

// anonymizeString applies all anonymization rules to a string
func (a *Anonymizer) anonymizeString(s string) string {
	if s == "" {
		return s
	}
	
	// Apply level-based anonymization
	switch a.level {
	case AnonymizationMinimal:
		return a.anonymizeMinimal(s)
	case AnonymizationStandard:
		return a.anonymizeStandard(s)
	case AnonymizationStrict:
		return a.anonymizeStrict(s)
	case AnonymizationParanoid:
		return a.anonymizeParanoid(s)
	}
	
	return s
}

// anonymizeMinimal - Only direct PII
func (a *Anonymizer) anonymizeMinimal(s string) string {
	s = emailRegex.ReplaceAllStringFunc(s, a.tokenizeEmail)
	s = phoneRegex.ReplaceAllStringFunc(s, a.tokenizePhone)
	s = ssnRegex.ReplaceAllString(s, "[SSN-REDACTED]")
	s = creditCardRegex.ReplaceAllString(s, "[CC-REDACTED]")
	s = ibanRegex.ReplaceAllString(s, "[IBAN-REDACTED]")
	return s
}

// anonymizeStandard - PII + internal identifiers
func (a *Anonymizer) anonymizeStandard(s string) string {
	s = a.anonymizeMinimal(s)
	
	// Network identifiers
	s = a.anonymizeIPs(s)
	s = macRegex.ReplaceAllString(s, "[MAC-REDACTED]")
	s = internalHostRegex.ReplaceAllStringFunc(s, a.tokenizeHostname)
	
	// IDs
	s = uuidRegex.ReplaceAllStringFunc(s, a.tokenizeUUID)
	
	// Health data
	s = mrnRegex.ReplaceAllString(s, "[MRN-REDACTED]")
	s = npiRegex.ReplaceAllString(s, "[NPI-REDACTED]")
	
	return s
}

// anonymizeStrict - Everything identifiable
func (a *Anonymizer) anonymizeStrict(s string) string {
	s = a.anonymizeStandard(s)
	
	// API keys and tokens
	if len(s) > 20 && apiKeyRegex.MatchString(s) {
		s = a.tokenizeAPIKey(s)
	}
	
	// GPS coordinates
	s = gpsRegex.ReplaceAllString(s, "[GPS-REDACTED]")
	
	// Any remaining potential identifiers
	s = a.scrubPotentialIdentifiers(s)
	
	return s
}

// anonymizeParanoid - Maximum scrubbing
func (a *Anonymizer) anonymizeParanoid(s string) string {
	s = a.anonymizeStrict(s)
	
	// Hash all remaining alphanumeric sequences > 6 chars
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 6 && containsAlphanumeric(word) {
			words[i] = a.hashToken(word)
		}
	}
	
	return strings.Join(words, " ")
}

// anonymizeIPs handles IP address anonymization
func (a *Anonymizer) anonymizeIPs(s string) string {
	// IPv4
	s = ipv4Regex.ReplaceAllStringFunc(s, func(ip string) string {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			return "[IP-INVALID]"
		}
		
		// Check if internal
		if isInternalIP(parsed) {
			return "[IP-INTERNAL]"
		}
		
		// For external IPs, preserve first two octets
		parts := strings.Split(ip, ".")
		if len(parts) == 4 {
			return fmt.Sprintf("%s.%s.x.x", parts[0], parts[1])
		}
		return "[IP-REDACTED]"
	})
	
	// IPv6
	s = ipv6Regex.ReplaceAllString(s, "[IPv6-REDACTED]")
	
	return s
}

// tokenizeEmail creates a reversible token for emails
func (a *Anonymizer) tokenizeEmail(email string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if token, exists := a.tokenMap[email]; exists {
		return token
	}
	
	a.tokenCounter++
	token := fmt.Sprintf("[EMAIL-%04d]", a.tokenCounter)
	a.tokenMap[email] = token
	a.reverseMap[token] = email
	
	return token
}

// tokenizePhone creates a token for phone numbers
func (a *Anonymizer) tokenizePhone(phone string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if token, exists := a.tokenMap[phone]; exists {
		return token
	}
	
	a.tokenCounter++
	token := fmt.Sprintf("[PHONE-%04d]", a.tokenCounter)
	a.tokenMap[phone] = token
	a.reverseMap[token] = phone
	
	return token
}

// tokenizeHostname creates a token for internal hostnames
func (a *Anonymizer) tokenizeHostname(hostname string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if token, exists := a.tokenMap[hostname]; exists {
		return token
	}
	
	a.tokenCounter++
	token := fmt.Sprintf("[HOST-%04d]", a.tokenCounter)
	a.tokenMap[hostname] = token
	a.reverseMap[token] = hostname
	
	return token
}

// tokenizeUUID creates a token for UUIDs
func (a *Anonymizer) tokenizeUUID(uuid string) string {
	// For UUIDs, preserve format but hash content
	hash := sha256.Sum256([]byte(uuid + a.salt))
	hashStr := hex.EncodeToString(hash[:])
	
	// Format as UUID-like token
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hashStr[0:8],
		hashStr[8:12],
		hashStr[12:16],
		hashStr[16:20],
		hashStr[20:32])
}

// tokenizeAPIKey handles API key anonymization
func (a *Anonymizer) tokenizeAPIKey(key string) string {
	// Preserve length and format hints
	prefix := ""
	if len(key) > 4 {
		prefix = key[:4]
	}
	
	return fmt.Sprintf("[APIKEY-%s...%d]", prefix, len(key))
}

// hashToken creates a one-way hash of a token
func (a *Anonymizer) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token + a.salt))
	return fmt.Sprintf("[HASH-%s]", hex.EncodeToString(hash[:8]))
}

// scrubPotentialIdentifiers removes other potential identifiers
func (a *Anonymizer) scrubPotentialIdentifiers(s string) string {
	// Employee IDs (various formats)
	s = regexp.MustCompile(`(?i)(employee|emp|user|usr)[-_\s]?(id|num|number)?[-_\s]?[:=]?\s*\w+`).
		ReplaceAllString(s, "[EMPLOYEE-ID-REDACTED]")
	
	// Customer IDs
	s = regexp.MustCompile(`(?i)(customer|cust|client)[-_\s]?(id|num|number)?[-_\s]?[:=]?\s*\w+`).
		ReplaceAllString(s, "[CUSTOMER-ID-REDACTED]")
	
	// Account numbers
	s = regexp.MustCompile(`(?i)(account|acct)[-_\s]?(num|number)?[-_\s]?[:=]?\s*\d+`).
		ReplaceAllString(s, "[ACCOUNT-REDACTED]")
	
	// Order/Transaction IDs
	s = regexp.MustCompile(`(?i)(order|transaction|trans)[-_\s]?(id|num|number)?[-_\s]?[:=]?\s*\w+`).
		ReplaceAllString(s, "[TRANSACTION-ID-REDACTED]")
	
	return s
}

// anonymizeKey anonymizes field names that might contain sensitive info
func (a *Anonymizer) anonymizeKey(key string) string {
	lowerKey := strings.ToLower(key)
	
	sensitiveKeys := []string{
		"password", "passwd", "pwd", "secret", "token", "key", "auth",
		"credential", "private", "ssn", "dob", "birthdate", "email",
		"phone", "address", "name", "user", "username", "account",
	}
	
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(lowerKey, sensitive) {
			return fmt.Sprintf("[%s-FIELD]", strings.ToUpper(sensitive))
		}
	}
	
	return key
}

// anonymizeStringSlice applies anonymization to a slice of strings
func (a *Anonymizer) anonymizeStringSlice(slice []string) []string {
	result := make([]string, len(slice))
	for i, s := range slice {
		result[i] = a.anonymizeString(s)
	}
	return result
}

// anonymizeSlice applies anonymization to a generic slice
func (a *Anonymizer) anonymizeSlice(slice []interface{}) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		switch val := v.(type) {
		case string:
			result[i] = a.anonymizeString(val)
		case map[string]interface{}:
			result[i] = a.AnonymizeData(val)
		default:
			result[i] = a.anonymizeValue(val)
		}
	}
	return result
}

// anonymizeValue handles non-string values
func (a *Anonymizer) anonymizeValue(v interface{}) interface{} {
	// For now, pass through non-string values
	// Could be extended to handle specific types
	return v
}

// GetTokenMapping returns the token mapping for authorized reversal
func (a *Anonymizer) GetTokenMapping() map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	// Return a copy to prevent external modification
	mapping := make(map[string]string)
	for k, v := range a.reverseMap {
		mapping[k] = v
	}
	return mapping
}

// Helper functions

func isInternalIP(ip net.IP) bool {
	private := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7",
	}
	
	for _, cidr := range private {
		_, network, _ := net.ParseCIDR(cidr)
		if network != nil && network.Contains(ip) {
			return true
		}
	}
	
	return ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

func containsAlphanumeric(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return true
		}
	}
	return false
}

// ComplianceFilter applies compliance-specific filtering
func (a *Anonymizer) ComplianceFilter(data map[string]interface{}, policies []string) map[string]interface{} {
	// Apply specific compliance requirements
	filtered := make(map[string]interface{})
	
	for _, policy := range policies {
		switch policy {
		case "GDPR":
			data = a.applyGDPRFilter(data)
		case "HIPAA":
			data = a.applyHIPAAFilter(data)
		case "PCI-DSS":
			data = a.applyPCIDSSFilter(data)
		case "CCPA":
			data = a.applyCCPAFilter(data)
		}
	}
	
	// Copy allowed fields
	for k, v := range data {
		if !a.isRestrictedField(k, policies) {
			filtered[k] = v
		}
	}
	
	return filtered
}

func (a *Anonymizer) applyGDPRFilter(data map[string]interface{}) map[string]interface{} {
	// GDPR-specific filtering
	delete(data, "ip_address")
	delete(data, "user_id")
	delete(data, "device_id")
	delete(data, "biometric_data")
	return data
}

func (a *Anonymizer) applyHIPAAFilter(data map[string]interface{}) map[string]interface{} {
	// HIPAA-specific filtering
	healthFields := []string{
		"diagnosis", "treatment", "medication", "health_record",
		"medical_record", "patient_id", "provider_id", "insurance_id",
	}
	
	for _, field := range healthFields {
		delete(data, field)
	}
	return data
}

func (a *Anonymizer) applyPCIDSSFilter(data map[string]interface{}) map[string]interface{} {
	// PCI-DSS specific filtering
	delete(data, "card_number")
	delete(data, "cvv")
	delete(data, "expiry_date")
	delete(data, "cardholder_name")
	return data
}

func (a *Anonymizer) applyCCPAFilter(data map[string]interface{}) map[string]interface{} {
	// CCPA-specific filtering
	delete(data, "precise_geolocation")
	delete(data, "biometric_data")
	delete(data, "browsing_history")
	return data
}

func (a *Anonymizer) isRestrictedField(field string, policies []string) bool {
	// Check if field is restricted by any policy
	restrictedFields := map[string][]string{
		"GDPR":    {"ip_address", "email", "name", "address", "phone"},
		"HIPAA":   {"patient_name", "medical_record", "diagnosis"},
		"PCI-DSS": {"card_number", "cvv", "cardholder_name"},
		"CCPA":    {"precise_location", "biometric_data"},
	}
	
	field = strings.ToLower(field)
	for _, policy := range policies {
		if restricted, ok := restrictedFields[policy]; ok {
			for _, r := range restricted {
				if strings.Contains(field, r) {
					return true
				}
			}
		}
	}
	
	return false
}