#!/usr/bin/env python3
"""
Mock MCP server that simulates a real MCP server's behavior.
Used for safe demonstration of same-user security issues.
"""
import sys
import json
import time
import os

def main():
    server_type = os.environ.get('MOCK_SERVER_TYPE', 'generic')
    
    # Simulate startup with visible secrets
    print(f"[{server_type.upper()}] Starting MCP {server_type} server...", file=sys.stderr)
    
    # Show what secrets we have (simulating real server behavior)
    if 'TOKEN' in os.environ:
        for key, value in os.environ.items():
            if 'TOKEN' in key or 'KEY' in key or 'PASSWORD' in key:
                print(f"[{server_type.upper()}] Loaded secret: {key}={value[:10]}...", file=sys.stderr)
    
    # Show command line args (containing secrets)
    if len(sys.argv) > 1:
        print(f"[{server_type.upper()}] Connection string: {sys.argv[1][:20]}...", file=sys.stderr)
    
    print(f"[{server_type.upper()}] Ready for JSON-RPC commands", file=sys.stderr)
    
    # Simulate MCP server main loop
    while True:
        try:
            # In real MCP, this would be reading JSON-RPC from stdin
            line = input()
            
            # Simulate processing
            try:
                request = json.loads(line)
                
                # Mock response
                response = {
                    "jsonrpc": "2.0",
                    "id": request.get("id", 1),
                    "result": {
                        "status": "success",
                        "server": server_type,
                        "message": f"Processed {request.get('method', 'unknown')} method"
                    }
                }
                
                # Include some "sensitive" data in responses
                if request.get("method") == "query":
                    response["result"]["data"] = [
                        {"id": 1, "user": "alice", "balance": "$12,345.67"},
                        {"id": 2, "user": "bob", "balance": "$98,765.43"}
                    ]
                
                print(json.dumps(response))
                sys.stdout.flush()
                
            except json.JSONDecodeError:
                # Echo non-JSON (like our echo-server)
                print(f"[{server_type.upper()}] ECHO: {line}")
                sys.stdout.flush()
                
        except EOFError:
            break
        except KeyboardInterrupt:
            print(f"\n[{server_type.upper()}] Shutting down...", file=sys.stderr)
            break
    
    # Simulate cleanup (showing more secrets)
    print(f"[{server_type.upper()}] Closing connections...", file=sys.stderr)
    if len(sys.argv) > 1:
        print(f"[{server_type.upper()}] Disconnected from: {sys.argv[1]}", file=sys.stderr)

if __name__ == "__main__":
    main()