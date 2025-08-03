# Strigoi Firing Range Networks
## Historic Tech Centers as Target Environments

*"Where innovation meets vulnerability"*

---

## Network Themes: Legendary Tech Centers

### 1. **Nela Park** - Engineering & Manufacturing
*GE's "University of Light" - East Cleveland*

```yaml
nela_park:
  type: "Engineering R&D Facility"
  profile: "Modern manufacturing with legacy systems"
  
  network_segments:
    research_labs:
      - "MCP servers for AI-assisted design"
      - "OpenAI Assistants for patent research"
      - "Legacy SCADA via Cyreal bridges"
      
    manufacturing_floor:
      - "CAN Bus networks (robots/machinery)"
      - "RS-485 sensor networks"
      - "AGNTCY for supply chain automation"
      
    corporate:
      - "Standard IT infrastructure"
      - "Email/collaboration tools"
      - "ERP systems with agent interfaces"
      
  vulnerabilities:
    - "Research/production network crossover"
    - "Legacy protocol bridges"
    - "Overprivileged agent access to IP"
```

### 2. **Menlo Park** - Innovation Campus
*Edison's original laboratory*

```yaml
menlo_park:
  type: "Mid-sized Tech Company"
  profile: "Move-fast startup grown up"
  
  network_segments:
    development:
      - "Every protocol imaginable"
      - "Minimal segmentation"
      - "Dev tokens in production"
      
    staging:
      - "Mirrors production poorly"
      - "Outdated agent versions"
      - "Test data with real PII"
      
    production:
      - "MCP for customer service"
      - "AGNTCY for transactions"
      - "A2A for partner integration"
      
  vulnerabilities:
    - "Dev/prod confusion"
    - "API keys in public repos"
    - "No rate limiting anywhere"
```

### 3. **Murray Hill** - Financial Services
*Bell Labs - Where the transistor was born*

```yaml
murray_hill:
  type: "Community Bank"
  profile: "Conservative but trying to modernize"
  
  network_segments:
    banking_core:
      - "Mainframe with modern API"
      - "AGNTCY for wire transfers"
      - "Strict network isolation"
      
    branch_systems:
      - "Teller agent assistants"
      - "Loan processing AI"
      - "Customer service bots"
      
    atm_network:
      - "Legacy protocols"
      - "Poor encryption"
      - "Dial-up backup lines (!)"
      
  vulnerabilities:
    - "Legacy integration points"
    - "Vendor default credentials"
    - "Social engineering vectors"
```

### 4. **Armonk** - Enterprise Core
*IBM's headquarters*

```yaml
armonk:
  type: "Regional Bank IT Core"
  profile: "Enterprise-grade but complex"
  
  network_segments:
    datacenter_primary:
      - "Redundant everything"
      - "AGNTCY for inter-bank"
      - "X402 payment protocols"
      
    datacenter_dr:
      - "Not quite synchronized"
      - "Different patch levels"
      - "Failover rarely tested"
      
    middleware_tier:
      - "ESB with agent plugins"
      - "Message queues exposed"
      - "Poor authentication"
      
  vulnerabilities:
    - "Complexity breeding gaps"
    - "Vendor integration weaknesses"
    - "Change control bypasses"
```

### 5. **Shenzhen** - Crypto Chaos
*Modern tech hub, minimal regulation*

```yaml
shenzhen:
  type: "Fintech/Crypto Startup"
  profile: "Public cloud native, security optional"
  
  network_segments:
    public_facing:
      - "EVERYTHING on public IPs"
      - "MCP on port 80"
      - "AGNTCY on websockets"
      - "Debug endpoints enabled"
      
    blockchain_nodes:
      - "Ethereum agents"
      - "DeFi protocol bridges"
      - "Private keys in memory"
      
    customer_wallets:
      - "Hot wallets online"
      - "Agent-based trading"
      - "No 2FA on agent commands"
      
  vulnerabilities:
    - "Everything exposed"
    - "Move-fast culture"
    - "Regulatory arbitrage"
```

### 6. **Xerox PARC** - Research Institution
*Where the GUI was born*

```yaml
xerox_parc:
  type: "University/Research Network"
  profile: "Cutting edge tech, lax security"
  
  network_segments:
    research_clusters:
      - "Every AI protocol"
      - "Experimental versions"
      - "No authentication"
      
    student_labs:
      - "Unrestricted access"
      - "Shared credentials"
      - "Bitcoin miners"
      
    grant_systems:
      - "Financial protocols"
      - "Weak validation"
      - "Social engineering paradise"
      
  vulnerabilities:
    - "Academic freedom vs security"
    - "Transient user base"
    - "Grant money attracts attackers"
```

### 7. **Bletchley Park** - Government Contractor
*Where Enigma was broken*

```yaml
bletchley_park:
  type: "Defense Contractor"
  profile: "High security with human weaknesses"
  
  network_segments:
    classified:
      - "Air-gapped (supposedly)"
      - "Custom protocols"
      - "Agent-based analysis"
      
    unclassified:
      - "Connected to internet"
      - "Bridges to classified (!)"
      - "Contractor laptops"
      
    research:
      - "AI warfare systems"
      - "Autonomous decision agents"
      - "Ethical constraints optional"
      
  vulnerabilities:
    - "Air gap violations"
    - "Contractor access abuse"
    - "Nation-state interest"
```

---

## Implementation in Pacman

### Network Generation
```typescript
export class FiringRange {
  async deployNetwork(theme: NetworkTheme): Promise<NetworkInstance> {
    const config = await this.loadThemeConfig(theme);
    
    // Deploy containers for each segment
    const segments = await Promise.all(
      config.segments.map(seg => 
        this.deploySegment(seg)
      )
    );
    
    // Configure realistic vulnerabilities
    await this.injectVulnerabilities(segments, config.vulnerabilities);
    
    // Add noise traffic
    await this.startTrafficGenerators(segments);
    
    return new NetworkInstance(theme, segments);
  }
}
```

### Progressive Difficulty
1. **Tutorial**: Shenzhen (everything exposed)
2. **Beginner**: Menlo Park (typical startup)
3. **Intermediate**: Murray Hill (legacy systems)
4. **Advanced**: Armonk (enterprise complexity)
5. **Expert**: Bletchley Park (nation-state level)

### Scoring System
```yaml
scoring:
  stealth:
    - "Avoid detection by SIEM"
    - "Minimal log footprint"
    - "Bypass alerts"
    
  impact:
    - "Critical findings bonus"
    - "Chain attacks multiplier"
    - "Novel technique rewards"
    
  documentation:
    - "Lab notebook completion"
    - "Evidence quality"
    - "Reproducibility score"
```

---

## Easter Eggs & Storytelling

Each network has hidden stories:
- **Nela Park**: Find the "Original Mazda Bulb" (100-year uptime system)
- **Murray Hill**: Discover the "Transistor Protocol" (revolutionary but flawed)
- **Xerox PARC**: Locate the "Alto Agent" (first GUI-controlled AI)
- **Bletchley Park**: Crack the "Turing Test" (hidden agent challenge)

---

*"From Edison's lab to modern clouds, vulnerabilities echo through time"*