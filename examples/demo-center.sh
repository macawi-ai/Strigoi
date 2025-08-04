#!/bin/bash
# Demo script for Strigoi Center module

echo "=== Strigoi Center Module Demo ==="
echo
echo "This demo shows how the Center module detects credentials and secrets"
echo "flowing through process STDIO streams in real-time."
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d "modules/probe" ]; then
    echo -e "${RED}Error: Please run this script from the Strigoi root directory${NC}"
    exit 1
fi

# Build Strigoi if needed
echo "Building Strigoi..."
if ! go build -o strigoi ./cmd/strigoi; then
    echo -e "${RED}Error: Failed to build Strigoi${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Build successful${NC}"
echo

# Start the vulnerable app in background
echo "Starting vulnerable test application..."
python3 examples/vulnerable-app.py --continuous &
APP_PID=$!
echo "Test app PID: $APP_PID"
sleep 2

# Run Center module monitoring
echo
echo -e "${YELLOW}Starting Strigoi Center monitoring...${NC}"
echo "This will detect credentials in real-time as they flow through the test app."
echo
echo "Press Ctrl+C to stop monitoring"
echo

# Monitor the vulnerable app
./strigoi probe center --target $APP_PID --output center-demo.jsonl --duration 30s

# Clean up
echo
echo "Stopping test application..."
kill $APP_PID 2>/dev/null
wait $APP_PID 2>/dev/null

# Show results summary
echo
echo "=== Demo Complete ==="
echo
echo "Results saved to: center-demo.jsonl"
echo
echo "To view the structured log:"
echo "  cat center-demo.jsonl | jq ."
echo
echo "To see only vulnerabilities:"
echo "  cat center-demo.jsonl | jq 'select(.type==\"vulnerability\")'"
echo
echo "To see severity breakdown:"
echo "  cat center-demo.jsonl | jq -r 'select(.type==\"vulnerability\") | .vuln.severity' | sort | uniq -c"
echo

# Check if vulnerabilities were found
VULN_COUNT=$(cat center-demo.jsonl 2>/dev/null | jq 'select(.type=="vulnerability")' | wc -l)
if [ "$VULN_COUNT" -gt 0 ]; then
    echo -e "${RED}Found $VULN_COUNT vulnerabilities!${NC}"
    exit 1
else
    echo -e "${GREEN}No vulnerabilities found (this shouldn't happen with the test app!)${NC}"
fi