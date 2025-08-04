import { execa, type ExecaReturnValue } from 'execa';
import { existsSync } from 'fs-extra';
import path from 'path';
import Joi from 'joi';

/**
 * Strigoi probe directions
 */
export type ProbeDirection = 'north' | 'south' | 'east' | 'west';

/**
 * Strigoi scan options
 */
export interface StrigoiScanOptions {
  target: string;
  probeDirection: ProbeDirection;
  scanMcp?: boolean;
  includeSelf?: boolean;
  verbose?: boolean;
  timeout?: string;
  outputFormat?: 'json' | 'yaml';
}

/**
 * Strigoi scan result structure
 */
export interface StrigoiScanResult {
  module: string;
  status: string;
  start_time: string;
  end_time?: string;
  data?: Record<string, any>;
  error?: string;
  // MCP-specific results
  mcp_tools?: Array<{
    id: string;
    name: string;
    type: string;
    status: string;
    security_risks?: Array<{
      id: string;
      category: string;
      severity: string;
      description: string;
      evidence: string;
      file_path: string;
      remediation: string;
    }>;
  }>;
}

/**
 * Validation schemas
 */
const scanOptionsSchema = Joi.object({
  target: Joi.string().required(),
  probeDirection: Joi.string().valid('north', 'south', 'east', 'west').required(),
  scanMcp: Joi.boolean().default(false),
  includeSelf: Joi.boolean().default(false),
  verbose: Joi.boolean().default(false),
  timeout: Joi.string().default('30s'),
  outputFormat: Joi.string().valid('json', 'yaml').default('json'),
});

/**
 * Strigoi CLI wrapper for Prismatic integration
 */
export class StrigoiWrapper {
  private strigoiPath: string;

  constructor(strigoiPath?: string) {
    // Try to find Strigoi binary
    this.strigoiPath = strigoiPath || this.findStrigoiBinary();
  }

  /**
   * Find Strigoi binary in common locations
   */
  private findStrigoiBinary(): string {
    const possiblePaths = [
      '/usr/local/bin/strigoi',
      '/usr/bin/strigoi',
      './strigoi',
      '../strigoi',
      '../../strigoi',
      process.env.STRIGOI_PATH,
    ].filter(Boolean) as string[];

    for (const binPath of possiblePaths) {
      if (existsSync(binPath)) {
        return path.resolve(binPath);
      }
    }

    // Default to 'strigoi' assuming it's in PATH
    return 'strigoi';
  }

  /**
   * Validate scan options
   */
  private validateOptions(options: StrigoiScanOptions): StrigoiScanOptions {
    const { error, value } = scanOptionsSchema.validate(options);
    if (error) {
      throw new Error(`Invalid scan options: ${error.message}`);
    }
    return value;
  }

  /**
   * Build Strigoi command arguments
   */
  private buildCommand(options: StrigoiScanOptions): string[] {
    const args = [
      'probe',
      options.probeDirection,
      options.target,
      '--output',
      options.outputFormat || 'json',
    ];

    if (options.timeout) {
      args.push('--timeout', options.timeout);
    }

    if (options.verbose) {
      args.push('--verbose');
    }

    // Probe-specific options
    if (options.probeDirection === 'south' && options.scanMcp) {
      args.push('--scan-mcp');
    }

    if (options.probeDirection === 'south' && options.includeSelf) {
      args.push('--include-self');
    }

    return args;
  }

  /**
   * Execute Strigoi scan
   */
  async scan(options: StrigoiScanOptions): Promise<StrigoiScanResult> {
    // Validate input options
    const validatedOptions = this.validateOptions(options);
    
    try {
      // Build command
      const args = this.buildCommand(validatedOptions);
      
      // Execute Strigoi
      const result: ExecaReturnValue = await execa(this.strigoiPath, args, {
        timeout: this.parseTimeout(validatedOptions.timeout || '30s'),
        encoding: 'utf8',
        reject: false, // Don't throw on non-zero exit codes
      });

      // Handle execution results
      if (result.exitCode !== 0) {
        throw new Error(`Strigoi execution failed (exit code ${result.exitCode}): ${result.stderr}`);
      }

      // Parse JSON output
      try {
        const scanResult = JSON.parse(result.stdout) as StrigoiScanResult;
        return scanResult;
      } catch (parseError) {
        throw new Error(`Failed to parse Strigoi output: ${parseError instanceof Error ? parseError.message : 'Unknown parse error'}`);
      }

    } catch (error) {
      if (error instanceof Error) {
        // Re-throw with context
        throw new Error(`Strigoi scan failed: ${error.message}`);
      }
      throw new Error(`Strigoi scan failed: Unknown error`);
    }
  }

  /**
   * Parse timeout string to milliseconds
   */
  private parseTimeout(timeout: string): number {
    const match = timeout.match(/^(\d+)([smh]?)$/);
    if (!match) {
      return 30000; // Default 30 seconds
    }

    const value = parseInt(match[1], 10);
    const unit = match[2] || 's';

    switch (unit) {
      case 's': return value * 1000;
      case 'm': return value * 60 * 1000;
      case 'h': return value * 60 * 60 * 1000;
      default: return value * 1000;
    }
  }

  /**
   * Get available probe directions
   */
  static getProbeDirections(): ProbeDirection[] {
    return ['north', 'south', 'east', 'west'];
  }

  /**
   * Validate target format
   */
  static validateTarget(target: string, probeDirection: ProbeDirection): boolean {
    switch (probeDirection) {
      case 'north':
        // URL validation for north probe
        try {
          new URL(target);
          return true;
        } catch {
          return false;
        }
      case 'south':
      case 'east':
      case 'west':
        // Path validation for file system probes
        return typeof target === 'string' && target.length > 0;
      default:
        return false;
    }
  }

  /**
   * Get Strigoi version info
   */
  async getVersion(): Promise<string> {
    try {
      const result = await execa(this.strigoiPath, ['--version'], { timeout: 5000 });
      return result.stdout.trim();
    } catch (error) {
      throw new Error(`Failed to get Strigoi version: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Test if Strigoi is available and working
   */
  async healthCheck(): Promise<boolean> {
    try {
      await this.getVersion();
      return true;
    } catch {
      return false;
    }
  }
}