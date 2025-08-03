# Strigoi Portfolio Repository
# DHR-inspired versioned protocol test management

## Structure

```
portfolio/
├── protocols/
│   ├── mcp/
│   │   ├── versions/
│   │   │   ├── 2025-03-26/
│   │   │   │   ├── contract.json         # Original MCP spec
│   │   │   │   ├── blueprint.yaml        # Our translation
│   │   │   │   ├── security-analysis.yaml
│   │   │   │   └── features/
│   │   │   │       ├── tools_list/
│   │   │   │       │   ├── blueprint.yaml
│   │   │   │       │   ├── build_ticket.yaml
│   │   │   │       │   ├── metacode.yaml
│   │   │   │       │   └── implementation.go
│   │   │   │       └── prompts_list/
│   │   │   │           └── ...
│   │   │   └── 2025-04-15/              # New version
│   │   │       └── ...
│   │   └── active -> versions/2025-03-26 # Symlink to current
│   └── openapi/
│       └── ...
├── test-runs/
│   ├── 2025-01-25-140523-a7f3e9b2/      # timestamp-build
│   │   ├── manifest.json                 # Signed manifold
│   │   ├── results.txt                   # STDIO output
│   │   ├── evidence/                     # Request/response pairs
│   │   └── logs/
│   └── latest -> 2025-01-25-140523-a7f3e9b2
└── reports/
    ├── prismatic-2025-01-25.pdf         # Generated for Buzz
    └── ...
```

## Version Control Philosophy

1. **Immutable History**: Once a protocol version is tested, that portfolio version is frozen
2. **New Versions**: Protocol updates create new version directories
3. **Test Runs**: Each execution creates timestamped directory with full evidence
4. **Active Symlinks**: Easy access to current versions

## Portfolio Operations

```bash
# Check in new protocol version
strigoi portfolio add --protocol=mcp --version=2025-04-15 --contract=new-spec.json

# List all tested versions
strigoi portfolio list --protocol=mcp

# Run test from portfolio
strigoi test --portfolio=mcp/2025-03-26 --target=http://localhost:8080

# Compare versions
strigoi portfolio diff --protocol=mcp --from=2025-03-26 --to=2025-04-15
```

## Benefits

- No file litter - everything organized by protocol/version/feature
- Complete audit trail of what was tested when
- Easy rollback to test older versions
- Clear separation of test definitions from test executions
- DHR-compliant tracking of all changes