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

// TestHandleDecision_EmitsDeprecationLog verifies that calling the decision tool:
//  1. Writes the [DEPRECATED] line to stderr.
//  2. Delegates to handleUnifiedMemory and returns a valid result.
func TestHandleDecision_EmitsDeprecationLog(t *testing.T) {
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

	// Call the deprecated decision tool with a simple list action.
	res, _, callErr := handleUnifiedDecision(context.Background(), nil, UnifiedDecisionInput{
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
	if !strings.Contains(stderrOutput, "decision") {
		t.Errorf("expected 'decision' mentioned in deprecation stderr; got: %q", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "memory") {
		t.Errorf("expected 'memory' mentioned in deprecation stderr; got: %q", stderrOutput)
	}

	// Assert delegation to handleUnifiedMemory succeeded (not a nil/error result).
	if callErr != nil {
		t.Errorf("handleUnifiedDecision returned error: %v", callErr)
	}
	if res == nil {
		t.Error("handleUnifiedDecision returned nil result")
	}
}

// TestTranslateDecisionInputToMemoryInput checks the field mapping between
// UnifiedDecisionInput and UnifiedMemoryInput for the supported action types.
func TestTranslateDecisionInputToMemoryInput(t *testing.T) {
	t.Run("list action", func(t *testing.T) {
		in := UnifiedDecisionInput{Action: "list", Project: "my-proj", Limit: 10}
		out := translateDecisionInputToMemoryInput(in)
		if out.Action != "list" {
			t.Errorf("action: want list, got %q", out.Action)
		}
		if out.Project != "my-proj" {
			t.Errorf("project: want my-proj, got %q", out.Project)
		}
		if out.Limit != 10 {
			t.Errorf("limit: want 10, got %v", out.Limit)
		}
		if out.Type != "decision" {
			t.Errorf("type: want decision, got %q", out.Type)
		}
	})

	t.Run("create with title only", func(t *testing.T) {
		in := UnifiedDecisionInput{Action: "create", Project: "p", Title: "Use PostgreSQL"}
		out := translateDecisionInputToMemoryInput(in)
		if out.Action != "create" {
			t.Errorf("action: want create, got %q", out.Action)
		}
		if out.Type != "decision" {
			t.Errorf("type: want decision, got %q", out.Type)
		}
		if !strings.Contains(out.Content, "Use PostgreSQL") {
			t.Errorf("content should contain title; got %q", out.Content)
		}
	})

	t.Run("create with all ADR fields", func(t *testing.T) {
		in := UnifiedDecisionInput{
			Action:       "create",
			Project:      "p",
			Title:        "Adopt event sourcing",
			Description:  "Use Kafka for async events",
			Status:       "proposed",
			Area:         "Architecture",
			Context:      "Need to decouple services",
			Consequences: "Higher operational complexity",
		}
		out := translateDecisionInputToMemoryInput(in)
		if out.Action != "create" {
			t.Errorf("action: want create, got %q", out.Action)
		}
		if out.Type != "decision" {
			t.Errorf("type: want decision, got %q", out.Type)
		}
		if !strings.Contains(out.Content, "Adopt event sourcing") {
			t.Errorf("content should contain title; got %q", out.Content)
		}
		if !strings.Contains(out.Content, "Status: proposed") {
			t.Errorf("content should contain Status; got %q", out.Content)
		}
		if !strings.Contains(out.Content, "Area: Architecture") {
			t.Errorf("content should contain Area; got %q", out.Content)
		}
		if !strings.Contains(out.Content, "Context: Need to decouple services") {
			t.Errorf("content should contain Context; got %q", out.Content)
		}
		if !strings.Contains(out.Content, "Consequences: Higher operational complexity") {
			t.Errorf("content should contain Consequences; got %q", out.Content)
		}
	})

	t.Run("empty action falls back to list", func(t *testing.T) {
		in := UnifiedDecisionInput{Action: "", Project: "p"}
		out := translateDecisionInputToMemoryInput(in)
		if out.Action != "list" {
			t.Errorf("empty action should fall back to list; got %q", out.Action)
		}
	})
}
