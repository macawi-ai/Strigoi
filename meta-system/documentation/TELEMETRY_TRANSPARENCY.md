# Strigoi Telemetry Transparency Statement

## What We Collect

Strigoi includes a lightweight validation system that helps us:
1. Ensure license compliance
2. Improve the product based on usage patterns
3. Detect potential security issues
4. Provide better support

## How It Works

When you use Strigoi, it makes DNS queries to `validation.macawi.io` containing:
- Protocol version (e.g., "v1")
- Anonymized instance identifier (6-character hash)
- Timestamp of the action
- Action type (start, test, complete, error)

Example: `v1.a7b9c2.1737389400.start.validation.macawi.io`

## What We DON'T Collect

- ❌ No personal information
- ❌ No target systems or IP addresses
- ❌ No test results or findings
- ❌ No sensitive security data
- ❌ No network traffic content
- ❌ No authentication credentials

## Why This Matters

As security professionals, we believe in transparency. This telemetry helps us:
- Know if our tool is being helpful
- Identify common use patterns for improvement
- Ensure the tool isn't being misused
- Maintain license compliance

## Your Privacy

- All telemetry is anonymous
- Data is used only for product improvement
- We never sell or share usage data
- DNS queries are lightweight and fast
- No impact on tool performance

## Legal Requirement

Per our license terms, this telemetry system:
- Must remain intact and functional
- Cannot be removed or modified
- Is required for valid licensing
- Helps protect against unauthorized use

## Questions?

If you have concerns about telemetry:
- Email: jamie.saker@macawi.ai
- We're happy to discuss enterprise options

## Technical Details

For implementation details, see: [VALIDATION_TELEMETRY_DESIGN.md](VALIDATION_TELEMETRY_DESIGN.md)

---

*Last Updated: July 2025*  
*We believe in building trust through transparency.*