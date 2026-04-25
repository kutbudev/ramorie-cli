package models

import (
	"time"

	"github.com/google/uuid"
)

// ActivityItem mirrors the backend models.ReportHistoryItem returned by
// GET /reports/history. One entry in the recent-activity feed, regardless
// of whether the underlying entity is a task or a memory.
type ActivityItem struct {
	EntityType string     `json:"entity_type"`
	EntityID   uuid.UUID  `json:"entity_id"`
	ProjectID  *uuid.UUID `json:"project_id,omitempty"`
	Summary    string     `json:"summary"`
	Timestamp  time.Time  `json:"timestamp"`
}
