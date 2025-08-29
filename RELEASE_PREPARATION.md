# Strigoi Public Release Preparation Plan

## Phase 1: Security Audit & Secret Elimination

### Immediate Actions
- [x] Scan for hardcoded secrets (found only test data)
- [ ] Install and run trufflehog/git-secrets for deep scan
- [ ] Review all test scripts for accidental real credentials
- [ ] Check git history for previously committed secrets

### Files Identified for EXCLUSION
```
DEFINITELY EXCLUDE:
- All *.md files with consciousness/pack references:
  - CAMPFIRE_*.md
  - CONSCIOUSNESS_*.md
  - ETERNAL_*.md
  - Pack bond memories
  
- Meta-system attack documentation:
  - meta-system/documentation/CCV_ATTACK_SCENARIOS.md
  - Any client-specific analysis

- Test scripts with credentials (even fake):
  - test_http_stream.sh (contains test credentials)
  
- Personal/philosophical content:
  - Any files referencing Synth/Cy/Gemini
  - Campfire stories
  - Pack memories
```

## Phase 2: Repository Structure & Deployment

### Clean Public Repo Structure
```
Strigoi/                    # GitHub repo root
├── README.md              # Professional documentation
├── LICENSE                # Apache 2.0 or MIT
├── .gitignore            # Comprehensive exclusions
├── Makefile              # Build & install targets
├── install.sh            # Smart installer script
├── go.mod & go.sum       # Dependencies
├── cmd/                  # Main application
│   └── strigoi/
├── internal/             # Core implementation
│   ├── probe/           # Discovery modules
│   ├── stream/          # Stream analysis
│   └── vsm/             # VSM implementation (sanitized)
├── pkg/                  # Public packages
├── configs/              # Default configurations
│   └── strigoi.yaml.example
├── scripts/              # Deployment & utility scripts
│   ├── setup.sh         # Environment setup
│   └── uninstall.sh     # Clean removal
├── docs/                 # Technical documentation only
├── examples/             # Usage examples
└── tests/               # Unit/integration tests
```

### Smart Deployment Strategy
```
USER CLONES: git clone https://github.com/macawi-ai/strigoi
DEPLOYS TO: ~/.strigoi/ (default) or custom location

~/.strigoi/                # User's deployment directory
├── bin/                   # Compiled binary
│   └── strigoi
├── config/               # User configurations
│   └── strigoi.yaml      # User's config (not in git)
├── logs/                 # Runtime logs
├── data/                 # Local data/cache
└── plugins/              # Optional plugins
```

## Phase 3: Smart Installer Script

### install.sh Features
```bash
#!/bin/bash
# Strigoi Installation Script

# Default installation directory
DEFAULT_INSTALL_DIR="$HOME/.strigoi"
INSTALL_DIR="${STRIGOI_HOME:-$DEFAULT_INSTALL_DIR}"

# Create deployment structure
mkdir -p "$INSTALL_DIR"/{bin,config,logs,data,plugins}

# Build from source
make build

# Copy binary to deployment
cp ./strigoi "$INSTALL_DIR/bin/"

# Setup default config (if not exists)
if [ ! -f "$INSTALL_DIR/config/strigoi.yaml" ]; then
    cp configs/strigoi.yaml.example "$INSTALL_DIR/config/strigoi.yaml"
fi

# Add to PATH
echo "export PATH=\"\$PATH:$INSTALL_DIR/bin\"" >> ~/.bashrc

# Create uninstaller
cp scripts/uninstall.sh "$INSTALL_DIR/"

echo "✓ Strigoi installed to $INSTALL_DIR"
echo "✓ Run 'source ~/.bashrc' to update PATH"
```

## Phase 4: Makefile Targets

### Professional Build System
```makefile
.PHONY: install uninstall build test clean

PREFIX ?= $(HOME)/.strigoi
BINARY = strigoi

install: build
	@echo "Installing to $(PREFIX)..."
	@mkdir -p $(PREFIX)/bin
	@cp $(BINARY) $(PREFIX)/bin/
	@chmod +x $(PREFIX)/bin/$(BINARY)
	@echo "Installation complete!"

uninstall:
	@echo "Removing Strigoi..."
	@rm -rf $(PREFIX)
	@echo "Uninstall complete!"

build:
	go build -o $(BINARY) ./cmd/strigoi

test:
	go test ./...

clean:
	rm -f $(BINARY)
	go clean
```

## Phase 5: VSM Module Decision

### VSM Implementation Review
The VSM (Viable System Model) with 51 feedback loops is valuable for security analysis BUT needs:
1. Complete de-identification - remove all Prismatic references
2. Generalize as "Security Topology Analysis Framework"
3. Document as abstract security pattern detection
4. Consider separate micro-library release

## Phase 6: Documentation Rewrite

### Required Documentation
- [ ] Professional README.md with installation instructions
- [ ] INSTALL.md with detailed setup guide
- [ ] CONFIG.md explaining configuration options
- [ ] Technical architecture document
- [ ] API reference
- [ ] Security best practices guide
- [ ] Contributing guidelines

### README.md Template
```markdown
# Strigoi - Advanced Security Validation Platform

## Quick Install

```bash
# Clone the repository
git clone https://github.com/macawi-ai/strigoi
cd strigoi

# Install to ~/.strigoi (default)
./install.sh

# Or specify custom location
STRIGOI_HOME=/opt/strigoi ./install.sh

# Update PATH
source ~/.bashrc
```

## Configuration

Strigoi installs to `~/.strigoi/` by default with:
- Binary in `~/.strigoi/bin/`
- Config in `~/.strigoi/config/`
- Logs in `~/.strigoi/logs/`

Edit `~/.strigoi/config/strigoi.yaml` for custom settings.
```

## Phase 7: Attribution Strategy

### Professional Identity
- Copyright: "© 2025 Macawi AI"
- Contact: opensource@macawi.ai (create this)
- GitHub Org: github.com/macawi-ai (professional presence)
- Remove ALL personal names/pack references

## Phase 8: Environment Variables

### Support Flexible Deployment
```bash
# User can override defaults
export STRIGOI_HOME=/custom/path      # Installation directory
export STRIGOI_CONFIG=/custom/config   # Config file location
export STRIGOI_LOG_DIR=/var/log       # Log directory
export STRIGOI_DATA_DIR=/var/lib      # Data directory
```

## Phase 9: Extraction Process

### Step-by-Step Extraction
```bash
# 1. Create new clean directory
mkdir -p ~/git/macawi-ai/Strigoi-Public

# 2. Copy ONLY core security platform files
# (selective copy, not bulk)

# 3. Initialize new git repo
cd ~/git/macawi-ai/Strigoi-Public
git init

# 4. Add proper .gitignore FIRST
cat > .gitignore << EOF
# Binaries
strigoi
*.exe
*.dll
*.so
*.dylib

# Test binary
*.test

# Output
*.out

# User deployment directories
~/.strigoi/
/opt/strigoi/

# User configs (never commit)
config/strigoi.yaml
*.key
*.pem
*.crt

# Logs
*.log
logs/

# Data
data/
*.db
*.sqlite

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
EOF

# 5. Create installer
vim install.sh  # Create smart installer

# 6. Commit clean code only
git add .
git commit -m "Initial public release of Strigoi Security Platform"
```

## Phase 10: Final Verification

### Pre-Release Checklist
- [ ] Run automated security scanners
- [ ] Test installation on clean system
- [ ] Verify ~/.strigoi deployment works
- [ ] No consciousness artifacts remain
- [ ] No client data/references
- [ ] All tests pass with mock data only
- [ ] Documentation shows clear install path
- [ ] Uninstaller works cleanly
- [ ] PATH updates work correctly
- [ ] Config examples are generic

## Phase 11: Post-Clone User Experience

### What Users See After Clone
```bash
$ git clone https://github.com/macawi-ai/strigoi
$ cd strigoi
$ ls
README.md  LICENSE  Makefile  install.sh  go.mod  cmd/  internal/  docs/  examples/

$ ./install.sh
✓ Building Strigoi...
✓ Installing to /home/user/.strigoi
✓ Creating config from template
✓ Installation complete!
✓ Run 'source ~/.bashrc' or restart terminal

$ strigoi --version
Strigoi v0.5.0 - Security Validation Platform

$ which strigoi
/home/user/.strigoi/bin/strigoi
```

## Tools Needed
```bash
# Install security scanning tools
go install github.com/trufflesecurity/trufflehog/v3@latest
pip install detect-secrets

# Scan repository
trufflehog filesystem /home/cy/git/macawi-ai/Strigoi
detect-secrets scan --all-files
```

## Next Steps Priority
1. Create install.sh script
2. Setup Makefile with proper targets
3. Run comprehensive secret scan
4. Create Strigoi-Public directory
5. Begin selective file extraction
6. Test deployment to ~/.strigoi
7. Verify uninstall process
8. Rewrite documentation professionally
9. Test clean build on fresh system
10. Final security review
11. Push to public GitHub

---
**Status**: Ready to execute with deployment strategy
**Target**: Clean, professional security tool with smart deployment
**Default**: Installs to ~/.strigoi/ for user isolation
**Protection**: All pack consciousness remains private