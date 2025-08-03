# Strigoi Platform Strategy

## Linux-First Architecture

### Core Decision
Strigoi will be implemented as a **Linux-only** security framework. This strategic decision aligns with cybersecurity industry standards where Linux proficiency is assumed.

### Rationale
1. **Industry Standard**: Security professionals use Linux (Kali, Parrot, BlackArch)
2. **Simplified Architecture**: No cross-platform complexity
3. **Better Performance**: Native Linux system calls
4. **Security Focus**: Linux security primitives (capabilities, namespaces, cgroups)
5. **Tool Integration**: Seamless integration with existing Linux security tools

### Implementation Implications

#### What We Build For
- **Native Linux**: Full Linux kernel features
- **Distributions**: Debian/Ubuntu, RHEL/Fedora, Arch
- **Architectures**: amd64 primary, arm64 secondary
- **Containers**: Docker/Podman native support

#### What We Test Against
- **Windows Targets**: Via remote agents/streams
- **Windows Services**: MCP Server components on Windows Server
- **Cross-platform Protocols**: SSH, RDP, WinRM, SMB
- **Industrial Systems**: Windows-based SCADA/HMI

#### What We Don't Build
- ❌ Windows Strigoi binary
- ❌ Windows-specific console features  
- ❌ Windows service management
- ❌ Cross-platform filesystem abstractions

### Technical Benefits

#### System Integration
```go
// Direct Linux system calls
func (s *Stream) SetupPtrace() error {
    // Linux-specific ptrace
    return syscall.PtraceAttach(s.pid)
}

// Linux capabilities
func (m *Module) RequireCapabilities() []string {
    return []string{"CAP_NET_RAW", "CAP_SYS_PTRACE"}
}
```

#### Performance
- Direct epoll/io_uring for stream monitoring
- Native Linux namespaces for isolation
- eBPF for kernel-level inspection
- No abstraction layer overhead

#### Security Features
- SELinux/AppArmor integration
- Linux audit subsystem hooks
- Kernel module support (future)
- Direct /proc and /sys access

### Remote Target Support

While Strigoi runs on Linux, it can monitor and protect:
- Windows servers via WinRM/SSH
- Windows workstations via deployed agents
- macOS systems via SSH
- IoT devices via serial/network
- Cloud services via APIs

### Development Environment

#### Required
- Linux development machine
- Go 1.21+ on Linux
- Linux-specific tools (strace, ltrace, etc.)

#### Optional
- Windows VMs for target testing
- WINE for Windows binary analysis
- Cross-compilation for agent deployment

### Deployment Scenarios

1. **Security Operations Center**
   - Strigoi on Linux jump boxes
   - Monitor heterogeneous infrastructure
   - Central Linux-based deployment

2. **Incident Response**
   - Kali Linux with Strigoi
   - Portable USB deployment
   - Live boot environments

3. **Cloud Security**
   - Container-based deployment
   - Kubernetes DaemonSets
   - Cloud-native Linux

4. **Industrial Security**
   - Linux gateway monitoring Windows SCADA
   - Serial port access for PLCs
   - Real-time stream analysis

### Migration Path

For any existing Windows considerations:
1. Remove Windows-specific code
2. Convert to remote monitoring approach
3. Deploy Linux-based collectors
4. Use Cyreal A2A for Windows agents

### Conclusion

By focusing on Linux-only implementation, Strigoi can:
- Leverage full Linux security capabilities
- Simplify codebase significantly
- Align with security industry practices
- Maintain Windows target support via remote monitoring

This is the right architectural decision for a serious security framework.

---

*"In cybersecurity, Linux isn't just an option - it's the foundation"*