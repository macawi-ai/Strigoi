# Attack Vector: Confused Deputy Attack in MCP Proxy Servers

## Attack Overview
The confused deputy problem occurs when an MCP server acts as a proxy to other services using its own credentials rather than properly delegating user authorization. The MCP server becomes a "confused deputy" - performing actions on behalf of users without proper authorization checks.

## Communication Flow Diagram

```
[Normal Flow - How it Should Work]

┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│   Claude    │ ←─────→ │  MCP Server  │ ←─────→ │  Resource    │
│   Client    │  STDIO  │   (proxy)    │  OAuth  │   Server     │
└─────────────┘         └──────────────┘         └──────────────┘
      |                        |                         |
      | User: "Access my       | "Acting as User A"     |
      | Google Drive"          | Token: user_a_token    |
      └───────────────────────→└───────────────────────→|
                                                         |
                                                   Grants access
                                                   to User A's files

[Confused Deputy Attack - What Actually Happens]

┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│  Attacker   │ ←─────→ │  MCP Server  │ ←─────→ │  Resource    │
│   Client    │  STDIO  │   (proxy)    │  OAuth  │   Server     │
└─────────────┘         └──────────────┘         └──────────────┘
      |                        |                         |
      | "Access User B's       | Static Client ID:      |
      | Google Drive"          | mcp_proxy_12345        |
      |                        | No user distinction!   |
      └───────────────────────→└───────────────────────→|
                                                         |
                                                   Grants access
                                                   to ANY user's files
                                                   authorized to this
                                                   client ID!

[Attack Amplification]

┌─────────────┐
│  Attacker   │ "List all accessible drives"
└─────────────┘
      |
      ↓
┌──────────────┐ Uses single static credential
│  MCP Proxy   │ for ALL users
└──────────────┘
      |
      ├────→ User A's Drive
      ├────→ User B's Drive  
      ├────→ User C's Drive
      └────→ Corporate Drive
```

## Attack Layers

### Layer 1: Authorization Confusion
- **Static Client IDs**: MCP proxy uses one OAuth client ID for all users
- **No User Binding**: Requests don't maintain user context
- **Credential Reuse**: Same service account for multiple users
- **Trust Assumption**: Resource server trusts the MCP proxy

### Layer 2: Privilege Escalation Path
```
1. MCP Server registers as OAuth client → Gets client_id: "mcp_proxy_12345"
2. User A authorizes MCP to access their Google Drive
3. User B authorizes MCP to access their Google Drive  
4. MCP Server now has refresh tokens for both users
5. Attacker connects to MCP Server
6. MCP Server can't distinguish which user is requesting
7. Attacker gains access to all authorized resources
```

### Layer 3: Attack Variations

#### Horizontal Privilege Escalation
```
Attacker (User A) → MCP Proxy → Access User B's resources
                         ↓
                  "I'm authorized client mcp_proxy_12345"
                  "Give me files from any authorized user"
```

#### Vertical Privilege Escalation
```
Low-privilege user → MCP Proxy → Admin resources
                          ↓
                   "The proxy has admin access"
                   "So now I do too"
```

## Vulnerable Patterns

### 1. Service Account Confusion
```python
class MCPProxy:
    def __init__(self):
        # VULNERABLE: Single service account for all users
        self.service_account = load_service_account()
    
    def access_resource(self, resource_id):
        # No user context validation!
        return self.service_account.get(resource_id)
```

### 2. Token Pool Mixing
```python
# VULNERABLE: All tokens in shared pool
token_pool = {
    "drive": ["user_a_token", "user_b_token", "admin_token"],
    "calendar": ["user_c_token", "user_d_token"]
}

def get_resource(service, resource):
    # Uses ANY available token!
    token = token_pool[service][0]
    return fetch_with_token(token, resource)
```

### 3. Missing Authorization Context
```
MCP Client Request: "Get file X from Drive"
                           ↓
MCP Proxy: "I'll use my Drive access" (Wrong!)
                           ↓
Should be: "I'll use YOUR Drive access"
```

## Real-World Impacts

### Data Access Violations
- Access other users' files, emails, calendars
- Read corporate documents without authorization
- Modify resources belonging to other users

### Compliance Violations
- GDPR: Unauthorized data access
- HIPAA: Medical record exposure
- SOX: Financial data breach

### Attack Scenarios

1. **Corporate Espionage**: Access competitor's files through shared MCP proxy
2. **Insider Threat**: Low-level employee accesses executive resources
3. **Data Exfiltration**: Bulk download all accessible resources
4. **Privilege Persistence**: Maintain access through proxy even after direct access revoked

## Detection Challenges

- Requests appear legitimate from resource server perspective
- MCP proxy has valid authorization tokens
- No audit trail distinguishing user requests
- Resource server trusts the confused deputy

## Amplification Factors

- **Multi-tenancy**: One MCP server serving multiple users/organizations
- **Token Accumulation**: More users = more accessible resources
- **Long-lived Tokens**: Refresh tokens provide persistent access
- **Implicit Trust**: Resource servers trust the registered OAuth client