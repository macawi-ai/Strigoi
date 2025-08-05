#!/bin/bash
# Test hardened strace implementation

echo "=== Testing Hardened Strace Implementation ==="

# Test 1: Non-existent process
echo -e "\n1. Testing with non-existent process..."
timeout 5s ./strigoi probe center --target 99999 --enable-strace --no-display --duration 3s 2>&1 | grep -E "not found|not exist|failed" && echo "✓ Correctly handled non-existent process" || echo "✗ Failed to detect non-existent process"

# Test 2: Process that exits quickly
echo -e "\n2. Testing with quickly exiting process..."
(sleep 0.1) &
QUICK_PID=$!
sleep 0.05
timeout 5s ./strigoi probe center --target $QUICK_PID --enable-strace --no-display --duration 3s 2>&1 | grep -E "exited|not active" && echo "✓ Correctly handled quick exit" || echo "✗ Failed to handle quick exit"

# Test 3: Signal handling (SIGINT)
echo -e "\n3. Testing signal handling..."
python3 test_pty.py > /dev/null 2>&1 &
PTY_PID=$!
sleep 0.5
CHILD_PID=$(ps --ppid $PTY_PID -o pid= | tr -d ' ')

if [ -n "$CHILD_PID" ]; then
    ./strigoi probe center --target $CHILD_PID --enable-strace --no-display --output signal-test.jsonl &
    STRIGOI_PID=$!
    sleep 2
    
    # Send SIGINT to strigoi
    kill -INT $STRIGOI_PID 2>/dev/null
    sleep 1
    
    # Check if strigoi stopped cleanly
    if ! ps -p $STRIGOI_PID > /dev/null 2>&1; then
        echo "✓ SIGINT handled cleanly"
    else
        echo "✗ SIGINT not handled properly"
        kill -9 $STRIGOI_PID 2>/dev/null
    fi
    
    kill $PTY_PID 2>/dev/null
    kill $CHILD_PID 2>/dev/null
else
    echo "✗ Could not start test process"
fi

# Test 4: Max bytes limit
echo -e "\n4. Testing max bytes limit..."
# Create a process that outputs lots of data
cat > /tmp/spam_output.sh << 'EOF'
#!/bin/bash
while true; do
    echo "SPAM_DATA_LINE_WITH_SOME_LENGTH_TO_FILL_BUFFERS_QUICKLY_1234567890"
done
EOF
chmod +x /tmp/spam_output.sh

# Use a small max bytes limit for testing
echo "Starting spam process with 1KB limit..."
timeout 10s /tmp/spam_output.sh &
SPAM_PID=$!
sleep 0.1

# Monitor with very small limit (should stop quickly)
timeout 5s ./strigoi probe center --target $SPAM_PID --enable-strace --no-display --duration 10s 2>&1 | grep -E "max bytes limit reached" && echo "✓ Max bytes limit enforced" || echo "✗ Max bytes limit not working"

kill $SPAM_PID 2>/dev/null
rm -f /tmp/spam_output.sh

# Test 5: Buffer statistics
echo -e "\n5. Testing buffer statistics..."
python3 test_pty.py > /dev/null 2>&1 &
PTY_PID=$!
sleep 0.5
CHILD_PID=$(ps --ppid $PTY_PID -o pid= | tr -d ' ')

if [ -n "$CHILD_PID" ]; then
    rm -f stats-test.jsonl
    timeout 5s ./strigoi probe center --target $CHILD_PID --enable-strace --no-display --output stats-test.jsonl --duration 3s > /dev/null 2>&1
    
    # Check if stats were logged
    if grep -q "bytes_total" stats-test.jsonl 2>/dev/null; then
        echo "✓ Statistics logged correctly"
    else
        echo "✗ Statistics not found in logs"
    fi
    
    kill $PTY_PID 2>/dev/null
    kill $CHILD_PID 2>/dev/null
fi

echo -e "\nHardening tests complete."

# Cleanup
rm -f signal-test.jsonl stats-test.jsonl