#!/bin/bash

# Strigoi Security Platform - Installation Script
# Copyright © 2025 Macawi AI
# 
# This script installs Strigoi to a user-specified directory
# Default: ~/.strigoi

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_INSTALL_DIR="$HOME/strigoi"
INSTALL_DIR="${STRIGOI_HOME:-$DEFAULT_INSTALL_DIR}"
BINARY_NAME="strigoi"

# Functions
print_banner() {
    echo -e "${GREEN}"
    echo "███████╗████████╗██████╗ ██╗ ██████╗  ██████╗ ██╗"
    echo "██╔════╝╚══██╔══╝██╔══██╗██║██╔════╝ ██╔═══██╗██║"
    echo "███████╗   ██║   ██████╔╝██║██║  ███╗██║   ██║██║"
    echo "╚════██║   ██║   ██╔══██╗██║██║   ██║██║   ██║██║"
    echo "███████║   ██║   ██║  ██║██║╚██████╔╝╚██████╔╝██║"
    echo "╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝ ╚═════╝  ╚═════╝ ╚═╝"
    echo -e "${NC}"
    echo "Security Validation Platform - Installer"
    echo "========================================="
    echo ""
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${YELLOW}→${NC} $1"
}

check_dependencies() {
    print_info "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21+ first."
        echo "Visit: https://golang.org/doc/install"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_success "Go $GO_VERSION found"
}

create_directories() {
    print_info "Creating installation directories..."
    
    mkdir -p "$INSTALL_DIR"/{bin,config,logs,data,plugins}
    print_success "Created directory structure at $INSTALL_DIR"
}

build_binary() {
    print_info "Building Strigoi from source..."
    
    cd "$SCRIPT_DIR"
    
    # Build with version info
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" \
        -o "$BINARY_NAME" ./cmd/strigoi
    
    if [ ! -f "$BINARY_NAME" ]; then
        print_error "Build failed"
        exit 1
    fi
    
    print_success "Build successful"
}

install_binary() {
    print_info "Installing binary..."
    
    cp "$SCRIPT_DIR/$BINARY_NAME" "$INSTALL_DIR/bin/"
    chmod +x "$INSTALL_DIR/bin/$BINARY_NAME"
    
    print_success "Binary installed to $INSTALL_DIR/bin/$BINARY_NAME"
}

install_config() {
    print_info "Setting up configuration..."
    
    CONFIG_FILE="$INSTALL_DIR/config/strigoi.yaml"
    
    if [ -f "$CONFIG_FILE" ]; then
        print_info "Config file already exists, skipping..."
    else
        if [ -f "$SCRIPT_DIR/configs/strigoi.yaml.example" ]; then
            cp "$SCRIPT_DIR/configs/strigoi.yaml.example" "$CONFIG_FILE"
            print_success "Created config from template"
        else
            # Create minimal config
            cat > "$CONFIG_FILE" << EOF
# Strigoi Configuration File
version: 1.0

# Logging configuration
logging:
  level: info
  file: $INSTALL_DIR/logs/strigoi.log

# Data directory
data_dir: $INSTALL_DIR/data

# Plugin directory
plugin_dir: $INSTALL_DIR/plugins

# Security settings
security:
  validate_tls: true
  timeout: 30s
EOF
            print_success "Created default configuration"
        fi
    fi
}

setup_uninstaller() {
    print_info "Creating uninstaller..."
    
    cat > "$INSTALL_DIR/uninstall.sh" << EOF
#!/bin/bash
# Strigoi Uninstaller

echo "Uninstalling Strigoi..."

# Remove from PATH
if grep -q "$INSTALL_DIR/bin" ~/.bashrc; then
    sed -i "\\|$INSTALL_DIR/bin|d" ~/.bashrc
    echo "✓ Removed from PATH"
fi

if grep -q "$INSTALL_DIR/bin" ~/.zshrc 2>/dev/null; then
    sed -i "\\|$INSTALL_DIR/bin|d" ~/.zshrc
    echo "✓ Removed from PATH (zsh)"
fi

# Backup config before removal
if [ -f "$INSTALL_DIR/config/strigoi.yaml" ]; then
    cp "$INSTALL_DIR/config/strigoi.yaml" ~/strigoi-config-backup-\$(date +%Y%m%d).yaml
    echo "✓ Config backed up to ~/strigoi-config-backup-\$(date +%Y%m%d).yaml"
fi

# Remove installation directory
read -p "Remove all Strigoi files from $INSTALL_DIR? (y/N) " -n 1 -r
echo
if [[ \$REPLY =~ ^[Yy]$ ]]; then
    rm -rf "$INSTALL_DIR"
    echo "✓ Removed $INSTALL_DIR"
fi

echo "Uninstall complete!"
EOF
    
    chmod +x "$INSTALL_DIR/uninstall.sh"
    print_success "Uninstaller created"
}

update_path() {
    print_info "Updating PATH..."
    
    SHELL_RC=""
    if [ -n "$BASH_VERSION" ]; then
        SHELL_RC="$HOME/.bashrc"
    elif [ -n "$ZSH_VERSION" ]; then
        SHELL_RC="$HOME/.zshrc"
    fi
    
    if [ -n "$SHELL_RC" ]; then
        if ! grep -q "$INSTALL_DIR/bin" "$SHELL_RC"; then
            echo "" >> "$SHELL_RC"
            echo "# Strigoi Security Platform" >> "$SHELL_RC"
            echo "export PATH=\"\$PATH:$INSTALL_DIR/bin\"" >> "$SHELL_RC"
            echo "export STRIGOI_HOME=\"$INSTALL_DIR\"" >> "$SHELL_RC"
            print_success "Added to PATH in $SHELL_RC"
        else
            print_info "PATH already configured"
        fi
    fi
}

print_completion() {
    echo ""
    echo "═══════════════════════════════════════════════════"
    print_success "Installation complete!"
    echo ""
    echo "Installed to: $INSTALL_DIR"
    echo ""
    echo "Next steps:"
    echo "  1. Reload your shell configuration:"
    echo "     source ~/.bashrc  (or ~/.zshrc)"
    echo ""
    echo "  2. Verify installation:"
    echo "     strigoi --version"
    echo ""
    echo "  3. Start using Strigoi:"
    echo "     strigoi --help"
    echo ""
    echo "Configuration: $INSTALL_DIR/config/strigoi.yaml"
    echo "Logs: $INSTALL_DIR/logs/"
    echo "Uninstall: $INSTALL_DIR/uninstall.sh"
    echo ""
    echo "Documentation: https://github.com/macawi-ai/strigoi"
    echo "═══════════════════════════════════════════════════"
}

# Main installation flow
main() {
    print_banner

    local auto_confirm=false
    local custom_prefix=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --prefix)
                custom_prefix="$2"
                shift 2
                ;;
            --yes|-y)
                auto_confirm=true
                shift
                ;;
            --help|-h)
                echo "Strigoi Installation Script"
                echo ""
                echo "Usage: $0 [options]"
                echo ""
                echo "Options:"
                echo "  --prefix DIR    Install to custom directory (default: ~/strigoi)"
                echo "  --yes, -y      Skip confirmation prompt"
                echo "  --help, -h     Show this help message"
                echo ""
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                print_info "Use --help for usage information"
                exit 1
                ;;
        esac
    done

    # Set installation directory
    if [ -n "$custom_prefix" ]; then
        INSTALL_DIR="$custom_prefix"
        print_info "Custom installation directory: $INSTALL_DIR"
    else
        print_info "Installing to: $INSTALL_DIR"
        print_info "(Set STRIGOI_HOME or use --prefix to customize)"
    fi

    # Confirmation prompt (unless --yes flag is used)
    if [ "$auto_confirm" = false ]; then
        echo ""
        read -p "Continue with installation? (Y/n) " -n 1 -r
        echo ""

        if [[ ! $REPLY =~ ^[Yy]$ ]] && [ -n "$REPLY" ]; then
            print_error "Installation cancelled"
            exit 1
        fi
    else
        print_info "Auto-confirming installation..."
    fi
    
    check_dependencies
    create_directories
    build_binary
    install_binary
    install_config
    setup_uninstaller
    update_path
    
    # Clean up build artifact
    rm -f "$SCRIPT_DIR/$BINARY_NAME"
    
    print_completion
}

# Run main function
main "$@"