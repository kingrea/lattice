package templates

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"
	"text/template"
)

func TestAuditTemplateReadDirIncludesDotfiles(t *testing.T) {
	t.Parallel()

	entries, err := fs.ReadDir(AuditTemplate, "audit")
	if err != nil {
		t.Fatalf("ReadDir audit: %v", err)
	}

	if !hasEntry(entries, ".opencode") {
		t.Fatalf("expected .opencode directory in embedded template")
	}

	if !hasEntry(entries, ".team.tmpl") {
		t.Fatalf("expected .team.tmpl file in embedded template")
	}
}

func TestAuditTemplateIncludesStaticAgentAndSkillFiles(t *testing.T) {
	t.Parallel()

	if _, err := fs.ReadFile(AuditTemplate, "audit/.opencode/agents/commissar.md"); err != nil {
		t.Fatalf("missing static agent file: %v", err)
	}

	if _, err := fs.ReadFile(AuditTemplate, "audit/.opencode/skills/compile-report/SKILL.md"); err != nil {
		t.Fatalf("missing static skill file: %v", err)
	}
}

func TestAuditTemplateFilesRenderWithTestData(t *testing.T) {
	t.Parallel()

	type testData struct {
		TeamName   string
		Intensity  int
		BeadPrefix string
		Target     string
		Roles      []string
		FocusAreas []string
	}

	data := testData{
		TeamName:   "audit-perf",
		Intensity:  3,
		BeadPrefix: "perf-12",
		Target:     "Checkout flow",
		Roles:      []string{"Senior engineer", "Staff engineer"},
		FocusAreas: []string{"render performance", "network waterfalls"},
	}

	assertRenderedContains(t, "audit/.team.tmpl", data, "team=audit-perf")
	assertRenderedContains(t, "audit/.team.tmpl", data, "intensity=3")
	assertRenderedContains(t, "audit/INSTRUCTIONS.md.tmpl", data, "Use the team bead prefix `perf-12`")
	assertRenderedContains(t, "audit/context/TASK.md.tmpl", data, "Checkout flow")
	assertRenderedContains(t, "audit/context/TASK.md.tmpl", data, "- Senior engineer")
	assertRenderedContains(t, "audit/context/TASK.md.tmpl", data, "- network waterfalls")
}

func TestRoleSessionTemplateReadDirIncludesDotfiles(t *testing.T) {
	t.Parallel()

	entries, err := fs.ReadDir(RoleSessionTemplate, "role-session")
	if err != nil {
		t.Fatalf("ReadDir role-session: %v", err)
	}

	if !hasEntry(entries, ".opencode") {
		t.Fatalf("expected .opencode directory in embedded template")
	}

	if !hasEntry(entries, ".team.tmpl") {
		t.Fatalf("expected .team.tmpl file in embedded template")
	}
}

func TestRoleSessionTemplateIncludesStaticAgentAndSkillFiles(t *testing.T) {
	t.Parallel()

	if _, err := fs.ReadFile(RoleSessionTemplate, "role-session/.opencode/agents/auditor.md"); err != nil {
		t.Fatalf("missing static agent file: %v", err)
	}

	if _, err := fs.ReadFile(RoleSessionTemplate, "role-session/.opencode/skills/compile-report/SKILL.md"); err != nil {
		t.Fatalf("missing static skill file: %v", err)
	}
}

func TestRoleSessionTemplateFilesRenderWithTestData(t *testing.T) {
	t.Parallel()

	type testData struct {
		TeamName     string
		EpicBeadID   string
		RoleBeadID   string
		RoleTitle    string
		RoleGuidance string
		Intensity    int
		BeadPrefix   string
		Target       string
		FocusAreas   []string
	}

	data := testData{
		TeamName:     "audit-role-security",
		EpicBeadID:   "epic-101",
		RoleBeadID:   "security-202",
		RoleTitle:    "Security specialist",
		RoleGuidance: "Prioritize auth boundaries, input handling, and data exposure.",
		Intensity:    2,
		BeadPrefix:   "sec-88",
		Target:       "Authentication middleware",
		FocusAreas:   []string{"token validation", "authorization checks"},
	}

	assertRenderedContainsFromFS(t, RoleSessionTemplate, "role-session/.team.tmpl", data, "team=audit-role-security")
	assertRenderedContainsFromFS(t, RoleSessionTemplate, "role-session/.team.tmpl", data, "epic_bead_id=epic-101")
	assertRenderedContainsFromFS(t, RoleSessionTemplate, "role-session/.team.tmpl", data, "role=Security specialist")
	assertRenderedContainsFromFS(t, RoleSessionTemplate, "role-session/INSTRUCTIONS.md.tmpl", data, "Use the role bead prefix `sec-88`")
	assertRenderedContainsFromFS(t, RoleSessionTemplate, "role-session/context/TASK.md.tmpl", data, "- Epic bead: `epic-101`")
	assertRenderedContainsFromFS(t, RoleSessionTemplate, "role-session/context/TASK.md.tmpl", data, "- authorization checks")
}

func assertRenderedContains(t *testing.T, filePath string, data any, want string) {
	t.Helper()

	assertRenderedContainsFromFS(t, AuditTemplate, filePath, data, want)
}

func assertRenderedContainsFromFS(t *testing.T, templateFS fs.FS, filePath string, data any, want string) {
	t.Helper()

	src, err := fs.ReadFile(templateFS, filePath)
	if err != nil {
		t.Fatalf("read %s: %v", filePath, err)
	}

	tmpl, err := template.New(filePath).Parse(string(src))
	if err != nil {
		t.Fatalf("parse %s: %v", filePath, err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		t.Fatalf("execute %s: %v", filePath, err)
	}

	if !strings.Contains(out.String(), want) {
		t.Fatalf("rendered %s missing %q", filePath, want)
	}
}

func hasEntry(entries []fs.DirEntry, name string) bool {
	for _, entry := range entries {
		if entry.Name() == name {
			return true
		}
	}

	return false
}
