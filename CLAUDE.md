# Ramorie CLI - Project Instructions

## Project Overview
Go CLI tool + MCP server for AI-powered task/memory management. Communicates with Ramorie backend API.

## Critical Rules

- **Language**: Go 1.24. NOT a Node.js project.
- **CLI Framework**: urfave/cli v2 (NOT cobra)
- **MCP SDK**: `modelcontextprotocol/go-sdk` v1.2.0 (official SDK)
- **Module Path**: `github.com/kutbudev/ramorie-cli`
- **Config Location**: `~/.ramorie/config.json`
- **API Base**: `https://api.ramorie.com/v1` with Bearer token auth
- **Encryption**: AES-256-GCM + PBKDF2-SHA256 (zero-knowledge, client-side only)
- **MCP Transport**: stdio (stdin/stdout)
- **Numbers in MCP**: Always `float64` (MCP protocol sends floats)

## Development

```bash
make build          # Build binary (current platform)
make build-all      # All platforms
make install        # Install to /usr/local/bin
make dev-install    # Install without sudo
make test           # Run tests
make test-coverage  # Coverage report
make fmt            # Format code
make lint           # golangci-lint
make setup-dev      # go mod tidy + download
make clean          # Remove artifacts
```

## Architecture

```
cmd/ramorie/main.go          → Entry point, command registration
internal/cli/commands/        → CLI commands (urfave/cli)
internal/mcp/                 → MCP server (tools, session, directives)
internal/api/client.go        → HTTP client for backend API
internal/config/              → Config management
internal/crypto/              → Zero-knowledge encryption
internal/models/              → Data models
internal/errors/              → Error parsing
```

## Available Skills

- `ramorie-cli-dev`: Full development guide, tech stack, architecture
- `ramorie-cli-command`: CLI command creation patterns (urfave/cli)
- `ramorie-cli-mcp`: MCP tool creation patterns (input, handler, response)

## Key Patterns

### Adding a New CLI Command
1. Create command file: `internal/cli/commands/feature.go`
2. Implement `NewFeatureCommand()` returning `*cli.Command`
3. Add subcommands: `featureListCmd()`, `featureCreateCmd()`, etc.
4. Register in `cmd/ramorie/main.go` Commands slice
5. Handle encryption if entity supports it

### Adding a New MCP Tool
1. Define input struct: `FeatureInput` with `json` tags
2. Implement handler: `handleFeature(ctx, req, input)`
3. Register in `registerTools()` with tier prefix in description
4. Add `checkSessionInit()` at handler start
5. Return via `formatMCPResponse()` wrapper
6. Add API method in `internal/api/client.go` if needed

### MCP Tool Description Format
```
"🟡 COMMON | Action description. REQUIRED: param1. Optional: param2, param3."
```

## MCP Server Entry
```bash
ramorie mcp serve    # Start MCP stdio server
```

## Distribution
- **Homebrew**: `brew install kutbudev/tap/ramorie`
- **GoReleaser**: Automated GitHub releases
- **Platforms**: Linux/macOS/Windows (amd64/arm64)
