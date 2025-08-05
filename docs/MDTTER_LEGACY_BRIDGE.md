# MDTTER Legacy Bridge - Evolution Without Abandonment

## The Pack Does Not Leave Members Behind

While we hunt in multi-dimensional topological space, we recognize that many security teams are still trapped in one-dimensional thinking. This bridge allows them to taste our power while speaking their familiar language.

## What We Built

### 1. **SIEM Translator** (`mdtter_siem_translator.go`)
Converts rich MDTTER events into formats legacy systems understand:
- **CEF** (Common Event Format) - For ArcSight
- **LEEF** (Log Event Extended Format) - For QRadar  
- **JSON** - Simplified for basic parsers
- **Splunk HEC** - Native Splunk format
- **Elasticsearch** - ECS-compatible

### 2. **Streaming Infrastructure** (`mdtter_streamer.go`)
- Batched delivery for performance
- Retry logic with exponential backoff
- Metrics tracking what's lost in translation
- Parallel streaming to multiple endpoints

## The Translation Reality

### What Legacy Systems See:
```json
{
  "timestamp": "2025-02-03T10:00:00Z",
  "source_ip": "192.168.1.100",
  "destination_ip": "10.0.0.50",
  "destination_port": 443,
  "severity": "high",
  "category": "exfiltration",
  "risk_score": 78,
  "message": "MDTTER: exfiltration activity detected..."
}
```

### What They're Missing:
- 128-dimensional behavioral embeddings
- Topological graph relationships
- Smooth manifold trajectories
- Probability distributions across 7 intent categories
- Dynamic topology morphing operations
- Variety absorption learning

### The Bridge Hints:
Every translated event includes hints about the richer data:
```json
{
  "mdtter_enabled": true,
  "mdtter_dimensions": 128,
  "mdtter_vam": 0.78,
  "mdtter_curvature": 0.45,
  "upgrade_available": "Contact Strigoi for multi-dimensional analysis"
}
```

## Implementation Examples

### Splunk Integration
```go
config := StreamerConfig{
    LegacyEndpoints: map[string]LegacyEndpointConfig{
        "splunk-prod": {
            Type:   FormatSplunk,
            URL:    "https://splunk.redcanary.com:8088/services/collector",
            APIKey: "xxx",
            Index:  "security",
        },
    },
}
```

The Splunk team sees events they can search:
```
index=security mdtter_enabled=true vam_score>70 
| stats count by threat_category
```

But we preserve the truth that they're only seeing shadows.

### Elasticsearch Integration
```go
"elastic-soc": {
    Type:   FormatElastic,
    URL:    "https://elastic.redcanary.com:9200",
    Index:  "security-events-*",
}
```

They can build dashboards showing:
- Event counts by category
- Top source IPs
- Risk score trends

While we track attack evolution through topological space.

## The Evolution Message

Every minute, the streamer logs:
```
MDTTER Streaming Metrics: 10,000 events translated, ~156 dimensions lost per event.
Legacy systems are seeing only shadows of the true security topology.
```

This reminds operators what they're missing.

## Migration Path

### Phase 1: Parallel Operations
- Legacy SIEMs receive translated events
- MDTTER-aware systems receive full events
- Security teams can correlate between both

### Phase 2: Gradual Enhancement  
- Add MDTTER visualization dashboards
- Train analysts on topological thinking
- Show value through detected attacks legacy missed

### Phase 3: Full Evolution
- Legacy systems become historical archives
- All detection runs on MDTTER
- Security operates in multiple dimensions

## The Sales Pitch for Executives

"Your current SIEM sees security like a 2D flatland resident - unable to comprehend the third dimension where attacks actually move. MDTTER shows you the full topology. 

We'll feed your existing tools what they can digest, but imagine seeing an attack's behavioral trajectory through space, watching its intent evolve, seeing your defensive surface automatically adapt.

That's not science fiction. That's MDTTER. That's what Red Canary needs."

## Technical Integration Guide

### For Splunk:
1. Install Strigoi MDTTER app from Splunkbase
2. Configure HEC endpoint in Strigoi
3. Start seeing enriched events immediately
4. Use pre-built dashboards for topology visualization

### For Elastic:
1. Install MDTTER index template
2. Configure Elasticsearch output
3. Import Kibana dashboards
4. Enable Machine Learning jobs for VAM anomalies

### For Legacy SIEMs:
1. Configure syslog/CEF output
2. Parse MDTTER custom fields
3. Build correlation rules on VAM scores
4. Alert on topology changes

## What This Enables

Even with degraded data, legacy systems gain:
- **Variety Detection**: VAM score shows truly novel attacks
- **Intent Classification**: Know if it's recon or exfil
- **Behavioral Anomalies**: Distance from normal baseline
- **Topology Awareness**: At least see node connections

But they'll know they're missing the real intelligence.

## The Future Is Now

While legacy systems parse our shadows, we're building:
- Real-time 3D attack visualization
- Predictive intent modeling
- Autonomous defensive adaptation
- Behavioral pattern learning

The bridge exists so no one gets left behind. But evolution doesn't wait.

---

*"We speak your language while hunting in dimensions you cannot yet see"*