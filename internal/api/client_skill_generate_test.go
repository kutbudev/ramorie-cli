// PR8 — Smart AI Generate CLI client tests.
package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestGenerateSkill_SmartStrategy_ForwardsAllFields guards the body
// serialisation for the PR8 smart path. Strategy + manual IDs + caps
// must all reach the backend exactly as supplied so the wizard's UI
// state matches what gets ranked.
func TestGenerateSkill_SmartStrategy_ForwardsAllFields(t *testing.T) {
	var captured map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories/generate-skill" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"skill": {
				"frontmatter": {"name":"x","description":"y","when_to_use":"z","tags":["a"]},
				"body": "# X"
			},
			"ai_model": "gemini-2.5-flash",
			"latency_ms": 42,
			"token_usage": {"input": 100, "output": 200},
			"context_items_used": 12
		}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	resp, err := c.GenerateSkillSmart(GenerateSkillOptions{
		Goal:            "deploy production",
		ProjectID:       "11111111-1111-1111-1111-111111111111",
		Model:           "gemini-2.5-flash",
		Strategy:        "manual",
		ManualMemoryIDs: []string{"mem-a", "mem-b"},
		ManualTaskIDs:   []string{"task-1"},
		MaxMemories:     15,
		MaxTasks:        10,
	})
	if err != nil {
		t.Fatalf("GenerateSkillSmart: %v", err)
	}

	// Body forwarding assertions — each field must reach the backend
	// with the right type. JSON unmarshals numbers as float64.
	if captured["goal"] != "deploy production" {
		t.Errorf("goal forwarded as %v", captured["goal"])
	}
	if captured["strategy"] != "manual" {
		t.Errorf("strategy forwarded as %v", captured["strategy"])
	}
	if captured["project_id"] != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("project_id missing/wrong: %v", captured["project_id"])
	}
	if captured["model"] != "gemini-2.5-flash" {
		t.Errorf("model missing: %v", captured["model"])
	}
	memIDs, ok := captured["manual_memory_ids"].([]interface{})
	if !ok || len(memIDs) != 2 || memIDs[0] != "mem-a" {
		t.Errorf("manual_memory_ids = %v", captured["manual_memory_ids"])
	}
	taskIDs, ok := captured["manual_task_ids"].([]interface{})
	if !ok || len(taskIDs) != 1 || taskIDs[0] != "task-1" {
		t.Errorf("manual_task_ids = %v", captured["manual_task_ids"])
	}
	if v, _ := captured["max_memories"].(float64); int(v) != 15 {
		t.Errorf("max_memories = %v", captured["max_memories"])
	}
	if v, _ := captured["max_tasks"].(float64); int(v) != 10 {
		t.Errorf("max_tasks = %v", captured["max_tasks"])
	}
	// Smart path must NOT send selected_ids — that's the legacy field.
	if _, present := captured["selected_ids"]; present {
		t.Errorf("smart path should omit selected_ids; got %v", captured["selected_ids"])
	}

	// Response round-trip
	if resp.ContextItemsUsed != 12 {
		t.Errorf("ContextItemsUsed round-trip = %d", resp.ContextItemsUsed)
	}
	if resp.AIModel != "gemini-2.5-flash" {
		t.Errorf("AIModel round-trip = %q", resp.AIModel)
	}
	if !strings.Contains(resp.Skill.Body, "# X") {
		t.Errorf("body round-trip lost content: %q", resp.Skill.Body)
	}
}

// TestGenerateSkill_RelevantStrategy_OmitsEmptyManualLists keeps the
// wire payload minimal for the default path — no noise fields, no
// empty arrays. Matters because the backend logs every request body.
func TestGenerateSkill_RelevantStrategy_OmitsEmptyManualLists(t *testing.T) {
	var captured map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"skill":{"frontmatter":{},"body":""},"ai_model":"","latency_ms":0,"token_usage":{"input":0,"output":0}}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.GenerateSkillSmart(GenerateSkillOptions{
		Goal:        "ship it",
		Strategy:    "relevant",
		MaxMemories: 20,
	})
	if err != nil {
		t.Fatalf("GenerateSkillSmart: %v", err)
	}

	if _, present := captured["manual_memory_ids"]; present {
		t.Errorf("empty manual_memory_ids should be omitted; got %v", captured["manual_memory_ids"])
	}
	if _, present := captured["manual_task_ids"]; present {
		t.Errorf("empty manual_task_ids should be omitted")
	}
	if _, present := captured["max_tasks"]; present {
		t.Errorf("zero max_tasks should be omitted (we want server defaults)")
	}
	if captured["strategy"] != "relevant" {
		t.Errorf("strategy=relevant should be forwarded explicitly")
	}
}

// TestGenerateSkill_LegacyFallback_BuildsSelectedIDs preserves the
// pre-PR8 wire contract: when Strategy is empty, the client must
// concatenate memory + task IDs into a flat `selected_ids` array so
// older backends still parse the request.
func TestGenerateSkill_LegacyFallback_BuildsSelectedIDs(t *testing.T) {
	var captured map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"skill":{"frontmatter":{},"body":""},"ai_model":"","latency_ms":0,"token_usage":{"input":0,"output":0}}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.GenerateSkillSmart(GenerateSkillOptions{
		Goal:            "x",
		ManualMemoryIDs: []string{"m1"},
		ManualTaskIDs:   []string{"t1"},
	})
	if err != nil {
		t.Fatalf("GenerateSkillSmart: %v", err)
	}
	ids, ok := captured["selected_ids"].([]interface{})
	if !ok || len(ids) != 2 || ids[0] != "m1" || ids[1] != "t1" {
		t.Errorf("legacy fallback should flatten IDs into selected_ids; got %v", captured["selected_ids"])
	}
	if _, present := captured["strategy"]; present {
		t.Errorf("legacy path must not send strategy field; got %v", captured["strategy"])
	}
}
