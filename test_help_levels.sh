#!/bin/bash

STRIGOI="./strigoi"

echo "=============================================="
echo "Testing All Help Levels/Modes"
echo "=============================================="
echo

echo "1. BRIEF HELP MODE (-h flag)"
echo "============================="
echo "Command line mode:"
$STRIGOI probe south -h 2>&1 | head -10
echo
echo "Interactive mode:"
echo -e "cd probe\nsouth -h\nexit" | $STRIGOI 2>&1 | grep -A8 "south:"
echo

echo "2. STANDARD HELP MODE (--help flag)"
echo "===================================="
echo "Command line mode:"
$STRIGOI probe south --help 2>&1 | head -15
echo
echo "Interactive mode:"
echo -e "cd probe\nsouth --help\nexit" | $STRIGOI 2>&1 | grep -A10 "Usage:"
echo

echo "3. FULL HELP MODE (--help-full flag)"
echo "====================================="
echo "Command line mode:"
$STRIGOI probe south --help-full 2>&1 | head -30
echo
echo "Interactive mode:"
echo -e "cd probe\nsouth --help-full\nexit" | $STRIGOI 2>&1 | grep -A20 "Advanced"
echo

echo "4. EXAMPLES MODE (--examples flag)"
echo "==================================="
echo "Command line mode:"
$STRIGOI probe south --examples 2>&1 | head -20
echo
echo "Interactive mode:"
echo -e "cd probe\nsouth --examples\nexit" | $STRIGOI 2>&1 | grep -A15 "Example"
echo

echo "5. TESTING HELP COMMAND VARIATIONS"
echo "==================================="
echo "help south (interactive):"
echo -e "cd probe\nhelp south\nexit" | $STRIGOI 2>&1 | grep -A5 "dependencies"
echo
echo "? command (interactive):"
echo -e "cd probe\n?\nexit" | $STRIGOI 2>&1 | grep -A5 "Available"
echo

echo "=============================================="
echo "Summary of Expected Behaviors:"
echo "=============================================="
echo "✓ -h: Brief one-liner with subcommands list"
echo "✓ --help: Standard help with usage, flags, examples, hints"
echo "✓ --help-full: Everything including advanced options, config, related commands"
echo "✓ --examples: Just the examples with descriptions"
echo "✓ help <cmd>: Same as --help for that command"
echo "✓ ?: Context help in interactive mode"