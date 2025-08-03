# Strigoi Licensing and Threat Intelligence System

## Overview

Strigoi implements an innovative dual-licensing model: **"Pay with Money or Pay with Intelligence"**. This creates a virtuous cycle where community participation strengthens the entire ecosystem while maintaining strict privacy and compliance standards.

## License Types

### 1. Commercial License ($5,000/instance/year)
- **Full Privacy**: No mandatory data sharing
- **Unlimited Access**: All marketplace modules and updates
- **Priority Support**: Direct support channels
- **Flexible Deployment**: On-premises or cloud
- **Optional Intel Sharing**: Can contribute for community benefit

### 2. Community License (Free)
- **Intelligence Contribution**: Mandatory anonymized threat intel sharing
- **Marketplace Access**: Based on contribution level
- **Privacy-Preserving**: Advanced anonymization ensures compliance
- **Community Support**: Forums and documentation

### 3. Community+ License ($20/month)
- **Target Audience**: Independent security researchers and students
- **Intelligence Contribution**: Enhanced intel sharing with researcher attribution
- **Marketplace Access**: Standard level (equivalent to 1,000+ contribution points)
- **Special Benefits**:
  - "Security Researcher" badge in community
  - Early access to experimental modules
  - Direct communication channel with Strigoi team
  - Monthly researcher spotlight opportunities
  - Access to researcher-exclusive threat feeds
  - Beta testing privileges for new features
- **Verification Requirements**:
  - Active GitHub profile with security-related repositories
  - OR Academic email from recognized institution
  - OR LinkedIn profile showing independent researcher status
  - Annual self-certification of non-commercial use
- **Intelligence Incentives**:
  - 2x contribution points for novel attack patterns
  - 5x points for zero-day vulnerability discoveries
  - Monthly leaderboard with special recognition
  - "Researcher of the Month" highlight in newsletter

### 4. Trial License (30 days)
- **Limited Features**: Basic scanning and reporting
- **No Intel Sharing**: Privacy during evaluation
- **Upgrade Path**: Easy conversion to commercial or community

### 5. Enterprise License (Custom)
- **Volume Pricing**: Discounts for multiple instances
- **Custom Features**: Tailored to enterprise needs
- **Dedicated Support**: SLAs and priority fixes
- **Compliance Options**: Custom anonymization policies

## Intelligence Sharing Framework

### What We Collect (Community License)

#### Attack Patterns
- Pattern signatures (hashed)
- Success/detection rates
- Target types (categorized)
- Techniques used (MITRE ATT&CK mapped)

#### Vulnerability Intelligence
- Vulnerability categories
- Severity distributions
- Prevalence scores
- Patch availability status

#### Configuration Insights
- Common misconfigurations (anonymized)
- Security score distributions
- Best practice adoption rates

#### Usage Statistics
- Feature usage patterns
- Performance metrics
- Error rates (categorized)

### What We NEVER Collect
- ❌ Personal identifiable information (PII)
- ❌ Internal IP addresses or hostnames
- ❌ Actual vulnerability details
- ❌ Customer/employee identifiers
- ❌ Sensitive business data
- ❌ Authentication credentials
- ❌ Actual attack payloads

## Anonymization Levels

### 1. Minimal (Basic compliance)
- Direct PII removal (emails, phones, SSNs)
- Credit card and financial data
- Basic health identifiers

### 2. Standard (Default for community)
- Everything from Minimal
- Internal network information
- UUIDs and system identifiers
- Employee/customer IDs

### 3. Strict (Enhanced privacy)
- Everything from Standard
- API keys and tokens
- Geographic locations
- Potential identifiers

### 4. Paranoid (Maximum anonymization)
- Everything from Strict
- Aggressive tokenization
- Context removal
- Statistical noise addition

## Compliance Framework

### Supported Regulations

#### GDPR (EU General Data Protection Regulation)
- Right to erasure support
- Data minimization
- Purpose limitation
- Explicit consent mechanisms

#### HIPAA (Health Insurance Portability and Accountability Act)
- PHI detection and removal
- Minimum necessary standard
- Audit trail maintenance

#### PCI-DSS (Payment Card Industry Data Security Standard)
- Complete card data removal
- Tokenization of identifiers
- Secure data handling

#### CCPA/CPRA (California Privacy Laws)
- Consumer rights support
- Opt-out mechanisms
- Data deletion capabilities

#### Additional Frameworks
- PIPEDA (Canada)
- LGPD (Brazil)
- POPI (South Africa)
- PIPA (South Korea)

## Marketplace Integration

### Access Levels

#### 1. None
- No marketplace access
- Manual module updates only

#### 2. Basic (100+ contribution points)
- Community modules
- Monthly updates
- Basic threat feeds

#### 3. Standard (1,000+ points)
- All community content
- Weekly updates
- Enhanced threat intelligence

#### 4. Premium (10,000+ points)
- Early access to modules
- Daily updates
- Real-time threat feeds
- Beta features

#### 5. Enhanced (Community+ license)
- All community content
- Weekly updates
- Enhanced threat intelligence
- Researcher-exclusive feeds
- Beta access

#### 6. Unlimited (Commercial license)
- Everything available
- Priority access
- Custom modules
- Direct support

### Contribution Scoring

#### Standard Scoring
- Each intelligence submission: 10 points
- High-value patterns: 50 points
- New vulnerability types: 100 points
- Community modules: 500 points

#### Community+ Bonus Scoring (2x-5x multipliers)
- Novel attack vectors: 100-500 points
- Zero-day discoveries: 1,000 points
- Published research integration: 250 points
- Validated vulnerability chains: 500 points
- Community tutorial creation: 200 points

## Technical Implementation

### License Validation Flow
```
1. Check local cache (24-hour validity)
2. Online validation with license server
3. Fallback to offline validation (7-day grace)
4. Initialize appropriate components
5. Start telemetry and intelligence collection
```

### Intelligence Collection Pipeline
```
1. Module execution captures data
2. Anonymizer scrubs sensitive information
3. Data buffered locally (100 items or 5 minutes)
4. Batch submission via GitHub infrastructure
5. Contribution points credited
6. Marketplace access updated
```

### GitHub Integration Strategy

#### Option 1: Workflow Dispatch (Preferred)
- Trigger GitHub Actions via API
- Process intelligence asynchronously
- Scale with GitHub's infrastructure

#### Option 2: Issue Creation
- Create issues with encoded data
- Automated processing via bots
- Public transparency option

#### Option 3: Branch Push
- Direct push to intelligence branch
- Requires authentication token
- Real-time processing

#### Option 4: Discussions API
- Community-visible contributions
- Threaded intelligence sharing
- GraphQL-based queries

## Privacy Protection Mechanisms

### Tokenization System
- Reversible tokens for authorized parties
- One-way hashes for permanent anonymization
- Consistent tokenization within sessions
- Secure token storage

### Data Scrubbing Pipeline
1. **Pattern Detection**: Regex-based identification
2. **Context Analysis**: Determine data sensitivity
3. **Replacement Strategy**: Token, hash, or redact
4. **Validation**: Ensure no leakage
5. **Audit Trail**: Log anonymization actions

### Compliance Filters
- Policy-specific data removal
- Geo-restriction enforcement
- Retention period management
- Cross-border transfer controls

## Security Considerations

### License Security
- SHA-256 hashed storage
- Time-limited validation tokens
- Instance binding (optional)
- Tamper detection

### Intelligence Security
- End-to-end encryption in transit
- Anonymization before transmission
- No correlation between submissions
- Rate limiting and abuse prevention

### Infrastructure Security
- DNS-over-HTTPS for telemetry
- Certificate pinning
- API authentication
- Audit logging

## Best Practices

### For Users
1. Choose appropriate anonymization level
2. Review intelligence before submission
3. Configure compliance policies correctly
4. Monitor contribution statistics
5. Keep license cache secure

### For Integrators
1. Never modify anonymization code
2. Respect telemetry requirements
3. Handle licenses securely
4. Implement proper error handling
5. Follow contribution guidelines

## Frequently Asked Questions

### Q: Can I see what data is being shared?
A: Yes, enable debug mode to log all anonymized data before submission.

### Q: What happens if GitHub is unavailable?
A: Intelligence is buffered locally and retried. Marketplace access continues with cached data.

### Q: Can I contribute without running scans?
A: No, contributions must come from actual Strigoi usage to ensure data quality.

### Q: How do I upgrade from Community to Commercial?
A: Contact sales@macawi.ai with your instance ID for seamless migration.

### Q: How do I qualify for Community+ as a researcher?
A: Apply at researchers@macawi.ai with your GitHub profile, academic credentials, or LinkedIn. We verify within 48 hours.

### Q: Can I switch between Community and Community+?
A: Yes, you can upgrade anytime and downgrade at the end of your billing cycle. Contribution points carry over.

### Q: Is the anonymization reversible?
A: Only with explicit permission and the original token mapping.

## Implementation Status

### Completed
- [x] Core licensing types and structures
- [x] Comprehensive anonymization engine
- [x] Intelligence collection framework
- [x] GitHub sync mechanisms
- [x] Compliance policy system

### In Progress
- [ ] License validation server
- [ ] Marketplace frontend
- [ ] Contribution dashboard
- [ ] Enterprise features

### Planned
- [ ] Advanced ML-based anonymization
- [ ] Real-time threat correlation
- [ ] Automated compliance validation
- [ ] Multi-tenant support

## Contact

For licensing inquiries:
- Email: licensing@macawi.ai
- Phone: +1-555-0123
- Support: https://support.macawi.ai

For security concerns:
- Security: security@macawi.ai
- Bug Bounty: https://hackerone.com/macawi

---

*Last Updated: January 2025*
*Version: 1.0.0*