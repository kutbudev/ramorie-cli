package mcpinstall

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---- styles -----------------------------------------------------------------

var (
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a87ff")).Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#8e8e8e"))
	goodStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#7ed491"))
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#e6b450"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6e6e"))
	focusStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a87ff")).Bold(true)
	checkboxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a87ff"))
	diffPlus      = lipgloss.NewStyle().Foreground(lipgloss.Color("#7ed491"))
	diffMinus     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6e6e"))
	diffCtx       = lipgloss.NewStyle().Foreground(lipgloss.Color("#8e8e8e"))
)

// ---- model ------------------------------------------------------------------

type stage int

const (
	stageSelect stage = iota
	stageScope
	stagePreview
	stageWriting
	stageSmoke
	stageDone
)

type clientItem struct {
	adapter   ClientAdapter
	detection DetectionResult
	selected  bool
	scope     Scope // chosen scope once confirmed

	// filled after Install/Smoke
	diff        Diff
	installErr  error
	smokeResult SmokeResult
}

type model struct {
	stage    stage
	items    []*clientItem
	cursor   int    // index into items or, in stageScope, into scopeChoices of currentScopeTarget
	binary   string // ramorie binary path
	args     []string
	err      error
	width    int

	// stageScope bookkeeping
	scopeTargetIdx int // which selected item we're asking scope for
	scopeChoices   []Scope
	scopeCursor    int

	// stagePreview bookkeeping
	previewCursor int // which selected client's diff we're viewing

	// Signals that a phase is complete and we can advance. Messages below
	// carry the actual results.
	writeResults []writeDone
	smokeResults []smokeDone
}

type writeDone struct {
	idx  int
	diff Diff
	err  error
}

type smokeDone struct {
	idx    int
	result SmokeResult
}

// Run launches the TUI. binary is the path to `ramorie` (usually
// os.Executable()) and args are the stdio serve args (default [mcp serve]).
func Run(binary string, args []string) error {
	if len(args) == 0 {
		args = []string{"mcp", "serve"}
	}

	items := []*clientItem{}
	for _, a := range Registry() {
		d := a.Detect()
		items = append(items, &clientItem{
			adapter:   a,
			detection: d,
			selected:  d.Installed, // pre-select detected clients for 1-keypress installs
		})
	}

	m := &model{
		stage:  stageSelect,
		items:  items,
		binary: binary,
		args:   args,
	}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func (m *model) Init() tea.Cmd { return nil }

// Update dispatches per-stage, with a global quit binding.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	switch m.stage {
	case stageSelect:
		return m.updateSelect(msg)
	case stageScope:
		return m.updateScope(msg)
	case stagePreview:
		return m.updatePreview(msg)
	case stageWriting:
		return m.updateWriting(msg)
	case stageSmoke:
		return m.updateSmoke(msg)
	case stageDone:
		return m.updateDone(msg)
	}
	return m, nil
}

func (m *model) View() string {
	header := titleStyle.Render("🧠 Ramorie — MCP Install") + "\n" +
		dimStyle.Render("  Configure the Ramorie MCP server in your AI clients.") + "\n\n"
	switch m.stage {
	case stageSelect:
		return header + m.viewSelect()
	case stageScope:
		return header + m.viewScope()
	case stagePreview:
		return header + m.viewPreview()
	case stageWriting:
		return header + m.viewWriting()
	case stageSmoke:
		return header + m.viewSmoke()
	case stageDone:
		return header + m.viewDone()
	}
	return ""
}

// ---- stage 1: select clients ------------------------------------------------

func (m *model) updateSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case " ", "space", "x":
		m.items[m.cursor].selected = !m.items[m.cursor].selected
	case "a":
		// toggle all detected
		allOn := true
		for _, it := range m.items {
			if it.detection.Installed && !it.selected {
				allOn = false
			}
		}
		for _, it := range m.items {
			if it.detection.Installed {
				it.selected = !allOn
			}
		}
	case "enter":
		if m.countSelected() == 0 {
			m.err = fmt.Errorf("pick at least one client (space to toggle)")
			return m, nil
		}
		m.err = nil
		m.beginScopeStage()
	}
	return m, nil
}

func (m *model) countSelected() int {
	c := 0
	for _, it := range m.items {
		if it.selected {
			c++
		}
	}
	return c
}

func (m *model) viewSelect() string {
	var b strings.Builder
	b.WriteString(focusStyle.Render("  Select clients — ") + dimStyle.Render("space toggles, [a] toggles detected, [enter] continue, [q] quit") + "\n\n")

	for i, it := range m.items {
		cur := "  "
		if i == m.cursor {
			cur = focusStyle.Render("▸ ")
		}
		box := "[ ]"
		if it.selected {
			box = checkboxStyle.Render("[✓]")
		}
		name := it.adapter.Name()
		if i == m.cursor {
			name = focusStyle.Render(name)
		}
		badge := dimStyle.Render("— " + it.detection.Detail)
		if it.detection.Installed {
			badge = goodStyle.Render("— " + it.detection.Detail)
		}
		fmt.Fprintf(&b, "%s%s  %-22s  %s\n", cur, box, name, badge)
	}
	if m.err != nil {
		fmt.Fprintf(&b, "\n%s\n", errStyle.Render("  "+m.err.Error()))
	}
	return b.String()
}

// ---- stage 2: scopes --------------------------------------------------------

func (m *model) beginScopeStage() {
	// Seed scope targets: each selected adapter whose SupportedScopes has >1
	// option needs a prompt. Others auto-pick.
	m.scopeTargetIdx = -1
	m.advanceToNextScopeTarget()
}

func (m *model) advanceToNextScopeTarget() {
	for i := m.scopeTargetIdx + 1; i < len(m.items); i++ {
		it := m.items[i]
		if !it.selected {
			continue
		}
		scopes := it.adapter.SupportedScopes()
		if len(scopes) == 1 {
			// Auto-pick the only supported scope.
			it.scope = scopes[0]
			continue
		}
		m.scopeTargetIdx = i
		m.scopeChoices = scopes
		m.scopeCursor = 0
		m.stage = stageScope
		return
	}
	// All scopes resolved → move to preview.
	m.beginPreviewStage()
}

func (m *model) updateScope(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "up", "k":
		if m.scopeCursor > 0 {
			m.scopeCursor--
		}
	case "down", "j":
		if m.scopeCursor < len(m.scopeChoices)-1 {
			m.scopeCursor++
		}
	case "enter":
		m.items[m.scopeTargetIdx].scope = m.scopeChoices[m.scopeCursor]
		m.advanceToNextScopeTarget()
	}
	return m, nil
}

func (m *model) viewScope() string {
	it := m.items[m.scopeTargetIdx]
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", focusStyle.Render("  Scope for"), focusStyle.Render(it.adapter.Name()))
	b.WriteString(dimStyle.Render("  [↑/↓ or j/k] move · [enter] pick · [q] quit") + "\n\n")

	for i, s := range m.scopeChoices {
		cur := "  "
		if i == m.scopeCursor {
			cur = focusStyle.Render("▸ ")
		}
		label := string(s)
		hint := ""
		switch s {
		case ScopeUser:
			hint = "global — applies to all directories"
		case ScopeProject:
			hint = "current directory only"
		}
		// Show config path for clarity.
		p, _ := it.adapter.ConfigPath(s)
		fmt.Fprintf(&b, "%s %-8s  %s\n    %s\n", cur, label, dimStyle.Render(hint), dimStyle.Render(p))
	}
	return b.String()
}

// ---- stage 3: preview -------------------------------------------------------

func (m *model) beginPreviewStage() {
	m.previewCursor = 0
	m.stage = stagePreview
	// Build the diffs by running Install in dry-run… but adapters mutate.
	// Instead we run Install but into a tmp snapshot — simpler: read current
	// config + predict after. Given Install is idempotent and we're going
	// to run it anyway on write stage, we build preview by calling it here
	// and reading the "After" string — BUT that writes. So: build a fake
	// diff without writing.
	for _, it := range m.items {
		if !it.selected {
			continue
		}
		path, err := it.adapter.ConfigPath(it.scope)
		if err != nil {
			it.installErr = err
			continue
		}
		before, err := readJSONObject(path)
		if err != nil {
			it.installErr = err
			continue
		}
		beforeStr := prettyPrint(before)
		after := predictAfter(it.adapter.ID(), cloneMap(before), m.binary, m.args)
		it.diff = Diff{Path: path, Before: beforeStr, After: prettyPrint(after)}
	}
}

// predictAfter returns the config map that WOULD result from installing —
// without touching disk. Mirrors the Install methods' mutations.
func predictAfter(clientID string, raw map[string]any, command string, args []string) map[string]any {
	entry := standardServerEntry(command, args, nil)
	switch clientID {
	case "vscode":
		return upsertVSCodeServer(raw, ServerName, vscodeServerEntry(command, args, nil))
	case "zed":
		return upsertZedServer(raw, ServerName, zedServerEntry(command, args, nil))
	default:
		return upsertMCPServer(raw, ServerName, entry)
	}
}

func (m *model) updatePreview(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	selected := m.selectedIndices()
	switch km.String() {
	case "up", "k", "left", "h":
		if m.previewCursor > 0 {
			m.previewCursor--
		}
	case "down", "j", "right", "l":
		if m.previewCursor < len(selected)-1 {
			m.previewCursor++
		}
	case "y", "enter":
		m.stage = stageWriting
		return m, m.writeAllCmd()
	case "n", "esc":
		// Back to select stage
		m.stage = stageSelect
	}
	return m, nil
}

func (m *model) selectedIndices() []int {
	out := []int{}
	for i, it := range m.items {
		if it.selected {
			out = append(out, i)
		}
	}
	return out
}

func (m *model) viewPreview() string {
	selected := m.selectedIndices()
	if len(selected) == 0 {
		return "  nothing selected\n"
	}
	idx := selected[m.previewCursor]
	it := m.items[idx]

	var b strings.Builder
	tabs := ""
	for i, s := range selected {
		n := m.items[s].adapter.Name()
		if i == m.previewCursor {
			tabs += focusStyle.Render("["+n+"] ")
		} else {
			tabs += dimStyle.Render(" "+n+"  ")
		}
	}
	b.WriteString("  " + tabs + "\n")
	b.WriteString(dimStyle.Render("  [←/→] switch · [y] write all · [n/esc] back · [q] quit") + "\n\n")
	b.WriteString("  " + focusStyle.Render("Path: ") + it.diff.Path + "\n")
	b.WriteString("  " + focusStyle.Render("Scope: ") + string(it.scope) + "\n\n")

	if it.installErr != nil {
		b.WriteString(errStyle.Render("  "+it.installErr.Error()) + "\n")
		return b.String()
	}
	b.WriteString(renderDiff(it.diff.Before, it.diff.After))
	return b.String()
}

// renderDiff produces a basic line-by-line prefixing diff.
func renderDiff(before, after string) string {
	oldLines := strings.Split(before, "\n")
	newLines := strings.Split(after, "\n")
	var b strings.Builder
	// Heuristic: show file as green (+) when it didn't exist or was empty.
	if strings.TrimSpace(before) == "" || before == "{}" {
		for _, l := range newLines {
			fmt.Fprintf(&b, "  %s\n", diffPlus.Render("+ "+l))
		}
		return b.String()
	}
	// Minimal diff: mark lines not in oldSet as +, lines not in newSet as -.
	oldSet := map[string]bool{}
	for _, l := range oldLines {
		oldSet[l] = true
	}
	newSet := map[string]bool{}
	for _, l := range newLines {
		newSet[l] = true
	}
	for _, l := range oldLines {
		if !newSet[l] {
			fmt.Fprintf(&b, "  %s\n", diffMinus.Render("- "+l))
		} else {
			fmt.Fprintf(&b, "  %s\n", diffCtx.Render("  "+l))
		}
	}
	for _, l := range newLines {
		if !oldSet[l] {
			fmt.Fprintf(&b, "  %s\n", diffPlus.Render("+ "+l))
		}
	}
	return b.String()
}

// ---- stage 4: writing -------------------------------------------------------

func (m *model) writeAllCmd() tea.Cmd {
	return func() tea.Msg {
		done := []writeDone{}
		for i, it := range m.items {
			if !it.selected {
				continue
			}
			if it.installErr != nil {
				done = append(done, writeDone{idx: i, err: it.installErr})
				continue
			}
			diff, err := it.adapter.Install(InstallOptions{
				Scope:   it.scope,
				Command: m.binary,
				Args:    m.args,
			})
			done = append(done, writeDone{idx: i, diff: diff, err: err})
		}
		return writeAllMsg{done}
	}
}

type writeAllMsg struct{ results []writeDone }

func (m *model) updateWriting(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case writeAllMsg:
		m.writeResults = msg.results
		for _, r := range msg.results {
			if r.err != nil {
				m.items[r.idx].installErr = r.err
				continue
			}
			m.items[r.idx].diff = r.diff
		}
		m.stage = stageSmoke
		return m, m.smokeAllCmd()
	}
	return m, nil
}

func (m *model) viewWriting() string {
	return "  " + warnStyle.Render("Writing configs…") + "\n"
}

// ---- stage 5: smoke test ----------------------------------------------------

func (m *model) smokeAllCmd() tea.Cmd {
	return func() tea.Msg {
		// Single smoke against the binary — if it works once, it works for
		// every client pointing at the same binary + args. This keeps the
		// TUI snappy (one 1-2s probe instead of 6).
		res := SmokeTest(context.Background(), m.binary, m.args, nil)
		out := []smokeDone{}
		for i, it := range m.items {
			if !it.selected || it.installErr != nil {
				continue
			}
			out = append(out, smokeDone{idx: i, result: res})
		}
		return smokeAllMsg{out}
	}
}

type smokeAllMsg struct{ results []smokeDone }

func (m *model) updateSmoke(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case smokeAllMsg:
		m.smokeResults = msg.results
		for _, r := range msg.results {
			m.items[r.idx].smokeResult = r.result
		}
		m.stage = stageDone
	}
	return m, nil
}

func (m *model) viewSmoke() string {
	return "  " + warnStyle.Render("Running stdio smoke test…") + "\n"
}

// ---- stage 6: done / summary ------------------------------------------------

func (m *model) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) viewDone() string {
	var b strings.Builder
	b.WriteString("  " + titleStyle.Render("Done.") + "\n\n")
	for _, it := range m.items {
		if !it.selected {
			continue
		}
		prefix := goodStyle.Render("✔")
		note := fmt.Sprintf("%s (%s)", it.diff.Path, it.scope)
		if it.installErr != nil {
			prefix = errStyle.Render("✗")
			note = it.installErr.Error()
		}
		fmt.Fprintf(&b, "  %s %-20s %s\n", prefix, it.adapter.Name(), dimStyle.Render(note))
		if it.smokeResult.OK {
			fmt.Fprintf(&b, "      %s %d tools, version %s (%s)\n",
				goodStyle.Render("smoke ok —"), it.smokeResult.Tools, it.smokeResult.Version, it.smokeResult.Duration.Round(1_000_000))
		} else if it.smokeResult.Err != nil {
			fmt.Fprintf(&b, "      %s %s\n", warnStyle.Render("smoke failed —"), it.smokeResult.Err.Error())
		}
	}
	b.WriteString("\n  " + dimStyle.Render("[any key] exit") + "\n")
	return b.String()
}

// ---- helpers ----------------------------------------------------------------

// BinaryPath returns the absolute path to the running ramorie binary, which
// is what we embed in the MCP config. Falls back to "ramorie" on PATH if
// os.Executable fails.
func BinaryPath() string {
	if exe, err := os.Executable(); err == nil {
		abs, err := filepath.Abs(exe)
		if err == nil {
			return abs
		}
		return exe
	}
	return "ramorie"
}

// RunUninstall is a non-interactive helper that strips the Ramorie server
// entry from every installed client it finds. Idempotent — reports what it
// touched and exits.
func RunUninstall() error {
	found := 0
	for _, a := range Registry() {
		for _, s := range a.SupportedScopes() {
			if !a.IsInstalled(s) {
				continue
			}
			if _, err := a.Uninstall(s); err != nil {
				fmt.Printf("  ✗ %s (%s): %v\n", a.Name(), s, err)
				continue
			}
			path, _ := a.ConfigPath(s)
			fmt.Printf("  ✓ removed Ramorie from %s (%s) — %s\n", a.Name(), s, path)
			found++
		}
	}
	if found == 0 {
		fmt.Println("  No Ramorie MCP entries found. Nothing to do.")
	}
	return nil
}
