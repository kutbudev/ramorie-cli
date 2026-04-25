package models

import "time"

// UserProfile is the response of GET /auth/profile.
// Mirrors the frontend UserProfile shape (src/store/api/authApi.ts).
type UserProfile struct {
	ID                   string     `json:"id"`
	Email                string     `json:"email"`
	FirstName            string     `json:"first_name,omitempty"`
	LastName             string     `json:"last_name,omitempty"`
	AvatarURL            string     `json:"avatar_url,omitempty"`
	APIKey               string     `json:"api_key,omitempty"`
	ActiveOrganizationID *string    `json:"active_organization_id,omitempty"`
	CreatedAt            *time.Time `json:"created_at,omitempty"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty"`
}

// AgentProfile is one of the user's registered agent identities
// (claude_code, cursor, windsurf, ...). Returned by GET /agents.
type AgentProfile struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	OrganizationID string     `json:"organization_id,omitempty"`
	AgentName      string     `json:"agent_name"`
	DisplayName    string     `json:"display_name,omitempty"`
	AgentType      string     `json:"agent_type"`
	AgentModel     string     `json:"agent_model,omitempty"`
	Icon           string     `json:"icon,omitempty"`
	Color          string     `json:"color,omitempty"`
	IsActive       bool       `json:"is_active"`
	LastEventAt    *time.Time `json:"last_event_at,omitempty"`
	TotalEvents    int        `json:"total_events"`
	LastSessionID  string     `json:"last_session_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// AgentListResponse is the wire envelope of GET /agents.
type AgentListResponse struct {
	Agents []AgentProfile `json:"agents"`
}

// TopAgent is the per-agent summary inside AgentEventStats.
type TopAgent struct {
	AgentName    string     `json:"agent_name"`
	DisplayName  string     `json:"display_name,omitempty"`
	AgentType    string     `json:"agent_type,omitempty"`
	EventCount   int        `json:"event_count"`
	LastEventAt  *time.Time `json:"last_event_at,omitempty"`
	IsActive     bool       `json:"is_active,omitempty"`
}

// AgentEventStats is the response of GET /agent-events/stats.
// Field names mirror the frontend AgentEventStats shape exactly.
type AgentEventStats struct {
	TotalEvents    int            `json:"total_events"`
	EventsLast24h  int            `json:"events_last_24h"`
	EventsLast7d   int            `json:"events_last_7d"`
	EventsByType   map[string]int `json:"events_by_type,omitempty"`
	EventsByAgent  map[string]int `json:"events_by_agent,omitempty"`
	EventsBySource map[string]int `json:"events_by_source,omitempty"`
	TopAgents      []TopAgent     `json:"top_agents,omitempty"`
}

// OAuthAccount is a linked OAuth provider for the current user
// (GET /auth/oauth/accounts).
type OAuthAccount struct {
	ID          string     `json:"id"`
	Provider    string     `json:"provider"`
	Email       string     `json:"email,omitempty"`
	Username    string     `json:"username,omitempty"`
	DisplayName string     `json:"display_name,omitempty"`
	AvatarURL   string     `json:"avatar_url,omitempty"`
	LinkedAt    *time.Time `json:"linked_at,omitempty"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}
