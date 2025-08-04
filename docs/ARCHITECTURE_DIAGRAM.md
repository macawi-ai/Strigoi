# Strigoi Architecture Diagram

```mermaid
graph TB
    subgraph "CLI Layer"
        CLI[strigoi CLI]
        REPL[Interactive REPL]
        CMD[Cobra Commands]
    end
    
    subgraph "Command Groups"
        PROBE[probe/*]
        STREAM[stream/*]
        SENSE[sense/*]
        MODULE[module/*]
    end
    
    subgraph "Module System"
        REG[Module Registry]
        LOADER[Module Loader]
        IFACE[Module Interface]
    end
    
    subgraph "Security Modules"
        PNORTH[probe/north<br/>API Discovery]
        PSOUTH[probe/south<br/>Dependencies]
        PEAST[probe/east<br/>Data Flows]
        PWEST[probe/west<br/>Auth Testing]
        STAP[stream/tap<br/>STDIO Monitor]
    end
    
    subgraph "Core Components"
        BASE[Base Module]
        TYPES[Type System]
        RESULTS[Result Handler]
    end
    
    subgraph "Output"
        JSON[JSON Format]
        TABLE[Table Format]
        YAML[YAML Format]
    end
    
    CLI --> CMD
    CLI --> REPL
    CMD --> PROBE
    CMD --> STREAM
    CMD --> SENSE
    CMD --> MODULE
    
    MODULE --> REG
    REG --> LOADER
    LOADER --> IFACE
    
    PROBE --> PNORTH
    PROBE --> PSOUTH
    PROBE --> PEAST
    PROBE --> PWEST
    STREAM --> STAP
    
    PNORTH --> BASE
    PSOUTH --> BASE
    PEAST --> BASE
    PWEST --> BASE
    STAP --> BASE
    
    BASE --> TYPES
    BASE --> RESULTS
    
    RESULTS --> JSON
    RESULTS --> TABLE
    RESULTS --> YAML
    
    style PNORTH fill:#90EE90
    style REG fill:#87CEEB
    style LOADER fill:#87CEEB
    style MODULE fill:#FFB6C1
```

## Component Descriptions

### CLI Layer
- **strigoi CLI**: Main entry point using Cobra framework
- **Interactive REPL**: Shell-like navigation with cd/ls/pwd commands
- **Cobra Commands**: Structured command hierarchy

### Command Groups
- **probe/\***: Discovery and reconnaissance tools
- **stream/\***: Real-time STDIO monitoring
- **sense/\***: Passive network monitoring
- **module/\***: Module management (list, info, search, use)

### Module System
- **Module Registry**: Thread-safe storage and retrieval of modules
- **Module Loader**: Handles built-in and plugin modules
- **Module Interface**: Common contract all modules implement

### Security Modules
- **probe/north**: ✅ Implemented - API endpoint discovery
- **probe/south**: 🚧 Planned - Dependency analysis
- **probe/east**: 🚧 Planned - Data flow tracing
- **probe/west**: 🚧 Planned - Authentication testing
- **stream/tap**: 🚧 Planned - STDIO monitoring

### Core Components
- **Base Module**: Common functionality shared by all modules
- **Type System**: Module types, options, results
- **Result Handler**: Formats output for different consumers

### Output Formats
- **JSON**: Machine-readable format
- **Table**: Human-readable colored output
- **YAML**: Configuration-friendly format

## Data Flow

1. User invokes command → CLI parses arguments
2. Command looks up module in Registry
3. Module is configured with options
4. Module executes and returns results
5. Results are formatted based on output flag
6. Output is displayed to user

## Extension Points

1. **New Modules**: Implement Module interface and register
2. **New Commands**: Add Cobra commands to cmd/strigoi
3. **New Output Formats**: Extend result formatting
4. **Plugin System**: Load external .so files (future)