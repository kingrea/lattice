package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lattice/internal/config"
	"lattice/internal/teams"
	"lattice/internal/tmux"
)

// SchedulerDeps defines all external dependencies for role advancement.
type SchedulerDeps struct {
	GenerateRoleSession func(params teams.RoleSessionParams) (string, error)
	TranslatePath       func(path string) (string, error)
	TmuxManager         launchTmuxManager
	CheckTmuxWindow     func(sessionName, windowName string) bool
	Now                 func() time.Time
}

// ScheduledRole captures one role that was launched by the scheduler.
type ScheduledRole struct {
	RoleBeadID string
	EpicBeadID string
	AuditType  string
	CodeName   string
	WindowName string
	SessionDir string
	LaunchedAt time.Time
}

// SchedulerResult reports all transitions performed in one scheduling pass.
type SchedulerResult struct {
	Launched  []ScheduledRole
	Completed []string
	Failed    []string
	AllDone   bool
}

// CheckAndAdvanceRoles advances role state machines and launches next roles.
func CheckAndAdvanceRoles(cwd string, cfg *config.Config, sessionName string, plan *teams.AuditPlan, deps SchedulerDeps) (SchedulerResult, error) {
	if strings.TrimSpace(cwd) == "" {
		return SchedulerResult{}, fmt.Errorf("working directory must not be empty")
	}
	if cfg == nil {
		return SchedulerResult{}, fmt.Errorf("config must not be nil")
	}
	if plan == nil {
		return SchedulerResult{}, fmt.Errorf("audit plan must not be nil")
	}
	if strings.TrimSpace(sessionName) == "" {
		return SchedulerResult{}, fmt.Errorf("session name must not be empty")
	}

	resolvedDeps, err := resolveSchedulerDeps(deps)
	if err != nil {
		return SchedulerResult{}, err
	}

	if cfg.Epics == nil {
		cfg.Epics = map[string]config.EpicState{}
	}
	if cfg.Roles == nil {
		cfg.Roles = map[string]config.RoleState{}
	}

	result := SchedulerResult{}
	for _, epic := range plan.Epics {
		auditTypeID := strings.TrimSpace(epic.AuditType.ID)
		if auditTypeID == "" {
			continue
		}

		if _, ok := cfg.Epics[auditTypeID]; !ok {
			cfg.Epics[auditTypeID] = config.EpicState{
				BeadID:     epic.BeadID,
				AuditType:  auditTypeID,
				AuditName:  epic.AuditType.Name,
				AgentCount: len(epic.RoleBeads),
				Status:     "running",
			}
		}

		roleBeads := orderedRoleBeads(epic, cfg)
		for idx, roleBead := range roleBeads {
			state := ensureRoleState(cfg.Roles[roleBead.BeadID], epic, roleBead)
			status := normalizeRoleStatus(state.Status)

			switch status {
			case "running":
				windowName := roleWindowName(auditTypeID, roleBead.CodeName)
				if resolvedDeps.CheckTmuxWindow(sessionName, windowName) {
					cfg.Roles[roleBead.BeadID] = state
					continue
				}

				teamStatus, err := readRoleTeamStatus(cwd, state, roleBead.BeadID)
				if err != nil {
					return result, fmt.Errorf("read role status for %s/%s: %w", auditTypeID, roleBead.CodeName, err)
				}

				if teamStatus == "complete" {
					state.Status = "complete"
					state.TmuxWindow = ""
					cfg.Roles[roleBead.BeadID] = state
					result.Completed = append(result.Completed, roleBead.BeadID)
				} else {
					state.Status = "failed"
					state.TmuxWindow = ""
					cfg.Roles[roleBead.BeadID] = state
					result.Failed = append(result.Failed, roleBead.BeadID)
				}

			case "pending":
				prevStatus := "complete"
				if idx > 0 {
					prevRole := cfg.Roles[roleBeads[idx-1].BeadID]
					prevStatus = normalizeRoleStatus(prevRole.Status)
				}

				if prevStatus == "failed" {
					cfg.Roles[roleBead.BeadID] = state
					continue
				}
				if prevStatus != "complete" {
					cfg.Roles[roleBead.BeadID] = state
					continue
				}

				launchedRole, updatedState, err := launchScheduledRole(cwd, sessionName, epic, state, roleBead, resolvedDeps)
				if err != nil {
					return result, err
				}

				cfg.Roles[roleBead.BeadID] = updatedState
				result.Launched = append(result.Launched, launchedRole)

			case "complete", "failed":
				cfg.Roles[roleBead.BeadID] = state
			default:
				state.Status = "pending"
				cfg.Roles[roleBead.BeadID] = state
			}
		}

		epicState := cfg.Epics[auditTypeID]
		epicState.BeadID = epic.BeadID
		epicState.AuditType = auditTypeID
		epicState.AuditName = epic.AuditType.Name
		epicState.Status = deriveEpicStateStatus(roleBeads, cfg)
		cfg.Epics[auditTypeID] = epicState
	}

	result.AllDone = allRolesTerminal(plan, cfg)
	return result, nil
}

func resolveSchedulerDeps(deps SchedulerDeps) (SchedulerDeps, error) {
	resolved := deps
	if resolved.GenerateRoleSession == nil {
		resolved.GenerateRoleSession = teams.GenerateRoleSession
	}
	if resolved.TranslatePath == nil {
		resolved.TranslatePath = tmux.TranslateToWSLPath
	}
	if resolved.Now == nil {
		resolved.Now = time.Now
	}
	if resolved.TmuxManager == nil {
		manager, err := newLaunchTmuxManager()
		if err != nil {
			return SchedulerDeps{}, fmt.Errorf("initialize tmux manager: %w", err)
		}
		resolved.TmuxManager = manager
	}
	if resolved.CheckTmuxWindow == nil {
		resolved.CheckTmuxWindow = tmuxWindowChecker(resolved.TmuxManager)
	}

	return resolved, nil
}

func tmuxWindowChecker(manager launchTmuxManager) func(sessionName, windowName string) bool {
	lister, ok := manager.(interface {
		ListWindows(session string) ([]tmux.WindowInfo, error)
	})
	if !ok {
		return func(_, _ string) bool { return false }
	}

	return func(sessionName, windowName string) bool {
		windows, err := lister.ListWindows(sessionName)
		if err != nil {
			return false
		}

		for _, window := range windows {
			if window.Name == windowName {
				return true
			}
		}

		return false
	}
}

func orderedRoleBeads(epic teams.EpicBead, cfg *config.Config) []teams.RoleBead {
	roleBeads := append([]teams.RoleBead(nil), epic.RoleBeads...)
	sort.SliceStable(roleBeads, func(i, j int) bool {
		leftOrder := roleBeads[i].Order
		rightOrder := roleBeads[j].Order

		if leftOrder == 0 {
			leftOrder = cfg.Roles[roleBeads[i].BeadID].Order
		}
		if rightOrder == 0 {
			rightOrder = cfg.Roles[roleBeads[j].BeadID].Order
		}

		if leftOrder == rightOrder {
			return roleBeads[i].BeadID < roleBeads[j].BeadID
		}

		return leftOrder < rightOrder
	})

	return roleBeads
}

func ensureRoleState(state config.RoleState, epic teams.EpicBead, role teams.RoleBead) config.RoleState {
	state.BeadID = role.BeadID
	state.EpicBeadID = epic.BeadID
	state.CodeName = role.CodeName
	state.Title = role.Title
	state.Guidance = role.Guidance
	state.BeadPrefix = role.BeadPrefix
	if state.Order == 0 {
		state.Order = role.Order
	}
	if strings.TrimSpace(state.Status) == "" {
		state.Status = "pending"
	}

	return state
}

func launchScheduledRole(cwd string, sessionName string, epic teams.EpicBead, state config.RoleState, role teams.RoleBead, deps SchedulerDeps) (ScheduledRole, config.RoleState, error) {
	params := teams.RoleSessionParams{
		Cwd:          cwd,
		EpicBeadID:   epic.BeadID,
		RoleBeadID:   role.BeadID,
		RoleTitle:    state.Title,
		RoleGuidance: state.Guidance,
		Intensity:    state.Intensity,
		BeadPrefix:   state.BeadPrefix,
		AuditTypeID:  epic.AuditType.ID,
		CodeName:     state.CodeName,
	}

	roleDir, err := deps.GenerateRoleSession(params)
	if err != nil {
		return ScheduledRole{}, state, fmt.Errorf("generate role session for %s/%s: %w", epic.AuditType.ID, role.CodeName, err)
	}

	windowName := roleWindowName(epic.AuditType.ID, role.CodeName)
	if err := deps.TmuxManager.CreateWindow(sessionName, windowName); err != nil {
		return ScheduledRole{}, state, fmt.Errorf("create tmux window for %s/%s: %w", epic.AuditType.ID, role.CodeName, err)
	}

	wslRoleDir, err := deps.TranslatePath(roleDir)
	if err != nil {
		return ScheduledRole{}, state, fmt.Errorf("translate role session path for %s/%s: %w", epic.AuditType.ID, role.CodeName, err)
	}

	command := fmt.Sprintf("cd %s && opencode run auditor", shellQuote(wslRoleDir))
	if err := deps.TmuxManager.SendKeys(sessionName, windowName, command); err != nil {
		return ScheduledRole{}, state, fmt.Errorf("launch auditor for %s/%s: %w", epic.AuditType.ID, role.CodeName, err)
	}

	state.Status = "running"
	state.TmuxWindow = fmt.Sprintf("%s:%s", sessionName, windowName)

	now := deps.Now().UTC()
	return ScheduledRole{
		RoleBeadID: role.BeadID,
		EpicBeadID: epic.BeadID,
		AuditType:  epic.AuditType.ID,
		CodeName:   role.CodeName,
		WindowName: windowName,
		SessionDir: roleDir,
		LaunchedAt: now,
	}, state, nil
}

func readRoleTeamStatus(cwd string, role config.RoleState, roleKey string) (string, error) {
	for _, dir := range dashboardRoleDirectories(role, roleKey) {
		teamFile := filepath.Join(cwd, config.DirName, "teams", dir, ".team")
		teamData, err := readTeamFile(teamFile)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}

		return strings.ToLower(strings.TrimSpace(teamData["status"])), nil
	}

	return "", nil
}

func deriveEpicStateStatus(roleBeads []teams.RoleBead, cfg *config.Config) string {
	if len(roleBeads) == 0 {
		return "running"
	}

	hasRunningOrPending := false
	hasFailed := false
	allComplete := true

	for _, role := range roleBeads {
		status := normalizeRoleStatus(cfg.Roles[role.BeadID].Status)
		switch status {
		case "running", "pending":
			hasRunningOrPending = true
			allComplete = false
		case "complete":
			continue
		case "failed":
			hasFailed = true
			allComplete = false
		default:
			hasRunningOrPending = true
			allComplete = false
		}
	}

	if hasRunningOrPending {
		if hasFailed {
			return "blocked"
		}
		return "running"
	}
	if hasFailed {
		return "failed"
	}
	if allComplete {
		return "complete"
	}

	return "running"
}

func allRolesTerminal(plan *teams.AuditPlan, cfg *config.Config) bool {
	if plan == nil || len(plan.Epics) == 0 {
		return false
	}

	hasRoles := false
	for _, epic := range plan.Epics {
		for _, role := range epic.RoleBeads {
			hasRoles = true
			switch normalizeRoleStatus(cfg.Roles[role.BeadID].Status) {
			case "complete", "failed":
				continue
			default:
				return false
			}
		}
	}

	return hasRoles
}

func roleWindowName(auditTypeID, codeName string) string {
	return "audit-" + strings.TrimSpace(auditTypeID) + "-" + strings.TrimSpace(codeName)
}
