# Strigoi Surfaces Model - Reconnaissance Architecture

## Core Concept: Attack Surfaces

Instead of forcing users to understand module internals, we present "surfaces" - different aspects of an agentic system that can be assessed.

## Proposed Surface Hierarchy

### 1. **Network Surface** ✓ (Currently Implemented)
- Remote MCP servers
- API endpoints
- WebSocket connections
- Authentication mechanisms
- Rate limiting detection

### 2. **Local Surface** (Proposed)
- Local MCP server configurations
- Agent installation paths
- Configuration files
- Credential storage
- Log files and artifacts

### 3. **Code Surface** (Proposed)
- Static prompt analysis
- Tool implementation review
- Resource handler inspection
- Extension/plugin code
- Integration scripts

### 4. **Data Surface** (Proposed)
- Accessible file systems
- Database connections
- Environment variables
- Memory/context stores
- Cached responses

### 5. **Integration Surface** (Proposed)
- VS Code extensions (Continue, Cursor, etc.)
- IDE connections
- Third-party service integrations
- OAuth/API key usage
- Webhook configurations

### 6. **Permission Surface** (Proposed)
- Tool capabilities mapping
- Resource access rights
- Execution boundaries
- Sandboxing effectiveness
- Privilege escalation paths

### 7. **IPC Surface** (Proposed)
- STDIO pipes between processes
- Named pipes and UNIX sockets
- Shared memory segments
- Message queues
- Process-to-process channels

### 8. **Supply Chain Surface** (Proposed)
- Package repositories (NPM, PyPI, GitHub)
- Plugin/extension stores
- Dependency management
- Update mechanisms
- Build pipeline integrity

### 9. **Transport Surface** (Proposed)
- STDIO pipes and streams
- HTTP/HTTPS endpoints
- Server-Sent Events (SSE)
- WebSocket connections
- Protocol-specific vulnerabilities

### 10. **Infrastructure Surface** (Discovered)
- Internal network topology
- Service dependencies
- DNS patterns and naming
- Backup/recovery systems
- Hidden admin endpoints

### 11. **Binary/Execution Surface** (Discovered)
- Process hierarchy trees
- Executable locations
- Library dependencies
- Startup sequences
- System call patterns

### 12. **Credential Management Surface** (Discovered)
- Configuration file storage
- Environment variables
- TLS certificates
- API keys and tokens
- Secret propagation paths

### 13. **Monitoring/Debug Surface** (Discovered)
- Debug endpoints
- Admin interfaces
- Logging mechanisms
- Performance metrics
- Diagnostic APIs

### 14. **Signal Permission Surface** (Privilege Analysis)
- Process signal relationships
- Kill/trace permission correlation
- Signal injection attacks
- Timing manipulation via signals
- Process group signal propagation

### 15. **Capability Inheritance Surface** (Privilege Analysis)
- CAP_SYS_PTRACE propagation
- Ambient capabilities
- Capability confusion in containers
- Privilege capability transitions
- Capability-based bypasses

### 16. **SUID/SGID Boundary Surface** (Privilege Analysis)
- Setuid program boundaries
- Privilege dropping failures
- Mixed privilege communication
- SUID/normal process interaction
- Capability retention in SUID

### 17. **User Namespace Surface** (Privilege Analysis)
- Namespace privilege models
- UID/GID mapping vulnerabilities
- Container boundary confusion
- Capability elevation via namespaces
- Cross-namespace attacks

### 18. **Process Group/Session Surface** (Privilege Analysis)
- Process group attacks
- Session leader hijacking
- Terminal control theft
- Group signal attacks
- Session boundary violations

## User Experience Flow

```
strigoi > recon
[*] What surface would you like to assess?
  1. network   - Test remote MCP servers and APIs
  2. local     - Examine local agent installations
  3. code      - Analyze agent implementation
  4. data      - Map accessible data stores
  5. integrate - Check third-party connections
  6. perms     - Assess permission boundaries

Select surface (1-6): 1

[*] Network reconnaissance selected
[*] Enter target (IP/hostname): localhost:3000

[*] Running network surface assessment...
  ✓ MCP server detected
  ✓ 4 tools exposed
  ✓ 2 prompts available
  ✓ 3 resources listed
  ⚠ No authentication required
  ⚠ No rate limiting detected

[*] Quick assessment: HIGH RISK - Unauthenticated MCP server with dangerous tools

Run 'recon details' for full report or 'use <module>' to exploit findings.
```

## Implementation Notes

### Recon Command Structure
```
recon                    # Interactive surface selection
recon network <target>   # Direct surface targeting
recon all <target>       # Assess all applicable surfaces
recon details           # Show detailed findings from last recon
recon save <filename>   # Save reconnaissance report
```

### Surface Detection Logic
Each surface has:
- Auto-detection capabilities
- Quick assessment algorithms
- Risk scoring metrics
- Drill-down module recommendations

### Progressive Disclosure
1. Start with high-level "what responds?"
2. Show risk summary
3. Offer detailed analysis options
4. Suggest relevant exploitation modules

## Benefits Over Traditional Approach

1. **Lower Barrier**: Users don't need to know module names
2. **Contextual**: Surfaces make sense in agent/AI context
3. **Progressive**: Start simple, dive deep when needed
4. **Actionable**: Direct path from recon to exploitation
5. **Comprehensive**: Covers more than just network attacks