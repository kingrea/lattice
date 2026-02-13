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
	if cfg.Epics == nil {
		t.Fatal("expected Epics map to be initialized")
	}
	if cfg.Roles == nil {
		t.Fatal("expected Roles map to be initialized")
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

func TestLoadSaveRoundTripWithEpicsAndRoles(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg, err := Init(tmp)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	cfg.Epics["ai-nl5"] = EpicState{
		BeadID:     "ai-nl5",
		AuditType:  "nlp",
		AuditName:  "narrative-audit",
		AgentCount: 3,
		Intensity:  2,
		Status:     "in_progress",
	}
	cfg.Roles["scribe"] = RoleState{
		BeadID:     "ai-nl6",
		EpicBeadID: "ai-nl5",
		CodeName:   "scribe",
		Title:      "Session Scribe",
		Guidance:   "Capture findings and decisions.",
		BeadPrefix: "ai-nl",
		Order:      1,
		Status:     "ready",
		TmuxWindow: "nl5-scribe",
		Intensity:  1,
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if !reflect.DeepEqual(loaded.Epics, cfg.Epics) {
		t.Fatalf("epic state mismatch after round trip: got %+v want %+v", loaded.Epics, cfg.Epics)
	}
	if !reflect.DeepEqual(loaded.Roles, cfg.Roles) {
		t.Fatalf("role state mismatch after round trip: got %+v want %+v", loaded.Roles, cfg.Roles)
	}
}

func TestLoadBackwardCompatibilityWithoutEpicsAndRoles(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	configDir := filepath.Join(tmp, DirName)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}

	legacyConfig := `bead_counter = 2

[session]
name = "legacy-session"
created_at = "2026-02-12T00:00:00Z"
working_dir = "/tmp/legacy"

[teams.alpha]
id = "alpha"
type = "legacy"
prefix = "legacy-2"
agent_count = 1
intensity = 1
status = "idle"
tmux_window = "alpha"
`

	configPath := filepath.Join(configDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(legacyConfig), 0o644); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if loaded.Epics == nil {
		t.Fatal("expected Epics to be initialized")
	}
	if loaded.Roles == nil {
		t.Fatal("expected Roles to be initialized")
	}
	if len(loaded.Epics) != 0 {
		t.Fatalf("expected Epics to be empty, got %d entries", len(loaded.Epics))
	}
	if len(loaded.Roles) != 0 {
		t.Fatalf("expected Roles to be empty, got %d entries", len(loaded.Roles))
	}
}

func TestSaveAtomicWriteCleansTempFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg, err := Init(tmp)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	cfg.BeadCounter = 9
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	tmpPath := filepath.Join(tmp, DirName, ConfigFileName+".tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatalf("expected temp file to be removed, stat err: %v", err)
	}
}

func TestNilMapGuardsOnSaveAndLoad(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	configDir := filepath.Join(tmp, DirName)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}

	cfg := &Config{
		Session: SessionMetadata{
			Name:       "nil-map-guard",
			CreatedAt:  "2026-02-13T00:00:00Z",
			WorkingDir: tmp,
		},
		filePath: filepath.Join(configDir, ConfigFileName),
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}
	if cfg.Teams == nil {
		t.Fatal("expected Teams map to be initialized after Save")
	}
	if cfg.Epics == nil {
		t.Fatal("expected Epics map to be initialized after Save")
	}
	if cfg.Roles == nil {
		t.Fatal("expected Roles map to be initialized after Save")
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if loaded.Teams == nil {
		t.Fatal("expected Teams map to be initialized after Load")
	}
	if loaded.Epics == nil {
		t.Fatal("expected Epics map to be initialized after Load")
	}
	if loaded.Roles == nil {
		t.Fatal("expected Roles map to be initialized after Load")
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
