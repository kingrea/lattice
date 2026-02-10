# Audit Team

This is an audit team. It does not build anything. It investigates an existing codebase, finds real issues, and turns them into actionable beads.

## Team Structure

- **Commissar** (1): Reads the task, assigns roles and focus areas to investigators, reviews findings after each loop, decides when to stop.
- **Investigators** (3): Perform the actual audit. Each is assigned a role perspective (e.g., senior engineer, staff engineer, security specialist). They use judgement, not rote instructions.
- **Scribe** (1): Compiles the final audit report after all loops are complete.

## How It Works

### Intensity

The `.team` file contains an `intensity` value (e.g., 3). This is the maximum number of audit loops the team will perform.

- **Loop 1**: Initial audit pass. Investigators examine the target area from their assigned role's perspective and create beads for findings.
- **Loop 2+**: Commissar re-prompts investigators: "keep looking, don't duplicate." Investigators go deeper.
- **Early exit**: If an investigator reports nothing more to find, they stop. When all investigators have stopped or intensity is reached, the audit ends.

### Roles

The task will specify 1–3 roles (e.g., "senior engineer", "staff engineer", "security specialist"). The commissar assigns each investigator a role. If fewer roles than investigators, unused investigators sit idle.

Each role brings a different perspective to the same target area. A senior engineer might focus on architectural issues while a security specialist focuses on vulnerabilities.

### Beads

Findings become beads. Before creating a bead, investigators MUST check if an existing bead already covers the issue. If so, update the existing bead's description rather than creating a duplicate.

Only create beads for issues worth raising. If the audit finds nothing, that is a valid outcome.
