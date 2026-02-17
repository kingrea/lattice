# OpenCode Audit Implementation Plan

This is an execution-ready plan to migrate from the Go/TUI audit runner to a single-session OpenCode workflow with beads, subagents, skills, and a mailbox that is ready for future multi-agent operation.

Each step includes:

- Deliverables (what to build)
- Verification (how to prove it works)
- Exit criteria (definition of done)

---

## Scope and Outcome

Target outcome:

1. Run an entire audit in one OpenCode session (no tmux required).
2. Keep beads (`bd`) as the source of truth for findings.
3. Use role-based subagents (`investigator-*`, `scribe`) orchestrated sequentially.
4. Persist inter-agent communication in a mailbox protocol that can later support parallel workers.

Non-goals for this implementation:

- No true parallel execution yet.
- No replacement of existing Go app behavior in-place.
- No forced plugin dependency in phase 1.

---

## Phase 0 - Baseline and Branch Safety

## Step 0.1: Capture current baseline

Deliverables:

- Confirm current project state and available OpenCode assets.

Verification:

- Run `git status` and confirm no accidental revert of existing work.
- Confirm existing templates are present under:
  - `templates/audit/.opencode/agents/`
  - `templates/audit/.opencode/skills/`
  - `templates/role-session/.opencode/agents/`
  - `templates/role-session/.opencode/skills/`

Exit criteria:

- Baseline is known and no unrelated files are modified.

---

## Phase 1 - Single-session orchestration scaffold

## Step 1.1: Create project-local orchestrator and subagent set

Deliverables:

- Add these files:
  - `.opencode/agents/audit-orchestrator.md` (primary)
  - `.opencode/agents/investigator-alpha.md`
  - `.opencode/agents/investigator-bravo.md`
  - `.opencode/agents/investigator-charlie.md`
  - `.opencode/agents/scribe.md`
- Ensure task permissions allow orchestrator -> investigators/scribe.
- Ensure investigator permissions allow `bd` and read-only code access.

Verification:

- OpenCode agent list shows all agents.
- `@audit-orchestrator` is selectable as primary or command target.
- Orchestrator can invoke `@investigator-alpha` and `@scribe` without permission deadlocks.

Exit criteria:

- Agent topology is usable end-to-end in one session.

## Step 1.2: Add workflow commands

Deliverables:

- Add command files:
  - `.opencode/commands/audit-start.md`
  - `.opencode/commands/audit-next-loop.md`
  - `.opencode/commands/audit-status.md`
  - `.opencode/commands/audit-close.md`
- Commands route to `audit-orchestrator`.

Verification:

- `/audit-start` initializes a new run folder.
- `/audit-next-loop` advances one loop only.
- `/audit-status` reports current loop, active investigators, and bead totals.
- `/audit-close` runs closeout checklist and updates completion state.

Exit criteria:

- Entire workflow is operable through these 4 commands.

---

## Phase 2 - Run-state contract and mailbox protocol

## Step 2.1: Define run folder structure

Deliverables:

- Implement run directory structure:

```
.opencode/audits/<run-id>/
  .team
  context/TASK.md
  context/REPORT.md
  mailbox/index.json
  mailbox/orchestrator.ndjson
  mailbox/investigator-alpha.ndjson
  mailbox/investigator-bravo.ndjson
  mailbox/investigator-charlie.ndjson
  mailbox/scribe.ndjson
  state/plan.json
  state/loop-status.json
  state/findings-map.json
  STATUS.md
```

Verification:

- `/audit-start` creates all required files (or all required except `REPORT.md`, which can be created on close).
- All files are valid UTF-8 text and parse correctly where JSON is expected.

Exit criteria:

- Run folder is deterministic and complete.

## Step 2.2: Freeze state schemas

Deliverables:

- Define and enforce JSON shape for:
  - `state/plan.json`
  - `state/loop-status.json`
  - `state/findings-map.json`
- Add schema notes inside `IMPLEMENTATION.md` appendix or separate `state/SCHEMA.md`.

Verification:

- Every command that writes state also validates required keys.
- On malformed state, orchestrator reports explicit repair instructions instead of proceeding silently.

Exit criteria:

- State files can be trusted for resume/recovery.

## Step 2.3: Implement mailbox protocol

Deliverables:

- Message format (one JSON object per line):
  - `ts`, `from`, `to`, `loop`, `kind`, `summary`, `beads`, `refs`, `thread` (optional)
- Add mailbox management skills:
  - `.opencode/skills/mailbox-write/SKILL.md`
  - `.opencode/skills/mailbox-read/SKILL.md`
- Optional shared stream:
  - `mailbox/chatroom.ndjson`

Verification:

- Each investigator invocation appends at least one message to mailbox.
- Orchestrator can reconstruct latest investigator status from mailbox only.
- Grep query over mailbox can list all findings with bead IDs.

Exit criteria:

- Mailbox is functional and auditable for future parallelization.

---

## Phase 3 - Skills and loop execution behavior

## Step 3.1: Promote and normalize core skills

Deliverables:

- Ensure these skills are available project-local:
  - `create-audit-plan`
  - `investigator-prompt`
  - `loop-prompt`
  - `compile-report`
  - `status-board`
  - `closeout-checklist`
- Enforce naming/frontmatter compatibility with OpenCode skill rules.

Verification:

- Skills are visible through the OpenCode skill tool.
- Orchestrator can load each skill without manual intervention.

Exit criteria:

- Skill layer is complete and callable in production runs.

## Step 3.2: Implement deterministic orchestrator loop

Deliverables:

- Loop algorithm:
  1. Load state and task context.
  2. For each active investigator, generate prompt and spawn subagent.
  3. Capture status (`FINDINGS` or `NOTHING_MORE`).
  4. Write mailbox event and update state files.
  5. Increment `.team current_loop` exactly once per global loop.
  6. Refresh `STATUS.md`.
  7. Early-exit when all investigators are `NOTHING_MORE`.

Verification:

- Dry run with intensity 3 produces exactly 0-3 loop increments (never over-increments).
- If all investigators return `NOTHING_MORE` after loop 1, loop 2 is not executed.
- Restarting session resumes from persisted `current_loop` and `loop-status.json`.

Exit criteria:

- Looping is deterministic, resumable, and bounded.

## Step 3.3: Bead discipline guardrails

Deliverables:

- Investigator instructions enforce `bd list` before `bd create`.
- Orchestrator triage behavior for duplicates and out-of-scope findings.
- Findings map links each bead to role + loop + references.

Verification:

- Test case with intentional duplicate finding results in update/comment, not duplicate bead creation.
- Test case with out-of-scope finding is closed or marked with rationale.

Exit criteria:

- Findings quality and dedupe rules are consistently enforced.

---

## Phase 4 - Reporting and closeout

## Step 4.1: Final report generation

Deliverables:

- `scribe` generates `context/REPORT.md` using bead-backed data only.
- Report includes:
  - scope
  - loops completed
  - findings by severity/area/role
  - bead references

Verification:

- Every reported finding maps to an existing bead ID.
- If there are zero findings, report explicitly states that result.

Exit criteria:

- Report is complete, traceable, and non-speculative.

## Step 4.2: Closeout checklist and session completion

Deliverables:

- `closeout-checklist` skill validates:
  - all actionable findings tracked in beads
  - duplicate pass complete
  - report generated
  - `bd sync` done
  - `git push` succeeded
  - `.team status=complete` set last

Verification:

- Checklist outputs pass/fail per item.
- Failed item blocks final complete state.

Exit criteria:

- Session cannot be marked complete without full closeout success.

---

## Phase 5 - Optional plugin hardening (next step)

## Step 5.1: Add runtime plugin

Deliverables:

- Add `.opencode/plugins/audit-runtime.ts` with hooks:
  - `tool.execute.before` guardrails by agent role
  - `tool.execute.after` mailbox event append
  - optional `shell.env` run context injection

Verification:

- Investigator attempts forbidden edits -> blocked with clear error.
- Successful tool executions auto-generate mailbox telemetry.

Exit criteria:

- Critical safety and observability are enforced programmatically.

## Step 5.2: Optional custom tools for mailbox/state

Deliverables:

- Implement plugin tools:
  - `mailbox_post`
  - `mailbox_pull`
  - `audit_state_get`
  - `audit_state_set`

Verification:

- Orchestrator can run without direct file parsing for mailbox/state operations.
- Tool outputs are deterministic and validated.

Exit criteria:

- Mailbox/state API is stable for future true multi-agent execution.

---

## Verification Matrix (End-to-end)

Run this acceptance flow:

1. Start new run with `/audit-start`.
2. Execute loop(s) with `/audit-next-loop` until done or early-exit.
3. Inspect `/audit-status` after each loop.
4. Complete with `/audit-close`.

Must verify:

- State files update correctly per loop.
- Mailbox captures all role handoffs.
- No duplicate beads for duplicate findings.
- Final report is bead-backed.
- `.team status=complete` only after sync/push/report success.

---

## Immediate Next Build Tasks (recommended order)

1. Create `audit-orchestrator` and copy/refine investigator/scribe agents.
2. Add `audit-start`, `audit-next-loop`, `audit-status`, `audit-close` commands.
3. Implement run folder + state schema files.
4. Implement `mailbox-write` and `mailbox-read` skills.
5. Add `status-board` and `closeout-checklist` skills.
6. Run one real audit and fix workflow friction.
7. Add plugin hardening only after workflow is stable.

---

## Definition of Complete (for this migration)

Migration is complete when all are true:

- A real audit can run end-to-end in one OpenCode session.
- Findings are created/updated in beads with dedupe discipline.
- Role handoff is persisted in mailbox logs.
- Report is generated from bead evidence.
- Closeout enforces sync/push and final state update.
- Workflow is command-driven and repeatable by another operator.
