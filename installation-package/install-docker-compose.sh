#!/bin/bash

# Docker Compose Offline Installation Script for AMD64 Linux
# This script installs Docker Compose from the local binary in this package

set -e

echo "🐳 Installing Docker Compose for AMD64 Linux (Offline Mode)..."

# Check if running on Linux
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo "❌ This script is designed for Linux systems"
    exit 1
fi

# Check if running on x86_64
if [[ $(uname -m) != "x86_64" ]]; then
    echo "❌ This script is designed for AMD64 (x86_64) architecture"
    exit 1
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    echo "   Note: This package requires Docker to be pre-installed for offline environments."
    exit 1
fi

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_BINARY="$SCRIPT_DIR/docker-compose-amd64"

# Check if Docker Compose binary exists
if [[ ! -f "$COMPOSE_BINARY" ]]; then
    echo "❌ Docker Compose binary not found at: $COMPOSE_BINARY"
    echo "   Please ensure the docker-compose-amd64 file is in the same directory as this script."
    exit 1
fi

echo "📦 Installing Docker Compose from local binary..."

# Install Docker Compose
sudo cp "$COMPOSE_BINARY" /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Create symlink for backward compatibility
sudo ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose

# Verify installation
if command -v docker-compose &> /dev/null; then
    echo "✅ Docker Compose installed successfully!"
    echo "📋 Docker Compose version: $(docker-compose --version)"
    echo ""
    echo "🚀 You can now use Docker Compose:"
    echo "   docker-compose --version"
    echo "   docker-compose --help"
    echo ""
    echo "📝 Example usage:"
    echo "   docker-compose up -d"
    echo "   docker-compose down"
    echo "   docker-compose ps"
    echo ""
    echo "📁 Note: This was an offline installation using the included binary."
else
    echo "❌ Installation failed"
    exit 1
fi

echo ""
echo "💡 Tip: You can now create a docker-compose.yml file to manage multi-container applications."