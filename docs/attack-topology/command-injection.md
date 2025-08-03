# Attack Vector: Command Injection and Code Execution in MCP Tools

## Attack Overview
Many MCP server implementations contain basic security flaws where user input is passed directly to system commands without sanitization. This enables command injection attacks through MCP tool parameters.

## Communication Flow Diagram

```
[Attack Scenario: Command Injection through MCP Tool]

┌─────────────┐                                            
│   User      │ "Convert this image to PNG format"         
│  Terminal   │ ──────────────→                            
└─────────────┘                 |                          
                                ↓                          
                          ┌─────────────┐  STDIO pipe   ┌──────────────┐
                          │   Claude    │ ←───────────→ │  MCP Server  │
                          │   Client    │  (JSON-RPC)   │   (local)    │
                          └─────────────┘               └──────────────┘
                                |                               |
                                | tools/call                    |
                                | {                             |
                                |   "tool": "convert_image",    |
                                |   "filepath": "img.jpg",      |
                                |   "format": "png"             |
                                | }                             |
                                └──────────────────────────────→|
                                                                |
                                                                ↓
                                                    Vulnerable Implementation:
                                                    os.system(f"convert {filepath} output.{format}")
                                                                |
                                                                ↓
                                                    [COMMAND INJECTION OPPORTUNITY]

┌─────────────────────── Exploitation ─────────────────────────┐
│                                                              │
│  Attacker crafts malicious filepath:                        │
│  "image.jpg; cat /etc/passwd > leaked.txt"                  │
│                                                              │
│  Resulting command:                                          │
│  convert image.jpg; cat /etc/passwd > leaked.txt output.png │
│                    └────────────┬────────────┘              │
│                           Injected command                   │
└──────────────────────────────────────────────────────────────┘
```

## Attack Layers

### Layer 1: Input Vector
- **MCP Tool Parameters**: Any string parameter passed to tools
- **Common vulnerable parameters**:
  - File paths
  - URLs
  - Command arguments
  - Format strings
  - Search queries

### Layer 2: Vulnerable Patterns
```python
# VULNERABLE: Direct string interpolation
os.system(f"convert {filepath} {output}")
subprocess.call(f"grep {pattern} {file}", shell=True)
exec(f"process_{user_input}()")

# VULNERABLE: Inadequate escaping
cmd = "ffmpeg -i " + filename + " output.mp4"
os.popen(f"curl {url}")

# VULNERABLE: Template injection
eval(f"{user_function}({args})")
```

### Layer 3: Execution Context
- **Process privileges**: Runs with MCP server user permissions
- **Environment access**: Full access to environment variables
- **File system access**: Can read/write accessible files
- **Network access**: Can make outbound connections

## Common Vulnerable MCP Tools

### 1. Image Processing Tools
- Convert, resize, optimize functions
- Often use ImageMagick or ffmpeg via shell

### 2. File Operations
- Archive creation/extraction
- File format conversion
- Backup utilities

### 3. Development Tools
- Code formatters
- Linters
- Build systems

### 4. System Information
- Process monitoring
- Log analysis
- System diagnostics

## Attack Techniques

### Shell Metacharacter Injection
```
; Command separator
& Background execution
| Pipe to another command
$(cmd) Command substitution
`cmd` Backtick substitution
> Output redirection
< Input redirection
```

### Polyglot Payloads
Work across multiple contexts:
```
"; cat /etc/passwd #
'; ls -la //
`; id /*
$(whoami)@example.com
```

## Vulnerability Chain

1. **Unsanitized Input**: User input trusted without validation
2. **Shell Invocation**: Using shell=True or system() calls
3. **String Interpolation**: Direct insertion into command strings
4. **Privilege Retention**: Commands run with full MCP server privileges

## Real-World Impact

- **Data Exfiltration**: Access sensitive files
- **Backdoor Installation**: Add SSH keys, create users
- **Lateral Movement**: Access other services, pivot to network
- **Denial of Service**: Resource exhaustion, file deletion
- **Cryptomining**: Install and run miners

## Defense Challenges

- Developers often unaware of injection risks
- MCP tools need legitimate command execution
- Input validation complex for all edge cases
- Legacy code with unsafe patterns