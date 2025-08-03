#!/bin/bash
# Cleanup script to stop all mock MCP servers

echo "=== Cleaning Up Mock MCP Servers ==="

if [ -f /tmp/mcp-demo-pids.txt ]; then
    echo "[*] Reading PIDs from /tmp/mcp-demo-pids.txt"
    PIDS=$(cat /tmp/mcp-demo-pids.txt)
    
    for PID in $PIDS; do
        if kill -0 $PID 2>/dev/null; then
            echo "[*] Stopping process $PID..."
            kill $PID 2>/dev/null
        fi
    done
    
    # Give processes time to exit cleanly
    sleep 1
    
    # Force kill any remaining
    for PID in $PIDS; do
        if kill -0 $PID 2>/dev/null; then
            echo "[!] Force killing process $PID..."
            kill -9 $PID 2>/dev/null
        fi
    done
    
    rm -f /tmp/mcp-demo-pids.txt
else
    echo "[*] No PID file found, searching for mock processes..."
    pkill -f "mock-mcp-server.py"
fi

# Clean up logs
if [ -d /tmp/mcp-demo-logs ]; then
    echo "[*] Removing log directory /tmp/mcp-demo-logs"
    rm -rf /tmp/mcp-demo-logs
fi

echo "[*] Cleanup complete"

# Verify
REMAINING=$(ps aux | grep -c "mock-mcp-server.py" | grep -v grep || echo 0)
if [ "$REMAINING" -eq "0" ]; then
    echo "[+] All mock MCP servers stopped successfully"
else
    echo "[!] Warning: Some mock processes may still be running"
    ps aux | grep "mock-mcp-server.py" | grep -v grep
fi