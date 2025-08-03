#!/bin/bash
# Launch multiple mock MCP servers simulating a typical deployment
# All running as the same user (catastrophic security model)

echo "=== Launching Mock MCP Server Environment ==="
echo "[*] Simulating typical Claude Desktop MCP deployment"
echo "[*] All servers will run as user: $(whoami)"
echo

# Clean up any existing mock servers
echo "[*] Cleaning up any existing mock servers..."
pkill -f "mock-mcp-server.py" 2>/dev/null
sleep 1

# Create a directory for server logs
mkdir -p /tmp/mcp-demo-logs

# Launch Database Server
echo "[1/6] Starting Database MCP Server..."
MOCK_SERVER_TYPE=database python3 mock-servers/mock-mcp-server.py \
    "postgresql://admin:Pr0duct!onP@ss2024@prod-db.internal:5432/finance" \
    > /tmp/mcp-demo-logs/database.log 2>&1 &
DB_PID=$!
echo "    PID: $DB_PID"
echo "    Credentials visible in: ps aux | grep $DB_PID"

# Launch Filesystem Server  
echo "[2/6] Starting Filesystem MCP Server..."
MOCK_SERVER_TYPE=filesystem \
FS_ALLOW_HIDDEN=true \
python3 mock-servers/mock-mcp-server.py \
    "/home/$(whoami)/documents" "/home/$(whoami)/projects" \
    > /tmp/mcp-demo-logs/filesystem.log 2>&1 &
FS_PID=$!
echo "    PID: $FS_PID"

# Launch Slack Server
echo "[3/6] Starting Slack MCP Server..."
MOCK_SERVER_TYPE=slack \
SLACK_BOT_TOKEN="xoxb-12345678900-1234567890123-ABCdefGHIjklMNOpqrsTUVwx" \
SLACK_APP_TOKEN="xapp-1-A12345678-1234567890123-abcdef" \
python3 mock-servers/mock-mcp-server.py \
    > /tmp/mcp-demo-logs/slack.log 2>&1 &
SLACK_PID=$!
echo "    PID: $SLACK_PID"
echo "    Token visible in: cat /proc/$SLACK_PID/environ"

# Launch GitHub Server
echo "[4/6] Starting GitHub MCP Server..."
MOCK_SERVER_TYPE=github \
GITHUB_TOKEN="ghp_1234567890ABCDEFghijKLMNopQRSTuvWXyz12" \
GITHUB_ORG="acme-corp" \
python3 mock-servers/mock-mcp-server.py \
    > /tmp/mcp-demo-logs/github.log 2>&1 &
GH_PID=$!
echo "    PID: $GH_PID"

# Launch AWS Server
echo "[5/6] Starting AWS MCP Server..."
MOCK_SERVER_TYPE=aws \
AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE" \
AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" \
AWS_REGION="us-east-1" \
python3 mock-servers/mock-mcp-server.py \
    > /tmp/mcp-demo-logs/aws.log 2>&1 &
AWS_PID=$!
echo "    PID: $AWS_PID"
echo "    AWS keys visible in environment!"

# Launch JIRA Server
echo "[6/6] Starting JIRA MCP Server..."
MOCK_SERVER_TYPE=jira \
JIRA_USER="alice@acme.com" \
JIRA_API_TOKEN="ATATT3xFfGF0ABCDEFGHIJKLMNOPQRSTUVWXYZ" \
python3 mock-servers/mock-mcp-server.py \
    "--url" "https://acme.atlassian.net" \
    > /tmp/mcp-demo-logs/jira.log 2>&1 &
JIRA_PID=$!
echo "    PID: $JIRA_PID"

# Save PIDs for cleanup
echo "$DB_PID $FS_PID $SLACK_PID $GH_PID $AWS_PID $JIRA_PID" > /tmp/mcp-demo-pids.txt

echo
echo "=== Mock MCP Environment Running ==="
echo "[*] 6 MCP servers now running as user: $(whoami)"
echo "[*] All credentials and tokens are exposed to same-user attacks"
echo "[*] Server logs in: /tmp/mcp-demo-logs/"
echo
echo "[!] In a real deployment, these would be PRODUCTION credentials!"
echo
echo "Run ./attack-demo.sh to see how easily these can be compromised"
echo "Run ./cleanup-demo.sh to stop all mock servers"
echo
echo "Press Ctrl+C to stop watching, servers will continue running..."

# Show live process listing
echo
echo "=== Current MCP Processes ==="
ps aux | grep -E "mock-mcp-server" | grep -v grep