# OpenCode Audit Migration Plan (No TUI Required)

This plan turns your current Go/TUI-driven audit workflow into a single long-running OpenCode session that uses subagents, skills, and beads.

Short answer: yes, this is very doable, and you already have most of the pieces.

---

## Executive Direction

Use **one primary orchestrator agent** in a single OpenCode session.

- Keep `bd` as the source of truth for findings and workflow state.
- Keep role specialization using subagents (`investigator-*`, `scribe`).
- Replace tmux scheduling with a deterministic in-session loop.
- Add a simple filesystem mailbox so agents can hand off context cleanly.

You do not need parallelism to get value. Sequential orchestration with strict state files is enough.

---

## What You Already Have (And Should Reuse)

Your repo already contains a strong blueprint:

- Agent definitions in `templates/audit/.opencode/agents/` and `templates/role-session/.opencode/agents/`
- Skills in `templates/audit/.opencode/skills/` and `templates/role-session/.opencode/skills/`
- Team/session state model via `.team` templates
- Bead-centric process and duplicate prevention in instruction templates
- Audit taxonomy/role lenses in `internal/teams/audit_types.go`

The Go app mainly adds:

- TUI steps and discovery UX (`internal/tui/audit_wizard.go`)
- tmux process launching (`internal/tui/launch.go`, `internal/tmux/manager.go`)
- scheduler progression (`internal/tui/scheduler.go`)

For your new approach, keep the first group, replace the second group.

---

## Mapping: Go App -> OpenCode-Native

| Current Go Component | Role Today | OpenCode Replacement |
|---|---|---|
| Audit wizard steps | collect mode/types/rigor | `/audit-start` custom command + startup skill |
| `BuildAuditPlan` | create epic/role bead plan | `create-audit-plan` skill (already present; refine output schema) |
| tmux windows per role | concurrent role execution | sequential subagent calls in one session |
| scheduler | role state transitions | orchestrator loop with state file + mailbox |
| dashboard | runtime visibility | markdown status board + mailbox index + periodic summary |

---

## Target Architecture (Single-Threaded)

### 1) Primary Agent: `audit-orchestrator`

Responsibilities:

- Load task context and `.team` state
- Run loop `1..intensity`
- Spawn role subagents one-at-a-time
- Validate findings against scope/duplicates
- Maintain mailbox and status board
- Spawn `scribe` at completion

This agent should be your daily driver for audits.

### 2) Subagents

- `investigator-alpha`
- `investigator-bravo`
- `investigator-charlie`
- `scribe`
- Optional: `triage-judge` (hidden) to enforce quality rules on dubious findings

### 3) Skills

Keep and sharpen:

- `create-audit-plan`
- `investigator-prompt`
- `loop-prompt`
- `compile-report`

Add:

- `mailbox-write`
- `mailbox-read`
- `status-board`
- `closeout-checklist`

### 4) Filesystem State (Replacing tmux state)

Create an explicit workspace:

```
.opencode/
  audits/
    <run-id>/
      .team
      context/
        TASK.md
        REPORT.md
      mailbox/
        index.json
        investigator-alpha.ndjson
        investigator-bravo.ndjson
        investigator-charlie.ndjson
        orchestrator.ndjson
        scribe.ndjson
      state/
        plan.json
        loop-status.json
        findings-map.json
      STATUS.md
```

---

## Mailbox / Agent Chatroom: Is There Legs?

Yes. In OpenCode, a file-backed mailbox is practical and robust.

### Minimal mailbox protocol

Each message is one JSON line in `mailbox/<agent>.ndjson`:

```json
{"ts":"2026-02-17T10:34:01Z","from":"investigator-alpha","to":"orchestrator","loop":1,"kind":"finding","beads":["sec-42"],"summary":"Potential authz bypass in admin route","refs":["src/api/admin.ts:88"]}
```

### Why this works

- Survives context compaction and session restarts
- Fully auditable and grep-friendly
- No plugin required to start
- Can later be wrapped by a plugin/custom tool API (`mailbox_post`, `mailbox_pull`)

### Chatroom mode

If you want “agent chatroom”, use a shared stream:

- `mailbox/chatroom.ndjson`
- include `thread` and `mentions` fields

This gives asynchronous team conversation without changing the core orchestration loop.

---

## OpenCode Ecosystem Fit (Quick Assessment)

Based on OpenCode docs:

- Agents support primary/subagent workflows and task permissions
- Skills are first-class, repo-local, and load on demand
- Commands can encode reusable workflows (`/audit-start`, `/audit-loop`, `/audit-close`)
- Plugins can add hooks/custom tools for mailbox and guardrails

So your desired architecture is natively supported, even before adding parallel execution.

---

## Concrete Build Plan

## Phase 1: Stabilize single-session orchestration (no plugin)

1. Promote one primary agent `audit-orchestrator` under `.opencode/agents/`.
2. Reuse/refine your existing investigator and scribe prompts from templates.
3. Add run folder convention under `.opencode/audits/<run-id>/`.
4. Add strict state files (`plan.json`, `loop-status.json`, `.team`).
5. Implement mailbox as ndjson files.
6. Add `STATUS.md` generator skill to summarize progress every loop.

Exit criteria:

- One full audit can run end-to-end in a single OpenCode session.
- Findings become beads with no duplicates.
- `context/REPORT.md` is generated and traceable.

## Phase 2: Workflow commands

Add custom commands in `.opencode/commands/`:

- `audit-start.md`
- `audit-next-loop.md`
- `audit-status.md`
- `audit-close.md`

Each command should target `audit-orchestrator` and keep prompts tiny/consistent.

Exit criteria:

- You can operate the audit with 4 predictable commands.
- Minimal prompt ceremony and fewer operator errors.

## Phase 3: Plugin hardening (TypeScript)

Create `.opencode/plugins/audit-runtime.ts` to add:

- Pre-tool guardrails (block writes outside run folder for investigator agents)
- Auto-append events to mailbox on `tool.execute.after`
- Optional custom tools:
  - `mailbox_post`
  - `mailbox_pull`
  - `audit_state_get`
  - `audit_state_set`

Exit criteria:

- Mailbox/state updates are automatic and consistent.
- Agent behavior constraints are enforced, not just instructed.

---

## Orchestrator Loop (Reference Algorithm)

```text
load TASK.md, .team, plan.json
if first run:
  run create-audit-plan
  persist plan.json and initial loop-status.json

for loop in 1..intensity:
  for each active investigator in assignment order:
    if investigator is done: continue
    generate prompt via investigator-prompt (loop1) or loop-prompt (loop2+)
    spawn investigator subagent
    parse response (FINDINGS | NOTHING_MORE)
    write mailbox event
    validate duplicate/scope discipline
    update loop-status.json

  increment .team current_loop
  regenerate STATUS.md
  if all investigators NOTHING_MORE:
    break

spawn scribe
generate REPORT.md
run closeout-checklist
set .team status=complete
```

---

## Data Contracts You Should Freeze Early

To keep this maintainable, define these JSON schemas (lightweight):

- `plan.json`
  - `run_id`, `target`, `audit_type`, `intensity`, `assignments[]`
- `loop-status.json`
  - `current_loop`, `max_loops`, `investigators[agent].status`, `last_summary`
- `findings-map.json`
  - bead id to role/loop/path evidence mapping

This gives deterministic behavior and easier future multi-agent migration.

---

## Beads Strategy (Keep It Simple)

- Keep role-specific bead prefixes (`<audit-prefix>-<role-slug>`), matching your existing model.
- Require `bd list` check before `bd create` in every investigator pass.
- Use orchestrator to close out-of-scope or duplicate beads with rationale comments.
- Keep final report bead-backed only (no new findings in report stage).

---

## Suggested Agent Set (Initial)

- `audit-orchestrator` (primary)
  - can read/edit run state + call investigator/scribe tasks + run `bd` and `git` commands
- `investigator-*` (subagent)
  - read-only code + `bd` commands, no file edits except mailbox append if you allow it
- `scribe` (subagent)
  - write `context/REPORT.md` only

If you want stronger safety, enforce these via permissions and later via plugin hooks.

---

## Known Risks and Mitigations

- Context drift in long sessions -> mitigate with `STATUS.md` regeneration and mailbox summaries.
- Duplicate findings across loops -> mitigate with mandatory bead map checks and `bd list` before every create.
- Overly verbose/weak findings -> add `triage-judge` subagent for QA before accepting new beads.
- Session interruption -> resume from run folder (`.team`, `loop-status.json`, mailbox logs).

---

## Migration Recommendation

Do this incrementally, not as a rewrite.

1. Lift your existing template agents/skills into project-level `.opencode/` for direct use.
2. Add the run-folder state protocol and mailbox.
3. Run real audits in single-thread mode until reliable.
4. Then add plugin automation and optional multi-agent parallelism.

You can get production usefulness before touching advanced orchestration.

---

## Optional Next Step (If You Want Me To Implement)

I can next create the first working scaffold in this repo:

- `.opencode/agents/audit-orchestrator.md`
- `.opencode/skills/mailbox-write/SKILL.md`
- `.opencode/skills/mailbox-read/SKILL.md`
- `.opencode/commands/audit-start.md`
- `.opencode/commands/audit-next-loop.md`
- `.opencode/commands/audit-close.md`

That would give you an immediately runnable single-session audit workflow with no Go/TUI dependency.
