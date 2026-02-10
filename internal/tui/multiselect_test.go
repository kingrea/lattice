package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMultiSelectToggleAndNavigation(t *testing.T) {
	t.Parallel()

	model := NewMultiSelectModel("Audit Types", []MultiSelectItem[int]{
		{Label: "Performance", Value: 1},
		{Label: "Security", Value: 2},
	})

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := model.Cursor(); got != 1 {
		t.Fatalf("expected cursor to move down to 1, got %d", got)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	items := model.Items()
	if !items[1].Selected {
		t.Fatal("expected second item to be selected after pressing space")
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	items = model.Items()
	if items[1].Selected {
		t.Fatal("expected second item to be unselected after pressing space again")
	}
}

func TestMultiSelectWrapNavigation(t *testing.T) {
	t.Parallel()

	model := NewMultiSelectModel("Audit Types", []MultiSelectItem[int]{
		{Label: "Performance", Value: 1},
		{Label: "Security", Value: 2},
	})

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got := model.Cursor(); got != 1 {
		t.Fatalf("expected cursor to wrap to last item, got %d", got)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := model.Cursor(); got != 0 {
		t.Fatalf("expected cursor to wrap to first item, got %d", got)
	}
}

func TestMultiSelectSelectAllAndConfirm(t *testing.T) {
	t.Parallel()

	model := NewMultiSelectModel("Audit Types", []MultiSelectItem[int]{
		{Label: "Performance", Value: 1},
		{Label: "Security", Value: 2},
		{Label: "Accessibility", Value: 3},
	})

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	for _, item := range model.Items() {
		if !item.Selected {
			t.Fatal("expected all items selected after pressing a")
		}
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !model.Confirmed() {
		t.Fatal("expected model to be confirmed after pressing enter")
	}
}

func TestMultiSelectItemsReturnsCopy(t *testing.T) {
	t.Parallel()

	model := NewMultiSelectModel("Audit Types", []MultiSelectItem[int]{
		{Label: "Performance", Value: 1, Selected: true},
	})

	items := model.Items()
	items[0].Selected = false

	if !model.Items()[0].Selected {
		t.Fatal("expected Items to return a copy, not mutate model state")
	}
}
