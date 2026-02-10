package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// MenuAction is the next app action selected from the main menu.
type MenuAction int

const (
	MenuActionNone MenuAction = iota
	MenuActionOpenAuditWizard
	MenuActionQuit
)

type menuItem struct {
	label       string
	description string
	action      MenuAction
}

// MenuModel renders and updates the top-level LATTICE menu.
type MenuModel struct {
	styles    Styles
	keyMap    KeyMap
	items     []menuItem
	cursor    int
	confirmed bool
	action    MenuAction
}

// NewMenuModel returns the initial main menu state.
func NewMenuModel() MenuModel {
	return MenuModel{
		styles: DefaultStyles(),
		keyMap: DefaultKeyMap(),
		items: []menuItem{
			{
				label:       "Audit",
				description: "Start the audit wizard",
				action:      MenuActionOpenAuditWizard,
			},
			{
				label:       "Quit",
				description: "Exit LATTICE",
				action:      MenuActionQuit,
			},
		},
		action: MenuActionNone,
	}
}

// SetStyles overrides visual styling used by the menu.
func (m MenuModel) SetStyles(styles Styles) MenuModel {
	m.styles = styles
	return m
}

// SetKeyMap overrides key bindings used by the menu.
func (m MenuModel) SetKeyMap(keyMap KeyMap) MenuModel {
	m.keyMap = keyMap
	return m
}

// Update applies key input and returns the next menu state.
func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	if m.confirmed {
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if len(m.items) == 0 {
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
	case key.Matches(keyMsg, m.keyMap.Select):
		m.confirmed = true
		m.action = m.items[m.cursor].action
	case key.Matches(keyMsg, m.keyMap.Quit):
		m.confirmed = true
		m.action = MenuActionQuit
	}

	return m, nil
}

// View renders the menu as a Bubble Tea view string.
func (m MenuModel) View() string {
	var lines []string

	lines = append(lines, m.styles.Header.Render("LATTICE"))
	lines = append(lines, m.styles.Subheader.Render("Main Menu"))
	lines = append(lines, "")

	if len(m.items) == 0 {
		lines = append(lines, m.styles.Muted.Render("No options available."))
	} else {
		for idx, item := range m.items {
			prefix := " "
			if idx == m.cursor {
				prefix = m.styles.FocusedMark.Render(">")
			}

			entry := fmt.Sprintf("%s %s", prefix, item.label)
			if idx == m.cursor {
				lines = append(lines, m.styles.Selected.Render(entry))
			} else {
				lines = append(lines, m.styles.ListItem.Render(entry))
			}

			if item.description != "" {
				lines = append(lines, m.styles.Muted.PaddingLeft(4).Render(item.description))
			}
		}
	}

	help := []string{
		fmt.Sprintf("%s: %s", m.keyMap.Up.Help().Key, m.keyMap.Up.Help().Desc),
		fmt.Sprintf("%s: %s", m.keyMap.Down.Help().Key, m.keyMap.Down.Help().Desc),
		fmt.Sprintf("%s: %s", m.keyMap.Select.Help().Key, m.keyMap.Select.Help().Desc),
		fmt.Sprintf("%s: %s", m.keyMap.Quit.Help().Key, m.keyMap.Quit.Help().Desc),
	}
	lines = append(lines, "")
	lines = append(lines, m.styles.Help.Render(strings.Join(help, " • ")))

	return strings.Join(lines, "\n")
}

// Cursor returns the current highlighted option index.
func (m MenuModel) Cursor() int {
	return m.cursor
}

// Confirmed reports whether a menu action was selected.
func (m MenuModel) Confirmed() bool {
	return m.confirmed
}

// Action returns the selected menu action.
func (m MenuModel) Action() MenuAction {
	return m.action
}
