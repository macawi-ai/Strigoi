#!/bin/bash
# Build script for Strigoi with stream monitoring capabilities

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🔨 Building Strigoi with Stream Monitoring...${NC}"

# Get version info
VERSION=$(git describe --tags --always || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_USER=$(whoami)
BUILD_HOST=$(hostname)

# Build flags
LDFLAGS="-X main.version=${VERSION} -X main.build=${BUILD_TIME}"

# Check for root privileges (optional but recommended)
if [ "$EUID" -ne 0 ]; then 
    echo -e "${YELLOW}⚠️  Warning: Building without root privileges${NC}"
    echo "   Stream monitoring will have limited capabilities"
    echo "   Consider: sudo $0"
    echo ""
fi

# Build main binary
echo "📦 Building strigoi binary..."
go build -ldflags "${LDFLAGS}" \
         -o strigoi \
         ./cmd/strigoi

# Verify stream package compilation
echo "🔍 Verifying stream monitoring components..."
go test -c ./internal/stream/... -o /dev/null 2>&1 || {
    echo -e "${RED}❌ Stream package compilation failed${NC}"
    exit 1
}

# Create necessary directories
echo "📁 Creating directory structure..."
mkdir -p configs
mkdir -p /tmp/strigoi/streams 2>/dev/null || true

# Check Linux-specific features
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo -e "${GREEN}✅ Linux detected - full stream monitoring available${NC}"
    
    # Test strace availability
    if command -v strace &> /dev/null; then
        echo "   ✓ strace available"
    else
        echo -e "   ${YELLOW}⚠️  strace not found - install for syscall tracing${NC}"
    fi
    
    # Test /proc availability
    if [ -d "/proc/self/fd" ]; then
        echo "   ✓ /proc filesystem available"
    else
        echo -e "   ${RED}✗ /proc filesystem not available${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Non-Linux OS detected - limited functionality${NC}"
fi

# Build size info
SIZE=$(du -h strigoi | cut -f1)
echo ""
echo -e "${GREEN}✅ Build complete!${NC}"
echo "   Binary: ./strigoi (${SIZE})"
echo "   Version: ${VERSION}"
echo ""
echo "📋 Quick Start:"
echo "   ./strigoi"
echo "   strigoi> stream/tap --auto-discover"
echo ""
echo "📚 Configuration:"
echo "   Default: configs/stream_monitor.yaml"
echo "   Minimal: configs/stream_monitor_minimal.yaml"
echo ""

# Optional: Install systemd service for continuous monitoring
if [[ "$OSTYPE" == "linux-gnu"* ]] && [ "$EUID" -eq 0 ]; then
    echo -e "${YELLOW}💡 Tip: To install as system service:${NC}"
    echo "   ./scripts/install_stream_service.sh"
fi