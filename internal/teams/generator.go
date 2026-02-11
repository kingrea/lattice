package teams

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"lattice/internal/config"
	"lattice/templates"
)

const (
	auditTemplateRoot = "audit"
	templateExt       = ".tmpl"
)

// Role is template-friendly role data for one active investigator.
type Role struct {
	CodeName string
	Title    string
	Guidance string
}

// String renders the role in list contexts.
func (r Role) String() string {
	return r.Title
}

// TemplateData contains values rendered into audit team templates.
type TemplateData struct {
	TeamName   string
	Intensity  int
	BeadPrefix string
	Target     string
	Roles      []Role
	FocusAreas []string
}

// GenerateParams defines all required inputs to generate an audit team folder.
type GenerateParams struct {
	WorkingDir string
	AuditType  AuditType
	AgentCount int
	Intensity  int
	Target     string
	BeadPrefix string
	FocusAreas []string
}

// Generate creates .lattice/teams/audit-{type}/ from embedded templates.
func Generate(params GenerateParams) (string, error) {
	if strings.TrimSpace(params.WorkingDir) == "" {
		return "", fmt.Errorf("working dir must not be empty")
	}
	if strings.TrimSpace(params.AuditType.ID) == "" {
		return "", fmt.Errorf("audit type id must not be empty")
	}
	if params.AgentCount < 1 || params.AgentCount > 3 {
		return "", fmt.Errorf("agent count must be between 1 and 3")
	}
	if params.Intensity < 1 {
		return "", fmt.Errorf("intensity must be at least 1")
	}
	if strings.TrimSpace(params.BeadPrefix) == "" {
		return "", fmt.Errorf("bead prefix must not be empty")
	}

	roles, err := activeRoles(params.AuditType, params.AgentCount)
	if err != nil {
		return "", err
	}

	teamName := "audit-" + params.AuditType.ID
	teamDir := filepath.Join(params.WorkingDir, config.DirName, "teams", teamName)

	if err := os.RemoveAll(teamDir); err != nil {
		return "", fmt.Errorf("reset team directory: %w", err)
	}
	if err := os.MkdirAll(teamDir, 0o755); err != nil {
		return "", fmt.Errorf("create team directory: %w", err)
	}

	focusAreas := params.AuditType.FocusAreas
	if len(params.FocusAreas) > 0 {
		focusAreas = append([]string(nil), params.FocusAreas...)
	}

	data := TemplateData{
		TeamName:   teamName,
		Intensity:  params.Intensity,
		BeadPrefix: strings.TrimSpace(params.BeadPrefix),
		Target:     params.Target,
		Roles:      roles,
		FocusAreas: focusAreas,
	}

	if err := fs.WalkDir(templates.AuditTemplate, auditTemplateRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == auditTemplateRoot {
			return nil
		}

		relPath, err := filepath.Rel(auditTemplateRoot, path)
		if err != nil {
			return fmt.Errorf("compute relative path for %q: %w", path, err)
		}
		relPath = filepath.ToSlash(relPath)

		if shouldSkipInvestigator(relPath, params.AgentCount) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		outputRel := strings.TrimSuffix(relPath, templateExt)
		outputPath := filepath.Join(teamDir, filepath.FromSlash(outputRel))

		if entry.IsDir() {
			return os.MkdirAll(outputPath, 0o755)
		}

		if strings.HasSuffix(relPath, templateExt) {
			return renderTemplateFile(path, outputPath, data)
		}

		return copyStaticFile(path, outputPath)
	}); err != nil {
		return "", fmt.Errorf("generate audit team files: %w", err)
	}

	return teamDir, nil
}

func activeRoles(auditType AuditType, agentCount int) ([]Role, error) {
	for _, roleConfig := range auditType.RoleConfigs {
		if roleConfig.AgentCount != agentCount {
			continue
		}

		roles := make([]Role, 0, len(roleConfig.Roles))
		for _, role := range roleConfig.Roles {
			roles = append(roles, Role{
				CodeName: role.CodeName,
				Title:    role.Title,
				Guidance: role.Guidance,
			})
		}
		return roles, nil
	}

	return nil, fmt.Errorf("audit type %q has no role config for %d agents", auditType.ID, agentCount)
}

func shouldSkipInvestigator(relPath string, agentCount int) bool {
	if relPath == ".opencode/agents/investigator-bravo.md" && agentCount < 2 {
		return true
	}

	if relPath == ".opencode/agents/investigator-charlie.md" && agentCount < 3 {
		return true
	}

	return false
}

func renderTemplateFile(srcPath, dstPath string, data TemplateData) error {
	src, err := fs.ReadFile(templates.AuditTemplate, srcPath)
	if err != nil {
		return fmt.Errorf("read template %q: %w", srcPath, err)
	}

	tmpl, err := template.New(srcPath).Parse(string(src))
	if err != nil {
		return fmt.Errorf("parse template %q: %w", srcPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("create directory for %q: %w", dstPath, err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return fmt.Errorf("execute template %q: %w", srcPath, err)
	}

	if err := os.WriteFile(dstPath, out.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write rendered file %q: %w", dstPath, err)
	}

	return nil
}

func copyStaticFile(srcPath, dstPath string) error {
	src, err := templates.AuditTemplate.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open embedded file %q: %w", srcPath, err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("create directory for %q: %w", dstPath, err)
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create file %q: %w", dstPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy static file %q: %w", srcPath, err)
	}

	return nil
}
