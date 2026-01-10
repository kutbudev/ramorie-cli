package mcp

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents an MCP agent session with context
type Session struct {
	ID              string
	Initialized     bool
	AgentName       string
	AgentModel      string
	ActiveProjectID *uuid.UUID
	ActiveOrgID     *uuid.UUID
	CreatedAt       time.Time
	LastActivityAt  time.Time
}

// SessionManager manages MCP sessions
// For now, we use a single global session since MCP stdio is typically one connection
// This will be enhanced in Phase 4 for multi-agent support
type SessionManager struct {
	currentSession *Session
	mu             sync.RWMutex
}

var sessionManager = &SessionManager{}

// GetCurrentSession returns the current session, creating one if needed
func GetCurrentSession() *Session {
	sessionManager.mu.Lock()
	defer sessionManager.mu.Unlock()

	if sessionManager.currentSession == nil {
		sessionManager.currentSession = &Session{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
		}
	}
	sessionManager.currentSession.LastActivityAt = time.Now()
	return sessionManager.currentSession
}

// InitializeSession marks the session as initialized with agent info
func InitializeSession(agentName, agentModel string) *Session {
	sessionManager.mu.Lock()
	defer sessionManager.mu.Unlock()

	if sessionManager.currentSession == nil {
		sessionManager.currentSession = &Session{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
		}
	}

	sessionManager.currentSession.Initialized = true
	sessionManager.currentSession.AgentName = agentName
	sessionManager.currentSession.AgentModel = agentModel
	sessionManager.currentSession.LastActivityAt = time.Now()

	return sessionManager.currentSession
}

// SetSessionProject sets the active project for the session
func SetSessionProject(projectID uuid.UUID) {
	sessionManager.mu.Lock()
	defer sessionManager.mu.Unlock()

	if sessionManager.currentSession != nil {
		sessionManager.currentSession.ActiveProjectID = &projectID
		sessionManager.currentSession.LastActivityAt = time.Now()
	}
}

// SetSessionOrganization sets the active organization for the session
func SetSessionOrganization(orgID uuid.UUID) {
	sessionManager.mu.Lock()
	defer sessionManager.mu.Unlock()

	if sessionManager.currentSession != nil {
		sessionManager.currentSession.ActiveOrgID = &orgID
		sessionManager.currentSession.LastActivityAt = time.Now()
	}
}

// IsSessionInitialized checks if the session has been properly initialized
func IsSessionInitialized() bool {
	sessionManager.mu.RLock()
	defer sessionManager.mu.RUnlock()

	return sessionManager.currentSession != nil && sessionManager.currentSession.Initialized
}

// RequiresActiveProject checks if the session has an active project set
func RequiresActiveProject() bool {
	sessionManager.mu.RLock()
	defer sessionManager.mu.RUnlock()

	return sessionManager.currentSession != nil && sessionManager.currentSession.ActiveProjectID != nil
}

// GetSessionContext returns a context string for response metadata
func GetSessionContext() string {
	sessionManager.mu.RLock()
	defer sessionManager.mu.RUnlock()

	if sessionManager.currentSession == nil {
		return "Session not initialized"
	}

	s := sessionManager.currentSession
	if !s.Initialized {
		return "Session not initialized - call setup_agent first"
	}

	context := "Agent: " + s.AgentName
	if s.AgentModel != "" {
		context += " (" + s.AgentModel + ")"
	}

	if s.ActiveProjectID != nil {
		context += " | Project: " + s.ActiveProjectID.String()[:8] + "..."
	} else {
		context += " | No active project"
	}

	return context
}

// ResetSession clears the current session (useful for testing)
func ResetSession() {
	sessionManager.mu.Lock()
	defer sessionManager.mu.Unlock()

	sessionManager.currentSession = nil
}

// AllowedWithoutInit returns true if the tool can be called without initialization
func AllowedWithoutInit(toolName string) bool {
	// These tools are allowed without initialization
	// They are part of the setup workflow: get_ramorie_info -> setup_agent -> list_projects -> set_active_project
	allowedTools := map[string]bool{
		"get_ramorie_info":   true, // Info tool, always available
		"setup_agent":        true, // This is the initialization tool itself
		"list_projects":      true, // Need to see projects to set active project
		"set_active_project": true, // Part of setup workflow, sets context
	}
	return allowedTools[toolName]
}

// RequiresProject returns true if the tool requires an active project to be set
func RequiresProject(toolName string) bool {
	// These tools require an active project
	projectTools := map[string]bool{
		"create_task":     true,
		"add_memory":      true,
		"add_task_note":   true,
		"update_progress": true,
	}
	return projectTools[toolName]
}
