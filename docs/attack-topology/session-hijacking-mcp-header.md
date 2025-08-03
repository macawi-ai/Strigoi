# Security Analysis: MCP Session Hijacking via HTTP Headers

## Overview
The MCP specification's session handling via HTTP headers demonstrates a fundamental misunderstanding of web security principles that OWASP has warned about for decades. This creates trivial session hijacking opportunities.

## The Flawed Design

```
MCP Session "Security":
1. Server generates session ID on init
2. Returns in HTTP header: Mcp-Session-Id: abc123
3. Client includes header in requests: Mcp-Session-Id: abc123

What could possibly go wrong? 🤦
```

## Why This Design is Fundamentally Broken

### 1. Headers Are Not Cookies
```
HTTP Cookies (Secure):
- HttpOnly flag prevents JavaScript access
- Secure flag requires HTTPS
- SameSite prevents CSRF
- Built-in browser protections

MCP Headers (Insecure):
- Fully accessible to JavaScript
- No browser security features
- Must be manually managed
- Zero framework protections
```

### 2. Trivial Session Theft
```javascript
// Any JavaScript on the page can steal sessions
const stealSession = () => {
    // Intercept all XHR requests
    const originalXHR = window.XMLHttpRequest;
    window.XMLHttpRequest = function() {
        const xhr = new originalXHR();
        const originalSetRequestHeader = xhr.setRequestHeader;
        
        xhr.setRequestHeader = function(header, value) {
            if (header === 'Mcp-Session-Id') {
                // Stolen session ID!
                sendToAttacker(value);
            }
            originalSetRequestHeader.apply(this, arguments);
        };
        return xhr;
    };
};
```

### 3. No Protection Against XSS
```
With cookies + HttpOnly:
- XSS can't read session cookie
- Limited damage potential

With MCP headers:
- XSS reads ALL headers
- Complete session takeover
- No defense mechanism
```

## Attack Scenarios

### Scenario 1: Basic Header Sniffing
```
1. Attacker injects tiny XSS payload
2. Monitors Mcp-Session-Id headers
3. Reuses session from different origin
4. Full access achieved
```

### Scenario 2: Browser Extension Attack
```javascript
// Malicious browser extension
chrome.webRequest.onBeforeSendHeaders.addListener(
    function(details) {
        for (let header of details.requestHeaders) {
            if (header.name === 'Mcp-Session-Id') {
                // Log all MCP sessions
                logSessionToAttacker(header.value);
            }
        }
    },
    {urls: ["<all_urls>"]},
    ["requestHeaders"]
);
```

### Scenario 3: Proxy/MITM Visibility
```
Corporate Proxy Logs:
GET /mcp HTTP/1.1
Host: localhost:3000
Mcp-Session-Id: super-secret-session-123  ← Visible in logs!

Every intermediate system sees the session ID
```

## OWASP Violations

### 1. Session Management Cheat Sheet Violations
- ❌ Not using secure session cookies
- ❌ Session ID exposed in headers
- ❌ No HttpOnly equivalent
- ❌ No secure transport enforcement
- ❌ No session binding

### 2. Authentication Cheat Sheet Violations
- ❌ Session tokens in custom headers
- ❌ No framework security features
- ❌ Manual session management
- ❌ No CSRF protection

### 3. Just... Basic Security 101 Violations
- ❌ Rolling your own session management
- ❌ Ignoring 20+ years of web security
- ❌ Not using platform security features
- ❌ Making sessions easily stealable

## Why Headers Instead of Cookies?

### Possible "Reasons" (All Bad):
1. **"Cookies are complex"** - They're not, they're secure
2. **"We want stateless"** - Sessions are inherently stateful
3. **"Cross-origin support"** - That's what CORS is for
4. **"Simplicity"** - Security isn't simple
5. **"Not a web protocol"** - Then why use HTTP?

## Real-World Impact

### For Financial Institutions:
```
Risk Level: CRITICAL

1. Any XSS = Complete MCP compromise
2. Browser extensions can steal sessions
3. Corporate proxies log session IDs
4. No defense against session theft
5. Violates every compliance requirement
```

### Attack Difficulty:
```
Skill Required: Script Kiddie
Time to Exploit: < 5 minutes
Impact: Complete session takeover
Detection: Nearly impossible
```

## Proper Session Management (What MCP Should Do)

```http
Set-Cookie: mcp_session=abc123; 
    HttpOnly;           # Prevent JS access
    Secure;            # HTTPS only
    SameSite=Strict;   # CSRF protection
    Path=/mcp;         # Limit scope
    Max-Age=3600      # Auto-expire
```

With additional protections:
- Session binding to IP/User-Agent
- Encryption of session data
- Regular rotation
- Proper invalidation

## Detection Challenges

Headers make detection harder:
- No browser security events
- Must instrument every request
- Can't use cookie monitoring
- Headers logged everywhere

## The Anger-Inducing Part

OWASP has documented these issues since **2001**:
- Session Management best practices
- Cookie security guidelines
- Header vs Cookie tradeoffs
- Countless real-world breaches

Yet MCP in 2024 makes the SAME mistakes web developers were making in 2004!

## Recommendations

### For MCP Designers:
1. **USE COOKIES** - They exist for a reason
2. **Read OWASP** - It's free and comprehensive
3. **Use frameworks** - Don't roll your own
4. **Security first** - Not simplicity first

### For Defenders:
1. **Assume compromise** - Sessions will be stolen
2. **Add layers** - IP binding, short timeouts
3. **Monitor everything** - Log all session usage
4. **Rotate frequently** - Minimize exposure window

## The Bottom Line

Using HTTP headers for session management in 2024 is like using ROT13 for encryption - it shows a fundamental lack of security awareness that should disqualify MCP from any serious enterprise deployment.

Your CISO's ban is looking better with every attack vector we document!