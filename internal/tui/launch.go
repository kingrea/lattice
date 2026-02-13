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
	target     string
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
	initConfig          func(cwd string) (*config.Config, error)
	newTmuxManager      func() (launchTmuxManager, error)
	generateRoleSession func(params teams.RoleSessionParams) (string, error)
	buildAuditPlan      func(auditTypes []teams.AuditType, agentCount int, intensity int, startCounter int) (*teams.AuditPlan, error)
	translatePath       func(path string) (string, error)
	now                 func() time.Time
}

func defaultLaunchDeps() launchDeps {
	return launchDeps{
		initConfig:          config.Init,
		newTmuxManager:      newLaunchTmuxManager,
		generateRoleSession: teams.GenerateRoleSession,
		buildAuditPlan:      teams.BuildAuditPlan,
		translatePath:       tmux.TranslateToWSLPath,
		now:                 time.Now,
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

	plan, err := deps.buildAuditPlan(req.auditTypes, req.agentCount, req.intensity, cfg.BeadCounter)
	if err != nil {
		return LaunchFailedMsg{Err: fmt.Errorf("build audit plan: %w", err)}
	}
	cfg.BeadCounter = plan.FinalCounter

	manager, err := deps.newTmuxManager()
	if err != nil {
		return LaunchFailedMsg{Err: fmt.Errorf("initialize tmux manager: %w", err)}
	}

	sessionName := "lattice-" + deps.now().UTC().Format("20060102-150405")
	if err := manager.CreateSession(sessionName); err != nil {
		return LaunchFailedMsg{Err: fmt.Errorf("create tmux session: %w", err)}
	}

	if cfg.Epics == nil {
		cfg.Epics = map[string]config.EpicState{}
	}
	if cfg.Roles == nil {
		cfg.Roles = map[string]config.RoleState{}
	}

	target := strings.TrimSpace(req.target)
	if target == "" {
		target = filepath.Base(req.cwd)
	}

	for _, epic := range plan.Epics {
		auditType := epic.AuditType
		cfg.Epics[auditType.ID] = config.EpicState{
			BeadID:     epic.BeadID,
			AuditType:  auditType.ID,
			AuditName:  auditType.Name,
			AgentCount: req.agentCount,
			Intensity:  req.intensity,
			Status:     "running",
		}

		for idx, role := range epic.RoleBeads {
			roleState := config.RoleState{
				BeadID:     role.BeadID,
				EpicBeadID: epic.BeadID,
				CodeName:   role.CodeName,
				Title:      role.Title,
				Guidance:   role.Guidance,
				BeadPrefix: role.BeadPrefix,
				Order:      role.Order,
				Status:     "pending",
				Intensity:  req.intensity,
			}

			if idx == 0 {
				roleDir, err := deps.generateRoleSession(teams.RoleSessionParams{
					Cwd:          req.cwd,
					EpicBeadID:   epic.BeadID,
					RoleBeadID:   role.BeadID,
					RoleTitle:    role.Title,
					RoleGuidance: role.Guidance,
					Intensity:    req.intensity,
					BeadPrefix:   role.BeadPrefix,
					Target:       target,
					FocusAreas:   req.focusAreas,
					AuditTypeID:  auditType.ID,
					CodeName:     role.CodeName,
				})
				if err != nil {
					return LaunchFailedMsg{Err: fmt.Errorf("generate role session for %s/%s: %w", auditType.ID, role.CodeName, err)}
				}

				windowName := "audit-" + auditType.ID + "-" + role.CodeName
				if err := manager.CreateWindow(sessionName, windowName); err != nil {
					return LaunchFailedMsg{Err: fmt.Errorf("create tmux window for %s/%s: %w", auditType.ID, role.CodeName, err)}
				}

				wslRoleDir, err := deps.translatePath(roleDir)
				if err != nil {
					return LaunchFailedMsg{Err: fmt.Errorf("translate role session path for %s/%s: %w", auditType.ID, role.CodeName, err)}
				}

				command := fmt.Sprintf("cd %s && opencode run auditor", shellQuote(wslRoleDir))
				if err := manager.SendKeys(sessionName, windowName, command); err != nil {
					return LaunchFailedMsg{Err: fmt.Errorf("launch auditor for %s/%s: %w", auditType.ID, role.CodeName, err)}
				}

				roleState.Status = "running"
				roleState.TmuxWindow = fmt.Sprintf("%s:%s", sessionName, windowName)
			}

			cfg.Roles[role.BeadID] = roleState
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
