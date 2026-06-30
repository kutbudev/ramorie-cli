package mcp

import "testing"

// TestDetectMemoryType_ScoreBased locks the scoring classifier behavior. The
// headline case is the keyword-shadow fix: "decided to always use yarn" used to
// route to decision because the decision list was checked before preference;
// the scorer must now pick preference (the stronger imperative-rule signal).
func TestDetectMemoryType_ScoreBased(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		// Keyword-shadow regression: decision verb + imperative rule → preference.
		{"decision_verb_shadows_preference", "decided to always use yarn", MemoryTypePreference},
		{"imperative_rule_beats_decision", "we should never use npm; always use yarn", MemoryTypePreference},

		// Plain decisions still classify as decision.
		{"plain_decision", "decided to use React over Vue", MemoryTypeDecision},
		{"adr_decision", "decision: adopt pgvector instead of pinecone", MemoryTypeDecision},
		{"instead_of_decision", "went with Postgres instead of Mongo", MemoryTypeDecision},

		// Bug fixes.
		{"bug_fix_root_cause", "root cause was a missing nonce; fixed parser", MemoryTypeBugFix},
		{"bug_fix_explicit", "fixed bug: race condition in cache", MemoryTypeBugFix},

		// Preferences.
		{"prefer", "prefer kebab-case for filenames", MemoryTypePreference},
		{"never_use", "never use npm in this repo", MemoryTypePreference},

		// best-practice de-overlap: preference phrasing beats pattern.
		{"best_practice_is_preference", "the best practice is to always use yarn", MemoryTypePreference},
		{"best_practice_pattern", "best practice for module structure", MemoryTypePattern},

		// Patterns.
		{"design_pattern", "design pattern: hexagonal architecture", MemoryTypePattern},

		// Skill branch (new): how-to / runbook language routes to skill.
		{"how_to_skill", "how to deploy the worker step by step", MemoryTypeSkill},
		{"runbook_skill", "runbook: rotate the signing certs", MemoryTypeSkill},

		// Reference is URL-gated.
		{"url_reference_short", "see: https://example.com/docs", MemoryTypeReference},
		{"docs_without_url_not_reference", "check the documentation for details", MemoryTypeGeneral},

		// Fallback.
		{"general", "random observation about the build", MemoryTypeGeneral},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DetectMemoryType(c.content); got != c.want {
				t.Errorf("DetectMemoryType(%q) = %q, want %q", c.content, got, c.want)
			}
		})
	}
}

// TestDetectMemoryType_StrongestSignalWins verifies the scorer picks the bucket
// with the highest cumulative weight rather than the first list to match.
func TestDetectMemoryType_StrongestSignalWins(t *testing.T) {
	// "fixed" (bug_fix, 2) vs "decided" (decision, 2) + "instead of" (decision, 2)
	// → decision should win on cumulative weight.
	content := "we decided to use a retry queue instead of inline retries; fixed flakiness"
	if got := DetectMemoryType(content); got != MemoryTypeDecision {
		t.Errorf("expected decision to outscore bug_fix, got %q", got)
	}
}
