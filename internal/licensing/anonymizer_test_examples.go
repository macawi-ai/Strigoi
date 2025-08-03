package licensing

// Example test cases demonstrating anonymization capabilities
// This file shows how different types of sensitive data are handled

var AnonymizationExamples = []struct {
	Name        string
	Input       map[string]interface{}
	Level       AnonymizationLevel
	Expected    map[string]interface{}
	Description string
}{
	{
		Name: "Basic PII Removal",
		Input: map[string]interface{}{
			"user_email":    "john.doe@company.com",
			"phone_number":  "+1-555-123-4567",
			"ssn":           "123-45-6789",
			"credit_card":   "4111-1111-1111-1111",
			"description":   "User john.doe@company.com reported issue",
		},
		Level: AnonymizationMinimal,
		Expected: map[string]interface{}{
			"user_email":    "[EMAIL-0001]",
			"phone_number":  "[PHONE-0002]",
			"ssn":           "[SSN-REDACTED]",
			"credit_card":   "[CC-REDACTED]",
			"description":   "User [EMAIL-0001] reported issue",
		},
		Description: "Minimal anonymization removes only direct PII",
	},
	
	{
		Name: "Network Information Scrubbing",
		Input: map[string]interface{}{
			"source_ip":      "192.168.1.100",
			"destination":    "web-server.internal.corp",
			"mac_address":    "00:11:22:33:44:55",
			"external_ip":    "203.0.113.45",
			"ipv6_address":   "2001:db8::1",
			"attack_details": "Scan from 192.168.1.100 to web-server.internal.corp",
		},
		Level: AnonymizationStandard,
		Expected: map[string]interface{}{
			"source_ip":      "[IP-INTERNAL]",
			"destination":    "[HOST-0001]",
			"mac_address":    "[MAC-REDACTED]",
			"external_ip":    "203.0.x.x",
			"ipv6_address":   "[IPv6-REDACTED]",
			"attack_details": "Scan from [IP-INTERNAL] to [HOST-0001]",
		},
		Description: "Standard anonymization handles network identifiers",
	},
	
	{
		Name: "Healthcare Data (HIPAA)",
		Input: map[string]interface{}{
			"patient_name":   "Jane Smith",
			"mrn":            "MRN: 123456",
			"diagnosis":      "Condition requiring treatment",
			"provider_npi":   "1234567890",
			"visit_date":     "2025-01-15",
			"notes":          "Patient MRN:123456 visited for condition",
		},
		Level: AnonymizationStrict,
		Expected: map[string]interface{}{
			"patient_name":   "[NAME-FIELD]",
			"mrn":            "[MRN-REDACTED]",
			"diagnosis":      "Condition requiring treatment",
			"provider_npi":   "[NPI-REDACTED]",
			"visit_date":     "2025-01-15",
			"notes":          "Patient [MRN-REDACTED] visited for condition",
		},
		Description: "HIPAA compliance requires strict health data anonymization",
	},
	
	{
		Name: "Financial Data (PCI-DSS)",
		Input: map[string]interface{}{
			"card_number":    "4532-1234-5678-9012",
			"cvv":            "123",
			"expiry":         "12/25",
			"cardholder":     "JOHN DOE",
			"amount":         "$1,234.56",
			"transaction_id": "TXN-2025-001234",
		},
		Level: AnonymizationStandard,
		Expected: map[string]interface{}{
			"card_number":    "[CC-REDACTED]",
			"cvv":            "[cvv-FIELD]",
			"expiry":         "12/25",
			"cardholder":     "JOHN DOE",
			"amount":         "$1,234.56",
			"transaction_id": "[TRANSACTION-ID-REDACTED]",
		},
		Description: "PCI-DSS requires complete card data removal",
	},
	
	{
		Name: "API Keys and Credentials",
		Input: map[string]interface{}{
			"api_key":        "sk_live_abcdef123456789012345678",
			"password":       "SuperSecret123!",
			"auth_token":     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			"database_url":   "postgres://user:pass@db.internal:5432/mydb",
			"config": map[string]interface{}{
				"secret_key": "my-secret-key-12345",
				"public_key": "pk_test_1234",
			},
		},
		Level: AnonymizationStrict,
		Expected: map[string]interface{}{
			"api_key":        "[APIKEY-sk_l...28]",
			"[PASSWORD-FIELD]": "[password-FIELD]",
			"[AUTH-FIELD]":    "[auth-FIELD]",
			"database_url":   "postgres://[REDACTED]@[HOST-0001]:5432/mydb",
			"config": map[string]interface{}{
				"[SECRET-FIELD]": "[secret-FIELD]",
				"public_key":     "pk_test_1234",
			},
		},
		Description: "Strict mode handles various credential types",
	},
	
	{
		Name: "Employee and Customer IDs",
		Input: map[string]interface{}{
			"employee_id":    "EMP-12345",
			"customer_num":   "CUST-67890",
			"user_id":        "usr_abc123def456",
			"account_number": "ACC-9876543210",
			"order_id":       "ORD-2025-001",
			"log_entry":      "Employee EMP-12345 accessed customer CUST-67890 data",
		},
		Level: AnonymizationStandard,
		Expected: map[string]interface{}{
			"employee_id":    "[EMPLOYEE-ID-REDACTED]",
			"customer_num":   "[CUSTOMER-ID-REDACTED]",
			"user_id":        "[USER-FIELD]",
			"account_number": "[ACCOUNT-REDACTED]",
			"order_id":       "[TRANSACTION-ID-REDACTED]",
			"log_entry":      "[EMPLOYEE-ID-REDACTED] accessed [CUSTOMER-ID-REDACTED] data",
		},
		Description: "Standard level removes internal identifiers",
	},
	
	{
		Name: "Paranoid Mode Example",
		Input: map[string]interface{}{
			"hostname":       "prod-server-01",
			"cluster_name":   "kubernetes-prod",
			"namespace":      "payment-processing",
			"service_name":   "auth-service-v2",
			"deployment_id":  "deploy-abc123xyz",
			"error_message":  "Connection to payment-processing failed",
		},
		Level: AnonymizationParanoid,
		Expected: map[string]interface{}{
			"hostname":       "[HASH-a1b2c3d4]",
			"cluster_name":   "[HASH-e5f6g7h8]",
			"namespace":      "[HASH-i9j0k1l2]",
			"service_name":   "[HASH-m3n4o5p6]",
			"deployment_id":  "[HASH-q7r8s9t0]",
			"error_message":  "Connection to [HASH-i9j0k1l2] failed",
		},
		Description: "Paranoid mode hashes everything potentially identifying",
	},
	
	{
		Name: "Attack Pattern Intelligence",
		Input: map[string]interface{}{
			"pattern_type":   "sql_injection",
			"target_url":     "https://victim.com/app/login.php",
			"payload":        "' OR '1'='1' --",
			"source_ip":      "192.168.10.50",
			"user_agent":     "Mozilla/5.0 (attacker-tools)",
			"success":        true,
			"extracted_data": "admin@victim.com, user123@victim.com",
		},
		Level: AnonymizationStandard,
		Expected: map[string]interface{}{
			"pattern_type":   "sql_injection",
			"target_url":     "https://[REDACTED]/app/login.php",
			"payload":        "' OR '1'='1' --",
			"source_ip":      "[IP-INTERNAL]",
			"user_agent":     "Mozilla/5.0 (attacker-tools)",
			"success":        true,
			"extracted_data": "[EMAIL-0001], [EMAIL-0002]",
		},
		Description: "Attack patterns preserve technique while removing identifiers",
	},
	
	{
		Name: "Multi-Compliance Scenario",
		Input: map[string]interface{}{
			"patient_email":  "patient@hospital.org",
			"card_last_four": "4532...9012",
			"diagnosis_code": "E11.9",
			"billing_amount": "$500.00",
			"provider_name":  "Dr. Smith",
			"facility":       "General Hospital",
			"geo_location":   "37.7749,-122.4194",
		},
		Level: AnonymizationStrict,
		Expected: map[string]interface{}{
			"patient_email":  "[EMAIL-0001]",
			"card_last_four": "[CC-REDACTED]",
			"diagnosis_code": "E11.9",
			"billing_amount": "$500.00",
			"provider_name":  "Dr. Smith",
			"facility":       "General Hospital",
			"geo_location":   "[GPS-REDACTED]",
		},
		Description: "Combined HIPAA and PCI-DSS compliance",
	},
}

// DemonstrateAnonymization shows how the anonymizer works
func DemonstrateAnonymization() {
	anonymizer := NewAnonymizer(AnonymizationStandard, "demo-salt")
	
	// Example: Module scan result
	scanResult := map[string]interface{}{
		"module":         "web-scanner",
		"target":         "https://10.0.1.50:8080/api",
		"findings": []interface{}{
			map[string]interface{}{
				"type":        "sql_injection",
				"url":         "https://10.0.1.50:8080/api/users",
				"parameter":   "id",
				"evidence":    "Error: You have an error in your SQL syntax near 'admin@company.com'",
				"severity":    "high",
				"remediation": "Use parameterized queries",
			},
			map[string]interface{}{
				"type":        "exposed_api_key",
				"location":    "/api/config",
				"key_type":    "stripe",
				"key_value":   "sk_live_4eC39HqLyjWDarjtT1zdp7dc",
				"severity":    "critical",
			},
		},
		"scan_metadata": map[string]interface{}{
			"scanner_ip":   "192.168.1.100",
			"scan_id":      "550e8400-e29b-41d4-a716-446655440000",
			"operator":     "security-team@company.com",
			"start_time":   "2025-01-20T10:00:00Z",
		},
	}
	
	// Anonymize the result
	anonymized := anonymizer.AnonymizeData(scanResult)
	
	// The output would look like:
	// {
	//   "module": "web-scanner",
	//   "target": "https://[IP-INTERNAL]:8080/api",
	//   "findings": [
	//     {
	//       "type": "sql_injection",
	//       "url": "https://[IP-INTERNAL]:8080/api/users",
	//       "parameter": "id",
	//       "evidence": "Error: You have an error in your SQL syntax near '[EMAIL-0001]'",
	//       "severity": "high",
	//       "remediation": "Use parameterized queries"
	//     },
	//     {
	//       "type": "exposed_api_key",
	//       "location": "/api/config",
	//       "key_type": "stripe",
	//       "key_value": "[APIKEY-sk_l...28]",
	//       "severity": "critical"
	//     }
	//   ],
	//   "scan_metadata": {
	//     "scanner_ip": "[IP-INTERNAL]",
	//     "scan_id": "550e8400-e29b-41d4-a716-446655440000",
	//     "operator": "[EMAIL-0002]",
	//     "start_time": "2025-01-20T10:00:00Z"
	//   }
	// }
}

// ComplianceScenarios shows different compliance requirements
var ComplianceScenarios = map[string][]string{
	"Healthcare Provider": {"HIPAA", "PCI-DSS"},           // Handles patient data and payments
	"E-commerce Platform": {"PCI-DSS", "GDPR", "CCPA"},    // Payment processing and EU customers
	"Financial Services":  {"GLBA", "PCI-DSS", "SOX"},     // Financial data protection
	"Global SaaS":         {"GDPR", "CCPA", "PIPEDA", "LGPD"}, // Multi-jurisdiction
	"Government Contract": {"FISMA", "HIPAA", "CJIS"},     // Federal requirements
}