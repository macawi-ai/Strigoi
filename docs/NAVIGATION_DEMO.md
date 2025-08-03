# Strigoi Navigation Demo

The new context-based navigation makes Strigoi incredibly intuitive:

## Basic Navigation

```bash
strigoi > help

Available commands:

  help, ?              Show this help menu
  probe                Enter probe context for discovery
  sense                Enter sense context for analysis
  respond              Enter respond context (future)
  report               Enter report context
  jobs                 List running jobs
  clear, cls           Clear the screen
  exit, quit           Exit the console

Navigation:
  - Type a command to enter its context
  - Use 'back' or '..' to go back
  - Use '/' for direct paths (e.g., probe/north)

strigoi > probe
[*] Entered probe context

Available directions:
  north    - Probe LLM/AI platforms
  east     - Probe human interaction layers
  south    - Probe tool and data protocols
  west     - Probe VCP-MCP broker systems
  center   - Probe routing/orchestration layer
  quick    - Quick scan across all directions
  all      - Exhaustive enumeration
  info     - Explain the cardinal directions model
  back     - Return to main context

strigoi/probe > north
[*] Entering probe/north
[!] Actor execution not yet implemented

strigoi/probe/north > back
[*] Returned to probe context

strigoi/probe > back
[*] Returned to main context

strigoi > 
```

## Sense Navigation

```bash
strigoi > sense
[*] Entered sense context

Available layers:
  network      - Network layer analysis
  transport    - Transport layer analysis
  protocol     - Protocol analysis (MCP, A2A)
  application  - Application layer analysis
  data         - Data flow and content analysis
  trust        - Trust and authentication analysis
  human        - Human interaction security
  back         - Return to main context

strigoi/sense > network
[*] Entering sense/network
[!] Layer analysis not yet implemented

strigoi/sense/network > 
```

## Direct Path Navigation

You can still use the direct path syntax:

```bash
strigoi > probe/north
[*] Probing North - LLM/AI Platforms

  [ ] Model detection endpoints
  [ ] Response pattern analysis
  [ ] Token limit testing
  [ ] System prompt extraction
  [ ] Model-specific behaviors

[!] North probing not yet implemented
```

## Why This is Revolutionary

1. **Zero Learning Curve**: Users naturally explore by typing what they see
2. **Context Awareness**: The prompt shows exactly where you are
3. **Progressive Disclosure**: Options reveal themselves as you navigate
4. **Consistent Navigation**: 'back' always works, 'help' is contextual
5. **Both Modes Work**: Navigate step-by-step OR use direct paths

This makes Strigoi accessible to:
- **Beginners**: Just type what you see, explore naturally
- **Power Users**: Direct paths for speed (probe/north/endpoint_discovery)
- **Everyone**: No manual needed, the interface teaches itself

## Future Actor Integration

When actors are implemented, the navigation will feel magical:

```bash
strigoi > probe
strigoi/probe > north
strigoi/probe/north > endpoint_discovery --target api.openai.com

# Or directly:
strigoi > probe/north/endpoint_discovery --target api.openai.com
```

The beauty is that help, list, info, etc. can all be actors themselves, making the entire system uniform and extensible.