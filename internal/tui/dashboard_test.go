package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lattice/internal/config"
)

func TestLoadDashboardSnapshotReadsTeamFiles(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	cfg, err := config.Init(workDir)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	cfg.Session.Name = "lattice-20260211-101010"
	cfg.Teams = map[string]config.TeamState{
		"perf": {
			ID:         "perf",
			Type:       "perf",
			AgentCount: 3,
			Intensity:  2,
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
	if err := os.WriteFile(filepath.Join(teamDir, ".team"), []byte("team=audit-perf\nintensity=7\ncurrent_loop=4\nstatus=active\n"), 0o644); err != nil {
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
	if len(snapshot.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(snapshot.Teams))
	}

	team := snapshot.Teams[0]
	if team.TeamName != "audit-perf" {
		t.Fatalf("unexpected team name: %q", team.TeamName)
	}
	if team.Status != "active" {
		t.Fatalf("unexpected team status: %q", team.Status)
	}
	if team.CurrentLoop != 4 {
		t.Fatalf("unexpected current loop: %d", team.CurrentLoop)
	}
	if team.Intensity != 7 {
		t.Fatalf("unexpected intensity: %d", team.Intensity)
	}
	if team.AgentCount != 3 {
		t.Fatalf("unexpected agent count: %d", team.AgentCount)
	}
}

func TestDashboardRefreshMessageUpdatesState(t *testing.T) {
	t.Parallel()

	model := NewDashboardModel("/tmp/work", DefaultStyles(), DefaultKeyMap())
	msg := dashboardRefreshMsg{Snapshot: dashboardSnapshot{
		SessionName: "lattice-20260211-101010",
		RefreshedAt: time.Date(2026, time.February, 11, 10, 10, 10, 0, time.UTC),
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
