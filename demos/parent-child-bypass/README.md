# Parent-Child YAMA Bypass Demonstration

This demonstration shows how YAMA ptrace_scope restrictions can be trivially bypassed using parent-child relationships. This works on both Linux and Windows (with equivalent tools).

## The Vulnerability

YAMA's ptrace_scope=1 (default on Ubuntu) only allows tracing between parent-child processes. However, if an attacker can become the parent of the MCP server, they can trace all communication.

## Safe Demonstration

We provide a simple echo server that accepts "credentials" via command line arguments (simulating how MCP servers receive database connection strings). The demonstration shows how launching under a tracer captures everything.

## Files

1. `echo-server.py` - Simple server that echoes received messages (simulates MCP)
2. `demo-linux.sh` - Linux demonstration using strace
3. `demo-windows.ps1` - Windows demonstration using equivalent tools
4. `safer-launch.sh` - Shows how credentials should NOT be passed

## Running the Demo

### Linux
```bash
./demo-linux.sh
```

### Windows
```powershell
.\demo-windows.ps1
```

## What You'll See

The demonstrations show:
1. Credentials passed via command line are visible to parent processes
2. All STDIO communication is intercepted
3. No special privileges are required
4. The attack works regardless of YAMA settings

## Key Takeaway

Never pass sensitive information via:
- Command line arguments
- Environment variables
- STDIO without encryption
- Any channel accessible to parent processes