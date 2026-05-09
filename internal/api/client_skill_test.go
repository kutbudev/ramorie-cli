package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kutbudev/ramorie-cli/internal/models"
)

// TestLoadSkill_UUIDPath_HitsRenderEndpointDirectly verifies that when
// the caller passes a full UUID we skip the search resolver and go
// straight to GET /memories/{id}/skill-render. This is the hot path
// for MCP callers — agents typically already have the id from a prior
// find/recall call.
func TestLoadSkill_UUIDPath_HitsRenderEndpointDirectly(t *testing.T) {
	skillID := uuid.New().String()

	var (
		renderHits int
		searchHits int
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/memories/"+skillID+"/skill-render" && r.Method == http.MethodGet:
			renderHits++
			resp := SkillRenderResponse{
				Skill: SkillRenderHeader{
					ID:          skillID,
					Name:        "deploy-prod",
					Description: "Ship to production safely",
					Version:     "1.0.0",
					Trigger:     "Before pushing to main",
					StepsCount:  5,
				},
				Body: "---\nname: deploy-prod\n---\n\n# Deploy Prod\n\n## Overview\n…",
				Source: SkillRenderSource{},
				Meta: map[string]interface{}{
					"tokens":    float64(1234),
					"encrypted": false,
				},
			}
			b, _ := json.Marshal(resp)
			_, _ = w.Write(b)
		case strings.HasPrefix(r.URL.Path, "/memories"):
			// Listed on purpose so the test fails loudly if the resolver
			// is invoked when it shouldn't be.
			searchHits++
			_, _ = w.Write([]byte(`{"memories":[],"total":0,"limit":5,"offset":0}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c := newTestClient(ts)
	resp, err := c.LoadSkill(skillID)
	if err != nil {
		t.Fatalf("LoadSkill: %v", err)
	}
	if renderHits != 1 {
		t.Errorf("render endpoint should be hit exactly once, got %d", renderHits)
	}
	if searchHits != 0 {
		t.Errorf("UUID input must skip name resolver, but search hit %d times", searchHits)
	}
	if resp.Skill.Name != "deploy-prod" {
		t.Errorf("skill.name round-trip failed: got %q", resp.Skill.Name)
	}
	if !strings.Contains(resp.Body, "# Deploy Prod") {
		t.Errorf("body should contain rendered markdown, got %q", resp.Body)
	}
	if resp.Skill.StepsCount != 5 {
		t.Errorf("steps_count round-trip failed: got %d", resp.Skill.StepsCount)
	}
}

// TestLoadSkill_NameResolution_SingleMatch covers the canonical happy
// path for human-friendly invocation: `ramorie skill use deploy-prod`.
// We expect /memories?search= → narrow to type=skill → GET render.
func TestLoadSkill_NameResolution_SingleMatch(t *testing.T) {
	skillID := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/memories" && r.Method == http.MethodGet:
			// Search response: include one skill + one non-skill so we
			// also verify the client-side type filter.
			resp := MemoriesListResponse{
				Memories: []models.Memory{
					{ID: skillID, Type: "skill", Content: "deploy-prod skill body"},
					{ID: uuid.New(), Type: "general", Content: "unrelated note"},
				},
				Total: 2, Limit: 5, Offset: 0,
			}
			b, _ := json.Marshal(resp)
			_, _ = w.Write(b)

		case r.URL.Path == "/memories/"+skillID.String()+"/skill-render":
			resp := SkillRenderResponse{
				Skill: SkillRenderHeader{ID: skillID.String(), Name: "deploy-prod", Version: "1.0.0", StepsCount: 3},
				Body:  "---\nname: deploy-prod\n---\n\nbody",
				Meta:  map[string]interface{}{"tokens": float64(42)},
			}
			b, _ := json.Marshal(resp)
			_, _ = w.Write(b)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c := newTestClient(ts)
	resp, err := c.LoadSkill("deploy-prod")
	if err != nil {
		t.Fatalf("LoadSkill: %v", err)
	}
	if resp.Skill.ID != skillID.String() {
		t.Errorf("resolver must pick the type=skill row, got id=%s", resp.Skill.ID)
	}
}

// TestLoadSkill_NameResolution_AmbiguousErrors guards the
// disambiguation contract: when multiple skills match, the client
// refuses rather than silently picking one. The CLI surfaces this
// error directly so the user can pass a UUID instead.
func TestLoadSkill_NameResolution_AmbiguousErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Two skill-typed candidates — must trigger ambiguity error.
		resp := MemoriesListResponse{
			Memories: []models.Memory{
				{ID: uuid.New(), Type: "skill", Content: "deploy-prod"},
				{ID: uuid.New(), Type: "skill", Content: "deploy-prod-staging"},
			},
			Total: 2, Limit: 5, Offset: 0,
		}
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.LoadSkill("deploy")
	if err == nil {
		t.Fatal("expected ambiguity error when >1 skill candidates match")
	}
	if !strings.Contains(err.Error(), "multiple skills match") {
		t.Errorf("error should explain ambiguity, got %v", err)
	}
}

// TestLoadSkill_NameResolution_NoMatchErrors covers the empty-result
// case (no skill memory matches the slug). Surfaces as a clear "no
// skill matches" so users don't blame the CLI for swallowing results.
func TestLoadSkill_NameResolution_NoMatchErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Search returns hits but none are type=skill — must still error.
		resp := MemoriesListResponse{
			Memories: []models.Memory{
				{ID: uuid.New(), Type: "general", Content: "not a skill"},
			},
			Total: 1, Limit: 5, Offset: 0,
		}
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.LoadSkill("nonexistent-skill")
	if err == nil {
		t.Fatal("expected error when no skill candidates match")
	}
	if !strings.Contains(err.Error(), "no skill memory matches") {
		t.Errorf("error should say no match, got %v", err)
	}
}

// TestLoadSkill_EmptyInputErrors guards the trivial input validation —
// keeps the failure mode out of the network layer so callers get a
// fast, descriptive error.
func TestLoadSkill_EmptyInputErrors(t *testing.T) {
	c := &Client{BaseURL: "http://unused", APIKey: "k", HTTPClient: http.DefaultClient}
	if _, err := c.LoadSkill("   "); err == nil {
		t.Fatal("expected error for empty skill_id")
	}
}
