package commands

import (
	"strings"
	"testing"
)

func TestDeriveSearchTerm_StemAndTwoParents(t *testing.T) {
	cases := []struct {
		path string
		want []string // must contain all of these tokens
	}{
		{
			path: "/Users/me/repo/src/components/memories/MemoriesPage.tsx",
			want: []string{"MemoriesPage", "components", "memories"},
		},
		{
			path: "/abs/project/internal/api/client.go",
			want: []string{"client", "internal", "api"},
		},
		{
			path: "simple.py",
			want: []string{"simple"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got := deriveSearchTerm(tc.path)
			for _, tok := range tc.want {
				if !strings.Contains(got, tok) {
					t.Errorf("derived term %q missing expected token %q", got, tok)
				}
			}
		})
	}
}

func TestDeriveSearchTerm_StripsExtension(t *testing.T) {
	got := deriveSearchTerm("/a/b/FooBar.tsx")
	if strings.Contains(got, ".tsx") {
		t.Errorf("extension should be stripped from stem, got %q", got)
	}
	if !strings.Contains(got, "FooBar") {
		t.Errorf("stem missing, got %q", got)
	}
}

func TestDeriveSearchTerm_EmptyDirSegmentsIgnored(t *testing.T) {
	// Double-slash: empty middle segment should not leak into term
	got := deriveSearchTerm("/a//b/c.go")
	tokens := strings.Fields(got)
	for _, tok := range tokens {
		if tok == "" {
			t.Errorf("empty token present in derived term: %q", got)
		}
	}
}
