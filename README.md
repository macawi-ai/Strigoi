# Strigoi - AI/LLM Security Assessment CLI

[![CI](https://github.com/macawi-ai/strigoi/actions/workflows/ci.yml/badge.svg)](https://github.com/macawi-ai/strigoi/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/macawi-ai/strigoi)](https://goreportcard.com/report/github.com/macawi-ai/strigoi)
[![License: CC BY-NC-SA 4.0](https://img.shields.io/badge/License-CC%20BY--NC--SA%204.0-lightgrey.svg)](https://creativecommons.org/licenses/by-nc-sa/4.0/)

Strigoi is an interactive CLI tool designed for security assessment of AI/LLM systems and integrations. It provides an intelligent command interface with directional reconnaissance capabilities specifically tailored for modern AI infrastructure security testing.

## Features

- 🤖 **AI-Focused Security**: Specialized for LLM and AI system security assessment
- 🧭 **Directional Reconnaissance**: Multi-perspective analysis (north/south/east/west)
  - **North**: API endpoints and external interfaces
  - **South**: Dependencies and supply chain analysis
  - **East**: Data flows and AI model integrations
  - **West**: Authentication and access controls
- 📊 **Intelligence Gathering**: Real-time monitoring and analysis capabilities
- 🎨 **Intuitive Interface**: Color-coded CLI with bash-like navigation
- 🔧 **Extensible Framework**: Modular architecture for custom AI security modules

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
# Start interactive AI security assessment
./strigoi

# Navigate the assessment framework
strigoi> ls
strigoi> cd probe
strigoi/probe> ls

# Conduct directional reconnaissance
strigoi/probe> north api.openai.com     # API security assessment
strigoi/probe> south requirements.txt   # Dependency analysis
strigoi/probe> east data_flow.json      # Data integration review
strigoi/probe> west auth_config.yml     # Access control analysis
strigoi/probe> all target_system        # Comprehensive assessment

# Monitor AI system interactions
strigoi> cd stream
strigoi/stream> tap llm_process_id

# Get contextual help
strigoi> help
strigoi> ?

# Exit
strigoi> exit
```

### AI Security Assessment Framework

```
strigoi/
├── probe/                    # AI/LLM Security Assessment
│   ├── north                # API endpoints & external interfaces
│   ├── south                # Dependencies & AI model supply chain
│   ├── east                 # Data flows & model integrations
│   ├── west                 # Authentication & access controls
│   ├── all                  # Comprehensive multi-directional scan
│   └── center               # Central intelligence coordination
└── stream/                   # AI System Monitoring
    ├── tap                  # Monitor LLM process interactions
    └── status               # Real-time assessment status
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

## AI Security Focus Areas

Strigoi is specifically designed to address the unique security challenges of AI/LLM systems:

### 🎯 **Target Environments**
- LLM API integrations and endpoints
- AI model deployment pipelines
- Machine learning inference systems
- AI-powered application stacks
- Model Context Protocol (MCP) implementations

### 🔍 **Assessment Capabilities**
- **API Security**: LLM endpoint vulnerabilities and misconfigurations
- **Supply Chain**: AI model and dependency integrity analysis
- **Data Flow**: Training data and inference pipeline security
- **Access Control**: AI system authentication and authorization
- **Behavioral Analysis**: Real-time LLM interaction monitoring

### 🚀 **Current Implementation Status**
- [x] Interactive AI-focused CLI framework
- [x] Directional probe architecture (north/south/east/west/all/center)
- [x] Color-coded intelligent interface
- [x] Stream monitoring for AI processes
- [x] Extensible module system for AI security tools
- [x] Professional installer and deployment

**Framework Status**: Production-ready CLI framework with modular architecture for AI security assessment tools. Active development of specialized AI/LLM security modules.

## Support

- 📧 Email: support@macawi.ai
- 🐛 Issues: [GitHub Issues](https://github.com/macawi-ai/strigoi/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/macawi-ai/strigoi/discussions)

## License

Copyright © September 2025 Macawi LLC. All Rights Reserved.

This work is licensed under the [Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International License](http://creativecommons.org/licenses/by-nc-sa/4.0/).

**Attribution Required**: "Strigoi Security Validation Platform by Macawi LLC"

**Commercial Use**: Contact support@macawi.ai for commercial licensing.

---

Built with ♥️ for the security community