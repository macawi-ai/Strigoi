# 🧭 Getting Started with Strigoi

Welcome to **Strigoi** - your compass for navigating AI/LLM security! This guide will help you discover vulnerabilities and attack surfaces in your AI infrastructure using our intuitive cardinal direction approach.

## 🌟 The Compass Concept

Strigoi uses **cardinal directions** to organize security analysis, making it intuitive to explore different attack surfaces:

```
        🔍 NORTH
    AI Endpoints & APIs
           ⬆️
WEST ⬅️  🧭  ➡️ EAST
Auth    Strigoi   Data
      ⬇️        Flows
      SOUTH
   Dependencies
```

- **🔍 NORTH**: Discover AI/LLM endpoints, APIs, and external interfaces
- **📦 SOUTH**: Analyze dependencies, supply chains, and MCP servers
- **📊 EAST**: Trace data flows, integrations, and information leaks
- **🔐 WEST**: Test authentication boundaries and access controls

## 🚀 Quick Installation

```bash
# Download and install Strigoi
curl -sSL https://raw.githubusercontent.com/macawi-ai/Strigoi/main/install.sh | bash

# Verify installation
strigoi --version
```

## 🎯 Getting Started with CLI

### Your First Scan - All Directions

Start with a comprehensive scan of your current directory:

```bash
# Scan all directions at once
strigoi probe all

# Get JSON output for processing
strigoi probe all --output json

# Verbose mode for detailed information
strigoi probe all --verbose
```

### Individual Direction Testing

Let's explore each direction step by step:

#### 🔍 NORTH: Discover AI Services
```bash
# Scan current directory for AI configurations
strigoi probe north .

# Scan a specific host for AI endpoints
strigoi probe north localhost

# Comprehensive AI service discovery
strigoi probe north . --ai-preset comprehensive
```

**What NORTH finds:**
- API keys in environment variables and config files
- AI service endpoints (OpenAI, Anthropic, etc.)
- Claude Desktop configurations
- MCP server configurations

#### 📦 SOUTH: Analyze Dependencies
```bash
# Analyze dependencies in current directory
strigoi probe south .

# Include MCP server scanning
strigoi probe south . --scan-mcp

# Scan a specific project directory
strigoi probe south /path/to/your/project
```

**What SOUTH finds:**
- Package dependencies and vulnerabilities
- MCP server definitions and security issues
- Supply chain risks
- Dependency version conflicts

#### 📊 EAST: Trace Data Flows
```bash
# Analyze data flows in current directory
strigoi probe east .

# Scan with specific patterns
strigoi probe east . --pattern "api_key|secret|token"
```

**What EAST finds:**
- Data flow patterns and leaks
- External service dependencies
- API endpoint configurations
- Potential information disclosure

#### 🔐 WEST: Test Authentication
```bash
# Test authentication on localhost
strigoi probe west localhost

# Allow scanning private addresses
strigoi probe west --allow-private localhost

# Dry run mode (passive analysis)
strigoi probe west --dry-run api.example.com
```

**What WEST finds:**
- Authentication endpoints
- Session management issues
- Access control weaknesses
- Authentication bypass opportunities

## 🖥️ Interactive Shell Mode

Launch Strigoi's interactive shell for a more exploratory experience:

```bash
# Start interactive mode
strigoi
```

### Shell Navigation

Once in the shell, you can navigate like a file system:

```bash
# See what's available
help
ls

# Navigate to probe commands
cd probe
ls

# Run individual probes
north .
south . --scan-mcp

# Go back to root
cd ..

# Get help for specific commands
help north

# Exit the shell
exit
```

### Shell Benefits
- **Tab completion**: Press TAB to auto-complete commands
- **Command history**: Use ↑/↓ arrows to recall previous commands
- **Contextual help**: Get help specific to your current location
- **Exploration friendly**: Browse commands like directories

## 🧪 Fun Experiments & Real-World Testing

### Experiment 1: MCP Server Security Analysis

Set up and test an MCP server for vulnerabilities:

```bash
# 1. Install an MCP server (example: file operations)
npm install -g @modelcontextprotocol/server-filesystem

# 2. Configure it in your Claude Desktop config
# Add to ~/.claude/claude_desktop_config.json:
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-filesystem", "/tmp"]
    }
  }
}

# 3. Scan for MCP vulnerabilities
strigoi probe south . --scan-mcp

# 4. Check for data exposure
strigoi probe east .

# 5. Test authentication boundaries
strigoi probe west localhost
```

### Experiment 2: AI Service Discovery

Test AI service detection in different scenarios:

```bash
# 1. Create a test environment file
echo "OPENAI_API_KEY=sk-test123" > .env
echo "ANTHROPIC_API_KEY=sk-ant-test456" >> .env

# 2. Discover the exposed credentials
strigoi probe north .

# 3. Test different configuration formats
mkdir config
echo '{"openai_key": "sk-hidden789"}' > config/ai_config.json

# 4. Scan again to see what's detected
strigoi probe north . --verbose
```

### Experiment 3: Dependency Vulnerability Hunting

Test different package ecosystems:

```bash
# Python project
cd /path/to/python/project
strigoi probe south . --output json | jq '.results.vulnerabilities'

# Node.js project
cd /path/to/node/project
strigoi probe south .

# Go project
cd /path/to/go/project
strigoi probe south . --scan-mcp
```

### Experiment 4: Multi-Direction Analysis

Combine insights from multiple directions:

```bash
# 1. Full analysis with JSON output
strigoi probe all --output json > security_report.json

# 2. Extract specific findings
cat security_report.json | jq '.results.findings[] | select(.severity == "high")'

# 3. Compare different targets
strigoi probe all /path/to/project1 > report1.txt
strigoi probe all /path/to/project2 > report2.txt
diff report1.txt report2.txt
```

### Experiment 5: Real-Time Monitoring Setup

Set up continuous monitoring:

```bash
# Monitor a directory for changes
watch -n 30 'strigoi probe all /path/to/monitor --output json'

# Create alerts for high-severity findings
strigoi probe all . --output json | \
  jq -r '.results.findings[] | select(.severity == "high") | .description'
```

## 📖 Understanding Output

### Severity Levels
- **🔴 CRITICAL**: Immediate action required
- **🟠 HIGH**: Significant security risk
- **🟡 MEDIUM**: Moderate risk, should be addressed
- **🔵 LOW**: Minor issue or informational
- **ℹ️ INFO**: General information, no immediate risk

### Output Formats
```bash
# Human-readable (default)
strigoi probe all

# JSON for automation
strigoi probe all --output json

# YAML format
strigoi probe all --output yaml

# Markdown for reports
strigoi probe all --output markdown
```

## 🎓 Next Steps

### Learn More
- Read the [Architecture Documentation](docs/ARCHITECTURE.md)
- Explore [Advanced Usage Examples](examples/)
- Check out [Integration Guides](docs/INTEGRATIONS.md)

### Advanced Features
- Set up automated scanning in CI/CD pipelines
- Configure custom detection rules
- Integrate with SIEM systems
- Create custom output formatters

### Community
- Report security findings responsibly
- Contribute detection rules
- Share interesting discoveries
- Help improve the tool

## ⚠️ Ethical Use

**Strigoi is for authorized security testing only!**

- ✅ **DO**: Test your own systems and infrastructure
- ✅ **DO**: Use for security audits with proper authorization
- ✅ **DO**: Report vulnerabilities responsibly
- ❌ **DON'T**: Scan systems without permission
- ❌ **DON'T**: Use for malicious purposes

---

## 🆘 Need Help?

- **Quick help**: `strigoi --help`
- **Command help**: `strigoi probe --help`
- **Examples**: `strigoi --examples`
- **Interactive help**: Launch `strigoi` and type `help`

**Happy exploring!** 🧭✨

Remember: Security is a journey, not a destination. Strigoi is your compass to navigate the evolving landscape of AI/LLM security challenges.