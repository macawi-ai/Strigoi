# Same-User Security Catastrophe Demo

This demonstration shows the devastating reality of MCP's same-user security model. In typical deployments, all MCP servers run as the same user, creating a massive attack surface where ANY compromise leads to TOTAL compromise.

## What This Demo Shows

1. **Typical MCP Deployment** - Multiple servers, all running as same user
2. **Attack Surface Enumeration** - Finding all MCP processes and their secrets
3. **Credential Extraction** - Multiple methods to steal credentials
4. **Impact Demonstration** - What an attacker gains from same-user access

## Demo Components

- `typical-deployment.json` - Real-world MCP configuration
- `launch-mcp-servers.sh` - Simulates starting multiple MCP servers
- `attack-demo.sh` - Shows various attack techniques
- `mock-servers/` - Simple mock MCP servers for safe demonstration
- `impact-report.sh` - Generates report of what attacker gained

## Running the Demo

### Step 1: Launch Mock MCP Environment
```bash
./launch-mcp-servers.sh
```

This starts several mock MCP servers simulating:
- Database server (with credentials in command line)
- File system server (with directory access)
- Slack integration (with API token)
- GitHub integration (with access token)

### Step 2: Run Attack Demonstration
```bash
./attack-demo.sh
```

This shows how an attacker with same-user access can:
- Enumerate all MCP processes
- Extract credentials from command lines
- Read environment variables
- Access STDIO pipes
- Dump process memory

### Step 3: View Impact Report
```bash
./impact-report.sh
```

See exactly what the attacker gained access to.

## Key Takeaways

1. **No Privilege Escalation Needed** - Same user = game over
2. **Multiple Attack Vectors** - Many ways to extract secrets
3. **Total Compromise** - One breach = access to everything
4. **Undetectable** - Looks like normal user activity
5. **Unfixable** - This is architectural, not a bug

## Warning

This demonstration uses mock servers and fake credentials. In a real environment, the exposed credentials would provide access to:
- Production databases
- Customer data
- Source code repositories
- Communication platforms
- Cloud infrastructure

**Never run real MCP servers with production credentials in this manner!**