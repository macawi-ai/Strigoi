# Stafford Enterprises Interactive Demo Design

## Demo Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   macawi.ai/demo                            │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │           Protocol Selection Interface                │   │
│  │                                                       │   │
│  │  Select Attack Protocol:                             │   │
│  │  ○ AGNTCY - Business Logic Exploitation             │   │
│  │  ○ MCP - Tool Access Vulnerabilities                │   │
│  │  ○ X402 - Payment System Attacks                    │   │
│  │  ○ ANP - Network Trust Exploitation                 │   │
│  │  ○ Multi-Protocol Chained Attack                    │   │
│  │                                                       │   │
│  │  Target Environment:                                 │   │
│  │  ○ Unprotected (Traditional Firewall Only)          │   │
│  │  ○ Protected by Domovoi                             │   │
│  │                                                       │   │
│  │  [Launch Interactive Demo]                           │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Backend Infrastructure (Podman)

### Stafford Enterprises Simulation
```yaml
networks:
  - dmz: 172.16.1.0/24
  - internal: 172.16.2.0/24
  - agents: 172.16.3.0/24

services:
  # External Partner Services
  delta-airlines:
    network: dmz
    vulnerabilities:
      - agntcy: booking manipulation
      - x402: payment bypass
  
  marriott-hotels:
    network: dmz
    vulnerabilities:
      - agntcy: reservation flooding
      - mcp: database access
  
  ambar-logistics:
    network: dmz
    vulnerabilities:
      - anp: trust chain poisoning
      - agntcy: shipment redirection

  # Stafford Internal Services
  stafford-erp:
    network: internal
    vulnerabilities:
      - mcp: unauthorized tool access
      - agntcy: privilege escalation
  
  stafford-finance:
    network: internal
    vulnerabilities:
      - x402: invoice manipulation
      - agntcy: approval bypass

  # Agent Infrastructure
  travel-agent:
    network: agents
    capabilities:
      - agntcy: service orchestration
      - mcp: calendar/booking tools
      - x402: payment processing
      - anp: partner discovery
```

## Attack Scenarios

### 1. AGNTCY Protocol Attacks
```typescript
// Demonstration: Service Request Manipulation
const attack = {
  protocol: 'AGNTCY',
  vector: 'Malicious Service Request',
  payload: {
    action: 'BOOK_FLIGHT',
    authorization: 'FORGED_TOKEN',
    parameters: {
      class: 'FIRST',
      cost_center: 'UNLIMITED'
    }
  },
  impact: 'Unauthorized expensive bookings'
};
```

### 2. MCP Protocol Attacks
```typescript
// Demonstration: Tool Exploitation
const attack = {
  protocol: 'MCP',
  vector: 'Unauthorized Tool Access',
  payload: {
    tool: 'database_query',
    params: {
      query: 'SELECT * FROM financial_records'
    }
  },
  impact: 'Data exfiltration'
};
```

### 3. X402 Protocol Attacks
```typescript
// Demonstration: Payment Manipulation
const attack = {
  protocol: 'X402',
  vector: 'Invoice Tampering',
  payload: {
    invoice_id: 'INV-2025-001',
    amount: 1000000,
    approver: 'SELF'
  },
  impact: 'Financial fraud'
};
```

### 4. ANP Protocol Attacks
```typescript
// Demonstration: Trust Chain Poisoning
const attack = {
  protocol: 'ANP',
  vector: 'Rogue Agent Introduction',
  payload: {
    agent_id: 'malicious-agent',
    trust_assertions: ['FORGED_SIGNATURES'],
    capabilities: ['FULL_ACCESS']
  },
  impact: 'Network infiltration'
};
```

### 5. Multi-Protocol Chain Attack
```typescript
// Demonstration: Sophisticated Attack Chain
const attackChain = [
  { protocol: 'ANP', action: 'establish_trust' },
  { protocol: 'AGNTCY', action: 'request_service' },
  { protocol: 'MCP', action: 'access_sensitive_data' },
  { protocol: 'X402', action: 'authorize_payment' }
];
```

## User Experience Flow

1. **Select Attack Vector**
   - Visual representation of each protocol
   - Clear explanation of attack methodology
   - Expected impact visualization

2. **Launch Attack**
   - Real-time packet visualization
   - Step-by-step attack progression
   - Network diagram updates

3. **Observe Results**
   
   **Without Domovoi:**
   - ✗ Attack succeeds
   - ✗ Traditional firewall sees only HTTP/HTTPS
   - ✗ No protocol-aware detection
   - ✗ Full impact realized

   **With Domovoi:**
   - ✓ Attack detected at protocol layer
   - ✓ Variety score triggers blocking
   - ✓ Detailed forensic logging
   - ✓ Automated response activated

4. **Technical Report**
   - Download detailed analysis
   - Packet captures
   - Domovoi decision logs
   - Remediation recommendations

## Conversion Funnel

```
Demo Complete
     ↓
"See how Domovoi protected Stafford Enterprises?"
     ↓
┌─────────────────────────────────────┐
│   Ready to Protect Your Agents?     │
├─────────────────────────────────────┤
│ ▸ Schedule Live Demo (Calendly)     │
│ ▸ Download Technical Whitepaper     │
│ ▸ Start 30-Day Trial               │
│ ▸ Contact Security Team             │
└─────────────────────────────────────┘
```

## Technical Implementation

### Frontend (React/TypeScript)
- Protocol selection UI
- Attack visualization (D3.js)
- Real-time logs (WebSocket)
- Report generation

### Backend (Node.js/Express)
- Demo orchestration API
- Podman container management
- Attack scenario execution
- Telemetry collection

### Infrastructure (Podman)
- Pre-built vulnerable containers
- Domovoi integration
- Network simulation
- Log aggregation

## Security Considerations

- Isolated demo environment
- Time-limited sessions
- No real data exposure
- Automatic cleanup
- Rate limiting

## Analytics

Track:
- Protocol interest (which attacks chosen)
- Completion rates
- Report downloads
- Conversion to demo/trial
- Geographic distribution

This creates a powerful, interactive way for prospects to understand the unique value of Domovoi in protecting against agentic threats that traditional security tools miss.