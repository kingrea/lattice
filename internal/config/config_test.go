package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestInitCreatesConfigWithDefaults(t *testing.T) {
	tmp := t.TempDir()
	fixed := time.Date(2026, time.February, 11, 1, 2, 3, 0, time.UTC)
	originalNow := now
	now = func() time.Time { return fixed }
	t.Cleanup(func() { now = originalNow })

	cfg, err := Init(tmp)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	if cfg.BeadCounter != 0 {
		t.Fatalf("expected BeadCounter=0, got %d", cfg.BeadCounter)
	}
	if cfg.Teams == nil {
		t.Fatal("expected Teams map to be initialized")
	}
	if cfg.Session.WorkingDir != tmp {
		t.Fatalf("expected Session.WorkingDir=%q, got %q", tmp, cfg.Session.WorkingDir)
	}
	if cfg.Session.Name != "lattice-20260211-010203" {
		t.Fatalf("unexpected session name: %q", cfg.Session.Name)
	}
	if cfg.Session.CreatedAt != "2026-02-11T01:02:03Z" {
		t.Fatalf("unexpected created_at value: %q", cfg.Session.CreatedAt)
	}

	if _, err := os.Stat(filepath.Join(tmp, DirName)); err != nil {
		t.Fatalf("expected lattice directory to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, DirName, ConfigFileName)); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
}

func TestInitLoadsExistingConfig(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg, err := Init(tmp)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	cfg.BeadCounter = 7
	cfg.Teams["team-alpha"] = TeamState{ID: "team-alpha", Type: "perf", Prefix: "perf-7", AgentCount: 2, Intensity: 3, Status: "running", TmuxWindow: "perf"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	reloaded, err := Init(tmp)
	if err != nil {
		t.Fatalf("Init() for existing config returned error: %v", err)
	}

	if reloaded.BeadCounter != 7 {
		t.Fatalf("expected BeadCounter=7, got %d", reloaded.BeadCounter)
	}
	if _, ok := reloaded.Teams["team-alpha"]; !ok {
		t.Fatal("expected existing team state to be preserved")
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg, err := Init(tmp)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	cfg.BeadCounter = 12
	cfg.Teams["team-security"] = TeamState{
		ID:         "team-security",
		Type:       "security",
		Prefix:     "sec-12",
		AgentCount: 3,
		Intensity:  4,
		Status:     "ready",
		TmuxWindow: "security",
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if !reflect.DeepEqual(loaded.Session, cfg.Session) {
		t.Fatalf("session mismatch after round trip: got %+v want %+v", loaded.Session, cfg.Session)
	}
	if loaded.BeadCounter != cfg.BeadCounter {
		t.Fatalf("bead counter mismatch: got %d want %d", loaded.BeadCounter, cfg.BeadCounter)
	}
	if !reflect.DeepEqual(loaded.Teams, cfg.Teams) {
		t.Fatalf("team state mismatch after round trip: got %+v want %+v", loaded.Teams, cfg.Teams)
	}
}

func TestSaveWithoutPathFails(t *testing.T) {
	t.Parallel()

	var cfg Config
	err := cfg.Save()
	if err == nil || !strings.Contains(err.Error(), "config file path is not set") {
		t.Fatalf("expected file path error, got: %v", err)
	}
}
