# Help as an Actor: Meta-Recursive Design

## Why Help Should Be an Actor

Making help an actor transforms it from static documentation into a living, intelligent agent that:

1. **Adapts to Context** - Different help based on where you are
2. **Learns from Usage** - Tracks what users struggle with
3. **Chains with Others** - "help | grep endpoint" could work
4. **Evolves Independently** - Update help without touching core
5. **Has Agency** - Help can suggest, guide, and teach

## Help Actor Design

```yaml
actor:
  uuid: "00000000-0000-0000-0000-000000000001"  # Special UUID for core actors
  name: "help"
  display_name: "Contextual Help System"
  version: "1.0.0"
  direction: "center"  # It's a router/orchestrator
  risk_level: "none"
  
capabilities:
  provided:
    - name: "contextual_help"
      description: "Provides context-aware help"
    - name: "actor_discovery"
      description: "Lists available actors in context"
    - name: "usage_examples"
      description: "Shows real examples"
    - name: "learning_mode"
      description: "Interactive tutorials"

interaction:
  chaining:
    can_initiate: false
    can_terminate: false
    chains_to: []  # Help doesn't chain
    chains_from: []  # Nothing chains to help
    
  assemblages:
    member_of: []  # Help is not part of any assemblage
    constraints:
      standalone: true  # Must run independently
      no_state_sharing: true  # Doesn't share state with other actors
```

## Implementation Patterns

### 1. Context-Aware Help
```bash
strigoi > help
# Shows main commands

strigoi/probe > help
# Shows probe-specific help and available actors

strigoi/probe/north > help
# Shows north-specific actors and options

strigoi/probe/north > help endpoint_discovery
# Shows detailed help for specific actor
```

### 2. Help as Navigator
```bash
strigoi > help find "api discovery"
# Help actor searches all actors and suggests:
# → probe/north/endpoint_discovery
# → sense/network/api_mapper
# → respond/block/rate_limiter

strigoi > help path-to "enumerate endpoints"
# Help actor creates a path:
# 1. probe/north/endpoint_discovery
# 2. sense/protocol
# 3. report/findings
```

### 3. Help with Intelligence
```bash
strigoi > help suggest --scenario "test OpenAI API"
# Help actor suggests:
# Based on your scenario, I recommend:
# 1. probe/north/endpoint_discovery --platform openai
# 2. chain with model_interrogation
# 3. sense/trust for auth analysis

strigoi > help learn probe
# Enters interactive learning mode:
# "Let's explore the probe system together..."
```

### 4. Help Queries
```bash
# Find all actors that work with OpenAI
strigoi > help actors --filter openai

# Get examples for all north actors  
strigoi > help examples --direction north

# Show what chains with endpoint_discovery
strigoi > help chains-with endpoint_discovery

# Help is standalone - these patterns use help's built-in filtering,
# not Unix-style piping or actor chaining
```

## Meta-Recursive Benefits

### Everything is Uniform
```bash
# These all work the same way:
strigoi > help
strigoi > probe
strigoi > endpoint_discovery

# Because they're all actors!
```

### Self-Documenting System
```yaml
# Even help documents itself
strigoi > help help
# Shows: "I'm the help actor. I provide contextual assistance..."

strigoi > help --version
# help actor v1.0.0

strigoi > help --capabilities
# - contextual_help
# - actor_discovery
# - usage_examples
```

### Extensible Help
```yaml
# Add new help capabilities via actors
actors:
  - help_video     # Video tutorials
  - help_ai        # AI-powered assistance  
  - help_translate # Multi-language help
```

## Implementation Hierarchy

```
help (base actor)
├── help_context    # Contextual help provider
├── help_search     # Search through all help
├── help_tutorial   # Interactive tutorials
├── help_examples   # Code examples
├── help_chains     # Show actor relationships
└── help_suggest    # AI-powered suggestions
```

## Special Commands via Help Actor

```bash
# Instead of hardcoded commands, these become help actor methods:
strigoi > ?           # Alias for help
strigoi > man         # Alias for help --detailed
strigoi > info        # Alias for help --info
strigoi > tutorial    # Alias for help --tutorial
strigoi > wtf         # Alias for help --explain-error
```

## Help Actor State

The help actor can maintain state:
- Recently asked questions
- Common navigation patterns  
- User expertise level
- Preferred help format

```bash
strigoi > help set-level expert
# Help actor adjusts verbosity

strigoi > help history
# Shows what you've been asking about

strigoi > help bookmark "openai testing"
# Saves current context for later
```

## Why Help is Special

Help is a **utility actor** - it has different rules:

1. **No Chaining** - Help provides information, it doesn't transform data
2. **No Assemblages** - Help isn't part of security workflows  
3. **Always Available** - Help works from any context
4. **No Side Effects** - Help only reads, never modifies
5. **Instant** - Help has no execution time, no async operations

This makes help fundamentally different from operational actors like `endpoint_discovery` or `model_interrogation`. It's a meta-actor that helps you understand and use other actors.

### Utility Actors Pattern

This establishes a pattern for other utility actors:
- `help` - Provides assistance
- `list` - Shows available actors  
- `info` - Displays actor details
- `version` - Shows version information
- `config` - Manages configuration

All utility actors share these properties:
- Don't participate in chains or assemblages
- Have no risk level
- Execute instantly
- Read-only operations

## Why This is Revolutionary

1. **Truly Uniform System** - No special cases, everything is an actor
2. **Intelligent Help** - Not just static text but active assistance
3. **Learnable** - Help that adapts to how you use it
4. **Composable** - Chain help with other tools
5. **Evolvable** - Upgrade help without touching core

This makes Strigoi not just a tool, but an intelligent assistant that helps users discover and master its capabilities naturally.