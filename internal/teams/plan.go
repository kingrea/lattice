package teams

import (
	"fmt"
	"strings"
	"unicode"
)

// RoleBead describes one role bead generated for an epic.
type RoleBead struct {
	BeadID     string
	CodeName   string
	Title      string
	Guidance   string
	BeadPrefix string
	Order      int
}

// EpicBead describes one audit epic and its role beads.
type EpicBead struct {
	BeadID    string
	AuditType AuditType
	RoleBeads []RoleBead
}

// AuditPlan contains all generated epics and final bead counter.
type AuditPlan struct {
	Epics        []EpicBead
	FinalCounter int
}

// BuildAuditPlan builds an ordered audit plan for the selected audit types.
func BuildAuditPlan(auditTypes []AuditType, agentCount int, intensity int, startCounter int) (*AuditPlan, error) {
	if len(auditTypes) == 0 {
		return nil, fmt.Errorf("at least one audit type is required")
	}
	if agentCount < 1 || agentCount > 3 {
		return nil, fmt.Errorf("agent count must be between 1 and 3")
	}
	if intensity < 1 {
		return nil, fmt.Errorf("intensity must be at least 1")
	}

	counter := startCounter
	plan := &AuditPlan{Epics: make([]EpicBead, 0, len(auditTypes))}

	for _, auditType := range auditTypes {
		roleConfig, ok := findRoleConfig(auditType, agentCount)
		if !ok {
			return nil, fmt.Errorf("audit type %q has no role config for %d agents", auditType.ID, agentCount)
		}

		counter++
		epic := EpicBead{
			BeadID:    fmt.Sprintf("audit-plan-%03d", counter),
			AuditType: auditType,
			RoleBeads: make([]RoleBead, 0, len(roleConfig.Roles)),
		}

		for idx, role := range roleConfig.Roles {
			counter++
			epic.RoleBeads = append(epic.RoleBeads, RoleBead{
				BeadID:     fmt.Sprintf("audit-plan-%03d", counter),
				CodeName:   role.CodeName,
				Title:      role.Title,
				Guidance:   role.Guidance,
				BeadPrefix: auditType.BeadPrefix + "-" + slugify(role.Title),
				Order:      idx + 1,
			})
		}

		plan.Epics = append(plan.Epics, epic)
	}

	plan.FinalCounter = counter
	return plan, nil
}

func findRoleConfig(auditType AuditType, agentCount int) (AgentConfigRoles, bool) {
	for _, roleConfig := range auditType.RoleConfigs {
		if roleConfig.AgentCount == agentCount {
			return roleConfig, true
		}
	}

	return AgentConfigRoles{}, false
}

func slugify(input string) string {
	normalized := strings.ToLower(strings.TrimSpace(input))
	if normalized == "" {
		return ""
	}

	normalized = strings.Join(strings.Fields(normalized), "-")

	var out strings.Builder
	lastHyphen := false
	for _, r := range normalized {
		switch {
		case r == '-':
			if !lastHyphen {
				out.WriteRune('-')
				lastHyphen = true
			}
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			out.WriteRune(r)
			lastHyphen = false
		}
	}

	slug := strings.Trim(out.String(), "-")
	if len(slug) <= 30 {
		return slug
	}

	truncated := slug[:30]
	if cut := strings.LastIndex(truncated, "-"); cut > 0 {
		truncated = truncated[:cut]
	}

	truncated = strings.Trim(truncated, "-")
	if truncated == "" {
		return strings.Trim(slug[:30], "-")
	}

	return truncated
}
