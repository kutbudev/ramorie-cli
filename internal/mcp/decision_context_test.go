package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

func float64Ptr(v float64) *float64 { return &v }

// TestProjectDecisionContext_UsesServerSideTypeFilter verifies item 9: the
// startup decision injection asks the backend for type=decision rather than
// paging through every memory and filtering client-side.
func TestProjectDecisionContext_UsesServerSideTypeFilter(t *testing.T) {
	projectID := uuid.New()
	var sawType string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		sawType = r.URL.Query().Get("type")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []models.Memory{{
				ID:        uuid.New(),
				ProjectID: projectID,
				Type:      "decision",
				Content:   "Decision: use GORM everywhere",
			}},
			"total":  1,
			"limit":  100,
			"offset": 0,
		})
	}))
	defer ts.Close()

	client := &api.Client{BaseURL: ts.URL, APIKey: "test-key", HTTPClient: ts.Client()}
	got := projectDecisionContext(client, projectID.String(), nil, 8)
	if len(got) != 1 {
		t.Fatalf("want 1 decision, got %d", len(got))
	}
	if sawType != "decision" {
		t.Fatalf("server must receive type=decision filter, got %q", sawType)
	}
}

// TestProjectDecisionContext_RelevanceAwareOrdering verifies item 7: with
// cwd-derived surface terms, a relevant-but-low-importance decision floats above
// a high-importance irrelevant one.
func TestProjectDecisionContext_RelevanceAwareOrdering(t *testing.T) {
	projectID := uuid.New()
	relevantID := uuid.New()
	importantID := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"memories": []models.Memory{
				{
					ID:         importantID,
					ProjectID:  projectID,
					Type:       "decision",
					Content:    "Decision: caching strategy for the billing service",
					Importance: float64Ptr(0.95),
				},
				{
					ID:         relevantID,
					ProjectID:  projectID,
					Type:       "decision",
					Content:    "Decision: mcp tool registration uses AddTool",
					Importance: float64Ptr(0.20),
				},
			},
			"total":  2,
			"limit":  100,
			"offset": 0,
		})
	}))
	defer ts.Close()

	client := &api.Client{BaseURL: ts.URL, APIKey: "test-key", HTTPClient: ts.Client()}

	// Without surface terms: importance wins → important decision first.
	plain := projectDecisionContext(client, projectID.String(), nil, 8)
	if len(plain) != 2 || plain[0]["id"] != importantID.String() {
		t.Fatalf("no-signal ordering must be importance-first; got %+v", plain)
	}

	// With cwd signal "mcp": the relevant low-importance decision floats up.
	scoped := projectDecisionContext(client, projectID.String(), []string{"mcp"}, 8)
	if len(scoped) != 2 || scoped[0]["id"] != relevantID.String() {
		t.Fatalf("relevance-aware ordering must float the mcp decision; got %+v", scoped)
	}
	if scoped[0]["relevance_matches"] != 1 {
		t.Fatalf("relevant decision should report relevance_matches=1; got %v", scoped[0]["relevance_matches"])
	}
}

func TestCwdSurfaceTerms(t *testing.T) {
	cases := []struct {
		cwd  string
		want []string
	}{
		{"/Users/me/Documents/GitHub/ramorie-cli/internal/mcp", []string{"ramorie", "cli", "mcp"}},
		{"", nil},
		{"/usr/local/bin", nil}, // all stopwords / too short
	}
	for _, c := range cases {
		got := cwdSurfaceTerms(c.cwd)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("cwdSurfaceTerms(%q) = %v, want %v", c.cwd, got, c.want)
		}
	}
}
