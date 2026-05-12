package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/kutbudev/ramorie-cli/internal/models"
)

func TestListProjectsDecodesSupportedResponseShapes(t *testing.T) {
	projectID := uuid.New()
	project := models.Project{ID: projectID, Name: "workflow"}

	cases := []struct {
		name    string
		payload any
	}{
		{"raw array", []models.Project{project}},
		{"projects envelope", map[string]any{"projects": []models.Project{project}, "total": 1}},
		{"data array envelope", map[string]any{"success": true, "data": []models.Project{project}}},
		{"data projects envelope", map[string]any{"data": map[string]any{"projects": []models.Project{project}, "total": 1}}},
		{"items envelope", map[string]any{"items": []models.Project{project}, "count": 1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/projects" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tc.payload)
			}))
			defer ts.Close()

			c := &Client{BaseURL: ts.URL, APIKey: "test", HTTPClient: ts.Client()}
			got, err := c.ListProjects()
			if err != nil {
				t.Fatalf("ListProjects returned error: %v", err)
			}
			if len(got) != 1 || got[0].ID != projectID || got[0].Name != "workflow" {
				t.Fatalf("ListProjects decoded %+v, want %s workflow", got, projectID)
			}
		})
	}
}

func TestListMemoriesPageSendsOffsetAndUsesTotalForHasMore(t *testing.T) {
	var saw bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/memories" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("page") != "3" || q.Get("page_size") != "25" || q.Get("limit") != "25" || q.Get("offset") != "50" {
			t.Fatalf("unexpected pagination query: %s", r.URL.RawQuery)
		}
		saw = true
		memories := make([]models.Memory, 10)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": memories,
			"total":    60,
			"limit":    25,
			"offset":   50,
		})
	}))
	defer ts.Close()

	c := &Client{BaseURL: ts.URL, APIKey: "test", HTTPClient: ts.Client()}
	items, hasMore, err := c.ListMemoriesPage("project-id", "", 3, 25)
	if err != nil {
		t.Fatalf("ListMemoriesPage returned error: %v", err)
	}
	if !saw {
		t.Fatalf("test server did not receive request")
	}
	if len(items) != 10 {
		t.Fatalf("items len = %d, want 10", len(items))
	}
	if hasMore {
		t.Fatalf("hasMore should use total+offset and be false on the final partial page")
	}
}
