# Audit Update Plan: Epic Beads & Per-Role Sessions

## Goal

Replace the current "one tmux window per audit type running a commissar" model with a bead-driven orchestration model where:

1. The launch step creates **epic entries** (local orchestration metadata in config.toml) for each audit type.
2. Under each epic, **role entries** are created for each role assignment within that audit type.
3. Each role spawns its **own opencode session** in a dedicated tmux window.
4. Roles within an epic run **sequentially** (senior dev finishes, then staff engineer starts).
5. Multiple epics run **in parallel** (Performance and Security at the same time).
6. Each session's findings are tracked with a deterministic bead prefix like `perf-senior-perf-specialist`.
7. When a role completes, its session ends and the next role in the same epic is spawned.

---

## Current Architecture (What Exists)

### Launch flow (`internal/tui/launch.go`)

```
For each selected audit type:
  1. Allocate bead prefix (e.g. "perf-1")
  2. Generate team folder from templates (commissar + investigators + scribe)
  3. Create tmux window "audit-{type}"
  4. Send: cd <team-dir> && opencode run commissar
  5. Save team state to config.toml
```

The commissar internally manages the audit loop, spawns investigators as subagents within the same opencode session, and the scribe writes the report. Everything lives in one long-running session per audit type.

### Template system (`templates/audit/`)

Agent definitions (commissar, investigator-alpha/bravo/charlie, scribe) are embedded templates. The generator walks the template tree, renders `.tmpl` files with `TemplateData`, and copies static files.

### Key files touched by this change

| File | Role |
|---|---|
| `internal/tui/launch.go` | Orchestrates tmux session/window creation and opencode commands |
| `internal/teams/generator.go` | Generates team folder from templates |
| `internal/teams/audit_types.go` | Defines audit types, roles, focus areas |
| `internal/teams/bead_prefix.go` | Allocates unique bead prefixes |
| `internal/config/config.go` | Persists session/team state to TOML |
| `internal/tui/dashboard.go` | Reads config + .team files for live status |
| `templates/audit/` | Embedded template tree for team folders |
| `internal/tui/audit_wizard.go` | Wizard flow (unmodified but consumes output) |

---

## Design Decisions

### Decision 1: Epic/role entries are local metadata, not `bd` items

Epic and role entries live in `config.toml` as orchestration state. They are **not** `bd` beads.

**Rationale**: `bd` is a tool that runs inside opencode sessions. At plan-build time, opencode sessions don't exist yet. Creating `bd` items from the Go app would require shelling out to `bd` (which may not be on PATH outside WSL), or reverse-engineering the bead file format. This creates a fragile circular dependency: the orchestrator depends on a tool that only exists inside the environment it orchestrates.

The `audit-plan-NNN` IDs are local identifiers used for config cross-referencing and human readability. Finding beads (created by auditors during their sessions) use `bd` as before — those are the real traceable artifacts.

### Decision 2: Scheduler is dashboard-coupled (with solid recovery)

The scheduler runs inside the dashboard's 3-second tick loop. This is an intentional choice, not a gap.

**Why this is acceptable**:
- The opencode sessions run in tmux **regardless** of whether the TUI is open. Work doesn't stop — only advancement to the next role pauses.
- This gives the user control: they can observe progress and intervene before the next role launches.
- A background daemon adds significant complexity (IPC, process lifecycle, PID management, log routing) for a narrow edge case.

**What makes it safe**: The recovery model (Decision 4) ensures that restarting the TUI picks up exactly where it left off. No work is lost. The worst case is a delay between role completions.

### Decision 3: Single counter, no PlanCounter

The existing `BeadCounter` in config.toml is used for everything — both `audit-plan-NNN` IDs and finding bead prefixes. One counter, sequential, no ambiguity about ownership or lifecycle.

`BuildAuditPlan` takes `startCounter` (read from `cfg.BeadCounter`), allocates IDs, and returns the final counter value. The caller writes it back.

### Decision 4: Role State Machine

Every role follows a strict state machine:

```
pending ──→ running ──→ complete
                │
                └──→ failed
```

**State transitions**:

| From | To | Trigger |
|---|---|---|
| `pending` | `running` | Scheduler launches role (previous role complete or first in epic) |
| `running` | `complete` | `.team` file shows `status=complete` |
| `running` | `failed` | Tmux window no longer exists AND `.team` status is not `complete` |
| `failed` | _(terminal)_ | No automatic retry. User must intervene. |

**Why no `stalled` state or timeout**: Audit sessions can legitimately run for a long time (the "Go Hard" rigor is 99 loops). Any timeout threshold would be arbitrary and risk false positives. The tmux window check is a reliable crash detector: if opencode dies, the shell exits, and the tmux window closes (or shows a dead shell). This is the only signal we need.

**Why no automatic retry**: If an opencode session crashes, the cause is unknown. Blindly relaunching it could repeat the crash, corrupt bead state, or duplicate findings. The dashboard shows `failed` status; the user decides what to do.

**Epic status** is derived from its roles:
- `running` if any role is `running` or `pending`
- `complete` if all roles are `complete`
- `failed` if any role is `failed` and no role is `running`

### Decision 5: Idempotent Scheduler

`CheckAndAdvanceRoles` must be safe to call repeatedly with the same state. Rules:

1. If a role is `running` and its tmux window exists → **do nothing**.
2. If a role is `running` and its tmux window does NOT exist AND `.team` status is `complete` → mark `complete`, advance to next.
3. If a role is `running` and its tmux window does NOT exist AND `.team` status is NOT `complete` → mark `failed`.
4. If a role is `pending` and the previous role in its epic is `complete` → launch it (generate folder, create window, send keys, mark `running`).
5. If a role is `pending` and the previous role is `failed` → **do nothing** (epic is blocked).
6. If a role is `complete` or `failed` → **do nothing**.

This is crash-safe. If the app restarts:
- Config.toml is the source of truth.
- Running roles are reconciled against tmux window existence.
- No duplicate launches because pending→running only happens when the previous role is `complete`.

### Decision 6: Atomic Config Writes

`Config.Save()` will be updated to write-to-temp-then-rename:

```go
func (c *Config) Save() error {
    tmp := c.filePath + ".tmp"
    // write to tmp
    // os.Rename(tmp, c.filePath)
}
```

This prevents partial writes if the app crashes mid-save. The Bubble Tea update loop is single-threaded so there's no concurrent access from within the app. The opencode sessions write to per-role `.team` files, not to `config.toml`, so there's no cross-process contention on the config file.

### Decision 7: No `AgentCount` on RoleState

The wizard's "agent count" (1/2/3) determines how many roles are activated per audit type. Once the plan is built, each role session is always a single-agent session. Storing `AgentCount: 1` on every role is redundant and confusing. The field is omitted from `RoleState`.

The wizard's agent count is recorded once on `EpicState` for audit trail purposes.

### Decision 8: Slug Derivation for Role Bead Prefixes

Role bead prefixes follow the format: `{audit-type-bead-prefix}-{role-slug}`

The slug is derived from the role title:
1. Lowercase the title.
2. Replace spaces with hyphens.
3. Remove non-alphanumeric-non-hyphen characters.
4. Collapse consecutive hyphens.
5. Truncate to 30 characters at a word boundary (no mid-word cuts).

**Why collisions aren't a real risk**: Role titles are static constants in `audit_types.go`. They're defined by us, not user input. Within a single audit type, each role has a unique title by construction. Across audit types, the `{audit-type-bead-prefix}` segment differentiates them (e.g. `perf-` vs `sec-`).

**Test requirement**: A test in `plan_test.go` will assert that all role bead prefixes across all audit types and agent counts are unique. This is a compile-time-adjacent guarantee — if someone adds a colliding role title, the test fails.

### Decision 9: Completed Windows Stay Open

When a role completes, its tmux window is **not** killed. It stays open so the user can inspect the session output. The dashboard marks it `complete` with a visual indicator.

This is a UX choice: audit output is valuable context. Killing windows silently discards it. If windows pile up, the user can kill them manually or we can add a cleanup key to the dashboard later.

---

## Conceptual Model

```
Epic: "Performance Audit"                  (config ID: audit-plan-001)
  ├── Role: "Senior performance specialist" (config ID: audit-plan-002)
  │     → tmux window: perf-alpha
  │     → finding bead prefix: perf-senior-performance-specialist
  │     → runs first, session ends on completion
  │
  ├── Role: "Staff performance engineer"    (config ID: audit-plan-003)
  │     → tmux window: perf-bravo
  │     → finding bead prefix: perf-staff-performance-engineer
  │     → runs after alpha completes
  │
  └── Role: "Runtime optimization specialist" (config ID: audit-plan-004)
        → tmux window: perf-charlie
        → finding bead prefix: perf-runtime-optimization-spec
        → runs after bravo completes

Epic: "Security Audit"                     (config ID: audit-plan-005)
  ├── Role: "Senior security specialist"   (config ID: audit-plan-006)
  │     ...runs in parallel with perf epic...
  └── ...
```

---

## Data Structures

### Audit Plan (build-time, in-memory)

```go
// internal/teams/plan.go

type RoleBead struct {
    BeadID      string // e.g. "audit-plan-002"
    CodeName    string // e.g. "alpha"
    Title       string // e.g. "Senior performance specialist"
    Guidance    string
    BeadPrefix  string // finding prefix: "perf-senior-performance-specialist"
    Order       int    // execution order within the epic (0-based)
}

type EpicBead struct {
    BeadID     string     // e.g. "audit-plan-001"
    AuditType  AuditType
    RoleBeads  []RoleBead // ordered by execution sequence
}

type AuditPlan struct {
    Epics       []EpicBead
    FinalCounter int       // BeadCounter value after all IDs allocated
}
```

### Config State (persisted to TOML)

```go
// internal/config/config.go

type EpicState struct {
    BeadID     string `toml:"bead_id"`      // "audit-plan-001"
    AuditType  string `toml:"audit_type"`   // "perf"
    AuditName  string `toml:"audit_name"`   // "Performance Audit"
    AgentCount int    `toml:"agent_count"`  // wizard selection (1-3)
    Intensity  int    `toml:"intensity"`
    Status     string `toml:"status"`       // derived: "running", "complete", "failed"
}

type RoleState struct {
    BeadID       string `toml:"bead_id"`       // "audit-plan-002"
    EpicBeadID   string `toml:"epic_bead_id"`  // "audit-plan-001"
    CodeName     string `toml:"code_name"`     // "alpha"
    Title        string `toml:"title"`         // "Senior performance specialist"
    Guidance     string `toml:"guidance"`
    BeadPrefix   string `toml:"bead_prefix"`   // "perf-senior-performance-specialist"
    Order        int    `toml:"order"`         // 0, 1, 2
    Status       string `toml:"status"`        // "pending", "running", "complete", "failed"
    TmuxWindow   string `toml:"tmux_window"`   // "lattice-...:perf-alpha"
    Intensity    int    `toml:"intensity"`
}

type Config struct {
    Session      SessionMetadata          `toml:"session"`
    BeadCounter  int                      `toml:"bead_counter"`
    Teams        map[string]TeamState     `toml:"teams"`     // kept for backward compat
    Epics        map[string]EpicState     `toml:"epics"`
    Roles        map[string]RoleState     `toml:"roles"`
    filePath     string                   `toml:"-"`
}
```

### Role Session Template Data (render-time)

```go
// internal/teams/generator.go

type RoleSessionData struct {
    TeamName     string   // e.g. "perf-alpha"
    EpicBeadID   string   // e.g. "audit-plan-001"
    RoleBeadID   string   // e.g. "audit-plan-002"
    RoleTitle    string   // e.g. "Senior performance specialist"
    RoleGuidance string   // role-specific guidance text
    Intensity    int
    BeadPrefix   string   // finding prefix: "perf-senior-performance-specialist"
    Target       string
    FocusAreas   []string
}
```

---

## BuildAuditPlan

```go
func BuildAuditPlan(
    auditTypes   []AuditType,
    agentCount   int,
    intensity    int,
    startCounter int,
) (*AuditPlan, error)
```

**Parameters**:
- `auditTypes`: selected audit types from wizard.
- `agentCount`: 1-3, determines which `RoleConfigs` entry to use per audit type.
- `intensity`: passed through to each `RoleBead` for template rendering. Does not affect plan structure.
- `startCounter`: current `cfg.BeadCounter` value.

**Algorithm**:
1. Validate inputs (at least 1 audit type, agentCount 1-3, intensity >= 1).
2. `counter := startCounter`
3. For each audit type:
   a. `counter++` → epic bead ID = `fmt.Sprintf("audit-plan-%03d", counter)`
   b. Look up `RoleConfigs` for the given `agentCount`. Error if not found.
   c. For each role in the config:
      - `counter++` → role bead ID = `fmt.Sprintf("audit-plan-%03d", counter)`
      - Derive `BeadPrefix` = `auditType.BeadPrefix + "-" + slugify(role.Title)`
      - Create `RoleBead` with order = role index.
   d. Create `EpicBead` with the role beads.
4. Return `AuditPlan{Epics: epics, FinalCounter: counter}`.

---

## Scheduler: `CheckAndAdvanceRoles`

```go
// internal/tui/scheduler.go

type SchedulerDeps struct {
    GenerateRoleSession func(params RoleSessionParams) (string, error)
    TranslatePath       func(path string) (string, error)
    TmuxManager         launchTmuxManager
    CheckTmuxWindow     func(sessionName, windowName string) bool
    Now                 func() time.Time
}

type ScheduledRole struct {
    BeadID     string
    CodeName   string
    Title      string
    EpicBeadID string
    TmuxWindow string
}

type SchedulerResult struct {
    Launched  []ScheduledRole
    Completed []string // role bead IDs
    Failed    []string // role bead IDs
    AllDone   bool
}

func CheckAndAdvanceRoles(
    cwd string,
    cfg *config.Config,
    sessionName string,
    plan *AuditPlan, // needed for template data when generating new role sessions
    deps SchedulerDeps,
) (SchedulerResult, error)
```

**CheckTmuxWindow**: A new helper that runs `tmux has-session -t <session>` or `tmux list-windows` and checks for the window. This is the crash detection mechanism. Added to `SchedulerDeps` for testability (faked in tests).

**Flow per tick**:
1. Load config (or use passed-in `cfg`).
2. Group roles by epic.
3. For each epic:
   a. Find the role with `status=running`.
   b. If found, check its `.team` file:
      - If `status=complete` → mark role `complete` in config, check for next pending role.
      - If `status!=complete`, check tmux window:
        - Window exists → do nothing (still running).
        - Window gone → mark role `failed` in config.
   c. If no running role, find first `pending` role whose predecessor is `complete`:
      - Generate role session folder.
      - Create tmux window, send opencode command.
      - Mark `running` in config.
   d. If all roles are `complete` or `failed` with none `pending` → epic is done.
4. Save config if anything changed.
5. Return result.

---

## New Templates: `templates/role-session/`

### `.team.tmpl`

```
team={{ .TeamName }}
epic_bead_id={{ .EpicBeadID }}
role_bead_id={{ .RoleBeadID }}
role={{ .RoleTitle }}
intensity={{ .Intensity }}
current_loop=0
status=active
```

### `INSTRUCTIONS.md.tmpl`

Adapted from current `INSTRUCTIONS.md.tmpl`. Key changes:
- Bead prefix is `{{ .BeadPrefix }}` (role-specific, not team-level).
- Session completion updates `.team` to `status=complete` as the **final mandatory step**.
- No references to commissar or multi-investigator coordination.

### `context/TASK.md.tmpl`

```
# Audit Task

## Epic
{{ .EpicBeadID }}: {{ .TeamName }}

## Your Role
**{{ .RoleTitle }}**

{{ .RoleGuidance }}

## Bead Prefix
Use `{{ .BeadPrefix }}` for any findings from this audit.

## Target
{{ .Target }}

## Focus Areas
{{- range .FocusAreas }}
- {{ . }}
{{- end }}

## Rules
- Only raise issues that have real impact. Do not manufacture problems.
- Do not raise issues outside the audit's focus areas.
- If an existing bead already covers a finding, update it — do not create a duplicate.
- If there are no issues, say so. An empty audit is a valid outcome.
```

### `.opencode/agents/auditor.md`

A single self-managing agent that combines the commissar's loop logic with the investigator's audit capability. Key behaviors:
- Reads `.team` for intensity and loop counter.
- On each loop: investigates from its role perspective, creates/updates beads.
- After each loop: increments `current_loop` in `.team`.
- If nothing more to find → exits early.
- When done (all loops or early exit): spawns scribe, follows completion steps, sets `.team` `status=complete`.

### Other files

- `DESCRIPTION.md` — explains the single-role session model.
- `opencode.jsonc` — same as current (`opencode-beads` plugin).
- `.opencode/agents/scribe.md` — same as current.
- `.opencode/skills/audit-prompt/SKILL.md` — adapted for single role.
- `.opencode/skills/loop-prompt/SKILL.md` — adapted for single role.
- `.opencode/skills/compile-report/SKILL.md` — same as current.

---

## Updated Launch Flow

```
1. Build AuditPlan from wizard selections
2. Write cfg.BeadCounter = plan.FinalCounter
3. Init config, create tmux session

4. For each epic in plan:
   a. Record EpicState in cfg.Epics[auditType.ID]
   b. For each role in epic:
      - Record RoleState in cfg.Roles[beadID]
      - If order == 0 (first role):
        - Generate role-session folder
        - Create tmux window
        - Send: cd <role-session-dir> && opencode run auditor
        - Set status = "running", record TmuxWindow
      - Else:
        - Set status = "pending"

5. Save config (atomic write)
```

---

## Updated Dashboard

### Snapshot struct

```go
type dashboardEpicStatus struct {
    EpicName      string
    BeadID        string
    AuditType     string
    Status        string // derived from roles
    RolesTotal    int
    RolesComplete int
    RolesFailed   int
    Roles         []dashboardRoleStatus
}

type dashboardRoleStatus struct {
    BeadID      string
    CodeName    string
    Title       string
    Status      string // pending, running, complete, failed
    CurrentLoop int
    Intensity   int
    BeadPrefix  string
}
```

### View

```
LATTICE
Post-launch Audit Status

Session: lattice-20260212-143201
Last refresh: 2:32 PM

EPIC                      STATUS     PROGRESS
Performance Audit         running    1/3 roles done
  alpha (Sr. Perf Spec)   complete   loop 3/3
  bravo (Staff Perf Eng)  running    loop 1/3
  charlie (Runtime Opt)   pending    -
Security Audit            running    0/2 roles done
  alpha (Sr. Sec Spec)    running    loop 2/3
  bravo (Staff AppSec)    pending    -

t: attach tmux  r: refresh  esc: menu  q: quit
```

### Scheduler integration

On each `dashboardTickMsg`:
1. Call `CheckAndAdvanceRoles`.
2. If roles were launched/completed/failed → emit `schedulerAdvancedMsg` with the result.
3. Dashboard update handler processes the message and refreshes the view.
4. If `AllDone` → show completion banner.

---

## Implementation Phases

### Phase 1: Core Data Model

**1.1** Create `internal/teams/plan.go`:
- `RoleBead`, `EpicBead`, `AuditPlan` structs.
- `BuildAuditPlan` function.
- `slugify` helper (lowercase, hyphens, truncate at 30 chars).

**1.2** Create `internal/teams/plan_test.go`:
- 1 audit type, 1 agent → 1 epic, 1 role.
- 2 audit types, 3 agents → 2 epics, 3+3 roles.
- Sequential IDs: audit-plan-001, 002, ...
- Correct bead prefix derivation.
- Error on 0 audit types, invalid agent count.
- **Uniqueness test**: all prefixes across all audit types x all agent counts are unique.

**1.3** Add `Epics`, `Roles` to `config.Config`. Add nil-map guards in `Load` and `Save` (same pattern as `Teams`). Remove `PlanCounter` — use `BeadCounter` only.

**1.4** Update `Config.Save()` to write-to-temp-then-rename (atomic write).

**1.5** Config tests: round-trip with Epics/Roles populated; load old config without those fields; atomic write doesn't corrupt on partial failure.

### Phase 2: Role Session Templates

**2.1** Create `templates/role-session/` tree (see Templates section above).

**2.2** Update `templates/embed.go`:
```go
//go:embed all:role-session
var RoleSessionTemplate embed.FS
```

**2.3** Add `RoleSessionParams`, `RoleSessionData`, and `GenerateRoleSession` to `internal/teams/generator.go`. Walks `templates/role-session/` with `RoleSessionData`.

**2.4** Generator tests:
- `.team` contains epic bead ID, role bead ID, role title.
- `context/TASK.md` contains role perspective, guidance, focus areas.
- `INSTRUCTIONS.md` contains correct bead prefix.
- `auditor.md` and `scribe.md` exist; no investigator files.
- No commissar file.

### Phase 3: Updated Launch

**3.1** Rewrite `launchAudit` in `internal/tui/launch.go`:
- Calls `BuildAuditPlan`.
- Creates tmux session.
- Launches first role per epic, records all roles.
- Saves config with atomic write.

**3.2** Update `launchDeps` to include `generateRoleSession` and `buildAuditPlan`.

**3.3** Update `launchRequest` — add `focusAreas` and `target` fields already present, remove any that are now derived from the plan.

**3.4** Launch tests:
- Windows created only for first roles.
- Config has correct epic states and role states.
- Pending roles recorded but no window/keys.
- `BeadCounter` advanced to `plan.FinalCounter`.

### Phase 4: Scheduler

**4.1** Create `internal/tui/scheduler.go`:
- `CheckAndAdvanceRoles` with deps injection.
- `CheckTmuxWindow` helper (real implementation uses `tmux list-windows`).
- Role state transitions per Decision 5.

**4.2** Add `HasWindow` or `ListWindows` usage to tmux manager (may already be sufficient with existing `ListWindows`).

**4.3** Scheduler tests:
- Running role completes → next role launches.
- Running role's window disappears without completion → marked `failed`.
- Last role completes → epic status derived as `complete`.
- Role fails → subsequent pending roles stay pending (epic blocked).
- Idempotent: calling twice with same state → no duplicate launches.
- Crash recovery: role marked `running` but window gone → detected as failed.

**4.4** Wire into dashboard: call on each tick, process `SchedulerResult`.

### Phase 5: Dashboard View

**5.1** Replace `dashboardTeamStatus` with `dashboardEpicStatus` / `dashboardRoleStatus`.

**5.2** Update `loadDashboardSnapshot` to read from `cfg.Epics` and `cfg.Roles`.

**5.3** Update `renderTeamTable` → `renderEpicTable` with hierarchical view.

**5.4** Handle `schedulerAdvancedMsg` in dashboard Update.

**5.5** Dashboard tests for new view and scheduler integration.

### Phase 6: Integration & Cleanup

**6.1** Backward compat: old configs with `Teams` but no `Epics`/`Roles` load without error.

**6.2** `go test ./...` passes.

**6.3** Manual e2e: wizard → plan built → first roles launch → role completes → next role auto-launches → dashboard shows hierarchy.

---

## Files Created

| File | Purpose |
|---|---|
| `internal/teams/plan.go` | AuditPlan, EpicBead, RoleBead, BuildAuditPlan, slugify |
| `internal/teams/plan_test.go` | Plan building + prefix uniqueness tests |
| `internal/tui/scheduler.go` | CheckAndAdvanceRoles, SchedulerDeps, role state logic |
| `internal/tui/scheduler_test.go` | Scheduler tests (state transitions, idempotency, crash recovery) |
| `templates/role-session/` (tree) | New template set for per-role sessions |

## Files Modified

| File | Changes |
|---|---|
| `internal/config/config.go` | Add Epics, Roles maps; nil-map guards; atomic Save |
| `internal/config/config_test.go` | Tests for new fields, atomic write, backward compat |
| `internal/teams/generator.go` | Add GenerateRoleSession, RoleSessionParams, RoleSessionData |
| `internal/teams/generator_test.go` | Tests for role session generation |
| `internal/tui/launch.go` | Rewrite launchAudit: build plan → launch first roles |
| `internal/tui/launch_test.go` | Tests for new launch flow |
| `internal/tui/dashboard.go` | Epic/role hierarchy view, scheduler integration |
| `internal/tui/dashboard_test.go` | Tests for new dashboard |
| `templates/embed.go` | Embed role-session template tree |

## Files NOT Modified

| File | Reason |
|---|---|
| `internal/tui/audit_wizard.go` | Wizard collects the same inputs; plan building happens downstream |
| `internal/tui/app.go` | Navigation unchanged; wizard → dashboard flow stays the same |
| `internal/tui/menu.go` | Unchanged |
| `internal/tui/multiselect.go` | Unchanged |
| `internal/tui/styles.go` | Unchanged |
| `internal/tui/keys.go` | Unchanged |
| `internal/tmux/manager.go` | Existing methods are sufficient (ListWindows covers window checks) |
| `internal/teams/audit_types.go` | Audit type definitions unchanged |
| `internal/teams/bead_prefix.go` | Prefix allocation logic unchanged |
| `internal/discovery/discover.go` | Discovery unchanged |
| `templates/audit/` | Kept intact for backward compatibility |

---

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Scheduler misses a completion while dashboard is closed | Opencode sessions still run and complete in tmux. On next dashboard open, scheduler reconciles state immediately. No work lost, just delayed advancement. |
| Role session crashes (opencode exits unexpectedly) | Tmux window disappears → scheduler detects on next tick → marks `failed`. No silent data loss. Dashboard shows failure clearly. |
| App crashes mid-config-write | Atomic write (temp + rename) prevents partial TOML. On restart, config is either the old valid state or the new valid state. |
| Duplicate role launch after app restart | Scheduler is idempotent: only transitions `pending → running` when predecessor is `complete`. A role already `running` (even if window is gone) is reconciled first, never re-launched. |
| Config.toml concurrent access | Not possible. Bubble Tea Update is single-threaded. Opencode sessions write to per-role `.team` files, never to `config.toml`. |
| Too many tmux windows | Max concurrent = number of epics (one role per epic at a time). Completed windows stay open but are inert. Worst case with 8 audit types × 3 roles = 24 windows total over the run's lifetime, but only 8 active at once. |
| Bead prefix collision | Static role titles in `audit_types.go` + unique audit type prefixes = structurally impossible within a plan. Enforced by a uniqueness test. |
| Old config.toml without Epics/Roles | nil-map guards on Load, same pattern as existing Teams field. |

---

## Migration & Backward Compatibility

- Old `.lattice/config.toml` files without `[epics]` or `[roles]` tables load cleanly. TOML decoder ignores missing sections; nil-map guards initialize empty maps.
- The `Teams` map is retained in the config struct. Existing code that reads `cfg.Teams` continues to work. New code uses `cfg.Epics` and `cfg.Roles`.
- The old `templates/audit/` tree is not deleted. It remains embedded and available. The new `templates/role-session/` tree is additive.
- If a user has a partially-completed old-style audit (commissar-based), the dashboard falls back to showing `cfg.Teams` if `cfg.Epics` is empty.
