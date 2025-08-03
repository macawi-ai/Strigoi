#!/bin/bash
# Interface Stability Test Script
# Tests all critical interface features to prevent regressions

echo "=== Strigoi Interface Stability Test ==="
echo ""

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to report test results
report_test() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC} $2"
        ((TESTS_FAILED++))
    fi
}

echo "1. Testing basic navigation..."
OUTPUT=$(echo -e "pwd\nls\ncd probe\npwd\ncd ..\npwd\nexit" | ./strigoi 2>&1)

# Test pwd shows /
echo "$OUTPUT" | grep -q "^/$" 
report_test $? "pwd shows root directory"

# Test cd probe changes directory
echo "$OUTPUT" | grep -q "Current directory: /probe"
report_test $? "cd probe changes directory"

# Test directories shown with /
echo "$OUTPUT" | grep -q "probe/"
report_test $? "Directories shown with trailing slash"

echo ""
echo "2. Testing command execution..."
OUTPUT=$(echo -e "help\nalias\nexit" | ./strigoi 2>&1)

# Test help command works
echo "$OUTPUT" | grep -q "Available commands:"
report_test $? "help command works"

# Test alias command works
echo "$OUTPUT" | grep -q "Configured aliases:"
report_test $? "alias command works"

echo ""
echo "3. Testing error handling..."
OUTPUT=$(echo -e "nosuchcmd\ncd nosuchdir\nexit" | ./strigoi 2>&1)

# Test command not found
echo "$OUTPUT" | grep -q "Command not found: nosuchcmd"
report_test $? "Invalid command shows proper error"

# Test directory not found
echo "$OUTPUT" | grep -q "directory not found: nosuchdir"
report_test $? "Invalid directory shows proper error"

echo ""
echo "4. Visual consistency check..."
OUTPUT=$(echo "exit" | ./strigoi 2>&1)

# Test banner displays
echo "$OUTPUT" | grep -q "Advanced Security Validation Platform"
report_test $? "Banner displays correctly"

# Test quick start guide updated
echo "$OUTPUT" | grep -q "Run './strigoi' to enter interactive mode"
report_test $? "Quick start guide shows correct instructions"

echo ""
echo "=== Test Summary ==="
echo "Passed: $TESTS_PASSED"
echo "Failed: $TESTS_FAILED"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi