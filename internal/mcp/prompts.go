package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerPrompts adds MCP prompt templates to the server
func registerPrompts(server *mcp.Server) {
	// Project Setup - Initialize workspace with project, context pack, initial tasks
	server.AddPrompt(&mcp.Prompt{
		Name:        "project_setup",
		Title:       "Project Setup",
		Description: "Initialize a new project workspace with context pack and initial tasks",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "project_name",
				Description: "Name of the project to set up",
				Required:    true,
			},
			{
				Name:        "description",
				Description: "Brief description of the project goals",
				Required:    true,
			},
			{
				Name:        "initial_tasks",
				Description: "Comma-separated list of initial tasks to create",
				Required:    false,
			},
		},
	}, handleProjectSetupPrompt)

	// Sprint Review - Review completed tasks, create ADRs, plan next sprint
	server.AddPrompt(&mcp.Prompt{
		Name:        "sprint_review",
		Title:       "Sprint Review",
		Description: "Review completed work, document decisions, and plan next sprint",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "project",
				Description: "Project name or ID to review",
				Required:    true,
			},
			{
				Name:        "sprint_goals",
				Description: "Goals for the next sprint (optional)",
				Required:    false,
			},
		},
	}, handleSprintReviewPrompt)

	// Memory Consolidation - Organize and link related memories
	server.AddPrompt(&mcp.Prompt{
		Name:        "memory_consolidation",
		Title:       "Memory Consolidation",
		Description: "Organize, deduplicate, and link related memories in a project",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "project",
				Description: "Project name or ID to consolidate memories for",
				Required:    true,
			},
			{
				Name:        "focus_area",
				Description: "Specific topic or area to focus consolidation on (optional)",
				Required:    false,
			},
		},
	}, handleMemoryConsolidationPrompt)

	// Daily Standup - Check pending tasks, recent activity, blockers
	server.AddPrompt(&mcp.Prompt{
		Name:        "daily_standup",
		Title:       "Daily Standup",
		Description: "Get a summary of pending tasks, recent activity, and blockers",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "project",
				Description: "Project name or ID for the standup",
				Required:    true,
			},
		},
	}, handleDailyStandupPrompt)
}

func handleProjectSetupPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	projectName := req.Params.Arguments["project_name"]
	description := req.Params.Arguments["description"]
	initialTasks := req.Params.Arguments["initial_tasks"]

	if projectName == "" {
		return nil, fmt.Errorf("project_name is required")
	}
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}

	var taskSection string
	if initialTasks != "" {
		tasks := strings.Split(initialTasks, ",")
		var taskLines []string
		for _, t := range tasks {
			t = strings.TrimSpace(t)
			if t != "" {
				taskLines = append(taskLines, fmt.Sprintf("- %s", t))
			}
		}
		if len(taskLines) > 0 {
			taskSection = fmt.Sprintf("\n\n## Initial Tasks to Create\n%s", strings.Join(taskLines, "\n"))
		}
	}

	promptText := fmt.Sprintf(`Please set up a new project workspace with the following configuration:

## Project Details
- **Name**: %s
- **Description**: %s

## Setup Steps
1. Create the project using create_project with the name and description above
2. Create a context pack named "%s-workspace" of type "project" to bundle related work
3. Set focus to the new context pack using set_focus%s
4. Start working on the first task if any were created

## Notes
- Check if a similar project already exists before creating
- The context pack serves as the workspace for this project
- All memories and tasks created while focused will be associated with this workspace`,
		projectName, description, projectName, taskSection)

	if taskSection != "" {
		promptText += "\n5. Create each initial task listed above using create_task"
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Set up workspace for project: %s", projectName),
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: promptText},
			},
		},
	}, nil
}

func handleSprintReviewPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	project := req.Params.Arguments["project"]
	sprintGoals := req.Params.Arguments["sprint_goals"]

	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	var goalsSection string
	if sprintGoals != "" {
		goalsSection = fmt.Sprintf(`

## Next Sprint Goals
%s
- Create tasks for each goal above`, sprintGoals)
	}

	promptText := fmt.Sprintf(`Conduct a sprint review for project "%s". Follow these steps:

## Step 1: Review Completed Work
- Call list_tasks with project="%s" and status="COMPLETED" to see what was accomplished
- Summarize key achievements and deliverables

## Step 2: Check In-Progress Work
- Call list_tasks with project="%s" and status="IN_PROGRESS" to identify ongoing work
- Note any tasks that seem stalled or blocked

## Step 3: Document Decisions
- For any significant technical decisions made during this sprint, create architectural decisions using create_decision
- Include context, consequences, and rationale

## Step 4: Identify Blockers
- Review pending tasks and note any dependencies or blockers
- Suggest actions to unblock stalled work

## Step 5: Plan Next Sprint
- Call get_next_tasks to see prioritized upcoming work
- Recommend which tasks to focus on next%s

## Output Format
Provide a structured summary with:
- Completed items count and highlights
- Blockers and risks
- Decisions documented
- Recommended focus for next sprint`,
		project, project, project, goalsSection)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Sprint review for project: %s", project),
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: promptText},
			},
		},
	}, nil
}

func handleMemoryConsolidationPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	project := req.Params.Arguments["project"]
	focusArea := req.Params.Arguments["focus_area"]

	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	var focusSection string
	if focusArea != "" {
		focusSection = fmt.Sprintf(`
- Focus specifically on memories related to: "%s"
- Use recall with term="%s" to find relevant memories`, focusArea, focusArea)
	}

	promptText := fmt.Sprintf(`Consolidate and organize memories for project "%s". Follow these steps:

## Step 1: Inventory Current Memories
- Call list_memories with project="%s" to get all memories
- Group them by type (general, decision, bug_fix, preference, pattern, reference, skill)%s

## Step 2: Identify Duplicates
- Look for memories with similar content or overlapping information
- When adding new memories, use dedup_mode="auto" to prevent future duplicates

## Step 3: Identify Gaps
- Are there important project decisions not captured?
- Are there patterns or preferences that should be documented?
- Are there bug fixes whose lessons should be preserved?

## Step 4: Link Related Memories
- Group related memories into context packs if not already organized
- Use manage_context_pack(action="link_memory") to associate memories with relevant context packs

## Step 5: Create Summary Memories
- For clusters of related memories, create a summary memory that synthesizes the key points
- Use type="reference" for summary memories

## Output Format
Provide a report with:
- Total memories reviewed
- Duplicates identified (and merged if using dedup_mode="auto")
- Gaps identified with suggestions
- Organization improvements made`,
		project, project, focusSection)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Memory consolidation for project: %s", project),
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: promptText},
			},
		},
	}, nil
}

func handleDailyStandupPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	project := req.Params.Arguments["project"]

	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	promptText := fmt.Sprintf(`Conduct a daily standup for project "%s". Gather and present the following information:

## Step 1: What was completed recently?
- Call list_tasks with project="%s" and status="COMPLETED"
- Focus on tasks completed in the last 24-48 hours (check updated_at timestamps)

## Step 2: What is currently in progress?
- Call list_tasks with project="%s" and status="IN_PROGRESS"
- For each in-progress task, note its current progress percentage

## Step 3: What are the blockers?
- Check for tasks with dependencies using get_task_dependencies
- Identify any tasks that have been in progress for too long
- Look for tasks that depend on completed prerequisites

## Step 4: What should be worked on next?
- Call get_next_tasks with project="%s" to get prioritized recommendations
- Highlight the top 3 priorities

## Step 5: Recent Activity
- Call get_agent_activity with project="%s" to see recent changes
- Summarize notable activity (new memories, decisions, task updates)

## Output Format
Present a concise standup report:
- **Done**: Recent completions (bullet list)
- **In Progress**: Current work with progress (bullet list)
- **Blocked**: Issues needing attention (bullet list)
- **Next Up**: Top 3 priorities (numbered list)
- **Activity**: Notable recent changes (brief)`,
		project, project, project, project, project)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Daily standup for project: %s", project),
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: promptText},
			},
		},
	}, nil
}
