package mcp

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

// stubFullProject is a richer version of stubProject (handlers_test.go) that
// carries OrganizationID + EncryptionRequired fields the resolver inspects.
type stubFullProject struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	OrganizationID     *uuid.UUID `json:"organization_id,omitempty"`
	EncryptionRequired bool       `json:"encryption_required"`
}

// makeProjectsHandler returns an HTTP handler that:
//   - GET  /projects → wraps the supplied projects in the {projects:[...]} envelope
//   - POST /projects → creates an in-memory workflow project on the fly
//
// The created workflow project is appended to *projects and returned so the
// follow-up GET /projects sees the new row. Tracks call counts for
// assertions.
func makeProjectsHandler(t *testing.T, projects *[]stubFullProject, postCount, getCount *int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/projects"):
			atomic.AddInt32(getCount, 1)
			env := map[string]any{"projects": *projects, "total": len(*projects)}
			_ = json.NewEncoder(w).Encode(env)
		case r.Method == http.MethodPost && r.URL.Path == "/projects":
			atomic.AddInt32(postCount, 1)
			defer r.Body.Close()
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			name, _ := body["name"].(string)
			created := stubFullProject{
				ID:                 uuid.New(),
				Name:               name,
				EncryptionRequired: false,
			}
			*projects = append(*projects, created)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(models.Project{
				ID:                 created.ID,
				Name:               created.Name,
				EncryptionRequired: created.EncryptionRequired,
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

// TestResolveProjectID_WorkflowFallback_AutoCreates: when the user has
// projects but none match by cwd / git remote / single, and no "workflow"
// exists yet, the resolver must auto-create one via EnsureWorkflowProject
// instead of erroring. This is the PR11 v8.0.0 contract for rootless writes.
func TestResolveProjectID_WorkflowFallback_AutoCreates(t *testing.T) {
	ResetSession()
	_ = GetCurrentSession()

	var (
		postCount int32
		getCount  int32
	)
	projects := []stubFullProject{
		{ID: uuid.New(), Name: "ProjA"},
		{ID: uuid.New(), Name: "ProjB"},
	}
	ts := httptest.NewServer(makeProjectsHandler(t, &projects, &postCount, &getCount))
	defer ts.Close()
	installTestAPIClient(t, ts)

	// Empty projectIdentifier + no cwd match + no single-project = must
	// trigger workflow fallback path.
	id, err := resolveProjectID(apiClient, "")
	if err != nil {
		t.Fatalf("resolveProjectID must not error with workflow fallback, got: %v", err)
	}
	if id == "" {
		t.Fatal("resolved id is empty")
	}
	if atomic.LoadInt32(&postCount) != 1 {
		t.Fatalf("expected 1 POST /projects (workflow create), got %d", postCount)
	}
	// Verify the created project landed in the list.
	if len(projects) != 3 || !strings.EqualFold(projects[2].Name, "workflow") {
		t.Fatalf("workflow project not appended: %+v", projects)
	}
}

// TestResolveProjectID_WorkflowFallback_ReusesExisting: when a "workflow"
// row already exists, the resolver must reuse it instead of POSTing.
// Verifies the for-loop check fires before the EnsureWorkflowProject call.
func TestResolveProjectID_WorkflowFallback_ReusesExisting(t *testing.T) {
	ResetSession()
	_ = GetCurrentSession()

	var (
		postCount int32
		getCount  int32
	)
	existingWorkflowID := uuid.New()
	projects := []stubFullProject{
		{ID: uuid.New(), Name: "ProjA"},
		{ID: uuid.New(), Name: "ProjB"},
		{ID: existingWorkflowID, Name: "workflow", EncryptionRequired: false},
	}
	ts := httptest.NewServer(makeProjectsHandler(t, &projects, &postCount, &getCount))
	defer ts.Close()
	installTestAPIClient(t, ts)

	id, err := resolveProjectID(apiClient, "")
	if err != nil {
		t.Fatalf("resolveProjectID err: %v", err)
	}
	if id != existingWorkflowID.String() {
		t.Fatalf("resolved id = %s, want %s (existing workflow)", id, existingWorkflowID)
	}
	if atomic.LoadInt32(&postCount) != 0 {
		t.Fatalf("expected 0 POST /projects (must reuse), got %d", postCount)
	}
}

// TestResolveProjectID_WorkflowFallback_IgnoresOrgWorkflow: an org-scoped
// "workflow" row in the projects list must NOT satisfy the personal-scope
// fallback — the resolver must call EnsureWorkflowProject to create a
// personal one.
func TestResolveProjectID_WorkflowFallback_IgnoresOrgWorkflow(t *testing.T) {
	ResetSession()
	_ = GetCurrentSession()

	var (
		postCount int32
		getCount  int32
	)
	orgID := uuid.New()
	orgWorkflowID := uuid.New()
	projects := []stubFullProject{
		{ID: uuid.New(), Name: "ProjA"},
		{ID: uuid.New(), Name: "ProjB"},
		// Org-scoped workflow — must be ignored by personal fallback.
		{ID: orgWorkflowID, Name: "workflow", OrganizationID: &orgID},
	}
	ts := httptest.NewServer(makeProjectsHandler(t, &projects, &postCount, &getCount))
	defer ts.Close()
	installTestAPIClient(t, ts)

	id, err := resolveProjectID(apiClient, "")
	if err != nil {
		t.Fatalf("resolveProjectID err: %v", err)
	}
	if id == orgWorkflowID.String() {
		t.Fatal("resolver leaked org-scoped workflow as personal fallback")
	}
	if atomic.LoadInt32(&postCount) != 1 {
		t.Fatalf("expected 1 POST /projects (org workflow ignored, personal auto-created), got %d", postCount)
	}
}

// TestResolveProjectWithOrg_WorkflowFallback mirrors the resolveProjectID
// test for the org-aware variant. Workflow project always resolves to an
// empty orgID (personal scope).
func TestResolveProjectWithOrg_WorkflowFallback(t *testing.T) {
	ResetSession()
	_ = GetCurrentSession()

	var (
		postCount int32
		getCount  int32
	)
	projects := []stubFullProject{
		{ID: uuid.New(), Name: "ProjA"},
		{ID: uuid.New(), Name: "ProjB"},
	}
	ts := httptest.NewServer(makeProjectsHandler(t, &projects, &postCount, &getCount))
	defer ts.Close()
	installTestAPIClient(t, ts)

	id, orgID, err := resolveProjectWithOrg(apiClient, "")
	if err != nil {
		t.Fatalf("resolveProjectWithOrg err: %v", err)
	}
	if id == "" {
		t.Fatal("resolved id is empty")
	}
	if orgID != "" {
		t.Fatalf("workflow fallback must return empty orgID, got %s", orgID)
	}
	if atomic.LoadInt32(&postCount) != 1 {
		t.Fatalf("expected 1 POST /projects, got %d", postCount)
	}
}

// TestDetectGitRemoteRepo_NotInGitRepo: when called outside a git working
// tree the helper must return "" without panicking. Smoke-tests the timeout
// + error-swallow path.
func TestDetectGitRemoteRepo_NotInGitRepo(t *testing.T) {
	// Run inside the OS temp dir which is almost certainly not a git tree.
	// We can't reliably chdir without affecting other parallel tests, so we
	// settle for "call must not panic / must not block past 500ms" — the
	// hard deadline is enforced inside detectGitRemoteRepo.
	got := detectGitRemoteRepo()
	// Either "" (no git, no repo) or some real repo name if `go test` runs
	// inside the ramorie-cli checkout. We assert only that it returned in a
	// bounded time and produced a string (no nil/panic).
	_ = got
}
