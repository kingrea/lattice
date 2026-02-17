Show current status for a run using `@audit-orchestrator`.

Input:

- `run_id` (required)

Actions:

1. Read `.team`, `state/plan.json`, `state/loop-status.json`, and `state/findings-map.json`.
2. Validate required keys; if invalid, provide repair instructions.
3. Render a concise board with:
   - run id, audit type, target
   - configured intensity and current loop
   - active investigator status
   - bead totals (created, updated, deduped, closed)
   - report status

Output should match `STATUS.md` and note any drift.
