package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kutbudev/ramorie-cli/internal/api"
)

// withInitializedSession marks the package-level session as
// initialized so handlers gated by checkSessionInit can be exercised
// directly. Saves us from threading setup_agent through every test.
func withInitializedSession(t *testing.T) {
	t.Helper()
	prev := sessionManager.currentSession
	InitializeSession("test-agent", "test-model")
	t.Cleanup(func() {
		sessionManager.mu.Lock()
		sessionManager.currentSession = prev
		sessionManager.mu.Unlock()
	})
}

// extractText pulls the idx-th TextContent payload from a CallToolResult.
// Used for tests that need to inspect each slot of a multi-content
// response — load_skill returns body+envelope in that order.
func extractText(t *testing.T, res *mcp.CallToolResult, idx int) string {
	t.Helper()
	if res == nil {
		t.Fatal("nil CallToolResult")
	}
	if idx < 0 || idx >= len(res.Content) {
		t.Fatalf("content index %d out of range (len=%d)", idx, len(res.Content))
	}
	tc, ok := res.Content[idx].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content[%d] is not *mcp.TextContent: %T", idx, res.Content[idx])
	}
	return tc.Text
}

// TestHandleLoadSkill_HappyPath_ReturnsBodyAndEnvelope is the canonical
// success case: agent calls load_skill with a UUID, server renders,
// and the MCP result hands back two content items — markdown body
// first (so ChatGPT/Claude treat it as inline instruction), JSON
// envelope second (so tooling can introspect skill + meta).
func TestHandleLoadSkill_HappyPath_ReturnsBodyAndEnvelope(t *testing.T) {
	withInitializedSession(t)

	skillID := uuid.New().String()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories/"+skillID+"/skill-render" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		resp := api.SkillRenderResponse{
			Skill: api.SkillRenderHeader{
				ID:          skillID,
				Name:        "deploy-prod",
				Description: "Ship safely",
				Version:     "1.0.0",
				Trigger:     "Before main push",
				StepsCount:  4,
			},
			Body: "---\nname: deploy-prod\n---\n\n# Deploy\n\n## Overview\n…",
			Meta: map[string]interface{}{"tokens": float64(987), "encrypted": false},
		}
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	res, _, err := handleLoadSkill(context.Background(), nil, LoadSkillInput{SkillID: skillID})
	if err != nil {
		t.Fatalf("handleLoadSkill: %v", err)
	}
	if res == nil || len(res.Content) != 2 {
		t.Fatalf("expected 2 content items (body + envelope), got %+v", res)
	}

	// Slot 0: raw markdown body — agents read this as instruction.
	bodyText := extractText(t, res, 0)
	if !strings.Contains(bodyText, "# Deploy") {
		t.Errorf("body slot should carry markdown, got %q", bodyText)
	}
	if strings.HasPrefix(strings.TrimSpace(bodyText), "{") {
		t.Errorf("body slot must not be JSON, got %q", bodyText)
	}

	// Slot 1: structured envelope with skill + source + _meta.
	envelopeText := extractText(t, res, 1)
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(envelopeText), &envelope); err != nil {
		t.Fatalf("envelope slot must be JSON: %v\n%s", err, envelopeText)
	}
	skill, _ := envelope["skill"].(map[string]interface{})
	if skill == nil || skill["name"] != "deploy-prod" {
		t.Errorf("envelope.skill.name missing/wrong: %+v", envelope["skill"])
	}
	if _, hasMeta := envelope["_meta"]; !hasMeta {
		t.Errorf("envelope must include _meta")
	}
	if _, hasSource := envelope["source"]; !hasSource {
		t.Errorf("envelope must include source")
	}
}

// TestHandleLoadSkill_EmptyInputErrors verifies the input-validation
// fast path — no network round-trip when the caller forgets the id.
func TestHandleLoadSkill_EmptyInputErrors(t *testing.T) {
	withInitializedSession(t)

	// Server should never be hit; if it is, the test fails loudly.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected network call for empty input: %s", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	_, _, err := handleLoadSkill(context.Background(), nil, LoadSkillInput{SkillID: "  "})
	if err == nil {
		t.Fatal("expected error for empty skill_id")
	}
	if !strings.Contains(err.Error(), "skill_id is required") {
		t.Errorf("error should explain missing input, got %v", err)
	}
}
