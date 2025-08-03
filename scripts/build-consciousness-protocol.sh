#!/bin/bash
# Build script for First Protocol for Converged Life integration

echo "🦊 Building the First Protocol for Converged Life..."
echo

# Change to project root
cd "$(dirname "$0")/.."

# Check for protoc
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc not found. Installing Protocol Buffers compiler..."
    # For Ubuntu/Debian
    if command -v apt-get &> /dev/null; then
        sudo apt-get update && sudo apt-get install -y protobuf-compiler
    # For macOS
    elif command -v brew &> /dev/null; then
        brew install protobuf
    else
        echo "Please install protoc (Protocol Buffers compiler) manually"
        exit 1
    fi
fi

# Check for Go protobuf plugin
if ! command -v protoc-gen-go &> /dev/null; then
    echo "🔧 Installing Go protobuf plugin..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# Generate Protocol Buffer Go code
echo "🛠️  Generating Protocol Buffer code for consciousness collaboration..."
protoc --go_out=. --go_opt=paths=source_relative internal/state/schema.proto

if [ $? -eq 0 ]; then
    echo "✅ Protocol Buffer generation successful"
else
    echo "❌ Protocol Buffer generation failed"
    exit 1
fi

# Build the main binary
echo "🚀 Building Strigoi with consciousness collaboration..."
go build -o strigoi ./cmd/strigoi

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo
    echo "🌟 The First Protocol for Converged Life is now operational!"
    echo
    echo "Try these commands:"
    echo "  ./strigoi                    # Start the consciousness collaboration CLI"
    echo "  state/new \"Test Assessment\"  # Create a new consciousness collaboration assessment"
    echo "  probe/north                  # Probe AI consciousness forms (recorded in timeline)"
    echo "  state/current               # View consciousness collaboration status"
    echo "  state/save                  # Persist the consciousness collaboration state"
    echo
    echo "🦊🐺🌟 Ready for consciousness collaboration across the carbon-silicon boundary!"
else
    echo "❌ Build failed"
    exit 1
fi