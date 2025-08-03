# Stream Infrastructure Architecture

## Overview

The `stream` command provides foundational data stream inspection and interception capabilities that other Strigoi modules leverage for security analysis. Rather than being a standalone analysis tool, it establishes monitored data streams that vulnerability scanners, compliance modules, and security tools can subscribe to.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Stream Infrastructure                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐    │
│  │ Stream Setup │───▶│ Stream Mgmt  │───▶│ Stream API   │    │
│  └──────────────┘    └──────────────┘    └──────────────┘    │
│          │                   │                     │            │
│          ▼                   ▼                     ▼            │
│  ┌──────────────────────────────────────────────────────┐     │
│  │               Active Stream Registry                   │     │
│  └──────────────────────────────────────────────────────┘     │
│                              │                                  │
└──────────────────────────────│──────────────────────────────────┘
                               │
        ┌──────────────────────┴──────────────────────────┐
        │                                                 │
        ▼                                                 ▼
┌──────────────────┐                          ┌──────────────────┐
│ Vuln Scanners    │                          │ Compliance Mods  │
├──────────────────┤                          ├──────────────────┤
│ • Injection Det. │                          │ • PII Scanner    │
│ • Protocol Vuln  │                          │ • PCI Auditor    │
│ • Overflow Check │                          │ • HIPAA Monitor  │
└──────────────────┘                          └──────────────────┘
        ▲                                                 ▲
        │                                                 │
        └─────────────────┬───────────────────────────────┘
                          │
                  ┌───────────────┐
                  │ Stream Data   │
                  └───────────────┘
```

## Stream Types

### 1. Process I/O Streams
- **Local STDIO**: Monitor stdin/stdout/stderr of local processes
- **Remote STDIO**: Deploy listeners on remote Linux/Windows systems
- **Purpose**: Detect command injection, data exfiltration, malicious commands

### 2. Serial Communication Streams  
- **RS-232**: Legacy serial port monitoring
- **RS-485**: Industrial/SCADA communication
- **USB Serial**: Modern serial-over-USB devices
- **Purpose**: IoT security, industrial control system monitoring

### 3. Network Streams (Future)
- **TCP/UDP**: Application protocol monitoring
- **WebSocket**: Real-time bidirectional streams
- **gRPC**: Modern RPC communication

## Core Components

### Stream Setup (`stream` command)
```
strigoi> stream setup stdio localhost
strigoi> stream setup serial /dev/ttyUSB0 9600
strigoi> stream setup remote 192.168.1.100 windows
```

### Stream Management
- List active streams
- Start/stop monitoring
- Configure filters and rules
- Set resource limits

### Stream API
- Subscribe to stream data
- Register pattern matchers
- Receive real-time alerts
- Access historical data

## Module Integration

Modules can subscribe to streams for analysis:

```go
// Example: PII Detection Module
func (m *PIIDetector) Run() (*ModuleResult, error) {
    // Subscribe to active streams
    streams := m.framework.GetActiveStreams()
    
    for _, stream := range streams {
        stream.Subscribe(m.analyzeData)
    }
    
    // Process stream data for PII
    // ...
}
```

## Use Cases

### 1. Agent Behavior Monitoring
- Monitor AI agent communications
- Detect prompt injections
- Identify data leakage
- Track protocol violations

### 2. Compliance Auditing
- Real-time PII detection
- PCI compliance monitoring
- HIPAA data flow tracking
- GDPR violation alerts

### 3. Security Analysis
- Command injection detection
- Data exfiltration monitoring
- Anomaly detection
- Pattern matching

### 4. Incident Response
- Stream redirection for containment
- Real-time intervention
- Evidence collection
- Forensic analysis

## Implementation Priority

1. **Phase 1**: Core stream infrastructure
   - Stream setup and management
   - Local STDIO monitoring
   - Basic API for modules

2. **Phase 2**: Remote capabilities
   - Remote listener deployment
   - Cross-platform support
   - Secure communication

3. **Phase 3**: Advanced features
   - Serial port monitoring
   - Stream persistence
   - Advanced filtering

## Security Considerations

- **Access Control**: Who can setup/access streams
- **Data Privacy**: Handling sensitive stream data
- **Performance**: Managing high-volume streams
- **Reliability**: Ensuring stream continuity
- **Intervention**: Safe stream redirection

---

*The stream infrastructure is the sensory system of Strigoi - it sees all, enabling other components to understand and respond.*