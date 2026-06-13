package commands

import (
	"reflect"
	"testing"
)

func TestExtractProjectFlag(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantProject string
		wantRest    []string
	}{
		{"no flag", []string{"my", "note"}, "", []string{"my", "note"}},
		{"-p space form after content", []string{"my note", "-p", "noop"}, "noop", []string{"my note"}},
		{"--project space form", []string{"note", "--project", "ramorie-cli"}, "ramorie-cli", []string{"note"}},
		{"-p=val form", []string{"note", "-p=noop"}, "noop", []string{"note"}},
		{"--project=val form", []string{"note", "--project=noop"}, "noop", []string{"note"}},
		{"-pval glued form", []string{"note", "-pnoop"}, "noop", []string{"note"}},
		{"flag before content", []string{"-p", "noop", "the note"}, "noop", []string{"the note"}},
		{"only first flag taken", []string{"note", "-p", "a", "-p", "b"}, "a", []string{"note", "-p", "b"}},
		{"trailing -p without value is dropped", []string{"note", "-p"}, "", []string{"note"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotProject, gotRest := extractProjectFlag(tc.args)
			if gotProject != tc.wantProject {
				t.Errorf("project: got %q, want %q", gotProject, tc.wantProject)
			}
			if !reflect.DeepEqual(gotRest, tc.wantRest) {
				t.Errorf("rest: got %v, want %v", gotRest, tc.wantRest)
			}
		})
	}
}
