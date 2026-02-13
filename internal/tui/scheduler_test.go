package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"lattice/internal/config"
	"lattice/internal/teams"
)

func TestCheckAndAdvanceRolesRunningCompletesThenNextLaunches(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	cfg := baseSchedulerConfig()
	plan := twoRolePlan("perf", "perf-alpha", "perf-bravo")

	cfg.Roles["r1"] = config.RoleState{BeadID: "r1", EpicBeadID: "e1", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: "perf-alpha", Order: 1, Status: "running", TmuxWindow: "sess:audit-perf-alpha", Intensity: 2}
	cfg.Roles["r2"] = config.RoleState{BeadID: "r2", EpicBeadID: "e1", CodeName: "bravo", Title: "Bravo", Guidance: "B", BeadPrefix: "perf-bravo", Order: 2, Status: "pending", Intensity: 2}
	cfg.Epics["perf"] = config.EpicState{BeadID: "e1", AuditType: "perf", AuditName: "Performance", Status: "running"}

	writeRoleTeamStatus(t, cwd, "perf-alpha", "complete")

	manager := &fakeLaunchTmuxManager{}
	res, err := CheckAndAdvanceRoles(cwd, cfg, "sess", plan, SchedulerDeps{
		GenerateRoleSession: func(params teams.RoleSessionParams) (string, error) {
			return filepath.Join(params.Cwd, config.DirName, "teams", params.AuditTypeID+"-"+params.CodeName), nil
		},
		TranslatePath: func(path string) (string, error) { return path, nil },
		TmuxManager:   manager,
		CheckTmuxWindow: func(sessionName, windowName string) bool {
			return false
		},
		Now: func() time.Time { return time.Date(2026, time.February, 13, 1, 2, 3, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("CheckAndAdvanceRoles() error = %v", err)
	}

	if len(res.Completed) != 1 || res.Completed[0] != "r1" {
		t.Fatalf("unexpected completed roles: %#v", res.Completed)
	}
	if len(res.Launched) != 1 || res.Launched[0].RoleBeadID != "r2" {
		t.Fatalf("unexpected launched roles: %#v", res.Launched)
	}
	if cfg.Roles["r1"].Status != "complete" {
		t.Fatalf("expected r1 complete, got %q", cfg.Roles["r1"].Status)
	}
	if cfg.Roles["r2"].Status != "running" {
		t.Fatalf("expected r2 running, got %q", cfg.Roles["r2"].Status)
	}
	if len(manager.windowCalls) != 1 || manager.windowCalls[0] != "sess:audit-perf-bravo" {
		t.Fatalf("unexpected window calls: %#v", manager.windowCalls)
	}
}

func TestCheckAndAdvanceRolesWindowGoneWithoutCompleteMarksFailed(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	cfg := baseSchedulerConfig()
	plan := oneRolePlan("perf", "perf-alpha")

	cfg.Roles["r1"] = config.RoleState{BeadID: "r1", EpicBeadID: "e1", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: "perf-alpha", Order: 1, Status: "running"}
	cfg.Epics["perf"] = config.EpicState{BeadID: "e1", AuditType: "perf", AuditName: "Performance", Status: "running"}

	res, err := CheckAndAdvanceRoles(cwd, cfg, "sess", plan, SchedulerDeps{
		GenerateRoleSession: func(params teams.RoleSessionParams) (string, error) { return "", nil },
		TranslatePath:       func(path string) (string, error) { return path, nil },
		TmuxManager:         &fakeLaunchTmuxManager{},
		CheckTmuxWindow:     func(sessionName, windowName string) bool { return false },
		Now:                 time.Now,
	})
	if err != nil {
		t.Fatalf("CheckAndAdvanceRoles() error = %v", err)
	}

	if len(res.Failed) != 1 || res.Failed[0] != "r1" {
		t.Fatalf("unexpected failed roles: %#v", res.Failed)
	}
	if cfg.Roles["r1"].Status != "failed" {
		t.Fatalf("expected r1 failed, got %q", cfg.Roles["r1"].Status)
	}
	if cfg.Epics["perf"].Status != "failed" {
		t.Fatalf("expected epic failed, got %q", cfg.Epics["perf"].Status)
	}
}

func TestCheckAndAdvanceRolesLastRoleCompleteSetsAllDone(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	cfg := baseSchedulerConfig()
	plan := oneRolePlan("perf", "perf-alpha")

	cfg.Roles["r1"] = config.RoleState{BeadID: "r1", EpicBeadID: "e1", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: "perf-alpha", Order: 1, Status: "running"}
	cfg.Epics["perf"] = config.EpicState{BeadID: "e1", AuditType: "perf", AuditName: "Performance", Status: "running"}
	writeRoleTeamStatus(t, cwd, "perf-alpha", "complete")

	res, err := CheckAndAdvanceRoles(cwd, cfg, "sess", plan, SchedulerDeps{
		GenerateRoleSession: func(params teams.RoleSessionParams) (string, error) { return "", nil },
		TranslatePath:       func(path string) (string, error) { return path, nil },
		TmuxManager:         &fakeLaunchTmuxManager{},
		CheckTmuxWindow:     func(sessionName, windowName string) bool { return false },
		Now:                 time.Now,
	})
	if err != nil {
		t.Fatalf("CheckAndAdvanceRoles() error = %v", err)
	}

	if !res.AllDone {
		t.Fatalf("expected AllDone=true")
	}
	if cfg.Epics["perf"].Status != "complete" {
		t.Fatalf("expected epic complete, got %q", cfg.Epics["perf"].Status)
	}
}

func TestCheckAndAdvanceRolesFailureBlocksSubsequentPending(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	cfg := baseSchedulerConfig()
	plan := twoRolePlan("perf", "perf-alpha", "perf-bravo")

	cfg.Roles["r1"] = config.RoleState{BeadID: "r1", EpicBeadID: "e1", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: "perf-alpha", Order: 1, Status: "running"}
	cfg.Roles["r2"] = config.RoleState{BeadID: "r2", EpicBeadID: "e1", CodeName: "bravo", Title: "Bravo", Guidance: "B", BeadPrefix: "perf-bravo", Order: 2, Status: "pending"}
	cfg.Epics["perf"] = config.EpicState{BeadID: "e1", AuditType: "perf", AuditName: "Performance", Status: "running"}

	manager := &fakeLaunchTmuxManager{}
	res, err := CheckAndAdvanceRoles(cwd, cfg, "sess", plan, SchedulerDeps{
		GenerateRoleSession: func(params teams.RoleSessionParams) (string, error) {
			return filepath.Join(params.Cwd, config.DirName, "teams", params.AuditTypeID+"-"+params.CodeName), nil
		},
		TranslatePath:   func(path string) (string, error) { return path, nil },
		TmuxManager:     manager,
		CheckTmuxWindow: func(sessionName, windowName string) bool { return false },
		Now:             time.Now,
	})
	if err != nil {
		t.Fatalf("CheckAndAdvanceRoles() error = %v", err)
	}

	if len(res.Failed) != 1 || res.Failed[0] != "r1" {
		t.Fatalf("unexpected failed roles: %#v", res.Failed)
	}
	if len(res.Launched) != 0 {
		t.Fatalf("expected no launches, got %#v", res.Launched)
	}
	if cfg.Roles["r2"].Status != "pending" {
		t.Fatalf("expected r2 pending, got %q", cfg.Roles["r2"].Status)
	}
	if len(manager.windowCalls) != 0 {
		t.Fatalf("expected no tmux windows, got %#v", manager.windowCalls)
	}
}

func TestCheckAndAdvanceRolesIdempotentNoDuplicateLaunch(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	cfg := baseSchedulerConfig()
	plan := twoRolePlan("perf", "perf-alpha", "perf-bravo")

	cfg.Roles["r1"] = config.RoleState{BeadID: "r1", EpicBeadID: "e1", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: "perf-alpha", Order: 1, Status: "complete"}
	cfg.Roles["r2"] = config.RoleState{BeadID: "r2", EpicBeadID: "e1", CodeName: "bravo", Title: "Bravo", Guidance: "B", BeadPrefix: "perf-bravo", Order: 2, Status: "pending"}
	cfg.Epics["perf"] = config.EpicState{BeadID: "e1", AuditType: "perf", AuditName: "Performance", Status: "running"}

	manager := &fakeLaunchTmuxManager{}
	deps := SchedulerDeps{
		GenerateRoleSession: func(params teams.RoleSessionParams) (string, error) {
			return filepath.Join(params.Cwd, config.DirName, "teams", params.AuditTypeID+"-"+params.CodeName), nil
		},
		TranslatePath:   func(path string) (string, error) { return path, nil },
		TmuxManager:     manager,
		CheckTmuxWindow: func(sessionName, windowName string) bool { return windowName == "audit-perf-bravo" },
		Now:             time.Now,
	}

	first, err := CheckAndAdvanceRoles(cwd, cfg, "sess", plan, deps)
	if err != nil {
		t.Fatalf("first CheckAndAdvanceRoles() error = %v", err)
	}
	second, err := CheckAndAdvanceRoles(cwd, cfg, "sess", plan, deps)
	if err != nil {
		t.Fatalf("second CheckAndAdvanceRoles() error = %v", err)
	}

	if len(first.Launched) != 1 {
		t.Fatalf("expected first call launch, got %#v", first.Launched)
	}
	if len(second.Launched) != 0 {
		t.Fatalf("expected second call no launch, got %#v", second.Launched)
	}
	if len(manager.windowCalls) != 1 {
		t.Fatalf("expected one tmux window creation, got %#v", manager.windowCalls)
	}
}

func TestCheckAndAdvanceRolesCrashRecoveryMarksFailed(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	cfg := baseSchedulerConfig()
	plan := oneRolePlan("perf", "perf-alpha")

	cfg.Roles["r1"] = config.RoleState{BeadID: "r1", EpicBeadID: "e1", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: "perf-alpha", Order: 1, Status: "running"}
	cfg.Epics["perf"] = config.EpicState{BeadID: "e1", AuditType: "perf", AuditName: "Performance", Status: "running"}

	res, err := CheckAndAdvanceRoles(cwd, cfg, "sess", plan, SchedulerDeps{
		GenerateRoleSession: func(params teams.RoleSessionParams) (string, error) { return "", nil },
		TranslatePath:       func(path string) (string, error) { return path, nil },
		TmuxManager:         &fakeLaunchTmuxManager{},
		CheckTmuxWindow:     func(sessionName, windowName string) bool { return false },
		Now:                 time.Now,
	})
	if err != nil {
		t.Fatalf("CheckAndAdvanceRoles() error = %v", err)
	}

	if len(res.Failed) != 1 || res.Failed[0] != "r1" {
		t.Fatalf("unexpected failed roles: %#v", res.Failed)
	}
	if !res.AllDone {
		t.Fatalf("expected AllDone=true when only role failed")
	}
}

func TestCheckAndAdvanceRolesMultipleEpicsAdvanceIndependently(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	cfg := baseSchedulerConfig()
	plan := &teams.AuditPlan{Epics: []teams.EpicBead{
		{BeadID: "e1", AuditType: teams.AuditType{ID: "perf", Name: "Performance"}, RoleBeads: []teams.RoleBead{{BeadID: "r1", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: "perf-alpha", Order: 1}, {BeadID: "r2", CodeName: "bravo", Title: "Bravo", Guidance: "B", BeadPrefix: "perf-bravo", Order: 2}}},
		{BeadID: "e2", AuditType: teams.AuditType{ID: "memleak", Name: "Memory"}, RoleBeads: []teams.RoleBead{{BeadID: "r3", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: "mem-alpha", Order: 1}, {BeadID: "r4", CodeName: "bravo", Title: "Bravo", Guidance: "B", BeadPrefix: "mem-bravo", Order: 2}}},
	}}

	cfg.Roles["r1"] = config.RoleState{BeadID: "r1", EpicBeadID: "e1", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: "perf-alpha", Order: 1, Status: "running"}
	cfg.Roles["r2"] = config.RoleState{BeadID: "r2", EpicBeadID: "e1", CodeName: "bravo", Title: "Bravo", Guidance: "B", BeadPrefix: "perf-bravo", Order: 2, Status: "pending"}
	cfg.Roles["r3"] = config.RoleState{BeadID: "r3", EpicBeadID: "e2", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: "mem-alpha", Order: 1, Status: "running"}
	cfg.Roles["r4"] = config.RoleState{BeadID: "r4", EpicBeadID: "e2", CodeName: "bravo", Title: "Bravo", Guidance: "B", BeadPrefix: "mem-bravo", Order: 2, Status: "pending"}
	cfg.Epics["perf"] = config.EpicState{BeadID: "e1", AuditType: "perf", AuditName: "Performance", Status: "running"}
	cfg.Epics["memleak"] = config.EpicState{BeadID: "e2", AuditType: "memleak", AuditName: "Memory", Status: "running"}

	writeRoleTeamStatus(t, cwd, "perf-alpha", "complete")

	manager := &fakeLaunchTmuxManager{}
	res, err := CheckAndAdvanceRoles(cwd, cfg, "sess", plan, SchedulerDeps{
		GenerateRoleSession: func(params teams.RoleSessionParams) (string, error) {
			return filepath.Join(params.Cwd, config.DirName, "teams", params.AuditTypeID+"-"+params.CodeName), nil
		},
		TranslatePath: func(path string) (string, error) { return path, nil },
		TmuxManager:   manager,
		CheckTmuxWindow: func(sessionName, windowName string) bool {
			return windowName == "audit-memleak-alpha"
		},
		Now: time.Now,
	})
	if err != nil {
		t.Fatalf("CheckAndAdvanceRoles() error = %v", err)
	}

	if len(res.Completed) != 1 || res.Completed[0] != "r1" {
		t.Fatalf("unexpected completed: %#v", res.Completed)
	}
	if len(res.Launched) != 1 || res.Launched[0].RoleBeadID != "r2" {
		t.Fatalf("unexpected launched: %#v", res.Launched)
	}
	if cfg.Roles["r4"].Status != "pending" {
		t.Fatalf("expected memleak r4 pending, got %q", cfg.Roles["r4"].Status)
	}
	if len(manager.windowCalls) != 1 || manager.windowCalls[0] != "sess:audit-perf-bravo" {
		t.Fatalf("unexpected windows: %#v", manager.windowCalls)
	}
}

func baseSchedulerConfig() *config.Config {
	return &config.Config{
		Epics: map[string]config.EpicState{},
		Roles: map[string]config.RoleState{},
	}
}

func oneRolePlan(auditTypeID, beadPrefix string) *teams.AuditPlan {
	return &teams.AuditPlan{Epics: []teams.EpicBead{
		{
			BeadID:    "e1",
			AuditType: teams.AuditType{ID: auditTypeID, Name: "Performance"},
			RoleBeads: []teams.RoleBead{{BeadID: "r1", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: beadPrefix, Order: 1}},
		},
	}}
}

func twoRolePlan(auditTypeID, firstPrefix, secondPrefix string) *teams.AuditPlan {
	return &teams.AuditPlan{Epics: []teams.EpicBead{
		{
			BeadID:    "e1",
			AuditType: teams.AuditType{ID: auditTypeID, Name: "Performance"},
			RoleBeads: []teams.RoleBead{
				{BeadID: "r1", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: firstPrefix, Order: 1},
				{BeadID: "r2", CodeName: "bravo", Title: "Bravo", Guidance: "B", BeadPrefix: secondPrefix, Order: 2},
			},
		},
	}}
}

func writeRoleTeamStatus(t *testing.T, cwd, dirName, status string) {
	t.Helper()

	dir := filepath.Join(cwd, config.DirName, "teams", dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := []byte("status=" + status + "\n")
	if err := os.WriteFile(filepath.Join(dir, ".team"), content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
