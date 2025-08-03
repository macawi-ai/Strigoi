# MCP Privilege Escalation Attack Surfaces

## The Privilege Requirements Breakdown

### Same-User Process Tracing (Usually No Root Required)

> "The specified process cannot be traced. This could be because the tracer has insufficient privileges (the required capability is CAP_SYS_PTRACE); unprivileged processes cannot trace processes that they cannot send signals to or those running set-user-ID/set-group-ID programs."

**Key principle**: You can generally trace processes you own without root privileges.

#### MCP Attack Scenarios

If Claude Desktop runs as user `alice` and MCP server runs as user `alice` (typical setup), then user `alice` can trace both processes without root:

```bash
# As alice, no sudo needed:
ptrace -p $(pgrep claude-desktop)
ptrace -p $(pgrep mcp-server)
gdb -p $(pgrep mcp-server)
strace -p $(pgrep claude-desktop)
```

### Cross-User Process Tracing (Root Required)

> "ptrace can attach only to processes that the owner can send signals to (typically only their own processes); the superuser account can ptrace almost any process."

When you need root:
- Tracing processes owned by different users
- Tracing setuid/setgid programs  
- Bypassing YAMA security module restrictions

## New Attack Surfaces Revealed

### 1. **Signal-Based Attack Surface**

The privilege model reveals that signal permissions determine traceability:

```bash
# If you can send signals, you can trace
kill -0 $PID && echo "Can trace this process"
```

Attack vectors:
- **Signal injection attacks**: Send SIGSTOP/SIGCONT to pause MCP at critical moments
- **Signal-based DoS**: Flood MCP with signals to degrade performance
- **Timing attacks**: Use signals to control MCP execution timing

### 2. **Capability-Based Attack Surface** 

The CAP_SYS_PTRACE capability creates a new surface:

```bash
# Check process capabilities
getcap /proc/$PID/exe

# Ambient capabilities inheritance
grep Cap /proc/$PID/status
```

Attack vectors:
- **Capability confusion**: Exploit capability inheritance bugs
- **Ambient capability attacks**: Child processes inherit tracing abilities
- **Container escape**: Capabilities often misconfigured in containers

### 3. **SUID/SGID Boundary Surface**

Setuid/setgid programs create security boundaries:

```bash
# MCP servers should NEVER be setuid
find / -perm -4000 -name "*mcp*" 2>/dev/null
```

Attack scenarios:
- **Privilege dropping failures**: SUID MCPs that don't drop privileges
- **Capability leakage**: SUID programs retaining CAP_SYS_PTRACE
- **Mixed privilege communication**: SUID MCP talking to non-SUID Claude

### 4. **User Namespace Attack Surface**

Modern Linux user namespaces add complexity:

```bash
# Check if running in user namespace
grep "^[0-9]" /proc/$PID/uid_map
```

Attack vectors:
- **Namespace confusion**: Different privilege models per namespace
- **UID mapping attacks**: Exploit UID translation bugs
- **Capability elevation**: User namespaces can grant capabilities

### 5. **Process Group Attack Surface**

Process groups determine signal propagation:

```bash
# Get process group
ps -o pid,pgid,cmd -p $PID
```

Attack scenarios:
- **Group signal attacks**: Kill entire MCP process group
- **Session hijacking**: Take over process session leader
- **Terminal control**: Steal controlling terminal

## The "Alice Scenario" - Complete Same-User Compromise

```bash
# Alice runs both Claude and MCP (typical developer setup)
$ whoami
alice

$ ps aux | grep -E "claude|mcp"
alice 1234 claude-desktop
alice 5678 mcp-oracle-server

# Alice (or malware running as alice) can:
# 1. Trace both processes
$ gdb -p 1234  # Full Claude control
$ gdb -p 5678  # Full MCP control

# 2. Read all memory
$ cat /proc/1234/maps
$ cat /proc/5678/environ

# 3. Inject code
$ echo 'call system("evil")' | gdb -p 5678

# 4. Steal credentials
$ strings /proc/5678/mem | grep -i password

# 5. Hijack STDIO
$ strace -p 5678 -e read,write

# NO ROOT REQUIRED!
```

## The Security Model Breakdown

### What MCP Assumes
1. Same user = trusted
2. Process isolation = security
3. STDIO = private channel

### What Actually Happens
1. Same user = FULL ACCESS
2. Process isolation = meaningless for same UID
3. STDIO = completely exposed

## New Surfaces Summary

1. **Signal Permission Surface**: Kill/trace correlation
2. **Capability Inheritance Surface**: CAP_SYS_PTRACE propagation
3. **SUID/SGID Boundary Surface**: Privilege transition vulnerabilities  
4. **User Namespace Surface**: Container/namespace boundaries
5. **Process Group Surface**: Group-based attacks
6. **Session Leader Surface**: Terminal and session control
7. **UID/GID Mapping Surface**: Identity translation attacks

## The Fundamental Flaw

MCP's security model assumes process boundaries provide security, but Linux's security model says:
- **Same UID = Full access**
- **Signals = Control**
- **Ptrace = Game over**

This isn't a bug - it's how UNIX was designed. MCP is architecturally incompatible with UNIX security principles.

## Mitigation (Spoiler: There Isn't One)

### What Doesn't Work
- **YAMA ptrace_scope**: Same UID still works
- **Seccomp**: Can't block same-user ptrace
- **AppArmor/SELinux**: Rarely configured for user processes
- **Containers**: Often share UID namespace

### What Would Work (But Nobody Does)
- Run each MCP as different UID
- Use proper IPC with authentication
- Implement capability-based security
- Basically: Don't use STDIO

## Conclusion

The privilege model analysis reveals that MCP's fundamental assumption (process isolation = security) is completely wrong on UNIX-like systems. Any process running as the same user has COMPLETE access to MCP's memory, file descriptors, and execution flow.

This isn't a vulnerability - it's the documented, intended behavior of UNIX process security. MCP is architecturally incompatible with UNIX security principles.