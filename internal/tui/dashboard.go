package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lattice/internal/config"
	"lattice/internal/teams"
	"lattice/internal/tmux"
)

const dashboardRefreshInterval = 3 * time.Second

type dashboardTeamStatus struct {
	TeamName    string
	Status      string
	CurrentLoop int
	Intensity   int
	AgentCount  int
}

type dashboardRoleStatus struct {
	BeadID      string
	CodeName    string
	Title       string
	Status      string
	CurrentLoop int
	Intensity   int
	BeadPrefix  string
}

type dashboardEpicStatus struct {
	EpicName      string
	BeadID        string
	AuditType     string
	Status        string
	RolesTotal    int
	RolesComplete int
	RolesFailed   int
	Roles         []dashboardRoleStatus
}

type dashboardSnapshot struct {
	SessionName string
	Epics       []dashboardEpicStatus
	Teams       []dashboardTeamStatus
	RefreshedAt time.Time
}

type dashboardRefreshMsg struct {
	Snapshot dashboardSnapshot
	Err      error
}

type dashboardTickMsg struct{}

type schedulerAdvancedMsg struct {
	Result SchedulerResult
	Err    error
}

type dashboardAttachDoneMsg struct {
	Err error
}

type dashboardLoadSnapshotFunc func(cwd string, now time.Time) (dashboardSnapshot, error)
type dashboardLoadConfigFunc func(cwd string) (*config.Config, error)
type dashboardBuildPlanFunc func(cfg *config.Config) *teams.AuditPlan
type dashboardCheckAndAdvanceRolesFunc func(cwd string, cfg *config.Config, sessionName string, plan *teams.AuditPlan, deps SchedulerDeps) (SchedulerResult, error)

// DashboardModel renders post-launch team status and actions.
type DashboardModel struct {
	styles Styles
	keyMap KeyMap
	cwd    string

	refreshInterval time.Duration
	loadSnapshot    dashboardLoadSnapshotFunc
	loadConfig      dashboardLoadConfigFunc
	buildPlan       dashboardBuildPlanFunc
	advanceRoles    dashboardCheckAndAdvanceRolesFunc
	schedulerDeps   SchedulerDeps
	now             func() time.Time

	sessionName string
	epics       []dashboardEpicStatus
	teams       []dashboardTeamStatus
	allDone     bool
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
		loadConfig:      config.Load,
		buildPlan:       buildDashboardPlanFromConfig,
		advanceRoles:    CheckAndAdvanceRoles,
		schedulerDeps:   SchedulerDeps{},
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
		m.epics = typed.Snapshot.Epics
		m.teams = typed.Snapshot.Teams
		m.allDone = snapshotAllDone(typed.Snapshot)
		m.lastUpdated = typed.Snapshot.RefreshedAt
		m.err = nil
		return m, nil
	case dashboardTickMsg:
		return m, tea.Batch(m.schedulerCmd(), m.refreshCmd(), m.tickCmd())
	case schedulerAdvancedMsg:
		if typed.Err != nil {
			m.err = typed.Err
			return m, nil
		}

		m.allDone = typed.Result.AllDone
		return m, m.refreshCmd()
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
		m.styles.Subheader.Render("Post-launch Audit Status"),
		"",
		m.styles.Body.Render(fmt.Sprintf("Session: %s", fallbackText(m.sessionName, "(loading...)"))),
	}

	if !m.lastUpdated.IsZero() {
		lines = append(lines, m.styles.Muted.Render(fmt.Sprintf("Last refresh: %s", m.lastUpdated.Format(time.Kitchen))))
	}

	if m.err != nil {
		lines = append(lines, "", m.styles.Error.Render(m.err.Error()))
	}
	if m.allDone {
		lines = append(lines, "", m.styles.Success.Render("All roles reached a terminal state. Review failed items before closing out."))
	}

	lines = append(lines, "", m.renderEpicTable(), "", m.styles.Help.Render("t: attach tmux  r: refresh  esc: menu  q: quit"))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m DashboardModel) renderEpicTable() string {
	if len(m.epics) == 0 {
		if len(m.teams) > 0 {
			return m.renderTeamTable()
		}

		return m.styles.Muted.Render("No running epics discovered yet.")
	}

	header := fmt.Sprintf("%-24s %-12s %-14s", "EPIC", "STATUS", "PROGRESS")
	rows := []string{m.styles.Muted.Render(header)}
	for _, epic := range m.epics {
		progress := fmt.Sprintf("%d/%d roles done", epic.RolesComplete, epic.RolesTotal)
		epicStatus := formatDashboardStatus(epic.Status)
		epicRow := fmt.Sprintf("%-24s %-12s %-14s", epic.EpicName, epicStatus, progress)
		switch strings.ToLower(strings.TrimSpace(epic.Status)) {
		case "failed", "blocked":
			rows = append(rows, m.styles.Error.Render(epicRow))
		case "complete":
			rows = append(rows, m.styles.Success.Render(epicRow))
		default:
			rows = append(rows, m.styles.Body.Render(epicRow))
		}

		for _, role := range epic.Roles {
			roleLabel := fmt.Sprintf("  %s (%s)", fallbackText(role.CodeName, "-"), fallbackText(role.Title, "-"))
			roleStatus := formatDashboardStatus(role.Status)
			roleRow := fmt.Sprintf("%-24s %-12s %-14s", roleLabel, roleStatus, formatRoleProgress(role))
			if strings.EqualFold(strings.TrimSpace(role.Status), "failed") {
				rows = append(rows, m.styles.Error.Render(roleRow))
				continue
			}
			if strings.EqualFold(strings.TrimSpace(role.Status), "complete") {
				rows = append(rows, m.styles.Success.Render(roleRow))
				continue
			}
			rows = append(rows, m.styles.Body.Render(roleRow))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
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

func (m DashboardModel) schedulerCmd() tea.Cmd {
	loadConfig := m.loadConfig
	buildPlan := m.buildPlan
	advanceRoles := m.advanceRoles
	deps := m.schedulerDeps
	cwd := m.cwd

	return func() tea.Msg {
		cfg, err := loadConfig(cwd)
		if err != nil {
			return schedulerAdvancedMsg{Err: fmt.Errorf("load lattice config: %w", err)}
		}
		if len(cfg.Epics) == 0 {
			return nil
		}

		plan := buildPlan(cfg)
		if plan == nil || len(plan.Epics) == 0 {
			return nil
		}

		result, err := advanceRoles(cwd, cfg, cfg.Session.Name, plan, deps)
		if err != nil {
			return schedulerAdvancedMsg{Err: err}
		}

		if len(result.Launched) == 0 && len(result.Completed) == 0 && len(result.Failed) == 0 {
			return nil
		}

		if err := cfg.Save(); err != nil {
			return schedulerAdvancedMsg{Err: fmt.Errorf("save scheduler updates: %w", err)}
		}

		return schedulerAdvancedMsg{Result: result}
	}
}

func (m DashboardModel) attachCmd() tea.Cmd {
	sessionName := m.sessionName
	attach := tea.ExecProcess(tmux.Command("attach-session", "-t", sessionName), func(err error) tea.Msg {
		return dashboardAttachDoneMsg{Err: err}
	})
	return tea.Sequence(attach, m.refreshCmd())
}

func loadDashboardSnapshot(cwd string, now time.Time) (dashboardSnapshot, error) {
	cfg, err := config.Load(cwd)
	if err != nil {
		return dashboardSnapshot{}, fmt.Errorf("load lattice config: %w", err)
	}

	if len(cfg.Epics) > 0 {
		epics, err := loadEpicStatuses(cwd, cfg)
		if err != nil {
			return dashboardSnapshot{}, err
		}

		return dashboardSnapshot{
			SessionName: cfg.Session.Name,
			Epics:       epics,
			RefreshedAt: now,
		}, nil
	}

	teams, err := loadLegacyTeams(cwd, cfg)
	if err != nil {
		return dashboardSnapshot{}, err
	}

	return dashboardSnapshot{
		SessionName: cfg.Session.Name,
		Teams:       teams,
		RefreshedAt: now,
	}, nil
}

func loadEpicStatuses(cwd string, cfg *config.Config) ([]dashboardEpicStatus, error) {
	type roleSnapshot struct {
		status dashboardRoleStatus
		order  int
	}

	epicKeys := make([]string, 0, len(cfg.Epics))
	for key := range cfg.Epics {
		epicKeys = append(epicKeys, key)
	}
	sort.Strings(epicKeys)

	rolesByEpic := make(map[string][]roleSnapshot)
	for roleKey, roleState := range cfg.Roles {
		roleData := map[string]string{}
		for _, roleDirName := range dashboardRoleDirectories(roleState, roleKey) {
			roleDir := filepath.Join(cwd, config.DirName, "teams", roleDirName)
			data, err := readTeamFile(filepath.Join(roleDir, ".team"))
			if err == nil {
				roleData = data
				break
			}
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("read role status for %s: %w", roleKey, err)
			}
		}

		status := normalizeRoleStatus(fallbackText(roleData["status"], roleState.Status))
		rolesByEpic[roleState.EpicBeadID] = append(rolesByEpic[roleState.EpicBeadID], roleSnapshot{
			status: dashboardRoleStatus{
				BeadID:      fallbackText(roleState.BeadID, roleKey),
				CodeName:    roleState.CodeName,
				Title:       roleState.Title,
				Status:      status,
				CurrentLoop: parseIntFallback(roleData["current_loop"], 0),
				Intensity:   parseIntFallback(roleData["intensity"], roleState.Intensity),
				BeadPrefix:  roleState.BeadPrefix,
			},
			order: roleState.Order,
		})
	}

	epics := make([]dashboardEpicStatus, 0, len(epicKeys))
	for _, epicKey := range epicKeys {
		epicState := cfg.Epics[epicKey]
		epicID := fallbackText(epicState.BeadID, epicKey)
		roleSnapshots := rolesByEpic[epicID]
		sort.Slice(roleSnapshots, func(i, j int) bool {
			if roleSnapshots[i].order != roleSnapshots[j].order {
				return roleSnapshots[i].order < roleSnapshots[j].order
			}
			if roleSnapshots[i].status.CodeName == roleSnapshots[j].status.CodeName {
				return roleSnapshots[i].status.BeadID < roleSnapshots[j].status.BeadID
			}

			return roleSnapshots[i].status.CodeName < roleSnapshots[j].status.CodeName
		})

		roles := make([]dashboardRoleStatus, 0, len(roleSnapshots))
		for _, roleSnapshot := range roleSnapshots {
			roles = append(roles, roleSnapshot.status)
		}

		rolesComplete := 0
		rolesFailed := 0
		for _, role := range roles {
			switch role.Status {
			case "complete":
				rolesComplete++
			case "failed":
				rolesFailed++
			}
		}

		epics = append(epics, dashboardEpicStatus{
			EpicName:      fallbackText(epicState.AuditName, fallbackText(epicState.AuditType, epicKey)),
			BeadID:        epicID,
			AuditType:     epicState.AuditType,
			Status:        deriveEpicStatus(roles, epicState.Status),
			RolesTotal:    len(roles),
			RolesComplete: rolesComplete,
			RolesFailed:   rolesFailed,
			Roles:         roles,
		})
	}

	return epics, nil
}

func loadLegacyTeams(cwd string, cfg *config.Config) ([]dashboardTeamStatus, error) {
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
			return nil, fmt.Errorf("read team status for %s: %w", teamKey, err)
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

	return teams, nil
}

func dashboardRoleDirectories(role config.RoleState, roleKey string) []string {
	codeName := fallbackText(role.CodeName, roleKey)
	beadPrefix := fallbackText(role.BeadPrefix, roleKey)

	dirs := make([]string, 0, 2)
	if auditTypeID := beadPrefixAuditTypeID(beadPrefix); auditTypeID != "" {
		dirs = append(dirs, auditTypeID+"-"+codeName)
	}
	dirs = append(dirs, beadPrefix+"-"+codeName)

	return dirs
}

func beadPrefixAuditTypeID(beadPrefix string) string {
	prefix := strings.TrimSpace(beadPrefix)
	if prefix == "" {
		return ""
	}

	parts := strings.SplitN(prefix, "-", 2)
	return strings.TrimSpace(parts[0])
}

func normalizeRoleStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case "active":
		return "running"
	case "pending", "running", "complete", "failed":
		return normalized
	default:
		return fallbackText(normalized, "unknown")
	}
}

func deriveEpicStatus(roles []dashboardRoleStatus, fallback string) string {
	if len(roles) == 0 {
		return fallbackText(fallback, "unknown")
	}

	allComplete := true
	hasRunningOrPending := false
	hasFailed := false
	for _, role := range roles {
		switch role.Status {
		case "running", "pending":
			hasRunningOrPending = true
			allComplete = false
		case "complete":
			continue
		case "failed":
			hasFailed = true
			allComplete = false
		default:
			allComplete = false
		}
	}

	if hasFailed && hasRunningOrPending {
		return "blocked"
	}
	if hasFailed {
		return "failed"
	}
	if hasRunningOrPending {
		return "running"
	}
	if allComplete {
		return "complete"
	}

	return fallbackText(fallback, "unknown")
}

func formatDashboardStatus(status string) string {
	value := strings.ToLower(strings.TrimSpace(status))
	switch value {
	case "failed":
		return "FAILED"
	case "blocked":
		return "BLOCKED"
	case "complete":
		return "complete"
	case "running":
		return "running"
	case "pending":
		return "pending"
	default:
		return fallbackText(value, "unknown")
	}
}

func snapshotAllDone(snapshot dashboardSnapshot) bool {
	if len(snapshot.Epics) == 0 {
		return false
	}

	hasRoles := false
	for _, epic := range snapshot.Epics {
		for _, role := range epic.Roles {
			hasRoles = true
			switch strings.ToLower(strings.TrimSpace(role.Status)) {
			case "complete", "failed":
				continue
			default:
				return false
			}
		}
	}

	return hasRoles
}

func buildDashboardPlanFromConfig(cfg *config.Config) *teams.AuditPlan {
	if cfg == nil || len(cfg.Epics) == 0 {
		return &teams.AuditPlan{}
	}

	rolesByEpic := make(map[string][]config.RoleState)
	for roleKey, role := range cfg.Roles {
		if strings.TrimSpace(role.BeadID) == "" {
			role.BeadID = roleKey
		}
		rolesByEpic[role.EpicBeadID] = append(rolesByEpic[role.EpicBeadID], role)
	}

	epicKeys := make([]string, 0, len(cfg.Epics))
	for epicKey := range cfg.Epics {
		epicKeys = append(epicKeys, epicKey)
	}
	sort.Strings(epicKeys)

	plan := &teams.AuditPlan{Epics: make([]teams.EpicBead, 0, len(epicKeys))}
	for _, epicKey := range epicKeys {
		epic := cfg.Epics[epicKey]
		epicID := fallbackText(epic.BeadID, epicKey)
		auditTypeID := fallbackText(epic.AuditType, epicKey)

		roles := rolesByEpic[epicID]
		sort.SliceStable(roles, func(i, j int) bool {
			if roles[i].Order == roles[j].Order {
				return fallbackText(roles[i].BeadID, "") < fallbackText(roles[j].BeadID, "")
			}
			return roles[i].Order < roles[j].Order
		})

		roleBeads := make([]teams.RoleBead, 0, len(roles))
		for _, role := range roles {
			roleBeads = append(roleBeads, teams.RoleBead{
				BeadID:     role.BeadID,
				CodeName:   role.CodeName,
				Title:      role.Title,
				Guidance:   role.Guidance,
				BeadPrefix: role.BeadPrefix,
				Order:      role.Order,
			})
		}

		plan.Epics = append(plan.Epics, teams.EpicBead{
			BeadID: epicID,
			AuditType: teams.AuditType{
				ID:   auditTypeID,
				Name: epic.AuditName,
			},
			RoleBeads: roleBeads,
		})
	}

	return plan
}

func formatRoleProgress(role dashboardRoleStatus) string {
	if role.Intensity <= 0 {
		return "-"
	}

	if role.Status == "pending" && role.CurrentLoop <= 0 {
		return "-"
	}

	return fmt.Sprintf("loop %d/%d", role.CurrentLoop, role.Intensity)
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
