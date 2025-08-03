# DeepSeek-R1 Integration via Together.ai

## Overview

Strigoi has access to DeepSeek-R1-0528, an advanced reasoning model, through the Together.ai API. This integration enables sophisticated analysis and reasoning capabilities for security validation tasks.

## Architecture

```
Strigoi <-> MCP Protocol <-> together-mcp server <-> Together.ai API <-> DeepSeek-R1
```

## Key Features

- **Model**: DeepSeek-R1-0528 (87.5% on AIME 2025)
- **Access**: Through Together.ai's infrastructure
- **Protocol**: MCP (Model Context Protocol)
- **Implementation**: Go-based high-performance server

## Current Status

✅ **API Connection**: Verified working  
✅ **MCP Server**: Running at `/home/cy/mcp-workspace/servers/together-server/together-mcp`  
⚠️ **Authentication**: API key needs to be in environment when MCP server starts  

## Usage in Strigoi

DeepSeek-R1 can be leveraged for:

1. **Complex Security Analysis**
   - Multi-step attack path reasoning
   - Vulnerability chain analysis
   - Impact assessment

2. **Code Review**
   - Security pattern recognition
   - Vulnerability detection
   - Best practice recommendations

3. **Report Generation**
   - Comprehensive security assessments
   - Executive summaries
   - Technical deep-dives

## API Test Results

Direct API test shows DeepSeek-R1's characteristic thinking process:
```
"<think>
Okay, the user just asked me to say hello and confirm that I'm DeepSeek-R1. 
[Shows internal reasoning process]
</think>"
```

## Configuration

The Together.ai MCP server requires:
- `TOGETHER_API_KEY` environment variable (uses `DEEPSEEK_API_KEY` from secure storage)
- Located at: `/home/cy/.vsm_secure_keys/api_keys_backup.env`

## Starting the Server

```bash
# With proper environment
cd /home/cy/mcp-workspace/servers/together-server
./start-with-key.sh
```

## Integration Points

1. **Security Validation**
   - Use for complex reasoning about attack paths
   - Analyze security implications of code changes
   - Generate comprehensive threat models

2. **Multi-LLM Collaboration**
   - Combine with Gemini for diverse perspectives
   - Use Claude for implementation, DeepSeek for reasoning
   - Cross-validate security findings

3. **Report Generation**
   - Executive summaries with business impact
   - Technical details with remediation steps
   - Compliance mapping and risk assessment

## Future Enhancements

1. **Automated Security Reasoning**
   - Integrate DeepSeek-R1 into Strigoi's sense/ actors
   - Automatic vulnerability chain analysis
   - Real-time security impact assessment

2. **Collaborative Analysis**
   - Multi-model security reviews
   - Consensus-based vulnerability scoring
   - Diverse perspective integration

## Notes

- DeepSeek-R1 excels at complex, multi-step reasoning
- The `<think>` tags show its reasoning process
- Best used for tasks requiring deep analysis rather than quick responses
- Complements Claude's implementation focus with reasoning depth