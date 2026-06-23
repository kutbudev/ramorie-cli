package mcp

import "testing"

func TestEffectiveRememberMemoryType_StructuredFieldsPromoteToSkill(t *testing.T) {
	got := effectiveRememberMemoryType(
		"fixed iOS build linker issue",
		"",
		"before:ios-build",
		[]string{"add -lswiftCompatibility56"},
		"Archive succeeds",
	)
	if got != MemoryTypeSkill {
		t.Fatalf("type = %q, want %q", got, MemoryTypeSkill)
	}
}

func TestEffectiveRememberMemoryType_StructuredFieldsOverrideExplicitNonSkill(t *testing.T) {
	got := effectiveRememberMemoryType(
		"bug_fix with reusable build procedure",
		MemoryTypeBugFix,
		"",
		[]string{"run pod install", "archive"},
		"",
	)
	if got != MemoryTypeSkill {
		t.Fatalf("structured runbook must be saved as skill, got %q", got)
	}
}

func TestEffectiveRememberMemoryType_NoStructuredFieldsKeepsDetection(t *testing.T) {
	got := effectiveRememberMemoryType("root cause was a nil project id; fixed parser", "", "", nil, "")
	if got != MemoryTypeBugFix {
		t.Fatalf("type = %q, want %q", got, MemoryTypeBugFix)
	}
}
