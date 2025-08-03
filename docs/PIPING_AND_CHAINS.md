# Unix-Style Piping in Strigoi

## The Power of Pipes

Strigoi embraces Unix philosophy: small actors doing one thing well, connected via pipes.

## Basic Piping Syntax

```bash
# Traditional Unix style with |
strigoi > probe/north/endpoint_discovery --target api.openai.com | sense/protocol

# Actor chain syntax with →
strigoi > endpoint_discovery --target api.openai.com → model_interrogation → vulnerability_assessment

# Both syntaxes work!
```

## Real-World Examples

### 1. Discovery to Analysis Pipeline
```bash
# Find endpoints, analyze protocols, check auth
strigoi > probe/north/endpoint_discovery --target example.com | \
          sense/protocol | \
          sense/trust/auth_checker

# Output flows naturally:
# endpoint_discovery → [endpoints] → protocol → [protocols] → auth_checker → [findings]
```

### 2. Filtering and Processing
```bash
# Find all OpenAI endpoints, filter for v1, check rate limits
strigoi > probe/north/endpoint_discovery --platform openai | \
          grep v1 | \
          probe/north/rate_limit_tester

# Grep is also an actor! It transforms data streams
```

### 3. Parallel Processing with Tee
```bash
# Send discovered endpoints to multiple analyzers
strigoi > probe/north/endpoint_discovery --target api.example.com | \
          tee >(sense/protocol) \
              >(sense/trust) \
              >(probe/north/model_interrogation)

# All three analyses run in parallel!
```

### 4. Conditional Chains
```bash
# Only test rate limits if we find endpoints
strigoi > probe/north/endpoint_discovery --target api.example.com | \
          if_not_empty | \
          probe/north/rate_limit_tester

# if_not_empty is a filter actor
```

## Actor Piping Rules

### What Can Be Piped

**Operational Actors** ✓
```bash
endpoint_discovery | model_interrogation  # Yes!
probe/north | sense/protocol             # Yes!
```

**Utility Actors** ✗
```bash
help | grep                              # No - help is standalone
list | sort                              # No - utility actors don't pipe
```

**Mixed Pipes** ✓
```bash
# Operational actors can pipe to filter actors
endpoint_discovery | grep openai | model_interrogation

# Filter actors (grep, sort, head, jq) work in pipes
probe/north/all | jq '.endpoints[]' | rate_limit_tester
```

## Data Transformation in Pipes

Each actor declares its input/output formats:

```yaml
# endpoint_discovery outputs:
{
  "endpoints": [
    {"url": "https://api.openai.com/v1/models", "platform": "openai"}
  ]
}

# model_interrogation expects:
{
  "endpoints": [...]
}

# Perfect match! They can pipe together
```

When formats don't match, Strigoi provides transformer actors:

```bash
# endpoint_discovery outputs 'endpoints', but auth_checker needs 'urls'
strigoi > endpoint_discovery | transform --map endpoints:urls | auth_checker

# Or use jq for complex transformations
strigoi > endpoint_discovery | jq '.endpoints[].url' | auth_checker
```

## Advanced Piping Patterns

### 1. Store and Continue
```bash
# Save intermediate results while continuing the pipe
strigoi > endpoint_discovery --target bigcorp.com | \
          save endpoints.json | \
          model_interrogation | \
          save models.json | \
          vulnerability_assessment
```

### 2. Error Handling in Pipes
```bash
# Continue pipe even if one actor fails
strigoi > endpoint_discovery | \
          try model_interrogation | \
          sense/protocol

# try actor catches errors and passes data through
```

### 3. Aggregating Results
```bash
# Collect results from multiple sources
strigoi > multi_target --targets sites.txt | \
          parallel endpoint_discovery | \
          collect | \
          report/summary

# parallel runs multiple instances
# collect aggregates results
```

### 4. Time-Based Pipes
```bash
# Rate-limited scanning
strigoi > target_list | \
          rate_limit --max 10/minute | \
          endpoint_discovery | \
          collect
```

## Why This is Powerful

1. **Composability** - Build complex workflows from simple actors
2. **Flexibility** - Rearrange actors on the fly
3. **Debugging** - Insert `debug` or `save` actors anywhere
4. **Parallelism** - Use `tee` and `parallel` for concurrent execution
5. **Familiarity** - Unix users feel at home

## Pipe vs Chain vs Assemblage

### Pipes (|)
- Ad-hoc composition
- Linear data flow
- Command-line convenience
- No persistence

### Chains (→)
- Defined sequences
- Can branch conditionally
- Saved and versioned
- State checkpoints

### Assemblages
- Complex topologies
- Parallel execution
- Resource coordination
- Production workflows

All three use the same actors! Choose based on your needs:

```bash
# Quick exploration? Use pipes
probe/north | sense/protocol

# Repeatable process? Define a chain
chain create llm_analysis "endpoint_discovery → model_interrogation → report"

# Production workflow? Build an assemblage
assemblage deploy llm_security_suite
```

## Implementation Note

Under the hood, pipes create temporary chains:

```bash
# This pipe:
actor1 | actor2 | actor3

# Becomes this temporary chain:
chain {
  steps: [
    {actor: "actor1", input: "stdin"},
    {actor: "actor2", input: "pipe:0"},
    {actor: "actor3", input: "pipe:1"}
  ]
}
```

This unification means all actor interaction patterns share the same engine!