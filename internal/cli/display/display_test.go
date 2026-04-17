package display

import (
	"strings"
	"testing"
	"time"
)

// Visual snapshots: we don't assert exact ANSI bytes (lipgloss strips them
// when running under go test because stdout isn't a TTY) — we assert the
// textual content is right and widths are consistent.

func TestRelative(t *testing.T) {
	now := time.Now()
	cases := []struct {
		in   time.Time
		want string
	}{
		{now, "just now"},
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5m ago"},
		{now.Add(-2 * time.Hour), "2h ago"},
		{now.Add(-3 * 24 * time.Hour), "3d ago"},
		{time.Time{}, ""},
	}
	for _, tc := range cases {
		got := Relative(tc.in)
		if got != tc.want {
			t.Errorf("Relative(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
	// Older than 30 days falls back to YYYY-MM-DD.
	old := now.Add(-60 * 24 * time.Hour)
	got := Relative(old)
	if !strings.ContainsRune(got, '-') || len(got) != 10 {
		t.Errorf("Relative(60d ago) should be YYYY-MM-DD, got %q", got)
	}
}

func TestRelative_FutureDateFallsBackToISO(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	got := Relative(future)
	if len(got) != 10 {
		t.Errorf("future dates should render as YYYY-MM-DD to avoid negative ages; got %q", got)
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"hello", 5, "hello"},
		{"hello world", 5, "hell…"},
		{"hello", 0, ""},
		{"hi", 1, "h"},
		{"αβγδε", 3, "αβγ"},             // multi-byte safe
		{"αβγδεζηθ", 5, "αβγδ…"},        // multi-byte with ellipsis
		{"with emoji 🎉 here", 14, "with emoji 🎉 …"}, // emoji counts as one rune
	}
	for _, tc := range cases {
		got := Truncate(tc.in, tc.n)
		if got != tc.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tc.in, tc.n, got, tc.want)
		}
	}
}

func TestSingleLine(t *testing.T) {
	in := "line1\nline2\r\nline3\tindented"
	got := SingleLine(in)
	if strings.ContainsAny(got, "\n\r\t") {
		t.Errorf("SingleLine must strip control chars, got %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Errorf("SingleLine must collapse double spaces, got %q", got)
	}
	if got != "line1 line2 line3 indented" {
		t.Errorf("SingleLine content mismatch: got %q", got)
	}
}

func TestStatusIcon_RendersAllKnownStatuses(t *testing.T) {
	// ANSI escapes are stripped in non-TTY test env, so we can assert on
	// the underlying glyph.
	cases := map[string]string{
		"TODO":        "○",
		"IN_PROGRESS": "◐",
		"COMPLETED":   "✓",
		"DONE":        "✓",
		"REVIEW":      "⊙",
		"CANCELED":    "✗",
		"CANCELLED":   "✗",
		"":            "○", // unknown → same as TODO (dim circle)
		"weirdvalue":  "○",
	}
	for input, wantGlyph := range cases {
		got := StatusIcon(input)
		if !strings.Contains(got, wantGlyph) {
			t.Errorf("StatusIcon(%q) = %q, must contain %q", input, got, wantGlyph)
		}
	}
}

func TestPriorityBadge_RendersAllKnown(t *testing.T) {
	cases := map[string]string{
		"H":      "[H]",
		"HIGH":   "[H]",
		"P1":     "[H]",
		"M":      "[M]",
		"MEDIUM": "[M]",
		"L":      "[L]",
		"LOW":    "[L]",
		"??":     "[?]",
		"":       "[?]",
	}
	for in, want := range cases {
		got := PriorityBadge(in)
		if !strings.Contains(got, want) {
			t.Errorf("PriorityBadge(%q) = %q, must contain %q", in, got, want)
		}
	}
}

func TestTypeBadge_PaddedToFixedWidth(t *testing.T) {
	// Every type label should render at the same visible width so list rows
	// align. The label goes into a 12-char bracket+label+pad sequence.
	widths := map[string]int{}
	for _, typ := range []string{"general", "decision", "bug_fix", "preference", "pattern", "reference", "skill", "unknown"} {
		got := TypeBadge(typ)
		// Strip ANSI (the test env already strips but be defensive)
		plain := stripAnsi(got)
		widths[typ] = len(plain)
	}
	// All should be identical width.
	seen := -1
	for typ, w := range widths {
		if seen == -1 {
			seen = w
		}
		if w != seen {
			t.Errorf("TypeBadge widths must be uniform for alignment; %q is %d, others are %d", typ, w, seen)
		}
	}
}

func TestTags_OverflowShowsPlusN(t *testing.T) {
	got := Tags([]string{"a", "b", "c", "d", "e"}, 2)
	if !strings.Contains(got, "#a") || !strings.Contains(got, "#b") {
		t.Errorf("first N tags must be present: %q", got)
	}
	if strings.Contains(got, "#c") {
		t.Errorf("tags beyond limit must NOT appear: %q", got)
	}
	if !strings.Contains(got, "+3") {
		t.Errorf("overflow count (+3) must appear: %q", got)
	}
}

func TestTags_EmptyReturnsEmpty(t *testing.T) {
	if got := Tags(nil, 3); got != "" {
		t.Errorf("nil tags → empty string (callers choose placeholder), got %q", got)
	}
	if got := Tags([]string{}, 3); got != "" {
		t.Errorf("empty tags → empty string, got %q", got)
	}
}

func TestTags_MaxZeroOrLargerThanLenRendersAll(t *testing.T) {
	tags := []string{"a", "b", "c"}
	for _, max := range []int{0, 3, 5, -1} {
		got := Tags(tags, max)
		for _, tag := range tags {
			if !strings.Contains(got, "#"+tag) {
				t.Errorf("Tags(%v, max=%d) should render all tags, missing #%s: got %q", tags, max, tag, got)
			}
		}
		if strings.Contains(got, "+") {
			t.Errorf("no overflow indicator when max>=len, got %q", got)
		}
	}
}

func TestTerminalWidth_FallsBackGracefully(t *testing.T) {
	// In the test env stdout isn't a TTY, so we exercise the fallback path.
	w := TerminalWidth()
	if w < 60 {
		t.Errorf("TerminalWidth must enforce a min of 60 for readable output, got %d", w)
	}
}

// ---- helpers ---------------------------------------------------------------

// stripAnsi removes ANSI escape sequences in case the test env renders them.
// Defensive — lipgloss usually strips on non-TTY already.
func stripAnsi(s string) string {
	out := make([]rune, 0, len(s))
	inEsc := false
	for _, r := range s {
		if r == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
