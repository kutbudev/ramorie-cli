---
name: ramorie-cli-mcp
description: Use when adding new MCP tools to the Ramorie CLI MCP server - provides exact patterns for tool registration, input structs, handler functions, session checks, and response formatting
---

# Ramorie CLI MCP Tool Patterns

## Overview

The MCP server in Ramorie CLI exposes tools that AI agents (Claude, Cursor, etc.) call via stdio transport. Tools are registered with the official `modelcontextprotocol/go-sdk` and follow a consistent input/handler/response pattern.

## MCP Architecture

```
AI Agent (Claude/Cursor) → stdio → MCP Server → Tool Handler → API Client → Backend
                                                      ↓
                                              Session Management
                                                      ↓
                                              Crypto/Vault (decryption)
```

## Tool Registration Pattern

```go
// In registerTools() function in tools.go

mcp.AddTool(server, &mcp.Tool{
	Name:        "feature_action",
	Description: "🟡 COMMON | Action description. REQUIRED: param1. Optional: param2, param3.",
}, handleFeatureAction)
```

### Description Format
- **Tier prefix**: `🔴 ESSENTIAL`, `🟡 COMMON`, or `🟢 ADVANCED`
- **Separator**: ` | `
- **Action**: What the tool does
- **REQUIRED**: List required params
- **Optional**: List optional params with brief explanation

### Tool Tiers
| Tier | When to Use |
|------|-------------|
| ESSENTIAL (7) | Core workflow tools every agent needs |
| COMMON (12) | Frequently used but not strictly required |
| ADVANCED (7) | Power features for complex workflows |

## Input Struct Pattern

```go
// Define input struct with JSON tags matching MCP parameter names
type FeatureActionInput struct {
	FeatureID   string  `json:"featureId"`               // REQUIRED
	Action      string  `json:"action"`                  // REQUIRED: create|update|delete
	Title       string  `json:"title,omitempty"`          // Optional
	Description string  `json:"description,omitempty"`    // Optional
	Force       bool    `json:"force,omitempty"`          // Optional boolean
	Limit       float64 `json:"limit,omitempty"`          // Use float64 for numbers (MCP sends floats)
}
```

**Important**: MCP sends all numbers as `float64`, not `int`. Use `float64` for numeric params and convert inside handler.

## Handler Function Pattern

```go
func handleFeatureAction(ctx context.Context, req *mcp.CallToolRequest, input FeatureActionInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	// 1. Check session initialization (skip for setup_agent only)
	if err := checkSessionInit("feature_action"); err != nil {
		return nil, nil, err
	}

	// 2. Validate required params
	featureID := strings.TrimSpace(input.FeatureID)
	if featureID == "" {
		return nil, nil, errors.New("'featureId' parameter is REQUIRED")
	}

	// 3. Call API client
	feature, err := apiClient.GetFeature(featureID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get feature: %w", err)
	}

	// 4. Handle encryption (if applicable)
	title := feature.Title
	if feature.IsEncrypted {
		title = decryptFeatureTitle(feature)
	}

	// 5. Build response
	result := map[string]interface{}{
		"id":     feature.ID,
		"title":  title,
		"status": feature.Status,
	}

	// 6. Return with context metadata
	return nil, formatMCPResponse(result, "Feature details retrieved"), nil
}
```

## Response Patterns

### Return Types

The handler signature supports two return patterns:

```go
// Pattern 1: Object response (most common)
func handler(...) (*mcp.CallToolResult, map[string]interface{}, error)
// Return: nil, responseMap, nil

// Pattern 2: Interface response (for simple values)
func handler(...) (*mcp.CallToolResult, interface{}, error)
// Return: nil, value, nil
```

### Response Formatting

```go
// Always wrap responses with formatMCPResponse for context metadata
return nil, formatMCPResponse(result, "Context description"), nil

// For lists - wrap in object to comply with MCP spec
items := make([]interface{}, len(features))
for i, f := range features {
    items[i] = map[string]interface{}{
        "id":     f.ID,
        "title":  f.Title,
        "status": f.Status,
    }
}
return nil, formatMCPResponse(items, "Features list"), nil

// For errors
return nil, nil, errors.New("descriptive error message")
// or
return nil, nil, fmt.Errorf("failed to %s: %w", action, err)
```

### wrapResultAsObject Rules
- Arrays → `{"items": [...], "count": N}`
- Maps → passed through directly
- nil → `{"data": null}`
- Other → `{"data": value}`

## Consolidated Tool Pattern (Multi-Action)

For related operations, use a single tool with an `action` parameter:

```go
type ManageFeatureInput struct {
	Action    string `json:"action"`              // REQUIRED: create|update|delete|archive
	FeatureID string `json:"featureId,omitempty"` // Required for update/delete
	Title     string `json:"title,omitempty"`     // Required for create
	Status    string `json:"status,omitempty"`    // Optional for update
}

func handleManageFeature(ctx context.Context, req *mcp.CallToolRequest, input ManageFeatureInput) (*mcp.CallToolResult, map[string]interface{}, error) {
	if err := checkSessionInit("manage_feature"); err != nil {
		return nil, nil, err
	}

	action := strings.TrimSpace(strings.ToLower(input.Action))

	switch action {
	case "create":
		if input.Title == "" {
			return nil, nil, errors.New("'title' is required for create action")
		}
		// ... create logic
		return nil, formatMCPResponse(result, "Feature created"), nil

	case "update":
		if input.FeatureID == "" {
			return nil, nil, errors.New("'featureId' is required for update action")
		}
		// ... update logic
		return nil, formatMCPResponse(result, "Feature updated"), nil

	case "delete":
		if input.FeatureID == "" {
			return nil, nil, errors.New("'featureId' is required for delete action")
		}
		// ... delete logic
		return nil, formatMCPResponse(map[string]interface{}{"deleted": true}, "Feature deleted"), nil

	default:
		return nil, nil, fmt.Errorf("invalid action '%s'. Valid: create, update, delete", action)
	}
}
```

## Project Resolution Pattern

```go
// Resolve project name or short-ID to full UUID
func resolveProjectID(client *api.Client, projectArg string) (string, error) {
	projectArg = strings.TrimSpace(projectArg)
	if projectArg == "" {
		return "", errors.New("project is required")
	}

	projects, err := client.ListProjects()
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	for _, p := range projects {
		if p.ID.String() == projectArg ||
			strings.HasPrefix(p.ID.String(), projectArg) ||
			strings.EqualFold(p.Name, projectArg) {
			return p.ID.String(), nil
		}
	}

	return "", fmt.Errorf("project '%s' not found", projectArg)
}
```

## Encryption Helpers

```go
// Decrypt memory content
func decryptMemoryContent(m *models.Memory) string {
	if !m.IsEncrypted {
		return m.Content
	}
	if !crypto.IsVaultUnlocked() {
		return "[Vault Locked - Unlock to view]"
	}
	plaintext, err := crypto.DecryptContent(m.EncryptedContent, m.ContentNonce, true)
	if err != nil {
		return "[Decryption Failed]"
	}
	return plaintext
}

// Decrypt task fields
func decryptTaskFields(t *models.Task) (title, description string) {
	if !t.IsEncrypted {
		return t.Title, t.Description
	}
	if !crypto.IsVaultUnlocked() {
		return "[Vault Locked]", "[Vault Locked]"
	}
	// ... decrypt each field
}
```

## Session Management

```go
// checkSessionInit ensures setup_agent was called first
func checkSessionInit(toolName string) error {
	if !IsSessionInitialized() {
		return fmt.Errorf("session not initialized. Call 'setup_agent' first before using '%s'", toolName)
	}
	return nil
}

// InitializeSession creates a new agent session
func InitializeSession(agentName, agentModel string) *Session {
	session = &Session{
		ID:          uuid.New().String(),
		AgentName:   agentName,
		AgentModel:  agentModel,
		Initialized: true,
	}
	return session
}
```

## Existing Tools Reference

**ESSENTIAL**: setup_agent, list_projects, list_tasks, create_task, add_memory, recall, manage_focus
**COMMON**: get_task, manage_task, add_task_note, list_memories, get_memory, list_context_packs, get_context_pack, manage_context_pack, create_decision, list_decisions, get_stats, get_agent_activity
**ADVANCED**: create_project, move_task, manage_subtasks, manage_dependencies, manage_plan, list_organizations, switch_organization

## Common Mistakes

- Forgetting `checkSessionInit()` at the start of handler (agents get cryptic errors)
- Using `int` instead of `float64` for numeric input fields (MCP sends floats)
- Returning raw arrays (must wrap via `wrapResultAsObject()`)
- Not calling `strings.TrimSpace()` on string inputs
- Not handling encrypted content (vault locked state)
- Forgetting to register tool in `registerTools()` function
- Not adding API client method in `internal/api/client.go`
- Missing `formatMCPResponse()` wrapper (loses context metadata)
- Using inconsistent action names (always lowercase: create, update, delete)
