import { component, input, action } from '@prismatic-io/spectral';
import { StrigoiWrapper, type ProbeDirection, type StrigoiScanOptions } from '../strigoi-wrapper';

/**
 * Strigoi Security Scanner Component for Prismatic
 */
export const strigoiScanner = component({
  key: 'strigoi-scanner',
  display: {
    label: 'Strigoi Security Scanner',
    description: 'Advanced security scanning with MCP (Model Context Protocol) analysis',
    iconPath: 'icon.png',
    category: 'Security',
  },
  inputs: {
    strigoiPath: input({
      label: 'Strigoi Binary Path',
      type: 'string',
      required: false,
      comments: 'Path to Strigoi binary. Leave empty to use PATH.',
      example: '/usr/local/bin/strigoi',
    }),
  },
  actions: {
    scan: action({
      display: {
        label: 'Security Scan',
        description: 'Perform security scan using Strigoi probe modules',
      },
      inputs: {
        target: input({
          label: 'Target',
          type: 'string',
          required: true,
          comments: 'URL for north probe, or file/directory path for other probes',
          example: 'https://api.example.com',
        }),
        probeDirection: input({
          label: 'Probe Direction',
          type: 'string',
          required: true,
          model: [
            { label: 'North (API Discovery)', value: 'north' },
            { label: 'South (Dependencies)', value: 'south' },
            { label: 'East (Data Flows)', value: 'east' },
            { label: 'West (Network Analysis)', value: 'west' },
          ],
          default: 'south',
          comments: 'Direction of security analysis focus',
        }),
        scanMcp: input({
          label: 'Enable MCP Scanning',
          type: 'boolean',
          required: false,
          default: true,
          comments: 'Scan for Model Context Protocol tools and security issues (south probe only)',
        }),
        includeSelf: input({
          label: 'Include Self Scan',
          type: 'boolean',
          required: false,
          default: false,
          comments: 'Include scanner\'s own files and processes in analysis',
        }),
        verbose: input({
          label: 'Verbose Output',
          type: 'boolean',
          required: false,
          default: false,
          comments: 'Enable detailed logging and output',
        }),
        timeout: input({
          label: 'Timeout',
          type: 'string',
          required: false,
          default: '30s',
          comments: 'Scan timeout (e.g., 30s, 5m, 1h)',
          example: '30s',
        }),
      },
      perform: async (context, params) => {
        const { logger } = context;
        const { strigoiPath } = context.config;

        try {
          // Initialize Strigoi wrapper
          const wrapper = new StrigoiWrapper(strigoiPath);

          // Health check
          const isHealthy = await wrapper.healthCheck();
          if (!isHealthy) {
            throw new Error('Strigoi binary not found or not working. Please check the installation.');
          }

          // Validate target format
          const probeDirection = params.probeDirection as ProbeDirection;
          if (!StrigoiWrapper.validateTarget(params.target, probeDirection)) {
            throw new Error(`Invalid target format for ${probeDirection} probe. Expected ${probeDirection === 'north' ? 'URL' : 'file/directory path'}.`);
          }

          // Prepare scan options
          const scanOptions: StrigoiScanOptions = {
            target: params.target,
            probeDirection,
            scanMcp: params.scanMcp ?? true,
            includeSelf: params.includeSelf ?? false,
            verbose: params.verbose ?? false,
            timeout: params.timeout || '30s',
            outputFormat: 'json',
          };

          logger.info(`Starting Strigoi ${probeDirection} scan for target: ${params.target}`);

          // Execute scan
          const result = await wrapper.scan(scanOptions);

          logger.info(`Strigoi scan completed with status: ${result.status}`);

          // Process and return results
          const output = {
            scanResult: result,
            summary: {
              module: result.module,
              status: result.status,
              hasError: !!result.error,
              errorMessage: result.error,
              executionTime: result.end_time && result.start_time 
                ? new Date(result.end_time).getTime() - new Date(result.start_time).getTime()
                : null,
            },
            mcpAnalysis: result.mcp_tools ? {
              toolsFound: result.mcp_tools.length,
              securityRisks: result.mcp_tools.reduce((total, tool) => 
                total + (tool.security_risks?.length || 0), 0
              ),
              criticalRisks: result.mcp_tools.reduce((total, tool) => 
                total + (tool.security_risks?.filter(risk => risk.severity === 'critical').length || 0), 0
              ),
              highRisks: result.mcp_tools.reduce((total, tool) => 
                total + (tool.security_risks?.filter(risk => risk.severity === 'high').length || 0), 0
              ),
            } : null,
          };

          return {
            data: output,
          };

        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
          logger.error(`Strigoi scan failed: ${errorMessage}`);
          
          throw new Error(`Security scan failed: ${errorMessage}`);
        }
      },
    }),

    healthCheck: action({
      display: {
        label: 'Health Check',
        description: 'Verify Strigoi installation and availability',
      },
      inputs: {},
      perform: async (context, params) => {
        const { logger } = context;
        const { strigoiPath } = context.config;

        try {
          const wrapper = new StrigoiWrapper(strigoiPath);
          
          const isHealthy = await wrapper.healthCheck();
          const version = isHealthy ? await wrapper.getVersion() : 'Not available';

          logger.info(`Strigoi health check: ${isHealthy ? 'PASS' : 'FAIL'}`);

          return {
            data: {
              healthy: isHealthy,
              version,
              binaryPath: wrapper['strigoiPath'], // Access private property for debugging
              availableProbes: StrigoiWrapper.getProbeDirections(),
            },
          };

        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
          logger.error(`Health check failed: ${errorMessage}`);

          return {
            data: {
              healthy: false,
              error: errorMessage,
              availableProbes: StrigoiWrapper.getProbeDirections(),
            },
          };
        }
      },
    }),
  },
});