# Probe West Module: Authentication & Access Control Analysis

## Overview

The Probe West module analyzes authentication, authorization, and access control mechanisms in target systems. Following the cardinal direction metaphor where "west" represents boundaries and gateways (like a sunset), this module focuses on the security perimeter between authenticated and unauthenticated access.

## Features

### Authentication Discovery
- Common authentication endpoint detection
- Authentication type identification (Basic, Bearer, OAuth, SAML, etc.)
- Multi-factor authentication (MFA) detection
- Session management analysis

### Vulnerability Detection
- Authentication bypass attempts (header manipulation)
- Missing security headers (HSTS, CSP, etc.)
- Insecure session cookie configurations
- Weak authentication patterns

### Security Analysis
- Access control matrix generation
- Role-based access patterns
- Security header analysis
- HTTPS enforcement verification

## Usage

### Basic Scan
```bash
strigoi probe west https://example.com
```

### Dry Run (Passive Only)
```bash
strigoi probe west https://example.com --dry-run
```

### Custom Configuration
```bash
strigoi probe west https://example.com \
  --rate-limit 5.0 \
  --timeout 60s \
  --max-concurrent 3 \
  -o json
```

### Testing Localhost
```bash
# By default, private addresses are blocked for security
strigoi probe west localhost:8080  # Will fail

# Use --allow-private for local testing
strigoi probe west localhost:8080 --allow-private
```

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--dry-run` | Perform passive analysis only | `false` |
| `--rate-limit` | Requests per second | `10.0` |
| `--timeout` | Request timeout | `30s` |
| `--max-concurrent` | Maximum concurrent requests | `5` |
| `--allow-private` | Allow scanning private/local addresses | `false` |
| `-o, --output` | Output format (json/yaml/table) | `json` |
| `-v, --verbose` | Enable verbose output | `false` |

## Security Features

### Rate Limiting
- Configurable requests per second
- Respects `Retry-After` headers
- Exponential backoff on rate limit responses

### Circuit Breaker
- Prevents cascade failures
- Automatic recovery testing
- Configurable thresholds

### Private Address Protection
- Blocks scanning of private IPs
- Prevents localhost scanning
- RFC1918 compliance

### TLS Security
- Minimum TLS 1.2
- Strong cipher suites only
- Certificate verification enabled

## Output Format

### JSON Output Structure
```json
{
  "target": "https://example.com",
  "auth_endpoints": [
    {
      "url": "https://example.com/login",
      "method": "GET",
      "auth_type": "Form-based",
      "requires_auth": true,
      "headers": {
        "Strict-Transport-Security": "max-age=31536000"
      },
      "vulnerabilities": []
    }
  ],
  "session_info": [
    {
      "type": "Cookie",
      "pattern": "SESSIONID",
      "secure": true,
      "httponly": true,
      "samesite": "Lax",
      "weaknesses": []
    }
  ],
  "vulnerabilities": [
    {
      "id": "WEST-1234567890",
      "name": "Missing HSTS Header",
      "severity": "Medium",
      "category": "Transport Security",
      "description": "Endpoint lacks HSTS header",
      "remediation": "Add Strict-Transport-Security header",
      "confidence": 1.0
    }
  ],
  "statistics": {
    "endpoints_discovered": 5,
    "auth_endpoints": 3,
    "vulnerabilities_found": 2,
    "critical_vulns": 0,
    "high_vulns": 1,
    "medium_vulns": 1
  },
  "recommendations": {
    "immediate": ["Fix critical vulnerabilities"],
    "short_term": ["Implement MFA"],
    "long_term": ["Adopt OAuth 2.0"]
  }
}
```

## Common Authentication Endpoints Checked

- `/login`, `/signin`, `/auth`, `/authenticate`
- `/api/login`, `/api/auth`, `/api/v1/auth`
- `/oauth/authorize`, `/oauth/token`
- `/.well-known/openid-configuration`
- `/saml/login`, `/saml/metadata`
- `/wp-login.php`, `/admin`, `/administrator`

## Vulnerability Categories

### Critical
- Authentication bypass vulnerabilities
- Complete access control failures
- Unprotected sensitive endpoints

### High
- Missing MFA on admin endpoints
- OAuth misconfigurations
- Weak session management

### Medium
- Missing security headers
- Insecure cookie configurations
- HTTP endpoints (should be HTTPS)

### Low
- Informational findings
- Best practice deviations

## Integration with Other Modules

- **Probe North**: Uses endpoint data for targeted auth testing
- **Probe East**: Correlates auth requirements with data flows
- **Probe South**: Checks auth-related dependencies
- **Stream Tap**: Real-time auth monitoring

## Ethical Considerations

This module is designed for:
- Authorized security assessments
- Defensive security analysis
- Compliance verification

**Never use this module against systems you don't own or lack permission to test.**

## Advanced Usage

### Custom Headers
```bash
strigoi probe west https://api.example.com \
  --headers "Authorization: Bearer token" \
  --headers "X-API-Key: key123"
```

### Session Persistence
```bash
# Save session for reuse
strigoi session save auth-test \
  --description "Production auth testing config"

# Load and run
strigoi session load auth-test
strigoi run
```

## Troubleshooting

### Common Issues

1. **Rate Limiting**: Reduce `--rate-limit` value
2. **Timeouts**: Increase `--timeout` value
3. **TLS Errors**: Target may use outdated TLS
4. **403 Errors**: Target may block security scanners

### Debug Mode
```bash
strigoi probe west https://example.com -v --dry-run
```

## Future Enhancements

- OAuth 2.0 flow testing
- SAML assertion validation
- JWT token analysis
- Kerberos authentication support
- WebAuthn/FIDO2 detection
- API key security analysis