# End-to-End Acceptance Matrix

Use this matrix for a full run verification.

## Flow

1. Run `/audit-start`.
2. Run `/audit-next-loop` until early-exit or max intensity.
3. Run `/audit-status` after each loop.
4. Run `/audit-close`.

## Required Checks

- Start created required run folder/files.
- State files validate against `.opencode/audits/SCHEMA.md`.
- Loop increments exactly once per global loop.
- Early-exit triggers when all investigators report `NOTHING_MORE`.
- Mailbox contains investigator and orchestrator handoff messages.
- Duplicate finding test updates existing bead instead of creating duplicate.
- Out-of-scope test is closed with rationale.
- Final report is bead-backed and references bead ids.
- Completion gate blocks `.team status=complete` until checklist passes.

## Evidence Template

For each check, capture:

- `Result`: PASS or FAIL
- `Evidence`: command output snippet or file reference
- `Notes`: remediation if failed
