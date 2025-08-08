#!/bin/bash

echo "🔧 Testing Flag Fix"
echo "==================="
echo

echo "1. Testing that commands actually execute (not show help):"
echo "--------------------------------------------------------"
echo "Command: south --scan-mcp"
echo -e "cd probe\nsouth --scan-mcp\nexit" | timeout 10s ./strigoi | head -15
echo

echo "2. Testing that help flags still work:"
echo "--------------------------------------"
echo "Command: south --brief"
echo -e "cd probe\nsouth --brief\nexit" | ./strigoi
echo

echo "Command: south --examples"
echo -e "cd probe\nsouth --examples\nexit" | ./strigoi | head -5
echo

echo "3. Testing command line mode still works:"
echo "-----------------------------------------"
echo "Command: ./strigoi probe south --brief"
./strigoi probe south --brief
echo

echo "✅ Fix verification complete!"