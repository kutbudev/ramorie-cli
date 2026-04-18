package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/internal/api"
)

// TestHandleMemoryGenerateSkill_WithAutoContext verifies the full three-step
// skill-generation chain:
//  1. POST /memories/suggest-context is called once (because auto_context=true).
//  2. POST /memories/generate-skill is called with the IDs from step 1.
//  3. POST /memories is called with type=skill and the serialized markdown.
//  4. The returned MCP result contains both "model" and "id".
func TestHandleMemoryGenerateSkill_WithAutoContext(t *testing.T) {
	projectID := uuid.New()
	memoryID := uuid.New()
	contextItemID := "ctx-item-1"

	var (
		suggestCalled  int
		generateCalled int
		createCalled   int
		generateBody   map[string]interface{}
		createBody     map[string]interface{}
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/projects":
			// resolveProjectID calls GET /projects — return a list with one project.
			projects := []map[string]interface{}{
				{"id": projectID.String(), "name": "SkillGenTest"},
			}
			b, _ := json.Marshal(projects)
			_, _ = w.Write(b)

		case r.URL.Path == "/memories/suggest-context" && r.Method == http.MethodPost:
			suggestCalled++
			resp := map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": contextItemID, "type": "memory", "title": "Auth setup", "snippet": "...", "score": 0.9, "est_tokens": 100},
				},
				"total_tokens": 100,
			}
			b, _ := json.Marshal(resp)
			_, _ = w.Write(b)

		case r.URL.Path == "/memories/generate-skill" && r.Method == http.MethodPost:
			generateCalled++
			_ = json.NewDecoder(r.Body).Decode(&generateBody)
			resp := map[string]interface{}{
				"skill": map[string]interface{}{
					"frontmatter": map[string]interface{}{
						"name":        "Add OAuth2 Login",
						"description": "How to add OAuth2 login",
						"when_to_use": "When adding auth",
						"tags":        []string{"auth", "oauth"},
					},
					"body": "## Steps\n1. Install library\n2. Configure provider",
				},
				"ai_model":   "gemini-2.0-flash",
				"latency_ms": 1200,
				"token_usage": map[string]interface{}{
					"input":  500,
					"output": 200,
				},
			}
			b, _ := json.Marshal(resp)
			_, _ = w.Write(b)

		case r.URL.Path == "/memories" && r.Method == http.MethodPost:
			createCalled++
			_ = json.NewDecoder(r.Body).Decode(&createBody)
			mem := map[string]interface{}{
				"id":         memoryID.String(),
				"project_id": projectID.String(),
				"content":    createBody["content"],
				"type":       createBody["type"],
				"tags":       createBody["tags"],
				"created_at": "2026-04-18T00:00:00Z",
				"updated_at": "2026-04-18T00:00:00Z",
			}
			b, _ := json.Marshal(mem)
			_, _ = w.Write(b)

		default:
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	res, _, err := handleUnifiedMemory(context.Background(), nil, UnifiedMemoryInput{
		Goal:        "add OAuth2 login",
		Project:     "SkillGenTest",
		AutoContext: true,
	})
	if err != nil {
		t.Fatalf("handleUnifiedMemory with goal: %v", err)
	}

	got := decodeToolResult(t, res)

	// Assert call counts.
	if suggestCalled != 1 {
		t.Errorf("expected suggest-context to be called once; got %d", suggestCalled)
	}
	if generateCalled != 1 {
		t.Errorf("expected generate-skill to be called once; got %d", generateCalled)
	}
	if createCalled != 1 {
		t.Errorf("expected POST /memories to be called once; got %d", createCalled)
	}

	// Assert generate-skill received the context item ID from suggest-context.
	if ids, ok := generateBody["selected_ids"].([]interface{}); !ok || len(ids) == 0 || ids[0] != contextItemID {
		t.Errorf("generate-skill should receive selected_ids=[%q] from suggest-context; got %v", contextItemID, generateBody["selected_ids"])
	}

	// Assert POST /memories was called with type=skill.
	if createBody["type"] != "skill" {
		t.Errorf("POST /memories type should be 'skill'; got %v", createBody["type"])
	}
	// Content should contain the frontmatter block.
	if content, ok := createBody["content"].(string); !ok || !strings.Contains(content, "---") {
		t.Errorf("saved content should contain frontmatter delimiters; got %v", createBody["content"])
	}

	// Assert response contains "id" and "model".
	if got["id"] != memoryID.String() {
		t.Errorf("result id should be %q; got %v", memoryID.String(), got["id"])
	}
	if got["model"] != "gemini-2.0-flash" {
		t.Errorf("result model should be 'gemini-2.0-flash'; got %v", got["model"])
	}
}

// TestHandleMemoryGenerateSkill_WithoutAutoContext verifies that when
// auto_context=false, suggest-context is NOT called and generate-skill
// is called with an empty selected_ids list.
func TestHandleMemoryGenerateSkill_WithoutAutoContext(t *testing.T) {
	projectID := uuid.New()
	memoryID := uuid.New()

	var (
		suggestCalled  int
		generateCalled int
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/memories/suggest-context":
			suggestCalled++
			b, _ := json.Marshal(map[string]interface{}{"items": []interface{}{}, "total_tokens": 0})
			_, _ = w.Write(b)

		case r.URL.Path == "/memories/generate-skill" && r.Method == http.MethodPost:
			generateCalled++
			resp := map[string]interface{}{
				"skill": map[string]interface{}{
					"frontmatter": map[string]interface{}{
						"name": "Deploy with Railway", "description": "Deploy guide",
						"when_to_use": "deploying", "tags": []string{"deploy"},
					},
					"body": "## Steps\n1. railway up",
				},
				"ai_model":   "gemini-2.0-flash",
				"latency_ms": 800,
				"token_usage": map[string]interface{}{"input": 100, "output": 50},
			}
			b, _ := json.Marshal(resp)
			_, _ = w.Write(b)

		case r.URL.Path == "/memories" && r.Method == http.MethodPost:
			mem := map[string]interface{}{
				"id": memoryID.String(), "project_id": projectID.String(),
				"content": "...", "type": "skill",
				"created_at": "2026-04-18T00:00:00Z", "updated_at": "2026-04-18T00:00:00Z",
			}
			b, _ := json.Marshal(mem)
			_, _ = w.Write(b)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	_, _, err := handleUnifiedMemory(context.Background(), nil, UnifiedMemoryInput{
		Goal:        "deploy with Railway",
		AutoContext: false, // no project needed when no auto_context
	})
	if err != nil {
		t.Fatalf("handleUnifiedMemory without auto_context: %v", err)
	}

	if suggestCalled != 0 {
		t.Errorf("suggest-context must NOT be called when auto_context=false; called %d times", suggestCalled)
	}
	if generateCalled != 1 {
		t.Errorf("generate-skill should be called once; got %d", generateCalled)
	}
}

// TestSerializeSkillMarkdown verifies the frontmatter serialization helper.
func TestSerializeSkillMarkdown_FormatIsCorrect(t *testing.T) {
	from := api.SkillFrontmatter{
		Name:        "Auth Setup",
		Description: "How to configure OAuth",
		WhenToUse:   "Adding auth",
		Tags:        []string{"auth", "oauth"},
	}
	body := "## Steps\n1. Do it"
	got := serializeSkillMarkdown(from, body)

	mustContain := []string{
		"---",
		"name: Auth Setup",
		"description: How to configure OAuth",
		"when_to_use: Adding auth",
		"tags: [auth, oauth]",
		"## Steps",
	}
	for _, s := range mustContain {
		if !strings.Contains(got, s) {
			t.Errorf("serialized markdown missing %q;\ngot:\n%s", s, got)
		}
	}
}
