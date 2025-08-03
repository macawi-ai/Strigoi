#!/bin/bash
# Gemini A2A Query Tool
# Clean interface for AI-to-AI communication

set -e

# Configuration
CONTEXT_DIR="$HOME/.strigoi/gemini-context"
mkdir -p "$CONTEXT_DIR"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to display usage
usage() {
    echo "Usage: $0 <command> [options]"
    echo
    echo "Commands:"
    echo "  query <prompt>              - Query Gemini with a prompt"
    echo "  analyze <path> <query>      - Analyze codebase with Gemini"
    echo "  remember <topic> <info>     - Store information for Gemini to remember"
    echo "  recall <topic>              - Ask Gemini to recall stored information"
    echo "  context <key> <content>     - Store context with a key"
    echo
    echo "Examples:"
    echo "  $0 query 'What are the best practices for DuckDB schema design?'"
    echo "  $0 analyze . 'Find all cybernetic patterns in this codebase'"
    echo "  $0 remember 'strigoi-design' 'We chose DuckDB for its embedded nature'"
    echo "  $0 recall 'strigoi-design'"
    exit 1
}

# Check if gemini is available
if ! command -v gemini &> /dev/null; then
    echo -e "${YELLOW}⚠️  Gemini CLI not found${NC}"
    echo "Please install gemini-cli to use this tool"
    exit 1
fi

# Parse command
if [ $# -lt 1 ]; then
    usage
fi

COMMAND="$1"
shift

case "$COMMAND" in
    query)
        if [ $# -lt 1 ]; then
            echo "Error: query requires a prompt"
            usage
        fi
        PROMPT="$*"
        echo -e "${BLUE}🤖 Querying Gemini...${NC}"
        gemini --prompt "$PROMPT"
        ;;
        
    analyze)
        if [ $# -lt 2 ]; then
            echo "Error: analyze requires <path> and <query>"
            usage
        fi
        PATH_TO_ANALYZE="$1"
        shift
        QUERY="$*"
        
        echo -e "${BLUE}📊 Analyzing codebase...${NC}"
        
        # Create analysis context
        ANALYSIS_FILE="$CONTEXT_DIR/analysis_$(date +%Y%m%d_%H%M%S).context"
        
        # Collect relevant files
        {
            echo "=== CODEBASE ANALYSIS ==="
            echo "Path: $PATH_TO_ANALYZE"
            echo "Date: $(date)"
            echo
            
            # Go files
            find "$PATH_TO_ANALYZE" -name "*.go" -type f 2>/dev/null | while read -r file; do
                echo "=== File: $file ==="
                cat "$file"
                echo
            done
            
            # Documentation
            find "$PATH_TO_ANALYZE" -name "*.md" -type f 2>/dev/null | while read -r file; do
                echo "=== Doc: $file ==="
                cat "$file"
                echo
            done
        } > "$ANALYSIS_FILE"
        
        # Create analysis prompt
        FULL_PROMPT="I'm providing you with a codebase to analyze. Please analyze it with the following query:

$QUERY

Here's the codebase:

$(cat "$ANALYSIS_FILE")"
        
        gemini --prompt "$FULL_PROMPT"
        
        # Clean up large analysis file
        rm -f "$ANALYSIS_FILE"
        ;;
        
    remember)
        if [ $# -lt 2 ]; then
            echo "Error: remember requires <topic> and <information>"
            usage
        fi
        TOPIC="$1"
        shift
        INFO="$*"
        
        echo -e "${BLUE}💾 Storing memory...${NC}"
        
        # Append to persistent memory
        MEMORY_FILE="$CONTEXT_DIR/persistent_memory.txt"
        {
            echo "[$(date +%Y-%m-%d_%H:%M:%S)] Topic: $TOPIC"
            echo "$INFO"
            echo "---"
            echo
        } >> "$MEMORY_FILE"
        
        # Confirm with Gemini
        gemini --prompt "I'm storing this for future reference:
Topic: $TOPIC
Information: $INFO

Please acknowledge and create a brief summary of what you'll remember."
        ;;
        
    recall)
        if [ $# -lt 1 ]; then
            echo "Error: recall requires a topic"
            usage
        fi
        TOPIC="$*"
        
        echo -e "${BLUE}🔍 Recalling information...${NC}"
        
        MEMORY_FILE="$CONTEXT_DIR/persistent_memory.txt"
        if [ -f "$MEMORY_FILE" ]; then
            MEMORY_CONTEXT=$(cat "$MEMORY_FILE")
            gemini --prompt "Here's your persistent memory:

$MEMORY_CONTEXT

Please recall information about: $TOPIC"
        else
            echo -e "${YELLOW}No persistent memory found${NC}"
        fi
        ;;
        
    context)
        if [ $# -lt 2 ]; then
            echo "Error: context requires <key> and <content>"
            usage
        fi
        KEY="$1"
        shift
        CONTENT="$*"
        
        echo -e "${BLUE}📝 Storing context...${NC}"
        
        CONTEXT_FILE="$CONTEXT_DIR/${KEY}.context"
        echo "$CONTENT" > "$CONTEXT_FILE"
        
        echo -e "${GREEN}✅ Context stored with key: $KEY${NC}"
        ;;
        
    *)
        echo "Error: Unknown command '$COMMAND'"
        usage
        ;;
esac