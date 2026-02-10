package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewAppStartsOnMenuScreen(t *testing.T) {
	t.Parallel()

	model := NewApp("/tmp/test")
	if got := model.Screen(); got != MenuScreen {
		t.Fatalf("expected app to start on menu screen, got %v", got)
	}
}

func TestAppRoutesMenuSelectionToWizard(t *testing.T) {
	t.Parallel()

	model := NewApp("/tmp/test")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(AppModel)

	if got := next.Screen(); got != WizardScreen {
		t.Fatalf("expected wizard screen after selecting audit, got %v", got)
	}
}

func TestAppBackFromWizardModeReturnsToMenu(t *testing.T) {
	t.Parallel()

	model := NewApp("/tmp/test")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(AppModel)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(AppModel)

	if got := model.Screen(); got != MenuScreen {
		t.Fatalf("expected menu screen after pressing esc on mode step, got %v", got)
	}
}

func TestAppRoutesLaunchCompletionToDashboard(t *testing.T) {
	t.Parallel()

	model := NewApp("/tmp/test")
	model.wizard = model.wizard.SetLaunchDelay(time.Millisecond)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(AppModel)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(AppModel)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(AppModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(AppModel)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(AppModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(AppModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(AppModel)

	if got := model.Screen(); got != WizardScreen {
		t.Fatalf("expected to still be on wizard before launch completion, got %v", got)
	}

	updated, _ = model.Update(auditWizardLaunchMsg{})
	model = updated.(AppModel)

	if got := model.Screen(); got != DashboardScreen {
		t.Fatalf("expected dashboard screen after launch completion, got %v", got)
	}
}

func TestDashboardBackNavigatesToMenu(t *testing.T) {
	t.Parallel()

	model := NewApp("/tmp/test")
	updated, _ := model.Update(AppNavigateMsg{Screen: DashboardScreen})
	model = updated.(AppModel)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(AppModel)
	if cmd == nil {
		t.Fatal("expected dashboard to return navigation command")
	}

	navMsg, ok := cmd().(AppNavigateMsg)
	if !ok {
		t.Fatalf("expected AppNavigateMsg from command, got %T", cmd())
	}

	updated, _ = model.Update(navMsg)
	model = updated.(AppModel)

	if got := model.Screen(); got != MenuScreen {
		t.Fatalf("expected menu screen after dashboard back, got %v", got)
	}
}

func TestAppQuitKeyReturnsQuitCommand(t *testing.T) {
	t.Parallel()

	model := NewApp("/tmp/test")
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}

	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", cmd())
	}
}
