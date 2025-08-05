#!/bin/bash
# Test HTTP dissector with stream data

echo "=== Testing HTTP Dissector with Stream Data ==="

# Create a test program that outputs HTTP data to stdout
cat > /tmp/http_stream_test.py << 'EOF'
#!/usr/bin/env python3
import sys
import time

# Output HTTP request to stdout
http_request = """GET /api/v1/users?api_key=sk-test-1234567890abcdef&token=bearer-xyz123 HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U
X-API-Key: api_1234567890_secret_key_abcdef
Cookie: sessionId=abc123def456; PHPSESSID=1234567890abcdef

{"username": "admin", "password": "SecretPass123!"}
"""

print("Sending HTTP request...")
sys.stdout.write(http_request)
sys.stdout.flush()

time.sleep(2)

# Output HTTP response to stdout
http_response = """HTTP/1.1 200 OK
Content-Type: application/json
Set-Cookie: auth_token=secret-session-token-xyz789; HttpOnly
X-Secret-Token: internal-api-token-1234567890

{
  "status": "success",
  "api_key": "AIzaSyDVUzr2jedg1Y6HTa-SERVER-KEY",
  "database_password": "postgres://user:SuperSecret123@db:5432",
  "jwt_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MTIzfQ.R0JhPRo9T3K9rXkiV6XzGOmV7DSlrcFKn-vPqQY3Fgw"
}
"""

print("\nReceiving HTTP response...")
sys.stdout.write(http_response)
sys.stdout.flush()

# Keep alive
time.sleep(5)
print("\nDone.")
EOF

chmod +x /tmp/http_stream_test.py

# Run test and monitor
echo "Starting HTTP stream test..."
python3 /tmp/http_stream_test.py &
TEST_PID=$!
sleep 0.5

# Monitor with Strigoi
echo -e "\nMonitoring process $TEST_PID with Strigoi..."
rm -f http-stream-test.jsonl
timeout 10s ./strigoi probe center --target $TEST_PID --show-activity --no-display --output http-stream-test.jsonl

echo -e "\n=== Analysis Results ==="

# Check for HTTP protocol detection
echo -n "HTTP protocol detected: "
if grep -q '"protocol":"HTTP"' http-stream-test.jsonl 2>/dev/null; then
    echo "✓ YES"
    echo "HTTP frames found:"
    grep '"protocol":"HTTP"' http-stream-test.jsonl | jq -r '.data.protocol' | sort | uniq -c
else
    echo "✗ NO"
fi

# Count vulnerabilities
echo -e "\nVulnerability Summary:"
echo -n "Total vulnerabilities: "
grep -c '"type":"vulnerability"' http-stream-test.jsonl 2>/dev/null || echo "0"

# Show vulnerability types
echo -e "\nVulnerability Types Found:"
grep '"type":"vulnerability"' http-stream-test.jsonl 2>/dev/null | jq -r '.vuln.subtype' | sort | uniq -c

echo -e "\n=== Sample Vulnerabilities ==="

# Show some detected credentials
echo "API Keys:"
grep '"subtype":"api_key' http-stream-test.jsonl 2>/dev/null | jq -r '[.vuln.location, .vuln.evidence] | join(": ")' | head -3

echo -e "\nTokens:"
grep '"subtype":".*token' http-stream-test.jsonl 2>/dev/null | grep -v api_key | jq -r '[.vuln.location, .vuln.evidence] | join(": ")' | head -3

echo -e "\nPasswords:"
grep '"subtype":"password' http-stream-test.jsonl 2>/dev/null | jq -r '[.vuln.location, .vuln.evidence] | join(": ")' | head -3

echo -e "\nSession IDs:"
grep '"subtype":"session' http-stream-test.jsonl 2>/dev/null | jq -r '[.vuln.location, .vuln.evidence] | join(": ")' | head -3

# Show activity
echo -e "\n=== HTTP Activity Captured ==="
grep '"type":"activity"' http-stream-test.jsonl 2>/dev/null | grep -E "(GET|POST|HTTP)" | jq -r '.data.preview' | head -5

# Cleanup
kill $TEST_PID 2>/dev/null
rm -f /tmp/http_stream_test.py

echo -e "\nTest complete."