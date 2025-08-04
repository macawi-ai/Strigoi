# Changelog

All notable changes to the Strigoi project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Session management roadmap documentation
- Changelog file for tracking releases

## [0.5.0] - 2024-01-15

### Added
- Complete Cobra CLI integration with REPL mode
- Module registry and loader system
- Probe/North module for HTTP endpoint discovery
- Module management commands (list, info, search, use)
- TAB completion for all commands
- Comprehensive test suite
- GitHub issue templates
- GitHub project board
- Development methodology documentation

### Changed
- Reduced codebase from 164 files (37,538 lines) to 18 files (2,628 lines)
- Migrated from custom CLI to Cobra framework
- Reorganized project structure for clarity
- Updated all imports to use new module path

### Removed
- 139 legacy Go files
- Archived experimental code (S1, S4, S5 directories)
- Duplicate implementations
- Vendored dependencies
- DuckDB direct dependencies

### Fixed
- TAB completion in REPL mode
- Import paths consistency
- All linting issues (errcheck, revive, godot, gofmt, staticcheck)
- Test isolation issues
- Git repository size (reduced from 1.3GB to 18MB)

### Security
- Implemented secure module loading
- Added input validation for all commands
- Module sandboxing preparation

## [0.4.0] - 2023-12-01

### Added
- Initial module system design
- Basic CLI structure
- Probe concept implementation

### Changed
- Project architecture exploration

## [0.3.0] - 2023-10-15

### Added
- Core validation engine
- Stream monitoring concepts
- Basic security checks

## [0.2.0] - 2023-09-01

### Added
- Project foundation
- Initial research phase
- Proof of concept code

## [0.1.0] - 2023-08-15

### Added
- Initial project creation
- Basic README
- License file

---

## Upcoming Releases

### [0.6.0] - Session Management (Planned)
- Core session save/load functionality
- AES-256-GCM encryption
- Argon2id key derivation
- Session versioning

### [0.7.0] - Enhanced Sessions (Planned)
- Session templates
- Environment variable support
- Session metadata and tagging
- Session diff functionality

### [0.8.0] - Advanced Features (Planned)
- Multi-module playbooks
- Session history tracking
- YAML/TOML format support
- Advanced security features

### [0.9.0] - Pre-1.0 Stabilization (Planned)
- Performance optimizations
- API stabilization
- Comprehensive documentation
- Security audit completion

### [1.0.0] - Production Release (Planned)
- First stable release
- Full feature set
- Enterprise readiness
- Long-term support (LTS)

[Unreleased]: https://github.com/macawi-ai/strigoi/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/macawi-ai/strigoi/releases/tag/v0.5.0
[0.4.0]: https://github.com/macawi-ai/strigoi/releases/tag/v0.4.0
[0.3.0]: https://github.com/macawi-ai/strigoi/releases/tag/v0.3.0
[0.2.0]: https://github.com/macawi-ai/strigoi/releases/tag/v0.2.0
[0.1.0]: https://github.com/macawi-ai/strigoi/releases/tag/v0.1.0