package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestHandleSkill_EmitsDeprecationLog verifies that calling the skill tool:
//  1. Writes the [DEPRECATED] line to stderr.
//  2. Delegates to handleUnifiedMemory and returns a valid result.
func TestHandleSkill_EmitsDeprecationLog(t *testing.T) {
	projectID := uuid.New()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/projects":
			projects := []map[string]interface{}{
				{"id": projectID.String(), "name": "TestProject"},
			}
			b, _ := json.Marshal(projects)
			_, _ = w.Write(b)
		case "/memories":
			if r.Method == http.MethodGet {
				resp := map[string]interface{}{
					"memories": []interface{}{},
					"total":    0,
					"limit":    20,
					"offset":   0,
				}
				b, _ := json.Marshal(resp)
				_, _ = w.Write(b)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		default:
			t.Logf("unexpected: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	// Redirect stderr to a pipe to capture the deprecation log.
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = origStderr })

	// Call the deprecated skill tool with a simple list action.
	res, _, callErr := handleUnifiedSkill(context.Background(), nil, UnifiedSkillInput{
		Action:  "list",
		Project: "TestProject",
	})

	// Close write end before reading.
	w.Close()
	os.Stderr = origStderr

	var buf strings.Builder
	tmp := make([]byte, 4096)
	for {
		n, readErr := r.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
		}
		if readErr != nil {
			break
		}
	}
	r.Close()

	stderrOutput := buf.String()

	// Assert deprecation message is present.
	if !strings.Contains(stderrOutput, "[DEPRECATED]") {
		t.Errorf("expected [DEPRECATED] in stderr; got: %q", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "skill") {
		t.Errorf("expected 'skill' mentioned in deprecation stderr; got: %q", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "memory") {
		t.Errorf("expected 'memory' mentioned in deprecation stderr; got: %q", stderrOutput)
	}

	// Assert delegation to handleUnifiedMemory succeeded (not a nil/error result).
	if callErr != nil {
		t.Errorf("handleUnifiedSkill returned error: %v", callErr)
	}
	if res == nil {
		t.Error("handleUnifiedSkill returned nil result")
	}
}

// TestTranslateSkillInputToMemoryInput checks the field mapping between
// UnifiedSkillInput and UnifiedMemoryInput for the supported action types.
func TestTranslateSkillInputToMemoryInput(t *testing.T) {
	t.Run("list action", func(t *testing.T) {
		in := UnifiedSkillInput{Action: "list", Project: "my-proj", Limit: 10}
		out := translateSkillInputToMemoryInput(in)
		if out.Action != "list" {
			t.Errorf("action: want list, got %q", out.Action)
		}
		if out.Project != "my-proj" {
			t.Errorf("project: want my-proj, got %q", out.Project)
		}
		if out.Limit != 10 {
			t.Errorf("limit: want 10, got %v", out.Limit)
		}
	})

	t.Run("generate action maps to goal", func(t *testing.T) {
		in := UnifiedSkillInput{Action: "generate", Description: "Deploy with Railway", Project: "p", AutoSave: true}
		out := translateSkillInputToMemoryInput(in)
		if out.Goal != "Deploy with Railway" {
			t.Errorf("goal: want 'Deploy with Railway', got %q", out.Goal)
		}
		if !out.AutoContext {
			t.Error("auto_context should be true when auto_save=true on generate")
		}
	})

	t.Run("unsupported action falls back to list", func(t *testing.T) {
		in := UnifiedSkillInput{Action: "execute", SkillID: "some-id"}
		out := translateSkillInputToMemoryInput(in)
		if out.Action != "list" {
			t.Errorf("unsupported action should fall back to list; got %q", out.Action)
		}
	})

	t.Run("get action maps skill_id to memoryId", func(t *testing.T) {
		in := UnifiedSkillInput{Action: "get", SkillID: "skill-uuid"}
		out := translateSkillInputToMemoryInput(in)
		if out.Action != "get" {
			t.Errorf("action: want get, got %q", out.Action)
		}
		if out.MemoryID != "skill-uuid" {
			t.Errorf("memoryId: want skill-uuid, got %q", out.MemoryID)
		}
	})
}
