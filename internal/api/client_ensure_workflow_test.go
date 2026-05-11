package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"github.com/kutbudev/ramorie-cli/internal/models"
)

// TestEnsureWorkflowProject_CreateSuccess locks the happy path: the CLI
// POSTs /projects with name="workflow" + encryption_required=false and the
// backend responds 201 with the new row. The struct returned by
// EnsureWorkflowProject must surface that row verbatim (id, encryption_required).
func TestEnsureWorkflowProject_CreateSuccess(t *testing.T) {
	workflowID := uuid.New()
	var postCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/projects" {
			atomic.AddInt32(&postCount, 1)
			// Drain body so the request fully completes (otherwise the
			// goroutine pool can deadlock on a slow reader).
			defer r.Body.Close()
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode POST body: %v", err)
			}
			if body["name"] != "workflow" {
				t.Errorf("POST body name = %v, want workflow", body["name"])
			}
			// encryption_required is sent as *bool → decodes as bool false.
			if body["encryption_required"] != false {
				t.Errorf("POST body encryption_required = %v, want false", body["encryption_required"])
			}
			project := models.Project{
				ID:                 workflowID,
				Name:               "workflow",
				EncryptionRequired: false,
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(project)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	c := &Client{BaseURL: ts.URL, APIKey: "test", HTTPClient: ts.Client()}
	got, err := c.EnsureWorkflowProject()
	if err != nil {
		t.Fatalf("EnsureWorkflowProject err: %v", err)
	}
	if got == nil || got.ID != workflowID {
		t.Fatalf("EnsureWorkflowProject returned %+v, want id %s", got, workflowID)
	}
	if got.EncryptionRequired {
		t.Fatalf("returned project.encryption_required = true, want false")
	}
	if atomic.LoadInt32(&postCount) != 1 {
		t.Fatalf("POST count = %d, want 1", postCount)
	}
}

// TestEnsureWorkflowProject_409Recover: a duplicate POST returns 409 — the
// CLI must transparently recover by listing personal projects and finding
// the existing workflow row. Idempotency contract: callers never see the
// 409 error string.
func TestEnsureWorkflowProject_409Recover(t *testing.T) {
	workflowID := uuid.New()
	otherID := uuid.New()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/projects":
			// Simulate duplicate row: backend folds it into 409.
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"duplicate_project","message":"Project 'workflow' already exists"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/projects":
			projects := []models.Project{
				{ID: otherID, Name: "Other"},
				{ID: workflowID, Name: "workflow", EncryptionRequired: false},
			}
			env := map[string]any{"projects": projects, "total": len(projects)}
			_ = json.NewEncoder(w).Encode(env)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c := &Client{BaseURL: ts.URL, APIKey: "test", HTTPClient: ts.Client()}
	got, err := c.EnsureWorkflowProject()
	if err != nil {
		t.Fatalf("EnsureWorkflowProject must recover from 409, got err: %v", err)
	}
	if got == nil || got.ID != workflowID {
		t.Fatalf("recovered project = %+v, want id %s", got, workflowID)
	}
}

// TestEnsureWorkflowProject_409Recover_IgnoresOrgWorkflow: a 409 must NOT
// be satisfied by an org-scoped "workflow" row — personal/org encryption
// semantics differ. If the only match is org-owned the recovery must error
// rather than silently return the wrong project.
func TestEnsureWorkflowProject_409Recover_IgnoresOrgWorkflow(t *testing.T) {
	orgWorkflowID := uuid.New()
	orgUUID := uuid.New()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/projects":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"duplicate_project"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/projects":
			projects := []models.Project{
				{ID: orgWorkflowID, Name: "workflow", OrganizationID: &orgUUID},
			}
			env := map[string]any{"projects": projects, "total": 1}
			_ = json.NewEncoder(w).Encode(env)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c := &Client{BaseURL: ts.URL, APIKey: "test", HTTPClient: ts.Client()}
	_, err := c.EnsureWorkflowProject()
	if err == nil {
		t.Fatalf("expected error when only org workflow is found, got nil")
	}
	if !strings.Contains(err.Error(), "no workflow row found") {
		t.Fatalf("error must mention 'no workflow row found', got: %v", err)
	}
}

// TestEnsureWorkflowProject_NonConflict_PropagatesError: a 500 (or any
// non-409 status) is NOT recovery territory — bubble the original error
// without listing, so callers see the real failure mode.
func TestEnsureWorkflowProject_NonConflict_PropagatesError(t *testing.T) {
	var listCalled int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/projects" {
			atomic.AddInt32(&listCalled, 1)
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"db down"}`))
	}))
	defer ts.Close()

	c := &Client{BaseURL: ts.URL, APIKey: "test", HTTPClient: ts.Client()}
	_, err := c.EnsureWorkflowProject()
	if err == nil {
		t.Fatalf("expected error on 500, got nil")
	}
	if atomic.LoadInt32(&listCalled) != 0 {
		t.Fatalf("must NOT call ListProjects on non-409 error (got %d list calls)", listCalled)
	}
}
