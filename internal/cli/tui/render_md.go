package tui

import (
	"hash/fnv"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

// sp/bp/up are pointer helpers for building an ansi.StyleConfig literal.
func sp(s string) *string { return &s }
func bp(b bool) *bool     { return &b }
func up(u uint) *uint     { return &u }

// baseStyleConfig returns the bundled glamour style for the named theme, used
// as the foundation we then tint with the app accent.
func baseStyleConfig(theme string) ansi.StyleConfig {
	switch theme {
	case ThemeLight:
		return styles.LightStyleConfig
	case ThemeDracula:
		return styles.DraculaStyleConfig
	case ThemeNotty:
		return styles.NoTTYStyleConfig
	default: // auto + dark
		return styles.DarkStyleConfig
	}
}

// buildAccentStyle clones the theme's base style and tints headings, links,
// strong text and inline code with the resolved accent, and drops glamour's
// default document margin so a narrow detail pane keeps every column.
func buildAccentStyle(theme, accent string) ansi.StyleConfig {
	c := baseStyleConfig(theme)
	c.Document.Margin = up(0)
	c.Heading.Color = sp(accent)
	c.Heading.Bold = bp(true)
	c.H1.Color = sp(accent)
	c.H2.Color = sp(accent)
	c.Link.Color = sp(accent)
	c.Link.Underline = bp(true)
	c.LinkText.Color = sp(accent)
	c.Strong.Color = sp(accent)
	c.Code.Color = sp(accent)
	return c
}

// Markdown rendering hot path notes:
//
//   - glamour.NewTermRenderer is expensive (chroma init + style parse). We
//     keep ONE renderer per (theme, bucketWidth) and reuse it forever.
//   - The cache key is (content hash, bucketWidth, theme). bucketWidth rounds
//     width down to the nearest 5 columns, which collapses near-identical
//     widths into one cache slot — almost-zero quality loss, much higher hit
//     rate when the user resizes by a single column.
//   - For trivial content (no markdown markers, short length) we bypass
//     glamour entirely and return the input verbatim. This keeps cursor
//     movement instant on plain-text descriptions.
var (
	mdCache    sync.Map // string → string
	rendererMu sync.Mutex
	renderers  = map[string]*glamour.TermRenderer{} // theme|bucketWidth → renderer
)

const widthBucket = 5

// bucketize rounds width DOWN to the nearest widthBucket multiple.
func bucketize(w int) int {
	if w <= 0 {
		return 0
	}
	return (w / widthBucket) * widthBucket
}

// renderMarkdown converts markdown source to ANSI-rendered text using glamour.
// Cached aggressively. Falls back to the raw input on any glamour error.
func renderMarkdown(input string, width int, theme, accent string) string {
	if input == "" || width <= 0 {
		return input
	}
	if !worthRendering(input) {
		return input
	}
	bw := bucketize(width)
	if bw < widthBucket {
		bw = widthBucket
	}

	key := mdKey(input, bw, theme, accent)
	if v, ok := mdCache.Load(key); ok {
		return v.(string)
	}

	r := getRenderer(theme, accent, bw)
	if r == nil {
		return input
	}
	out, err := r.Render(input)
	if err != nil {
		return input
	}
	mdCache.Store(key, out)
	return out
}

// worthRendering returns false for content that gains nothing from glamour:
// short, no markdown markers, no code fences, no headings.
func worthRendering(s string) bool {
	if len(s) < 80 {
		return false
	}
	// Cheap marker scan — any of these means glamour adds value.
	return strings.ContainsAny(s, "#*`>|-") || strings.Contains(s, "```") ||
		strings.Contains(s, "http://") || strings.Contains(s, "https://")
}

func getRenderer(theme, accent string, bw int) *glamour.TermRenderer {
	cacheKey := theme + "|" + accent + "|" + strconv.Itoa(bw)

	rendererMu.Lock()
	defer rendererMu.Unlock()
	if r, ok := renderers[cacheKey]; ok {
		return r
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(buildAccentStyle(theme, accent)),
		glamour.WithWordWrap(bw),
		glamour.WithEmoji(),
	)
	if err != nil {
		return nil
	}
	renderers[cacheKey] = r
	return r
}

func mdKey(content string, width int, theme, accent string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(content))
	return strconv.FormatUint(h.Sum64(), 36) + "|" + strconv.Itoa(width) + "|" + theme + "|" + accent
}

// invalidateMarkdownCache clears the rendered output cache AND the renderer
// pool — called when the user cycles theme.
func invalidateMarkdownCache() {
	mdCache = sync.Map{}
	rendererMu.Lock()
	renderers = map[string]*glamour.TermRenderer{}
	rendererMu.Unlock()
}
