#!/bin/bash

# Demo script for the enhanced Strigoi help system
# Shows how the new context-aware, progressive disclosure help works

STRIGOI="../strigoi"

echo "======================================"
echo "Strigoi Enhanced Help System Demo"
echo "======================================"
echo

echo "1. Testing interactive mode context awareness"
echo "----------------------------------------------"
echo "When users type 'probe south --help' at root, we guide them:"
echo
echo -e "probe south --help\nexit" | $STRIGOI 2>/dev/null | sed -n '/You can navigate/,+3p'
echo

echo "2. Testing error suggestions in interactive mode"
echo "-------------------------------------------------"
echo "When users type an unknown command, we suggest similar ones:"
echo
echo -e "cd probe\nsout\nexit" | $STRIGOI 2>/dev/null | grep -A5 "Command not found"
echo

echo "3. Testing contextual help in probe directory"
echo "----------------------------------------------"
echo -e "cd probe\nhelp\nexit" | $STRIGOI 2>/dev/null | grep -A10 "Current context: /probe"
echo

echo "4. Testing standard command-line help"
echo "--------------------------------------"
$STRIGOI probe south --help 2>/dev/null | head -15
echo

echo "5. Testing help for built-in commands"
echo "--------------------------------------"
echo -e "help cd\nexit" | $STRIGOI 2>/dev/null | grep -A5 "cd: Built-in"
echo

echo "======================================"
echo "Key Improvements:"
echo "======================================"
echo "✓ Context-aware help in interactive mode"
echo "✓ Smart error messages with command suggestions"
echo "✓ Clear navigation guidance"
echo "✓ Progressive disclosure (brief, standard, full)"
echo "✓ Rich examples and contextual hints"
echo "✓ Handles 'probe south --help' confusion gracefully"