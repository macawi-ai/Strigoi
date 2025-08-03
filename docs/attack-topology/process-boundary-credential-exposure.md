# Security Analysis: Process Boundary and Credential Exposure Vectors

## Overview
Process boundaries, database connections, and configuration storage create critical credential exposure points that are often overlooked in MCP security assessments. These vectors are particularly dangerous because they're considered "normal" system behavior.

## 1. Process Boundary (STDIO) - Credential Exposure in Process Arguments

### The Critical Flaw
```bash
# What developers do:
mcp-server --api-key="sk-1234567890abcdef" --db-password="MyS3cr3t!" --port=3000

# What attackers see with 'ps aux':
USER  PID  COMMAND
alice 1234 mcp-server --api-key=sk-1234567890abcdef --db-password=MyS3cr3t! --port=3000
      ↑
      Visible to ALL users on the system!
```

### Attack Vectors

#### Simple Process Listing
```bash
# Any user can see all process arguments
$ ps aux | grep mcp
alice 1234 node mcp-server.js --token=secret-token-123 --admin-key=admin123

# Even easier with pgrep
$ pgrep -a mcp
1234 mcp-server --api-key=sk-production-key --webhook-secret=whsec_123
```

#### Historical Exposure
```bash
# Process arguments logged in:
- /var/log/audit/audit.log
- System monitoring tools
- Process accounting logs
- Security scanners
- Container logs

# Credentials persist long after process ends!
```

#### Automated Harvesting
```python
# Attacker script running on compromised system
import subprocess
import re

def harvest_mcp_credentials():
    ps_output = subprocess.check_output(['ps', 'aux']).decode()
    
    # Extract API keys
    api_keys = re.findall(r'--api-key[= ]([^ ]+)', ps_output)
    
    # Extract passwords
    passwords = re.findall(r'--password[= ]([^ ]+)', ps_output)
    
    # Extract tokens
    tokens = re.findall(r'--token[= ]([^ ]+)', ps_output)
    
    return {
        'api_keys': api_keys,
        'passwords': passwords,
        'tokens': tokens
    }
```

### Real-World MCP Examples
```bash
# Claude Desktop spawning MCP servers
claude-desktop --spawn "npx mcp-filesystem --api-key=$API_KEY"

# Docker containers
docker run mcp-server --env API_KEY=visible-in-docker-inspect

# Systemd services
ExecStart=/usr/bin/mcp-server --oauth-token=ghp_1234567890
```

## 2. Database Network Connection Security

### Standard Database Vulnerabilities in MCP Context

#### Connection String Exposure
```javascript
// Common MCP database patterns
const mcpServer = new MCPServer({
    database: "postgresql://user:password@db.internal:5432/mcp_data"
    //                          ^^^^^^^^ Plaintext password in connection string
});

// MongoDB with credentials
mongodb://mcp_user:SuperSecret123@mongo.internal:27017/mcp?authSource=admin
```

#### Network Traffic Interception
```
MCP Server → Database Server

Without TLS:
- Credentials sent in plaintext
- Queries visible to network sniffers
- Results (including tokens) exposed

Even with TLS:
- Certificate validation often skipped
- MITM possible with weak validation
- Connection pooling leaks
```

#### Database-Specific MCP Risks
```sql
-- MCP often stores sensitive data
SELECT oauth_token, refresh_token, api_key 
FROM mcp_credentials 
WHERE user_id = 'current_user';

-- Attackers target these tables
-- SQL injection = all tokens exposed
```

### Attack Scenarios

#### 1. Connection Pool Poisoning
```python
# MCP servers often use connection pools
# One poisoned connection affects all users
def poison_connection_pool():
    # Inject malicious connection
    # All subsequent queries compromised
    # Tokens/credentials harvested silently
```

#### 2. Database Audit Log Mining
```sql
-- Database logs contain:
-- Failed auth attempts (with passwords!)
-- Successful queries (showing tokens)
-- Connection strings (in error messages)

SELECT * FROM pg_stat_activity 
WHERE query LIKE '%token%';
```

## 3. Configuration Storage - Credential Files

### Common MCP Configuration Patterns

#### Plain Text JSON
```json
// ~/.mcp/config.json
{
  "servers": {
    "github": {
      "token": "ghp_RealGitHubTokenHere123",
      "type": "oauth"
    },
    "openai": {
      "api_key": "sk-RealOpenAIKeyHere456",
      "org_id": "org-MyOrgId789"
    },
    "database": {
      "connection": "postgres://user:password@localhost/mcp"
    }
  }
}
```

#### Environment File Exposure
```bash
# .env files in MCP projects
MCP_GITHUB_TOKEN=ghp_1234567890abcdef
MCP_OPENAI_KEY=sk-abcdef1234567890
MCP_DB_PASSWORD=MyDatabasePassword123
MCP_ADMIN_SECRET=SuperSecretAdminKey

# Often committed to git!
# Or readable by other processes
```

#### XML Configuration
```xml
<!-- mcp-config.xml -->
<mcp-configuration>
  <credentials>
    <github token="ghp_exposed_token"/>
    <slack webhook="https://hooks.slack.com/secret"/>
    <aws key="AKIA_EXPOSED_KEY" secret="exposed_secret"/>
  </credentials>
</mcp-configuration>
```

### File Permission Vulnerabilities
```bash
# Common misconfigurations
$ ls -la ~/.mcp/
-rw-r--r-- 1 alice alice 2048 Jan 1 config.json
           ↑
     World readable! Any process can read!

# Should be:
-rw------- 1 alice alice 2048 Jan 1 config.json
```

### Configuration Injection Attacks
```json
// User controls part of config
{
  "user_settings": {
    "theme": "dark",
    "__proto__": {
      "admin_token": "injected_token"
    }
  }
}
```

## Composite Attack Chain

### Full Credential Compromise Flow
```
1. Process Arguments → Discover MCP server running
   ps aux | grep mcp → --config=/home/user/.mcp/config.json

2. Read Config File → Extract database credentials
   cat /home/user/.mcp/config.json → postgres://user:pass@db:5432

3. Connect to Database → Dump all tokens
   psql -h db -U user → SELECT * FROM oauth_tokens;

4. Use Tokens → Access all integrated services
   GitHub, Slack, AWS, OpenAI - all compromised!
```

## Detection and Monitoring

### Process Argument Monitoring
```python
# Detect credential exposure in process args
def monitor_process_credentials():
    dangerous_args = [
        '--password', '--api-key', '--token',
        '--secret', '--credential', '--auth'
    ]
    
    for proc in psutil.process_iter(['pid', 'name', 'cmdline']):
        cmdline = ' '.join(proc.info['cmdline'] or [])
        for arg in dangerous_args:
            if arg in cmdline:
                alert(f"Credential in process args: {proc.info['name']}")
```

### Configuration File Monitoring
```bash
# Watch for credential files
inotifywait -m -r ~/.mcp/ -e modify,create |
while read path action file; do
    if [[ "$file" =~ (config|credential|secret) ]]; then
        check_file_permissions "$path/$file"
        scan_for_exposed_credentials "$path/$file"
    fi
done
```

## Mitigation Strategies

### 1. Process Arguments
- Use environment variables (still risky but better)
- Read credentials from files
- Use credential management services
- Implement proper secret injection

### 2. Database Connections
- Always use TLS/SSL
- Rotate credentials regularly
- Use connection pooling carefully
- Implement query auditing

### 3. Configuration Storage
- Encrypt configuration files
- Use proper file permissions (600)
- Never commit credentials to git
- Use secret management tools

## The Bottom Line

These three vectors create a **credential exposure triangle**:
- Process boundaries leak to all users
- Database connections leak over network
- Config files leak to filesystem

Together, they ensure that MCP credentials are exposed at multiple points, making comprehensive security nearly impossible without fundamental architectural changes.