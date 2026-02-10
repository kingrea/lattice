package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MultiSelectItem is one selectable option in the list.
type MultiSelectItem[T any] struct {
	Label       string
	Description string
	Selected    bool
	Value       T
}

// MultiSelectKeyMap defines key bindings for the multi-select component.
type MultiSelectKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Toggle    key.Binding
	Confirm   key.Binding
	SelectAll key.Binding
}

// DefaultMultiSelectKeyMap returns key bindings for list navigation and selection.
func DefaultMultiSelectKeyMap() MultiSelectKeyMap {
	return MultiSelectKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "select all"),
		),
	}
}

// MultiSelectModel is a reusable Bubble Tea component for multi-select lists.
type MultiSelectModel[T any] struct {
	title     string
	styles    Styles
	keyMap    MultiSelectKeyMap
	items     []MultiSelectItem[T]
	cursor    int
	confirmed bool
}

// NewMultiSelectModel creates a new multi-select list with copied item state.
func NewMultiSelectModel[T any](title string, items []MultiSelectItem[T]) MultiSelectModel[T] {
	itemsCopy := make([]MultiSelectItem[T], len(items))
	copy(itemsCopy, items)

	return MultiSelectModel[T]{
		title:  title,
		styles: DefaultStyles(),
		keyMap: DefaultMultiSelectKeyMap(),
		items:  itemsCopy,
	}
}

// SetStyles overrides the visual styling used by the component.
func (m MultiSelectModel[T]) SetStyles(styles Styles) MultiSelectModel[T] {
	m.styles = styles
	return m
}

// SetKeyMap overrides key bindings used by the component.
func (m MultiSelectModel[T]) SetKeyMap(keyMap MultiSelectKeyMap) MultiSelectModel[T] {
	m.keyMap = keyMap
	return m
}

// Update applies key input and returns the next model state.
func (m MultiSelectModel[T]) Update(msg tea.Msg) (MultiSelectModel[T], tea.Cmd) {
	if m.confirmed {
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if len(m.items) == 0 {
		if key.Matches(keyMsg, m.keyMap.Confirm) {
			m.confirmed = true
		}
		return m, nil
	}

	switch {
	case key.Matches(keyMsg, m.keyMap.Up):
		if m.cursor == 0 {
			m.cursor = len(m.items) - 1
		} else {
			m.cursor--
		}
	case key.Matches(keyMsg, m.keyMap.Down):
		m.cursor = (m.cursor + 1) % len(m.items)
	case key.Matches(keyMsg, m.keyMap.Toggle):
		m.items[m.cursor].Selected = !m.items[m.cursor].Selected
	case key.Matches(keyMsg, m.keyMap.SelectAll):
		for idx := range m.items {
			m.items[idx].Selected = true
		}
	case key.Matches(keyMsg, m.keyMap.Confirm):
		m.confirmed = true
	}

	return m, nil
}

// View renders the component as a Bubble Tea view string.
func (m MultiSelectModel[T]) View() string {
	var lines []string

	if m.title != "" {
		lines = append(lines, m.styles.Header.Render(m.title))
	}

	if len(m.items) == 0 {
		lines = append(lines, m.styles.Muted.Render("No options available."))
	} else {
		for idx, item := range m.items {
			check := "[ ]"
			if item.Selected {
				check = "[x]"
			}

			prefix := " "
			if idx == m.cursor {
				prefix = m.styles.FocusedMark.Render(">")
			}

			line := fmt.Sprintf("%s %s %s", prefix, check, item.Label)
			if idx == m.cursor {
				lines = append(lines, m.styles.Selected.Render(line))
			} else {
				lines = append(lines, m.styles.ListItem.Render(line))
			}

			if item.Description != "" {
				detail := lipgloss.NewStyle().PaddingLeft(6).Inherit(m.styles.Muted).Render(item.Description)
				lines = append(lines, detail)
			}
		}
	}

	help := []string{
		fmt.Sprintf("%s: %s", m.keyMap.Up.Help().Key, m.keyMap.Up.Help().Desc),
		fmt.Sprintf("%s: %s", m.keyMap.Down.Help().Key, m.keyMap.Down.Help().Desc),
		fmt.Sprintf("%s: %s", m.keyMap.Toggle.Help().Key, m.keyMap.Toggle.Help().Desc),
		fmt.Sprintf("%s: %s", m.keyMap.SelectAll.Help().Key, m.keyMap.SelectAll.Help().Desc),
		fmt.Sprintf("%s: %s", m.keyMap.Confirm.Help().Key, m.keyMap.Confirm.Help().Desc),
	}
	lines = append(lines, "")
	lines = append(lines, m.styles.Help.Render(strings.Join(help, " • ")))

	return strings.Join(lines, "\n")
}

// Items returns a copy of all items, including selection state.
func (m MultiSelectModel[T]) Items() []MultiSelectItem[T] {
	out := make([]MultiSelectItem[T], len(m.items))
	copy(out, m.items)
	return out
}

// SelectedItems returns all currently selected items.
func (m MultiSelectModel[T]) SelectedItems() []MultiSelectItem[T] {
	selected := make([]MultiSelectItem[T], 0, len(m.items))
	for _, item := range m.items {
		if item.Selected {
			selected = append(selected, item)
		}
	}

	return selected
}

// Cursor returns the current highlighted item index.
func (m MultiSelectModel[T]) Cursor() int {
	return m.cursor
}

// Confirmed reports whether the user confirmed the current selection.
func (m MultiSelectModel[T]) Confirmed() bool {
	return m.confirmed
}
