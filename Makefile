.PHONY: build install clean test release build-all version help

# Variables
BINARY_NAME = ramorie
BUILD_DIR = ./bin
VERSION = $(shell git describe --tags --dirty --always 2>/dev/null || echo "dev")
LDFLAGS = -ldflags "-X main.Version=$(VERSION)"

# Build the CLI binary for current platform
build:
	@echo "Building $(BINARY_NAME) v$(VERSION) for current platform..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/jbraincli

# Build for all platforms
build-all: clean
	@echo "Building $(BINARY_NAME) v$(VERSION) for all platforms..."
	@mkdir -p $(BUILD_DIR)

	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/jbraincli
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/jbraincli

	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/jbraincli
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/jbraincli

	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/jbraincli
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe ./cmd/jbraincli

	@echo "✅ All binaries built successfully in $(BUILD_DIR)/"

# Install the CLI globally
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "✅ $(BINARY_NAME) installed successfully!"

# Local development install (without sudo)
dev-install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/ (may require permissions)..."
	cp $(BINARY_NAME) /usr/local/bin/
	@echo "✅ $(BINARY_NAME) installed successfully!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	@echo "✅ Clean completed!"

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Lint code
lint:
	@echo "Running golangci-lint..."
	golangci-lint run

# Show current version
version:
	@echo "$(VERSION)"

# Create and push a new tag for release
tag:
	@read -p "Enter version (e.g., v1.0.0): " version; \
	git tag $$version && \
	git push origin $$version && \
	echo "✅ Tag $$version created and pushed!"

# Create a new release (requires tag)
release:
	@echo "To create a release:"
	@echo "1. make tag                    # Create and push a new tag"
	@echo "2. GitHub Actions will automatically build and create the release"
	@echo ""
	@echo "Or manually:"
	@echo "1. git tag v1.0.0"
	@echo "2. git push origin v1.0.0"

# Setup development environment
setup-dev:
	@echo "Setting up development environment..."
	go mod tidy
	go mod download
	@echo "✅ Development environment ready!"

# Show help
help:
	@echo "🧠 Ramorie CLI Build Commands"
	@echo "==================================="
	@echo ""
	@echo "Building:"
	@echo "  build        - Build binary for current platform"
	@echo "  build-all    - Build binaries for all platforms"
	@echo "  install      - Install CLI globally (requires sudo)"
	@echo "  dev-install  - Install without sudo"
	@echo ""
	@echo "Development:"
	@echo "  setup-dev    - Setup development environment"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  fmt          - Format Go code"
	@echo "  lint         - Run linter"
	@echo ""
	@echo "Release:"
	@echo "  tag          - Create and push a new version tag"
	@echo "  release      - Show release instructions"
	@echo "  version      - Show current version"
	@echo ""
	@echo "Maintenance:"
	@echo "  clean        - Remove build artifacts"
	@echo "  help         - Show this help message"