package mcp

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// completionHandler provides autocomplete suggestions for prompt and resource arguments
func completionHandler(_ context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	argName := req.Params.Argument.Name
	argValue := strings.ToLower(req.Params.Argument.Value)

	var values []string

	switch argName {
	case "project", "project_name":
		values = completeProjectNames(argValue)
	case "status":
		values = completeStaticValues(argValue, []string{"TODO", "IN_PROGRESS", "COMPLETED"})
	case "priority":
		values = completeStaticValues(argValue, []string{"L", "M", "H"})
	case "type":
		// Memory types or context pack types
		if req.Params.Ref != nil && req.Params.Ref.Type == "ref/prompt" {
			values = completeStaticValues(argValue, []string{"project", "feature", "sprint", "research"})
		} else {
			values = completeStaticValues(argValue, []string{"general", "decision", "bug_fix", "preference", "pattern", "reference", "skill"})
		}
	case "dedup_mode":
		values = completeStaticValues(argValue, []string{"auto", "off"})
	case "action":
		values = completeStaticValues(argValue, []string{"start", "complete", "stop"})
	case "focus_area":
		// No autocomplete for free text fields
		values = []string{}
	default:
		values = []string{}
	}

	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			Values:  values,
			Total:   len(values),
			HasMore: false,
		},
	}, nil
}

// completeProjectNames fetches project names and filters by prefix
func completeProjectNames(prefix string) []string {
	if apiClient == nil {
		return []string{}
	}

	var orgID string
	session := GetCurrentSession()
	if session != nil && session.ActiveOrgID != nil {
		orgID = session.ActiveOrgID.String()
	}

	projects, err := apiClient.ListProjects(orgID)
	if err != nil {
		return []string{}
	}

	var matches []string
	for _, p := range projects {
		name := p.Name
		if prefix == "" || strings.HasPrefix(strings.ToLower(name), prefix) {
			matches = append(matches, name)
		}
		if len(matches) >= 20 {
			break
		}
	}

	return matches
}

// completeStaticValues filters a static list of values by prefix
func completeStaticValues(prefix string, options []string) []string {
	if prefix == "" {
		return options
	}

	var matches []string
	for _, opt := range options {
		if strings.HasPrefix(strings.ToLower(opt), prefix) {
			matches = append(matches, opt)
		}
	}
	return matches
}
