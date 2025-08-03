#!/bin/bash
# Example: Stream Analysis Pipeline
# This demonstrates how Strigoi can be used like tcpdump/Wireshark for STDIO

# Create a named pipe
PIPE="/tmp/strigoi-analysis.pipe"
mkfifo "$PIPE" 2>/dev/null || true

echo "=== Strigoi Stream Analysis Pipeline Demo ==="
echo ""

# Example 1: Real-time pattern detection
echo "1. Real-time Pattern Detection Pipeline:"
echo "   Terminal 1: ./strigoi"
echo "   > stream/tap --auto-discover --output pipe:analysis"
echo ""
echo "   Terminal 2: Watch for security patterns"
echo "   > tail -f /tmp/strigoi-analysis.pipe | jq 'select(.type==\"alert\")'"
echo ""

# Example 2: Stream to analysis server
echo "2. Stream to Analysis Server:"
echo "   # Start analysis server"
echo "   > nc -l 9999 | jq ."
echo ""
echo "   # Stream events"
echo "   > stream/tap --auto-discover --output tcp:localhost:9999"
echo ""

# Example 3: Multi-stage pipeline
echo "3. Multi-Stage Analysis Pipeline:"
cat << 'EOF'
   # Stage 1: Capture
   stream/tap --auto-discover --output file:/tmp/capture.jsonl

   # Stage 2: Filter for specific patterns
   cat /tmp/capture.jsonl | jq 'select(.data | contains("password"))'

   # Stage 3: Generate alerts
   cat /tmp/capture.jsonl | ./analyze-patterns.py --alert-threshold high
EOF
echo ""

# Example 4: Integration with existing tools
echo "4. Integration with Security Tools:"
echo "   # Send to Elasticsearch"
echo "   > stream/tap --auto-discover --output tcp:elastic.local:9200"
echo ""
echo "   # Send to Splunk HEC"
echo "   > stream/tap --auto-discover --output tcp:splunk.local:8088"
echo ""
echo "   # Send to local rsyslog"
echo "   > stream/tap --auto-discover --output integration:syslog"
echo ""

# Example 5: Forensics workflow
echo "5. Forensics Workflow:"
cat << 'EOF'
   # Capture everything to timestamped file
   TIMESTAMP=$(date +%Y%m%d_%H%M%S)
   stream/tap --pid $SUSPICIOUS_PID \
     --follow-children \
     --duration 5m \
     --output file:/forensics/case123/stdio_${TIMESTAMP}.jsonl

   # Later analysis
   cat /forensics/case123/stdio_*.jsonl | \
     ./strigoi-analyze --timeline --detect-injection
EOF
echo ""

# Example 6: Development/Debug workflow
echo "6. Development Debug Workflow:"
cat << 'EOF'
   # Watch MCP server communications in real-time
   stream/tap --auto-discover --output stdout | \
     jq -r 'select(.type=="event") | 
            "\(.timestamp) [\(.direction)] \(.summary)"'

   # Or with color coding
   stream/tap --auto-discover --output stdout | \
     jq -r 'select(.type=="event") | 
            if .direction=="inbound" then 
              "\u001b[32m→ \(.summary)\u001b[0m" 
            else 
              "\u001b[31m← \(.summary)\u001b[0m" 
            end'
EOF
echo ""

echo "=== Advanced Features (Future) ==="
echo ""
echo "• BPF-style filters:"
echo "  stream/tap --filter 'pid==1234 && size>1000'"
echo ""
echo "• Protocol decoding:"
echo "  stream/tap --decode json-rpc --output stdout"
echo ""
echo "• Replay captured streams:"
echo "  stream/replay /tmp/capture.jsonl --speed 2x"
echo ""
echo "• Generate PCAP format for Wireshark:"
echo "  stream/tap --output file:/tmp/stdio.pcap --format pcap"
echo ""