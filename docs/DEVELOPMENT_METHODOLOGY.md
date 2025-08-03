# Strigoi Development Methodology

## Overview
This document defines the development methodology for Strigoi v0.5.0 and beyond, based on industry best practices for building reliable security tools.

## 1. Development Approach

### Agile Framework: Kanban
- **Why**: Single developer flexibility with ability to scale
- **Implementation**:
  - GitHub Project board with columns: Backlog → In Progress → Review → Done
  - Work-in-progress (WIP) limits: Max 3 items in progress
  - Weekly reviews of board status

### Issue Tracking
- **Tool**: GitHub Issues
- **Labels**:
  - `bug`: Something isn't working
  - `feature`: New feature or request
  - `security`: Security-related changes
  - `documentation`: Documentation improvements
  - `enhancement`: Improvement to existing feature
  - `good-first-issue`: Good for newcomers

### Git Workflow: GitHub Flow
```
main (stable)
  └── feature/implement-real-probe
  └── fix/tab-completion-bug
  └── docs/update-readme
```

**Process**:
1. Create issue describing work
2. Create feature branch from main
3. Make changes with clear commits
4. Open PR with issue reference
5. Code review (self-review initially)
6. Merge to main
7. Delete feature branch

## 2. Documentation Standards

### Required Documentation

#### Code-Level
```go
// Package probe implements network discovery and reconnaissance
// functionality for the Strigoi security validation platform.
package probe

// DiscoverEndpoints probes the target for exposed API endpoints
// using various discovery techniques including:
//   - Common path enumeration
//   - OpenAPI/Swagger detection
//   - Directory bruteforcing (if enabled)
//
// Returns a slice of discovered endpoints and any errors encountered.
func DiscoverEndpoints(target string, opts *ProbeOptions) ([]Endpoint, error) {
    // Implementation...
}
```

#### Project-Level
- `README.md`: Installation, quick start, overview
- `docs/ARCHITECTURE.md`: System design and components
- `docs/API.md`: Public API documentation
- `docs/SECURITY.md`: Security considerations
- `docs/CONTRIBUTING.md`: How to contribute

### Documentation Tools
- **godoc**: For Go package documentation
- **Mermaid**: For architecture diagrams in Markdown
- **asciinema**: For terminal session recordings

## 3. Testing Strategy

### Test Pyramid
```
        /\
       /E2E\      (5%)  - Full security scenarios
      /------\
     /  Integ \   (15%) - Component interactions
    /----------\
   /    Unit    \ (80%) - Individual functions
  /--------------\
```

### Test Structure
```
test/
├── unit/
│   ├── probe_test.go
│   └── stream_test.go
├── integration/
│   ├── mcp_discovery_test.go
│   └── module_loading_test.go
└── e2e/
    ├── full_scan_test.go
    └── scenarios/
```

### Testing Requirements
- Minimum 80% unit test coverage
- All new features must include tests
- Integration tests for component boundaries
- E2E tests for critical user workflows

### Test Commands
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific test suite
go test ./test/unit/...

# Run with race detector
go test -race ./...
```

## 4. Code Quality Standards

### Pre-commit Checks
```yaml
# .github/pre-commit.yaml
- gofmt: Format code
- golint: Lint code
- go vet: Static analysis
- go test: Run tests
- security: Run gosec
```

### Code Review Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] No security vulnerabilities
- [ ] Follows Go conventions
- [ ] Error handling appropriate
- [ ] Logging sufficient
- [ ] Performance considered

## 5. Release Process

### Version Numbering (SemVer)
- **MAJOR.MINOR.PATCH** (e.g., 0.5.0)
- **MAJOR**: Breaking changes
- **MINOR**: New features (backwards compatible)
- **PATCH**: Bug fixes

### Release Workflow
1. Create release branch: `release/v0.5.0`
2. Update CHANGELOG.md
3. Update version in code
4. Create release PR
5. After merge, tag: `git tag v0.5.0`
6. Build release artifacts
7. Create GitHub release with notes

### Release Checklist
- [ ] All tests passing
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Security scan clean
- [ ] Performance benchmarks acceptable
- [ ] Release notes written

## 6. Development Environment

### Required Tools
```bash
# Go development
go version  # 1.21+
golangci-lint version
gosec version

# Git hooks
pre-commit install

# Documentation
godoc -http=:6060

# Testing
gotestsum  # Better test output
```

### IDE Setup
- VSCode with Go extension
- GoLand
- vim with vim-go

### Debugging
```bash
# Debug REPL
dlv debug ./cmd/strigoi

# Debug specific test
dlv test ./test/unit -- -test.run TestProbeNorth
```

## 7. Continuous Integration

### GitHub Actions Workflow
```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: make test
      - run: make lint
      - run: make security-scan
```

### CI Requirements
- All tests must pass
- Code coverage > 80%
- No linting errors
- Security scan clean
- Build successful

## 8. Security Practices

### Security-First Development
- **Input validation**: All user input sanitized
- **Error handling**: No sensitive info in errors
- **Logging**: No credentials in logs
- **Dependencies**: Regular vulnerability scans
- **Code review**: Security-focused reviews

### Security Tools
```bash
# Vulnerability scanning
gosec ./...

# Dependency check
go list -m all | nancy sleuth

# License compliance
go-licenses check ./...
```

## 9. Performance Guidelines

### Benchmarking
```go
func BenchmarkProbeEndpoints(b *testing.B) {
    for i := 0; i < b.N; i++ {
        DiscoverEndpoints("localhost", DefaultOptions())
    }
}
```

### Performance Targets
- REPL response: < 100ms
- Probe scan: < 5s for 100 endpoints
- Memory usage: < 100MB for typical session

## 10. Communication

### Channels
- **GitHub Issues**: Feature requests, bugs
- **GitHub Discussions**: Design decisions
- **Pull Requests**: Code reviews

### Decision Records
- Major decisions documented in `docs/decisions/`
- Format: ADR (Architecture Decision Records)

## Getting Started

1. Fork the repository
2. Clone your fork
3. Install pre-commit hooks: `pre-commit install`
4. Create feature branch
5. Make changes with tests
6. Submit PR

## Maintenance

### Weekly Tasks
- Review open issues
- Update dependencies
- Check security advisories
- Review PR queue

### Monthly Tasks
- Performance benchmark review
- Documentation audit
- Dependency updates
- Security scan

This methodology ensures Strigoi remains a reliable, secure, and maintainable security validation platform.