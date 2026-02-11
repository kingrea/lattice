package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lattice/internal/config"
	"lattice/internal/teams"
	"lattice/internal/tmux"
)

// LaunchCompleteMsg indicates audit launch finished successfully.
type LaunchCompleteMsg struct{}

// LaunchFailedMsg indicates audit launch failed.
type LaunchFailedMsg struct {
	Err error
}

type launchRequest struct {
	cwd        string
	auditTypes []teams.AuditType
	agentCount int
	intensity  int
	focusAreas []string
}

type launchTmuxManager interface {
	CreateSession(name string) error
	CreateWindow(session, name string) error
	SendKeys(session, window, command string) error
}

type launchDeps struct {
	initConfig         func(cwd string) (*config.Config, error)
	newTmuxManager     func() (launchTmuxManager, error)
	allocateBeadPrefix func(cfg *config.Config, typePrefix string) (string, error)
	generateTeam       func(params teams.GenerateParams) (string, error)
	translatePath      func(path string) (string, error)
	now                func() time.Time
}

func defaultLaunchDeps() launchDeps {
	return launchDeps{
		initConfig:         config.Init,
		newTmuxManager:     newLaunchTmuxManager,
		allocateBeadPrefix: teams.AllocateBeadPrefix,
		generateTeam:       teams.Generate,
		translatePath:      tmux.TranslateToWSLPath,
		now:                time.Now,
	}
}

func newLaunchTmuxManager() (launchTmuxManager, error) {
	return tmux.NewManager()
}

func launchAuditCmd(req launchRequest) tea.Cmd {
	deps := defaultLaunchDeps()
	return func() tea.Msg {
		return launchAudit(req, deps)
	}
}

func launchAudit(req launchRequest, deps launchDeps) tea.Msg {
	if strings.TrimSpace(req.cwd) == "" {
		return LaunchFailedMsg{Err: fmt.Errorf("working directory must not be empty")}
	}
	if len(req.auditTypes) == 0 {
		return LaunchFailedMsg{Err: fmt.Errorf("select at least one audit type")}
	}

	cfg, err := deps.initConfig(req.cwd)
	if err != nil {
		return LaunchFailedMsg{Err: fmt.Errorf("initialize lattice config: %w", err)}
	}

	manager, err := deps.newTmuxManager()
	if err != nil {
		return LaunchFailedMsg{Err: fmt.Errorf("initialize tmux manager: %w", err)}
	}

	sessionName := "lattice-" + deps.now().UTC().Format("20060102-150405")
	if err := manager.CreateSession(sessionName); err != nil {
		return LaunchFailedMsg{Err: fmt.Errorf("create tmux session: %w", err)}
	}

	if cfg.Teams == nil {
		cfg.Teams = map[string]config.TeamState{}
	}

	for _, auditType := range req.auditTypes {
		prefix, err := deps.allocateBeadPrefix(cfg, auditType.BeadPrefix)
		if err != nil {
			return LaunchFailedMsg{Err: fmt.Errorf("allocate bead prefix for %s: %w", auditType.ID, err)}
		}

		teamDir, err := deps.generateTeam(teams.GenerateParams{
			WorkingDir: req.cwd,
			AuditType:  auditType,
			AgentCount: req.agentCount,
			Intensity:  req.intensity,
			Target:     filepath.Base(req.cwd),
			BeadPrefix: prefix,
			FocusAreas: req.focusAreas,
		})
		if err != nil {
			return LaunchFailedMsg{Err: fmt.Errorf("generate team folder for %s: %w", auditType.ID, err)}
		}

		windowName := "audit-" + auditType.ID
		if err := manager.CreateWindow(sessionName, windowName); err != nil {
			return LaunchFailedMsg{Err: fmt.Errorf("create tmux window for %s: %w", auditType.ID, err)}
		}

		wslTeamDir, err := deps.translatePath(teamDir)
		if err != nil {
			return LaunchFailedMsg{Err: fmt.Errorf("translate team path for %s: %w", auditType.ID, err)}
		}

		command := fmt.Sprintf("cd %s && opencode run commissar", shellQuote(wslTeamDir))
		if err := manager.SendKeys(sessionName, windowName, command); err != nil {
			return LaunchFailedMsg{Err: fmt.Errorf("launch commissar for %s: %w", auditType.ID, err)}
		}

		cfg.Teams[auditType.ID] = config.TeamState{
			ID:         auditType.ID,
			Type:       auditType.ID,
			Prefix:     prefix,
			AgentCount: req.agentCount,
			Intensity:  req.intensity,
			Status:     "running",
			TmuxWindow: fmt.Sprintf("%s:%s", sessionName, windowName),
		}
	}

	cfg.Session.Name = sessionName
	cfg.Session.CreatedAt = deps.now().UTC().Format(time.RFC3339)
	cfg.Session.WorkingDir = req.cwd

	if err := cfg.Save(); err != nil {
		return LaunchFailedMsg{Err: fmt.Errorf("save launch config: %w", err)}
	}

	return LaunchCompleteMsg{}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
