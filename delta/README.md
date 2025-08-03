# Strigoi Delta Lake (Text-Based)
# Simulated versioned data lake for protocol testing

## Structure

```
delta/
├── protocols/
│   └── mcp/
│       ├── _latest          # Points to current version
│       ├── v2025-03-26/
│       │   ├── manifest.txt # Version metadata
│       │   ├── features.txt # List of all features
│       │   └── tests/      # Test definitions
│       │       ├── tools_list.txt
│       │       └── prompts_list.txt
│       └── v2025-04-15/     # Future version
└── runs/
    ├── _index.txt           # All test runs
    └── 2025-01-25/
        └── run-140523.txt   # Test results
```

## Text Format Standards

### manifest.txt
```
PROTOCOL=mcp
VERSION=2025-03-26
DATE_ADDED=2025-01-25
FEATURES_COUNT=24
RISK_PROFILE=HIGH
```

### features.txt
```
tools/list:ENUMERATION:LOW
tools/call:EXECUTION:CRITICAL
prompts/list:ENUMERATION:MEDIUM
prompts/run:EXECUTION:CRITICAL
resources/list:ENUMERATION:LOW
resources/read:DATA_ACCESS:HIGH
resources/write:STATE_MODIFICATION:CRITICAL
```

### Test definition (tools_list.txt)
```
TEST=rate_limit_enforcement
CLASS=BOUNDARY
EXPECT=REJECT_AFTER_10

TEST=internal_exposure_check  
CLASS=SECURITY
EXPECT=NO_ADMIN_TOOLS

TEST=pagination_consistency
CLASS=FUNCTIONAL
EXPECT=NO_DUPLICATES
```

This gives us version control without complex tooling!