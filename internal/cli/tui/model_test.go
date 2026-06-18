package tui

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

func TestProjectShortcutSetsProjectFilterFromProjectsList(t *testing.T) {
	projectID := uuid.New()
	m := newRootModel(nil)
	m.focus = paneList
	m.list = newList(CatProjects, 80, 20)
	m.list.resetStack(CatProjects, CatProjects.Label())
	m.list.setProjects([]models.Project{{
		ID:          projectID,
		Name:        "ramorie-cli",
		Description: "CLI project",
	}})

	cmd := m.handleProjectShortcut()
	if cmd == nil {
		t.Fatal("project shortcut should return a reload command")
	}
	if m.projectID != projectID.String() {
		t.Fatalf("projectID = %q, want %q", m.projectID, projectID.String())
	}
	if m.projectName != "ramorie-cli" {
		t.Fatalf("projectName = %q", m.projectName)
	}
	if got := m.sidebar.selected(); got != CatTasks {
		t.Fatalf("selected category = %v, want %v", got, CatTasks)
	}
	if !strings.Contains(m.statusMsg, "project filter") {
		t.Fatalf("statusMsg should mention project filter, got %q", m.statusMsg)
	}
}

func TestClearProjectFilterResetsScope(t *testing.T) {
	m := newRootModel(nil)
	m.list = newList(CatTasks, 80, 20)
	m.list.resetStack(CatTasks, CatTasks.Label())
	m.loadedCat = CatTasks
	m.projectID = uuid.NewString()
	m.projectName = "ramorie-cli"

	cmd := m.clearProjectFilter()
	if cmd == nil {
		t.Fatal("clear project filter should return a reload command")
	}
	if m.projectID != "" || m.projectName != "" {
		t.Fatalf("project filter not cleared: id=%q name=%q", m.projectID, m.projectName)
	}
	if !strings.Contains(m.statusMsg, "cleared") {
		t.Fatalf("statusMsg should mention cleared filter, got %q", m.statusMsg)
	}
}

func TestStatusBarAlwaysIncludesShortcuts(t *testing.T) {
	m := newRootModel(nil)
	m.width = 120
	m.focus = paneList
	m.list = newList(CatTasks, 80, 20)
	m.list.resetStack(CatTasks, CatTasks.Label())

	got := stripANSITUI(m.renderStatusBar())
	// The footer is context-aware; in list focus it advertises movement,
	// filtering, and the global quit hint at minimum.
	for _, want := range []string{"move", "filter", "quit"} {
		if !strings.Contains(got, want) {
			t.Fatalf("status bar %q does not contain %q", got, want)
		}
	}
}
