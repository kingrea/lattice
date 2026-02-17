Close an audit run using `@audit-orchestrator`.

Input:

- `run_id` (required)

Actions:

1. Validate state files and mailbox index.
2. Spawn `@scribe` and generate `context/REPORT.md` from bead evidence only.
3. Run `closeout-checklist` and emit pass/fail items.
4. If all checks pass, set `.team status=complete` as the final state write.
5. Refresh `STATUS.md` with final summary.

Block completion when checklist contains failures.
