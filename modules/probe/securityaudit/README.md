# Strigoi Security Audit Framework

A comprehensive security audit framework for the Strigoi security platform that performs static and dynamic analysis to identify vulnerabilities, misconfigurations, and security issues.

## Features

### Security Scanners

1. **Code Security Scanner**
   - Detects unsafe code patterns
   - Identifies potential race conditions
   - Finds unhandled panics and error conditions
   - Checks for weak cryptographic algorithms

2. **Injection Scanner**
   - SQL injection detection
   - Command injection detection
   - Path traversal vulnerabilities
   - LDAP/XML injection patterns

3. **Cryptography Scanner**
   - Weak algorithm usage (MD5, SHA1, DES, RC4)
   - Insecure TLS configurations
   - Missing certificate verification
   - Weak cipher suites

4. **Authentication Scanner**
   - Hardcoded credentials
   - Missing authentication
   - Weak authentication mechanisms
   - JWT vulnerabilities

5. **Dependency Scanner**
   - Vulnerable dependency detection
   - License compliance checking
   - Outdated package identification
   - Supply chain security

6. **Configuration Scanner**
   - Insecure file permissions
   - Debug mode in production
   - Default credentials
   - Exposed services

7. **Secrets Scanner**
   - API key detection
   - Private key exposure
   - Password/token detection
   - High entropy string analysis

8. **Runtime Scanner**
   - Memory safety issues
   - Race condition detection
   - Resource leaks
   - Buffer overflows

9. **Network Scanner**
   - Unencrypted communications
   - Missing timeouts
   - Exposed ports
   - Insecure cookies

10. **Compliance Scanner**
    - PCI-DSS compliance
    - OWASP Top 10
    - CIS Benchmarks

## Installation

```bash
go install github.com/macawi-ai/strigoi/modules/probe/security_audit/cmd/audit@latest
```

## Usage

### Basic Audit

```bash
# Audit current directory
audit

# Audit specific path
audit -path ./src

# Generate report
audit -output security-report.md
```

### Full Security Audit

```bash
# Enable all scanners
audit -all -output full-report.html -format html

# With runtime analysis (requires tests)
audit -all -runtime -output complete-audit.json -format json
```

### CI/CD Integration

```bash
# Fail if critical issues found
audit -max-critical 0 -max-high 5

# Quick scan for CI
audit -code -deps -config -format json
```

### Compliance Checking

```bash
# Check OWASP compliance
audit -compliance OWASP

# Multiple standards
audit -compliance PCI-DSS,OWASP,CIS
```

## Output Formats

### Markdown Report (Default)
- Human-readable format
- Executive summary
- Detailed findings
- Prioritized recommendations

### JSON Report
- Machine-readable format
- Integration with other tools
- Detailed metrics and scores
- Structured issue data

### HTML Report
- Interactive web report
- Visual security score
- Sortable issues
- Export capabilities

## Security Issues

The framework detects various security issues categorized by severity:

### Critical (Score Impact: -20)
- SQL/Command injection
- Hardcoded credentials
- Exposed private keys
- TLS verification disabled

### High (Score Impact: -10)
- Weak cryptography
- Missing authentication
- Vulnerable dependencies
- Path traversal

### Medium (Score Impact: -5)
- Debug mode enabled
- Missing timeouts
- Non-atomic updates
- Large allocations

### Low (Score Impact: -2)
- TODO/FIXME comments
- Exposed non-standard ports
- Missing license files

## Integration

### GitHub Actions

```yaml
name: Security Audit
on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    
    - name: Setup Go
      uses: actions/setup-go@v2
      with:
        go-version: 1.19
    
    - name: Install Audit Tool
      run: go install github.com/macawi-ai/strigoi/modules/probe/security_audit/cmd/audit@latest
    
    - name: Run Security Audit
      run: audit -max-critical 0 -format json -output audit.json
    
    - name: Upload Report
      uses: actions/upload-artifact@v2
      with:
        name: security-audit
        path: audit.json
```

### GitLab CI

```yaml
security_audit:
  stage: test
  script:
    - go install github.com/macawi-ai/strigoi/modules/probe/security_audit/cmd/audit@latest
    - audit -max-critical 0 -max-high 10
  artifacts:
    reports:
      junit: audit-report.xml
```

## Customization

### Custom Scanners

```go
type CustomScanner struct{}

func (s *CustomScanner) Name() string {
    return "Custom Security Scanner"
}

func (s *CustomScanner) Scan(path string, config AuditConfig) ([]SecurityIssue, error) {
    // Implementation
}

// Add to framework
framework.AddScanner(&CustomScanner{})
```

### Custom Reporters

```go
type CustomReporter struct{}

func (r *CustomReporter) GenerateReport(results *AuditResults, output io.Writer) error {
    // Implementation
}

// Add to framework
framework.AddReporter(&CustomReporter{})
```

## Best Practices

1. **Regular Audits**
   - Run audits on every commit
   - Schedule weekly full audits
   - Monitor security score trends

2. **Threshold Management**
   - Start with loose thresholds
   - Gradually tighten as issues are fixed
   - Never allow critical issues

3. **False Positive Handling**
   - Mark false positives in code
   - Document security exceptions
   - Review marked issues regularly

4. **Remediation Process**
   - Fix critical issues immediately
   - Plan sprints for high issues
   - Track technical debt

## Security Considerations

- The audit tool requires read access to source code
- Runtime scanning may execute test code
- Network scanning may probe local services
- Sensitive findings should be handled securely

## Contributing

1. Add new vulnerability patterns
2. Improve detection accuracy
3. Add compliance standards
4. Enhance reporting formats

## License

Part of the Strigoi Security Platform - see main LICENSE file.