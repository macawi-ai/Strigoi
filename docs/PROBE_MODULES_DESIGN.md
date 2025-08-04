# Probe Modules Design - South & East

## Overview
This document outlines the design for the Probe/South (Dependencies & Supply Chain) and Probe/East (Data Flows & Integrations) modules, incorporating security best practices and defensive-only principles.

---

## Probe/South - Dependencies & Supply Chain Analysis

### Purpose
Analyze the "foundation" of applications by examining dependencies, libraries, and supply chain vulnerabilities.

### MVP Features
1. **Package Manager Detection**
   - Support for: npm, pip, go.mod, cargo, maven, gradle
   - Auto-detection based on manifest files
   - Validation of manifest file integrity

2. **Dependency Analysis**
   - Parse direct dependencies
   - Build full dependency tree (including transitive)
   - Extract version constraints
   - Identify dependency paths

3. **Vulnerability Scanning**
   - Integration with native tools (npm audit, pip-audit, etc.)
   - CVE database checking
   - CVSS score reporting
   - Clear remediation guidance

4. **License Analysis**
   - SPDX license identification
   - License compatibility matrix
   - Copyleft detection
   - Commercial license flagging

### Security Measures
- **Read-only operations** - Never modify files
- **Path sanitization** - Prevent directory traversal
- **Sandboxed execution** - Run external tools safely
- **No network writes** - Only read from vulnerability databases
- **Structured output** - Prevent injection via output

### Data Structure
```json
{
  "module": "probe/south",
  "package_manager": "npm",
  "manifest_file": "package.json",
  "summary": {
    "total_dependencies": 150,
    "direct_dependencies": 12,
    "transitive_dependencies": 138,
    "vulnerabilities": {
      "critical": 2,
      "high": 5,
      "medium": 8,
      "low": 12
    },
    "licenses": {
      "permissive": 120,
      "copyleft": 25,
      "commercial": 3,
      "unknown": 2
    }
  },
  "vulnerabilities": [
    {
      "cve": "CVE-2021-23337",
      "package": "lodash",
      "version": "4.17.19",
      "severity": "high",
      "cvss_score": 7.2,
      "description": "Command injection via template",
      "remediation": "Update to lodash@4.17.21",
      "dependency_path": ["myapp", "express", "lodash"],
      "confidence": "high"
    }
  ],
  "dependency_graph": {
    "nodes": [...],
    "edges": [...]
  }
}
```

### Implementation Plan
1. Create base dependency analyzer interface
2. Implement package manager detectors
3. Build safe external tool runners
4. Create vulnerability aggregator
5. Implement license analyzer
6. Add caching layer for performance

---

## Probe/East - Data Flows & Integration Analysis

### Purpose
Trace "outbound" connections, data flows, and potential information leakage paths.

### MVP Features
1. **Hardcoded Secrets Detection**
   - Pattern-based scanning (API keys, passwords, tokens)
   - Entropy analysis for high-randomness strings
   - Context-aware detection
   - Safe redaction in output

2. **API Integration Discovery**
   - Configuration file parsing
   - Source code analysis for API calls
   - Environment variable mapping
   - Service dependency graphing

3. **Data Flow Analysis**
   - Input sources identification
   - Data transformation points
   - Output destinations mapping
   - Cross-boundary data movement

4. **Information Leakage Detection**
   - Error message verbosity
   - Debug endpoint discovery
   - Exposed internal paths
   - Sensitive data in logs

### Security Measures
- **Pattern redaction** - Never output actual secrets
- **Limited scope** - Only analyze specified paths
- **No execution** - Static analysis only
- **Confidence scoring** - Clear false positive indicators
- **Safe defaults** - Conservative detection thresholds

### Data Structure
```json
{
  "module": "probe/east",
  "summary": {
    "external_services": 8,
    "potential_secrets": 3,
    "data_flows": 15,
    "leak_points": 5
  },
  "findings": [
    {
      "type": "hardcoded_secret",
      "category": "api_key",
      "location": "src/config/api.js:42",
      "confidence": "high",
      "evidence": "const apiKey = 'sk_test_[REDACTED]'",
      "remediation": "Use environment variable or secret manager",
      "data_flow": ["config", "api_client", "external_service"]
    },
    {
      "type": "data_leak",
      "category": "verbose_error",
      "location": "src/handlers/error.js:18",
      "confidence": "medium",
      "evidence": "Stack trace exposed in production",
      "impact": "Internal paths and dependencies revealed",
      "remediation": "Implement proper error handling"
    }
  ],
  "data_flows": [
    {
      "source": "user_input",
      "transformations": ["validation", "sanitization"],
      "destination": "payment_api",
      "sensitive_data": ["credit_card"],
      "protection": ["encryption", "tokenization"]
    }
  ],
  "external_services": [
    {
      "domain": "api.stripe.com",
      "purpose": "payment_processing",
      "authentication": "bearer_token",
      "data_shared": ["customer_id", "amount"]
    }
  ]
}
```

### Implementation Plan
1. Create pattern library for secret detection
2. Build AST-based code analyzer
3. Implement configuration parsers
4. Create data flow tracer
5. Build service dependency mapper
6. Add reporting with remediation guidance

---

## Shared Components

### Base Security Framework
```go
// Secure execution wrapper
type SecureExecutor struct {
    allowedPaths []string
    timeout      time.Duration
    maxOutput    int64
}

// Path validator
func (s *SecureExecutor) ValidatePath(path string) error {
    cleaned := filepath.Clean(path)
    abs, err := filepath.Abs(cleaned)
    if err != nil {
        return err
    }
    
    // Check against whitelist
    allowed := false
    for _, base := range s.allowedPaths {
        if strings.HasPrefix(abs, base) {
            allowed = true
            break
        }
    }
    
    if !allowed {
        return fmt.Errorf("path outside allowed directories")
    }
    
    return nil
}
```

### Output Sanitization
- Redact sensitive values
- Truncate long outputs
- Escape special characters
- Validate JSON structure

### Performance Optimization
- Concurrent analysis where safe
- Result caching with TTL
- Incremental scanning
- Early termination on errors

---

## Integration Guidelines

### External Tool Integration
1. **Preferred Order**:
   - Native Go libraries
   - Tool APIs (if available)
   - Command execution (last resort)

2. **Safety Measures**:
   - Timeout all operations
   - Limit output size
   - Validate tool presence
   - Handle tool failures gracefully

3. **Supported Tools**:
   - npm audit (Node.js)
   - pip-audit (Python)
   - cargo audit (Rust)
   - govulncheck (Go)
   - safety (Python)
   - trivy (universal)

### Error Handling
- Descriptive error messages
- Actionable remediation steps
- Links to documentation
- Fallback strategies

---

## Testing Strategy

### Unit Tests
- Path validation logic
- Pattern matching accuracy
- Output sanitization
- Error handling paths

### Integration Tests
- Real package managers
- Sample vulnerable projects
- Performance benchmarks
- Security boundary tests

### Security Tests
- Path traversal attempts
- Command injection attempts
- Output injection attempts
- Resource exhaustion

---

## Future Enhancements

### Advanced Features (Post-MVP)
1. **Supply Chain**:
   - SBOM generation
   - Dependency confusion detection
   - Update cadence analysis
   - Typosquatting detection

2. **Data Flows**:
   - Runtime flow observation
   - API contract validation
   - Data residency checking
   - PII detection

3. **Integration**:
   - CI/CD pipeline integration
   - IDE plugin support
   - Continuous monitoring
   - Trend analysis

### Performance Improvements
- Distributed scanning
- Incremental updates
- Smart caching
- Parallel processing

---

## Security Principles

1. **Defensive Only**: No exploitation capabilities
2. **Least Privilege**: Minimal system access
3. **Transparency**: Clear about what we scan
4. **Privacy**: No data exfiltration
5. **Reliability**: Consistent, accurate results

---

*This design prioritizes security, accuracy, and usability while maintaining the defensive-only principle of Strigoi.*