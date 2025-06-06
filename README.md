# JosephsBrain Go CLI - AI Agent Guide

**A powerful command-line productivity tool for task management, project organization, and memory storage with PostgreSQL persistence.**

## 🚀 Quick Start

```bash
# Install (binary available at /usr/local/bin/jbraincli)
jbraincli --help

# Set up a project
jbraincli project init "my-project"
jbraincli project use my-project

# Create tasks
jbraincli task create "Implement new feature" --priority H
jbraincli task create "Fix bug #123" --priority M

# Store memories
jbraincli remember "Fixed the database connection issue with connection pooling"

# View kanban board
jbraincli kanban
```

## 📋 Core Features

### 1. **Task Management**
- ✅ Create, update, delete tasks
- ✅ Priority levels (H/M/L) 
- ✅ Status tracking (TODO/IN_PROGRESS/IN_REVIEW/COMPLETED)
- ✅ Progress tracking (0-100%)
- ✅ Task dependencies

### 2. **Project Organization** 
- ✅ Multi-project support
- ✅ Active project switching
- ✅ Project-scoped tasks and memories

### 3. **Memory System**
- ✅ Store development insights and learnings
- ✅ Search and recall memories
- ✅ Project-specific or global memory views

### 4. **Visual Management**
- ✅ Beautiful kanban board with priority indicators
- ✅ Terminal-width responsive design
- ✅ Real-time task counts per status

## 🛠 Installation & Setup

### Prerequisites
```bash
# Ensure PostgreSQL is running
psql --version

# Environment variables (create .env file)
PG_HOST=localhost
PG_PORT=5432
PG_USER=postgres
PG_PASSWORD=your_password
PG_DATABASE=jbrain_dev
PG_SSL_MODE=disable
```

### Database Setup
The CLI automatically connects to your PostgreSQL database using the `.env` configuration. Tables are created automatically via GORM migrations.

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
jbraincli task create <description> [flags]    # Create task
jbraincli task list [flags]                    # List tasks
jbraincli task start <id>                      # Start working on task
jbraincli task done <id>                       # Mark task complete
jbraincli task progress <id> <0-100>           # Update progress
jbraincli task modify <id> [flags]             # Modify task properties
jbraincli task delete <id>                     # Delete task
jbraincli task info <id>                       # Show task details

# Flags
--priority, -p    # H (High), M (Medium), L (Low)
--context, -c     # Context name
--tags, -t        # Comma-separated tags

# Examples
jbraincli task create "Implement user authentication" --priority H
jbraincli task create "Write unit tests" --priority M --tags "testing,backend"
jbraincli task list --priority H --status TODO
jbraincli task start a1b2c3d4    # Using task ID
jbraincli task progress a1b2c3d4 75
jbraincli task done a1b2c3d4
```

### **Memory Commands**
```bash
# Memory management
jbraincli remember <text>                      # Store new memory
jbraincli memories [flags]                     # List memories 
jbraincli memory recall <search_term>          # Search memories
jbraincli memory forget <id>                   # Delete memory

# Flags
--all, -a         # Show memories from all projects
--search, -s      # Search term filter

# Examples
jbraincli remember "Use connection pooling for better database performance"
jbraincli remember "Bug in API rate limiting - fix with exponential backoff"
jbraincli memories --all                       # See all 282+ memories
jbraincli memory recall "database"             # Search for database-related memories
jbraincli memory forget f557dd79               # Delete specific memory
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

## 💡 AI Agent Tips

1. **Always check active project**: `jbraincli project list` shows which project is active (✅)
2. **Use kanban for overview**: `jbraincli kanban` gives complete project status
3. **Store insights immediately**: Use `jbraincli remember` to capture learnings as they happen
4. **Search before creating**: Use `jbraincli memory recall` to check if similar work was done
5. **Use --all flag**: Add `--all` to memory commands to search across all projects (282+ memories)
6. **Task IDs are UUIDs**: Use first 8 characters for task operations (e.g., `a1b2c3d4`)

## 🔌 Integration

The CLI integrates with:
- **PostgreSQL** - Primary data storage
- **Environment files** - Configuration via `.env`
- **Terminal** - Full CLI interface with colors and tables
- **File system** - Automatic .env discovery from multiple paths

This tool is designed for developers who want powerful task management with persistent memory storage, all accessible through a beautiful command-line interface. 