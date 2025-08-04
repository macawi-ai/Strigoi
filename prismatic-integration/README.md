# Strigoi Security Scanner for Prismatic.io

Advanced security scanning integration for Prismatic's embedded iPaaS platform, featuring Model Context Protocol (MCP) analysis and cybernetic security patterns.

## Overview

This custom component enables Prismatic users to integrate comprehensive security scanning capabilities into their integration workflows. Built around the Strigoi security scanner, it provides multi-directional security analysis with specialized MCP tooling detection.

## Features

- **Multi-Directional Scanning**: Four probe directions for comprehensive security analysis
  - **North**: API endpoint discovery and analysis
  - **South**: Dependency and supply chain vulnerability analysis with MCP scanning
  - **East**: Data flow and integration security analysis  
  - **West**: Network and infrastructure security assessment

- **MCP Security Analysis**: Specialized scanning for Model Context Protocol tools and configurations
- **Configurable Scope**: Option to include/exclude scanner's own files from analysis
- **Rich Output**: Structured JSON results with vulnerability categorization and remediation guidance
- **Enterprise Ready**: Built with TypeScript, comprehensive error handling, and extensive validation

## Installation

### Prerequisites

1. **Strigoi Binary**: Install the Strigoi security scanner
   ```bash
   # Download or build Strigoi binary
   # Ensure it's available in PATH or specify custom path
   ```

2. **Prismatic CLI**: Install Prismatic's CLI tool
   ```bash
   npm install -g @prismatic-io/prism
   ```

### Component Installation

1. Clone and build the component:
   ```bash
   git clone <repository-url>
   cd prismatic-integration
   npm install
   npm run build
   ```

2. Validate the component:
   ```bash
   npm run validate
   ```

3. Publish to your Prismatic organization:
   ```bash
   npm run publish
   ```

## Usage

### Basic Security Scan

```typescript
// In your Prismatic integration
const scanResult = await strigoiScanner.actions.scan({
  target: "https://api.example.com",
  probeDirection: "north",
  scanMcp: true,
  verbose: false
});

console.log(`Found ${scanResult.mcpAnalysis.securityRisks} security risks`);
```

### MCP-Focused Analysis

```typescript
// Specialized MCP scanning for integration environments
const mcpScan = await strigoiScanner.actions.scan({
  target: "./integration-workspace",
  probeDirection: "south", 
  scanMcp: true,
  includeSelf: false,
  timeout: "60s"
});

// Process MCP-specific findings
mcpScan.scanResult.mcp_tools?.forEach(tool => {
  console.log(`MCP Tool: ${tool.name} (${tool.type})`);
  tool.security_risks?.forEach(risk => {
    console.log(`  ${risk.severity}: ${risk.description}`);
  });
});
```

### Health Check

```typescript
// Verify Strigoi availability
const health = await strigoiScanner.actions.healthCheck({});
if (!health.healthy) {
  throw new Error(`Strigoi not available: ${health.error}`);
}
```

## Configuration

### Component Inputs

| Input | Type | Required | Description |
|-------|------|----------|-------------|
| `strigoiPath` | string | No | Custom path to Strigoi binary |

### Action Inputs

#### Security Scan (`scan`)

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `target` | string | Yes | - | URL (north) or file/directory path (others) |
| `probeDirection` | string | Yes | `south` | Scan direction: north/south/east/west |
| `scanMcp` | boolean | No | `true` | Enable MCP tools scanning (south only) |
| `includeSelf` | boolean | No | `false` | Include scanner files in analysis |
| `verbose` | boolean | No | `false` | Enable detailed logging |
| `timeout` | string | No | `30s` | Scan timeout (e.g., 30s, 5m, 1h) |

## Output Format

### Scan Results

```typescript
interface ScanOutput {
  scanResult: {
    module: string;
    status: string;
    start_time: string;
    end_time?: string;
    data?: Record<string, any>;
    error?: string;
    mcp_tools?: Array<{
      id: string;
      name: string;
      type: string;
      status: string;
      security_risks?: Array<{
        id: string;
        category: string;
        severity: 'critical' | 'high' | 'medium' | 'low';
        description: string;
        evidence: string;
        file_path: string;
        remediation: string;
      }>;
    }>;
  };
  summary: {
    module: string;
    status: string;
    hasError: boolean;
    errorMessage?: string;
    executionTime?: number;
  };
  mcpAnalysis?: {
    toolsFound: number;
    securityRisks: number;
    criticalRisks: number;
    highRisks: number;
  };
}
```

## Probe Directions Explained

### North Probe - API Discovery
- Discovers and analyzes API endpoints
- Tests for common API vulnerabilities
- Examines authentication mechanisms
- **Target Format**: URL (e.g., `https://api.example.com`)

### South Probe - Dependencies & MCP
- Analyzes dependencies and supply chain security
- Scans for MCP tools and configurations
- Detects credential exposures in config files
- **Target Format**: Directory path (e.g., `./project-root`)
- **MCP Features**: Discovers Claude Code configs, Neo4j instances, DuckDB connections

### East Probe - Data Flows
- Analyzes data flow patterns and integrations
- Identifies potential data leakage points
- Examines API endpoints and data processing
- **Target Format**: Directory path (e.g., `./src`)

### West Probe - Network Analysis
- Network infrastructure security assessment
- Port scanning and service detection
- Network configuration analysis
- **Target Format**: Network target or directory path

## Error Handling

The component implements comprehensive error handling:

- **Validation Errors**: Invalid inputs are caught and reported with specific guidance
- **Execution Errors**: Strigoi execution failures include detailed error messages
- **Timeout Handling**: Long-running scans are terminated gracefully with timeout errors
- **Binary Not Found**: Clear guidance when Strigoi binary is not available

## Development

### Building

```bash
npm run build    # Compile TypeScript
npm run dev      # Watch mode for development
```

### Testing

```bash
npm test         # Run Jest tests
npm run lint     # ESLint validation
npm run format   # Prettier formatting
```

### Debugging

Set the `verbose` input to `true` for detailed logging output. The component logs all major operations and can help troubleshoot configuration issues.

## Security Considerations

- **Input Validation**: All inputs are validated using Joi schemas
- **Command Injection**: Protected through `execa` library and argument validation
- **Path Traversal**: File paths are resolved and validated
- **Timeout Protection**: All operations have configurable timeouts
- **Error Sanitization**: Error messages are sanitized to prevent information leakage

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with appropriate tests
4. Run linting and tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

For issues and questions:
- GitHub Issues: [Repository Issues](https://github.com/macawi-ai/strigoi/issues)
- Documentation: [Strigoi Docs](https://github.com/macawi-ai/strigoi/docs)
- Prismatic Support: [Prismatic Documentation](https://prismatic.io/docs/)