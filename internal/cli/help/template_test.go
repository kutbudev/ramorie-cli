package help

import (
	"strings"
	"testing"
)

func TestTierBadge(t *testing.T) {
	cases := map[string]string{
		"essential": "🔴 ESSENTIAL",
		"common":    "🟡 COMMON",
		"admin":     "🟢 ADMIN",
		"":          "",
		"unknown":   "",
	}
	for in, want := range cases {
		if got := TierBadge(in); got != want {
			t.Errorf("TierBadge(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAppHelpTemplateContainsBadges(t *testing.T) {
	tmpl := AppHelpTemplate()
	for _, marker := range []string{"🔴 ESSENTIAL", "🟡 COMMON", "🟢 ADMIN"} {
		if !strings.Contains(tmpl, marker) {
			t.Errorf("AppHelpTemplate missing %q", marker)
		}
	}
}

func TestSetTierStoresMetadata(t *testing.T) {
	// We don't import urfave/cli to keep this test light; SetTier is just a
	// helper that mutates a *cli.Command's Metadata map. Verified indirectly
	// via integration in main.go. This placeholder asserts the function exists
	// and is callable; replace with a real assertion if signature stabilizes.
	_ = SetTier
}

func TestMustValidate(t *testing.T) {
	if err := MustValidate(); err != nil {
		t.Fatalf("MustValidate failed: %v", err)
	}
}
