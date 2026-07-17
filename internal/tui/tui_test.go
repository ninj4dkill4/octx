package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/ninj4dkill4/octx/internal/config"
)

func TestViewShowsUnsetProfilesOption(t *testing.T) {
	m := model{
		projects: []config.Project{{Code: "core"}},
	}

	view := m.View()
	if !strings.Contains(view, "core") {
		t.Fatalf("view missing project option: %s", view)
	}
	if !strings.Contains(view, "unset") {
		t.Fatalf("view missing unset option: %s", view)
	}
	if !strings.Contains(view, "Use arrow keys") {
		t.Fatalf("view missing arrow key hint: %s", view)
	}
	if strings.Index(view, "core") > strings.Index(view, "unset") {
		t.Fatalf("unset option should be below project options: %s", view)
	}
}

func TestRenderProjectNameUsesProjectColor(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	rendered := renderProjectName(config.Project{Code: "core", Color: "#22c55e"}, false)
	if !strings.Contains(rendered, "\x1b[") {
		t.Fatalf("expected colored output, got %q", rendered)
	}
	if !strings.Contains(rendered, "core") {
		t.Fatalf("expected project code in output, got %q", rendered)
	}
}

func TestRenderProjectNameKeepsDefaultWithoutColor(t *testing.T) {
	rendered := renderProjectName(config.Project{Code: "core"}, false)
	if rendered != "core" {
		t.Fatalf("rendered = %q, want core", rendered)
	}
}

func TestEnterOnLastItemPicksClear(t *testing.T) {
	m := model{
		projects: []config.Project{{Code: "core"}},
		cursor:   1,
		pickOnly: true,
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(model)
	if updated.picked == nil || !updated.picked.Clear {
		t.Fatalf("enter on first item should pick clear: %#v", updated.picked)
	}
}

func TestCursorWrapsAround(t *testing.T) {
	projects := []config.Project{
		{Code: "core"},
		{Code: "pay"},
	}

	up, _ := model{projects: projects, cursor: 0}.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got := up.(model).cursor; got != 2 {
		t.Fatalf("up from first cursor = %d, want 2", got)
	}

	down, _ := model{projects: projects, cursor: 2}.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := down.(model).cursor; got != 0 {
		t.Fatalf("down from unset cursor = %d, want 0", got)
	}
}

func TestInitialCursorPrefersCurrentProject(t *testing.T) {
	projects := []config.Project{
		{Code: "core"},
		{Code: "pay"},
	}

	if got := initialCursor(projects, "pay"); got != 1 {
		t.Fatalf("cursor = %d, want 1", got)
	}
}

func TestInitialCursorPrefersUnsetState(t *testing.T) {
	projects := []config.Project{
		{Code: "core"},
		{Code: "pay"},
	}

	if got := initialCursor(projects, config.UnsetProjectCode); got != 2 {
		t.Fatalf("cursor = %d, want 2", got)
	}
}

func TestInitialCursorDefaultsToClearForUnknownState(t *testing.T) {
	projects := []config.Project{
		{Code: "core"},
		{Code: "pay"},
	}

	if got := initialCursor(projects, "missing"); got != 2 {
		t.Fatalf("cursor = %d, want 2", got)
	}
}

func TestInitialCursorDefaultsToClearWhenNoProjects(t *testing.T) {
	if got := initialCursor(nil, "missing"); got != 0 {
		t.Fatalf("cursor = %d, want 0", got)
	}
}
