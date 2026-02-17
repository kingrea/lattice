---
name: mailbox-read
description: Read and summarize mailbox streams for orchestrator decisions.
---

# Mailbox Read

Inputs:

- `run_id`
- `agent` (optional)
- `since_loop` (optional)

Steps:

1. Read `mailbox/index.json`.
2. Read requested mailbox streams (`orchestrator`, investigators, `scribe`, optional `chatroom`).
3. Parse NDJSON and ignore malformed lines with explicit warnings.
4. Summarize latest status per investigator and list bead references.
5. Return deterministic output suitable for status and loop progression.
