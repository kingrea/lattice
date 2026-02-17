# OpenCode Audit Runtime

This folder contains the project-local audit runtime used by the OpenCode workflow.

Contents:

- `agents/`: orchestrator, investigators, and scribe roles
- `commands/`: command entrypoints (`/audit-start`, `/audit-next-loop`, `/audit-status`, `/audit-close`)
- `skills/`: plan, loop, mailbox, reporting, status, and closeout skills
- `audits/`: per-run folders (`<run-id>`) and schema/acceptance references

This runtime is built to execute a full audit in one OpenCode session with beads (`bd`) as source of truth.
