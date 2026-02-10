package teams

import "testing"

func TestAuditTypesRegistryShape(t *testing.T) {
	t.Parallel()

	if len(AuditTypes) != 8 {
		t.Fatalf("expected 8 audit types, got %d", len(AuditTypes))
	}

	seen := map[string]struct{}{}
	for _, auditType := range AuditTypes {
		if auditType.ID == "" {
			t.Fatal("audit type id must not be empty")
		}
		if _, ok := seen[auditType.ID]; ok {
			t.Fatalf("duplicate audit type id: %s", auditType.ID)
		}
		seen[auditType.ID] = struct{}{}

		if auditType.Name == "" || auditType.BeadPrefix == "" || auditType.Description == "" {
			t.Fatalf("audit type %s must define name, bead prefix, and description", auditType.ID)
		}

		if len(auditType.FocusAreas) != 6 {
			t.Fatalf("audit type %s must define 6 focus areas, got %d", auditType.ID, len(auditType.FocusAreas))
		}

		if len(auditType.RoleConfigs) != 3 {
			t.Fatalf("audit type %s must define 3 role configs, got %d", auditType.ID, len(auditType.RoleConfigs))
		}

		for idx, roleConfig := range auditType.RoleConfigs {
			expectedAgents := idx + 1
			if roleConfig.AgentCount != expectedAgents {
				t.Fatalf("audit type %s role config %d expects AgentCount=%d, got %d", auditType.ID, idx, expectedAgents, roleConfig.AgentCount)
			}

			if len(roleConfig.Roles) != expectedAgents {
				t.Fatalf("audit type %s role config %d must define %d roles, got %d", auditType.ID, idx, expectedAgents, len(roleConfig.Roles))
			}

			for _, role := range roleConfig.Roles {
				if role.CodeName == "" || role.Title == "" || role.Guidance == "" {
					t.Fatalf("audit type %s role config %d has incomplete role definition", auditType.ID, idx)
				}
			}
		}
	}
}
