#!/bin/bash

STRIGOI="./strigoi"

echo "================================================"
echo "Testing All Help Levels - Command Line Mode"
echo "================================================"
echo

echo "1. STANDARD HELP (--help)"
echo "========================="
$STRIGOI probe south --help 2>&1 | head -15
echo

echo "2. BRIEF HELP (--brief)"
echo "======================="
$STRIGOI probe south --brief 2>&1 | head -10
echo

echo "3. FULL HELP (--full)"
echo "====================="
$STRIGOI probe south --full 2>&1 | head -35
echo

echo "4. EXAMPLES ONLY (--examples)"
echo "=============================="
$STRIGOI probe south --examples 2>&1 | head -15
echo

echo "================================================"
echo "Testing All Help Levels - Interactive Mode"
echo "================================================"
echo

echo "1. Standard help command:"
echo -e "cd probe\nhelp south\nexit" | $STRIGOI 2>&1 | grep -A10 "dependencies" | head -15
echo

echo "2. Command with --brief:"
echo -e "cd probe\nsouth --brief\nexit" | $STRIGOI 2>&1 | grep -A5 "south:" | head -10
echo

echo "3. Command with --full:"
echo -e "cd probe\nsouth --full\nexit" | $STRIGOI 2>&1 | grep -A10 "Advanced" | head -15
echo

echo "4. Command with --examples:"
echo -e "cd probe\nsouth --examples\nexit" | $STRIGOI 2>&1 | grep -A10 "Example" | head -15
echo

echo "================================================"
echo "Summary of Help Modes:"
echo "================================================"
echo "✓ --help: Standard help with usage, flags, quick examples, hints"
echo "✓ --brief: One-line description with subcommands list"
echo "✓ --full: Everything including advanced options, config, related commands"
echo "✓ --examples: Just the examples with descriptions and output"
echo
echo "Note: -h and --help both trigger standard help (Cobra default)"
echo "      Use --brief for minimal help, --full for comprehensive help"