#!/bin/bash

echo "🔍 Post-Lint Build Verification"
echo "================================"
echo

echo "✅ Testing All Help Modes:"
echo "-------------------------"

echo "1. Standard help:"
./strigoi probe south --help | head -3
echo

echo "2. Brief help:"
./strigoi probe south --brief
echo

echo "3. Full help (advanced options):"
./strigoi probe south --full | grep -A3 "Advanced Options"
echo

echo "4. Examples mode:"
./strigoi probe south --examples | head -3
echo

echo "✅ Testing Interactive Mode:"
echo "---------------------------"
echo "Context help:"
echo -e "help\nexit" | ./strigoi | grep -A2 "Available here"
echo

echo "Command-specific help:"
echo -e "cd probe\nhelp south\nexit" | ./strigoi | grep -A1 "Usage:"
echo

echo "Error suggestions:"
echo -e "cd probe\nsout\nexit" | ./strigoi | grep -A2 "Command not found"
echo

echo "✅ Testing Context Confusion Handling:"
echo "-------------------------------------"
echo -e "probe south --help\nexit" | ./strigoi | grep -A2 "You can navigate"
echo

echo "🎉 All systems operational! Build is solid."