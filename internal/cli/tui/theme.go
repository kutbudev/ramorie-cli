package tui

// Theme name constants for glamour + lipgloss styling.
const (
	ThemeAuto    = "auto"
	ThemeDark    = "dark"
	ThemeLight   = "light"
	ThemeDracula = "dracula"
	ThemeNotty   = "notty"
)

// allThemes is the cycle order for the `t` keybinding.
var allThemes = []string{
	ThemeAuto, ThemeDark, ThemeLight, ThemeDracula, ThemeNotty,
}

// nextTheme returns the theme after current (wrap-around).
func nextTheme(current string) string {
	for i, n := range allThemes {
		if n == current {
			return allThemes[(i+1)%len(allThemes)]
		}
	}
	return ThemeAuto
}
