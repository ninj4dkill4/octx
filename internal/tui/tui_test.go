package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ninj4dkill4/octx/internal/config"
)

func TestViewShowsUnsetProfilesOption(t *testing.T) {
	m := model{
		projects: []config.Project{{Code: "core"}},
	}

	view := m.View()
	if !strings.Contains(view, "unset profiles") {
		t.Fatalf("view missing unset profiles option: %s", view)
	}
	if !strings.Contains(view, "core") {
		t.Fatalf("view missing project option: %s", view)
	}
}

func TestEnterOnFirstItemPicksClear(t *testing.T) {
	m := model{
		projects: []config.Project{{Code: "core"}},
		pickOnly: true,
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(model)
	if updated.picked == nil || !updated.picked.Clear {
		t.Fatalf("enter on first item should pick clear: %#v", updated.picked)
	}
}
