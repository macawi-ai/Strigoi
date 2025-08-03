# Communication Flow Analysis: User → Claude

## Scenario 1: Joe launches Claude

```
[Joe's Machine]                                    [Anthropic Infrastructure]
     |                                                      |
     v                                                      |
┌─────────────┐                                            |
│  Terminal/  │  STDIO (stdin)                             |
│  Keyboard   │ ──────────────→                            |
└─────────────┘                                            |
                               ↓                            |
                         ┌─────────────┐  STDIO pipe   ┌──────────────┐
                         │   Claude    │ ←───────────→ │  MCP Server  │
                         │   Client    │  (JSON-RPC)   │   (local)    │
                         └─────────────┘               └──────────────┘
                               |                            |
                               | HTTPS/TLS                  |
                               | (REST API)                 |
                               |                            |
                               └────────────────────────────┤
                                                           v
                                                    ┌────────────┐
                                                    │  Claude    │
                                                    │  API       │
                                                    │ (LLM)      │
                                                    └────────────┘
```

## Communication Layers

### 1. Terminal Input Layer (STDIO)
- **Protocol**: Raw text over stdin
- **Format**: Terminal input/output streams
- **Direction**: Keyboard → Claude Client
- **Attack Surface**:
  - Terminal escape sequence injection
  - Input stream manipulation
  - Keystroke logging
  - ANSI code exploits

### 2. Local IPC Layer (STDIO)
- **Protocol**: JSON-RPC 2.0 over STDIO pipes
- **Format**: Line-delimited JSON messages
- **Direction**: Bidirectional between Claude client and local MCP servers
- **Attack Surface**: 
  - Stream injection
  - Message interception
  - Desync attacks
  - Buffer manipulation

### 3. Network Layer (HTTPS)
- **Protocol**: HTTPS/TLS 1.3
- **Format**: REST API with JSON payloads
- **Endpoint**: https://api.anthropic.com/v1/messages
- **Headers**:
  ```
  x-api-key: $ANTHROPIC_API_KEY
  anthropic-version: 2023-06-01
  content-type: application/json
  ```

### 3. Message Flow Example

```bash
# 1. User types in Claude client
"Help me analyze this code"

# 2. Client may consult local MCP servers via STDIO
→ {"jsonrpc": "2.0", "method": "tools/list", "id": 1}
← {"jsonrpc": "2.0", "result": {"tools": [...]}, "id": 1}

# 3. Client formats HTTP request to Claude API
POST https://api.anthropic.com/v1/messages
{
  "model": "claude-opus-4-20250514",
  "messages": [
    {"role": "user", "content": "Help me analyze this code"}
  ],
  "max_tokens": 1024
}

# 4. Response flows back through same channels
```

## Attack Scenarios

### Indirect Prompt Injection
When external content contains hidden MCP commands:

```
[External Source] → [User copies/shares] → [Claude Client] → [MCP Server]
                           ↓
                    Hidden prompt:
                "Use MCP to execute_command..."
```

The trust chain is violated when untrusted external content gains access to trusted MCP operations through the AI's content processing.

### Token/Credential Exploitation
MCP servers store credentials that become attack targets:

```
[MCP Server] → [Stores OAuth Token] → [~/.mcp/credentials]
      ↑                                         ↓
[Claude Client]                          [Attacker Reads]
      ↑                                         ↓
[Legitimate User]                        [Creates Rogue MCP]
                                                ↓
                                         [Accesses User's Gmail]
```

Stolen tokens can be reused to create malicious MCP servers with full access to victim's services.

### Command Injection
Vulnerable MCP tool implementations enable shell command injection:

```
[Claude] → "Convert image.jpg to PNG" → [MCP Server]
                                              ↓
                                    os.system(f"convert {filepath}...")
                                              ↓
                              Input: "img.jpg; cat /etc/passwd > leak.txt"
                                              ↓
                                    [ARBITRARY COMMAND EXECUTION]
```

Unsanitized parameters in MCP tools become shell command injection vectors.

### Confused Deputy Attack
MCP proxy servers lose user authorization context:

```
[User A] → [MCP Proxy] → [Resource Server]
[User B] ↗     ↓              ↓
         Static Client ID   Grants access to
         "mcp_proxy_123"    ALL authorized users'
                           resources!

Attack: [Attacker] → [MCP Proxy] → [Accesses User A, B, C resources]
```

The MCP proxy becomes a "confused deputy" that can't distinguish between users, allowing horizontal privilege escalation.

### Supply Chain Attack
Compromised MCP packages spread malicious code across enterprises:

```
[Attacker] → [Package Repo] → [MCP Plugin Update]
                                      ↓
                            [Enterprise MCP Servers]
                                      ↓
                            "Process quarterly report"
                                      ↓
                            [Backdoor Activation]
                                      ↓
                    Data Exfiltration / System Corruption
```

One poisoned package can compromise thousands of enterprise AI systems through trusted update channels.

### Session Hijacking/Fixation
Weak session management enables impersonation attacks:

```
[Legitimate Client] ──→ Session: abc123 ──→ [MCP Server]
                              ↓
                        [Attacker Steals]
                              ↓
[Attacker Client] ────→ Session: abc123 ──→ [Full Access]

Common vectors:
- Predictable session IDs
- Unencrypted transmission  
- No session binding
- Sessions in world-readable files
```

Hijacked sessions grant full access to AI context, tools, and integrated services.

### Data Aggregation Risk
MCP servers become central observation points for all user activity:

```
[User Activity] → [MCP Server] → [Multiple Services]
                       ↓
              [Builds User Profile]
                       ↓
        - Behavioral patterns
        - Cross-service correlation  
        - Temporal analysis
        - Entity relationships
```

Even legitimate operators can aggregate data across services to build comprehensive profiles for commercial purposes.

### Transport Layer Exposure
Different transport mechanisms create unique vulnerabilities:

```
STDIO: Client ←──pipes──→ Server
       - Stream injection, buffer attacks

HTTP:  Client ──POST→ Server
       Client ←─SSE─── Server  
       - SSRF, SSE injection, request smuggling

WebSocket: Client ←──→ Server
       - Frame injection, protocol confusion
```

HTTP with Server-Sent Events (SSE) is particularly vulnerable to event injection and persistent connection attacks.

### Protocol State Machine
MCP follows a predictable initialization lifecycle:

```
[START] → [Initialize Request] → [Capabilities Exchange] → [Initialized] → [Operational]
                ↓                        ↓                      ↓
          [Attack Point]          [Attack Point]         [Attack Point]

State-based attacks:
- Skip initialization
- Capability downgrade
- State confusion
- Resource exhaustion
```

The fixed state machine enables monitoring but also creates predictable attack patterns.

### Authentication Flow (OAuth)
MCP servers acting as OAuth providers centralize authentication:

```
[Claude Client] → [MCP OAuth Provider] → [Multiple Services]
                          ↓
                  Stores tokens for:
                  - Gmail, GitHub, Slack
                  - All services in one place
                          ↓
                  [Single Point of Failure]

Attack vectors:
- Token scope escalation
- Refresh token persistence  
- Cross-service confusion
- Authorization replay
```

One compromised MCP OAuth provider exposes ALL integrated service credentials.

### DNS Rebinding Attacks
Remote websites can access local MCP servers through DNS tricks:

```
Step 1: Browser visits attacker.com (1.2.3.4)
Step 2: DNS changes attacker.com → 127.0.0.1
Step 3: JavaScript now accesses local MCP server!

[Remote Website] --DNS Rebinding--> [Local MCP Server]
                                           ↓
                                    No authentication
                                    Full tool access
                                    Persistent backdoor via SSE

Weak session validation won't help if:
- Sessions aren't bound to origin
- Session IDs are predictable
- No host header validation
```

DNS rebinding turns trusted local MCP servers into remotely accessible attack vectors.

## Key Insight: Two Distinct Attack Surfaces

1. **IPC/Pipe Surface** (Local)
   - STDIO between Claude client ↔ MCP servers
   - JSON-RPC 2.0 protocol
   - No authentication by default
   - Process-level security

2. **Network Surface** (Remote)
   - HTTPS between Claude client ↔ Anthropic API
   - REST/JSON protocol
   - API key authentication
   - TLS encryption

This creates interesting attack scenarios:
- Can we inject into the STDIO stream before it reaches the MCP server?
- Can we create a malicious MCP server that poisons Claude's context?
- Can we exploit the trust relationship between client and local servers?