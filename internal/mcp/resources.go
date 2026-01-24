package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerResources adds MCP resource templates to the server
func registerResources(server *mcp.Server) {
	// Static resource: list all projects
	server.AddResource(&mcp.Resource{
		URI:         "ramorie://projects",
		Name:        "projects",
		Description: "List of all projects in the current organization",
		MIMEType:    "application/json",
	}, handleProjectsResource)

	// Template: single project details
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "ramorie://projects/{id}",
		Name:        "project",
		Description: "Detailed information about a specific project",
		MIMEType:    "application/json",
	}, handleProjectResource)

	// Template: tasks in a project
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "ramorie://projects/{id}/tasks",
		Name:        "project-tasks",
		Description: "All tasks in a specific project",
		MIMEType:    "application/json",
	}, handleProjectTasksResource)

	// Template: memories in a project
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "ramorie://projects/{id}/memories",
		Name:        "project-memories",
		Description: "All memories in a specific project",
		MIMEType:    "application/json",
	}, handleProjectMemoriesResource)

	// Template: single task details
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "ramorie://tasks/{id}",
		Name:        "task",
		Description: "Detailed task information including notes and subtasks",
		MIMEType:    "application/json",
	}, handleTaskResource)

	// Template: single memory content
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "ramorie://memories/{id}",
		Name:        "memory",
		Description: "Memory content and metadata",
		MIMEType:    "application/json",
	}, handleMemoryResource)

	// Template: context pack with contents
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "ramorie://context-packs/{id}",
		Name:        "context-pack",
		Description: "Context pack details with linked memories and tasks",
		MIMEType:    "application/json",
	}, handleContextPackResource)
}

// extractIDFromURI extracts the {id} portion from a resource URI
func extractIDFromURI(uri, prefix, suffix string) string {
	s := strings.TrimPrefix(uri, prefix)
	if suffix != "" {
		s = strings.TrimSuffix(s, suffix)
	}
	return s
}

func handleProjectsResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	var orgID string
	session := GetCurrentSession()
	if session != nil && session.ActiveOrgID != nil {
		orgID = session.ActiveOrgID.String()
	}

	projects, err := apiClient.ListProjects(orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal projects: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func handleProjectResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id := extractIDFromURI(req.Params.URI, "ramorie://projects/", "")
	if id == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	// Use ListProjects and find matching - API doesn't have GetProject
	var orgID string
	session := GetCurrentSession()
	if session != nil && session.ActiveOrgID != nil {
		orgID = session.ActiveOrgID.String()
	}

	projects, err := apiClient.ListProjects(orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	for _, p := range projects {
		if p.ID.String() == id || p.Name == id {
			data, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal project: %w", err)
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(data),
				}},
			}, nil
		}
	}

	return nil, mcp.ResourceNotFoundError(req.Params.URI)
}

func handleProjectTasksResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id := extractIDFromURI(req.Params.URI, "ramorie://projects/", "/tasks")
	if id == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	tasks, err := apiClient.ListTasks(id, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tasks: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func handleProjectMemoriesResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id := extractIDFromURI(req.Params.URI, "ramorie://projects/", "/memories")
	if id == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	memories, err := apiClient.ListMemories(id, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}

	data, err := json.MarshalIndent(memories, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal memories: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func handleTaskResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id := extractIDFromURI(req.Params.URI, "ramorie://tasks/", "")
	if id == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	task, err := apiClient.GetTask(id)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func handleMemoryResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id := extractIDFromURI(req.Params.URI, "ramorie://memories/", "")
	if id == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	memory, err := apiClient.GetMemory(id)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	data, err := json.MarshalIndent(memory, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal memory: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func handleContextPackResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id := extractIDFromURI(req.Params.URI, "ramorie://context-packs/", "")
	if id == "" {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	pack, err := apiClient.GetContextPack(id)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	data, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context pack: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}
