package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lattice/internal/config"
)

const dashboardRefreshInterval = 3 * time.Second

type dashboardTeamStatus struct {
	TeamName    string
	Status      string
	CurrentLoop int
	Intensity   int
	AgentCount  int
}

type dashboardSnapshot struct {
	SessionName string
	Teams       []dashboardTeamStatus
	RefreshedAt time.Time
}

type dashboardRefreshMsg struct {
	Snapshot dashboardSnapshot
	Err      error
}

type dashboardTickMsg struct{}

type dashboardAttachDoneMsg struct {
	Err error
}

type dashboardLoadSnapshotFunc func(cwd string, now time.Time) (dashboardSnapshot, error)

// DashboardModel renders post-launch team status and actions.
type DashboardModel struct {
	styles Styles
	keyMap KeyMap
	cwd    string

	refreshInterval time.Duration
	loadSnapshot    dashboardLoadSnapshotFunc
	now             func() time.Time

	sessionName string
	teams       []dashboardTeamStatus
	lastUpdated time.Time
	err         error
}

// NewDashboardModel creates the post-launch dashboard.
func NewDashboardModel(cwd string, styles Styles, keyMap KeyMap) DashboardModel {
	return DashboardModel{
		styles:          styles,
		keyMap:          keyMap,
		cwd:             cwd,
		refreshInterval: dashboardRefreshInterval,
		loadSnapshot:    loadDashboardSnapshot,
		now:             time.Now,
	}
}

// Init starts periodic status refresh for the dashboard.
func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(m.refreshCmd(), m.tickCmd())
}

// Update handles dashboard key input and refresh events.
func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	switch typed := msg.(type) {
	case dashboardRefreshMsg:
		if typed.Err != nil {
			m.err = typed.Err
			return m, nil
		}

		m.sessionName = typed.Snapshot.SessionName
		m.teams = typed.Snapshot.Teams
		m.lastUpdated = typed.Snapshot.RefreshedAt
		m.err = nil
		return m, nil
	case dashboardTickMsg:
		return m, tea.Batch(m.refreshCmd(), m.tickCmd())
	case dashboardAttachDoneMsg:
		if typed.Err != nil {
			m.err = fmt.Errorf("attach tmux session %q: %w", m.sessionName, typed.Err)
			return m, nil
		}
		return m, m.refreshCmd()
	case tea.KeyMsg:
		if key.Matches(typed, m.keyMap.Back) {
			return m, func() tea.Msg { return NavigateTo(MenuScreen) }
		}

		switch strings.ToLower(typed.String()) {
		case "r":
			return m, m.refreshCmd()
		case "t":
			if strings.TrimSpace(m.sessionName) == "" {
				m.err = fmt.Errorf("no active tmux session found")
				return m, nil
			}
			return m, m.attachCmd()
		}
	}

	return m, nil
}

// View renders dashboard status and key hints.
func (m DashboardModel) View() string {
	lines := []string{
		m.styles.Header.Render("LATTICE"),
		m.styles.Subheader.Render("Post-launch Team Status"),
		"",
		m.styles.Body.Render(fmt.Sprintf("Session: %s", fallbackText(m.sessionName, "(loading...)"))),
	}

	if !m.lastUpdated.IsZero() {
		lines = append(lines, m.styles.Muted.Render(fmt.Sprintf("Last refresh: %s", m.lastUpdated.Format(time.Kitchen))))
	}

	if m.err != nil {
		lines = append(lines, "", m.styles.Error.Render(m.err.Error()))
	}

	lines = append(lines, "", m.renderTeamTable(), "", m.styles.Help.Render("t: attach tmux  r: refresh  esc: menu  q: quit"))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m DashboardModel) renderTeamTable() string {
	if len(m.teams) == 0 {
		return m.styles.Muted.Render("No running teams discovered yet.")
	}

	header := fmt.Sprintf("%-24s %-12s %-14s %-6s", "TEAM", "STATUS", "LOOP", "AGENTS")
	rows := []string{m.styles.Muted.Render(header)}
	for _, team := range m.teams {
		loop := fmt.Sprintf("%d/%d", team.CurrentLoop, team.Intensity)
		rows = append(rows, m.styles.Body.Render(fmt.Sprintf("%-24s %-12s %-14s %-6d", team.TeamName, team.Status, loop, team.AgentCount)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m DashboardModel) refreshCmd() tea.Cmd {
	cwd := m.cwd
	loadSnapshot := m.loadSnapshot
	now := m.now
	return func() tea.Msg {
		snapshot, err := loadSnapshot(cwd, now().UTC())
		return dashboardRefreshMsg{Snapshot: snapshot, Err: err}
	}
}

func (m DashboardModel) tickCmd() tea.Cmd {
	interval := m.refreshInterval
	return tea.Tick(interval, func(time.Time) tea.Msg { return dashboardTickMsg{} })
}

func (m DashboardModel) attachCmd() tea.Cmd {
	sessionName := m.sessionName
	attach := tea.ExecProcess(exec.Command("wsl", "tmux", "attach-session", "-t", sessionName), func(err error) tea.Msg {
		return dashboardAttachDoneMsg{Err: err}
	})
	return tea.Sequence(attach, m.refreshCmd())
}

func loadDashboardSnapshot(cwd string, now time.Time) (dashboardSnapshot, error) {
	cfg, err := config.Load(cwd)
	if err != nil {
		return dashboardSnapshot{}, fmt.Errorf("load lattice config: %w", err)
	}

	teamKeys := make([]string, 0, len(cfg.Teams))
	for key := range cfg.Teams {
		teamKeys = append(teamKeys, key)
	}
	sort.Strings(teamKeys)

	teams := make([]dashboardTeamStatus, 0, len(teamKeys))
	for _, teamKey := range teamKeys {
		teamState := cfg.Teams[teamKey]
		teamDir := filepath.Join(cwd, config.DirName, "teams", "audit-"+teamKey)
		teamData, err := readTeamFile(filepath.Join(teamDir, ".team"))
		if err != nil && !os.IsNotExist(err) {
			return dashboardSnapshot{}, fmt.Errorf("read team status for %s: %w", teamKey, err)
		}

		teamName := fallbackText(teamData["team"], "audit-"+teamKey)
		status := fallbackText(teamData["status"], fallbackText(teamState.Status, "unknown"))
		currentLoop := parseIntFallback(teamData["current_loop"], 0)
		intensity := parseIntFallback(teamData["intensity"], teamState.Intensity)

		teams = append(teams, dashboardTeamStatus{
			TeamName:    teamName,
			Status:      status,
			CurrentLoop: currentLoop,
			Intensity:   intensity,
			AgentCount:  teamState.AgentCount,
		})
	}

	return dashboardSnapshot{
		SessionName: cfg.Session.Name,
		Teams:       teams,
		RefreshedAt: now,
	}, nil
}

func readTeamFile(filePath string) (map[string]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}

		result[key] = value
	}

	return result, nil
}

func parseIntFallback(value string, fallback int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func fallbackText(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}

	return trimmed
}
