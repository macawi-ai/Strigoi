# Protocol Pipeline Automation Status

## Pipeline Steps Grid

| Step | Description | Current State | Automation Target | Tool/System |
|------|-------------|---------------|-------------------|-------------|
| 1. Discovery | Learn about new protocol (e.g., Google announces A2A) | **MANUAL** - Cy reads news/feeds | Automated monitoring | RSS feeds, Google Alerts, GitHub releases, HN API |
| 2. Locate Definition | Find formal protocol spec/docs | **MANUAL** - Cy searches for docs | Semi-automated search | Web scraper, GitHub API, protocol registries |
| 3. Parse & Classify | Analyze protocol for risk/priority | **MANUAL** - Human judgment | **AUTOMATED** - Parser | Protocol classifier (to build) |
| 4. Queue Placement | Add to backlog with priority | **MANUAL** - We decided priority | **AUTOMATED** - Based on classification | Backlog queue system (to build) |
| 5. Dequeue & Deconstruct | Pull from backlog for implementation | **NOT BUILT** | Automated | Protocol deconstructor (future) |

## Current Reality (July 2025)

### What We Have:
- Classification model (1-5 risk, Week 1/2/3 priority)
- Manual decision process
- Basic protocol structure in `S1-operations/protocols/`

### What We Did Manually:
```yaml
# Example: When we learned about AGNTCY
1. Discovery: "Cy learned about AGNTCY from industry contacts"
2. Located: "Found internal docs"
3. Classified: "Decided it's Critical/Week 2"
4. Queued: "Put in our mental backlog"
5. Built: "Started implementation"
```

## Automation Focus: Steps 3 & 4

### Step 3: Protocol Classification Parser
**Input**: Protocol specification document (PDF, HTML, Markdown)
**Output**: Classification object
```json
{
  "protocol": "Google A2A",
  "risk_level": 5,
  "priority": "week_1",
  "complexity": "high",
  "market_impact": "massive",
  "rationale": {
    "risk": "Agent autonomy with minimal human oversight",
    "priority": "Google = instant adoption",
    "complexity": "Multi-agent coordination is complex"
  }
}
```

### Step 4: Automated Backlog Queue
**Input**: Classification object
**Output**: Queued in priority order
```yaml
backlog:
  week_1_critical:
    - google_a2a (added: 2025-07-24, risk: 5)
    - openai_v2 (added: 2025-07-20, risk: 4)
  week_2_high:
    - agntcy (added: 2025-07-15, risk: 4)
  week_3_medium:
    - langchain_hub (added: 2025-07-10, risk: 3)
```

## Next Implementation Steps

1. **Build Protocol Classifier** (Step 3)
   - Parse protocol docs
   - Extract key indicators
   - Apply classification rules
   - Output priority score

2. **Build Backlog Queue** (Step 4)
   - Priority queue data structure
   - Persistence (file/DB)
   - API for adding/removing
   - Status dashboard

---

*Moving from manual heroics to automated pipeline*