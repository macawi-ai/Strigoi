# Legacy Code Cleanup Plan

## Overview
This document outlines the plan to clean up legacy code from Strigoi v0.5.0 to create a cleaner, more maintainable codebase.

## Legacy Components Identified

### 1. Legacy Command Line Tools (`cmd/`)
- `cmd/migrate/` - Old module migration tool
- `cmd/migrate-vulns/` - Vulnerability migration tool
- `cmd/registry/` - Old registry system
- `cmd/registry-query/` - Registry query tool
- `cmd/test-json/` - JSON testing utility
- `cmd/test-registry/` - Registry testing
- `cmd/gemini-bridge/` - Gemini AI integration
- `cmd/package-demo/` - Demo packaging tool
- `cmd/strigoi-debug/` - Debug utility

### 2. Internal Legacy Packages (`internal/`)
- `internal/actors/` - Old actor system
- `internal/ai/` - AI integration (old)
- `internal/core/` - Old core framework
- `internal/licensing/` - Licensing system
- `internal/marketplace/` - Module marketplace
- `internal/modules.bak/` - Backup of old modules
- `internal/packages/` - Package management
- `internal/registry/` - Old registry implementation
- `internal/security/` - Security utilities
- `internal/state/` - State management
- `internal/stream/` - Old stream processing

### 3. Other Legacy Items
- `protocols/` - Old protocol implementations
- `demos/` - Demo applications
- Various test files using old framework

## Cleanup Strategy

### Phase 1: Backup Current State
```bash
# Create backup of entire project
tar -czf strigoi-v0.5.0-pre-cleanup.tar.gz .

# Create specific backup of legacy code
mkdir -p /home/cy/archives/Strigoi/legacy-code-backup
cp -r cmd/migrate* cmd/registry* cmd/test-* cmd/gemini-bridge cmd/package-demo /home/cy/archives/Strigoi/legacy-code-backup/
cp -r internal/ /home/cy/archives/Strigoi/legacy-code-backup/
cp -r protocols/ /home/cy/archives/Strigoi/legacy-code-backup/
cp -r demos/ /home/cy/archives/Strigoi/legacy-code-backup/
```

### Phase 2: Remove Legacy Commands
```bash
# Remove old command line tools
rm -rf cmd/migrate/
rm -rf cmd/migrate-vulns/
rm -rf cmd/registry/
rm -rf cmd/registry-query/
rm -rf cmd/test-json/
rm -rf cmd/test-registry/
rm -rf cmd/gemini-bridge/
rm -rf cmd/package-demo/
rm -rf cmd/strigoi-debug/
```

### Phase 3: Remove Legacy Internal Packages
```bash
# Remove all internal packages (keeping only what's actively used)
rm -rf internal/actors/
rm -rf internal/ai/
rm -rf internal/core/
rm -rf internal/licensing/
rm -rf internal/marketplace/
rm -rf internal/modules.bak/
rm -rf internal/packages/
rm -rf internal/registry/
rm -rf internal/security/
rm -rf internal/state/
rm -rf internal/stream/
```

### Phase 4: Remove Other Legacy Items
```bash
# Remove old protocols and demos
rm -rf protocols/
rm -rf demos/
```

### Phase 5: Clean Up References
1. Update go.mod to remove unused dependencies
2. Run `go mod tidy` to clean up
3. Fix any import errors in remaining code

## What to Keep

### Essential Components
- `cmd/strigoi/` - Main CLI application
- `modules/` - New module system
- `pkg/` - Shared packages
- `docs/` - Documentation
- `.github/` - GitHub integration
- Core project files (go.mod, README.md, etc.)

### Review Before Deletion
- Check if any internal packages are imported by cmd/strigoi
- Verify no critical functionality depends on legacy code

## Expected Outcome

### Before Cleanup
- 157 Go files
- Multiple unused packages
- Confusing directory structure
- Build errors from legacy code

### After Cleanup
- ~40 Go files (estimated)
- Clean, focused codebase
- Clear module architecture
- No build errors

## Risks and Mitigation

### Risk 1: Removing Used Code
- **Mitigation**: Full backup before deletion
- **Mitigation**: Test build after each phase

### Risk 2: Breaking Dependencies
- **Mitigation**: Use go compiler to identify issues
- **Mitigation**: Keep backup readily available

### Risk 3: Lost Functionality
- **Mitigation**: Document what each component did
- **Mitigation**: Plan to reimplement if needed

## Execution Timeline

1. **Backup**: 5 minutes
2. **Remove Commands**: 2 minutes
3. **Remove Internal**: 2 minutes
4. **Remove Others**: 2 minutes
5. **Fix References**: 10-20 minutes
6. **Testing**: 10 minutes

**Total**: ~30-40 minutes

## Success Criteria

- [ ] All legacy code removed
- [ ] Project builds without errors
- [ ] All tests pass
- [ ] Binary runs correctly
- [ ] Module system still functional
- [ ] Documentation updated