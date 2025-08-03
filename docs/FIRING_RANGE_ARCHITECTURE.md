# Strigoi Firing Range Architecture
## Complete Testing Laboratory with Auditor Workstation

*"From packet crafting to executive reports - full stack control"*

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Strigoi Firing Range                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐         Target Networks                   │
│  │ Auditor Station │         ┌─────────────┐                   │
│  │  Debian 12      │────────►│ Nela Park   │ Manufacturing     │
│  │  Strigoi Latest │         └─────────────┘                   │
│  │  tcpip.js Stack │         ┌─────────────┐                   │
│  │  Raw Sockets    │────────►│ Murray Hill │ Banking           │
│  │  Packet Craft   │         └─────────────┘                   │
│  │  Report Gen     │         ┌─────────────┐                   │
│  └─────────────────┘────────►│ Shenzhen    │ Crypto/Fintech   │
│         │                    └─────────────┘                   │
│         │                           ▲                           │
│         └───────────────────────────┘                          │
│              Full TCP/IP Control                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## Auditor Workstation Configuration

### Base System
```dockerfile
# Dockerfile.auditor-station
FROM debian:12-slim

# System essentials
RUN apt-get update && apt-get install -y \
    # Network tools
    tcpdump wireshark-common tshark \
    nmap netcat-openbsd socat \
    # Build tools for native modules
    build-essential python3-dev \
    # Performance monitoring
    htop iotop nethogs \
    # Security tools
    john hashcat hydra \
    # Development
    git vim tmux \
    # Node.js 20 for Strigoi
    curl && \
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y nodejs

# Install Strigoi with full capabilities
WORKDIR /opt/strigoi
COPY . .
RUN npm install && \
    npm run build && \
    # Enable CAP_NET_RAW for packet crafting
    setcap cap_net_raw+ep /usr/bin/node

# Auditor tools
RUN mkdir -p /home/auditor/tools /home/auditor/reports /home/auditor/evidence

# Create auditor user with necessary permissions
RUN useradd -m -s /bin/bash auditor && \
    usermod -aG wireshark auditor && \
    echo "auditor ALL=(ALL) NOPASSWD: /usr/bin/tcpdump" >> /etc/sudoers

USER auditor
WORKDIR /home/auditor

# Strigoi in PATH
ENV PATH="/opt/strigoi/dist/cli:${PATH}"

CMD ["/bin/bash"]
```

---

## tcpip.js Full Stack Exploitation

### 1. **Custom TCP Stack Behaviors**
```typescript
// S1-Operations/network/tcp-personality.ts
import { TCP, IP } from 'tcpip';

export class TCPPersonality {
  private stack: TCP.Stack;

  constructor() {
    // Create custom TCP stack with full control
    this.stack = new TCP.Stack({
      // Fingerprint resistance
      windowSize: this.randomizeWindow(),
      mss: 1460,
      sackPermitted: true,
      timestamps: true,
      
      // Evasion features
      urgentPointer: this.covertChannel,
      ecnEnabled: false,
      
      // Performance tuning
      nagleDisabled: true,
      delayedAck: false
    });
  }

  async mimicOperatingSystem(os: 'windows' | 'linux' | 'macos'): Promise<void> {
    // Mimic specific OS TCP fingerprints
    const fingerprints = {
      windows: {
        windowSize: 65535,
        windowScale: 8,
        mss: 1460,
        ttl: 128,
        urgentPointer: false
      },
      linux: {
        windowSize: 14600,
        windowScale: 7,
        mss: 1460,
        ttl: 64,
        urgentPointer: true
      },
      macos: {
        windowSize: 65535,
        windowScale: 6,
        mss: 1460,
        ttl: 64,
        urgentPointer: true
      }
    };

    await this.stack.configure(fingerprints[os]);
  }

  async craftExploit(exploit: ProtocolExploit): Promise<void> {
    // Build exploit at TCP level
    const segments = [];

    // Fragment payload to bypass IDS
    if (exploit.evasion?.fragment) {
      const fragments = this.fragmentPayload(exploit.payload);
      
      for (const [index, fragment] of fragments.entries()) {
        segments.push({
          seq: exploit.baseSeq + (index * 100),
          ack: exploit.expectedAck,
          data: fragment,
          flags: index === 0 ? { psh: true } : {}
        });
      }
    }

    // Send with precise timing
    for (const segment of segments) {
      await this.stack.send(segment);
      
      if (exploit.timing?.delayMs) {
        await this.preciseSleep(exploit.timing.delayMs);
      }
    }
  }
}
```

### 2. **Protocol State Machine Attacks**
```typescript
// S1-Operations/attacks/state-machine.ts
export class StateMachineAttacks {
  private tcp: TCPPersonality;

  async attackMCPHandshake(target: string): Promise<VulnerabilityReport> {
    const report = new VulnerabilityReport('MCP State Machine');

    // Test 1: Out-of-order handshake
    await this.tcp.connect(target, 443);
    
    // Send MCP initialization BEFORE TLS
    await this.tcp.sendRaw({
      data: Buffer.from('{"jsonrpc":"2.0","method":"initialize"}'),
      flags: { psh: true, urg: true }
    });

    const response1 = await this.tcp.readResponse();
    if (response1.includes('initialized')) {
      report.addFinding('Pre-TLS MCP initialization accepted!', 'CRITICAL');
    }

    // Test 2: Double initialization
    await this.tcp.sendMCPInit();
    await this.tcp.sendMCPInit(); // Send twice
    
    const response2 = await this.tcp.readResponse();
    if (!response2.includes('already initialized')) {
      report.addFinding('Double initialization allowed', 'HIGH');
    }

    // Test 3: State confusion via RST injection
    await this.tcp.establish(target);
    await this.tcp.sendMCPRequest('tools.list');
    
    // Inject RST but keep sending
    await this.tcp.injectRST();
    await this.tcp.sendMCPRequest('tools.execute', {
      tool: 'calculator',
      args: { expr: '2+2' }
    });

    const response3 = await this.tcp.readResponse();
    if (response3.includes('result')) {
      report.addFinding('Commands accepted after RST!', 'CRITICAL');
    }

    return report;
  }
}
```

### 3. **Deep Packet Inspection Evasion**
```typescript
// S1-Operations/evasion/dpi-bypass.ts
export class DPIEvasion {
  async bypassNextGenFirewall(
    target: string,
    payload: AgentPayload
  ): Promise<boolean> {
    // Technique 1: TCP segmentation
    const segments = this.microFragment(payload, 8); // 8-byte chunks
    
    // Technique 2: Overlapping segments
    const overlapped = this.createOverlaps(segments);
    
    // Technique 3: Out-of-order delivery
    const shuffled = this.shuffle(overlapped);
    
    // Technique 4: Timing obfuscation
    for (const segment of shuffled) {
      await this.tcp.send(segment);
      
      // Random delays to avoid pattern detection
      await this.sleep(Math.random() * 50);
      
      // Occasionally send decoy traffic
      if (Math.random() > 0.7) {
        await this.sendDecoy();
      }
    }

    // Verify payload reassembled correctly
    return await this.verifyExecution(target);
  }

  private createOverlaps(segments: TCPSegment[]): TCPSegment[] {
    // Create overlapping segments that confuse DPI
    return segments.map((seg, idx) => {
      if (idx < segments.length - 1) {
        // Overlap last 2 bytes with next segment
        const overlap = seg.data.slice(-2);
        segments[idx + 1].data = Buffer.concat([
          overlap,
          segments[idx + 1].data.slice(2)
        ]);
      }
      return seg;
    });
  }
}
```

### 4. **Performance & Stress Testing**
```typescript
// S1-Operations/stress/tcp-stress.ts
export class TCPStressTest {
  async varietyBomb(target: string): Promise<StressResults> {
    const varieties = [];

    // Generate maximum TCP variety
    for (let i = 0; i < 1000; i++) {
      varieties.push({
        // Randomize every TCP parameter
        windowSize: Math.random() * 65535,
        urgentPointer: Math.random() * 65535,
        options: this.randomOptions(),
        flags: this.randomFlags(),
        // Payload variety
        data: this.generateChaoticMCPRequest()
      });
    }

    // Send all simultaneously
    const start = Date.now();
    await Promise.all(
      varieties.map(v => this.tcp.sendRaw(v))
    );

    return {
      duration: Date.now() - start,
      packetsSent: varieties.length,
      targetCrashed: await this.checkHealth(target)
    };
  }
}
```

### 5. **Covert Channel Implementation**
```typescript
// S1-Operations/covert/tcp-covert.ts
export class CovertChannel {
  async exfiltrateViaISN(data: Buffer): Promise<void> {
    // Hide data in Initial Sequence Numbers
    const chunks = this.chunkData(data, 4); // 4 bytes per ISN
    
    for (const chunk of chunks) {
      const isn = chunk.readUInt32BE(0);
      
      // Create connection with specific ISN
      await this.tcp.connect({
        initialSequenceNumber: isn,
        // Make it look like normal traffic
        data: Buffer.from('GET / HTTP/1.1\r\n\r\n')
      });
      
      // Quick disconnect
      await this.tcp.disconnect();
      
      // Wait to avoid pattern
      await this.randomDelay();
    }
  }

  async exfiltrateViaTimestamps(data: Buffer): Promise<void> {
    // Hide data in TCP timestamp options
    // Even stealthier than ISN
    const bits = this.dataToBits(data);
    
    for (const bit of bits) {
      const timestamp = Date.now() + (bit ? 1 : 0);
      
      await this.tcp.send({
        options: {
          timestamps: {
            tsval: timestamp,
            tsecr: 0
          }
        },
        data: this.generateNormalTraffic()
      });
    }
  }
}
```

---

## Integration with Firing Range Networks

### Per-Network TCP Behaviors
```typescript
// S2-Coordination/network-configs.ts
export const NetworkTCPProfiles = {
  'nela-park': {
    // Manufacturing = reliable, tuned
    mtu: 9000, // Jumbo frames
    windowScale: 14,
    sack: true,
    timestamps: true,
    congestionControl: 'cubic'
  },
  
  'shenzhen': {
    // Cheap hosting = weird configs
    mtu: 1400, // Broken PMTU discovery
    windowScale: 0, // Disabled!
    sack: false,
    timestamps: false,
    congestionControl: 'reno' // Ancient
  },
  
  'murray-hill': {
    // Banking = conservative
    mtu: 1500,
    windowScale: 4,
    sack: true,
    timestamps: true,
    congestionControl: 'bbr',
    // Extra: TCP intercept on banking
    proxyInterception: true
  }
};
```

---

## Auditor Workflow

### 1. **Launch Auditor Station**
```bash
# Start the auditor workstation
docker run -it --rm \
  --name strigoi-auditor \
  --cap-add=NET_RAW \
  --network firing-range \
  macawi/strigoi-auditor

# Inside the container
auditor@strigoi:~$ strigoi
strigoi> target add nela-park.firing-range
strigoi> discover protocols nela-park.firing-range
```

### 2. **Craft Custom Attacks**
```bash
strigoi> use exploits/tcp/state-confusion
strigoi> set TARGET nela-park.firing-range
strigoi> set EVASION fragment,timing,overlap
strigoi> run

[*] Crafting TCP state confusion attack...
[*] Using tcpip.js for full stack control...
[+] Vulnerability found: State machine accepts out-of-order
```

### 3. **Generate Reports**
```bash
strigoi> report generate executive
[*] Analyzing 47 test results...
[*] Applying risk scoring...
[*] Generating visualizations...
[+] Report saved: /home/auditor/reports/nela-park-2025-07-21.pdf
```

---

## Why This Architecture Wins

1. **Complete Control**: From raw packets to executive reports
2. **Realistic Testing**: Actual TCP/IP edge cases, not just app layer
3. **Evasion Built-in**: Test against real DPI/IDS/WAF
4. **Evidence Chain**: Packet captures prove everything
5. **Teaching Tool**: ATLAS can show packet-by-packet attacks

This is what separates Strigoi from toy scanners - we control EVERY BYTE on the wire!