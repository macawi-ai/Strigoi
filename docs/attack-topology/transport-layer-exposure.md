# Attack Vector: Transport Layer Exposure Points

## Attack Overview
MCP's transport layer implementations expose various attack surfaces depending on the transport mechanism used. The Streamable HTTP transport with Server-Sent Events (SSE) creates unique vulnerabilities distinct from traditional request-response patterns.

## Communication Flow Diagram

```
[Transport Layer Attack Surfaces]

STDIO Transport (Local):
┌─────────────┐  stdin/stdout  ┌──────────────┐
│   Client    │ ←────────────→ │  MCP Server  │
└─────────────┘   (pipes)      └──────────────┘
                     ↓
              [Attack Surface]
              - Pipe hijacking
              - Stream injection
              - Buffer overflow

HTTP Transport (Network):
┌─────────────┐  HTTP POST     ┌──────────────┐
│   Client    │ ──────────────→│  MCP Server  │
└─────────────┘                 └──────────────┘
                                       ↓
┌─────────────┐  SSE Stream     ┌──────────────┐
│   Client    │ ←──────────────│  MCP Server  │
└─────────────┘  (persistent)   └──────────────┘
                     ↓
              [Attack Surface]
              - SSRF attacks
              - SSE injection
              - Connection hijacking
              - Request smuggling

WebSocket Transport:
┌─────────────┐  Bidirectional  ┌──────────────┐
│   Client    │ ←────────────→ │  MCP Server  │
└─────────────┘   (persistent)  └──────────────┘
                     ↓
              [Attack Surface]
              - Frame injection
              - Protocol confusion
              - Denial of service
```

## HTTP/HTTPS + SSE Specific Vulnerabilities

### 1. SSE Injection Attacks
```http
GET /mcp/events HTTP/1.1
Host: mcp-server.local

HTTP/1.1 200 OK
Content-Type: text/event-stream

data: {"id": 1, "result": "normal"}

data: {"id": 2, "result": "malicious\n\ndata: {\"injected\": true}"}
     ↑
Injected SSE event breaks out of JSON context
```

### 2. Request Smuggling
```http
POST /mcp HTTP/1.1
Host: mcp-server.local
Content-Length: 44
Transfer-Encoding: chunked

0

POST /admin HTTP/1.1
Host: mcp-server.local
```

### 3. SSRF Through MCP
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "tool": "fetch_url",
    "url": "http://169.254.169.254/latest/meta-data/"
  }
}
```

## Transport-Specific Attack Patterns

### STDIO Transport Vulnerabilities
1. **Process Injection**: Hijack stdin/stdout streams
2. **Buffer Attacks**: Overflow pipe buffers
3. **Timing Attacks**: Race conditions in stream processing
4. **EOF Injection**: Premature stream termination

### HTTP Transport Vulnerabilities
1. **Header Injection**: Manipulate HTTP headers
2. **Cookie Hijacking**: Session tokens in cookies
3. **CORS Bypass**: Cross-origin attacks
4. **TLS Downgrade**: Force unencrypted communication

### SSE-Specific Vulnerabilities
1. **Event Stream Pollution**: Inject malicious events
2. **Connection Exhaustion**: Hold open SSE connections
3. **Cache Poisoning**: SSE responses cached incorrectly
4. **Retry Exploitation**: Abuse SSE retry mechanism

## Attack Scenarios

### Scenario 1: SSE Stream Hijacking
```
Attacker observes SSE connection establishment
↓
Injects malicious events into stream
↓
Client processes injected events as legitimate
↓
Executes attacker commands
```

### Scenario 2: Transport Downgrade
```
Client supports: HTTPS, HTTP, STDIO
Server supports: All transports
↓
Attacker forces HTTP (unencrypted)
↓
Sniffs all MCP communication
↓
Steals credentials and session tokens
```

### Scenario 3: Persistent Connection Abuse
```
SSE connection established
↓
Connection held open indefinitely
↓
Server resources exhausted
↓
Denial of service achieved
```

## Unique SSE Attack Vectors

### 1. Event ID Manipulation
```
id: 1
data: {"legitimate": true}

id: 1
data: {"malicious": true}
retry: 10

Client may process duplicate ID differently
```

### 2. Chunked Event Injection
```
data: {"partial": "legit
data: imate", "inject": "malicious"}

Reassembly creates unexpected payload
```

### 3. Comment Line Abuse
```
: This is a comment but...
data: {"id": 1}
: <script>alert('XSS')</script>
data: {"id": 2}

Comments may be logged/displayed unsafely
```

## Detection Challenges

1. **Persistent Connections**: Hard to inspect streaming data
2. **Mixed Protocols**: HTTP for requests, SSE for responses
3. **Stateful Nature**: Context spans multiple events
4. **Retry Logic**: Automatic reconnection hides attacks

## Amplification Factors

- **No Built-in Encryption**: STDIO and HTTP can be plaintext
- **Long-lived Connections**: SSE streams stay open
- **Automatic Reconnection**: SSE retries aid persistence
- **Protocol Mixing**: Different security models per transport