package teams

import (
	"strings"
	"testing"
)

func TestBuildAuditPlanOneAuditTypeOneAgent(t *testing.T) {
	t.Parallel()

	plan, err := BuildAuditPlan([]AuditType{AuditTypes[0]}, 1, 1, 0)
	if err != nil {
		t.Fatalf("BuildAuditPlan() returned error: %v", err)
	}

	if len(plan.Epics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(plan.Epics))
	}

	epic := plan.Epics[0]
	if len(epic.RoleBeads) != 1 {
		t.Fatalf("expected 1 role bead, got %d", len(epic.RoleBeads))
	}

	if plan.FinalCounter != 2 {
		t.Fatalf("expected FinalCounter=2, got %d", plan.FinalCounter)
	}
}

func TestBuildAuditPlanTwoAuditTypesThreeAgents(t *testing.T) {
	t.Parallel()

	plan, err := BuildAuditPlan([]AuditType{AuditTypes[0], AuditTypes[1]}, 3, 2, 0)
	if err != nil {
		t.Fatalf("BuildAuditPlan() returned error: %v", err)
	}

	if len(plan.Epics) != 2 {
		t.Fatalf("expected 2 epics, got %d", len(plan.Epics))
	}

	roles := 0
	for _, epic := range plan.Epics {
		roles += len(epic.RoleBeads)
	}
	if roles != 6 {
		t.Fatalf("expected 6 role beads, got %d", roles)
	}

	if plan.FinalCounter != 8 {
		t.Fatalf("expected FinalCounter=8, got %d", plan.FinalCounter)
	}
}

func TestBuildAuditPlanAssignsSequentialIDs(t *testing.T) {
	t.Parallel()

	plan, err := BuildAuditPlan([]AuditType{AuditTypes[0], AuditTypes[1]}, 1, 1, 0)
	if err != nil {
		t.Fatalf("BuildAuditPlan() returned error: %v", err)
	}

	ids := []string{
		plan.Epics[0].BeadID,
		plan.Epics[0].RoleBeads[0].BeadID,
		plan.Epics[1].BeadID,
		plan.Epics[1].RoleBeads[0].BeadID,
	}

	want := []string{"audit-plan-001", "audit-plan-002", "audit-plan-003", "audit-plan-004"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("unexpected bead id at position %d: got %q want %q", i, ids[i], want[i])
		}
	}
}

func TestBuildAuditPlanDerivesRoleBeadPrefix(t *testing.T) {
	t.Parallel()

	plan, err := BuildAuditPlan([]AuditType{AuditTypes[0]}, 1, 1, 0)
	if err != nil {
		t.Fatalf("BuildAuditPlan() returned error: %v", err)
	}

	got := plan.Epics[0].RoleBeads[0].BeadPrefix
	if got != "perf-senior-performance-specialist" {
		t.Fatalf("unexpected bead prefix: got %q", got)
	}
}

func TestBuildAuditPlanReturnsValidationErrors(t *testing.T) {
	t.Parallel()

	if _, err := BuildAuditPlan(nil, 1, 1, 0); err == nil || !strings.Contains(err.Error(), "at least one audit type") {
		t.Fatalf("expected empty audit type error, got: %v", err)
	}

	if _, err := BuildAuditPlan([]AuditType{AuditTypes[0]}, 0, 1, 0); err == nil || !strings.Contains(err.Error(), "agent count must be between 1 and 3") {
		t.Fatalf("expected invalid agent count error, got: %v", err)
	}
}

func TestBuildAuditPlanRolePrefixesUniqueAcrossAuditTypesAndAgentCounts(t *testing.T) {
	t.Parallel()

	for agentCount := 1; agentCount <= 3; agentCount++ {
		plan, err := BuildAuditPlan(AuditTypes, agentCount, 1, 0)
		if err != nil {
			t.Fatalf("BuildAuditPlan() returned error for %d agents: %v", agentCount, err)
		}

		seen := map[string]struct{}{}
		for _, epic := range plan.Epics {
			for _, roleBead := range epic.RoleBeads {
				if _, ok := seen[roleBead.BeadPrefix]; ok {
					t.Fatalf("duplicate role bead prefix for %d agents: %q", agentCount, roleBead.BeadPrefix)
				}
				seen[roleBead.BeadPrefix] = struct{}{}
			}
		}
	}
}

func TestSlugifyTruncatesAtWordBoundary(t *testing.T) {
	t.Parallel()

	got := slugify("Senior Specialist for Ultra Complex Cross Functional Investigations")
	if got != "senior-specialist-for-ultra" {
		t.Fatalf("unexpected slug: got %q", got)
	}
}
