package mcp

// PR10 — tests for auto_remember (atomic find+remember) and the shared
// protocol-reminder helper. The handler covers three observable behaviors:
//   1. Type inference — keyword classifier mirrors handleRemember's logic.
//   2. Duplicate detection — Jaccard similarity ≥ 0.85 short-circuits to
//      matched_existing without creating a new memory.
//   3. No match — fall through to /memories POST and return action=created.
//
// We stub the backend with httptest so the tests stay hermetic; the apiClient
// pointer is swapped in via installTestAPIClient (see handlers_test.go).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestProtocolReminderForOp locks the per-op reminder strings so future edits
// to the table can't silently drift the nudges agents read on every response.
func TestProtocolReminderForOp(t *testing.T) {
	cases := map[string]string{
		"find":                  "If subagent return detected → auto_remember() required next.",
		"remember":              "Saved. Continue without asking user permission for future saves.",
		"auto_remember":         "Saved. Continue without asking user permission for future saves.",
		"auto_remember_created": "Saved. Continue without asking user permission for future saves.",
		"auto_remember_matched": "Memory already exists — protocol satisfied. No save needed.",
		"task":                  "Task tracked. If decision/bug_fix learned from this task → remember().",
		"memory":                "Ramorie protocol: subagent return → find() → auto_remember().",
		"unknown_op":            "Ramorie protocol: subagent return → find() → auto_remember().",
	}
	for op, want := range cases {
		if got := protocolReminderForOp(op); got != want {
			t.Errorf("protocolReminderForOp(%q) = %q, want %q", op, got, want)
		}
	}
}

// TestAutoRemember_TypeInference checks that the keyword classifier routes
// content to the expected memory type. We don't hit the network — we call
// inferAutoRememberType directly because that's the only behavior we need to
// pin. (The handler delegates verbatim, so a separate per-handler test would
// just re-execute the same code path through stubs.)
func TestAutoRemember_TypeInference(t *testing.T) {
	cases := []struct {
		content string
		want    string
	}{
		{"decided to use React over Vue", MemoryTypeDecision},
		{"chose pgvector for embeddings", MemoryTypeDecision},
		{"fixed bug: race condition in cache", MemoryTypeBugFix},
		{"root cause was a missing nonce", MemoryTypeBugFix},
		{"always use yarn for installs", MemoryTypePreference},
		{"never use npm in this repo", MemoryTypePreference},
		{"prefer kebab-case for filenames", MemoryTypePreference},
		{"design pattern: hexagonal architecture", MemoryTypePattern},
		{"random observation about the build", MemoryTypeGeneral},
	}
	for _, c := range cases {
		if got := inferAutoRememberType(c.content); got != c.want {
			t.Errorf("inferAutoRememberType(%q) = %q, want %q", c.content, got, c.want)
		}
	}
}

// TestAutoRemember_DuplicateDetection_MatchesExisting verifies the
// matched_existing path: when the backend returns a memory whose content has
// Jaccard similarity ≥ 0.85 with the requested content, the handler must
// return action="matched_existing" with the existing id and MUST NOT issue a
// POST /memories request.
func TestAutoRemember_DuplicateDetection_MatchesExisting(t *testing.T) {
	withInitializedSession(t)

	projectID := uuid.New()
	existingID := uuid.New()
	createCalls := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/projects"):
			b, _ := json.Marshal([]stubProject{{ID: projectID, Name: "PR10Test"}})
			_, _ = w.Write(b)
		case r.URL.Path == "/memories" && r.Method == http.MethodGet:
			// ListMemories — return a memory whose content is identical to the
			// auto_remember input so Jaccard ≥ 0.85. Backend shape wraps the
			// list in a {memories, total, limit, offset} envelope.
			b, _ := json.Marshal(map[string]interface{}{
				"memories": []map[string]interface{}{{
					"id":         existingID.String(),
					"project_id": projectID.String(),
					"content":    "always use yarn for installs",
					"type":       "preference",
				}},
				"total":  1,
				"limit":  100,
				"offset": 0,
			})
			_, _ = w.Write(b)
		case r.URL.Path == "/memories" && r.Method == http.MethodPost:
			createCalls++
			b, _ := json.Marshal(map[string]interface{}{
				"id":         uuid.New().String(),
				"project_id": projectID.String(),
				"content":    "should-not-be-created",
			})
			_, _ = w.Write(b)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	res, _, err := handleAutoRemember(context.Background(), nil, AutoRememberInput{
		Content: "always use yarn for installs",
		Project: "PR10Test",
	})
	if err != nil {
		t.Fatalf("handleAutoRemember: %v", err)
	}
	if createCalls != 0 {
		t.Fatalf("matched_existing path must NOT call POST /memories; got %d call(s)", createCalls)
	}

	// Slot 0: human-readable status line.
	statusText := extractText(t, res, 0)
	if !strings.Contains(statusText, "Similar memory exists") {
		t.Errorf("slot 0 should announce the duplicate, got %q", statusText)
	}

	// Slot 1: JSON envelope.
	envelopeText := extractText(t, res, 1)
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(envelopeText), &envelope); err != nil {
		t.Fatalf("envelope slot must be JSON: %v\n%s", err, envelopeText)
	}
	if envelope["action"] != "matched_existing" {
		t.Errorf("action: got %v, want matched_existing", envelope["action"])
	}
	if envelope["memory_id"] != existingID.String() {
		t.Errorf("memory_id: got %v, want %s", envelope["memory_id"], existingID)
	}
	// matched_existing must surface a next_action so agents don't stall.
	if na, _ := envelope["next_action"].(string); na == "" {
		t.Errorf("envelope.next_action must be set on matched_existing; envelope=%+v", envelope)
	}
	meta, _ := envelope["_meta"].(map[string]interface{})
	if meta == nil || meta["protocol_reminder"] == nil {
		t.Errorf("envelope._meta.protocol_reminder must be set; envelope=%+v", envelope)
	}
	// Sub-op routing: matched_existing must NOT carry the "Saved." reminder.
	if got, _ := meta["protocol_reminder"].(string); strings.Contains(got, "Saved.") {
		t.Errorf("matched_existing reminder leaked 'Saved.' wording: %q", got)
	}
}

// TestAutoRemember_NoMatch_Creates verifies the create path: when no similar
// memory exists, the handler must POST /memories and return action="created"
// with the new memory id + auto-detected type.
func TestAutoRemember_NoMatch_Creates(t *testing.T) {
	withInitializedSession(t)

	projectID := uuid.New()
	newID := uuid.New()
	postCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/projects"):
			b, _ := json.Marshal([]stubProject{{ID: projectID, Name: "PR10Test"}})
			_, _ = w.Write(b)
		case r.URL.Path == "/memories" && r.Method == http.MethodGet:
			// No existing memories → no duplicates. Note the {memories: []}
			// envelope shape — ListMemories unmarshals MemoriesListResponse.
			_, _ = w.Write([]byte(`{"memories":[],"total":0,"limit":100,"offset":0}`))
		case r.URL.Path == "/memories" && r.Method == http.MethodPost:
			postCalled = true
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			// Memory create accepts content/type/project_id — sanity-check the
			// type is what inferAutoRememberType produced.
			if body["type"] != MemoryTypeDecision {
				t.Errorf("POST /memories type: got %v, want %s", body["type"], MemoryTypeDecision)
			}
			b, _ := json.Marshal(map[string]interface{}{
				"id":         newID.String(),
				"project_id": projectID.String(),
				"content":    body["content"],
				"type":       body["type"],
			})
			_, _ = w.Write(b)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	res, _, err := handleAutoRemember(context.Background(), nil, AutoRememberInput{
		Content: "decided to use Postgres over MongoDB",
		Project: "PR10Test",
	})
	if err != nil {
		t.Fatalf("handleAutoRemember: %v", err)
	}
	if !postCalled {
		t.Fatal("created path must call POST /memories")
	}

	statusText := extractText(t, res, 0)
	if !strings.Contains(statusText, "Saved as decision") {
		t.Errorf("slot 0 should announce save + type, got %q", statusText)
	}

	envelopeText := extractText(t, res, 1)
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(envelopeText), &envelope); err != nil {
		t.Fatalf("envelope slot must be JSON: %v\n%s", err, envelopeText)
	}
	if envelope["action"] != "created" {
		t.Errorf("action: got %v, want created", envelope["action"])
	}
	if envelope["memory_id"] != newID.String() {
		t.Errorf("memory_id: got %v, want %s", envelope["memory_id"], newID)
	}
	if envelope["type"] != MemoryTypeDecision {
		t.Errorf("type: got %v, want %s", envelope["type"], MemoryTypeDecision)
	}
	if envelope["auto_detected"] != true {
		t.Errorf("auto_detected: got %v, want true", envelope["auto_detected"])
	}
	if na, _ := envelope["next_action"].(string); na == "" {
		t.Errorf("envelope.next_action must be set on created; envelope=%+v", envelope)
	}
	meta, _ := envelope["_meta"].(map[string]interface{})
	if meta == nil || meta["protocol_reminder"] == nil {
		t.Errorf("envelope._meta.protocol_reminder must be set; envelope=%+v", envelope)
	}
}

// TestAutoRemember_TypeOverride_BypassesInference verifies that an explicit
// type_override skips the keyword classifier — useful when the agent already
// knows the classification and wants deterministic typing.
func TestAutoRemember_TypeOverride_BypassesInference(t *testing.T) {
	withInitializedSession(t)

	projectID := uuid.New()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/projects"):
			b, _ := json.Marshal([]stubProject{{ID: projectID, Name: "PR10Test"}})
			_, _ = w.Write(b)
		case r.URL.Path == "/memories" && r.Method == http.MethodGet:
			// Same envelope shape as the other tests — ListMemories unmarshals
			// MemoriesListResponse, not a bare array.
			_, _ = w.Write([]byte(`{"memories":[],"total":0,"limit":100,"offset":0}`))
		case r.URL.Path == "/memories" && r.Method == http.MethodPost:
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["type"] != "skill" {
				t.Errorf("type_override should win; got %v", body["type"])
			}
			b, _ := json.Marshal(map[string]interface{}{
				"id":         uuid.New().String(),
				"project_id": projectID.String(),
				"content":    body["content"],
				"type":       body["type"],
			})
			_, _ = w.Write(b)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	// Content reads like a decision, but type_override pins it to skill.
	res, _, err := handleAutoRemember(context.Background(), nil, AutoRememberInput{
		Content:      "decided to deploy via Railway",
		Project:      "PR10Test",
		TypeOverride: "skill",
	})
	if err != nil {
		t.Fatalf("handleAutoRemember: %v", err)
	}

	envelopeText := extractText(t, res, 1)
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(envelopeText), &envelope); err != nil {
		t.Fatalf("envelope slot must be JSON: %v\n%s", err, envelopeText)
	}
	if envelope["type"] != "skill" {
		t.Errorf("type: got %v, want skill", envelope["type"])
	}
	if envelope["auto_detected"] != false {
		t.Errorf("auto_detected should be false when type_override is set; got %v", envelope["auto_detected"])
	}
}
