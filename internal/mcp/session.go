package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/internal/api"
)

// Session represents an MCP agent session with context
type Session struct {
	ID                string
	Initialized       bool
	AgentName         string
	AgentModel        string
	ActiveOrgID       *uuid.UUID
	LastUsedProjectID *uuid.UUID // Track last used project for auto-detection
	CreatedAt         time.Time
	LastActivityAt    time.Time
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

	// Persist session for recovery across MCP restarts
	go PersistSession()

	return sessionManager.currentSession
}

// SetSessionOrganization sets the active organization for the session
func SetSessionOrganization(orgID uuid.UUID) {
	sessionManager.mu.Lock()
	defer sessionManager.mu.Unlock()

	if sessionManager.currentSession != nil {
		sessionManager.currentSession.ActiveOrgID = &orgID
		sessionManager.currentSession.LastActivityAt = time.Now()

		// Persist session for recovery across MCP restarts
		go PersistSession()
	}
}

// IsSessionInitialized checks if the session has been properly initialized
func IsSessionInitialized() bool {
	sessionManager.mu.RLock()
	defer sessionManager.mu.RUnlock()

	return sessionManager.currentSession != nil && sessionManager.currentSession.Initialized
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

	return context
}

// GetSessionActiveOrgID returns the active organization ID from the current session
func GetSessionActiveOrgID() *uuid.UUID {
	sessionManager.mu.RLock()
	defer sessionManager.mu.RUnlock()
	if sessionManager.currentSession != nil {
		return sessionManager.currentSession.ActiveOrgID
	}
	return nil
}

// SetSessionLastProject sets the last used project for the session
func SetSessionLastProject(projectID uuid.UUID) {
	sessionManager.mu.Lock()
	defer sessionManager.mu.Unlock()
	if sessionManager.currentSession != nil {
		sessionManager.currentSession.LastUsedProjectID = &projectID
		sessionManager.currentSession.LastActivityAt = time.Now()
		go PersistSession()
	}
}

// GetSessionLastProjectID returns the last used project ID from the current session
func GetSessionLastProjectID() *uuid.UUID {
	sessionManager.mu.RLock()
	defer sessionManager.mu.RUnlock()
	if sessionManager.currentSession != nil {
		return sessionManager.currentSession.LastUsedProjectID
	}
	return nil
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
	// They are part of the setup workflow: get_ramorie_info -> setup_agent -> list_projects/list_organizations
	allowedTools := map[string]bool{
		"get_ramorie_info":   true, // Info tool, always available
		"setup_agent":        true, // This is the initialization tool itself
		"list_projects":      true, // Need to see projects before creating tasks/memories
		"list_organizations": true, // Need to see orgs before switching
	}
	return allowedTools[toolName]
}

// setAgentInfoFromSession sets agent metadata from the current session onto an API client
func setAgentInfoFromSession(client *api.Client) {
	session := GetCurrentSession()
	if session != nil && session.AgentName != "" {
		client.SetAgentInfo(session.AgentName, session.AgentModel, session.ID)
	}
}

// ============================================================================
// SESSION PERSISTENCE
// Persists session state to disk for MCP stdio restarts
// ============================================================================

// persistedSessionFile is the file name for the persisted session
const persistedSessionFile = "ramorie-mcp-session.json"

// PersistedSession is the JSON structure for saving session to disk
type PersistedSession struct {
	ID                string `json:"id"`
	AgentName         string `json:"agent_name"`
	AgentModel        string `json:"agent_model"`
	ActiveOrgID       string `json:"active_org_id,omitempty"`
	LastUsedProjectID string `json:"last_used_project_id,omitempty"`
	CreatedAt         int64  `json:"created_at"`
	ExpiresAt         int64  `json:"expires_at"`
}

// getSessionFilePath returns the path to the session persistence file
func getSessionFilePath() string {
	tmpDir := os.TempDir()
	return filepath.Join(tmpDir, persistedSessionFile)
}

// PersistSession saves the current session to disk for recovery across MCP restarts
func PersistSession() error {
	sessionManager.mu.RLock()
	defer sessionManager.mu.RUnlock()

	if sessionManager.currentSession == nil {
		return nil
	}

	s := sessionManager.currentSession
	persisted := PersistedSession{
		ID:         s.ID,
		AgentName:  s.AgentName,
		AgentModel: s.AgentModel,
		CreatedAt:  s.CreatedAt.Unix(),
		ExpiresAt:  time.Now().Add(24 * time.Hour).Unix(), // Expire after 24 hours
	}

	if s.ActiveOrgID != nil {
		persisted.ActiveOrgID = s.ActiveOrgID.String()
	}

	if s.LastUsedProjectID != nil {
		persisted.LastUsedProjectID = s.LastUsedProjectID.String()
	}

	data, err := json.Marshal(persisted)
	if err != nil {
		return err
	}

	return os.WriteFile(getSessionFilePath(), data, 0600)
}

// LoadPersistedSession attempts to load a persisted session from disk
// Returns true if a valid session was loaded
func LoadPersistedSession() bool {
	data, err := os.ReadFile(getSessionFilePath())
	if err != nil {
		return false
	}

	var persisted PersistedSession
	if err := json.Unmarshal(data, &persisted); err != nil {
		return false
	}

	// Check if session has expired
	if time.Now().Unix() > persisted.ExpiresAt {
		// Clean up expired session file
		os.Remove(getSessionFilePath())
		return false
	}

	sessionManager.mu.Lock()
	defer sessionManager.mu.Unlock()

	// Restore session
	sessionManager.currentSession = &Session{
		ID:             persisted.ID,
		Initialized:    true, // If persisted, it was initialized
		AgentName:      persisted.AgentName,
		AgentModel:     persisted.AgentModel,
		CreatedAt:      time.Unix(persisted.CreatedAt, 0),
		LastActivityAt: time.Now(),
	}

	if persisted.ActiveOrgID != "" {
		orgID, err := uuid.Parse(persisted.ActiveOrgID)
		if err == nil {
			sessionManager.currentSession.ActiveOrgID = &orgID
		}
	}

	if persisted.LastUsedProjectID != "" {
		projectID, err := uuid.Parse(persisted.LastUsedProjectID)
		if err == nil {
			sessionManager.currentSession.LastUsedProjectID = &projectID
		}
	}

	return true
}

// ClearPersistedSession removes the persisted session file
func ClearPersistedSession() {
	os.Remove(getSessionFilePath())
}

