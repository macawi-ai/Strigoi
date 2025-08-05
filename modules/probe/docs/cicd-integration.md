# CI/CD Integration Guide for Strigoi Security Platform

This guide provides comprehensive instructions for integrating the Strigoi security audit framework into your CI/CD pipelines.

## Overview

The Strigoi platform includes built-in CI/CD configurations for major platforms:
- GitHub Actions
- GitLab CI
- Jenkins
- CircleCI

Each configuration provides:
- Multi-version Go testing
- Security scanning and vulnerability detection
- Performance benchmarking
- Load testing
- Docker image building
- Automated releases

## GitHub Actions

### Basic Setup

The GitHub Actions workflow is located at `.github/workflows/security-pipeline.yml`.

Key features:
- Matrix testing across Go versions 1.19, 1.20, and 1.21
- Parallel security scanning with multiple tools
- Performance regression detection for PRs
- Automated release process

### Customization

```yaml
# Customize Go versions
strategy:
  matrix:
    go-version: ['1.19', '1.20', '1.21', '1.22']

# Adjust security thresholds
- name: Run Strigoi Security Audit
  run: |
    audit -all -max-critical 0 -max-high 10 -format json
```

### Required Secrets
- `GITHUB_TOKEN` (automatically provided)
- `CODECOV_TOKEN` (for coverage reporting)

## GitLab CI

### Basic Setup

The GitLab CI configuration is in `.gitlab-ci.yml`.

Key features:
- Docker-in-Docker support for container scanning
- Integration with GitLab security dashboards
- Scheduled nightly security scans
- GitLab Pages for documentation

### Variables

```yaml
variables:
  GO_VERSION: "1.20"
  GOLANGCI_LINT_VERSION: "v1.54.2"
  AUDIT_FLAGS: "-all -runtime -network"
```

### Container Registry

```yaml
deploy:docker:
  script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
```

## Jenkins

### Basic Setup

The Jenkins pipeline is defined in `Jenkinsfile`.

Requirements:
- Jenkins with Pipeline plugin
- Docker support on agents
- SonarQube integration (optional)

### Pipeline Configuration

```groovy
environment {
    GO_VERSION = '1.20'
    DOCKER_REGISTRY = credentials('docker-registry')
    SONAR_TOKEN = credentials('sonar-token')
}
```

### Quality Gates

```groovy
stage('Quality Gate') {
    steps {
        timeout(time: 5, unit: 'MINUTES') {
            waitForQualityGate abortPipeline: true
        }
    }
}
```

### Notifications

Configure email notifications:

```groovy
post {
    failure {
        emailext(
            subject: "FAILURE: ${env.JOB_NAME}",
            body: "Pipeline failed for ${env.GIT_COMMIT_SHORT}",
            to: '${DEFAULT_RECIPIENTS}'
        )
    }
}
```

## CircleCI

### Basic Setup

CircleCI configuration is in `.circleci/config.yml`.

Features:
- Orbs for simplified configuration
- Workspace persistence for efficiency
- Scheduled nightly workflows
- Performance comparison for branches

### Executors

```yaml
executors:
  go-executor:
    docker:
      - image: cimg/go:1.20
    resource_class: large
```

### Workflows

```yaml
workflows:
  nightly:
    triggers:
      - schedule:
          cron: "0 0 * * *"
          filters:
            branches:
              only: main
```

## Common Integration Patterns

### 1. Pull Request Validation

All platforms support PR validation:

```yaml
# GitHub Actions
on:
  pull_request:
    branches: [ main ]

# GitLab CI
only:
  - merge_requests

# Jenkins
when {
    changeRequest()
}

# CircleCI
filters:
  branches:
    ignore: main
```

### 2. Security Thresholds

Fail builds based on security findings:

```bash
# Critical issues block deployment
audit -max-critical 0

# Allow some high-severity issues during development
audit -max-critical 0 -max-high 5

# Strict mode for production
audit -all -max-critical 0 -max-high 0 -max-medium 10
```

### 3. Performance Regression Detection

Compare benchmarks between branches:

```bash
# Run benchmarks on base branch
git checkout main
go test -bench=. -benchmem > base-bench.txt

# Run benchmarks on feature branch
git checkout feature-branch
go test -bench=. -benchmem > feature-bench.txt

# Compare results
benchstat base-bench.txt feature-bench.txt
```

### 4. Compliance Checking

Integrate compliance validation:

```bash
# Check specific standards
audit -compliance OWASP,PCI-DSS,CIS

# Generate compliance report
audit -compliance OWASP -format html -output compliance.html
```

## Security Best Practices

### 1. Secret Management

Never hardcode secrets in pipeline files:

```yaml
# Good - use secret management
env:
  API_KEY: ${{ secrets.API_KEY }}

# Bad - hardcoded secret
env:
  API_KEY: "sk_live_abcd1234"
```

### 2. Artifact Security

Sign and verify artifacts:

```yaml
- name: Sign artifacts
  run: |
    cosign sign-blob \
      --key cosign.key \
      --output-signature artifact.sig \
      artifact.tar.gz
```

### 3. Container Scanning

Scan Docker images before deployment:

```yaml
- name: Scan Docker image
  run: |
    trivy image \
      --severity HIGH,CRITICAL \
      --exit-code 1 \
      myapp:latest
```

### 4. Dependency Verification

Verify dependencies haven't been tampered with:

```bash
go mod verify
go list -json -deps ./... | nancy sleuth
```

## Monitoring and Alerting

### 1. Security Metrics

Track security trends over time:

```bash
# Extract metrics from audit reports
jq '.metrics.security_score' audit-report.json

# Store in time-series database
curl -X POST http://prometheus:9090/metrics/job/security \
  -d "security_score{branch=\"main\"} $(jq '.metrics.security_score' audit-report.json)"
```

### 2. Dashboard Integration

Create security dashboards:

```yaml
# Grafana dashboard query
SELECT 
  time,
  security_score,
  critical_issues,
  high_issues
FROM security_metrics
WHERE branch = 'main'
ORDER BY time DESC
```

### 3. Alert Rules

Set up alerts for security regressions:

```yaml
# Prometheus alert rule
groups:
  - name: security
    rules:
      - alert: SecurityScoreDropped
        expr: security_score < 80
        for: 5m
        annotations:
          summary: "Security score dropped below 80"
```

## Troubleshooting

### Common Issues

1. **Timeout errors**
   ```yaml
   # Increase timeout
   timeout: 30m
   ```

2. **Memory issues**
   ```yaml
   # Use larger runners
   resource_class: xlarge
   ```

3. **Cache problems**
   ```yaml
   # Clear cache
   - run: go clean -modcache
   ```

### Debug Mode

Enable debug output:

```bash
# Verbose audit output
audit -all -verbose -debug

# Trace execution
set -x
audit -all
```

## Integration Examples

### 1. Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Run quick security check
audit -code -config -max-critical 0

if [ $? -ne 0 ]; then
    echo "Security issues found. Commit blocked."
    exit 1
fi
```

### 2. Deployment Gate

```yaml
# Kubernetes admission webhook
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: strigoi-security
webhooks:
  - name: security.strigoi.io
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["apps"]
        apiVersions: ["v1"]
        resources: ["deployments"]
```

### 3. ChatOps Integration

```javascript
// Slack bot command
app.command('/security-audit', async ({ command, ack, respond }) => {
  await ack();
  
  const result = await runAudit(command.text);
  await respond({
    text: `Security Score: ${result.score}/100`,
    attachments: [{
      color: result.score > 80 ? 'good' : 'danger',
      fields: [{
        title: 'Critical Issues',
        value: result.critical_issues,
        short: true
      }]
    }]
  });
});
```

## Next Steps

1. Choose the CI/CD platform that best fits your workflow
2. Copy the appropriate configuration file to your repository
3. Customize thresholds and settings for your security requirements
4. Set up notifications and monitoring
5. Train your team on interpreting security reports

For more information, see:
- [Security Audit Documentation](./security_audit/README.md)
- [API Documentation](./api-documentation.md)
- [Performance Tuning Guide](./performance-tuning.md)