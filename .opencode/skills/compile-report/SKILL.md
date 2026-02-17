---
name: compile-report
description: Compile final run report from bead and run-state evidence.
---

# Compile Report

Write `context/REPORT.md` from:

- `context/TASK.md`
- `.team`
- `state/plan.json`
- `state/loop-status.json`
- `state/findings-map.json`
- bead data from `bd list` and `bd show`

Report requirements:

- include scope, loops completed, and investigator assignments
- group findings by severity and area
- include bead ids for every finding line
- note duplicate/out-of-scope triage actions
- if no findings, state that clearly

Do not introduce findings that are not bead-backed.
