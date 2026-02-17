Advance exactly one audit loop using `@audit-orchestrator`.

Inputs:

- `run_id` (required)

Actions:

1. Validate state files against `.opencode/audits/SCHEMA.md`.
2. Execute one deterministic loop only:
   - spawn active investigators in fixed order (alpha, bravo, charlie)
   - use `investigator-prompt` on loop 1 and `loop-prompt` on loop 2+
   - capture status per investigator (`FINDINGS` or `NOTHING_MORE`)
   - append mailbox messages
   - update loop status and findings map
3. Increment `.team current_loop` once.
4. Refresh `STATUS.md` via `status-board`.
5. If all investigators are `NOTHING_MORE`, mark early-exit in status.

Output:

- loop number executed
- investigator statuses
- bead create/update counts
- early-exit flag
