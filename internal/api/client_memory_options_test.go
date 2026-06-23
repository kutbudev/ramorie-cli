package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildCreateMemoryReqBody_IncludesStructuredSkillFields(t *testing.T) {
	body := buildCreateMemoryReqBody(CreateMemoryOptions{
		ProjectID:  "project-1",
		Content:    "iOS build runbook",
		Type:       "skill",
		Tags:       []string{"ios", "build"},
		Trigger:    "before:ios-build",
		Steps:      []string{"add OTHER_LDFLAGS", "pod install", "archive"},
		Validation: "Archive succeeds",
	})

	if body["type"] != "skill" {
		t.Fatalf("type = %v, want skill", body["type"])
	}
	if body["trigger"] != "before:ios-build" {
		t.Fatalf("trigger = %v", body["trigger"])
	}
	steps, ok := body["steps"].([]string)
	if !ok || len(steps) != 3 || steps[0] != "add OTHER_LDFLAGS" {
		t.Fatalf("steps = %#v, want structured steps", body["steps"])
	}
	if body["validation"] != "Archive succeeds" {
		t.Fatalf("validation = %v", body["validation"])
	}
}

func TestBuildCreateMemoryReqBody_OmitsEmptyStructuredSkillFields(t *testing.T) {
	body := buildCreateMemoryReqBody(CreateMemoryOptions{
		ProjectID: "project-1",
		Content:   "plain memory",
	})

	if _, ok := body["trigger"]; ok {
		t.Fatal("empty trigger must be omitted")
	}
	if _, ok := body["steps"]; ok {
		t.Fatal("empty steps must be omitted")
	}
	if _, ok := body["validation"]; ok {
		t.Fatal("empty validation must be omitted")
	}
}

func TestCreateEncryptedMemoryWithOptions_ForwardsTypeAndStructuredSkillFields(t *testing.T) {
	var captured map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"11111111-1111-1111-1111-111111111111","project_id":"22222222-2222-2222-2222-222222222222","content":"[Encrypted]","type":"skill","is_encrypted":true}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.CreateEncryptedMemoryWithOptions(CreateEncryptedMemoryOptions{
		ProjectID:        "project-1",
		EncryptedContent: "ciphertext",
		ContentNonce:     "nonce",
		ContentHash:      "hash",
		Type:             "skill",
		Tags:             []string{"ios"},
		Trigger:          "before:ios-build",
		Steps:            []string{"add OTHER_LDFLAGS", "archive"},
		Validation:       "Archive succeeds",
	})
	if err != nil {
		t.Fatalf("CreateEncryptedMemoryWithOptions: %v", err)
	}

	if captured["type"] != "skill" {
		t.Fatalf("type = %v, want skill", captured["type"])
	}
	if captured["trigger"] != "before:ios-build" {
		t.Fatalf("trigger = %v", captured["trigger"])
	}
	steps, ok := captured["steps"].([]interface{})
	if !ok || len(steps) != 2 || steps[0] != "add OTHER_LDFLAGS" {
		t.Fatalf("steps = %#v, want structured steps", captured["steps"])
	}
	if captured["validation"] != "Archive succeeds" {
		t.Fatalf("validation = %v", captured["validation"])
	}
}
