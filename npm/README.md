# Ramorie CLI

AI-powered task and memory management CLI for developers and AI agents.

## Installation

```bash
npm install -g ramorie
```

## Alternative Installation Methods

### Homebrew (macOS/Linux)
```bash
brew tap kutbudev/tap
brew install ramorie
```

### Direct Download
Download pre-built binaries from [GitHub Releases](https://github.com/kutbudev/ramorie-cli/releases/latest).

### Go Install
```bash
go install github.com/kutbudev/ramorie-cli/cmd/jbraincli@latest
```

## Quick Start

```bash
# Create an account
ramorie setup register

# Create your first project
ramorie project init "My Project"
ramorie project use "My Project"

# Create a task
ramorie task create "Implement user authentication" --priority H

# View kanban board
ramorie kanban

# Store knowledge
ramorie remember "Use bcrypt with 12 rounds for password hashing"

# Search memories
ramorie memory recall "password"
```

## Features

- 🎯 **Smart Task Management** - Priorities, progress tracking, subtasks
- 🧠 **Memory System** - Store and search development insights
- 🤖 **AI Integration** - Gemini-powered suggestions and analysis
- 📊 **Visual Dashboards** - Kanban boards, burndown charts, statistics
- 🔗 **MCP Support** - Model Context Protocol for AI agents (Cursor, Claude, etc.)

## Documentation

- **Website:** [ramorie.com](https://ramorie.com)
- **Docs:** [ramorie.com/docs](https://ramorie.com/docs)
- **GitHub:** [github.com/kutbudev/ramorie-cli](https://github.com/kutbudev/ramorie-cli)

## License

MIT

