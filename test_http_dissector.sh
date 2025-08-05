#!/bin/bash
# Test HTTP dissector functionality

echo "=== Testing HTTP Protocol Dissector ==="

# Create a test server that sends HTTP responses with sensitive data
cat > /tmp/http_test_server.py << 'EOF'
#!/usr/bin/env python3
import socket
import time

def handle_client(client_socket):
    # Receive request
    request = client_socket.recv(1024).decode('utf-8')
    print(f"Received: {request[:50]}...")
    
    # Send HTTP response with various sensitive data
    response = """HTTP/1.1 200 OK
Content-Type: application/json
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
X-API-Key: sk-proj-abcdef1234567890abcdef1234567890
Set-Cookie: sessionId=a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6; HttpOnly; Secure

{
  "message": "Test response",
  "api_key": "AIzaSyDVUzr2jedg1Y6HTa-INTERNAL-KEY",
  "password": "SuperSecretPassword123!",
  "token": "ghp_1234567890abcdefghijklmnopqrstuvwxyz"
}"""
    
    client_socket.send(response.encode('utf-8'))
    client_socket.close()

def run_server():
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(('127.0.0.1', 8888))
    server.listen(1)
    print("HTTP test server listening on port 8888...")
    
    # Accept one connection
    client, addr = server.accept()
    print(f"Connection from {addr}")
    handle_client(client)
    
    server.close()
    print("Server closed")

if __name__ == "__main__":
    run_server()
EOF

chmod +x /tmp/http_test_server.py

# Start the test server
echo "Starting HTTP test server..."
python3 /tmp/http_test_server.py &
SERVER_PID=$!
sleep 1

# Create a client that makes HTTP requests
cat > /tmp/http_test_client.py << 'EOF'
#!/usr/bin/env python3
import socket
import time
import sys

def make_request():
    # Connect to server
    client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    client.connect(('127.0.0.1', 8888))
    
    # Send HTTP request with sensitive data in URL
    request = """GET /api/v1/data?api_key=client-secret-key-1234567890&token=auth-token-xyz HTTP/1.1
Host: localhost:8888
Authorization: Basic YWRtaW46cGFzc3dvcmQ=
X-Custom-Token: custom-auth-token-1234567890
Cookie: PHPSESSID=9876543210abcdef; auth_token=bearer-1234567890

"""
    
    print("Sending HTTP request with sensitive data...")
    client.send(request.encode('utf-8'))
    
    # Receive response
    response = client.recv(4096)
    print("Received response:")
    print(response.decode('utf-8')[:200] + "...")
    
    client.close()
    
    # Keep process alive for monitoring
    print("\nKeeping connection data in memory for 10 seconds...")
    sys.stdout.flush()
    time.sleep(10)

if __name__ == "__main__":
    make_request()
EOF

chmod +x /tmp/http_test_client.py

# Run the client
echo "Starting HTTP client..."
python3 /tmp/http_test_client.py &
CLIENT_PID=$!
sleep 0.5

# Monitor with Strigoi
echo -e "\nMonitoring HTTP traffic with Strigoi..."
rm -f http-test.jsonl
timeout 8s ./strigoi probe center --target $CLIENT_PID --show-activity --no-display --output http-test.jsonl

echo -e "\n=== Analysis Results ==="

# Check for HTTP protocol detection
echo -n "HTTP protocol detected: "
if grep -q '"protocol":"HTTP"' http-test.jsonl 2>/dev/null; then
    echo "✓ YES"
else
    echo "✗ NO"
fi

# Count vulnerabilities found
echo -n "Total vulnerabilities found: "
grep -c '"type":"vulnerability"' http-test.jsonl 2>/dev/null || echo "0"

echo -e "\n=== Detected Sensitive Data ==="

# Check for specific vulnerability types
echo "Checking for API keys in URL:"
grep '"subtype":"api_key_in_url"' http-test.jsonl 2>/dev/null | jq -r '.vuln.evidence' 2>/dev/null | head -3

echo -e "\nChecking for tokens in URL:"
grep '"subtype":"token_in_url"' http-test.jsonl 2>/dev/null | jq -r '.vuln.evidence' 2>/dev/null | head -3

echo -e "\nChecking for Authorization headers:"
grep '"subtype":"basic_auth"' http-test.jsonl 2>/dev/null | jq -r '.vuln.evidence' 2>/dev/null | head -3

echo -e "\nChecking for API keys in headers:"
grep '"location":"HTTP header"' http-test.jsonl 2>/dev/null | grep api_key | jq -r '.vuln.evidence' 2>/dev/null | head -3

echo -e "\nChecking for session cookies:"
grep '"subtype":"session_cookie"' http-test.jsonl 2>/dev/null | jq -r '.vuln.evidence' 2>/dev/null | head -3

echo -e "\nChecking for credentials in body:"
grep '"location":"HTTP body"' http-test.jsonl 2>/dev/null | jq -r '.vuln.evidence' 2>/dev/null | head -3

# Show sample activity
echo -e "\n=== Sample HTTP Activity ==="
grep '"type":"activity"' http-test.jsonl 2>/dev/null | tail -5 | jq -r '.data.preview' 2>/dev/null

# Cleanup
kill $CLIENT_PID 2>/dev/null
kill $SERVER_PID 2>/dev/null
rm -f /tmp/http_test_server.py /tmp/http_test_client.py

echo -e "\nTest complete."