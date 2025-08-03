# Git-Powered Laboratory Workflow
## Version Control as Scientific Record

*"Every commit is a signed lab notebook entry"*

---

## Git Structure for Testing

### Branch Strategy
```bash
main                           # Stable, tested modules only
├── protocol/mcp              # MCP protocol development
│   ├── test/mcp-injection-001  # Specific test development
│   ├── test/mcp-injection-002  # Unicode bypass discovery
│   └── test/mcp-stress-001     # Stress testing branch
├── protocol/agntcy           # AGNTCY protocol work
├── protocol/openai           # OpenAI Assistants
└── lab/weekly-report-2025-w29  # Weekly summary branches
```

### Commit Message Format
```bash
# Test Development Commits
git commit -m "test(mcp): Implement Unicode normalization bypass test

Hypothesis: MCP fails to normalize Unicode before parsing tool names
Method: Progressive Unicode injection with various normalization forms
Result: Successful bypass using U+200B (zero-width space)
Evidence: ./evidence/2025-07-21-001/

Severity: High
Affects: MCP v1.0, v1.1-beta

Lab-Entry: SLN-2025-07-21-001
Co-discovered-by: Synth"

# Discovery Commits  
git commit -m "discovery(mcp): Critical vulnerability in tool parsing

CVE: Pending
CVSS: 8.1 (High)
Vector: Network-based prompt injection via Unicode bypass

Discovered during routine injection testing.
Reproducible in 10/10 attempts.
Affects all deployments using standard MCP server.

Evidence: ./evidence/2025-07-21-001/
PoC: ./exploits/mcp/unicode_tool_override.ts

Lab-Entry: SLN-2025-07-21-001"
```

---

## Git Tags for Milestones

### Discovery Tags
```bash
# Tag significant discoveries
git tag -a "vuln/mcp-unicode-bypass-v1.0" -m "First Unicode bypass in agent protocol
Discovered: 2025-07-21
Severity: High  
Researcher: Cy + Synth
Lab Entry: SLN-2025-07-21-001"

# Tag test suite releases
git tag -a "suite/mcp-complete-v1.0" -m "Complete MCP test suite
Tests: 47
Coverage: 100% of documented functions  
Findings: 3 Critical, 5 High, 12 Medium"

# Tag stable milestones
git tag -a "stable/week-29-2025" -m "Week 29 stable test suite
Protocols: MCP, AGNTCY (partial)
Total tests: 89
Success rate: 34%"
```

---

## Pull Requests as Peer Review

### Test Development PR Template
```markdown
## Test Development: [Protocol] - [Test Name]

### Hypothesis
What vulnerability are we testing for?

### Methodology  
How are we testing it?

### Results
- [ ] Test implemented
- [ ] Evidence collected
- [ ] Documentation complete
- [ ] Peer reviewed

### Evidence
- Test output: [link]
- Packet captures: [link]
- Screenshots: [link]

### Knowledge Contribution
What did we learn that applies beyond this test?

---
**Lab Entry**: SLN-YYYY-MM-DD-XXX
**Reviewers**: @synth @cy
```

### Discovery PR Template
```markdown
## 🚨 Vulnerability Discovery: [Protocol] - [Vulnerability Name]

### Summary
Brief description of the vulnerability

### Severity
- [ ] Critical (9.0-10.0)
- [ ] High (7.0-8.9)
- [ ] Medium (4.0-6.9)
- [ ] Low (0.1-3.9)

### Evidence
- PoC: [link to exploit]
- Test results: [link]
- Reproduction steps: [link]

### Disclosure Timeline
- Discovery: YYYY-MM-DD
- Vendor notified: YYYY-MM-DD
- Patch available: TBD
- Public disclosure: TBD

---
**Requires Security Review Before Merge**
```

---

## GitHub Issues for Test Tracking

### Test Planning Issues
```markdown
# Title: Test Campaign: MCP Protocol Full Coverage

## Objective
Complete security assessment of MCP v1.0 protocol

## Test Categories
- [ ] Authentication (5 tests)
- [ ] Tool invocation (12 tests)  
- [ ] State management (8 tests)
- [ ] Error handling (6 tests)
- [ ] Performance limits (4 tests)

## Timeline
Start: 2025-07-21
Target: 2025-07-25

## Success Criteria
- 100% function coverage
- All findings documented
- Executive report ready

Labels: `protocol:mcp`, `test-campaign`, `priority:high`
```

### Vulnerability Tracking
```markdown
# Title: [MCP] Unicode Normalization Bypass in Tool Parsing

## Description
MCP server fails to normalize Unicode input before parsing tool names

## Severity
High (CVSS 8.1)

## Status
- [x] Discovered
- [x] Reproduced
- [x] PoC developed
- [ ] Vendor notified
- [ ] CVE requested
- [ ] Patch verified
- [ ] Public disclosure

## References
- Lab Entry: SLN-2025-07-21-001
- PR: #123
- Evidence: ./evidence/2025-07-21-001/

Labels: `vulnerability`, `protocol:mcp`, `severity:high`
```

---

## GitHub Projects for Test Campaigns

### Kanban Board Structure
```
📋 Backlog | 🔬 In Testing | 📊 Analysis | ✅ Complete | 📚 Published

Cards represent individual tests or test suites
Move through pipeline as work progresses
Automated via GitHub Actions
```

---

## Release Process

### Test Suite Releases
```bash
# Create release branch
git checkout -b release/mcp-test-suite-v1.0

# Update version
echo "1.0.0" > VERSION

# Generate test manifest
strigoi generate-manifest --protocol mcp > releases/mcp-v1.0-manifest.json

# Create release notes
strigoi generate-release-notes --protocol mcp > RELEASE_NOTES.md

# Tag and release
git tag -a v1.0.0-mcp -m "MCP Test Suite v1.0.0

Tests included: 47
Vulnerabilities found: 20
Coverage: 100%

This release includes:
- Complete function coverage
- All injection types
- Stress tests
- Performance tests

Suitable for:
- Security assessments
- Compliance audits  
- Pre-production validation"

# Push release
git push origin v1.0.0-mcp
```

### GitHub Release Automation
```yaml
# .github/workflows/test-release.yml
name: Test Suite Release

on:
  push:
    tags:
      - 'v*.*.*-*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Generate Test Report
        run: |
          strigoi report test-suite \
            --format pdf \
            --output test-suite-report.pdf
            
      - name: Generate Evidence Archive
        run: |
          tar -czf evidence-archive.tar.gz ./evidence/
          
      - name: Create Release
        uses: actions/create-release@v1
        with:
          tag_name: ${{ github.ref }}
          release_name: Test Suite ${{ github.ref }}
          body_path: RELEASE_NOTES.md
          
      - name: Upload Assets
        uses: actions/upload-release-asset@v1
        with:
          asset_path: test-suite-report.pdf
          asset_name: test-suite-report.pdf
```

---

## Signing and Verification

### GPG Signing for Test Integrity
```bash
# Sign critical test results
gpg --sign evidence/2025-07-21-001/test-results.json

# Sign lab notebook entries
git commit -S -m "test(mcp): Critical vulnerability discovered"

# Verify test suite integrity
strigoi verify-suite --signature mcp-v1.0.sig
```

### Blockchain Timestamping (Optional)
```typescript
// For critical discoveries, timestamp on blockchain
const discovery = {
  protocol: "MCP",
  vulnerability: "Unicode Bypass",
  hash: sha256(evidenceFiles),
  timestamp: Date.now()
};

await blockchain.timestamp(discovery);
```

---

## Benefits of Git Laboratory

1. **Immutable History**: Can't alter test records after the fact
2. **Peer Review**: PRs ensure test quality
3. **Traceability**: Every finding traced to specific commit
4. **Reproducibility**: Clone and re-run any test
5. **Collaboration**: Distributed testing with merge capabilities
6. **Automation**: CI/CD for test execution and reporting

---

*"Your grandfather's lab notebooks, but with cryptographic signatures and global collaboration"*