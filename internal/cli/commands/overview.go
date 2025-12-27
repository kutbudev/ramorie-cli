package commands

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// NewOverviewCommand creates the overview command.
func NewOverviewCommand() *cli.Command {
	return &cli.Command{
		Name:    "overview",
		Aliases: []string{"help-all"},
		Usage:   "Show all available features and commands",
		Action: func(c *cli.Context) error {
			fmt.Print(`
╔═══════════════════════════════════════════════════════════════════╗
║                       🧠 Ramorie CLI                              ║
║                      Feature Overview                             ║
╚═══════════════════════════════════════════════════════════════════╝

📋 TASK MANAGEMENT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ramorie task list              List all tasks
  ramorie task create "title"    Create a new task
  ramorie task show <id>         Show task details
  ramorie task update <id>       Update task properties
  ramorie task start <id>        Start a task (IN_PROGRESS)
  ramorie task done <id>         Complete a task (COMPLETED)
  ramorie task delete <id>       Delete a task
  ramorie task duplicate <id>    Duplicate a task with notes
  ramorie task move <ids> -p X   Move tasks to another project
  ramorie task next              Show next tasks by priority
  ramorie task progress <id> N   Update task progress (0-100)
  ramorie task elaborate <id>    AI elaboration on task

📁 PROJECT MANAGEMENT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ramorie project list           List all projects
  ramorie project create "name"  Create a new project
  ramorie project show <id>      Show project details
  ramorie project use <name>     Set active project
  ramorie project delete <id>    Delete a project

🧠 MEMORY (KNOWLEDGE BASE)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ramorie remember "content"     Add a new memory
  ramorie memory list            List all memories
  ramorie memory search "term"   Search memories
  ramorie memory show <id>       Show memory details
  ramorie memory delete <id>     Delete a memory

🔗 LINKING (TASK ↔ MEMORY)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ramorie link <task> <memory>   Link a task to a memory
  ramorie task-memories <id>     List memories for a task
  ramorie memory-tasks <id>      List tasks for a memory

📝 SUBTASKS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ramorie subtask list <task>    List subtasks
  ramorie subtask add <task> "X" Add a subtask
  ramorie subtask done <t> <s>   Complete a subtask
  ramorie subtask delete <t> <s> Delete a subtask

📌 ANNOTATIONS (NOTES)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ramorie annotate <id> "note"   Add a note to a task
  ramorie task-annotations <id>  List notes for a task

🎯 CONTEXTS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ramorie context list           List all contexts
  ramorie context create "name"  Create a new context
  ramorie context use <name>     Set active context
  ramorie context delete <name>  Delete a context

📊 REPORTS & VIEWS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ramorie kanban                 Kanban board view
  ramorie reports stats          Task statistics

⚙️  CONFIGURATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ramorie setup                  Configure authentication
  ramorie setup login            Login with credentials
  ramorie setup logout           Remove saved credentials
  ramorie setup status           Check auth status
  ramorie config                 View/edit configuration
  ramorie set-gemini-key         Set Gemini API key

🤖 MCP (Model Context Protocol)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ramorie mcp start              Start MCP server
  ramorie mcp status             Check MCP server status

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💡 TIP: Use 'ramorie <command> --help' for detailed command usage.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`)
			return nil
		},
	}
}
