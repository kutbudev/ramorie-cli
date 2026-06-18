package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// accent.go resolves the TUI accent color from a user spec. The headline
// feature: an ANSI-indexed color ("0".."15") is painted by the terminal from
// ITS OWN theme palette, so the accent looks native and adapts to whatever
// scheme the user runs — unlike a fixed hex. termenv only the 0-15 form follows
// the theme (16-255 and hex are fixed RGB), and we deliberately avoid live OSC
// palette queries because they race bubbletea's raw-mode input reader.

// accentPalette is the resolved accent in the three forms the app needs.
type accentPalette struct {
	Accent  lipgloss.Color // resting accent (borders, footer keys, titles)
	Bright  lipgloss.Color // focused-pane title / brighter accent
	Glamour string         // value handed to glamour (ANSI index or hex)
}

const (
	brandAccent = "#8a87ff"
	brandBright = "#a5a3ff"
	autoAccent  = "5"  // ANSI magenta — closest theme slot to the brand violet
	autoBright  = "13" // ANSI bright magenta
)

// resolveAccent maps a config/flag spec to a concrete palette.
//
//	""|"auto"   → terminal-theme magenta (ANSI 5 / bright 13)
//	"brand"     → the fixed #8a87ff violet identity
//	"0".."15"   → that ANSI slot (bright = slot|8, capped at 15)
//	"#rrggbb"   → that exact hex (fixed RGB)
//	anything else → auto
func resolveAccent(spec string) accentPalette {
	s := strings.TrimSpace(strings.ToLower(spec))
	switch {
	case s == "" || s == "auto":
		return accentPalette{lipgloss.Color(autoAccent), lipgloss.Color(autoBright), autoAccent}
	case s == "brand":
		return accentPalette{lipgloss.Color(brandAccent), lipgloss.Color(brandBright), brandAccent}
	case isHexColor(s):
		return accentPalette{lipgloss.Color(s), lipgloss.Color(s), s}
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n <= 15 {
		bright := n | 8
		if bright > 15 {
			bright = 15
		}
		idx := strconv.Itoa(n)
		return accentPalette{lipgloss.Color(idx), lipgloss.Color(strconv.Itoa(bright)), idx}
	}
	return accentPalette{lipgloss.Color(autoAccent), lipgloss.Color(autoBright), autoAccent}
}

// isHexColor reports whether s is a valid #rgb or #rrggbb color (s is assumed
// already lower-cased).
func isHexColor(s string) bool {
	if (len(s) != 4 && len(s) != 7) || s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// accentCycle is the order the `A` key steps through in the TUI.
var accentCycle = []string{"auto", "brand", "4", "5", "6", "2", "3"}

// nextAccentSpec returns the spec after current in accentCycle (wrap-around).
func nextAccentSpec(current string) string {
	cur := strings.TrimSpace(strings.ToLower(current))
	if cur == "" {
		cur = "auto"
	}
	for i, s := range accentCycle {
		if s == cur {
			return accentCycle[(i+1)%len(accentCycle)]
		}
	}
	return "auto"
}
