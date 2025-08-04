# Strigoi Development Roadmap

## Overview
This document tracks planned features, enhancements, and architectural improvements for Strigoi.

---

## Current Version: v0.5.0
- ✅ Core REPL framework
- ✅ Modular probe architecture (North/South/East/West)
- ✅ MCP security scanning
- ✅ Secure command execution
- ✅ Enhanced process fingerprinting (Phase 1)

---

## Upcoming Features

### v0.6.0 - Session Management
**Timeline**: Q1 2025  
**Status**: In Development

- Core session encryption and storage
- Session templates and playbooks
- See [SESSION_ROADMAP.md](docs/SESSION_ROADMAP.md) for details

### v0.7.0 - Privileged Mode & Enhanced Security
**Timeline**: Q2 2025  
**Status**: Planning

#### Privileged Mode Implementation
- **Elevated Permission Features**:
  - Full network exposure analysis (requires root/CAP_NET_ADMIN)
  - Deep process inspection (/proc access)
  - System-wide dependency scanning
  - Kernel module detection
  - Raw socket analysis
  
- **Security Requirements**:
  - Explicit user consent for privileged operations
  - Audit logging of all privileged actions
  - Privilege separation architecture
  - Temporary privilege escalation
  - Secure credential handling
  
- **Implementation Details**:
  - `--privileged` flag for explicit activation
  - Capability-based permissions (Linux capabilities)
  - Fallback graceful degradation
  - Clear indication of privileged mode in output

#### GraphQL Security Scanner
- Introspection vulnerability detection
- Query complexity analysis
- JWT security assessment
- Authorization bypass detection
- See [GraphQL Scanner Design](docs/GraphQL-Scanner-Technical-Design.md)

### v0.8.0 - Enterprise Features
**Timeline**: Q3 2025  
**Status**: Future

- **CI/CD Integration**:
  - GitHub Actions support
  - GitLab CI templates
  - Jenkins plugins
  - Automated security gates
  
- **Reporting & Compliance**:
  - SARIF output format
  - SBOM generation
  - Compliance mapping (SOC2, ISO 27001)
  - Executive dashboards
  
- **Team Collaboration**:
  - Centralized findings database
  - Role-based access control
  - Audit trails
  - Policy management

### v0.9.0 - Advanced Analysis
**Timeline**: Q4 2025  
**Status**: Concept

- **AI-Powered Analysis**:
  - Vulnerability pattern learning
  - False positive reduction
  - Automated remediation suggestions
  - Threat intelligence integration
  
- **Stream Processing** (Stream Tap Module):
  - Real-time security monitoring
  - Event stream analysis
  - Anomaly detection
  - See [Stream Infrastructure](docs/architecture/stream-infrastructure.md)
  
- **External Tool Integration**:
  - Semgrep integration
  - Trivy scanner support
  - Custom rule engines
  - Third-party API support

### v1.0.0 - Production Ready
**Timeline**: 2026  
**Status**: Vision

- Performance optimizations
- Comprehensive documentation
- Enterprise support options
- Security certifications
- Stable API guarantees

---

## Feature Backlog

### High Priority
- [ ] Priority 1 Security Rules (HTTP transport, default passwords)
- [ ] System-level dependency scanning enhancement
- [ ] Comprehensive test coverage for probe modules
- [ ] Container/Docker security scanning
- [ ] Kubernetes admission controller

### Medium Priority
- [ ] Web UI for result visualization
- [ ] Historical trend analysis
- [ ] Custom rule builder GUI
- [ ] Integration with bug bounty platforms
- [ ] Distributed scanning architecture

### Low Priority
- [ ] Mobile app security scanning
- [ ] Hardware security module support
- [ ] Blockchain smart contract analysis
- [ ] IoT device security assessment

---

## Technical Debt

### Code Quality
- [ ] Increase test coverage to 80%+
- [ ] Standardize error handling patterns
- [ ] Improve logging consistency
- [ ] Performance profiling and optimization

### Architecture
- [ ] Plugin system for custom modules
- [ ] gRPC API for remote operations
- [ ] Metrics and observability (OpenTelemetry)
- [ ] Database backend for large-scale deployments

### Documentation
- [ ] API reference generation
- [ ] Video tutorials
- [ ] Integration cookbooks
- [ ] Security best practices guide

---

## Community Requests

### From Prismatic.io Engagement
- [x] GraphQL security scanning
- [x] Node.js integration wrapper
- [ ] Webhook notifications
- [ ] Custom component marketplace

### From Security Community
- [ ] OWASP Top 10 coverage
- [ ] CIS benchmark scanning
- [ ] Supply chain security (SLSA)
- [ ] Container registry scanning

---

## Research & Innovation

### Experimental Features
- Quantum-resistant cryptography
- Homomorphic encryption for sensitive scans
- Federated learning for threat detection
- Zero-knowledge proof validations

### Academic Collaboration
- University partnerships for security research
- Student project opportunities
- Research paper publications
- Conference presentations

---

## Success Metrics

### Technical KPIs
- Scan performance: <1 minute for average project
- False positive rate: <5%
- Memory usage: <500MB for standard scans
- Vulnerability detection rate: >90% of known CVEs

### Community KPIs
- GitHub stars: 1000+ by v1.0
- Active contributors: 50+
- Enterprise adopters: 10+
- Security findings reported: 1000+

---

## Release Cadence

- **Major versions** (x.0.0): Annual
- **Minor versions** (0.x.0): Quarterly
- **Patch versions** (0.0.x): As needed for security/bugs
- **Release candidates**: 2 weeks before minor/major releases

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Feature request process
- Development guidelines
- Security disclosure policy
- Code of conduct

---

*This roadmap is a living document and subject to change based on community needs and security landscape evolution.*

**Last Updated**: August 2025  
**Next Review**: September 2025  
**Maintainer**: Strigoi Core Team