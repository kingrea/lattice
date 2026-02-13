package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lattice/internal/config"
	"lattice/internal/teams"
)

func TestLoadDashboardSnapshotReadsEpicRoles(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	cfg, err := config.Init(workDir)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	cfg.Session.Name = "lattice-20260211-101010"
	cfg.Epics = map[string]config.EpicState{
		"perf": {
			BeadID:    "audit-plan-001",
			AuditType: "perf",
			AuditName: "Performance Audit",
			Status:    "running",
		},
	}
	cfg.Roles = map[string]config.RoleState{
		"alpha": {
			BeadID:     "audit-plan-002",
			EpicBeadID: "audit-plan-001",
			CodeName:   "alpha",
			Title:      "Sr. Perf Spec",
			Status:     "running",
			Intensity:  3,
			BeadPrefix: "perf-121",
		},
		"bravo": {
			BeadID:     "audit-plan-003",
			EpicBeadID: "audit-plan-001",
			CodeName:   "bravo",
			Title:      "Staff Perf Eng",
			Status:     "pending",
			Intensity:  3,
			BeadPrefix: "perf-122",
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	roleDir := filepath.Join(workDir, config.DirName, "teams", "perf-alpha")
	if err := os.MkdirAll(roleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(roleDir, ".team"), []byte("team=perf-alpha\nintensity=3\ncurrent_loop=1\nstatus=active\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	now := time.Date(2026, time.February, 11, 10, 10, 10, 0, time.UTC)
	snapshot, err := loadDashboardSnapshot(workDir, now)
	if err != nil {
		t.Fatalf("loadDashboardSnapshot() returned error: %v", err)
	}

	if snapshot.SessionName != "lattice-20260211-101010" {
		t.Fatalf("unexpected session name: %q", snapshot.SessionName)
	}
	if len(snapshot.Epics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(snapshot.Epics))
	}

	epic := snapshot.Epics[0]
	if epic.EpicName != "Performance Audit" {
		t.Fatalf("unexpected epic name: %q", epic.EpicName)
	}
	if epic.Status != "running" {
		t.Fatalf("unexpected epic status: %q", epic.Status)
	}
	if epic.RolesTotal != 2 || epic.RolesComplete != 0 || epic.RolesFailed != 0 {
		t.Fatalf("unexpected role counts: %+v", epic)
	}
	if len(epic.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(epic.Roles))
	}

	alpha := epic.Roles[0]
	if alpha.CodeName != "alpha" {
		t.Fatalf("unexpected role order/code name: %q", alpha.CodeName)
	}
	if alpha.Status != "running" {
		t.Fatalf("unexpected role status: %q", alpha.Status)
	}
	if alpha.CurrentLoop != 1 {
		t.Fatalf("unexpected current loop: %d", alpha.CurrentLoop)
	}
}

func TestRenderEpicTableShowsHierarchy(t *testing.T) {
	t.Parallel()

	model := NewDashboardModel("/tmp/work", DefaultStyles(), DefaultKeyMap())
	model.epics = []dashboardEpicStatus{{
		EpicName:      "Performance Audit",
		Status:        "running",
		RolesTotal:    3,
		RolesComplete: 1,
		Roles: []dashboardRoleStatus{
			{CodeName: "alpha", Title: "Sr. Perf Spec", Status: "complete", CurrentLoop: 3, Intensity: 3},
			{CodeName: "bravo", Title: "Staff Perf Eng", Status: "running", CurrentLoop: 1, Intensity: 3},
		},
	}}

	view := model.renderEpicTable()
	for _, fragment := range []string{
		"EPIC",
		"Performance Audit",
		"1/3 roles done",
		"  alpha (Sr. Perf Spec)",
		"  bravo (Staff Perf Eng)",
		"loop 1/3",
	} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("expected epic table to include %q", fragment)
		}
	}
}

func TestFormatRoleProgress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role dashboardRoleStatus
		want string
	}{
		{
			name: "pending without loop uses dash",
			role: dashboardRoleStatus{Status: "pending", CurrentLoop: 0, Intensity: 3},
			want: "-",
		},
		{
			name: "running shows loop progress",
			role: dashboardRoleStatus{Status: "running", CurrentLoop: 1, Intensity: 3},
			want: "loop 1/3",
		},
		{
			name: "no intensity uses dash",
			role: dashboardRoleStatus{Status: "running", CurrentLoop: 1, Intensity: 0},
			want: "-",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatRoleProgress(tt.role)
			if got != tt.want {
				t.Fatalf("formatRoleProgress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDashboardRoleDirectoriesPreferAuditTypeNameThenLegacyPrefix(t *testing.T) {
	t.Parallel()

	dirs := dashboardRoleDirectories(config.RoleState{BeadPrefix: "perf-senior-performance-specialist", CodeName: "alpha"}, "role-1")
	if len(dirs) != 2 {
		t.Fatalf("expected 2 directory candidates, got %d", len(dirs))
	}
	if dirs[0] != "perf-alpha" {
		t.Fatalf("unexpected primary directory: %q", dirs[0])
	}
	if dirs[1] != "perf-senior-performance-specialist-alpha" {
		t.Fatalf("unexpected fallback directory: %q", dirs[1])
	}
}

func TestLoadDashboardSnapshotBackwardCompatibleTeams(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	cfg, err := config.Init(workDir)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	cfg.Teams = map[string]config.TeamState{
		"perf": {
			ID:         "perf",
			Type:       "perf",
			AgentCount: 2,
			Intensity:  3,
			Status:     "running",
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	teamDir := filepath.Join(workDir, config.DirName, "teams", "audit-perf")
	if err := os.MkdirAll(teamDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(teamDir, ".team"), []byte("team=audit-perf\nintensity=3\ncurrent_loop=2\nstatus=active\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	now := time.Date(2026, time.February, 11, 10, 10, 10, 0, time.UTC)
	snapshot, err := loadDashboardSnapshot(workDir, now)
	if err != nil {
		t.Fatalf("loadDashboardSnapshot() returned error: %v", err)
	}

	if len(snapshot.Epics) != 0 {
		t.Fatalf("expected 0 epics for teams-only config, got %d", len(snapshot.Epics))
	}
	if len(snapshot.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(snapshot.Teams))
	}

	model := NewDashboardModel(workDir, DefaultStyles(), DefaultKeyMap())
	model.epics = snapshot.Epics
	model.teams = snapshot.Teams
	view := model.renderEpicTable()
	if !strings.Contains(view, "TEAM") || !strings.Contains(view, "audit-perf") {
		t.Fatalf("expected fallback team table in view, got: %q", view)
	}
}

func TestDashboardRefreshMessageUpdatesState(t *testing.T) {
	t.Parallel()

	model := NewDashboardModel("/tmp/work", DefaultStyles(), DefaultKeyMap())
	msg := dashboardRefreshMsg{Snapshot: dashboardSnapshot{
		SessionName: "lattice-20260211-101010",
		RefreshedAt: time.Date(2026, time.February, 11, 10, 10, 10, 0, time.UTC),
		Epics: []dashboardEpicStatus{{
			EpicName:      "Performance Audit",
			Status:        "running",
			RolesTotal:    1,
			RolesComplete: 0,
		}},
		Teams: []dashboardTeamStatus{{
			TeamName:    "audit-perf",
			Status:      "active",
			CurrentLoop: 2,
			Intensity:   5,
			AgentCount:  2,
		}},
	}}

	updated, cmd := model.Update(msg)
	if cmd != nil {
		t.Fatal("expected no command from refresh message")
	}

	if updated.sessionName != "lattice-20260211-101010" {
		t.Fatalf("unexpected session name: %q", updated.sessionName)
	}
	if len(updated.epics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(updated.epics))
	}
	if len(updated.teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(updated.teams))
	}
}

func TestDashboardModelBackNavigatesToMenu(t *testing.T) {
	t.Parallel()

	model := NewDashboardModel("/tmp/work", DefaultStyles(), DefaultKeyMap())
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected navigation command")
	}

	navMsg, ok := cmd().(AppNavigateMsg)
	if !ok {
		t.Fatalf("expected AppNavigateMsg, got %T", cmd())
	}
	if navMsg.Screen != MenuScreen {
		t.Fatalf("expected MenuScreen, got %v", navMsg.Screen)
	}

	if updated.sessionName != "" {
		t.Fatalf("expected unchanged model state, got session name %q", updated.sessionName)
	}
}

func TestDashboardTickTriggersSchedulerCheck(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	cfg, err := config.Init(workDir)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}
	cfg.Session.Name = "lattice-20260213-010203"
	cfg.Epics = map[string]config.EpicState{
		"perf": {BeadID: "e1", AuditType: "perf", AuditName: "Performance", Status: "running"},
	}
	cfg.Roles = map[string]config.RoleState{
		"r1": {BeadID: "r1", EpicBeadID: "e1", CodeName: "alpha", Title: "Alpha", Guidance: "A", BeadPrefix: "perf-alpha", Order: 1, Status: "running", Intensity: 2},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	model := NewDashboardModel(workDir, DefaultStyles(), DefaultKeyMap())
	var schedulerChecks int
	model.advanceRoles = func(cwd string, cfg *config.Config, sessionName string, plan *teams.AuditPlan, deps SchedulerDeps) (SchedulerResult, error) {
		schedulerChecks++
		return SchedulerResult{}, nil
	}

	_, cmd := model.Update(dashboardTickMsg{})
	if cmd == nil {
		t.Fatal("expected tick command")
	}

	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", cmd())
	}
	if len(batch) < 1 {
		t.Fatalf("expected scheduler command in batch, got %d commands", len(batch))
	}

	if msg := batch[0](); msg != nil {
		t.Fatalf("expected no scheduler message when unchanged, got %T", msg)
	}
	if schedulerChecks != 1 {
		t.Fatalf("expected scheduler check once, got %d", schedulerChecks)
	}
}

func TestSchedulerAdvancedMessageRefreshesDashboardState(t *testing.T) {
	t.Parallel()

	model := NewDashboardModel("/tmp/work", DefaultStyles(), DefaultKeyMap())
	refreshCalled := 0
	model.loadSnapshot = func(cwd string, now time.Time) (dashboardSnapshot, error) {
		refreshCalled++
		return dashboardSnapshot{SessionName: "lattice-20260213-010203", RefreshedAt: now}, nil
	}

	updated, cmd := model.Update(schedulerAdvancedMsg{Result: SchedulerResult{Completed: []string{"r1"}}})
	if cmd == nil {
		t.Fatal("expected refresh command")
	}

	msg := cmd()
	if _, ok := msg.(dashboardRefreshMsg); !ok {
		t.Fatalf("expected dashboardRefreshMsg, got %T", msg)
	}
	if refreshCalled != 1 {
		t.Fatalf("expected refresh once, got %d", refreshCalled)
	}
	if updated.allDone {
		t.Fatal("expected allDone to remain false")
	}
}

func TestDashboardViewShowsCompletionBannerWhenAllDone(t *testing.T) {
	t.Parallel()

	model := NewDashboardModel("/tmp/work", DefaultStyles(), DefaultKeyMap())
	updated, _ := model.Update(schedulerAdvancedMsg{Result: SchedulerResult{AllDone: true}})

	view := updated.View()
	if !strings.Contains(view, "All roles reached a terminal state") {
		t.Fatalf("expected completion banner in view, got: %q", view)
	}
}

func TestRenderEpicTableHighlightsFailedRolesClearly(t *testing.T) {
	t.Parallel()

	model := NewDashboardModel("/tmp/work", DefaultStyles(), DefaultKeyMap())
	model.epics = []dashboardEpicStatus{{
		EpicName:      "Performance Audit",
		Status:        "blocked",
		RolesTotal:    2,
		RolesComplete: 0,
		RolesFailed:   1,
		Roles: []dashboardRoleStatus{
			{CodeName: "alpha", Title: "Lead", Status: "failed", CurrentLoop: 1, Intensity: 3},
			{CodeName: "bravo", Title: "Support", Status: "pending", CurrentLoop: 0, Intensity: 3},
		},
	}}

	view := model.renderEpicTable()
	if !strings.Contains(view, "BLOCKED") {
		t.Fatalf("expected blocked epic status, got: %q", view)
	}
	if !strings.Contains(view, "FAILED") {
		t.Fatalf("expected failed role status to be prominent, got: %q", view)
	}
}
