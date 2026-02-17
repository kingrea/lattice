---
name: create-audit-plan
description: Build assignment and loop plan for a run and persist plan state.
---

# Create Audit Plan

Process:

1. Read `context/TASK.md` and `.team`.
2. Parse `target`, `audit_type`, `roles`, `focus_areas`, and `intensity`.
3. Run `bd list` to capture existing beads.
4. Assign roles in fixed order:
   - 1 role: alpha active, others idle
   - 2 roles: alpha/bravo active, charlie idle
   - 3 roles: all active
5. Write assignment and metadata to `state/plan.json`.

Output fields in `state/plan.json` must include:

- `run_id`
- `target`
- `audit_type`
- `intensity`
- `assignments` (array of investigator objects)
- `existing_beads` (array of bead ids known at run start)
