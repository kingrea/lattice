package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuSelectAudit(t *testing.T) {
	t.Parallel()

	model := NewMenuModel()
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !model.Confirmed() {
		t.Fatal("expected menu to be confirmed after pressing enter")
	}

	if got := model.Action(); got != MenuActionOpenAuditWizard {
		t.Fatalf("expected audit wizard action, got %v", got)
	}
}

func TestMenuNavigationWrapsAndSelectQuit(t *testing.T) {
	t.Parallel()

	model := NewMenuModel()
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})

	if got := model.Cursor(); got != 1 {
		t.Fatalf("expected cursor to wrap to last menu item, got %d", got)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := model.Action(); got != MenuActionQuit {
		t.Fatalf("expected quit action from second menu item, got %v", got)
	}
}

func TestMenuViewShowsBrandingAndOptions(t *testing.T) {
	t.Parallel()

	view := NewMenuModel().View()

	for _, fragment := range []string{"LATTICE", "Main Menu", "Audit", "Quit"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("expected view to include %q", fragment)
		}
	}
}
