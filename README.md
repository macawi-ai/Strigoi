# Strigoi - Advanced Security Validation Platform

[![CI](https://github.com/macawi-ai/strigoi/actions/workflows/ci.yml/badge.svg)](https://github.com/macawi-ai/strigoi/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/macawi-ai/strigoi)](https://goreportcard.com/report/github.com/macawi-ai/strigoi)
[![License](https://img.shields.io/badge/License-Proprietary-red.svg)](LICENSE)

Strigoi is an advanced security validation platform that helps security professionals discover and validate attack surfaces in modern architectures, with a special focus on AI/LLM integrations and the Model Context Protocol (MCP).

## Features

- 🔍 **Interactive REPL**: Bash-like navigation with context-aware commands
- 🎯 **Smart Discovery**: Probe in cardinal directions for different attack surfaces
- 📊 **Stream Analysis**: Real-time STDIO monitoring and analysis
- 🤖 **AI-Aware**: Specialized modules for LLM and MCP security testing
- 🎨 **Color-Coded Interface**: Visual distinction between directories, commands, and utilities
- 📝 **Comprehensive Logging**: Detailed audit trails for all operations

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/macawi-ai/strigoi.git
cd strigoi

# Build the binary
make build

# Or install globally
make install
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
│   ├── north        # API endpoints and external interfaces
│   ├── south        # Dependencies and supply chain
│   ├── east         # Data flows and integrations
│   └── west         # Authentication and access controls
├── stream/          # STDIO monitoring
│   ├── tap          # Monitor process STDIO in real-time
│   ├── record       # Record streams for analysis
│   └── status       # Show monitoring status
└── sense/           # (Coming soon) Environmental awareness
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

## Roadmap

- [x] Interactive REPL with navigation
- [x] Basic probe commands
- [x] Color-coded interface
- [ ] Real security module implementation
- [ ] MCP vulnerability detection
- [ ] AI/LLM attack surface mapping
- [ ] Integration with popular security tools
- [ ] Web UI dashboard

## Support

- 📧 Email: support@macawi.ai
- 🐛 Issues: [GitHub Issues](https://github.com/macawi-ai/strigoi/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/macawi-ai/strigoi/discussions)

## License

Copyright © 2025 Macawi - James R. Saker Jr.

This is proprietary software. See [LICENSE](LICENSE) for details.

---

Built with ♥️ for the security community