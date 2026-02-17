---
name: status-board
description: Regenerate STATUS.md from run state and mailbox evidence.
---

# Status Board

Generate `.opencode/audits/<run-id>/STATUS.md` with:

- run metadata (target, type, intensity)
- current loop and early-exit state
- investigator table (role, status, last update)
- bead counts (new, updated, deduped, closed)
- report/checklist completion state

Always source values from persisted files, not memory-only conversation state.
