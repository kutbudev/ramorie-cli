package models

import "time"

// CommentAuthor represents the author of a comment.
type CommentAuthor struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// MentionInfo describes a user mention inside a comment.
type MentionInfo struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
}

// Comment is a generic comment attached to either a task or a memory.
// Backend route: GET /comments?entity_type=...&entity_id=...
//
// Mirrors the frontend Comment shape (src/types/comment.ts).
type Comment struct {
	ID         string  `json:"id"`
	EntityType string  `json:"entity_type"`
	EntityID   string  `json:"entity_id"`
	ParentID   *string `json:"parent_id,omitempty"`
	Content    string  `json:"content"`

	Mentions []MentionInfo `json:"mentions,omitempty"`

	IsEdited bool       `json:"is_edited"`
	EditedAt *time.Time `json:"edited_at,omitempty"`

	Author          *CommentAuthor `json:"author,omitempty"`
	CreatedByAgent  string         `json:"created_by_agent,omitempty"`
	AgentModel      string         `json:"agent_model,omitempty"`
	CreatedVia      string         `json:"created_via,omitempty"`

	IsEncrypted      bool   `json:"is_encrypted"`
	EncryptedContent string `json:"encrypted_content,omitempty"`
	ContentNonce     string `json:"content_nonce,omitempty"`

	Replies    []Comment `json:"replies,omitempty"`
	ReplyCount int       `json:"reply_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CommentListResponse is the wire envelope returned by GET /comments.
type CommentListResponse struct {
	Comments []Comment `json:"comments"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
	HasMore  bool      `json:"has_more"`
}
