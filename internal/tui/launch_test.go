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
	fakeManager := &fakeLaunchTmuxManager{}
	fixedNow := time.Date(2026, time.February, 11, 14, 32, 1, 0, time.UTC)

	deps := launchDeps{
		initConfig:         config.Init,
		newTmuxManager:     func() (launchTmuxManager, error) { return fakeManager, nil },
		allocateBeadPrefix: teams.AllocateBeadPrefix,
		generateTeam: func(params teams.GenerateParams) (string, error) {
			teamDir := filepath.Join(params.WorkingDir, config.DirName, "teams", "audit-"+params.AuditType.ID)
			return teamDir, nil
		},
		translatePath: func(path string) (string, error) { return path, nil },
		now:           func() time.Time { return fixedNow },
	}

	req := launchRequest{
		cwd:        workDir,
		auditTypes: []teams.AuditType{teams.AuditTypes[0], teams.AuditTypes[1]},
		agentCount: 2,
		intensity:  3,
	}

	msg := launchAudit(req, deps)
	if _, ok := msg.(LaunchCompleteMsg); !ok {
		t.Fatalf("expected LaunchCompleteMsg, got %T", msg)
	}

	if len(fakeManager.sessionNames) != 1 || fakeManager.sessionNames[0] != "lattice-20260211-143201" {
		t.Fatalf("unexpected session names: %#v", fakeManager.sessionNames)
	}

	if len(fakeManager.windowCalls) != 2 {
		t.Fatalf("expected 2 window calls, got %d", len(fakeManager.windowCalls))
	}
	if fakeManager.windowCalls[0] != "lattice-20260211-143201:audit-perf" {
		t.Fatalf("unexpected first window call: %q", fakeManager.windowCalls[0])
	}
	if fakeManager.windowCalls[1] != "lattice-20260211-143201:audit-memleak" {
		t.Fatalf("unexpected second window call: %q", fakeManager.windowCalls[1])
	}

	if len(fakeManager.keyCalls) != 2 {
		t.Fatalf("expected 2 send-keys calls, got %d", len(fakeManager.keyCalls))
	}
	if !strings.Contains(fakeManager.keyCalls[0], "cd '") || !strings.Contains(fakeManager.keyCalls[0], "&& opencode run commissar") {
		t.Fatalf("unexpected first send-keys command: %q", fakeManager.keyCalls[0])
	}

	cfg, err := config.Load(workDir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Session.Name != "lattice-20260211-143201" {
		t.Fatalf("unexpected session name in config: %q", cfg.Session.Name)
	}
	if len(cfg.Teams) != 2 {
		t.Fatalf("expected 2 teams in config, got %d", len(cfg.Teams))
	}
	if got := cfg.Teams["perf"].Prefix; got != "perf-1" {
		t.Fatalf("unexpected perf prefix: %q", got)
	}
	if got := cfg.Teams["memleak"].Prefix; got != "mem-2" {
		t.Fatalf("unexpected memleak prefix: %q", got)
	}
}

func TestLaunchAuditReturnsFailedMessageWhenSessionCreationFails(t *testing.T) {
	t.Parallel()

	fakeManager := &fakeLaunchTmuxManager{createSessionErr: fmt.Errorf("boom")}

	deps := launchDeps{
		initConfig:         config.Init,
		newTmuxManager:     func() (launchTmuxManager, error) { return fakeManager, nil },
		allocateBeadPrefix: teams.AllocateBeadPrefix,
		generateTeam:       teams.Generate,
		translatePath:      func(path string) (string, error) { return path, nil },
		now:                time.Now,
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
