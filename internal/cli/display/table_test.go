package display

import (
	"strings"
	"testing"
)

// Tests run with stdout = a pipe (not a TTY), so TerminalWidth() returns
// the fallback 100. That's still useful — we can verify the dropping
// behavior by squeezing the column mins past 100.

func TestNewResponsiveTable_KeepsAllColumnsOnWideEnoughTerminal(t *testing.T) {
	cols := []Column{
		{Title: "S", Min: 3, Weight: 0},
		{Title: "P", Min: 3, Weight: 0},
		{Title: "ID", Min: 8, Weight: 0},
		{Title: "TITLE", Min: 24, Weight: 4},
		{Title: "TAGS", Min: 14, Weight: 1},
		{Title: "UPDATED", Min: 10, Weight: 0},
	}
	out := NewResponsiveTable(cols, [][]string{
		{"o", "[H]", "abc12345", "demo task title", "#tag1", "2h ago"},
	})
	for _, want := range []string{"S", "P", "ID", "TITLE", "TAGS", "UPDATED"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected header %q in output, got:\n%s", want, out)
		}
	}
}

func TestNewResponsiveTable_DropsWeightedColumnsWhenTooNarrow(t *testing.T) {
	// Cols whose total Min (3+3+8+80+80+10 = 184) far exceeds the 100
	// fallback width. The two weighted 80-min columns must drop.
	cols := []Column{
		{Title: "S", Min: 3, Weight: 0},
		{Title: "P", Min: 3, Weight: 0},
		{Title: "ID", Min: 8, Weight: 0},
		{Title: "TITLE", Min: 80, Weight: 4},
		{Title: "TAGS", Min: 80, Weight: 1},
		{Title: "UPDATED", Min: 10, Weight: 0},
	}
	out := NewResponsiveTable(cols, [][]string{
		{"o", "[H]", "abc12345", "demo task", "#tag1", "2h ago"},
	})
	if strings.Contains(out, "TAGS") {
		t.Errorf("TAGS column should have been dropped, got:\n%s", out)
	}
	for _, want := range []string{"S", "P", "ID", "UPDATED"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected fixed column %q to survive, got:\n%s", want, out)
		}
	}
}
