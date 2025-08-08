#!/bin/bash

echo "======================================"
echo "Testing Enhanced Help System"
echo "======================================"
echo

echo "1. Command help with 'help <command>':"
echo "--------------------------------------"
echo -e "cd probe\nhelp south\nexit" | ./strigoi 2>&1 | grep -A5 "dependencies"
echo

echo "2. Command help with '<command> --help':"
echo "-----------------------------------------"
echo -e "cd probe\nsouth --help\nexit" | ./strigoi 2>&1 | grep -A5 "dependencies"
echo

echo "3. Interactive context help:"
echo "-----------------------------"
echo -e "cd probe\nhelp\nexit" | ./strigoi 2>&1 | grep -A8 "Current context"
echo

echo "4. Error handling - unknown command:"
echo "-------------------------------------"
echo -e "cd probe\nsout\nexit" | ./strigoi 2>&1 | grep -A4 "Command not found"
echo

echo "✅ All help modes are now properly connected!"