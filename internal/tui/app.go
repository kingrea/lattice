package tui

import (
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppScreen identifies the active top-level app screen.
type AppScreen int

const (
	MenuScreen AppScreen = iota
	WizardScreen
	DashboardScreen
)

// AppNavigateMsg requests a top-level screen change.
type AppNavigateMsg struct {
	Screen AppScreen
}

// NavigateTo returns a message that switches the app to the target screen.
func NavigateTo(screen AppScreen) tea.Msg {
	return AppNavigateMsg{Screen: screen}
}

// AppModel routes Bubble Tea messages between top-level screens.
type AppModel struct {
	cwd string

	styles Styles
	keyMap KeyMap

	screen    AppScreen
	menu      MenuModel
	wizard    AuditWizardModel
	dashboard DashboardModel

	launchStarted bool

	width  int
	height int
}

// NewApp creates the root app model at the main menu.
func NewApp(cwd string) AppModel {
	styles := DefaultStyles()
	keyMap := DefaultKeyMap()

	menu := NewMenuModel().SetStyles(styles).SetKeyMap(keyMap)
	wizard := NewAuditWizardModel().SetStyles(styles).SetKeyMap(keyMap).SetProjectDir(cwd)

	return AppModel{
		cwd:       cwd,
		styles:    styles,
		keyMap:    keyMap,
		screen:    MenuScreen,
		menu:      menu,
		wizard:    wizard,
		dashboard: NewDashboardModel(cwd, styles, keyMap),
	}
}

// Init initializes the root app model.
func (m AppModel) Init() tea.Cmd {
	return nil
}

// Update routes events to the active screen and handles app-level navigation.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		return m, nil
	case AppNavigateMsg:
		m.screen = typed.Screen
		if typed.Screen != WizardScreen {
			m.launchStarted = false
		}
		if typed.Screen == DashboardScreen {
			m.dashboard = NewDashboardModel(m.cwd, m.styles, m.keyMap)
			return m, m.dashboard.Init()
		}
		return m, nil
	case tea.KeyMsg:
		if key.Matches(typed, m.keyMap.Quit) {
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd

	switch m.screen {
	case MenuScreen:
		m.menu, cmd = m.menu.Update(msg)
		if m.menu.Confirmed() {
			switch m.menu.Action() {
			case MenuActionOpenAuditWizard:
				m.wizard = NewAuditWizardModel().SetStyles(m.styles).SetKeyMap(m.keyMap).SetProjectDir(m.cwd)
				m.screen = WizardScreen
				return m, nil
			case MenuActionQuit:
				return m, tea.Quit
			}
		}
	case WizardScreen:
		if keyMsg, ok := msg.(tea.KeyMsg); ok && key.Matches(keyMsg, m.keyMap.Back) && m.wizard.Step() == AuditWizardStepMode {
			m.screen = MenuScreen
			m.menu = NewMenuModel().SetStyles(m.styles).SetKeyMap(m.keyMap)
			m.launchStarted = false
			return m, nil
		}

		m.wizard, cmd = m.wizard.Update(msg)
		if m.wizard.Step() != AuditWizardStepGenerating {
			m.launchStarted = false
		}
		if m.wizard.Step() == AuditWizardStepGenerating && !m.wizard.Launched() && !m.launchStarted {
			m.launchStarted = true
			launchCmd := launchAuditCmd(launchRequest{
				cwd:        m.cwd,
				target:     filepath.Base(m.cwd),
				auditTypes: m.wizard.SelectedAuditTypes(),
				agentCount: m.wizard.AgentCount(),
				intensity:  m.wizard.Rigor().Loops,
				focusAreas: m.wizard.DiscoveredFocusAreas(),
			})
			if cmd == nil {
				return m, launchCmd
			}
			return m, tea.Batch(cmd, launchCmd)
		}

		if m.wizard.Step() == AuditWizardStepGenerating && m.wizard.Launched() {
			m.launchStarted = false
			m.screen = DashboardScreen
			m.dashboard = NewDashboardModel(m.cwd, m.styles, m.keyMap)
			return m, m.dashboard.Init()
		}
	case DashboardScreen:
		m.dashboard, cmd = m.dashboard.Update(msg)
	}

	return m, cmd
}

// View renders the active top-level screen.
func (m AppModel) View() string {
	var view string

	switch m.screen {
	case MenuScreen:
		view = m.menu.View()
	case WizardScreen:
		view = m.wizard.View()
	case DashboardScreen:
		view = m.dashboard.View()
	default:
		view = m.styles.Error.Render("Unknown app screen")
	}

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, view)
	}

	return view
}

// Screen returns the active top-level screen.
func (m AppModel) Screen() AppScreen {
	return m.screen
}
