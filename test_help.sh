#!/bin/bash
echo "Testing help system..."
echo
echo "1. Testing 'help north' in probe directory:"
echo "============================================"
echo -e "cd probe\nhelp north\nexit" | ./strigoi 2>&1 | grep -A10 "Probe north"

echo
echo "2. Testing 'north --help' in probe directory:"
echo "=============================================="
echo -e "cd probe\nnorth --help\nexit" | ./strigoi 2>&1 | grep -A10 "endpoints"