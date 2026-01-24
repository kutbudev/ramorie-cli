---
name: ramorie-cli-dev
description: Use when working on any task in the Ramorie CLI project - provides architecture, conventions, tech stack, and development patterns for the Go CLI with MCP server
---

# Ramorie CLI Development Guide

## Overview

Ramorie CLI is a **Go 1.24** command-line tool and **MCP server** for AI-powered task/memory management. It communicates with the Ramorie backend API and provides both human-interactive commands and AI agent integration via Model Context Protocol.

## Tech Stack Quick Reference

| Layer | Technology |
|-------|-----------|
| Language | Go 1.24 |
| CLI Framework | urfave/cli v2 |
| MCP SDK | modelcontextprotocol/go-sdk v1.2.0 |
| HTTP Client | net/http (custom wrapper) |
| Config | spf13/viper + JSON config file |
| Crypto | AES-256-GCM + PBKDF2-SHA256 |
| Keyring | zalando/go-keyring |
| Interactive | AlecAivazis/survey/v2 |
| Output | text/tabwriter |
| UUID | google/uuid |
| Distribution | GoReleaser + Homebrew |

## Project Structure

```
ramorie-cli/
├── cmd/
│   └── ramorie/main.go          # Entry point, app & command registration
├── internal/
│   ├── api/client.go            # HTTP API client (all backend calls)
│   ├── cli/commands/            # CLI command implementations
│   │   ├── task.go              # Task commands
│   │   ├── memory.go            # Memory commands
│   │   ├── project.go           # Project commands
│   │   ├── decision.go          # Decision (ADR) commands
│   │   ├── organization.go      # Org commands
│   │   ├── context_pack.go      # Context pack commands
│   │   ├── focus.go             # Workspace focus commands
│   │   ├── plan.go              # AI planner commands
│   │   ├── subtask.go           # Subtask commands
│   │   ├── setup.go             # Setup/auth commands
│   │   └── ...                  # Other command files
│   ├── mcp/                     # MCP server implementation
│   │   ├── server.go            # MCP stdio server entry
│   │   ├── tools.go             # 26 MCP tool definitions + handlers
│   │   ├── session.go           # Agent session management
│   │   └── directives.go        # Agent behavior directives
│   ├── config/config.go         # Config management (~/.ramorie/config.json)
│   ├── crypto/                  # Zero-knowledge encryption
│   │   ├── crypto.go            # AES-256-GCM primitives
│   │   ├── vault.go             # Vault state (lock/unlock)
│   │   └── keyring.go           # System keyring
│   ├── models/models.go         # Data models (Task, Memory, Project, etc.)
│   ├── service/api.go           # Service layer
│   └── errors/api_errors.go     # API error parsing
├── Makefile                     # Build/test/install commands
├── .goreleaser.yaml             # Release automation
└── go.mod                       # Go modules
```

## Architecture

```
User/Agent → CLI Commands   → API Client → Backend API
                ↓
           MCP Server (stdio) → Tool Handlers → API Client → Backend API
                ↓
           Crypto/Vault → Keyring (system-level key storage)
```

### Two Entry Paths
1. **Human CLI**: `ramorie task list` → urfave/cli command → API client
2. **MCP Server**: `ramorie mcp serve` → stdio transport → tool handler → API client

## Conventions

### Naming
- **Commands**: `internal/cli/commands/feature.go` → `NewFeatureCommand()` returning `*cli.Command`
- **Subcommands**: `featureListCmd()`, `featureCreateCmd()`, etc.
- **MCP Tools**: `handleFeatureTool()` with `FeatureToolInput` struct
- **API Methods**: `client.ListFeatures()`, `client.CreateFeature()`, etc.

### Module Path
```go
import "github.com/kutbudev/ramorie-cli/internal/api"
import "github.com/kutbudev/ramorie-cli/internal/models"
```

## API Client

- **Base URL**: `https://api.ramorie.com/v1`
- **Auth**: `Authorization: Bearer <API_KEY>`
- **Timeout**: 30 seconds
- **Agent Headers** (MCP mode):
  - `X-Created-Via: mcp`
  - `X-Agent-Name: <name>`
  - `X-Agent-Model: <model>`
  - `X-Agent-Session-ID: <session_id>`

## Configuration

**Location**: `~/.ramorie/config.json`
```json
{
  "api_key": "your-api-key",
  "encryption_enabled": true,
  "encrypted_symmetric_key": "base64...",
  "key_nonce": "base64...",
  "salt": "base64...",
  "kdf_iterations": 310000,
  "kdf_algorithm": "pbkdf2-sha256"
}
```

## Encryption (Zero-Knowledge)

- **Algorithm**: AES-256-GCM
- **KDF**: PBKDF2-SHA256 (310,000 iterations)
- **Salt**: 16 bytes, **Nonce**: 12 bytes
- **Vault**: System keyring stores unlock state
- **Decryption**: Client-side only, master password never leaves device

## MCP Tool Tiers

| Tier | Count | Purpose |
|------|-------|---------|
| ESSENTIAL | 7 | Core workflow (setup, tasks, memories, focus) |
| COMMON | 12 | Extended features (details, context packs, decisions) |
| ADVANCED | 7 | Power features (subtasks, dependencies, plans) |

## Development Commands

```bash
make build          # Build for current platform
make build-all      # All platforms (linux/darwin/windows, amd64/arm64)
make install        # Install to /usr/local/bin (sudo)
make dev-install    # Install without sudo
make test           # Run tests
make test-coverage  # Coverage report
make fmt            # Format code
make lint           # golangci-lint
make clean          # Remove build artifacts
make setup-dev      # go mod tidy + download
make tag            # Create release tag
```

## Common Mistakes

- Forgetting to check `checkSessionInit()` in MCP tool handlers
- Not resolving project name to UUID before API calls
- Not handling vault-locked state for encrypted content
- Missing `strings.TrimSpace()` on user input
- Not wrapping MCP responses via `wrapResultAsObject()` (causes MCP spec errors)
- Forgetting to add new commands to `main.go` app registration
- Not updating `go.mod` after adding dependencies (`go mod tidy`)
- Using npm (this is Go, use `make`)
