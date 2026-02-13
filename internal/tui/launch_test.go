package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lattice/internal/config"
	"lattice/internal/teams"
)

type fakeLaunchTmuxManager struct {
	sessionNames []string
	windowCalls  []string
	keyCalls     []string

	createSessionErr error
}

func (m *fakeLaunchTmuxManager) CreateSession(name string) error {
	m.sessionNames = append(m.sessionNames, name)
	return m.createSessionErr
}

func (m *fakeLaunchTmuxManager) CreateWindow(session, name string) error {
	m.windowCalls = append(m.windowCalls, fmt.Sprintf("%s:%s", session, name))
	return nil
}

func (m *fakeLaunchTmuxManager) SendKeys(session, window, command string) error {
	m.keyCalls = append(m.keyCalls, fmt.Sprintf("%s:%s:%s", session, window, command))
	return nil
}

func TestLaunchAuditOrchestratesSessionAndTeams(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	existingCfg, err := config.Init(workDir)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}
	existingCfg.BeadCounter = 41
	if err := existingCfg.Save(); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	fakeManager := &fakeLaunchTmuxManager{}
	fixedNow := time.Date(2026, time.February, 11, 14, 32, 1, 0, time.UTC)
	plan := &teams.AuditPlan{
		Epics: []teams.EpicBead{
			{
				BeadID:    "audit-plan-042",
				AuditType: teams.AuditTypes[0],
				RoleBeads: []teams.RoleBead{
					{BeadID: "audit-plan-043", CodeName: "alpha", Title: "Alpha", Guidance: "Do alpha", BeadPrefix: "perf-alpha", Order: 1},
					{BeadID: "audit-plan-044", CodeName: "bravo", Title: "Bravo", Guidance: "Do bravo", BeadPrefix: "perf-bravo", Order: 2},
				},
			},
			{
				BeadID:    "audit-plan-045",
				AuditType: teams.AuditTypes[1],
				RoleBeads: []teams.RoleBead{
					{BeadID: "audit-plan-046", CodeName: "alpha", Title: "Alpha 2", Guidance: "Do alpha 2", BeadPrefix: "mem-alpha", Order: 1},
					{BeadID: "audit-plan-047", CodeName: "bravo", Title: "Bravo 2", Guidance: "Do bravo 2", BeadPrefix: "mem-bravo", Order: 2},
				},
			},
		},
		FinalCounter: 47,
	}

	var roleSessionCalls []teams.RoleSessionParams
	var planStartCounter int

	deps := launchDeps{
		initConfig:     config.Init,
		newTmuxManager: func() (launchTmuxManager, error) { return fakeManager, nil },
		buildAuditPlan: func(_ []teams.AuditType, _ int, _ int, startCounter int) (*teams.AuditPlan, error) {
			planStartCounter = startCounter
			return plan, nil
		},
		generateRoleSession: func(params teams.RoleSessionParams) (string, error) {
			roleSessionCalls = append(roleSessionCalls, params)
			return filepath.Join(params.Cwd, config.DirName, "teams", params.AuditTypeID+"-"+params.CodeName), nil
		},
		translatePath: func(path string) (string, error) { return path, nil },
		now:           func() time.Time { return fixedNow },
	}

	req := launchRequest{
		cwd:        workDir,
		target:     "acme-app",
		auditTypes: []teams.AuditType{teams.AuditTypes[0], teams.AuditTypes[1]},
		agentCount: 2,
		intensity:  3,
		focusAreas: []string{"hot path", "heap growth"},
	}

	msg := launchAudit(req, deps)
	if _, ok := msg.(LaunchCompleteMsg); !ok {
		t.Fatalf("expected LaunchCompleteMsg, got %T", msg)
	}
	if planStartCounter != 41 {
		t.Fatalf("expected plan start counter 41, got %d", planStartCounter)
	}

	if len(fakeManager.sessionNames) != 1 || fakeManager.sessionNames[0] != "lattice-20260211-143201" {
		t.Fatalf("unexpected session names: %#v", fakeManager.sessionNames)
	}

	if len(fakeManager.windowCalls) != 2 {
		t.Fatalf("expected 2 window calls, got %d", len(fakeManager.windowCalls))
	}
	if fakeManager.windowCalls[0] != "lattice-20260211-143201:audit-perf-alpha" {
		t.Fatalf("unexpected first window call: %q", fakeManager.windowCalls[0])
	}
	if fakeManager.windowCalls[1] != "lattice-20260211-143201:audit-memleak-alpha" {
		t.Fatalf("unexpected second window call: %q", fakeManager.windowCalls[1])
	}

	if len(fakeManager.keyCalls) != 2 {
		t.Fatalf("expected 2 send-keys calls, got %d", len(fakeManager.keyCalls))
	}
	if !strings.Contains(fakeManager.keyCalls[0], "cd '") || !strings.Contains(fakeManager.keyCalls[0], "&& opencode run auditor") {
		t.Fatalf("unexpected first send-keys command: %q", fakeManager.keyCalls[0])
	}
	if strings.Contains(strings.Join(fakeManager.keyCalls, "\n"), "perf-bravo") || strings.Contains(strings.Join(fakeManager.keyCalls, "\n"), "mem-bravo") {
		t.Fatalf("pending roles should not have send-keys calls: %#v", fakeManager.keyCalls)
	}

	if len(roleSessionCalls) != 2 {
		t.Fatalf("expected 2 role session generations, got %d", len(roleSessionCalls))
	}

	cfg, err := config.Load(workDir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Session.Name != "lattice-20260211-143201" {
		t.Fatalf("unexpected session name in config: %q", cfg.Session.Name)
	}
	if cfg.BeadCounter != 47 {
		t.Fatalf("expected BeadCounter=47, got %d", cfg.BeadCounter)
	}
	if len(cfg.Epics) != 2 {
		t.Fatalf("expected 2 epics in config, got %d", len(cfg.Epics))
	}
	if got := cfg.Epics["perf"].BeadID; got != "audit-plan-042" {
		t.Fatalf("unexpected perf epic bead id: %q", got)
	}
	if got := cfg.Epics["memleak"].BeadID; got != "audit-plan-045" {
		t.Fatalf("unexpected memleak epic bead id: %q", got)
	}

	if len(cfg.Roles) != 4 {
		t.Fatalf("expected 4 roles in config, got %d", len(cfg.Roles))
	}
	if role := cfg.Roles["audit-plan-043"]; role.Status != "running" || role.TmuxWindow == "" {
		t.Fatalf("expected first perf role running with tmux window, got %+v", role)
	}
	if role := cfg.Roles["audit-plan-046"]; role.Status != "running" || role.TmuxWindow == "" {
		t.Fatalf("expected first mem role running with tmux window, got %+v", role)
	}
	if role := cfg.Roles["audit-plan-044"]; role.Status != "pending" || role.TmuxWindow != "" {
		t.Fatalf("expected second perf role pending with no tmux window, got %+v", role)
	}
	if role := cfg.Roles["audit-plan-047"]; role.Status != "pending" || role.TmuxWindow != "" {
		t.Fatalf("expected second mem role pending with no tmux window, got %+v", role)
	}
}

func TestLaunchAuditReturnsFailedMessageWhenSessionCreationFails(t *testing.T) {
	t.Parallel()

	fakeManager := &fakeLaunchTmuxManager{createSessionErr: fmt.Errorf("boom")}

	deps := launchDeps{
		initConfig:          config.Init,
		newTmuxManager:      func() (launchTmuxManager, error) { return fakeManager, nil },
		generateRoleSession: teams.GenerateRoleSession,
		buildAuditPlan:      teams.BuildAuditPlan,
		translatePath:       func(path string) (string, error) { return path, nil },
		now:                 time.Now,
	}

	msg := launchAudit(launchRequest{
		cwd:        t.TempDir(),
		auditTypes: []teams.AuditType{teams.AuditTypes[0]},
		agentCount: 1,
		intensity:  1,
	}, deps)

	failed, ok := msg.(LaunchFailedMsg)
	if !ok {
		t.Fatalf("expected LaunchFailedMsg, got %T", msg)
	}
	if !strings.Contains(failed.Err.Error(), "create tmux session") {
		t.Fatalf("unexpected error: %v", failed.Err)
	}
}
