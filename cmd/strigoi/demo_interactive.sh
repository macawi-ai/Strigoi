#!/bin/bash
# Demo of Strigoi Cobra Interactive REPL Mode

echo "🎮 Strigoi Cobra Interactive REPL Demo"
echo "===================================="
echo
echo "The Cobra version now has a full interactive shell!"
echo

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Features:${NC}"
echo "✅ Interactive REPL mode (like original Strigoi)"
echo "✅ Navigation commands: cd, ls, pwd"
echo "✅ Color-coded output (directories blue, commands green)"
echo "✅ Command execution from any directory"
echo "✅ Context-aware prompt shows current location"
echo "✅ History support (up/down arrows)"
echo

echo -e "${BLUE}Demo Commands:${NC}"
echo "1. Start interactive mode:"
echo "   ./strigoi-cobra"
echo
echo "2. Navigate the command tree:"
echo "   strigoi> ls                    # List root directory"
echo "   strigoi> cd probe              # Enter probe directory"
echo "   strigoi/probe> ls              # List probe commands"
echo "   strigoi/probe> pwd             # Show current path"
echo
echo "3. Execute commands:"
echo "   strigoi/probe> north localhost # Run probe north command"
echo "   strigoi> cd stream"
echo "   strigoi/stream> tap nginx      # Run stream tap command"
echo
echo "4. TAB completion (still being enhanced):"
echo "   strigoi> cd pr[TAB]            # Completes to 'probe'"
echo "   strigoi/probe> no[TAB]         # Completes to 'north'"
echo

echo -e "${YELLOW}Comparison with Original:${NC}"
echo "✅ Original: Single-word TAB completion only"
echo "✅ Cobra: Multi-word TAB completion support"
echo "✅ Both: Interactive navigation and execution"
echo

echo -e "${GREEN}Try it now:${NC}"
echo "./strigoi-cobra"