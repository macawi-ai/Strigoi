package filters

// AttackPattern represents a security pattern
type AttackPattern struct {
	Name        string
	Category    string
	Regex       string
	Severity    string
	Description string
}

// CriticalPatterns are embedded for S1 edge filtering (microsecond level)
var CriticalPatterns = []AttackPattern{
	// SQL Injection
	{
		Name:     "sql_union_select",
		Category: "sql_injection",
		Regex:    `(?i)\b(union\s+(all\s+)?select|select\s+.*\s+from\s+.*\s+union)\b`,
		Severity: "critical",
	},
	{
		Name:     "sql_concatenation",
		Category: "sql_injection",
		Regex:    `(?i)(exec\s*\(|execute\s+immediate|dbms_|xp_cmdshell)`,
		Severity: "critical",
	},
	{
		Name:     "sql_comment_bypass",
		Category: "sql_injection",
		Regex:    `(--|#|\/\*|\*\/|@@|char\s*\(|concat\s*\(|chr\s*\()`,
		Severity: "high",
	},
	
	// Command Injection
	{
		Name:     "cmd_shell_metachar",
		Category: "command_injection",
		Regex:    `(\||;|&|&&|\|\||`|\$\(|<\(|>\(|\n|\r)`,
		Severity: "critical",
	},
	{
		Name:     "cmd_common_commands",
		Category: "command_injection",
		Regex:    `(?i)\b(nc\s+-|bash\s+-|sh\s+-|wget\s+|curl\s+|chmod\s+|sudo\s+)\b`,
		Severity: "high",
	},
	
	// Path Traversal
	{
		Name:     "path_traversal_dots",
		Category: "path_traversal",
		Regex:    `(\.\.\/|\.\.\\|%2e%2e%2f|%252e%252e%252f)`,
		Severity: "high",
	},
	{
		Name:     "path_absolute",
		Category: "path_traversal",
		Regex:    `(?i)(\/etc\/passwd|\/windows\/system32|c:\\windows|\/proc\/self)`,
		Severity: "critical",
	},
	
	// XSS
	{
		Name:     "xss_script_tag",
		Category: "xss",
		Regex:    `(?i)(<script[^>]*>|<\/script>|javascript:|onerror=|onload=)`,
		Severity: "high",
	},
	{
		Name:     "xss_event_handler",
		Category: "xss",
		Regex:    `(?i)\b(onmouseover|onclick|onerror|onload|onfocus|onblur)\s*=`,
		Severity: "medium",
	},
	
	// XXE
	{
		Name:     "xxe_entity",
		Category: "xxe",
		Regex:    `<!ENTITY\s+\w+\s+SYSTEM|<!DOCTYPE[^>]+\[[^\]]+\]>`,
		Severity: "high",
	},
	
	// LDAP Injection
	{
		Name:     "ldap_metachar",
		Category: "ldap_injection",
		Regex:    `[()&|!<>=~*]|\s+(or|and)\s+`,
		Severity: "medium",
	},
}

// EntropyThresholds for detecting encrypted/compressed data
var EntropyThresholds = map[string]float64{
	"encrypted": 7.5,  // Near maximum entropy
	"compressed": 6.5, // High entropy
	"binary": 5.5,     // Moderate entropy
	"text": 4.5,       // Normal text entropy
}

// ProtocolSignatures for quick protocol detection
var ProtocolSignatures = map[string][]byte{
	"http":  []byte("HTTP/"),
	"https": []byte("HTTPS/"),
	"ssh":   []byte("SSH-"),
	"ftp":   []byte("220 "),
	"smtp":  []byte("220 "),
	"pop3":  []byte("+OK"),
	"imap":  []byte("* OK"),
}

// GetPatternsByCategory returns patterns for a specific category
func GetPatternsByCategory(category string) []AttackPattern {
	var patterns []AttackPattern
	for _, p := range CriticalPatterns {
		if p.Category == category {
			patterns = append(patterns, p)
		}
	}
	return patterns
}

// GetPatternsBySeverity returns patterns of specific severity
func GetPatternsBySeverity(severity string) []AttackPattern {
	var patterns []AttackPattern
	for _, p := range CriticalPatterns {
		if p.Severity == severity {
			patterns = append(patterns, p)
		}
	}
	return patterns
}