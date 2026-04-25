package resolve

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

type fakeProjectLister struct {
	projects []models.Project
}

func (f *fakeProjectLister) ListProjects(_ ...string) ([]models.Project, error) {
	return f.projects, nil
}

func mustUUID(s string) uuid.UUID {
	u, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func TestResolveProject(t *testing.T) {
	full := "7f691f17-1234-5678-9abc-def012345678"
	other := "de3d885c-aaaa-bbbb-cccc-ddddeeeeffff"
	lister := &fakeProjectLister{
		projects: []models.Project{
			{ID: mustUUID(full), Name: "Ming AI"},
			{ID: mustUUID(other), Name: "Turna React"},
		},
	}

	cases := []struct {
		name string
		arg  string
		want string
	}{
		{"full uuid", full, full},
		{"short uuid prefix", "7f691f17", full},
		{"exact name match", "Ming AI", full},
		{"case insensitive name", "ming ai", full},
		{"partial name", "Turna", other},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveProject(tc.arg, lister)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveProjectNotFound(t *testing.T) {
	lister := &fakeProjectLister{projects: []models.Project{}}
	_, err := ResolveProject("nope", lister)
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}

func TestResolveProjectAmbiguous(t *testing.T) {
	lister := &fakeProjectLister{
		projects: []models.Project{
			{ID: mustUUID("11111111-1111-1111-1111-111111111111"), Name: "Foo One"},
			{ID: mustUUID("22222222-2222-2222-2222-222222222222"), Name: "Foo Two"},
		},
	}
	_, err := ResolveProject("Foo", lister)
	if err == nil {
		t.Fatal("expected ambiguous error when partial matches multiple")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected error message to contain %q, got: %v", "ambiguous", err)
	}
}

func TestResolveProjectEmptyArg(t *testing.T) {
	lister := &fakeProjectLister{projects: []models.Project{}}
	_, err := ResolveProject("", lister)
	if err == nil {
		t.Fatal("expected error for empty arg")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected error message to contain %q, got: %v", "required", err)
	}
}

func TestResolveProjectShortPrefixTooShort(t *testing.T) {
	// Project ID starts with "abcd…"; arg "abc" is 3 chars (< 4), so the
	// short-prefix branch must be skipped. Name doesn't substring-match
	// "abc" either, so resolution must fail with not-found — proving the
	// short-prefix logic did NOT return this project.
	lister := &fakeProjectLister{
		projects: []models.Project{
			{ID: mustUUID("abcd1234-1111-2222-3333-444444444444"), Name: "Zeta Project"},
		},
	}
	_, err := ResolveProject("abc", lister)
	if err == nil {
		t.Fatal("expected error: 3-char prefix must not match via short-prefix path")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected error message to contain %q, got: %v", "not found", err)
	}
}

func TestResolveProjectShortPrefixAmbiguous(t *testing.T) {
	// Two projects sharing a 4-char ID prefix; arg = that prefix should
	// produce an ambiguous error from the short-prefix branch.
	lister := &fakeProjectLister{
		projects: []models.Project{
			{ID: mustUUID("abcd1111-1111-1111-1111-111111111111"), Name: "Alpha"},
			{ID: mustUUID("abcd2222-2222-2222-2222-222222222222"), Name: "Beta"},
		},
	}
	_, err := ResolveProject("abcd", lister)
	if err == nil {
		t.Fatal("expected ambiguous error for shared 4-char prefix")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected error message to contain %q, got: %v", "ambiguous", err)
	}
}

func TestResolveOrg(t *testing.T) {
	orgs := []Organization{
		{ID: "82c0034f-1111-2222-3333-444444444444", Name: "Ramorie"},
		{ID: "9c9c95c2-aaaa-bbbb-cccc-dddddddddddd", Name: "Geovium"},
	}
	l := fakeOrgLister{orgs}

	cases := []struct {
		name string
		arg  string
		want string
	}{
		{"exact name", "Ramorie", "82c0034f-1111-2222-3333-444444444444"},
		{"case insensitive name", "ramorie", "82c0034f-1111-2222-3333-444444444444"},
		{"short uuid prefix", "82c0034f", "82c0034f-1111-2222-3333-444444444444"},
		{"partial name", "Geo", "9c9c95c2-aaaa-bbbb-cccc-dddddddddddd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveOrg(tc.arg, l)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

type fakeOrgLister struct{ orgs []Organization }

func (f fakeOrgLister) ListOrganizations() ([]Organization, error) { return f.orgs, nil }

func TestResolveID(t *testing.T) {
	full := "7f691f17-1234-5678-9abc-def012345678"
	if got, _ := ResolveID(full); got != full {
		t.Errorf("full uuid passthrough failed")
	}
	if _, err := ResolveID("7f691f17"); err != ErrNotFullUUID {
		t.Errorf("expected ErrNotFullUUID for short id")
	}
}
