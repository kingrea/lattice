# Role Session

This is a single-role audit session. One auditor runs the full loop from one role perspective, records actionable findings as beads, and exits when no further useful findings remain.

## Session Model

- **Auditor** (1): Reads the task, audits the target area using the assigned role guidance, creates or updates beads, and manages loop progression.
- **Scribe** (1): Compiles the final report from bead history and loop outcomes after auditing is complete.

## How It Works

The `.team` file tracks session state:

- `intensity` is the maximum loop count.
- `current_loop` is incremented after each completed loop.
- `status` starts as `active` and is set to `complete` when all completion steps are finished.

Each loop should search for real issues from the assigned role perspective while avoiding duplicates. Early exit is expected when no additional high-value findings remain.
