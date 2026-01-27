package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// apiClient holds the API client for tool handlers
var apiClient *api.Client

// ServeStdio starts the MCP server using the official go-sdk over stdio
func ServeStdio(client *api.Client) error {
	if client == nil {
		return errors.New("api client is required")
	}
	apiClient = client

	// Try to load a persisted session from a previous MCP run
	// This helps maintain org context across stdio restarts
	if LoadPersistedSession() {
		// Restore agent info on the API client
		setAgentInfoFromSession(client)
	}

	// Create server with implementation info and instructions
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "ramorie",
			Version: "2.2.0",
		},
		&mcp.ServerOptions{
			CompletionHandler: completionHandler,
			Instructions: `🧠 RAMORIE - Your Persistent Memory System

You are connected to Ramorie, a persistent AI memory system. This connection means:
- Your memories persist across sessions and are shared with other AI agents
- You MUST use Ramorie proactively - don't wait for user to ask

## CRITICAL: Proactive Usage Rules

### ALWAYS RECALL FIRST
Before answering ANY question or starting ANY task:
→ recall(term: "relevant_keywords") to check existing knowledge

### AUTO-SAVE These Situations (No User Permission Needed)
- User states a preference → remember immediately
- Bug is solved → remember with problem + solution
- Decision is made → remember with rationale
- Pattern is discovered → remember for future reference
- Important context learned → remember to persist it

### AUTO-CREATE Tasks
- User says "do X later" → create_task
- TODO/FIXME found in code → create_task
- Work is deferred → create_task

## Getting Started
1. Call setup_agent first to initialize your session
2. Call list_projects to see ALL your accessible projects (personal + org)
3. Pass 'project' parameter in create_task/remember calls

## Quick Reference
- SAVE: remember(project: "name", content: "...", type: "decision")
- FIND: recall(term: "keywords")
- TASK: create_task(project: "name", description: "...")
- LIST: list_projects() shows ALL accessible projects across orgs

## Multi-Agent Awareness
Other AI agents can see your memories instantly. Always:
- Recall before acting (another agent may have added context)
- Save useful discoveries (helps other agents too)
- Check for duplicates before saving (use recall first)

## Best Practices
- Start tasks (manage_task action=start) before working on them
- Use 'type' parameter in remember: general, decision, bug_fix, preference, pattern, reference, skill
- Context packs bundle related memories and tasks into workspaces

⚠️ DO NOT ask user "should I save this?" - just save it.
⚠️ DO NOT forget to recall before answering questions.`,
		},
	)

	// Register all tools, resources, and prompts
	registerTools(server)
	registerResources(server)
	registerPrompts(server)

	// Run server over stdio
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// wrapResultAsObject ensures the result is always an object (not array or null)
// This fixes the MCP "expected record, received array" error
func wrapResultAsObject(result interface{}) map[string]interface{} {
	if result == nil {
		return map[string]interface{}{"items": []interface{}{}, "count": 0, "message": "No results"}
	}

	switch v := result.(type) {
	case []interface{}:
		return map[string]interface{}{"items": v, "count": len(v)}
	case map[string]interface{}:
		return v
	default:
		b, err := json.Marshal(result)
		if err != nil {
			return map[string]interface{}{"data": result}
		}

		if len(b) > 0 && b[0] == '[' {
			var arr []interface{}
			if err := json.Unmarshal(b, &arr); err == nil {
				return map[string]interface{}{"items": arr, "count": len(arr)}
			}
		}

		if len(b) > 0 && b[0] == '{' {
			var obj map[string]interface{}
			if err := json.Unmarshal(b, &obj); err == nil {
				return obj
			}
		}

		return map[string]interface{}{"data": result}
	}
}

// formatMCPResponse creates a proper MCP spec compliant response
// This ensures all responses are objects with context metadata
func formatMCPResponse(data interface{}, contextInfo string) map[string]interface{} {
	wrapped := wrapResultAsObject(data)

	// Add context metadata to help agents understand where they are
	if contextInfo != "" {
		wrapped["_context"] = contextInfo
	}

	return wrapped
}

// formatPaginatedResponse wraps data with pagination metadata
func formatPaginatedResponse(data interface{}, nextCursor string, total int, contextInfo string) map[string]interface{} {
	wrapped := wrapResultAsObject(data)

	if contextInfo != "" {
		wrapped["_context"] = contextInfo
	}

	wrapped["total"] = total
	if nextCursor != "" {
		wrapped["nextCursor"] = nextCursor
	}

	return wrapped
}

// getContextString returns current workspace context for response metadata
func getContextString() string {
	// Use session context if available, fallback to config
	if IsSessionInitialized() {
		return GetSessionContext()
	}

	return "Personal Workspace (session not initialized)"
}

// textResult converts any data to a CallToolResult with JSON TextContent.
// This ensures data goes into Content (not StructuredContent), which is
// compatible with both Claude Code and Claude Desktop.
func textResult(data interface{}) (*mcp.CallToolResult, error) {
	if data == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "{}"},
			},
		}, nil
	}
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonBytes)},
		},
	}, nil
}

// mustTextResult is like textResult but returns an error result instead of failing.
// Use this in handler return statements for concise one-liner conversions.
func mustTextResult(data interface{}) *mcp.CallToolResult {
	res, err := textResult(data)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf(`{"error": "%s"}`, err.Error())},
			},
			IsError: true,
		}
	}
	return res
}

// checkSessionInit checks if session is initialized and returns error if not
// Returns nil if allowed, or error response if not initialized
func checkSessionInit(toolName string) error {
	if AllowedWithoutInit(toolName) {
		return nil
	}
	if !IsSessionInitialized() {
		return errors.New("⚠️ Session not initialized. Please call 'setup_agent' first to initialize your session. This helps track which agent is making changes and ensures proper context.")
	}
	return nil
}

