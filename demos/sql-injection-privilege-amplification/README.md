# SQL Injection + Privilege Amplification Demo

This demonstration shows how MCP's architecture transforms simple SQL injection into database administrative takeover through privilege inheritance.

## What This Demo Shows

1. **Credential Extraction** - Finding database credentials in MCP process arguments
2. **JSON-RPC SQL Injection** - Injecting SQL through MCP's interface
3. **Privilege Amplification** - How user-level compromise becomes admin-level database access
4. **Impact Demonstration** - What attackers can accomplish with inherited privileges

## Safe SQLite Demonstration

This demo uses SQLite to safely show the attack pattern without requiring real database infrastructure.

## Demo Components

- `setup-demo-db.sh` - Creates SQLite database with realistic sensitive data
- `mock-sqlite-mcp-server.py` - MCP server that connects to SQLite with "admin" privileges
- `sql-injection-attack.sh` - Demonstrates the complete attack chain
- `impact-analysis.sh` - Shows what the attacker gained access to

## Running the Demo

### Step 1: Set Up Demo Database
```bash
./setup-demo-db.sh
```

Creates `enterprise-demo.db` with:
- Customer PII (SSNs, credit cards)
- Employee records (salaries, personal info)
- Financial transactions
- Executive compensation data

### Step 2: Launch Mock MCP Server
```bash
./launch-mock-server.sh
```

Starts MCP server with SQLite connection string visible in process arguments.

### Step 3: Run Attack Demonstration
```bash
./sql-injection-attack.sh
```

Shows the complete attack chain:
1. Process enumeration to find credentials
2. JSON-RPC injection to execute arbitrary SQL
3. Data exfiltration using inherited privileges
4. Persistent backdoor creation

### Step 4: View Impact Analysis
```bash
./impact-analysis.sh
```

Analyzes what data was compromised and calculates business impact.

## Key Learning Points

1. **No Privilege Escalation Needed** - User account compromise = database admin
2. **Credential Visibility** - Database passwords visible in process listings
3. **Arbitrary SQL Execution** - MCP allows any SQL through JSON-RPC
4. **Maximum Privilege Inheritance** - Attacker gains all database permissions
5. **Invisible to Network Security** - Local STDIO bypasses monitoring

## Comparison: Traditional vs MCP SQL Injection

### Traditional Web App SQL Injection
- Limited to application database user privileges
- Often restricted to specific tables
- WAF/IDS can detect and block
- Network traffic analysis possible

### MCP SQL Injection (This Demo)
- Full database administrative privileges
- Access to all tables and system functions
- No network security detection
- Appears as legitimate MCP operation

## Warning

This demonstration uses mock data and SQLite. In a real environment with PostgreSQL/MySQL and SUPERUSER privileges, the same attack would enable:
- File system access
- Command execution
- Cross-database breaches
- Persistent malware installation
- Complete infrastructure compromise

**Never deploy MCP with database administrative credentials!**