import { StrigoiWrapper } from '../src/strigoi-wrapper';

describe('StrigoiWrapper', () => {
  let wrapper: StrigoiWrapper;

  beforeEach(() => {
    wrapper = new StrigoiWrapper('./strigoi');
  });

  describe('validateTarget', () => {
    it('should validate URL for north probe', () => {
      expect(StrigoiWrapper.validateTarget('https://example.com', 'north')).toBe(true);
      expect(StrigoiWrapper.validateTarget('http://localhost:3000', 'north')).toBe(true);
      expect(StrigoiWrapper.validateTarget('invalid-url', 'north')).toBe(false);
    });

    it('should validate paths for other probes', () => {
      expect(StrigoiWrapper.validateTarget('/path/to/target', 'south')).toBe(true);
      expect(StrigoiWrapper.validateTarget('./relative/path', 'east')).toBe(true);
      expect(StrigoiWrapper.validateTarget('', 'west')).toBe(false);
    });
  });

  describe('getProbeDirections', () => {
    it('should return all probe directions', () => {
      const directions = StrigoiWrapper.getProbeDirections();
      expect(directions).toEqual(['north', 'south', 'east', 'west']);
    });
  });

  describe('buildCommand', () => {
    it('should build basic command arguments', () => {
      const options = {
        target: 'https://example.com',
        probeDirection: 'north' as const,
        outputFormat: 'json' as const,
      };

      const args = wrapper['buildCommand'](options);
      expect(args).toContain('probe');
      expect(args).toContain('north');
      expect(args).toContain('https://example.com');
      expect(args).toContain('--output');
      expect(args).toContain('json');
    });

    it('should include MCP scanning for south probe', () => {
      const options = {
        target: './target',
        probeDirection: 'south' as const,
        scanMcp: true,
        outputFormat: 'json' as const,
      };

      const args = wrapper['buildCommand'](options);
      expect(args).toContain('--scan-mcp');
    });

    it('should include self scanning when requested', () => {
      const options = {
        target: './target',
        probeDirection: 'south' as const,
        includeSelf: true,
        outputFormat: 'json' as const,
      };

      const args = wrapper['buildCommand'](options);
      expect(args).toContain('--include-self');
    });
  });

  describe('parseTimeout', () => {
    it('should parse timeout strings correctly', () => {
      expect(wrapper['parseTimeout']('30s')).toBe(30000);
      expect(wrapper['parseTimeout']('5m')).toBe(300000);
      expect(wrapper['parseTimeout']('1h')).toBe(3600000);
      expect(wrapper['parseTimeout']('invalid')).toBe(30000); // default
    });
  });
});