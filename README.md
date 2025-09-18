# Strigoi - Interactive Security Shell

[![CI](https://github.com/macawi-ai/strigoi/actions/workflows/ci.yml/badge.svg)](https://github.com/macawi-ai/strigoi/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/macawi-ai/strigoi)](https://goreportcard.com/report/github.com/macawi-ai/strigoi)
[![License](https://img.shields.io/badge/License-Proprietary-red.svg)](LICENSE)

Strigoi is an interactive shell/REPL for security reconnaissance. It provides a bash-like interface with directional probe commands for exploring different aspects of target systems.

## Features

- 🔍 **Interactive REPL**: Bash-like navigation and command execution
- 🧭 **Directional Probes**: Explore targets from different perspectives (north/south/east/west)
- 📊 **Stream Monitoring**: Basic STDIO monitoring capabilities
- 🎨 **Color-Coded Interface**: Visual distinction between directories and commands
- 🔧 **Extensible**: Command tree structure for adding new probe types

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/macawi-ai/strigoi.git
cd strigoi

# Build the binary
make build

# Or install with the installer script
./install.sh
```

### Basic Usage

```bash
# Start interactive mode
./strigoi

# Navigate the command tree
strigoi> ls
strigoi> cd probe
strigoi/probe> ls
strigoi/probe> north localhost

# Get help
strigoi> help
strigoi> ?

# Exit
strigoi> exit
```

### Command Structure

```
strigoi/
├── probe/           # Discovery and reconnaissance
│   ├── north        # Probe north direction (endpoints)
│   ├── south        # Probe south direction (dependencies)
│   ├── east         # Probe east direction (data flows)
│   ├── west         # Probe west direction (integrations)
│   ├── all          # Probe all directions
│   └── center       # Central monitoring
└── stream/          # Stream monitoring
    ├── tap          # Monitor process STDIO
    └── status       # Show monitoring status
```

## Development

### Prerequisites

- Go 1.21 or higher
- Make
- Git

### Building from Source

```bash
# Get dependencies
make deps

# Run tests
make test

# Run linters
make lint

# Run security scan
make security

# Build binary
make build
```

### Contributing

Please read our [Development Methodology](docs/DEVELOPMENT_METHODOLOGY.md) for details on our code of conduct, development process, and how to submit pull requests.

### Project Structure

```
strigoi/
├── cmd/strigoi/      # Main application entry point
├── internal/         # Private application code
│   ├── core/         # Core framework
│   ├── modules/      # Security modules
│   └── actors/       # Actor model implementation
├── pkg/              # Public libraries
├── docs/             # Documentation
├── test/             # Test files
├── scripts/          # Build and utility scripts
└── examples/         # Example configurations
```

## Security Notice

⚠️ **WARNING**: This tool is designed for authorized security testing only. 

- Only use on systems you own or have explicit permission to test
- Follows responsible disclosure practices
- No warranty provided - use at your own risk

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - System design and components
- [Development Guide](docs/DEVELOPMENT_METHODOLOGY.md) - Contributing and development practices
- [API Reference](docs/API.md) - Public API documentation
- [Security Guide](docs/SECURITY.md) - Security considerations

## Current Status

This is an early-stage interactive shell framework. Currently implemented:

- [x] Interactive REPL with navigation
- [x] Basic probe command structure (north/south/east/west/all/center)
- [x] Color-coded interface
- [x] Stream monitoring framework
- [x] Installer script

**Note**: The probe commands currently provide basic framework functionality. Actual security scanning implementations are planned for future releases.

## Support

- 📧 Email: support@macawi.ai
- 🐛 Issues: [GitHub Issues](https://github.com/macawi-ai/strigoi/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/macawi-ai/strigoi/discussions)

## License

Copyright © 2025 Macawi - James R. Saker Jr.

This is proprietary software. See [LICENSE](LICENSE) for details.

---

Built with ♥️ for the security community