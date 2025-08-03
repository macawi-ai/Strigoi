# Strigoi Beta Release Notes

## Version 0.1.0-beta

### Overview
Strigoi is now ready for beta testing with critical MCP security assessment capabilities. This release includes the core framework and 4 critical attack detection modules that address the most severe vulnerabilities in MCP implementations.

### Core Features
- **MSF-style Console**: Familiar interface for security professionals
- **Module System**: Extensible architecture for adding new security tests
- **Package Loading**: Dynamic module updates via APMS format
- **Attack Surface Model**: 13 distinct surfaces for comprehensive testing

### Critical Modules Included

#### 1. Command Injection Scanner (`mcp/validation/command_injection`)
- **Risk**: Critical
- **Description**: Detects unsanitized shell execution vulnerabilities
- **Tests**: 7 injection patterns including semicolon, pipe, backtick, and environment variables
- **Usage**:
  ```
  use mcp/validation/command_injection
  set TARGET http://localhost:3000
  run
  ```

#### 2. Session Hijacking Scanner (`mcp/session/header_hijack`)
- **Risk**: Critical  
- **Description**: Detects session management vulnerabilities
- **Tests**: Session ID exposure, predictable sessions, fixation attacks, missing security flags
- **Usage**:
  ```
  use mcp/session/header_hijack
  set TARGET http://localhost:3000
  run
  ```

#### 3. STDIO MitM Detection (`mcp/stdio/mitm_intercept`)
- **Risk**: Catastrophic
- **Description**: Detects STDIO interception vulnerabilities
- **Tests**: Process arguments, environment variables, file descriptor access, ptrace attachment
- **Usage**:
  ```
  use mcp/stdio/mitm_intercept
  set TARGET mcp-server
  run
  ```

#### 4. Config Credential Scanner (`mcp/config/credential_storage`)
- **Risk**: Critical
- **Description**: Scans for plaintext credentials in configuration files
- **Tests**: JSON, YAML, .env files, detects API keys, passwords, tokens, connection strings
- **Usage**:
  ```
  use mcp/config/credential_storage
  set TARGET_DIR /path/to/mcp/config
  run
  ```

### Getting Started

1. **Installation**:
   ```bash
   git clone https://github.com/macawi-ai/strigoi
   cd strigoi
   ./install.sh
   ```

2. **Basic Usage**:
   ```bash
   strigoi
   show modules
   use <module_path>
   show options
   set <option> <value>
   run
   ```

3. **Quick Assessment**:
   ```bash
   # Scan local MCP configuration
   use mcp/config/credential_storage
   set TARGET_DIR ~/.mcp
   run
   
   # Test running MCP server
   use mcp/validation/command_injection
   set TARGET http://localhost:8080
   run
   ```

### Beta Testing Focus

Please help test:
1. **Module Functionality**: Do the modules detect vulnerabilities correctly?
2. **False Positives**: Are there cases where safe configurations are flagged?
3. **Performance**: How do the modules perform against large MCP deployments?
4. **Usability**: Is the console interface intuitive?
5. **Coverage**: What attack vectors are we missing?

### Known Limitations
- Network transport modules still loading from packages
- Static and dynamic code analysis deferred to next release
- Report generation is text-only (PDF reports coming soon)
- Windows-specific attacks not yet implemented

### Reporting Issues
https://github.com/macawi-ai/strigoi/issues

### Security Note
Strigoi is designed for authorized security testing only. The critical vulnerabilities it detects represent fundamental flaws in MCP's security architecture. Based on our research, MCP in its current form is unsuitable for production use in security-conscious environments, particularly financial institutions.

### Next Release Preview
- Additional attack modules covering all 36 documented vectors
- Simulation lab for testing MCP implementations
- CVE-style risk rating system for MCP vulnerabilities
- Enhanced reporting with executive summaries
- MITRE ATT&CK mapping for AI/Agent attacks