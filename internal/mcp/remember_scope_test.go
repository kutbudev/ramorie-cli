package mcp

import "testing"

func TestResolveMemoryScope(t *testing.T) {
	tests := []struct {
		name   string
		scope  string
		global bool
		tags   []string
		want   string
	}{
		{name: "default project-scoped", want: ""},
		{name: "explicit personal", scope: "personal", want: "personal"},
		{name: "explicit global alias", scope: "global", want: "personal"},
		{name: "case-insensitive", scope: "  Personal ", want: "personal"},
		{name: "global flag", global: true, want: "personal"},
		{name: "legacy scope:global tag", tags: []string{"agent:bob", "scope:global"}, want: "personal"},
		{name: "legacy tag case-insensitive", tags: []string{"Scope:Global"}, want: "personal"},
		{name: "unrelated tags stay project-scoped", tags: []string{"scope:project", "yarn"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveMemoryScope(tt.scope, tt.global, tt.tags); got != tt.want {
				t.Fatalf("resolveMemoryScope(%q, %v, %v) = %q, want %q", tt.scope, tt.global, tt.tags, got, tt.want)
			}
		})
	}
}
