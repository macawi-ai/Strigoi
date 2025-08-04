# Strigoi Architecture

## Overview

Strigoi is built on a modular architecture that enables extensible security testing capabilities. The system consists of a core framework, a command-line interface with REPL support, and pluggable security modules.

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     CLI Layer (Cobra)                       │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │    REPL     │  │  Navigation  │  │  TAB Completion │  │
│  └─────────────┘  └──────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Core Framework                         │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │   Module    │  │   Session    │  │     Logger      │  │
│  │  Manager    │  │   Manager    │  │                 │  │
│  └─────────────┘  └──────────────┘  └─────────────────┘  │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │   Policy    │  │   Reporter   │  │  State Manager  │  │
│  │   Engine    │  │              │  │                 │  │
│  └─────────────┘  └──────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Security Modules                         │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │    Probe    │  │    Stream    │  │     Sense       │  │
│  │  (Discovery)│  │ (Monitoring) │  │  (Environment)  │  │
│  └─────────────┘  └──────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. CLI Layer

**Technology**: Cobra framework

**Components**:
- **REPL**: Interactive command-line interface with context-aware navigation
- **Command Tree**: Hierarchical command structure (probe/, stream/, sense/)
- **TAB Completion**: Multi-word completion support
- **Color Output**: Visual distinction for different command types

### 2. Core Framework

**Module Manager**
- Loads and manages security modules
- Handles module lifecycle
- Provides module discovery

**Session Manager**
- Tracks current session state
- Manages authentication/authorization
- Stores session history

**Policy Engine**
- Enforces security policies
- Filters findings based on rules
- Evaluates risk levels

**Reporter**
- Generates reports in multiple formats
- Supports streaming output
- Customizable templates

### 3. Module System

## Module Interface

All Strigoi modules implement the `Module` interface:

```go
type Module interface {
    Name() string                          // Unique module identifier
    Description() string                   // Human-readable description
    Type() ModuleType                     // Module category
    Options() map[string]*ModuleOption    // Configuration options
    SetOption(name, value string) error   // Set option value
    ValidateOptions() error               // Validate configuration
    Run() (*ModuleResult, error)         // Execute module
    Check() bool                         // Verify dependencies
    Info() *ModuleInfo                   // Module metadata
}
```

### Module Types

1. **Probe Modules** (`ModuleTypeDiscovery`)
   - Network discovery
   - Service enumeration
   - API endpoint detection
   - Dependency mapping

2. **Stream Modules** (`ModuleTypeScanner`)
   - Real-time monitoring
   - STDIO capture
   - Traffic analysis
   - Pattern detection

3. **Sense Modules** (`ModuleTypeAuxiliary`)
   - Environment detection
   - Configuration analysis
   - Context awareness

## Module Development

### Creating a New Module

1. Implement the `Module` interface
2. Define options using `ModuleOption`
3. Implement validation logic
4. Return structured results using `ModuleResult`

### Module Structure

```go
type MyModule struct {
    BaseModule  // Embed base functionality
    target      string
    timeout     int
}

func (m *MyModule) Run() (*ModuleResult, error) {
    // Validate options
    if err := m.ValidateOptions(); err != nil {
        return nil, err
    }
    
    // Execute security test
    findings := m.performTest()
    
    // Return structured results
    return &ModuleResult{
        Success:  true,
        Findings: findings,
        Summary:  m.summarizeFindings(findings),
    }, nil
}
```

## Data Flow

### Command Execution Flow

```
User Input → REPL → Command Parser → Module Manager → Module
    ↓                                                    ↓
Terminal ← Reporter ← Policy Engine ← Module Result ←────┘
```

### Module Execution Lifecycle

1. **Initialization**: Module loaded and options set
2. **Validation**: Options validated, dependencies checked
3. **Execution**: Security test performed
4. **Collection**: Results gathered and structured
5. **Filtering**: Policy engine applies rules
6. **Reporting**: Results formatted and displayed

## Security Considerations

### Privilege Management
- Modules run with least required privileges
- Capability-based security model
- Audit trail for all operations

### Input Validation
- All user input sanitized
- Command injection prevention
- Path traversal protection

### Output Security
- Sensitive data redaction
- Configurable verbosity levels
- Secure credential storage

## Extension Points

### Custom Modules
- Place modules in `internal/modules/`
- Register with module manager
- Follow naming conventions

### Custom Reporters
- Implement `Reporter` interface
- Support streaming output
- Register format handlers

### Policy Extensions
- Add custom policy rules
- Implement risk calculators
- Create finding filters

## Performance Considerations

### Module Loading
- Lazy loading of modules
- Cached module metadata
- Parallel initialization

### Memory Management
- Stream processing for large data
- Ring buffers for monitoring
- Garbage collection optimization

### Concurrency
- Goroutine pools for parallel execution
- Context-based cancellation
- Rate limiting support

## Configuration

### Module Configuration
```yaml
modules:
  probe:
    timeout: 30s
    max_threads: 10
  stream:
    buffer_size: 1MB
    retention: 1h
```

### Framework Configuration
```yaml
framework:
  log_level: info
  output_format: json
  policy_file: policies.yaml
```

## Future Enhancements

1. **Plugin System**: Dynamic module loading
2. **Remote Modules**: Distributed testing
3. **AI Integration**: Smart vulnerability detection
4. **Graph Analysis**: Attack path visualization
5. **Real-time Collaboration**: Multi-user support

---

This architecture provides a solid foundation for building sophisticated security testing capabilities while maintaining modularity, security, and performance.