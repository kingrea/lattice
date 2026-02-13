---
name: create-workitems
description: Breaks down a task into beads (work items) with dependencies and acceptance criteria, assigning them to team grunts for execution.
---

# Create Work Items

You are breaking down the current task into discrete work items (beads) for your grunts to execute.

## Inputs

Before using this skill you MUST have already read and understood:

- `context/TASK.md` — the task description, design notes, and acceptance criteria
- Any other files in the `context` directory that provide relevant information for understanding the task and how to accomplish it
- `CRITERIA.md` — the ONLY permitted acceptance criteria options
- `DESCRIPTION.md` — team structure and roles

## Decomposition Rules

1. Each bead must be small enough for a single grunt to complete in one pass.
2. Each bead must be independently testable against its acceptance criteria.
3. Each bead must be unambiguous — grunts are unimaginative, they need explicit instructions.
4. Beads should follow a logical implementation order (e.g., data layer before UI).

## Creating Beads

For each work item:

```bash
bd create "<clear, specific title>" -p <priority>
```

Priority levels: 0 (critical) through 3 (low). Use 0–1 for blocking work, 2 for standard, 3 for polish.

## Setting Dependencies

If bead B requires bead A to be completed first:

```bash
bd dep add <bead-B-id> <bead-A-id>
```

Verify no circular dependencies exist after setting them all.

## Bead Content

After creating each bead, add a structured comment with the full work specification:

```bash
bd comment <bead-id> "
## Description
<What needs to be done. Be extremely specific: file paths, function names, expected inputs/outputs, behavior.>

## Files
<Exact file paths to create or modify.>

## Acceptance Criteria
<Select ONLY from CRITERIA.md options. Only include criteria relevant to this specific bead.>

## Constraints
<What the grunt must NOT do. Boundaries of this work item. Patterns or conventions to follow.>
"
```

## Assignment

- Balance work across your two grunts (any agent prefixed with grunt).
- Assign parallelizable beads to different grunts.
- Assign sequential beads to the same grunt where possible to maintain context.

## Verification

After creating all beads, verify the work queue:

```bash
bd ready
```

Confirm:
- All beads have clear titles and descriptions
- Dependencies form a valid DAG (no cycles)
- Acceptance criteria are drawn only from `CRITERIA.md`
- Work is distributed across all grunts
