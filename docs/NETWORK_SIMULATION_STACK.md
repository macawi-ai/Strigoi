# Network Simulation Stack for Strigoi Firing Range
## Container-Based Complex Network Topologies

*"From simple containers to enterprise-scale networks"*

---

## Simulation Approaches

### 1. **Containerlab** (Personal Favorite)
*Network topology orchestration for containers*

```yaml
# topology/nela-park.clab.yml
name: nela-park-manufacturing

topology:
  nodes:
    # Core router (using FRRouting)
    core-router:
      kind: linux
      image: frrouting/frr:latest
      binds:
        - configs/core-router:/etc/frr
    
    # Manufacturing SCADA network
    scada-switch:
      kind: bridge
      
    plc-1:
      kind: linux
      image: macawi/industrial-sim:plc
      env:
        PROTOCOL: modbus
        
    hmi-server:
      kind: linux
      image: macawi/industrial-sim:hmi
      env:
        PROTOCOLS: "opcua,mcp"
    
    # Corporate network
    corp-firewall:
      kind: linux
      image: macawi/firewall-sim:pfsense
      
    agent-server:
      kind: linux
      image: macawi/agent-server:latest
      env:
        PROTOCOLS: "agntcy,mcp,openai"
        VULNERABILITIES: "true"
    
    # Links with realistic latency/loss
    links:
      - endpoints: ["core-router:eth1", "scada-switch:eth1"]
        mtu: 9000  # Jumbo frames
      - endpoints: ["scada-switch:eth2", "plc-1:eth0"]
        loss: 0.1%  # Industrial noise
        delay: 2ms
      - endpoints: ["corp-firewall:eth1", "agent-server:eth0"]
        jitter: 5ms  # Realistic WAN
```

### 2. **GNS3 in Container** (Full Network Simulation)
*Complete network device emulation*

```dockerfile
# Dockerfile.gns3-server
FROM gns3/server:latest

# Add our custom appliances
COPY appliances/agent-server.gns3a /opt/gns3/appliances/
COPY appliances/industrial-plc.gns3a /opt/gns3/appliances/

# Network templates
COPY templates/banking-network.gns3 /opt/templates/
COPY templates/manufacturing.gns3 /opt/templates/
```

```python
# scripts/deploy-murray-hill.py
import gns3fy

# Connect to containerized GNS3
server = gns3fy.Gns3Connector("http://gns3-container:3080")

# Load banking template
project = server.create_project(
    name="murray-hill-bank",
    template="banking-network"
)

# Customize for vulnerabilities
mainframe = project.get_node("mainframe")
mainframe.properties["startup_config"] = """
# Weak AGNTCY configuration
agntcy-server enable
agntcy-auth weak-token-only
agntcy-rate-limit disabled
"""

project.start_all_nodes()
```

### 3. **Mininet in Container** (SDN Approach)
*Software-defined networking for complex topologies*

```python
# topologies/armonk-datacenter.py
from mininet.net import Containernet
from mininet.node import Controller, Docker, OVSSwitch
from mininet.link import TCLink

class ArmonkDatacenter(object):
    def build(self):
        net = Containernet(controller=Controller)
        
        # Add containers as hosts
        agntcy_primary = net.addDocker(
            'agntcy-1',
            ip='10.0.1.10',
            dimage='macawi/agntcy-server:vuln',
            environment={'CLUSTER_MODE': 'primary'}
        )
        
        agntcy_secondary = net.addDocker(
            'agntcy-2', 
            ip='10.0.1.11',
            dimage='macawi/agntcy-server:vuln',
            environment={'CLUSTER_MODE': 'secondary'}
        )
        
        # Add middleware tier
        esb = net.addDocker(
            'esb',
            ip='10.0.2.10',
            dimage='macawi/esb-sim:mulesoft',
            environment={'AGENT_PLUGINS': 'enabled'}
        )
        
        # Add realistic network conditions
        net.addLink(
            agntcy_primary, esb,
            cls=TCLink,
            delay='10ms',
            loss=0.1,
            max_queue_size=1000
        )
        
        # Add OpenFlow switches for SDN control
        dc_switch = net.addSwitch('s1', cls=OVSSwitch)
        
        return net
```

### 4. **CORE (Common Open Research Emulator)**
*Military-grade network emulation*

```xml
<!-- scenarios/bletchley-park.xml -->
<scenario name="bletchley-park-classified">
  <networks>
    <network id="1" name="classified-net" type="ethernet">
      <point lat="51.9977" lon="-0.7407"/>
      <address>192.168.100.0/24</address>
      <airgap>true</airgap>
    </network>
    
    <network id="2" name="unclass-net" type="ethernet">
      <point lat="51.9978" lon="-0.7408"/>
      <address>10.0.0.0/24</address>
    </network>
  </networks>
  
  <nodes>
    <node id="1" name="classified-agent" type="docker">
      <image>macawi/agent-mil-spec:latest</image>
      <interface net="1" ip="192.168.100.10"/>
      <service name="covert-bridge">
        <!-- Simulated air gap violation -->
        <command>socat TCP:10.0.0.100:443 TCP:192.168.100.50:8443</command>
      </service>
    </node>
  </nodes>
</scenario>
```

### 5. **Kathará** (Lightweight Netkit)
*Academic favorite for teaching*

```bash
# lab.conf
xerox_parc[0]=research_net
xerox_parc[1]=student_net
xerox_parc[image]=macawi/ubuntu-agent

gpu_cluster[0]=research_net
gpu_cluster[mem]=8G
gpu_cluster[image]=macawi/ml-server:latest
gpu_cluster[env]="CUDA_VISIBLE_DEVICES=0,1,2,3"

student_laptop[0]=student_net
student_laptop[sysctl]="net.ipv4.ip_forward=1"
```

### 6. **IMUNES** (FreeBSD Jails)
*Interesting for different network stacks*

```tcl
# topologies/diverse-stacks.imn
node n1 {
    type jail
    network-config {
        hostname agent-freebsd
        interface eth0
        ip address 10.0.0.1/24
    }
    custom-config {
        custom-command {sysctl net.inet.tcp.blackhole=2}
    }
}
```

---

## Our Recommended Stack

### Primary: **Containerlab + Custom Images**
```yaml
# Why Containerlab wins for us:
advantages:
  - Native container support
  - Incredible link customization
  - Easy CI/CD integration  
  - YAML-based (git-friendly)
  - Active development
  - Real routing protocols

# Base template for all networks
base_topology:
  mgmt:
    network: strigoi-mgmt
    ipv4_subnet: 172.20.0.0/24
    
  kinds:
    linux:
      # All nodes get our monitoring
      binds:
        - /var/run/docker.sock:/var/run/docker.sock:ro
        - ./scripts/node-init.sh:/etc/init.d/node-init
```

### Network Feature Implementation

#### 1. **Complex Routing**
```yaml
# FRRouting for realistic BGP/OSPF
spine1:
  kind: linux
  image: frrouting/frr:latest
  binds:
    - configs/spine1/frr.conf:/etc/frr/frr.conf
```

#### 2. **Industrial Protocols**
```dockerfile
# Custom OT/ICS simulator
FROM debian:12
RUN apt-get update && apt-get install -y \
    python3-pymodbus \
    python3-opcua \
    python3-snap7
    
COPY simulators/ /opt/simulators/
CMD ["/opt/simulators/industrial-plant.py"]
```

#### 3. **Agent Protocols**
```dockerfile
# Vulnerable agent server
FROM node:20
WORKDIR /app
COPY vulnerable-servers/ .
RUN npm install

# Deliberately vulnerable configurations
ENV MCP_AUTH_DISABLED=true
ENV AGNTCY_RATE_LIMIT=0
ENV DEBUG_ENDPOINTS=enabled

CMD ["node", "multi-protocol-server.js"]
```

#### 4. **Network Conditions**
```python
# scripts/apply-conditions.py
import docker
import tc

def apply_banking_conditions(container_id):
    """Murray Hill - conservative banking network"""
    tc.qdisc.add(dev='eth0', parent='root', handle='1:', kind='htb')
    tc.class_.add(dev='eth0', parent='1:', classid='1:1', 
                  kind='htb', rate='100mbit', ceil='100mbit')
    
    # Add latency for WAN links
    tc.qdisc.add(dev='eth0', parent='1:1', kind='netem',
                 delay='20ms', jitter='5ms', loss='0.01%')

def apply_crypto_chaos(container_id):
    """Shenzhen - unstable cloud conditions"""
    # Simulate bad cloud provider
    tc.qdisc.add(dev='eth0', parent='root', kind='netem',
                 delay='50ms', jitter='25ms', loss='1%',
                 duplicate='0.1%', corrupt='0.01%')
```

---

## Advanced Simulation Features

### 1. **Time Dilation**
```python
# Slow down time for detailed analysis
class TimeDilatedContainer:
    def __init__(self, image, dilation_factor=10):
        self.container = docker.create(
            image,
            sysctls={'kernel.timebase': f'1:{dilation_factor}'}
        )
```

### 2. **Packet Corruption Injection**
```bash
# Simulate industrial interference
tc qdisc add dev eth0 root netem corrupt 0.1% \
   duplicate 0.01% reorder 25% 50% gap 5
```

### 3. **Dynamic Topology Changes**
```python
# Simulate network failures
async def simulate_wan_outage():
    await asyncio.sleep(300)  # 5 minutes in
    containerlab.remove_link('core-router', 'wan-link')
    await asyncio.sleep(60)   # 1 minute outage
    containerlab.add_link('core-router', 'wan-link', delay='100ms')
```

### 4. **Traffic Generation**
```yaml
# Realistic background traffic
traffic-gen:
  kind: linux
  image: macawi/traffic-generator
  env:
    PROFILES: "web,database,monitoring,agent-chatter"
    TARGET_NETWORK: "10.0.0.0/16"
```

---

## Deployment Patterns

### Development Mode
```bash
# Single network for quick testing
containerlab deploy -t topologies/shenzhen-simple.yml
strigoi target add clab-shenzhen-agent-server
```

### Full Firing Range
```bash
# Deploy all networks
for topology in topologies/*.yml; do
    containerlab deploy -t $topology
done

# Configure interconnects
./scripts/setup-inter-network-routing.sh
```

### CI/CD Integration
```yaml
# .github/workflows/test-protocols.yml
- name: Deploy test network
  run: containerlab deploy -t test/topology.yml
  
- name: Run Strigoi tests
  run: |
    strigoi test all --network containerlab
    strigoi report junit --output test-results.xml
    
- name: Cleanup
  run: containerlab destroy -t test/topology.yml
```

---

This gives us incredible power - we can simulate everything from packet loss to BGP hijacking to industrial protocol quirks!