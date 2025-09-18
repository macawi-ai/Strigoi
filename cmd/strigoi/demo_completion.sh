#!/bin/bash
# Demo script showing Cobra's multi-word TAB completion

echo "🎯 Strigoi Cobra Multi-Word TAB Completion Demo"
echo "=============================================="
echo

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}Setting up completion...${NC}"
./strigoi-cobra completion bash > /tmp/strigoi-demo.bash
echo "complete -o default -o nospace -F __start_strigoi ./strigoi-cobra" >> /tmp/strigoi-demo.bash

echo -e "\n${GREEN}✓ TAB completion is now available!${NC}"
echo -e "\n${YELLOW}Examples of multi-word completion:${NC}"
echo
echo "1. Root level:"
echo "   ./strigoi-cobra [TAB]"
echo "   → Shows: completion, probe, stream"
echo
echo "2. After 'probe':"
echo "   ./strigoi-cobra probe [TAB]"
echo "   → Shows: all, east, north, south, west"
echo
echo "3. After 'probe north' (with target suggestions):"
echo "   ./strigoi-cobra probe north [TAB]"
echo "   → Shows: localhost, api.example.com, https://target.com"
echo
echo "4. Flags completion:"
echo "   ./strigoi-cobra probe north --[TAB]"
echo "   → Shows: --verbose, --output, --timeout, --follow-redirects, --headers"
echo
echo -e "\n${BLUE}Comparing with readline approach:${NC}"
echo "✅ Cobra: Context-aware completions at every level"
echo "❌ Readline: Lost context after first word"
echo
echo -e "\n${GREEN}Run this to test interactively:${NC}"
echo "source /tmp/strigoi-demo.bash"
echo "./strigoi-cobra [start typing and use TAB]"