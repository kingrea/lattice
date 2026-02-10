package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAuditWizardEndToEndFlow(t *testing.T) {
	t.Parallel()

	model := NewAuditWizardModel().SetLaunchDelay(time.Millisecond)

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := model.Step(); got != AuditWizardStepTypes {
		t.Fatalf("expected step types, got %v", got)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := model.Step(); got != AuditWizardStepAgentCount {
		t.Fatalf("expected step agent count, got %v", got)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := model.AgentCount(); got != 2 {
		t.Fatalf("expected agent count 2, got %d", got)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := model.Rigor().Loops; got != 99 {
		t.Fatalf("expected rigor loops 99, got %d", got)
	}

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := model.Step(); got != AuditWizardStepGenerating {
		t.Fatalf("expected step generating, got %v", got)
	}
	if cmd == nil {
		t.Fatal("expected generating command to be returned")
	}

	model, _ = model.Update(auditWizardLaunchMsg{})
	if !model.Launched() {
		t.Fatal("expected model launched after generation message")
	}

	if got := model.Mode(); got != WizardModeManual {
		t.Fatalf("expected manual mode, got %v", got)
	}

	selected := model.SelectedAuditTypes()
	if len(selected) != 1 {
		t.Fatalf("expected one selected audit type, got %d", len(selected))
	}
}

func TestAuditWizardEscBackNavigation(t *testing.T) {
	t.Parallel()

	model := NewAuditWizardModel()

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := model.Step(); got != AuditWizardStepAgentCount {
		t.Fatalf("expected step agent count, got %v", got)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if got := model.Step(); got != AuditWizardStepTypes {
		t.Fatalf("expected back to types step, got %v", got)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if got := model.Step(); got != AuditWizardStepMode {
		t.Fatalf("expected back to mode step, got %v", got)
	}
}

func TestAuditWizardRequiresAtLeastOneAuditType(t *testing.T) {
	t.Parallel()

	model := NewAuditWizardModel()
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if got := model.Step(); got != AuditWizardStepTypes {
		t.Fatalf("expected to remain on types step, got %v", got)
	}

	view := model.View()
	if !strings.Contains(view, "Select at least one audit type") {
		t.Fatalf("expected validation error in view, got %q", view)
	}
}

func TestAuditWizardAllowsEditingSelectionsAfterGoingBack(t *testing.T) {
	t.Parallel()

	model := NewAuditWizardModel()
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if got := model.Step(); got != AuditWizardStepTypes {
		t.Fatalf("expected back to types step, got %v", got)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if got := model.Step(); got != AuditWizardStepTypes {
		t.Fatalf("expected to stay on types step after removing all selections, got %v", got)
	}
}
