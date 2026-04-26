<p align="center">
  <img src="https://ramorie.com/logo.png" alt="Ramorie" width="120" height="120">
</p>

<h1 align="center">Ramorie CLI</h1>

<p align="center">
  <strong>AI-powered task and memory management for developers and AI agents</strong>
</p>

<p align="center">
  <a href="https://ramorie.com">Website</a> •
  <a href="https://ramorie.com/docs">Documentation</a> •
  <a href="https://github.com/kutbudev/ramorie-cli/releases">Releases</a>
</p>

<p align="center">
  <img src="https://img.shields.io/github/v/release/kutbudev/ramorie-cli?style=flat-square&color=00d4aa" alt="Release">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue?style=flat-square" alt="Platform">
  <img src="https://img.shields.io/badge/go-%3E%3D1.21-00ADD8?style=flat-square&logo=go" alt="Go Version">
  <img src="https://img.shields.io/github/license/kutbudev/ramorie-cli?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/MCP-compatible-purple?style=flat-square" alt="MCP Compatible">
</p>

---

## ✨ What is Ramorie?

**Ramorie** is a productivity platform that combines task management with an intelligent memory system. The CLI provides:

- **🎯 Smart Task Management** — Create, organize, and track tasks with priorities, tags, and progress
- **🧠 Memory System** — Store and retrieve knowledge, insights, and learnings with semantic search
- **🤖 AI Integration** — Gemini-powered suggestions, task analysis, and intelligent tagging
- **📊 Visual Dashboards** — Kanban boards, burndown charts, and project statistics
- **🔗 MCP Support** — Model Context Protocol integration for AI agents (Cursor, Claude, etc.)

> **Perfect for developers, AI agents, and anyone who wants to capture knowledge while managing tasks.**

---

## 🚀 Installation

### Homebrew (Recommended for macOS/Linux)

```bash
brew tap kutbudev/homebrew-tap
brew install ramorie
```

### npm (Cross-platform)

```bash
npm install -g ramorie
```

### Go Install

```bash
go install github.com/kutbudev/ramorie-cli/cmd/jbraincli@latest
```

### Direct Download

Download pre-built binaries from the [releases page](https://github.com/kutbudev/ramorie-cli/releases/latest):

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS | Apple Silicon (M1/M2/M3) | [ramorie_darwin_arm64.tar.gz](https://github.com/kutbudev/ramorie-cli/releases/latest) |
| macOS | Intel | [ramorie_darwin_amd64.tar.gz](https://github.com/kutbudev/ramorie-cli/releases/latest) |
| Linux | x86_64 | [ramorie_linux_amd64.tar.gz](https://github.com/kutbudev/ramorie-cli/releases/latest) |
| Linux | ARM64 | [ramorie_linux_arm64.tar.gz](https://github.com/kutbudev/ramorie-cli/releases/latest) |
| Windows | x86_64 | [ramorie_windows_amd64.zip](https://github.com/kutbudev/ramorie-cli/releases/latest) |

### curl Install Script

```bash
curl -sSL https://raw.githubusercontent.com/kutbudev/ramorie-cli/main/install.sh | bash
```

### Build from Source

```bash
git clone https://github.com/kutbudev/ramorie-cli.git
cd ramorie-cli
go build -o ramorie ./cmd/jbraincli
```

---

## 🏁 Quick Start

### 1. Authenticate

Run the interactive setup menu (it can register a new account or log in to an existing one):

```bash
ramorie setup            # interactive: Login / Register / API key / status
# or directly:
ramorie setup login      # email + password
```

Credentials are written to `~/.ramorie/config.json`.

### 2. Create Your First Project

```bash
ramorie project create "My Project"
ramorie project list
```

### 3. Start Managing Tasks

```bash
# Create a task
ramorie task create "Implement user authentication" --priority H

# View your kanban board
ramorie kanban

# Start working on a task
ramorie task start <task-id>

# Mark it complete
ramorie task complete <task-id>
```

### 4. Store Knowledge

```bash
# Remember important insights (project resolves by name, short id, or UUID)
ramorie remember "Use bcrypt with 12 rounds for password hashing" -p "My Project"

# Pipe content from stdin (great for multi-line / generated text)
echo "Use bcrypt with 12 rounds for password hashing" | ramorie remember -p "My Project"

# Comma-separated tags + JSON output for scripts/agents
cat memo.md | ramorie remember -p "ramorie-cli" -t cli,docs --json

# Search your memories
ramorie find "password"
```

---

## 📋 Core Features

### 1. **Task Management**
- ✅ Create, list, start, complete tasks
- ✅ Priority levels (H/M/L)
- ✅ Status tracking (TODO/IN_PROGRESS/IN_REVIEW/COMPLETED)
- ✅ Progress tracking (0-100%)
- ✅ Task annotations and notes
- ✅ Detailed task information display
- 🔄 Task dependencies (coming soon)

### 2. **Project Organization**
- ✅ Multi-project support
- ✅ Active project switching
- ✅ Project-scoped tasks and memories

### 3. **Memory System**
- ✅ Store development insights and learnings
- ✅ Search and recall memories
- ✅ Project-specific or global memory views

### 4. **Annotation System**
- ✅ Add notes to tasks
- ✅ Track implementation details
- ✅ View annotation history with timestamps

### 5. **Visual Management**
- ✅ Beautiful kanban board with priority indicators
- ✅ Terminal-width responsive design
- ✅ Real-time task counts per status

## 🛠 Configuration

### Cloud Service
The CLI connects to a hosted PostgreSQL service automatically. No database setup required! Simply register for an account and start using the tool.

### Account Setup
After installation, authenticate:
```bash
# Interactive menu (includes register, login, API key, status)
ramorie setup

# Or directly log in with an existing account
ramorie setup login
```

Your API key will be stored in `~/.ramorie/config.json`.

### Gemini API Key Setup

Some advanced features (AI-powered suggestions, tag generation, etc.) require a [Google Gemini API key](https://aistudio.google.com/app/apikey).
**Prerequisite:** Register for a Gemini API key if you haven't already.

**To securely set or update your Gemini API key:**
```bash
ramorie config set-gemini-key
```
You will be prompted to enter your key, which will be stored securely in your home directory (`~/.ramorie_gemini_key`, permissions 0600).

**To remove your Gemini API key:**
```bash
ramorie config set-gemini-key --remove
```

**Environment Variables:**
If you prefer, you can set the `GEMINI_API_KEY` environment variable instead of using the CLI command. The CLI will prioritize the environment variable if both are set.


### Data Model

**Key Tables:**
- `tasks` - Task management
- `projects` - Project organization
- `memory_items` - Knowledge storage
- `contexts` - Context grouping
- `annotations` - Task notes
- `tags` - Tagging system

## Commands

### 🔴 Essential — daily

| Command | Purpose |
|---|---|
| `ramorie task` | List, create, update, link, note tasks |
| `ramorie memory` | List, get, link memories |
| `ramorie project` | Manage projects (accepts name, short id, or UUID) |
| `ramorie remember <text>` | Quick memory create (auto-detects type, supports stdin pipe + `--json`) |
| `ramorie find <term>` | Hybrid memory search (HyDE + rerank + entity graph) |
| `ramorie ui` | Interactive 3-pane TUI navigator (Yazi-style) |

### 🟡 Common — frequent

| Command | Purpose |
|---|---|
| `ramorie kanban -p <project>` | Beautified three-column board |
| `ramorie stats` | Task counts (todo / in-progress / done) — auto-JSON when piped |
| `ramorie activity [--burndown]` | Activity feed or burndown report — auto-JSON when piped |
| `ramorie subtask` | Manage subtasks |
| `ramorie context` | Manage contexts and packs |

### 🟢 Admin — setup

| Command | Purpose |
|---|---|
| `ramorie setup` | Interactive auth + `vault unlock\|lock\|status` subgroup |
| `ramorie unlock` | Unlock the encrypted vault (alias of `setup vault unlock`, since v6.3.5) |
| `ramorie lock` | Lock the encrypted vault (alias of `setup vault lock`, since v6.3.5) |
| `ramorie config` | Show config, set API key, set Gemini key |
| `ramorie mcp` | MCP server management |
| `ramorie hook` | Claude Code PreToolUse hook |
| `ramorie org` | Organizations + `vault` + `encryption` subgroups |

> Project / org / task arguments accept name, 8-char short ID, or full UUID.
>
> **List commands** (`task list`, `memory list`, `activity`) render as
> responsive lipgloss tables. Default order is **chronological ascending**
> (newest at the bottom), with a `"N of M · ↓ newest"` subtitle. Use
> `--limit N` to cap rows and `--newest-first` to flip the order.

## 🤖 AI Agent Decision Guide

### **When to use `task create` vs `remember`**

#### Use `task create` for:
- ✅ **Actionable work items** that need to be completed
- ✅ **Future tasks** you need to track and execute
- ✅ **Bugs to fix** or **features to implement**
- ✅ **Work that has clear completion criteria**

```bash
# Examples of GOOD task create usage:
ramorie task create "Fix authentication bug in login endpoint" --priority H
ramorie task create "Implement user profile editing feature"
ramorie task create "Write unit tests for payment module"
ramorie task create "Deploy version 2.1 to production"
```

#### Use `remember` for:
- ✅ **Insights and learnings** from completed work
- ✅ **Technical solutions** you discovered
- ✅ **Best practices** and patterns
- ✅ **Things to avoid** or lessons learned
- ✅ **Knowledge** that will help future development

```bash
# Examples of GOOD remember usage:
ramorie remember "OAuth requires redirect_uri to match exactly - case sensitive" -p "My Project"
ramorie remember "Use bcrypt with 12 rounds for password hashing" -p "My Project" -t auth,security
echo "Redis connection pooling reduces latency by 40%" | ramorie remember -p "My Project"
cat lessons.md | ramorie remember -p "My Project" -t perf --json
```

> `-p` accepts project **name**, 8-char short ID, or full UUID. Content
> can come from positional args **or** piped stdin (v6.4.0+).

#### Use `task note` for:
- ✅ **Progress updates** on existing tasks
- ✅ **Implementation details** and decisions
- ✅ **Blocking issues** or dependencies
- ✅ **Code snippets** or specific technical notes

```bash
# Examples of GOOD task note usage:
ramorie task note a1b2c3d4 "Switched from JWT to session-based auth for better security"
ramorie task note a1b2c3d4 "Blocked: waiting for API key from third-party service"
ramorie task note a1b2c3d4 "Performance improved 3x after adding database indexes"
```

## 🎯 Workflow Examples

### **Starting a New Feature**
```bash
# 1. Make sure the project exists (projects are passed per-command via --project)
ramorie project list

# 2. Create high-priority task
ramorie task create "Implement OAuth integration" --project my-app --priority H --tags "auth,api"

# 3. Start working
ramorie task start <task-id>

# 4. View progress
ramorie kanban

# 5. Update progress as you work
ramorie task progress <task-id> 50

# 6. Store learnings
ramorie remember "OAuth requires redirect_uri to match exactly - case sensitive" -p my-app

# 7. Complete task
ramorie task complete <task-id>
```

### **Daily Standup Prep**
```bash
# Check kanban for current status
ramorie kanban

# List your in-progress tasks
ramorie task list --status IN_PROGRESS

# Review recent memories for insights
ramorie memory list | head -10

# Check completed tasks
ramorie task list --status COMPLETED
```

### **Project Retrospective**
```bash
# Review all project memories
ramorie memory list

# Check task completion stats via kanban
ramorie kanban

# Search for specific learnings
ramorie find "performance"
ramorie find "bug"
```

## 🔧 Advanced Usage

### **Cross-Project Memory Access**
```bash
# View memories from all projects
ramorie memory list --all

# Search across all projects
ramorie find "database" --all
```

### **Task Dependencies & Workflows**
```bash
# Create linked tasks with context
ramorie task create "Backend API" --priority H --context "feature-x"
ramorie task create "Frontend UI" --priority M --context "feature-x"
ramorie task create "Integration tests" --priority L --context "feature-x"
```

### **Bulk Operations**
```bash
# List tasks by multiple criteria
ramorie task list --priority H --status TODO --context "urgent"

# Filter memories by search
ramorie find "TypeScript" --all
```

## 📊 Data Model

The CLI uses a PostgreSQL database with the following key relationships:

- **Projects** → contain **Tasks** and **Memory Items**
- **Tasks** → have **Annotations** and **Tags**
- **Memory Items** → have **Tags** and can link to **Tasks**
- **Contexts** → group related **Tasks** and **Memories**

All data is persisted and synchronized across CLI sessions.

## 🚨 Status Codes & Priorities

### **Task Status**
- `TODO` - Not started
- `IN_PROGRESS` - Currently working
- `IN_REVIEW` - Awaiting review/QA
- `COMPLETED` - Finished

### **Priority Levels**
- 🔴 `H` (High) - Urgent/Critical
- 🟡 `M` (Medium) - Normal priority
- 🟢 `L` (Low) - Nice to have

## 💡 AI Agent Best Practices

### **For Task Management:**
1. **Always specify project**: Use `--project` flag to specify which project to operate on
2. **Use descriptive task names**: Include what, not how
3. **Track progress with annotations**: Document blockers, decisions, and progress
4. **Use task info for context**: Before working on a task, review its full details
5. **Complete tasks promptly**: Mark done when finished to maintain accurate status

### **For Knowledge Management:**
1. **Store insights immediately**: Use `ramorie remember` to capture learnings as they happen
2. **Be specific in memories**: Include context, not just solutions
3. **Search before creating**: Check existing memories and tasks to avoid duplication
4. **Use annotations for implementation details**: Keep task-specific notes with the task
5. **Separate concerns**: Use tasks for work items, memories for knowledge, annotations for progress

### **For Workflow Optimization:**
1. **Start with kanban overview**: `ramorie kanban` gives complete project status
2. **Use partial UUIDs**: First 8 characters are sufficient for task operations (e.g., `a1b2c3d4`)
3. **Review task info before starting**: Understand context and previous annotations
4. **Document as you work**: Add annotations during development, not just at the end
5. **Capture learnings in the moment**: Don't wait until the end to record insights

### **Common Anti-Patterns to Avoid:**
❌ Creating tasks for already completed work
❌ Using remember for future work items
❌ Storing implementation details in memories instead of task annotations
❌ Creating duplicate tasks without checking existing ones
❌ Forgetting to mark tasks as complete

## 🔌 Integration

The CLI integrates with:
- **Cloud PostgreSQL** - Hosted data storage (no setup required)
- **API Authentication** - Secure API key-based access
- **Terminal** - Full CLI interface with colors and tables
- **Cross-platform** - Works on macOS, Linux, and Windows

This tool is designed for developers and AI agents who want powerful task management with persistent memory storage, all accessible through a beautiful command-line interface.

## 🤖 MCP Integration (AI Agents)

Ramorie CLI includes a built-in **Model Context Protocol (MCP)** server, making it compatible with AI coding assistants like **Cursor**, **Claude Desktop**, and other MCP-enabled tools.

### Setup for Cursor/Claude

Add to your MCP configuration:

```json
{
  "mcpServers": {
    "ramorie": {
      "command": "ramorie",
      "args": ["mcp", "serve"]
    }
  }
}
```

### Available MCP Tools (v5.0.0 — 14 tools)

| Tool | Tier | Description |
|------|------|-------------|
| `setup_agent` | Core | Initialize session, auto-detect project from cwd |
| `list_projects` | Core | List personal + org projects |
| `remember` | Core | Store memory (auto type detection; `todo:` prefix → task) |
| `find` | Core | Hybrid semantic + lexical search (HyDE + rerank) |
| `recall` | Core | FTS search; `precision: true` routes to `find` |
| `task` | Core | Unified task ops: list/get/create/start/complete/stop/progress/note/move |
| `memory` | Common | Unified memory ops + skill generation via `goal` |
| `get_stats` | Common | Task statistics |
| `get_agent_activity` | Common | Agent timeline query |
| `surface_context` | Common | File/domain-scoped context surfacing |
| `create_project` | Advanced | Create project |
| `manage_subtasks` | Advanced | Subtask CRUD |
| `entity` | Advanced | Knowledge graph ops (10 actions) |
| `admin` | Advanced | consolidate/cleanup/orgs/export/import/plan/analyze |

> **v5.0.0 upgrade note:** Legacy tools (`create_task`, `add_memory`,
> `search_memories`, `create_decision`, `activate_context_pack`, `skill`,
> `decision`, `get_active_task`, `manage_focus`) have been removed. Restart
> your MCP client (Claude Code / Cursor / Windsurf) after upgrading.
> Run `ramorie mcp tools` for the live list.

---

## 📊 Reports & Analytics

```bash
# Project statistics
ramorie stats

# Activity feed (last 7 days)
ramorie activity -d 7

# Burndown report
ramorie activity --burndown
```

---

## 🔧 Configuration

### API Key Storage

Your credentials are stored securely in `~/.ramorie/config.json`.

### Gemini AI Setup (Optional)

For AI-powered features (suggestions, analysis, auto-tagging):

```bash
ramorie config set-gemini-key
```

Or set the environment variable:
```bash
export GEMINI_API_KEY="your-api-key"
```

---

## 🤖 MCP Integration (AI Agents)

Ramorie includes a built-in MCP (Model Context Protocol) server, allowing AI agents like **Cursor**, **Windsurf**, **Claude Desktop**, and others to manage tasks and memories directly.

### Quick Setup

Run this command to get your MCP configuration:

```bash
ramorie mcp config
# or for Codex CLI users
ramorie mcp config --client codex
```

### Windsurf / Cursor Configuration

Add the following to your MCP config file:

**Windsurf:** `~/.codeium/windsurf/mcp_config.json`
**Cursor:** `~/.cursor/mcp.json`

#### If installed via Homebrew (macOS/Linux):
```json
{
  "mcpServers": {
    "ramorie": {
      "command": "ramorie",
      "args": ["mcp", "serve"]
    }
  }
}
```

#### If installed via npm:
```json
{
  "mcpServers": {
    "ramorie": {
      "command": "npx",
      "args": ["-y", "@ramorie/cli", "mcp", "serve"]
    }
  }
}
```

#### If installed via Go:
```json
{
  "mcpServers": {
    "ramorie": {
      "command": "ramorie",
      "args": ["mcp", "serve"]
    }
  }
}
```

> **Note:** Make sure `ramorie` is in your PATH, or use the full path (e.g., `/opt/homebrew/bin/ramorie` or `~/.local/bin/ramorie`).

### Codex CLI Configuration

Codex CLI uses `~/.codex/config.toml`. Generate a ready-to-paste snippet with:

```bash
ramorie mcp config --client codex
```

Then append the output to your config file:

```toml
[mcp_servers.ramorie]
command = "ramorie"
args = ["mcp", "serve"]
enabled = true
```

Restart Codex CLI (or the Codex app) after editing so it reloads the MCP settings.

### Available MCP Tools

See the full 14-tool table above (v5.0.0). For a live list straight from the binary:

```bash
ramorie mcp tools
```

### Verify MCP Server

```bash
# List all available tools
ramorie mcp tools

# Start MCP server manually (for testing)
ramorie mcp serve
```

---

## 📝 Quick Reference

```bash
# Essential Commands
ramorie setup                            # Authenticate (email + password)
ramorie project list                     # List all projects
ramorie kanban                           # Visual task board
ramorie task create "Description"        # New task
ramorie task show <id>                   # Task details
ramorie task start <id>                  # Begin working
ramorie task note <id> "Note"            # Add progress note
ramorie remember "Insight" -p <project>  # Store knowledge (or pipe via stdin)
ramorie find "term"                      # Hybrid memory search
ramorie unlock                           # Unlock encrypted vault (alias of `setup vault unlock`)
ramorie lock                             # Lock encrypted vault (alias of `setup vault lock`)
ramorie task complete <id>               # Complete task

# Decision Guide
# Need to do work?      → task create
# Making progress?      → task note
# Learned something?    → remember
# Need overview?        → kanban
# Need details?         → task show
```

---

## 🌐 Links

- **Website:** [ramorie.com](https://ramorie.com)
- **Documentation:** [ramorie.com/docs](https://ramorie.com/docs)
- **Releases:** [GitHub Releases](https://github.com/kutbudev/ramorie-cli/releases)
- **Homebrew Tap:** [kutbudev/homebrew-tap](https://github.com/kutbudev/homebrew-tap)

---

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/terzigolu">terzigolu</a>
</p>

<p align="center">
  <a href="https://ramorie.com">🌐 ramorie.com</a>
</p>
