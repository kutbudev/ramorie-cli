# JosephsBrain CLI

A powerful command-line interface for managing tasks, projects, and memories with JosephsBrain.

## 🚀 Installation

### Option 1: Using go install (Requires Go 1.24+)
```bash
go install github.com/terzigolu/josepshbrain-go/cmd/jbraincli@latest
```

### Option 2: Download Pre-built Binaries

#### macOS
```bash
# For Intel Macs
curl -L -o jbraincli https://github.com/terzigolu/josepshbrain-go/releases/latest/download/jbraincli-darwin-amd64
chmod +x jbraincli
sudo mv jbraincli /usr/local/bin/

# For Apple Silicon Macs (M1/M2)
curl -L -o jbraincli https://github.com/terzigolu/josepshbrain-go/releases/latest/download/jbraincli-darwin-arm64
chmod +x jbraincli
sudo mv jbraincli /usr/local/bin/
```

#### Linux
```bash
# For AMD64
curl -L -o jbraincli https://github.com/terzigolu/josepshbrain-go/releases/latest/download/jbraincli-linux-amd64
chmod +x jbraincli
sudo mv jbraincli /usr/local/bin/

# For ARM64
curl -L -o jbraincli https://github.com/terzigolu/josepshbrain-go/releases/latest/download/jbraincli-linux-arm64
chmod +x jbraincli
sudo mv jbraincli /usr/local/bin/
```

#### Windows
1. Download the appropriate binary:
   - [AMD64](https://github.com/terzigolu/josepshbrain-go/releases/latest/download/jbraincli-windows-amd64.exe)
   - [ARM64](https://github.com/terzigolu/josepshbrain-go/releases/latest/download/jbraincli-windows-arm64.exe)
2. Rename to `jbraincli.exe` and add to your PATH

### Option 3: One-line Installation Script

#### macOS/Linux
```bash
curl -sSL https://raw.githubusercontent.com/terzigolu/josepshbrain-go/main/install.sh | bash
```

#### Windows (PowerShell)
```powershell
iwr -useb https://raw.githubusercontent.com/terzigolu/josepshbrain-go/main/install.ps1 | iex
```

## 🏁 Quick Start

After installation, run:
```bash
jbraincli setup register
```

## 📚 Usage

See the full documentation in our [user guide](jbraincli_quick_reference_updated.md).

Quick commands:
- `jbraincli task create "My new task"` - Create a task
- `jbraincli task list` - List all tasks
- `jbraincli project create "My Project"` - Create a project
- `jbraincli kanban` - View tasks in kanban board
- `jbraincli remember "Important information"` - Store a memory

## 🔧 Development

To build from source:
```bash
git clone https://github.com/terzigolu/josepshbrain-go.git
cd josepshbrain-go
go build -o jbraincli ./cmd/jbraincli
```

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
After installation, create your account:
```bash
# Register new account
jbraincli setup register

# Or login with existing account
jbraincli setup login
```

Your API key will be stored securely in `~/.jbrain/config.json`.

### Gemini API Key Setup

Some advanced features (AI-powered suggestions, tag generation, etc.) require a [Google Gemini API key](https://aistudio.google.com/app/apikey).
**Prerequisite:** Register for a Gemini API key if you haven't already.

**To securely set or update your Gemini API key:**
```bash
jbraincli set-gemini-key
```
You will be prompted to enter your key, which will be stored securely in your home directory (`~/.jbrain_gemini_key`, permissions 0600).

**To remove your Gemini API key:**
```bash
jbraincli set-gemini-key --remove
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

## 📚 Command Reference

### **Project Commands**
```bash
# Project lifecycle
jbraincli project init <name>              # Create new project
jbraincli project use [name]               # Set active project  
jbraincli project list                     # List all projects
jbraincli project delete <name>            # Delete project

# Examples
jbraincli project init "orkai-backend"
jbraincli project use orkai-backend
```

### **Task Commands**
```bash
# Task creation & management
jbraincli task create <description>         # Create task
jbraincli task list                        # List tasks
jbraincli task info <id>                   # Show detailed task information
jbraincli task start <id>                  # Start working on task
jbraincli task done <id>                   # Mark task complete

# Coming soon:
# jbraincli task progress <id> <0-100>     # Update progress
# jbraincli task modify <id> [flags]       # Modify task properties
# jbraincli task delete <id>               # Delete task

# Examples
jbraincli task create "Implement user authentication"
jbraincli task create "Write unit tests"
jbraincli task start a1b2c3d4              # Using partial task ID
jbraincli task info a1b2c3d4               # View full details
jbraincli task done a1b2c3d4               # Mark complete
```

### **Annotation Commands**
```bash
# Add notes and details to tasks
jbraincli annotate <task-id> <note>        # Add annotation to task
jbraincli task-annotations <task-id>       # List all annotations for task

# Examples
jbraincli annotate a1b2c3d4 "Fixed authentication bug by updating JWT validation"
jbraincli annotate a1b2c3d4 "Used bcrypt for password hashing"
jbraincli task-annotations a1b2c3d4       # View all notes for this task
```

### **Memory Commands**
```bash
# Memory management
jbraincli remember <text>                   # Store new insight/learning
jbraincli memories [flags]                  # List memories

jbraincli memory recall <search_term>    # Search memories
jbraincli memory get <id>                # View a memory by ID
jbraincli memory forget <id>             # Delete memory

# Examples
jbraincli remember "Use connection pooling for better database performance"
jbraincli remember "Bug in API rate limiting - fix with exponential backoff"
jbraincli memories                         # See project memories
```

### **Visual Commands**
```bash
# Kanban board
jbraincli kanban                               # Display kanban board

# Output example:
┌──────────────┬──────────────┬──────────────┬──────────────┐
│ TODO (3)     │ IN_PROGRESS  │ IN_REVIEW (1)│ COMPLETED (2)│
│              │ (2)          │              │              │
├──────────────┼──────────────┼──────────────┼──────────────┤
│ 🔴 Fix login │ 🟡 API tests │ 🟢 User auth │ ✅ Database  │
│ 🟡 Add logs  │ 🔴 Security  │              │ ✅ Setup CI  │
│ 🟢 Cleanup   │              │              │              │
└──────────────┴──────────────┴──────────────┴──────────────┘
```

## 🤖 AI Agent Decision Guide

### **When to use `task create` vs `remember`**

#### Use `task create` for:
- ✅ **Actionable work items** that need to be completed
- ✅ **Future tasks** you need to track and execute
- ✅ **Bugs to fix** or **features to implement**
- ✅ **Work that has clear completion criteria**

```bash
# Examples of GOOD task create usage:
jbraincli task create "Fix authentication bug in login endpoint"
jbraincli task create "Implement user profile editing feature"
jbraincli task create "Write unit tests for payment module"
jbraincli task create "Deploy version 2.1 to production"
```

#### Use `remember` for:
- ✅ **Insights and learnings** from completed work
- ✅ **Technical solutions** you discovered
- ✅ **Best practices** and patterns
- ✅ **Things to avoid** or lessons learned
- ✅ **Knowledge** that will help future development

```bash
# Examples of GOOD remember usage:
jbraincli remember "OAuth requires redirect_uri to match exactly - case sensitive"
jbraincli remember "Use bcrypt with 12 rounds for password hashing - good performance/security balance"
jbraincli remember "Redis connection pooling reduces latency by 40% in high-traffic scenarios"
jbraincli remember "Avoid using SELECT * in production queries - causes performance issues"
```

#### Use `annotate` for:
- ✅ **Progress updates** on existing tasks
- ✅ **Implementation details** and decisions
- ✅ **Blocking issues** or dependencies
- ✅ **Code snippets** or specific technical notes

```bash
# Examples of GOOD annotate usage:
jbraincli annotate a1b2c3d4 "Switched from JWT to session-based auth for better security"
jbraincli annotate a1b2c3d4 "Blocked: waiting for API key from third-party service"
jbraincli annotate a1b2c3d4 "Performance improved 3x after adding database indexes"
```

## 🎯 Workflow Examples

### **Starting a New Feature**
```bash
# 1. Create or switch to project
jbraincli project use "my-app"

# 2. Create high-priority task
jbraincli task create "Implement OAuth integration" --priority H --tags "auth,api"

# 3. Start working
jbraincli task start <task-id>

# 4. View progress
jbraincli kanban

# 5. Update progress as you work
jbraincli task progress <task-id> 50

# 6. Store learnings
jbraincli remember "OAuth requires redirect_uri to match exactly - case sensitive"

# 7. Complete task
jbraincli task done <task-id>
```

### **Daily Standup Prep**
```bash
# Check kanban for current status
jbraincli kanban

# List your in-progress tasks
jbraincli task list --status IN_PROGRESS

# Review recent memories for insights
jbraincli memories | head -10

# Check completed tasks
jbraincli task list --status COMPLETED
```

### **Project Retrospective**
```bash
# Review all project memories
jbraincli memories

# Check task completion stats via kanban
jbraincli kanban

# Search for specific learnings
jbraincli memory recall "performance"
jbraincli memory recall "bug"
```

## 🔧 Advanced Usage

### **Cross-Project Memory Access**
```bash
# View memories from all projects (282+ total memories available)
jbraincli memories --all

# Search across all projects
jbraincli memory recall "database" --all
```

### **Task Dependencies & Workflows**
```bash
# Create linked tasks with context
jbraincli task create "Backend API" --priority H --context "feature-x"
jbraincli task create "Frontend UI" --priority M --context "feature-x"
jbraincli task create "Integration tests" --priority L --context "feature-x"
```

### **Bulk Operations**
```bash
# List tasks by multiple criteria
jbraincli task list --priority H --status TODO --context "urgent"

# Filter memories by search
jbraincli memory recall "TypeScript" --all
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
1. **Always check active project**: `jbraincli project list` shows which project is active (✅)
2. **Use descriptive task names**: Include what, not how
3. **Track progress with annotations**: Document blockers, decisions, and progress
4. **Use task info for context**: Before working on a task, review its full details
5. **Complete tasks promptly**: Mark done when finished to maintain accurate status

### **For Knowledge Management:**
1. **Store insights immediately**: Use `jbraincli remember` to capture learnings as they happen
2. **Be specific in memories**: Include context, not just solutions
3. **Search before creating**: Check existing memories and tasks to avoid duplication
4. **Use annotations for implementation details**: Keep task-specific notes with the task
5. **Separate concerns**: Use tasks for work items, memories for knowledge, annotations for progress

### **For Workflow Optimization:**
1. **Start with kanban overview**: `jbraincli kanban` gives complete project status
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

## 🚀 Distribution & Release

### For Maintainers
```bash
# Create a new release
git tag v1.0.2
git push origin v1.0.2

# This triggers GitHub Actions to:
# - Build binaries for all platforms (Linux, macOS, Windows)
# - Create GitHub release with automated changelog
# - Upload assets: .tar.gz for Unix, .zip for Windows
```

### Current Status
- **✅ GitHub Releases**: Automated via GoReleaser + GitHub Actions
- **✅ GitHub CLI Install**: Primary installation method (works immediately)
- **✅ Direct Download**: Manual download from releases page
- **✅ Source Build**: Clone and compile locally
- **❌ Homebrew**: Unavailable (requires public repository)

### Installation Methods Available
- **GitHub CLI**: `gh release download --repo terzigolu/josepshbrain-go ...` (Recommended)
- **Direct Download**: [GitHub releases page](https://github.com/terzigolu/josepshbrain-go/releases)
- **Source Build**: `git clone && make install`

---

## 📝 Quick Reference Card

```bash
# Essential Commands (Most Used)
jbraincli project list                     # Check active project
jbraincli kanban                          # Overview of all work
jbraincli task create "Description"       # New work item
jbraincli task info <id>                  # Task details + notes
jbraincli task start <id>                 # Begin working
jbraincli annotate <id> "Progress note"   # Document progress
jbraincli remember "Learning or insight"  # Store knowledge
jbraincli task done <id>                  # Mark complete

# Decision Tree
# Need to do work? → task create
# Making progress? → annotate  
# Learned something? → remember
# Need overview? → kanban
# Need details? → task info
``` 