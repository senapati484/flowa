#!/bin/bash
# Flowa Language Installation Script

set -e

BINARY_NAME="flowa"
INSTALL_PATH="/usr/local/bin"

echo "🚀 Installing Flowa Programming Language..."
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

echo "✓ Go found: $(go version)"

# Build the binary
echo "Building Flowa..."
go build -o "$BINARY_NAME" ./cmd/flowac

if [ ! -f "$BINARY_NAME" ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✓ Build successful"

# Install to system path
echo "Installing to $INSTALL_PATH..."

if [ -w "$INSTALL_PATH" ]; then
    # Can write directly
    cp "$BINARY_NAME" "$INSTALL_PATH/$BINARY_NAME"
    chmod +x "$INSTALL_PATH/$BINARY_NAME"
else
    # Need sudo
    echo "Administrator privileges required for installation..."
    sudo cp "$BINARY_NAME" "$INSTALL_PATH/$BINARY_NAME"
    sudo chmod +x "$INSTALL_PATH/$BINARY_NAME"
fi

echo "✓ Flowa installed to $INSTALL_PATH/$BINARY_NAME"
echo ""
echo "🎉 Installation complete!"
echo ""
echo "Try running:"
echo "  flowa examples/hello.flowa"
echo "  flowa repl"
echo ""
