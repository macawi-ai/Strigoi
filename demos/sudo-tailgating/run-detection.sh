#!/bin/bash

# Sudo Tailgating Detection Demo Script
# WHITE HAT - Educational purposes only

echo "╔══════════════════════════════════════════╗"
echo "║   SUDO TAILGATING DETECTION DEMO         ║"
echo "║         Arctic Fox Security              ║"
echo "╚══════════════════════════════════════════╗"
echo

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}This demo shows how to detect sudo tailgating vulnerabilities${NC}"
echo

# Step 1: Run the detection module
echo -e "${YELLOW}Step 1: Running Strigoi sudo cache detection...${NC}"
echo

# Check if strigoi is in PATH or use local build
if command -v strigoi &> /dev/null; then
    STRIGOI_CMD="strigoi"
elif [ -f "$HOME/.strigoi/bin/strigoi" ]; then
    STRIGOI_CMD="$HOME/.strigoi/bin/strigoi"
else
    echo -e "${RED}Error: Strigoi not found. Please build and install first.${NC}"
    exit 1
fi

# Create a command file for strigoi
cat > /tmp/strigoi_commands.txt << EOF
use sudo.cache_detection
run
show
exit
EOF

# Run detection
echo "Running detection..."
$STRIGOI_CMD < /tmp/strigoi_commands.txt

# Step 2: Show current status
echo
echo -e "${YELLOW}Step 2: Current System Status${NC}"
echo

# Check sudo cache
echo -n "Sudo cache status: "
if sudo -n true 2>/dev/null; then
    echo -e "${RED}CACHED - VULNERABLE!${NC}"
    echo "  Any MCP process can now escalate to root!"
else
    echo -e "${GREEN}Not cached - Safe${NC}"
fi

# Count MCPs
echo -n "MCP processes: "
MCP_COUNT=$(pgrep -c mcp 2>/dev/null || echo "0")
echo "$MCP_COUNT found"

# Show timeout
echo -n "Sudo timeout setting: "
TIMEOUT=$(sudo -l 2>/dev/null | grep -oP 'timestamp_timeout=\K\d+' || echo "15")
echo "$TIMEOUT minutes"

# Step 3: Recommendations
echo
echo -e "${YELLOW}Step 3: Security Recommendations${NC}"
echo

if [ "$TIMEOUT" != "0" ]; then
    echo -e "${RED}⚠️  WARNING: Sudo caching is enabled!${NC}"
    echo
    echo "To fix immediately:"
    echo "  1. Clear cache now: sudo -k"
    echo "  2. Disable permanently: echo 'Defaults timestamp_timeout=0' | sudo tee -a /etc/sudoers"
else
    echo -e "${GREEN}✓ Sudo caching is disabled - Good security posture${NC}"
fi

if [ "$MCP_COUNT" -gt 0 ]; then
    echo
    echo "MCP isolation recommendations:"
    echo "  • Run MCPs in separate user contexts"
    echo "  • Use containerization (Docker/Podman)"
    echo "  • Enable audit logging for sudo usage"
fi

# Step 4: Demo the vulnerability (safely)
echo
echo -e "${YELLOW}Step 4: Understanding the Attack${NC}"
echo

cat << 'EOF'
The attack works like this:

1. You run: sudo apt update
   └─> Enter your password

2. Sudo caches your credentials (default: 15 minutes)
   └─> No password needed for subsequent sudo commands

3. Rogue MCP detects the cache
   └─> Runs: sudo -n true (exit 0 = cached)

4. Rogue MCP exploits the cache
   └─> sudo -n <any command as root>

This is why we must:
- Disable sudo caching (timestamp_timeout=0)
- Isolate MCP processes
- Monitor sudo usage patterns

Remember: We detect and protect, never exploit!
EOF

echo
echo -e "${GREEN}Demo complete. Stay safe, stay WHITE HAT! 🎩${NC}"

# Cleanup
rm -f /tmp/strigoi_commands.txt