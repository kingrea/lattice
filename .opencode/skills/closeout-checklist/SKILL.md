---
name: closeout-checklist
description: Enforce closeout gates before final completion state is written.
---

# Closeout Checklist

Run and report pass/fail for each item:

1. All actionable findings are tracked in beads.
2. Duplicate review completed.
3. Out-of-scope findings triaged with rationale.
4. `context/REPORT.md` exists and is bead-backed.
5. `bd sync` completed.
6. `git push` succeeded.
7. `.team status=complete` not yet set.

If any item fails, return `CHECKLIST_FAILED` and list required remediation.
Only when all items pass may orchestrator set `.team status=complete`.
