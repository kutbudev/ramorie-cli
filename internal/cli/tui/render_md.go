package tui

import (
	"hash/fnv"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
)

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
func renderMarkdown(input string, width int, theme string) string {
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

	key := mdKey(input, bw, theme)
	if v, ok := mdCache.Load(key); ok {
		return v.(string)
	}

	r := getRenderer(theme, bw)
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

func getRenderer(theme string, bw int) *glamour.TermRenderer {
	style := theme
	switch theme {
	case ThemeAuto, "":
		style = "auto"
	}
	cacheKey := style + "|" + strconv.Itoa(bw)

	rendererMu.Lock()
	defer rendererMu.Unlock()
	if r, ok := renderers[cacheKey]; ok {
		return r
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(bw),
		glamour.WithEmoji(),
	)
	if err != nil {
		return nil
	}
	renderers[cacheKey] = r
	return r
}

func mdKey(content string, width int, theme string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(content))
	return strconv.FormatUint(h.Sum64(), 36) + "|" + strconv.Itoa(width) + "|" + theme
}

// invalidateMarkdownCache clears the rendered output cache AND the renderer
// pool — called when the user cycles theme.
func invalidateMarkdownCache() {
	mdCache = sync.Map{}
	rendererMu.Lock()
	renderers = map[string]*glamour.TermRenderer{}
	rendererMu.Unlock()
}
