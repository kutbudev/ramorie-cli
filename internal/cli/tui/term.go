package tui

import (
	"os"
	"regexp"
)

// terminalCaps describes what tricks the active terminal supports. Detected
// once at startup from environment variables. Used to gracefully degrade
// OSC 8 hyperlinks and (future) image protocols.
type terminalCaps struct {
	iTerm     bool
	wezTerm   bool
	kitty     bool
	apple     bool
	tmux      bool
	osc8      bool // hyperlinks
	truecolor bool
}

func detectTerminal() terminalCaps {
	var c terminalCaps
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		c.iTerm = true
	}
	if os.Getenv("TERM_PROGRAM") == "WezTerm" || os.Getenv("WEZTERM_PANE") != "" {
		c.wezTerm = true
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("TERM") == "xterm-kitty" {
		c.kitty = true
	}
	if os.Getenv("TERM_PROGRAM") == "Apple_Terminal" {
		c.apple = true
	}
	if os.Getenv("TMUX") != "" {
		c.tmux = true
	}
	// OSC 8 — supported by iTerm2, WezTerm, Kitty (recent), Alacritty, GNOME Terminal.
	c.osc8 = c.iTerm || c.wezTerm || c.kitty
	if cs := os.Getenv("COLORTERM"); cs == "truecolor" || cs == "24bit" {
		c.truecolor = true
	}
	return c
}

// hyperlink wraps `text` in OSC 8 escape so capable terminals make it clickable.
// In non-capable terminals returns plain "text (url)".
func (c terminalCaps) hyperlink(url, text string) string {
	if !c.osc8 || url == "" {
		if url != "" && text != url {
			return text + " (" + url + ")"
		}
		return text
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// bareURLRegex matches bare http(s) URLs in plain text (excluding trailing
// punctuation that's likely sentence-related, not part of the URL).
var bareURLRegex = regexp.MustCompile(`https?://[^\s)\]]+`)

// linkifyText finds bare http(s) URLs in text and wraps each in OSC 8 if
// supported by the terminal. Markdown-style links `[text](url)` are not
// touched (glamour handles those).
func linkifyText(text string, caps terminalCaps) string {
	if !caps.osc8 || text == "" {
		return text
	}
	return bareURLRegex.ReplaceAllStringFunc(text, func(url string) string {
		return caps.hyperlink(url, url)
	})
}
