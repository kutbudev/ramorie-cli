.PHONY: build install clean test release

# Build the CLI binary
build:
	go build -o jbraincli cmd/jbraincli/main.go

# Install the CLI globally
install: build
	sudo cp jbraincli /usr/local/bin/

# Clean build artifacts
clean:
	rm -f jbraincli

# Run tests
test:
	go test ./...

# Create a new release (requires tag)
release:
	@echo "To create a release:"
	@echo "1. git tag v1.0.0"
	@echo "2. git push origin v1.0.0"
	@echo "3. GitHub Actions will handle the rest!"

# Local development install
dev-install: build
	cp jbraincli /usr/local/bin/

# Show help
help:
	@echo "Available commands:"
	@echo "  build        - Build the CLI binary"
	@echo "  install      - Install CLI globally (requires sudo)"
	@echo "  dev-install  - Install without sudo (may require permissions)"
	@echo "  clean        - Remove build artifacts"
	@echo "  test         - Run tests"
	@echo "  release      - Show release instructions"