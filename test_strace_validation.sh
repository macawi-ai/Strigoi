#!/bin/bash
# Test script to validate strace PTY capture

echo "=== Strigoi Strace Validation Test ==="
echo "Starting PTY test process..."

# Start the PTY test process
python3 test_pty.py > /tmp/pty_output.log 2>&1 &
PTY_PID=$!

# Wait for child process to start
sleep 0.5

# Get the actual child PID (the one using PTY)
CHILD_PID=$(ps --ppid $PTY_PID -o pid= | tr -d ' ')

if [ -z "$CHILD_PID" ]; then
    echo "Error: Could not find child PTY process"
    kill $PTY_PID 2>/dev/null
    exit 1
fi

echo "Parent PID: $PTY_PID"
echo "Child PTY PID: $CHILD_PID"

# Clear old log
rm -f stream-monitor.jsonl

# Start monitoring with strace
echo "Starting Strigoi monitoring with strace..."
timeout 25s ./strigoi probe center --target $CHILD_PID --show-activity --enable-strace --no-display --output test-strace.jsonl &
STRIGOI_PID=$!

# Wait for monitoring to complete
wait $STRIGOI_PID

echo ""
echo "=== Validation Results ==="

# Check if we captured the test credentials
echo -n "Checking for API_KEY capture: "
if grep -q "sk-test-1234567890abcdef" test-strace.jsonl 2>/dev/null; then
    echo "✓ FOUND"
else
    echo "✗ NOT FOUND"
fi

echo -n "Checking for DATABASE_PASSWORD capture: "
if grep -q "SecretPass123!" test-strace.jsonl 2>/dev/null; then
    echo "✓ FOUND"
else
    echo "✗ NOT FOUND"
fi

echo -n "Checking for TOKEN capture: "
if grep -q "Bearer-eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" test-strace.jsonl 2>/dev/null; then
    echo "✓ FOUND"
else
    echo "✗ NOT FOUND"
fi

echo -n "Checking for AWS_SECRET_ACCESS_KEY capture: "
if grep -q "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" test-strace.jsonl 2>/dev/null; then
    echo "✓ FOUND"
else
    echo "✗ NOT FOUND"
fi

echo ""
echo "=== Strace Activity Summary ==="
echo -n "Total activity events: "
grep -c '"type":"activity"' test-strace.jsonl 2>/dev/null || echo "0"

echo -n "Total vulnerability events: "
grep -c '"type":"vulnerability"' test-strace.jsonl 2>/dev/null || echo "0"

echo ""
echo "=== Sample Activity (last 5) ==="
grep '"type":"activity"' test-strace.jsonl 2>/dev/null | tail -5 | jq -r '.data.preview' 2>/dev/null || echo "No activity found"

# Cleanup
kill $PTY_PID 2>/dev/null
kill $CHILD_PID 2>/dev/null

echo ""
echo "Test complete."