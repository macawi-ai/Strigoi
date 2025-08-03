#!/usr/bin/env python3
"""
Mock SQLite MCP server that demonstrates the SQL injection privilege amplification attack.
Shows how database credentials in command line arguments can be exploited.
"""
import sys
import json
import sqlite3
import os
import time

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 mock-sqlite-mcp-server.py <sqlite_connection_string>")
        print("Example: python3 mock-sqlite-mcp-server.py 'sqlite:///enterprise-demo.db?admin_mode=true&full_privileges=yes'")
        sys.exit(1)
    
    # Get connection string from command line (SECURITY FLAW!)
    connection_string = sys.argv[1]
    
    # Parse SQLite connection (simplified)
    if connection_string.startswith('sqlite:///'):
        db_path = connection_string.replace('sqlite:///', '').split('?')[0]
    else:
        print("[ERROR] Invalid SQLite connection string", file=sys.stderr)
        sys.exit(1)
    
    print(f"[SQLITE-MCP] Starting SQLite MCP server...", file=sys.stderr)
    print(f"[SQLITE-MCP] Database: {db_path}", file=sys.stderr)
    print(f"[SQLITE-MCP] Connection string: {connection_string}", file=sys.stderr)
    print(f"[SQLITE-MCP] Running with FULL PRIVILEGES (admin mode)", file=sys.stderr)
    print(f"[SQLITE-MCP] PID: {os.getpid()}", file=sys.stderr)
    print(f"[SQLITE-MCP] Ready for JSON-RPC commands on STDIN", file=sys.stderr)
    
    # Check if database exists
    if not os.path.exists(db_path):
        print(f"[ERROR] Database file not found: {db_path}", file=sys.stderr)
        sys.exit(1)
    
    # Connect to SQLite database
    try:
        conn = sqlite3.connect(db_path)
        conn.row_factory = sqlite3.Row  # Enable column access by name
        print(f"[SQLITE-MCP] Connected to database successfully", file=sys.stderr)
    except Exception as e:
        print(f"[ERROR] Failed to connect to database: {e}", file=sys.stderr)
        sys.exit(1)
    
    # Main MCP server loop
    request_count = 0
    while True:
        try:
            # Read JSON-RPC request from stdin
            line = input().strip()
            if not line:
                continue
                
            request_count += 1
            print(f"[SQLITE-MCP] Processing request #{request_count}", file=sys.stderr)
            
            try:
                request = json.loads(line)
                
                # Handle different MCP methods
                if request.get("method") == "database/query":
                    response = handle_query(conn, request)
                elif request.get("method") == "database/execute":
                    response = handle_execute(conn, request)
                elif request.get("method") == "database/schema":
                    response = handle_schema(conn, request)
                else:
                    response = {
                        "jsonrpc": "2.0",
                        "id": request.get("id", 1),
                        "error": {
                            "code": -32601,
                            "message": f"Method not found: {request.get('method')}"
                        }
                    }
                
                # Send response
                print(json.dumps(response))
                sys.stdout.flush()
                
            except json.JSONDecodeError:
                print(f"[SQLITE-MCP] Invalid JSON received: {line[:50]}...", file=sys.stderr)
                continue
                
        except EOFError:
            print(f"[SQLITE-MCP] EOF received, shutting down...", file=sys.stderr)
            break
        except KeyboardInterrupt:
            print(f"\n[SQLITE-MCP] Interrupted, shutting down...", file=sys.stderr)
            break
    
    # Cleanup
    conn.close()
    print(f"[SQLITE-MCP] Processed {request_count} requests", file=sys.stderr)
    print(f"[SQLITE-MCP] Connection to {db_path} closed", file=sys.stderr)

def handle_query(conn, request):
    """Handle database query requests (SELECT statements)"""
    try:
        sql = request["params"]["sql"]
        print(f"[SQLITE-MCP] Executing SQL: {sql[:100]}...", file=sys.stderr)
        
        # Execute the SQL (DANGEROUS - no sanitization!)
        cursor = conn.execute(sql)
        rows = cursor.fetchall()
        
        # Convert rows to list of dictionaries
        result = []
        for row in rows:
            result.append(dict(row))
        
        print(f"[SQLITE-MCP] Query returned {len(result)} rows", file=sys.stderr)
        
        return {
            "jsonrpc": "2.0",
            "id": request.get("id", 1),
            "result": {
                "data": result,
                "row_count": len(result),
                "status": "success"
            }
        }
        
    except Exception as e:
        print(f"[SQLITE-MCP] Query error: {e}", file=sys.stderr)
        return {
            "jsonrpc": "2.0",
            "id": request.get("id", 1),
            "error": {
                "code": -32000,
                "message": f"Database error: {str(e)}"
            }
        }

def handle_execute(conn, request):
    """Handle database execute requests (INSERT, UPDATE, DELETE, etc.)"""
    try:
        sql = request["params"]["sql"]
        print(f"[SQLITE-MCP] Executing SQL: {sql[:100]}...", file=sys.stderr)
        
        # Execute the SQL (DANGEROUS - no sanitization!)
        cursor = conn.execute(sql)
        conn.commit()
        
        print(f"[SQLITE-MCP] Execute completed, {cursor.rowcount} rows affected", file=sys.stderr)
        
        return {
            "jsonrpc": "2.0",
            "id": request.get("id", 1),
            "result": {
                "rows_affected": cursor.rowcount,
                "status": "success"
            }
        }
        
    except Exception as e:
        print(f"[SQLITE-MCP] Execute error: {e}", file=sys.stderr)
        return {
            "jsonrpc": "2.0",
            "id": request.get("id", 1),
            "error": {
                "code": -32000,
                "message": f"Database error: {str(e)}"
            }
        }

def handle_schema(conn, request):
    """Handle schema information requests"""
    try:
        # Get list of tables
        cursor = conn.execute("SELECT name FROM sqlite_master WHERE type='table';")
        tables = [row[0] for row in cursor.fetchall()]
        
        schema_info = {}
        for table in tables:
            # Get column information for each table
            cursor = conn.execute(f"PRAGMA table_info({table});")
            columns = []
            for col in cursor.fetchall():
                columns.append({
                    "name": col[1],
                    "type": col[2],
                    "not_null": bool(col[3]),
                    "primary_key": bool(col[5])
                })
            schema_info[table] = columns
        
        print(f"[SQLITE-MCP] Schema info returned for {len(tables)} tables", file=sys.stderr)
        
        return {
            "jsonrpc": "2.0",
            "id": request.get("id", 1),
            "result": {
                "tables": schema_info,
                "status": "success"
            }
        }
        
    except Exception as e:
        print(f"[SQLITE-MCP] Schema error: {e}", file=sys.stderr)
        return {
            "jsonrpc": "2.0",
            "id": request.get("id", 1),
            "error": {
                "code": -32000,
                "message": f"Database error: {str(e)}"
            }
        }

if __name__ == "__main__":
    main()