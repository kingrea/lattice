Start a new audit run using `@audit-orchestrator`.

Inputs:

- `run_id` (required)
- `target` (required)
- `audit_type` (required)
- `intensity` (required, integer >= 1)
- `roles` (required, 1-3 role labels)
- `focus_areas` (required list)

Actions:

1. Create `.opencode/audits/<run-id>/` with:
   - `.team`
   - `context/TASK.md`
   - `mailbox/index.json`
   - `mailbox/orchestrator.ndjson`
   - `mailbox/investigator-alpha.ndjson`
   - `mailbox/investigator-bravo.ndjson`
   - `mailbox/investigator-charlie.ndjson`
   - `mailbox/scribe.ndjson`
   - `mailbox/chatroom.ndjson`
   - `state/plan.json`
   - `state/loop-status.json`
   - `state/findings-map.json`
   - `STATUS.md`
2. Initialize state using schemas in `.opencode/audits/SCHEMA.md`.
3. Validate required keys before finishing.
4. Report created paths and readiness status.
