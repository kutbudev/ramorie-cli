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
			Version: "2.4.0",
		},
		&mcp.ServerOptions{
			CompletionHandler: completionHandler,
			Instructions: `🧠 RAMORIE - Persistent Memory for AI Agents

Memories persist across sessions and are shared with other agents. Use
Ramorie proactively — don't wait to be asked.

## 3 Rules

1. FIND FIRST — call find(term, [project]) BEFORE any response. Hybrid
   (semantic + lexical + recency + usage) ranking. Auto-scopes to cwd
   project via X-Project-Hint header. Default response fits ≤2000 tokens.
   ` + "`recall`" + ` is the legacy lexical-only alternative.

2. REMEMBER ALWAYS — remember(content, project) on every preference,
   bug fix, decision, or pattern. Don't ask permission.

3. TASK EVERYTHING — task(action=create, project, description) when the
   user says "do X", "fix Y", "later", or defers work.

## Session start

- setup_agent returns a compact session payload by default (~500 token).
  Pass full:true only when you specifically want recent_memories,
  workflow_pattern, recommended_actions.
- list_projects returns [{id, name, org}] by default. Pass verbose:true
  for full nested metadata.

## Auto-surfacing (optional)

If you install the Claude Code hook (` + "`ramorie hook install`" + `), the
system calls find-related for every Edit/Write/Read and injects a short
summary as a system-reminder — so you see related memories without
manually calling find.

⚠️ DO NOT ask "should I save this?" — just save it.
⚠️ DO NOT forget to find() before answering.`,
		},
	)

	// Register all tools, resources, and prompts
	// v4: Simplified from 49 tools to 15 action-based tools
	registerToolsV4(server)
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

