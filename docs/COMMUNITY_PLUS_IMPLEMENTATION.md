# Community+ Tier Implementation Guide

## Overview

The Community+ tier is designed for independent security researchers who want affordable access to professional tools while contributing valuable intelligence to the Strigoi ecosystem. This tier bridges the gap between free community access and commercial licensing.

## Target Audience

**Independent Security Researchers**:
- Personal lab environments
- Learning and experimentation
- Non-commercial use only
- Active in security research community
- Students and academics
- Independent consultants (for research, not client work)

## Pricing Model

- **Monthly**: $20/month
- **Annual**: $200/year (2 months free)
- **Student Discount**: $10/month with valid .edu email

## Key Features

### 1. Enhanced Intelligence Contribution
- **2x-5x contribution point multipliers**
- **Researcher attribution** (optional pseudonym)
- **Novel attack vectors**: 100-500 points
- **Zero-day discoveries**: 1,000 points
- **Published research integration**: 250 points

### 2. Marketplace Access
- **Enhanced tier access** (equivalent to 1,000+ contribution points)
- **Weekly updates** instead of monthly
- **Early access** to experimental modules
- **Beta testing privileges**
- **Researcher-exclusive threat feeds**

### 3. Community Recognition
- **Security Researcher badge** (Bronze/Silver/Gold/Platinum)
- **Monthly researcher spotlight**
- **Leaderboard visibility**
- **Direct communication channel** with Strigoi team
- **Priority feature requests**

### 4. Technical Benefits
- **Extended API rate limits**
- **Priority support queue**
- **Access to research datasets**
- **Custom module development kit**
- **Integration with academic tools**

## Verification Process

### Method 1: GitHub Verification
**Requirements**:
- Active GitHub account (6+ months old)
- At least 2 security-related repositories
- Regular commit activity
- Security-focused bio/profile

**Process**:
```bash
strigoi license verify --method github --username <github_username>
```

### Method 2: Academic Verification
**Requirements**:
- Valid .edu email address
- From recognized institution
- Academic profile/page (optional)

**Process**:
```bash
strigoi license verify --method academic --email <academic_email>
```

### Method 3: LinkedIn Verification
**Requirements**:
- LinkedIn profile showing:
  - "Security Researcher" or similar title
  - Independent/Freelance status
  - Security certifications (optional)
  - Published security content

**Process**:
```bash
strigoi license verify --method linkedin --profile <linkedin_url>
```

### Manual Review Process
For cases that don't fit standard verification:
1. Email researchers@macawi.ai with:
   - Brief bio (2-3 paragraphs)
   - Links to published research
   - Proof of independent status
   - Research interests/focus areas

## Intelligence Sharing Incentives

### Enhanced Scoring System
```
Standard Community Scoring:
- Basic submission: 10 points
- High-value pattern: 50 points
- New vulnerability: 100 points

Community+ Enhanced Scoring:
- Basic submission: 20-50 points (2-5x multiplier)
- Novel attack vector: 100-500 points
- Zero-day discovery: 1,000 points
- Vulnerability chain: 500 points
- Research integration: 250 points
- Tutorial creation: 200 points
```

### Monthly Competitions
- **Researcher of the Month**: Featured in newsletter
- **Most Novel Finding**: 500 bonus points
- **Best Tutorial**: Direct collaboration opportunity
- **Top Contributor**: Platinum badge upgrade

### Badge Progression
1. **Bronze** (Starting level)
   - 2x contribution multiplier
   - Basic researcher benefits

2. **Silver** (100+ contributions, 2+ novel findings)
   - 3x contribution multiplier
   - Monthly spotlight eligibility

3. **Gold** (500+ contributions, 5+ novel findings)
   - 4x contribution multiplier
   - Direct team communication

4. **Platinum** (1000+ contributions, 10+ novel findings)
   - 5x contribution multiplier
   - Co-development opportunities

## Implementation Details

### License Structure
```go
type CommunityPlusLicense struct {
    BaseFields
    ResearcherVerification ResearcherVerification
    EnhancedTracking      EnhancedContributionTracking
    RecognitionStatus     RecognitionStatus
}
```

### Verification API
```go
// Verify researcher credentials
verification, err := verifier.VerifyResearcher(ctx, "github", "security-researcher")

// Create Community+ license
license := &License{
    Type: LicenseTypeCommunityPlus,
    ResearcherVerification: verification,
    IntelSharingConfig: &IntelSharingConfig{
        ShareAttackPatterns: true,
        ShareVulnerabilities: true,
        AnonymizationLevel: AnonymizationStandard,
        MarketplaceAccessLevel: MarketplaceEnhanced,
    },
}
```

### Intelligence Attribution
```go
// Submit intelligence with researcher attribution
contribution := &ResearcherContribution{
    ResearcherID: license.ResearcherVerification.VerificationID,
    Type: "novel_attack",
    BasePoints: 100,
    Multiplier: verification.ContributionMultiplier,
    FinalPoints: 200,
    Description: "Novel MCP server authentication bypass",
    Impact: "high",
}
```

## Compliance and Restrictions

### Usage Restrictions
- **Non-commercial use only**
- **No client engagements**
- **No enterprise deployments**
- **Personal research only**

### Audit Requirements
- **Annual self-certification**
- **Random compliance checks**
- **Usage pattern monitoring**
- **Automatic commercial use detection**

### Violation Consequences
1. Warning (first violation)
2. Temporary suspension (second violation)
3. Permanent ban (third violation)
4. Migration to commercial license (if appropriate)

## Migration Paths

### From Community (Free)
- Keep all contribution points
- Immediate enhanced access
- Historical contributions recalculated with multiplier

### To Commercial
- Full history retention
- Seamless upgrade process
- Pro-rated billing
- Keep researcher badges

### To Enterprise
- Volume discount consideration
- Custom researcher program
- Team collaboration features

## Support and Resources

### Dedicated Resources
- **Slack Channel**: #community-plus-researchers
- **Monthly Webinars**: Advanced techniques
- **Research Dataset Access**: Anonymized attack data
- **Module Development Kit**: Create custom modules

### Priority Support
- **Response Time**: 24-48 hours
- **Direct Escalation**: To senior team
- **Feature Requests**: Quarterly review
- **Beta Access**: Automatic enrollment

## Success Metrics

### For Researchers
- Skill development through real-world data
- Community recognition and networking
- Portfolio building with attributed findings
- Direct impact on security ecosystem

### For Strigoi
- Rich intelligence from motivated researchers
- Novel attack vector discovery
- Community-driven innovation
- Expanded security coverage

## Implementation Timeline

### Phase 1 (Month 1)
- GitHub verification system
- Basic contribution tracking
- Enhanced marketplace access

### Phase 2 (Month 2)
- Academic email verification
- Badge system implementation
- Researcher leaderboard

### Phase 3 (Month 3)
- LinkedIn verification
- Monthly competitions
- Exclusive content channels

### Phase 4 (Month 4+)
- Advanced analytics for researchers
- Collaborative research projects
- Integration with academic institutions

## FAQ

### Q: Can I use Community+ for bug bounties?
A: No, bug bounty hunting is considered commercial activity. Use the commercial license.

### Q: What if I graduate and get a job?
A: Congratulations! You'll need to switch to a commercial license for work use, but can keep Community+ for personal research.

### Q: Can I share my findings publicly?
A: Yes! We encourage responsible disclosure and academic publication. Attribution is optional.

### Q: How do you prevent commercial abuse?
A: Usage patterns, deployment scale, and intelligence contribution patterns help identify commercial use.

### Q: Can academic institutions get group Community+ licenses?
A: Yes, contact academics@macawi.ai for educational institution pricing.

---

*The Community+ tier represents our commitment to nurturing the next generation of security researchers while building the world's most comprehensive agentic security intelligence database.*