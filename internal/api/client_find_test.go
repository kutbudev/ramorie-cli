package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a Client wired to ts.URL with api-key + agent headers
// pre-populated, so each test can inspect what the CLI actually sends.
func newTestClient(ts *httptest.Server) *Client {
	return &Client{
		BaseURL:        ts.URL,
		APIKey:         "test-api-key",
		AgentName:      "claude-test",
		AgentModel:     "claude-opus-4-7",
		AgentSessionID: "sess-123",
		HTTPClient:     ts.Client(),
	}
}

func TestMakeRequestWithHeaders_SendsAuthAgentAndExtraHeaders(t *testing.T) {
	var captured http.Header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	extra := map[string]string{"X-Project-Hint": "ramorie-cli"}

	_, err := c.makeRequestWithHeaders("POST", "/memory/find", map[string]any{"term": "x"}, extra)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Auth
	if got := captured.Get("Authorization"); got != "Bearer test-api-key" {
		t.Errorf("Authorization header missing/wrong: %q", got)
	}
	// Agent triad
	if captured.Get("X-Agent-Name") != "claude-test" {
		t.Errorf("X-Agent-Name not set")
	}
	if captured.Get("X-Agent-Model") != "claude-opus-4-7" {
		t.Errorf("X-Agent-Model not set")
	}
	if captured.Get("X-Agent-Session-ID") != "sess-123" {
		t.Errorf("X-Agent-Session-ID not set")
	}
	if captured.Get("X-Created-Via") != "mcp" {
		t.Errorf("X-Created-Via should identify the call path for backend event tracking")
	}
	// Content-Type when body present
	if ct := captured.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q want application/json", ct)
	}
	// Extra header
	if captured.Get("X-Project-Hint") != "ramorie-cli" {
		t.Errorf("custom X-Project-Hint header not forwarded")
	}
}

func TestMakeRequestWithHeaders_NoBodyOmitsContentType(t *testing.T) {
	var captured http.Header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
	}))
	defer ts.Close()
	c := newTestClient(ts)

	_, err := c.makeRequestWithHeaders("GET", "/projects", nil, nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if ct := captured.Get("Content-Type"); ct != "" {
		t.Errorf("Content-Type should be empty for body-less requests, got %q", ct)
	}
}

func TestMakeRequestWithHeaders_SurfacesHTTPErrorBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad term"}`))
	}))
	defer ts.Close()
	c := newTestClient(ts)

	_, err := c.makeRequestWithHeaders("POST", "/memory/find", map[string]any{"term": ""}, nil)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "bad term") {
		t.Errorf("error should include status + body, got %v", err)
	}
}

func TestFindMemories_BuildsExpectedBody(t *testing.T) {
	var capturedBody map[string]any
	var capturedHint string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHint = r.Header.Get("X-Project-Hint")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &capturedBody)

		// Minimal valid FindResponse
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"_meta":{"total":0,"returned":0,"ranking_mode":"hybrid","applied_scope":"project","latency_ms":1}}`))
	}))
	defer ts.Close()
	c := newTestClient(ts)

	resp, err := c.FindMemories(FindMemoriesOptions{
		Term:             "rtk query pattern",
		Project:          "Ramorie Frontend",
		ProjectHint:      "ramorie-frontend",
		Types:            []string{"pattern", "skill"},
		Tags:             []string{"frontend"},
		Limit:            5,
		BudgetTokens:     1500,
		MinScore:         0.2,
		IncludeDecisions: true,
		Purpose:          "coding",
	})
	if err != nil {
		t.Fatalf("FindMemories: %v", err)
	}
	if resp == nil {
		t.Fatal("response must not be nil on success")
	}
	if resp.Meta.RankingMode != "hybrid" {
		t.Errorf("meta round-trip failed, got %+v", resp.Meta)
	}

	// X-Project-Hint forwarded separately from body.Project
	if capturedHint != "ramorie-frontend" {
		t.Errorf("X-Project-Hint header: got %q want %q", capturedHint, "ramorie-frontend")
	}
	// Body fields
	mustEq := func(key string, want any) {
		if got := capturedBody[key]; got != want {
			t.Errorf("body[%s]: got %v want %v", key, got, want)
		}
	}
	mustEq("term", "rtk query pattern")
	mustEq("project", "Ramorie Frontend")
	mustEq("purpose", "coding")
	mustEq("include_decisions", true)

	// limit/budget are JSON numbers (float64)
	if capturedBody["limit"] != float64(5) {
		t.Errorf("limit: got %v want 5", capturedBody["limit"])
	}
	if capturedBody["budget_tokens"] != float64(1500) {
		t.Errorf("budget_tokens: got %v want 1500", capturedBody["budget_tokens"])
	}
	// Slices present
	if arr, ok := capturedBody["types"].([]any); !ok || len(arr) != 2 {
		t.Errorf("types slice not forwarded: %v", capturedBody["types"])
	}
	if arr, ok := capturedBody["tags"].([]any); !ok || len(arr) != 1 {
		t.Errorf("tags slice not forwarded: %v", capturedBody["tags"])
	}
}

func TestFindMemories_OmitsZeroValueFields(t *testing.T) {
	var capturedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &capturedBody)
		_, _ = w.Write([]byte(`{"items":[],"_meta":{}}`))
	}))
	defer ts.Close()
	c := newTestClient(ts)

	_, err := c.FindMemories(FindMemoriesOptions{Term: "x"})
	if err != nil {
		t.Fatalf("FindMemories: %v", err)
	}

	// Keys for zero-value fields should NOT appear — keeps payloads lean and
	// makes backend default handling unambiguous.
	mustAbsent := []string{"project", "types", "tags", "limit", "budget_tokens", "min_score", "include_decisions", "purpose"}
	for _, k := range mustAbsent {
		if _, present := capturedBody[k]; present {
			t.Errorf("body must omit zero-value field %q, but it was present: %v", k, capturedBody[k])
		}
	}
	if capturedBody["term"] != "x" {
		t.Errorf("term must always be present")
	}
}

func TestFindMemories_InvalidJSONResponseSurfacesAsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json at all`))
	}))
	defer ts.Close()
	c := newTestClient(ts)

	_, err := c.FindMemories(FindMemoriesOptions{Term: "x"})
	if err == nil {
		t.Fatal("malformed JSON response should surface as error")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error should mention unmarshal, got %v", err)
	}
}
