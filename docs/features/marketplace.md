# Strigoi Module Marketplace

## Overview

The Strigoi Module Marketplace provides a secure, cryptographically-verified distribution system for security modules. All modules are SHA-256 verified and third-party modules require explicit user consent.

## Architecture

### Hub-and-Spoke Model
- **Central Hub**: GitHub repository at `https://github.com/macawi-ai/marketplace` (to be created)
- **Metadata Distribution**: YAML manifests with full provenance tracking
- **Binary Distribution**: Separate CDN/storage for module packages
- **Trust Levels**: Official vs Community namespaces

### Security Features
- **SHA-256 Verification**: Every module download is cryptographically verified
- **Provenance Tracking**: Complete pipeline history from request to release
- **Trust Warnings**: Interactive consent for third-party modules
- **Namespace Separation**: Clear distinction between official and community modules

## Console Commands

### Search for Modules
```
strigoi> marketplace search <query>
```
Searches both official and community repositories for modules matching the query.

### Install a Module
```
strigoi> marketplace install <module>[@version]
```
Examples:
- `marketplace install mcp/sudo-tailgate` - Install official module
- `marketplace install johnsmith/custom-scanner@1.0.0` - Install specific version

### List Installed Modules
```
strigoi> marketplace list
```
Shows all modules installed from the marketplace, grouped by trust level.

### Update Cache
```
strigoi> marketplace update
```
Refreshes the local marketplace cache with latest module listings.

### Module Information
```
strigoi> marketplace info <module>
```
Displays detailed information about a specific module.

## Module Manifest Structure

```yaml
strigoi_module:
  identity:
    id: MOD-2025-00001
    name: "Module Name"
    version: "1.0.0"
    type: "scanner|attack|discovery|auxiliary"
    
  classification:
    risk_level: "low|medium|high|critical"
    white_hat_permitted: true
    ethical_constraints:
      - "For authorized testing only"
      
  specification:
    targets: ["linux", "windows", "macos"]
    capabilities:
      - "What the module can do"
    prerequisites:
      - "Required conditions"
      
  provenance:
    pipeline_run_id: "PIPE-2025-001"
    source_repository: "https://github.com/org/repo"
    pipeline_stages:
      request:
        document: "REQ-2025-001.md"
        commit: "abc123"
        timestamp: "2025-01-26T10:00:00Z"
      # ... other stages
      
  distribution:
    channel: "official|community"
    uri: "https://download.url/module.tar.gz"
    verification:
      method: "sha256"
      hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
      size_bytes: 45678
```

## Trust Model

### Official Modules
- Developed by Macawi-AI team
- Thoroughly tested and audited
- No warning prompts on installation
- Namespaces: `mcp/`, `network/`, `web/`, etc.

### Community Modules
- Third-party contributions
- Require explicit user consent
- Display prominent security warnings
- Username-based namespaces: `johnsmith/module-name`

## Implementation Status

✅ **Completed**:
- Marketplace client infrastructure
- SHA-256 verification system
- Trust manager with consent prompts
- Console command integration
- Module manifest parser
- Search and install functionality

⏳ **Pending**:
- GitHub repository setup
- Official module catalog
- CDN integration for downloads
- Module submission pipeline
- Community contribution guidelines

## Security Considerations

1. **Verification**: All modules MUST pass SHA-256 verification
2. **Consent**: Third-party modules require explicit user approval
3. **Isolation**: Modules run in constrained environments
4. **Auditing**: All installations are logged for security review
5. **Updates**: Regular security patches through version management

## Future Enhancements

- GPG signature verification
- Module sandboxing
- Dependency resolution
- Automatic security updates
- Community ratings and reviews
- Integration with vulnerability databases