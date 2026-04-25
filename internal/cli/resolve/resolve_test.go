package resolve

import (
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
}

func TestResolveOrg(t *testing.T) {
	orgs := []Organization{
		{ID: "82c0034f-1111-2222-3333-444444444444", Name: "Ramorie"},
		{ID: "9c9c95c2-aaaa-bbbb-cccc-dddddddddddd", Name: "Geovium"},
	}
	l := fakeOrgLister{orgs}

	cases := map[string]string{
		"Ramorie":  "82c0034f-1111-2222-3333-444444444444",
		"ramorie":  "82c0034f-1111-2222-3333-444444444444",
		"82c0034f": "82c0034f-1111-2222-3333-444444444444",
		"Geo":      "9c9c95c2-aaaa-bbbb-cccc-dddddddddddd",
	}
	for arg, want := range cases {
		got, err := ResolveOrg(arg, l)
		if err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
		if got != want {
			t.Errorf("%s: got %s want %s", arg, got, want)
		}
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
