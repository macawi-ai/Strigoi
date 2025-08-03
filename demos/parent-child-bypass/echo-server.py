#!/usr/bin/env python3
"""
Simple echo server that simulates an MCP server receiving credentials.
This demonstrates the parent-child YAMA bypass vulnerability.

Usage: python echo-server.py <connection_string>
Example: python echo-server.py "user:secretpassword@localhost:5432/mydb"
"""
import sys
import time
import json

def main():
    if len(sys.argv) < 2:
        print("Usage: python echo-server.py <connection_string>")
        sys.exit(1)
    
    # Simulate receiving credentials via command line (BAD PRACTICE!)
    connection_string = sys.argv[1]
    
    print(f"[SERVER] Starting echo server...")
    print(f"[SERVER] Initialized with connection: {connection_string[:10]}...")
    
    # Simulate MCP server behavior
    while True:
        try:
            # Read from stdin (simulating MCP JSON-RPC messages)
            line = input()
            
            # Parse as JSON (like MCP would)
            try:
                message = json.loads(line)
                
                # Simulate processing with credentials
                if message.get("method") == "query":
                    response = {
                        "jsonrpc": "2.0",
                        "id": message.get("id"),
                        "result": {
                            "data": f"Query executed on {connection_string}",
                            "status": "success"
                        }
                    }
                else:
                    response = {
                        "jsonrpc": "2.0",
                        "id": message.get("id"),
                        "result": f"Echo: {message}"
                    }
                
                # Send response
                print(json.dumps(response))
                sys.stdout.flush()
                
            except json.JSONDecodeError:
                # Echo raw messages
                print(f"[ECHO] {line}")
                sys.stdout.flush()
                
        except EOFError:
            break
        except KeyboardInterrupt:
            print("\n[SERVER] Shutting down...")
            break
    
    print(f"[SERVER] Final connection used: {connection_string}")

if __name__ == "__main__":
    main()