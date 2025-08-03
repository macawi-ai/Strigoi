# Attack Vector: Session Management Flaws in MCP

## Attack Overview
MCP servers often implement session management to maintain state across multiple requests. Poor session handling creates opportunities for session hijacking, fixation, and replay attacks, allowing attackers to impersonate legitimate users and AI assistants.

## Communication Flow Diagram

```
[Normal Session Flow]

┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│   Claude    │ ──────> │  MCP Server  │ ──────> │   Resource   │
│   Client    │  Init   │              │  Auth   │    Server    │
└─────────────┘         └──────────────┘         └──────────────┘
       |                        |                         
       | 1. initialize          | 2. Creates session      
       └───────────────────────>│    ID: abc123          
                                │                        
       |<───────────────────────┘                        
       | 3. Returns session ID                           
       |                                                
       | 4. Subsequent requests                         
       | Header: Session-ID: abc123                     
       └───────────────────────>│                        
                                │ 5. Validates session   
                                │    Processes request   

[Session Hijacking Attack]

┌─────────────┐                          ┌─────────────┐
│ Legitimate  │                          │  Attacker   │
│   Client    │                          │             │
└─────────────┘                          └─────────────┘
       |                                        |
       | Session-ID: abc123                     | Intercepts
       └──────────────────┐                     | Session ID
                          ↓                     |
                   ┌──────────────┐            |
                   │  MCP Server  │<───────────┘
                   └──────────────┘    Session-ID: abc123
                          |            (Hijacked)
                          |
                   "I'm the legitimate client!"
                          |
                          ↓
                   Full access granted

[Session Fixation Attack]

┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│  Attacker   │ ──────> │  MCP Server  │ <────── │   Victim    │
└─────────────┘  Sets   └──────────────┘  Uses   └─────────────┘
       |         fixed         |          fixed          |
       |         session       |          session        |
       └──────────────────────>│<────────────────────────┘
         Session-ID: evil666    Session-ID: evil666
                               |
                        Attacker controls
                        victim's session!
```

## Attack Layers

### Layer 1: Session Token Weaknesses
- **Predictable IDs**: Sequential, time-based, weak random
- **No encryption**: Tokens sent in plaintext
- **Long-lived**: Sessions never expire
- **No binding**: Token works from any client/IP

### Layer 2: Storage Vulnerabilities
```
Common Insecure Storage Locations:
├── /tmp/mcp-sessions/         (World readable)
├── ~/.mcp/sessions.json       (Plain text)
├── Environment variables      (Process inspection)
├── Browser localStorage       (XSS vulnerable)
└── Log files                  (Sessions logged)
```

### Layer 3: Transmission Flaws
- **STDIO pipes**: Unencrypted local communication
- **HTTP headers**: Session tokens in clear text
- **URL parameters**: Sessions in GET requests
- **WebSocket**: No TLS for local connections

## Vulnerability Patterns

### 1. Weak Session Generation
```python
# VULNERABLE: Predictable session IDs
def create_session():
    return f"session_{int(time.time())}_{user_id}"
    
# VULNERABLE: MD5 of timestamp
def create_session():
    return hashlib.md5(str(time.time()).encode()).hexdigest()
```

### 2. Missing Session Validation
```javascript
// VULNERABLE: No session expiry
sessions[sessionId] = {
    user: userId,
    created: Date.now()
    // No expiry field!
};

// VULNERABLE: No origin validation
function validateSession(sessionId) {
    return sessions[sessionId] !== undefined;
    // Doesn't check IP, user agent, etc.
}
```

### 3. Session Fixation
```go
// VULNERABLE: Accepts client-provided session ID
func handleInit(r *Request) {
    sessionID := r.Header.Get("Session-ID")
    if sessionID == "" {
        sessionID = generateSessionID()
    }
    // Uses attacker-provided session!
    sessions[sessionID] = newSession()
}
```

## Attack Scenarios

### Scenario 1: Local Session Hijacking
```bash
# Attacker on same machine
$ ps aux | grep mcp
mcp-server --session-file=/tmp/mcp-sessions.json

$ cat /tmp/mcp-sessions.json
{"abc123": {"user": "admin", "expires": null}}

# Attacker uses stolen session
$ mcp-client --session-id=abc123
> Full admin access achieved!
```

### Scenario 2: Network Sniffing
```
[Claude Client] ─────STDIO────> [MCP Server]
                      │
                 [Attacker sniffs]
                      │
                  Sees: {"session": "xyz789"}
                      │
                 Replays session
```

### Scenario 3: Cross-User Contamination
```python
# Multiple users on same MCP instance
User A: Creates session → abc123
User B: Creates session → def456

# Poor isolation allows:
User A: Can guess/access User B's session
User A: Send request with session: def456
Result: User A acts as User B
```

## Session Attack Techniques

### 1. **Session Prediction**
- Analyze session ID patterns
- Predict next valid session
- Brute force likely values

### 2. **Session Fixation**
- Set victim's session to known value
- Wait for victim to authenticate
- Use pre-set session

### 3. **Session Sidejacking**
- Sniff session tokens in transit
- Replay tokens before expiry
- Maintain persistent access

### 4. **Session Donation**
- Legitimate user shares session
- Attacker abuses shared access
- Original user unaware

## MCP-Specific Risks

### Persistent AI Context
```
Session contains:
- Conversation history
- Tool authorizations
- Resource permissions
- API credentials
- User preferences

Hijacking grants ALL of the above!
```

### Multi-Surface Impact
```
Hijacked MCP Session →
    ├── Access to AI conversations (Privacy breach)
    ├── Execute authorized tools (Command execution)
    ├── Access integrated services (Lateral movement)
    └── Steal credentials (Token harvesting)
```

## Amplification Factors

### 1. **No Standard Session Management**
MCP spec doesn't mandate session security, leaving implementations vulnerable

### 2. **Trust Assumptions**
Local MCP servers often assume local = trusted

### 3. **Stateful Operations**
MCP's stateful nature requires sessions, increasing attack surface

### 4. **Integration Complexity**
Sessions span multiple services, expanding compromise impact