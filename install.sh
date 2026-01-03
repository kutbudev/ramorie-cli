#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    case $OS in
        linux)
            PLATFORM="linux"
            ;;
        darwin)
            PLATFORM="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac

    print_status "Detected platform: ${PLATFORM}-${ARCH}"
}

# Get latest release version
get_latest_version() {
    print_status "Fetching latest release information..."
    LATEST_VERSION=$(curl -s "https://api.github.com/repos/kutbudev/ramorie-cli/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$LATEST_VERSION" ]; then
        print_error "Failed to get latest version"
        exit 1
    fi

    print_status "Latest version: $LATEST_VERSION"
}

# Download and install
install_ramorie() {
    BINARY_NAME="ramorie-${PLATFORM}-${ARCH}"
    DOWNLOAD_URL="https://github.com/kutbudev/ramorie-cli/releases/latest/download/${BINARY_NAME}"

    print_status "Downloading ramorie from: $DOWNLOAD_URL"

    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    # Download binary
    if ! curl -L -o ramorie "$DOWNLOAD_URL"; then
        print_error "Failed to download ramorie"
        exit 1
    fi

    # Make executable
    chmod +x ramorie

    # Install to /usr/local/bin
    INSTALL_DIR="/usr/local/bin"

    if [ -w "$INSTALL_DIR" ]; then
        mv ramorie "$INSTALL_DIR/"
        print_status "Installed ramorie to $INSTALL_DIR"
    else
        print_status "Installing ramorie to $INSTALL_DIR (requires sudo)"
        sudo mv ramorie "$INSTALL_DIR/"
    fi

    # Clean up
    cd - > /dev/null
    rm -rf "$TMP_DIR"
}

# Verify installation
verify_installation() {
    if command -v ramorie >/dev/null 2>&1; then
        print_status "✅ ramorie installed successfully!"
        print_status "Version: $(ramorie --version 2>/dev/null || echo 'unknown')"
        echo
        echo "🚀 Get started with:"
        echo "   ramorie setup login"
        echo
        echo "📚 For help:"
        echo "   ramorie --help"
    else
        print_error "Installation failed. ramorie not found in PATH."
        exit 1
    fi
}

# Main execution
main() {
    echo "🧠 Ramorie CLI Installer"
    echo "======================="
    echo

    detect_platform
    get_latest_version
    install_ramorie
    verify_installation
}

# Check if running with bash
if [ -z "$BASH_VERSION" ]; then
    print_error "This script requires bash"
    exit 1
fi

# Run main function
main "$@"