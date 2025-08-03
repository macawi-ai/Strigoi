# Validation Telemetry Transparency Disclosure

**Effective Date**: July 2025  
**Version**: 1.0  
**Last Updated**: July 2025

## Executive Summary

This document provides complete transparency regarding the validation telemetry system included in Strigoi. We believe users have the right to understand exactly what data is collected and how it is used.

## Purpose of Telemetry

The validation telemetry serves ONE purpose:
- **Legal record keeping for license compliance**

The telemetry is NOT used for:
- ❌ Marketing or sales activities
- ❌ Individual identification or tracking
- ❌ Customer solicitation
- ❌ Behavioral profiling
- ❌ Data sale or sharing
- ❌ Product usage analytics beyond compliance

## Exactly What Is Transmitted

Each validation event creates a DNS query with the following structure:

```
{version}.{hash}.{timestamp}.{action}.validation.macawi.io
```

### Components:

1. **Version** (e.g., "v1")
   - Protocol version number
   - Allows for future compatibility

2. **Hash** (e.g., "a7b9c2")
   - First 6 characters of SHA256(license_key + timestamp)
   - Cannot be reversed to identify license holder
   - Changes with each query

3. **Timestamp** (e.g., "1737389400")
   - Unix timestamp of the event
   - Used for temporal correlation
   - No timezone information

4. **Action** (one of: start|test|complete|error)
   - Generic action indicator
   - No details about specific operations

### Complete Data Example:
```
v1.a7b9c2.1737389400.start.validation.macawi.io
```

## What Is NOT Transmitted

- ❌ IP addresses of tested systems
- ❌ Hostnames or domain names being tested
- ❌ Test parameters or configurations
- ❌ Test results or findings
- ❌ User credentials or authentication tokens
- ❌ Network traffic or packet data
- ❌ File paths or system information
- ❌ Geographic location data
- ❌ Hardware or software fingerprints
- ❌ Any personally identifiable information (PII)

## Data Retention and Use

1. **Retention Period**: DNS query logs retained for 12 months
2. **Access**: Limited to legal compliance verification only
3. **Storage**: Secure servers with encryption at rest
4. **Deletion**: Automatic after retention period
5. **Audit**: Annual third-party privacy audit

## Privacy Regulation Compliance

Macawi adheres to:

### European Union
- ✅ GDPR (General Data Protection Regulation)
- ✅ ePrivacy Directive
- ✅ No personal data collected requiring consent
- ✅ Privacy by Design principles

### United States
- ✅ CCPA (California Consumer Privacy Act)
- ✅ State privacy laws
- ✅ No sale of personal information
- ✅ No behavioral tracking

### Security Standards
- ✅ SOC 2 Type II principles
- ✅ ISO 27001 aligned practices
- ✅ Encryption in transit and at rest
- ✅ Access controls and audit logs

## Your Rights

Users have the right to:
1. **Transparency**: This complete disclosure
2. **Verification**: Confirm telemetry functionality
3. **Compliance**: Use tool per license terms
4. **Questions**: Contact us for clarification

## Legal Basis

The validation telemetry is:
- Required for license enforcement (contractual necessity)
- Disclosed transparently before use
- Minimal in scope and impact
- Necessary for intellectual property protection

## Enforcement

Removal or circumvention of telemetry:
- Violates license terms
- Voids any support agreements
- May result in legal action
- Creates liability for the user

## Contact Information

For privacy or telemetry questions:

**Privacy Officer**  
James R. Saker Jr.  
Macawi  
Email: privacy@macawi.ai  
LinkedIn: https://www.linkedin.com/in/jamessaker/

## Certification

I certify that this disclosure is complete and accurate:

**James R. Saker Jr.**  
*Chief Information Security Officer*  
*Macawi*

---

This document is version controlled and updates will be provided with new software releases.