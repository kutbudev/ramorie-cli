package mcp

import "testing"

// shouldBeTask is the heuristic that decides whether a remember() call
// produces a memory (default) or promotes to a task. It used to middle-match
// on "todo", "must", "should", "need to" — which misclassified prose like
// "fixed 2 TODOs" and "you must push before EOD" as tasks. Lockdown:

func TestShouldBeTask_ExplicitPrefixes(t *testing.T) {
	cases := []string{
		"todo: fix auth flow",
		"TODO: fix auth flow",
		"todo implement cache",
		"later: review the PR",
		"task: migrate to pgvector",
		"action: backup prod DB",
		"reminder: call the vendor",
		"followup: revisit after Q3",
		"follow-up: second check",
		// Leading whitespace should not prevent match.
		"   TODO: ship tomorrow",
	}
	for _, in := range cases {
		if !shouldBeTask(in) {
			t.Errorf("expected task for %q", in)
		}
	}
}

func TestShouldBeTask_MentioningKeywordsInProseIsMemory(t *testing.T) {
	// Each of these contains a task-word but not as an explicit directive.
	// All must stay as memories.
	cases := []string{
		"Fixed 2 TODOs in the resolver",
		"You must push before EOD",
		"We should always use yarn, never npm",
		"Needed to upgrade pg to 16",
		"CHANGELOG: Semantic & RAG Overhaul (Phase A-D)",
		"The later version introduced caching",
		"Bug: action returns null when input is empty",
		"Reminder is stored inline; check docs",  // word appears but no colon
		"Cleanup task was scheduled yesterday",   // ditto
	}
	for _, in := range cases {
		if shouldBeTask(in) {
			t.Errorf("expected memory (not task) for %q", in)
		}
	}
}

func TestShouldBeTask_EmptyStringIsMemory(t *testing.T) {
	if shouldBeTask("") || shouldBeTask("   ") {
		t.Error("empty or whitespace input must not classify as task")
	}
}

func TestExtractTaskDescription_StripsPrefix(t *testing.T) {
	cases := map[string]string{
		"todo: fix auth":            "fix auth",
		"TODO: fix auth":            "fix auth",
		"later: review the PR":      "review the PR",
		"task: migrate to pgvector": "migrate to pgvector",
		"   TODO: ship tomorrow":    "ship tomorrow",
	}
	for in, want := range cases {
		if got := extractTaskDescription(in); got != want {
			t.Errorf("extractTaskDescription(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractTaskDescription_NonPrefixReturnsOriginal(t *testing.T) {
	// Content that didn't pass shouldBeTask shouldn't be mangled either —
	// if the caller bypassed the heuristic and still called extract, they
	// should get the original back unchanged.
	original := "Fixed 2 TODOs in the resolver"
	if got := extractTaskDescription(original); got != original {
		t.Errorf("non-prefixed content must round-trip; got %q", got)
	}
}

func TestExtractTaskDescription_EmptyBodyReturnsOriginal(t *testing.T) {
	// "todo:" with nothing after → fall back to original so the caller
	// doesn't end up with an empty task description.
	if got := extractTaskDescription("todo:"); got != "todo:" {
		t.Errorf("empty task body must return original; got %q", got)
	}
}
