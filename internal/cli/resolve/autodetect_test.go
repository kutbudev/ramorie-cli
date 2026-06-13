package resolve

import (
	"strings"
	"testing"

	"github.com/kutbudev/ramorie-cli/internal/models"
)

func TestAutoResolveProject_ExplicitName(t *testing.T) {
	// Isolate config writes (RememberLastProject) to a temp HOME.
	t.Setenv("HOME", t.TempDir())

	full := "7f691f17-1234-5678-9abc-def012345678"
	lister := &fakeProjectLister{projects: []models.Project{
		{ID: mustUUID(full), Name: "Ramorie CLI"},
		{ID: mustUUID("de3d885c-aaaa-bbbb-cccc-ddddeeeeffff"), Name: "Other"},
	}}

	// Lowercase / normalized name must resolve.
	got, err := AutoResolveProject("ramorie cli", lister)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != full {
		t.Errorf("got %q, want %q", got, full)
	}
}

func TestAutoResolveProject_SingleProjectFallback(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	only := "7f691f17-1234-5678-9abc-def012345678"
	lister := &fakeProjectLister{projects: []models.Project{
		{ID: mustUUID(only), Name: "SomeUnrelatedName"},
	}}

	// Empty arg + single project → that project (cwd won't match the name).
	got, err := AutoResolveProject("", lister)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != only {
		t.Errorf("got %q, want %q", got, only)
	}
}

func TestAutoResolveProject_AmbiguousEmptyErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	lister := &fakeProjectLister{projects: []models.Project{
		{ID: mustUUID("7f691f17-1234-5678-9abc-def012345678"), Name: "ZzzNoMatchA"},
		{ID: mustUUID("de3d885c-aaaa-bbbb-cccc-ddddeeeeffff"), Name: "ZzzNoMatchB"},
	}}

	// Empty arg, multiple projects, no cwd/git/last-used match → helpful error.
	_, err := AutoResolveProject("", lister)
	if err == nil {
		t.Fatal("expected an error for ambiguous auto-detect")
	}
	if !strings.Contains(err.Error(), "auto-detect") {
		t.Errorf("error should mention auto-detect, got: %v", err)
	}
}

func TestNormalizeForMatch(t *testing.T) {
	cases := map[string]string{
		"Ramorie CLI": "ramoriecli",
		"ramorie-cli": "ramoriecli",
		"ramorie_cli": "ramoriecli",
		"Ramorie.CLI": "ramoriecli",
	}
	for in, want := range cases {
		if got := normalizeForMatch(in); got != want {
			t.Errorf("normalizeForMatch(%q) = %q, want %q", in, got, want)
		}
	}
}
