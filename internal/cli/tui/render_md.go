package tui

import (
	"hash/fnv"
	"strconv"
	"sync"

	"github.com/charmbracelet/glamour"
)

var (
	mdCache sync.Map // key string → string
)

// renderMarkdown converts markdown source to ANSI-rendered text using glamour.
// Cached per (content, width, theme). Width is the target column count.
// Falls back to the raw input on any glamour error.
func renderMarkdown(input string, width int, theme string) string {
	if input == "" || width <= 0 {
		return input
	}
	key := mdKey(input, width, theme)
	if v, ok := mdCache.Load(key); ok {
		return v.(string)
	}

	style := theme
	switch theme {
	case ThemeAuto, "":
		style = "auto"
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	)
	if err != nil {
		return input
	}
	out, err := r.Render(input)
	if err != nil {
		return input
	}
	mdCache.Store(key, out)
	return out
}

func mdKey(content string, width int, theme string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(content))
	return strconv.FormatUint(h.Sum64(), 36) + "|" + strconv.Itoa(width) + "|" + theme
}

// invalidateMarkdownCache clears the cache (for theme switch).
func invalidateMarkdownCache() {
	mdCache = sync.Map{}
}
