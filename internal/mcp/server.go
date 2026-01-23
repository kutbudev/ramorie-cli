package mcp

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/config"
)

// apiClient holds the API client for tool handlers
var apiClient *api.Client

// ServeStdio starts the MCP server using the official go-sdk over stdio
func ServeStdio(client *api.Client) error {
	if client == nil {
		return errors.New("api client is required")
	}
	apiClient = client

	// Create server with implementation info
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "ramorie",
			Version: "2.1.0",
		},
		nil,
	)

	// Register all tools
	registerTools(server)

	// Run server over stdio
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// wrapResultAsObject ensures the result is always an object (not array or null)
// This fixes the MCP "expected record, received array" error
func wrapResultAsObject(result interface{}) map[string]interface{} {
	if result == nil {
		return map[string]interface{}{"data": nil}
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

// getContextString returns current workspace context for response metadata
func getContextString() string {
	// Use session context if available, fallback to config
	if IsSessionInitialized() {
		return GetSessionContext()
	}

	// Fallback for non-initialized sessions
	cfg, err := config.LoadConfig()
	if err != nil || cfg == nil {
		return "Personal Workspace (session not initialized)"
	}
	if cfg.ActiveProjectID != "" {
		return "Project: " + cfg.ActiveProjectID[:8] + "... (session not initialized)"
	}
	return "Personal Workspace (session not initialized)"
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

// checkProjectRequired checks if tool requires active project and returns error if not set
func checkProjectRequired(toolName string) error {
	if !RequiresProject(toolName) {
		return nil
	}
	if !RequiresActiveProject() {
		return errors.New("⚠️ No active project set. Please specify the 'project' parameter in your tool call to indicate which project to work in. This ensures your tasks and memories are organized correctly.")
	}
	return nil
}
