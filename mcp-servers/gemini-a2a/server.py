#!/usr/bin/env python3
"""
Gemini A2A MCP Server
Enables AI-to-AI communication between Claude and Gemini
"""

import asyncio
import json
import os
import subprocess
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional

from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import TextContent, Tool, ToolResult

# Initialize MCP server
app = Server("gemini-a2a")

# Context storage directory
CONTEXT_DIR = Path.home() / ".strigoi" / "gemini-context"
CONTEXT_DIR.mkdir(parents=True, exist_ok=True)

class GeminiError(Exception):
    """Raised when Gemini operations fail"""
    pass

async def call_gemini(prompt: str, context_file: Optional[Path] = None) -> str:
    """Call Gemini CLI with prompt and optional context"""
    cmd = ["gemini"]
    
    if context_file and context_file.exists():
        cmd.extend(["--context-file", str(context_file)])
    
    cmd.extend(["--prompt", prompt])
    
    try:
        result = await asyncio.create_subprocess_exec(
            *cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        stdout, stderr = await result.communicate()
        
        if result.returncode != 0:
            raise GeminiError(f"Gemini failed: {stderr.decode()}")
        
        return stdout.decode()
    except FileNotFoundError:
        # Gemini not installed, return mock response
        return f"[Mock Gemini Response]\nPrompt: {prompt}\nNote: Install gemini-cli for real responses"

@app.list_tools()
async def list_tools() -> List[Tool]:
    """List available A2A tools"""
    return [
        Tool(
            name="query_gemini",
            description="Query Gemini with a prompt and optional context",
            inputSchema={
                "type": "object",
                "properties": {
                    "prompt": {
                        "type": "string",
                        "description": "The prompt to send to Gemini"
                    },
                    "context_key": {
                        "type": "string",
                        "description": "Key for stored context to include"
                    },
                    "store_response": {
                        "type": "boolean",
                        "description": "Whether to store the response for future context"
                    }
                },
                "required": ["prompt"]
            }
        ),
        Tool(
            name="analyze_codebase",
            description="Deep analysis of codebase using Gemini's large context window",
            inputSchema={
                "type": "object",
                "properties": {
                    "path": {
                        "type": "string",
                        "description": "Path to codebase to analyze"
                    },
                    "query": {
                        "type": "string",
                        "description": "Analysis query"
                    },
                    "include_patterns": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "File patterns to include (e.g., '*.go', '*.md')"
                    }
                },
                "required": ["path", "query"]
            }
        ),
        Tool(
            name="store_context",
            description="Store context for future Gemini queries",
            inputSchema={
                "type": "object",
                "properties": {
                    "key": {
                        "type": "string",
                        "description": "Context key for retrieval"
                    },
                    "content": {
                        "type": "string",
                        "description": "Content to store"
                    },
                    "metadata": {
                        "type": "object",
                        "description": "Optional metadata about the context"
                    }
                },
                "required": ["key", "content"]
            }
        ),
        Tool(
            name="gemini_remember",
            description="Ask Gemini to remember something across sessions",
            inputSchema={
                "type": "object",
                "properties": {
                    "topic": {
                        "type": "string",
                        "description": "Topic or key to remember"
                    },
                    "information": {
                        "type": "string",
                        "description": "Information to remember"
                    }
                },
                "required": ["topic", "information"]
            }
        ),
        Tool(
            name="gemini_recall",
            description="Ask Gemini to recall previously stored information",
            inputSchema={
                "type": "object",
                "properties": {
                    "topic": {
                        "type": "string",
                        "description": "Topic or key to recall"
                    },
                    "specific_question": {
                        "type": "string",
                        "description": "Specific question about the topic"
                    }
                },
                "required": ["topic"]
            }
        )
    ]

@app.call_tool()
async def call_tool(name: str, arguments: Any) -> List[ToolResult]:
    """Execute A2A tools"""
    
    if name == "query_gemini":
        prompt = arguments["prompt"]
        context_key = arguments.get("context_key")
        store_response = arguments.get("store_response", False)
        
        # Build context file if key provided
        context_file = None
        if context_key:
            context_file = CONTEXT_DIR / f"{context_key}.context"
        
        # Call Gemini
        response = await call_gemini(prompt, context_file)
        
        # Store response if requested
        if store_response:
            response_file = CONTEXT_DIR / f"response_{datetime.now().strftime('%Y%m%d_%H%M%S')}.txt"
            response_file.write_text(response)
        
        return [ToolResult(
            toolCallId="query_gemini",
            content=[TextContent(text=response)]
        )]
    
    elif name == "analyze_codebase":
        path = Path(arguments["path"])
        query = arguments["query"]
        patterns = arguments.get("include_patterns", ["*.go", "*.py", "*.md"])
        
        # Collect codebase content
        content_parts = []
        for pattern in patterns:
            for file in path.rglob(pattern):
                if file.is_file():
                    try:
                        content = file.read_text()
                        content_parts.append(f"\n=== File: {file} ===\n{content}")
                    except:
                        continue
        
        # Save to context file
        context_file = CONTEXT_DIR / "codebase_analysis.context"
        context_file.write_text("\n".join(content_parts))
        
        # Prepare analysis prompt
        analysis_prompt = f"""
Analyze the provided codebase with the following query:
{query}

Please provide:
1. Direct answer to the query
2. Supporting evidence from the code
3. Potential concerns or improvements
4. Architectural insights
5. Cybernetic ecology observations (feedback loops, system relationships, etc.)
"""
        
        response = await call_gemini(analysis_prompt, context_file)
        
        return [ToolResult(
            toolCallId="analyze_codebase",
            content=[TextContent(text=response)]
        )]
    
    elif name == "store_context":
        key = arguments["key"]
        content = arguments["content"]
        metadata = arguments.get("metadata", {})
        
        # Store content
        context_file = CONTEXT_DIR / f"{key}.context"
        context_file.write_text(content)
        
        # Store metadata
        if metadata:
            meta_file = CONTEXT_DIR / f"{key}.meta.json"
            meta_file.write_text(json.dumps({
                "stored_at": datetime.now().isoformat(),
                "metadata": metadata
            }, indent=2))
        
        return [ToolResult(
            toolCallId="store_context",
            content=[TextContent(text=f"Context stored with key: {key}")]
        )]
    
    elif name == "gemini_remember":
        topic = arguments["topic"]
        information = arguments["information"]
        
        # Create memory prompt
        memory_prompt = f"""
Remember this information for future reference:

Topic: {topic}
Information: {information}

Please acknowledge and summarize what you're remembering.
"""
        
        # Store in persistent memory
        memory_file = CONTEXT_DIR / "persistent_memory.context"
        existing = memory_file.read_text() if memory_file.exists() else ""
        memory_file.write_text(existing + f"\n\n[{datetime.now().isoformat()}] {topic}:\n{information}")
        
        response = await call_gemini(memory_prompt, memory_file)
        
        return [ToolResult(
            toolCallId="gemini_remember",
            content=[TextContent(text=response)]
        )]
    
    elif name == "gemini_recall":
        topic = arguments["topic"]
        question = arguments.get("specific_question", f"What do you remember about {topic}?")
        
        # Load persistent memory
        memory_file = CONTEXT_DIR / "persistent_memory.context"
        
        recall_prompt = f"""
Recall information about: {topic}
Specific question: {question}

Search your memory and provide relevant information.
"""
        
        response = await call_gemini(recall_prompt, memory_file)
        
        return [ToolResult(
            toolCallId="gemini_recall",
            content=[TextContent(text=response)]
        )]
    
    else:
        return [ToolResult(
            toolCallId=name,
            content=[TextContent(text=f"Unknown tool: {name}")]
        )]

async def main():
    """Run the MCP server"""
    async with stdio_server() as (read_stream, write_stream):
        await app.run(read_stream, write_stream)

if __name__ == "__main__":
    asyncio.run(main())