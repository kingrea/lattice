package teams

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lattice/internal/config"
)

func TestGenerateCreatesRenderedAuditTeamFolder(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	teamDir, err := Generate(GenerateParams{
		WorkingDir: workDir,
		AuditType:  AuditTypes[0],
		AgentCount: 2,
		Intensity:  4,
		Target:     "Checkout flow",
		BeadPrefix: "perf-12",
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	wantTeamDir := filepath.Join(workDir, config.DirName, "teams", "audit-perf")
	if teamDir != wantTeamDir {
		t.Fatalf("unexpected team directory: got %q want %q", teamDir, wantTeamDir)
	}

	assertFileExists(t, filepath.Join(teamDir, ".team"))
	assertFileExists(t, filepath.Join(teamDir, "INSTRUCTIONS.md"))
	assertFileExists(t, filepath.Join(teamDir, "context", "TASK.md"))
	assertFileExists(t, filepath.Join(teamDir, "DESCRIPTION.md"))
	assertFileExists(t, filepath.Join(teamDir, "opencode.jsonc"))
	assertFileExists(t, filepath.Join(teamDir, ".opencode", "agents", "commissar.md"))
	assertFileExists(t, filepath.Join(teamDir, ".opencode", "skills", "compile-report", "SKILL.md"))

	assertFileExists(t, filepath.Join(teamDir, ".opencode", "agents", "investigator-alpha.md"))
	assertFileExists(t, filepath.Join(teamDir, ".opencode", "agents", "investigator-bravo.md"))
	assertFileNotExists(t, filepath.Join(teamDir, ".opencode", "agents", "investigator-charlie.md"))

	teamFile, err := os.ReadFile(filepath.Join(teamDir, ".team"))
	if err != nil {
		t.Fatalf("ReadFile(.team) returned error: %v", err)
	}
	teamText := string(teamFile)
	if !strings.Contains(teamText, "team=audit-perf") {
		t.Fatalf("expected .team to include team name, got %q", teamText)
	}
	if !strings.Contains(teamText, "intensity=4") {
		t.Fatalf("expected .team to include intensity, got %q", teamText)
	}

	instructions, err := os.ReadFile(filepath.Join(teamDir, "INSTRUCTIONS.md"))
	if err != nil {
		t.Fatalf("ReadFile(INSTRUCTIONS.md) returned error: %v", err)
	}
	if !strings.Contains(string(instructions), "Use the team bead prefix `perf-12`") {
		t.Fatalf("expected instructions to include rendered bead prefix")
	}

	task, err := os.ReadFile(filepath.Join(teamDir, "context", "TASK.md"))
	if err != nil {
		t.Fatalf("ReadFile(context/TASK.md) returned error: %v", err)
	}
	taskText := string(task)
	if !strings.Contains(taskText, "Checkout flow") {
		t.Fatalf("expected task to include target text")
	}
	if !strings.Contains(taskText, "- Senior performance specialist") {
		t.Fatalf("expected task to include alpha role")
	}
	if !strings.Contains(taskText, "- Staff performance engineer") {
		t.Fatalf("expected task to include bravo role")
	}
	if !strings.Contains(taskText, "- render throughput and frame stability") {
		t.Fatalf("expected task to include focus areas")
	}
}

func TestGenerateSkipsUnusedInvestigatorsByAgentCount(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	teamDir, err := Generate(GenerateParams{
		WorkingDir: workDir,
		AuditType:  AuditTypes[3],
		AgentCount: 1,
		Intensity:  2,
		Target:     "Auth flows",
		BeadPrefix: "sec-7",
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	assertFileExists(t, filepath.Join(teamDir, ".opencode", "agents", "investigator-alpha.md"))
	assertFileNotExists(t, filepath.Join(teamDir, ".opencode", "agents", "investigator-bravo.md"))
	assertFileNotExists(t, filepath.Join(teamDir, ".opencode", "agents", "investigator-charlie.md"))
}

func TestGenerateUsesCustomFocusAreasWhenProvided(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	teamDir, err := Generate(GenerateParams{
		WorkingDir: workDir,
		AuditType:  AuditTypes[0],
		AgentCount: 1,
		Intensity:  1,
		Target:     "Checkout flow",
		BeadPrefix: "perf-3",
		FocusAreas: []string{
			"Routing (internal/tui): Review key navigation transitions.",
			"Templates (templates/audit): Validate prompt boundaries and output consistency.",
			"Team Generation (internal/teams): Check rendered role/focus wiring.",
		},
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	task, err := os.ReadFile(filepath.Join(teamDir, "context", "TASK.md"))
	if err != nil {
		t.Fatalf("ReadFile(context/TASK.md) returned error: %v", err)
	}

	taskText := string(task)
	if !strings.Contains(taskText, "Routing (internal/tui)") {
		t.Fatalf("expected custom focus area in task file, got %q", taskText)
	}
	if strings.Contains(taskText, "render throughput and frame stability") {
		t.Fatalf("expected default focus areas to be replaced, got %q", taskText)
	}
}

func TestGenerateReturnsErrorWhenRoleConfigMissing(t *testing.T) {
	t.Parallel()

	_, err := Generate(GenerateParams{
		WorkingDir: t.TempDir(),
		AuditType: AuditType{
			ID:         "custom",
			FocusAreas: []string{"one"},
			RoleConfigs: []AgentConfigRoles{
				{AgentCount: 1, Roles: []RoleDefinition{{CodeName: "alpha", Title: "One", Guidance: "One"}}},
			},
		},
		AgentCount: 2,
		Intensity:  1,
		Target:     "target",
		BeadPrefix: "c-1",
	})

	if err == nil || !strings.Contains(err.Error(), "has no role config") {
		t.Fatalf("expected missing role config error, got: %v", err)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %q to exist: %v", path, err)
	}
}

func assertFileNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file %q to not exist", path)
	}
}
