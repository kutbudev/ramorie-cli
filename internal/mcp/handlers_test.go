package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kutbudev/ramorie-cli/internal/api"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

// Helpers ---------------------------------------------------------------------

// installTestAPIClient swaps the package-level apiClient with one pointed at
// ts.URL and restores the previous client on cleanup — so parallel test files
// don't see stale state.
func installTestAPIClient(t *testing.T, ts *httptest.Server) {
	t.Helper()
	prev := apiClient
	apiClient = &api.Client{
		BaseURL:    ts.URL,
		APIKey:     "test-key",
		HTTPClient: ts.Client(),
	}
	t.Cleanup(func() { apiClient = prev })
}

// decodeToolResult extracts the JSON text payload produced by textResult and
// returns it as a generic map for field inspection.
func decodeToolResult(t *testing.T, res *mcp.CallToolResult) map[string]any {
	t.Helper()
	if res == nil {
		t.Fatal("nil CallToolResult")
	}
	if len(res.Content) == 0 {
		t.Fatal("CallToolResult has no Content")
	}
	txt, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", res.Content[0])
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(txt.Text), &out); err != nil {
		t.Fatalf("unmarshal CallToolResult text: %v\npayload=%s", err, txt.Text)
	}
	return out
}

// stubProjectsEndpoint returns a handler serving a deterministic project list
// so detectCwdProject has something to find. Matches the minimal JSON shape
// api.ListProjects expects.
func stubProjectsEndpoint(projects []stubProject) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(projects)
		_, _ = w.Write(b)
	}
}

type stubProject struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// handleSetupAgent compact response --------------------------------------------

func TestHandleSetupAgent_CompactResponseShape(t *testing.T) {
	projectID := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/projects"):
			// cwd detection + setupAgent's own projects_count call both hit
			// this — serve the same shape.
			stubProjectsEndpoint([]stubProject{{ID: projectID, Name: "HandlersTest"}})(w, r)
		case strings.HasPrefix(r.URL.Path, "/reports/stats"):
			_, _ = w.Write([]byte(`{"todo":0,"in_progress":0,"completed":0,"total":0}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	// Without matching cwd→project, detected_project simply isn't set.
	res, _, err := handleSetupAgent(context.Background(), nil, SetupAgentInput{
		AgentName:  "smoke-agent",
		AgentModel: "test-model",
	})
	if err != nil {
		t.Fatalf("handleSetupAgent: %v", err)
	}
	got := decodeToolResult(t, res)

	// Compact response must include the lean signal set.
	mustHave := []string{"session", "status", "version", "stats", "next_action", "projects_count"}
	for _, k := range mustHave {
		if _, ok := got[k]; !ok {
			t.Errorf("compact response missing %q; payload=%v", k, got)
		}
	}
	// And must NOT include the verbose-only fields.
	mustNotHave := []string{"workflow_pattern", "workflow_guide", "recommended_actions", "last_used_project", "context_injection"}
	for _, k := range mustNotHave {
		if _, ok := got[k]; ok {
			t.Errorf("compact response must omit %q (legacy field); payload=%v", k, got)
		}
	}

	session, _ := got["session"].(map[string]any)
	if session == nil || session["agent_name"] != "smoke-agent" {
		t.Errorf("session.agent_name should echo input; got %v", session)
	}
}

func TestHandleSetupAgent_CwdMatchedPutsDetectedProjectInResponse(t *testing.T) {
	// Create a tmpdir whose basename matches a project name, then chdir in.
	root := t.TempDir()
	matchDir := filepath.Join(root, "Ramorie-Frontend")
	_ = os.Mkdir(matchDir, 0o755)
	origCwd, _ := os.Getwd()
	if err := os.Chdir(matchDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })

	projectID := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/projects"):
			stubProjectsEndpoint([]stubProject{{ID: projectID, Name: "Ramorie Frontend"}})(w, r)
		case strings.HasPrefix(r.URL.Path, "/reports/stats"):
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	res, _, err := handleSetupAgent(context.Background(), nil, SetupAgentInput{
		AgentName: "test",
	})
	if err != nil {
		t.Fatalf("handleSetupAgent: %v", err)
	}
	got := decodeToolResult(t, res)

	detected, ok := got["detected_project"].(map[string]any)
	if !ok || detected == nil {
		t.Fatalf("cwd matching project by normalized-name should populate detected_project; got %v", got["detected_project"])
	}
	if detected["name"] != "Ramorie Frontend" {
		t.Errorf("detected_project.name: got %v want %q", detected["name"], "Ramorie Frontend")
	}
	if detected["id"] != projectID.String() {
		t.Errorf("detected_project.id: got %v want %s", detected["id"], projectID)
	}

	next, _ := got["next_action"].(string)
	if !strings.Contains(next, "Ramorie Frontend") {
		t.Errorf("next_action should mention the detected project so the agent uses it; got %q", next)
	}
}

func TestHandleSetupAgent_FullModeAddsLegacyFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/projects"):
			_, _ = w.Write([]byte(`[]`))
		case strings.HasPrefix(r.URL.Path, "/reports/stats"):
			_, _ = w.Write([]byte(`{}`))
		default:
			// Focus, tasks, agent-events all optional — return empty/204 equivalents.
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	res, _, err := handleSetupAgent(context.Background(), nil, SetupAgentInput{
		AgentName: "test",
		Full:      true,
	})
	if err != nil {
		t.Fatalf("handleSetupAgent full: %v", err)
	}
	got := decodeToolResult(t, res)

	// Full mode must include the verbose payload agents opting in expect.
	mustHave := []string{"workflow_pattern", "workflow_guide", "recommended_actions"}
	for _, k := range mustHave {
		if _, ok := got[k]; !ok {
			t.Errorf("full mode missing %q; payload keys=%v", k, keysOf(got))
		}
	}
	// And MUST NOT include `next_action` (that's compact-mode's nudge).
	if _, ok := got["next_action"]; ok {
		t.Errorf("full mode should use workflow_pattern/workflow_guide instead of next_action")
	}
}

// handleSetupAgent active preferences injection -------------------------------

// setup_agent must surface the user's top-5 preferences so Claude / Cursor see
// durable rules ("always yarn", "never push without approval") before any
// tool call. The endpoint is new (backend /v1/memory/preferences) — these
// tests lock the wire-up so a future refactor doesn't silently drop it.

func TestHandleSetupAgent_CompactIncludesActivePreferences(t *testing.T) {
	projectID := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/projects"):
			stubProjectsEndpoint([]stubProject{{ID: projectID, Name: "HandlersTest"}})(w, r)
		case strings.HasPrefix(r.URL.Path, "/reports/stats"):
			_, _ = w.Write([]byte(`{"todo":0,"in_progress":0,"completed":0,"total":0}`))
		case strings.HasPrefix(r.URL.Path, "/memory/preferences"):
			// Three preferences, descending access_count — matches backend's
			// ORDER BY access_count DESC ordering.
			_, _ = w.Write([]byte(`{
				"preferences": [
					{"id":"p1","content":"Always use yarn, never npm","access_count":42,"updated_at":"2026-04-18T12:00:00Z"},
					{"id":"p2","content":"Never push without explicit approval","access_count":30,"updated_at":"2026-04-18T12:00:00Z"},
					{"id":"p3","content":"Spec belliyken direkt devam","access_count":15,"updated_at":"2026-04-18T12:00:00Z"}
				],
				"count": 3
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	res, _, err := handleSetupAgent(context.Background(), nil, SetupAgentInput{AgentName: "pref-test"})
	if err != nil {
		t.Fatalf("handleSetupAgent: %v", err)
	}
	got := decodeToolResult(t, res)

	prefsAny, ok := got["active_preferences"]
	if !ok {
		t.Fatalf("compact response missing active_preferences; payload=%v", got)
	}
	prefs, ok := prefsAny.([]any)
	if !ok {
		t.Fatalf("active_preferences should be []; got %T", prefsAny)
	}
	if len(prefs) != 3 {
		t.Fatalf("expected 3 preferences; got %d", len(prefs))
	}
	first, _ := prefs[0].(map[string]any)
	if first["content"] != "Always use yarn, never npm" {
		t.Errorf("top preference by access_count wrong; got %v", first["content"])
	}
	// access_count must survive round-trip (json numbers → float64).
	if ac, _ := first["access_count"].(float64); ac != 42 {
		t.Errorf("access_count not preserved; got %v", first["access_count"])
	}
}

func TestHandleSetupAgent_PreferencesFailureIsNonFatal(t *testing.T) {
	// Contract: if /memory/preferences is down, session still opens cleanly —
	// we must NOT fail the whole setup_agent call just because preferences
	// aren't available.
	projectID := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/projects"):
			stubProjectsEndpoint([]stubProject{{ID: projectID, Name: "HandlersTest"}})(w, r)
		case strings.HasPrefix(r.URL.Path, "/reports/stats"):
			_, _ = w.Write([]byte(`{"todo":0,"in_progress":0,"completed":0,"total":0}`))
		case strings.HasPrefix(r.URL.Path, "/memory/preferences"):
			// Simulate backend outage — 500 with a plausible error body.
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"database error"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	res, _, err := handleSetupAgent(context.Background(), nil, SetupAgentInput{AgentName: "pref-fail-test"})
	if err != nil {
		t.Fatalf("handleSetupAgent must not fail on preferences outage; got %v", err)
	}
	got := decodeToolResult(t, res)

	// No active_preferences key when the endpoint failed.
	if _, ok := got["active_preferences"]; ok {
		t.Errorf("failed preferences call must NOT populate active_preferences; got %v", got["active_preferences"])
	}
	// Rest of the session payload must still be intact.
	for _, k := range []string{"session", "status", "stats", "projects_count"} {
		if _, ok := got[k]; !ok {
			t.Errorf("failure on preferences broke field %q; payload=%v", k, got)
		}
	}
}

// handleListProjects shape tests ---------------------------------------------

func TestHandleListProjects_CompactDefaultShape(t *testing.T) {
	orgID := uuid.New()
	pID := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ListProjects returns []models.Project; we serialize a minimal version.
		payload := []map[string]any{
			{
				"id":   pID.String(),
				"name": "Alpha",
				"organization": map[string]any{
					"id":   orgID.String(),
					"name": "OrgAlpha",
				},
			},
		}
		b, _ := json.Marshal(payload)
		_, _ = w.Write(b)
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	res, _, err := handleListProjects(context.Background(), nil, ListProjectsInput{})
	if err != nil {
		t.Fatalf("handleListProjects: %v", err)
	}
	got := decodeToolResult(t, res)

	items, ok := got["items"].([]any)
	if !ok {
		t.Fatalf("compact response must wrap in items[]; got %v", got)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0].(map[string]any)
	// Compact shape: id + name + optional org — nothing else.
	if item["id"] != pID.String() {
		t.Errorf("id missing or wrong: %v", item["id"])
	}
	if item["name"] != "Alpha" {
		t.Errorf("name missing: %v", item["name"])
	}
	if item["org"] != "OrgAlpha" {
		t.Errorf("org name should be flattened as `org`: %v", item["org"])
	}
	// Compact shape should NOT leak nested objects / timestamps.
	forbidden := []string{"organization", "created_at", "updated_at", "description"}
	for _, f := range forbidden {
		if _, present := item[f]; present {
			t.Errorf("compact item leaked %q — defeats the token-budget goal", f)
		}
	}
}

func TestHandleListProjects_VerboseOptInReturnsFullShape(t *testing.T) {
	pID := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := []map[string]any{
			{
				"id":          pID.String(),
				"name":        "Verbose",
				"description": "Lorem ipsum",
				"organization": map[string]any{
					"id": uuid.New().String(), "name": "Org",
				},
				"created_at": "2026-01-01T00:00:00Z",
			},
		}
		b, _ := json.Marshal(payload)
		_, _ = w.Write(b)
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	res, _, err := handleListProjects(context.Background(), nil, ListProjectsInput{Verbose: true})
	if err != nil {
		t.Fatalf("handleListProjects verbose: %v", err)
	}
	got := decodeToolResult(t, res)

	// Verbose path uses formatMCPResponse — items live under "items" with full
	// nested shape (organization object, timestamps, description).
	items, ok := got["items"].([]any)
	if !ok {
		t.Fatalf("verbose response should still use items[] envelope; got %v", got)
	}
	if len(items) == 0 {
		t.Fatal("no items in verbose response")
	}
	item := items[0].(map[string]any)
	if _, ok := item["organization"].(map[string]any); !ok {
		t.Errorf("verbose must preserve nested organization object; got %v", item["organization"])
	}
	if item["description"] != "Lorem ipsum" {
		t.Errorf("verbose must include description; got %v", item["description"])
	}
}

// handleFind forwards args correctly -----------------------------------------

func TestHandleFind_BodyForwardingAndProjectHint(t *testing.T) {
	var capturedBody map[string]any
	var capturedHint string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/memory/find":
			capturedHint = r.Header.Get("X-Project-Hint")
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &capturedBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"id":"aaa","type":"memory","title":"t","preview":"p","score":0.9,"created_at":"2026-01-01T00:00:00Z"}],"_meta":{"total":1,"returned":1,"ranking_mode":"hybrid"}}`))
		default:
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer ts.Close()
	installTestAPIClient(t, ts)

	// Force a chdir to a directory whose basename doesn't match any project —
	// detectCwdProject returns nil → no hint.
	origCwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origCwd) })
	noMatch := t.TempDir()
	_ = os.Chdir(noMatch)

	incl := true
	res, _, err := handleFind(context.Background(), nil, FindInput{
		Term:             "rtk pattern",
		Project:          "Ramorie Frontend",
		Types:            []string{"pattern"},
		Limit:            3,
		BudgetTokens:     1500,
		IncludeDecisions: &incl,
		Purpose:          "coding",
	})
	if err != nil {
		t.Fatalf("handleFind: %v", err)
	}
	got := decodeToolResult(t, res)
	if items, _ := got["items"].([]any); len(items) != 1 {
		t.Errorf("response should round-trip; got %v", got)
	}

	// Body checks
	if capturedBody["term"] != "rtk pattern" {
		t.Errorf("term not forwarded: %v", capturedBody["term"])
	}
	if capturedBody["project"] != "Ramorie Frontend" {
		t.Errorf("project not forwarded: %v", capturedBody["project"])
	}
	if capturedBody["include_decisions"] != true {
		t.Errorf("include_decisions true should forward; got %v", capturedBody["include_decisions"])
	}

	// No cwd match → no X-Project-Hint header.
	if capturedHint != "" {
		t.Errorf("X-Project-Hint should be empty when no cwd project matches and project is explicit; got %q", capturedHint)
	}
}

func TestHandleFind_MissingTermErrors(t *testing.T) {
	_, _, err := handleFind(context.Background(), nil, FindInput{Term: "   "})
	if err == nil {
		t.Fatal("empty/whitespace term must error — backend rejects it too")
	}
	if !strings.Contains(err.Error(), "term") {
		t.Errorf("error should mention term, got %v", err)
	}
}

// detectCwdProject — fuzzy matching --------------------------------------------

func TestDetectCwdProject_MatchesByNormalizedName(t *testing.T) {
	origCwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origCwd) })

	root := t.TempDir()
	dir := filepath.Join(root, "ramorie_frontend") // underscore form
	_ = os.Mkdir(dir, 0o755)
	_ = os.Chdir(dir)

	projectID := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stubProjectsEndpoint([]stubProject{{ID: projectID, Name: "Ramorie Frontend"}})(w, r)
	}))
	defer ts.Close()
	c := &api.Client{BaseURL: ts.URL, APIKey: "k", HTTPClient: ts.Client()}

	p, cwd, err := detectCwdProject(c)
	if err != nil {
		t.Fatalf("detectCwdProject: %v", err)
	}
	if !strings.HasSuffix(cwd, dir) {
		t.Errorf("cwd should match chdir'd path; got %q", cwd)
	}
	if p == nil {
		t.Fatal("project should match via normalizeForMatch (underscores/spaces/case equivalence)")
	}
	if p.ID != projectID {
		t.Errorf("wrong project resolved; got %s want %s", p.ID, projectID)
	}
}

func TestDetectCwdProject_NoMatchReturnsNilNoError(t *testing.T) {
	origCwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origCwd) })
	_ = os.Chdir(t.TempDir())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stubProjectsEndpoint([]stubProject{{ID: uuid.New(), Name: "SomethingElse"}})(w, r)
	}))
	defer ts.Close()
	c := &api.Client{BaseURL: ts.URL, APIKey: "k", HTTPClient: ts.Client()}

	p, _, err := detectCwdProject(c)
	if err != nil {
		t.Fatalf("no-match path should not error; got %v", err)
	}
	if p != nil {
		t.Errorf("no match → nil project; got %v", p)
	}
}

func TestDetectCwdProject_ShortNameIgnored(t *testing.T) {
	// normalizeForMatch-derived match requires len >= 4 so 3-letter project
	// names don't falsely match arbitrary cwd segments.
	origCwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origCwd) })
	_ = os.Chdir(t.TempDir())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stubProjectsEndpoint([]stubProject{{ID: uuid.New(), Name: "api"}})(w, r)
	}))
	defer ts.Close()
	c := &api.Client{BaseURL: ts.URL, APIKey: "k", HTTPClient: ts.Client()}

	p, _, _ := detectCwdProject(c)
	if p != nil {
		t.Errorf("short project name (<4 chars) should not auto-match; got %v", p)
	}
}

// Ensure models import is used (avoid unused-import lint in some layouts).
var _ = models.Project{}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
