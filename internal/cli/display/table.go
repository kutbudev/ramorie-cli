// Package display — table.go provides a thin responsive wrapper around
// lipgloss/table so list commands ('task list', 'memory list',
// 'project list') auto-fit the current terminal width and gracefully drop
// less-important columns when space is tight, instead of wrapping into
// garbage.
package display

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// Column describes one column for NewResponsiveTable.
//
// Min is the smallest content width (excluding border + padding) at which
// the column is still considered acceptable; if the terminal can't fit the
// column at this width it is dropped from the rendered table.
//
// Weight is the relative share of LEFTOVER width distributed after every
// kept column has been allocated its Min. A Weight of 0 marks the column as
// fixed (e.g. status icon, short id) — it gets exactly Min and is never a
// candidate for dropping. Weight > 0 columns are dropped rightmost-first
// when the terminal is too narrow.
type Column struct {
	Title  string
	Min    int
	Weight int
}

// NewResponsiveTable builds a lipgloss table sized for the current terminal.
//
// Cell rows must be passed in the SAME order the columns are listed; cells
// for columns that get dropped are skipped silently. Returns the rendered
// string ready to print.
func NewResponsiveTable(cols []Column, rows [][]string) string {
	width := TerminalWidth()
	if width <= 0 {
		width = 100
	}

	// Borders + padding eat ~3 cols per column + 2 outer; small fudge.
	borderOverhead := 3*len(cols) + 6

	// Start with every column kept; drop weighted columns rightmost-first
	// until the sum of mins fits.
	keptIdx := make([]int, 0, len(cols))
	for i := range cols {
		keptIdx = append(keptIdx, i)
	}
	avail := width - borderOverhead
	for {
		need := 0
		for _, i := range keptIdx {
			need += cols[i].Min
		}
		if need <= avail || len(keptIdx) <= 1 {
			break
		}
		dropAt := -1
		for j := len(keptIdx) - 1; j >= 0; j-- {
			if cols[keptIdx[j]].Weight > 0 {
				dropAt = j
				break
			}
		}
		if dropAt == -1 {
			// Only fixed columns left — nothing more we can drop.
			break
		}
		keptIdx = append(keptIdx[:dropAt], keptIdx[dropAt+1:]...)
	}

	// Distribute leftover width across remaining weighted columns.
	totalMin := 0
	totalWeight := 0
	for _, i := range keptIdx {
		totalMin += cols[i].Min
		totalWeight += cols[i].Weight
	}
	leftover := avail - totalMin
	if leftover < 0 {
		leftover = 0
	}
	widths := make([]int, len(keptIdx))
	for j, i := range keptIdx {
		w := cols[i].Min
		if totalWeight > 0 && cols[i].Weight > 0 {
			w += (leftover * cols[i].Weight) / totalWeight
		}
		widths[j] = w
	}

	// Build header + filtered rows. Walk the original column order so a
	// row's cell slice can be longer than len(keptIdx).
	headers := make([]string, len(keptIdx))
	for j, i := range keptIdx {
		headers[j] = cols[i].Title
	}
	keepSet := make(map[int]bool, len(keptIdx))
	for _, i := range keptIdx {
		keepSet[i] = true
	}
	filtered := make([][]string, 0, len(rows))
	for _, row := range rows {
		out := make([]string, 0, len(keptIdx))
		for i, cell := range row {
			if keepSet[i] {
				out = append(out, cell)
			}
		}
		filtered = append(filtered, out)
	}

	// lipgloss.Width includes Padding, so add the 2 chars of horizontal
	// padding back into the per-column width — otherwise content is forced
	// into a column width-2 chars narrower than Min and wraps awkwardly
	// (e.g. a 3-char "[H]" badge renders one char per row).
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorDim)).
		Headers(headers...).
		Rows(filtered...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().Padding(0, 1)
			if col >= 0 && col < len(widths) {
				s = s.Width(widths[col] + 2)
			}
			if row == table.HeaderRow {
				return s.Bold(true).Foreground(ColorAccent)
			}
			return s
		})
	return t.Render()
}
