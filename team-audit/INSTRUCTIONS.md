# Agent Instructions

## Issue Tracking

This project uses **bd (beads)** for issue tracking. Run `bd prime` for workflow
context, or install hooks (`bd hooks install`) for auto-injection.

**Quick reference:**

- `bd ready` - Find available work
- `bd list` - List all issues
- `bd show <id>` - View issue details
- `bd create "<title>" -p <priority>` - Create an issue
- `bd comment <id> "<text>"` - Add a comment
- `bd update <id> --status <status>` - Update status
- `bd close <id>` - Complete work
- `bd sync` - Sync with git

For full workflow details: `bd prime`

## Duplicate Prevention

Before creating any bead, ALWAYS search existing beads first:

```bash
bd list
```

If an existing bead covers the same issue:
- Update its description with your additional findings via `bd comment`
- Do NOT create a duplicate

## Landing the Plane (Session Completion)

**When the commissar ends a work session**, you MUST complete ALL steps below.

**MANDATORY WORKFLOW:**

1. **Verify all findings are tracked** - Every actionable finding has a bead
2. **Verify no duplicates** - Run `bd list` and check
3. **Spawn the scribe** - Produce the final audit report
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Verify** - All beads synced and pushed

**CRITICAL RULES:**

- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing
- If push fails, resolve and retry until it succeeds
