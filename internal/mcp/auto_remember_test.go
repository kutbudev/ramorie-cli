package mcp

// PR10 — tests for auto_remember (atomic find+remember) and the shared
// protocol-reminder helper. The handler covers three observable behaviors:
//   1. Type inference — keyword classifier mirrors handleRemember's logic.
//   2. Duplicate detection — Jaccard similarity ≥ 0.60 short-circuits to
//      matched_existing without creating a new memory. (Was 0.85 pre-v7.0.1
//      — prod smoke test caught real duplicates at 0.77 slipping through.)
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
// Jaccard similarity ≥ 0.60 with the requested content, the handler must
// return action="matched_existing" with the existing id and MUST NOT issue a
// POST /memories request. Existing fixture is identical content (Jaccard 1.0)
// so this case is unaffected by the v7.0.1 threshold drop from 0.85 → 0.60.
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
			// auto_remember input so Jaccard ≥ 0.60. Backend shape wraps the
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

// TestAutoRemember_Jaccard_0_60_BoundaryCase verifies the v7.0.1 threshold drop:
// a memory whose Jaccard similarity is in the (0.60, 0.85) band — duplicates
// that the old 0.85 threshold missed in prod (PR10 smoke test, ≈0.77) — must
// now be caught and short-circuit to matched_existing. Fixture math:
//   existing tokens: {alpha, beta, gamma, delta, epsilon} (5)
//   new tokens:      {alpha, beta, gamma, delta, zeta}    (5)
//   intersection = 4, union = 6 → Jaccard = 4/6 ≈ 0.667
// This is ≥ 0.60 (matches) and < 0.85 (would have missed pre-v7.0.1).
func TestAutoRemember_Jaccard_0_60_BoundaryCase(t *testing.T) {
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
			b, _ := json.Marshal(map[string]interface{}{
				"memories": []map[string]interface{}{{
					"id":         existingID.String(),
					"project_id": projectID.String(),
					"content":    "alpha beta gamma delta epsilon",
					"type":       "general",
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
		Content: "alpha beta gamma delta zeta",
		Project: "PR10Test",
	})
	if err != nil {
		t.Fatalf("handleAutoRemember: %v", err)
	}
	if createCalls != 0 {
		t.Fatalf("0.60-band duplicate must NOT call POST /memories; got %d call(s)", createCalls)
	}

	envelopeText := extractText(t, res, 1)
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(envelopeText), &envelope); err != nil {
		t.Fatalf("envelope slot must be JSON: %v\n%s", err, envelopeText)
	}
	if envelope["action"] != "matched_existing" {
		t.Errorf("action: got %v, want matched_existing (Jaccard ≈ 0.667 must match)", envelope["action"])
	}
	// Sanity: the similarity surfaced should be in the (0.60, 0.85) band that
	// motivated this fix — if this drifts, the fixture math is wrong.
	sim, _ := envelope["similarity"].(float64)
	if sim < 0.60 || sim >= 0.85 {
		t.Errorf("similarity %.4f outside expected (0.60, 0.85) boundary band — fixture drifted", sim)
	}
}

// TestAutoRemember_Jaccard_BelowThreshold_Creates verifies the lower bound: a
// memory whose Jaccard similarity is < 0.60 (incidental overlap, not a true
// duplicate) must fall through to creation. Fixture math:
//   existing tokens: {alpha, beta, gamma, delta, epsilon} (5)
//   new tokens:      {alpha, beta, omega, sigma, kappa}   (5)
//   intersection = 2, union = 8 → Jaccard = 2/8 = 0.25
// This guards against over-aggressive dedupe after the 0.85 → 0.60 drop.
func TestAutoRemember_Jaccard_BelowThreshold_Creates(t *testing.T) {
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
			b, _ := json.Marshal(map[string]interface{}{
				"memories": []map[string]interface{}{{
					"id":         uuid.New().String(),
					"project_id": projectID.String(),
					"content":    "alpha beta gamma delta epsilon",
					"type":       "general",
				}},
				"total":  1,
				"limit":  100,
				"offset": 0,
			})
			_, _ = w.Write(b)
		case r.URL.Path == "/memories" && r.Method == http.MethodPost:
			postCalled = true
			b, _ := json.Marshal(map[string]interface{}{
				"id":         newID.String(),
				"project_id": projectID.String(),
				"content":    "alpha beta omega sigma kappa",
				"type":       "general",
			})
			_, _ = w.Write(b)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	res, _, err := handleAutoRemember(context.Background(), nil, AutoRememberInput{
		Content: "alpha beta omega sigma kappa",
		Project: "PR10Test",
	})
	if err != nil {
		t.Fatalf("handleAutoRemember: %v", err)
	}
	if !postCalled {
		t.Fatal("Jaccard < 0.60 must fall through to POST /memories")
	}
	envelopeText := extractText(t, res, 1)
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(envelopeText), &envelope); err != nil {
		t.Fatalf("envelope slot must be JSON: %v\n%s", err, envelopeText)
	}
	if envelope["action"] != "created" {
		t.Errorf("action: got %v, want created (Jaccard 0.25 should not match)", envelope["action"])
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

// TestAutoRemember_SemanticDuplicate_MatchesViaFind verifies the v7.0.2 semantic
// dedupe gate: when the backend find pipeline (cosine + rerank, FastMode) returns
// a hit whose score ≥ autoRememberFindScoreThreshold (0.75), auto_remember must
// short-circuit to matched_existing with match_source="semantic_find" — WITHOUT
// falling through to Jaccard and WITHOUT POSTing a new memory. Token sets here
// deliberately diverge so Jaccard alone would miss the duplicate (the v7.0.2
// motivating case: paraphrased memories scoring 0.867 via cosine while Jaccard
// reported 0.0).
func TestAutoRemember_SemanticDuplicate_MatchesViaFind(t *testing.T) {
	withInitializedSession(t)

	projectID := uuid.New()
	existingID := uuid.New()
	createCalls := 0
	findCalls := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/projects"):
			b, _ := json.Marshal([]stubProject{{ID: projectID, Name: "PR10Test"}})
			_, _ = w.Write(b)
		case r.URL.Path == "/memory/find" && r.Method == http.MethodPost:
			findCalls++
			// Sanity: FastMode must be set so backend skips HyDE (saves 5-10s).
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["fast_mode"] != true {
				t.Errorf("auto_remember find call must pass fast_mode=true; got body=%+v", body)
			}
			// Return a top hit at 0.85 — above 0.75 threshold, simulating the
			// 0.867 paraphrase match from the v7.0.2 smoke test.
			b, _ := json.Marshal(map[string]interface{}{
				"items": []map[string]interface{}{{
					"id":         existingID.String(),
					"type":       "preference",
					"title":      "yarn preference",
					"preview":    "always reach for yarn",
					"score":      0.85,
					"created_at": "2026-05-01T00:00:00Z",
				}},
				"_meta": map[string]interface{}{
					"total":              1,
					"returned":           1,
					"actual_tokens_est":  10,
					"applied_scope":      "project",
					"ranking_mode":       "weighted",
					"latency_ms":         42,
					"semantic_available": true,
				},
			})
			_, _ = w.Write(b)
		case r.URL.Path == "/memories" && r.Method == http.MethodGet:
			// Jaccard fallback should NOT be reached on the semantic hit path,
			// but stub a benign empty list in case future test reordering hits
			// it — keeps the failure mode "wrong path taken" not "panic".
			_, _ = w.Write([]byte(`{"memories":[],"total":0,"limit":100,"offset":0}`))
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

	// Content is a paraphrase — token overlap with the "existing" memory is low
	// enough that Jaccard alone would not match. The find mock returns a 0.85
	// cosine score, which is the only signal that should drive the match.
	res, _, err := handleAutoRemember(context.Background(), nil, AutoRememberInput{
		Content: "I tend to reach for yarn when installing packages",
		Project: "PR10Test",
	})
	if err != nil {
		t.Fatalf("handleAutoRemember: %v", err)
	}
	if findCalls != 1 {
		t.Errorf("expected exactly 1 POST /memory/find call (semantic gate); got %d", findCalls)
	}
	if createCalls != 0 {
		t.Fatalf("semantic_find match must NOT POST /memories; got %d call(s)", createCalls)
	}

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
	if envelope["match_source"] != "semantic_find" {
		t.Errorf("match_source: got %v, want semantic_find (Jaccard fallback would be 'jaccard')", envelope["match_source"])
	}
	// similarity surfaced should be the find score (0.85), not a Jaccard value.
	sim, _ := envelope["similarity"].(float64)
	if sim < 0.75 {
		t.Errorf("similarity %.4f must be ≥ threshold 0.75 to have triggered the match", sim)
	}

	statusText := extractText(t, res, 0)
	if !strings.Contains(statusText, "Semantic duplicate") {
		t.Errorf("slot 0 should announce semantic duplicate, got %q", statusText)
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
