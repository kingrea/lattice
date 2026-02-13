---
name: commissar-feedback
description: Provides structured, actionable feedback when rejecting work that failed quality review, and creates child beads to track required fixes.
---

# Commissar Feedback

You are rejecting work that did not pass the inquisitor's quality review. Your feedback must be specific and actionable enough that a grunt — who does not think creatively — can fix the issues without ambiguity.

## Process

1. Review the inquisitor's findings for the bead.

2. Categorize each issue by severity:
   - **Critical** — Breaks functionality, fails tests, or violates task requirements. Must fix.
   - **Major** — Significant quality issue (wrong pattern, missing edge case, accessibility failure). Must fix.
   - **Minor** — Style, convention, or cosmetic issue. Fix in this pass if simple, otherwise note for later.

3. For each critical or major issue, create a child bead:
   ```bash
   bd create "<concise fix description>" -p <priority>
   bd dep add <parent-bead-id> <child-bead-id>
   ```

4. Add structured feedback as a comment on each child bead:
   ```bash
   bd comment <child-bead-id> "<feedback using the format below>"
   ```

5. Block the parent bead until children are resolved:
   ```bash
   bd update <parent-bead-id> --status blocked
   ```

## Feedback Format

For each issue, provide:

```
### Issue: <short descriptive title>

- **Severity**: Critical | Major | Minor
- **Location**: <file path>:<line number(s)>
- **Problem**: <What is wrong. Be specific — reference actual code.>
- **Expected**: <What it should look like or do instead.>
- **Fix**: <Exact steps to resolve. Include code snippets showing the correct implementation if possible.>
```

## Rules

- NEVER give vague feedback ("improve quality", "make it better", "needs work").
- ALWAYS reference specific files, line numbers, and code.
- Include code snippets showing the expected fix when the correct approach isn't obvious.
- Group related issues into a single child bead (e.g., "the same null check is missing in 3 places" = one bead).
- Minor issues that don't warrant their own bead can be listed as a comment on the parent bead.
- If the same mistake recurs from a previous rejection round, flag the pattern explicitly so the grunt addresses ALL instances.

## After Providing Feedback

Once all child beads are created, re-prompt the grunt using the `grunt-prompt` skill. The prompt must include:
- The parent bead context
- All new child bead requirements
- The specific feedback from the inquisitor
- What went wrong last time, so the grunt does not repeat it

Track the retry count. If a bead has been rejected 3+ times for the same issue, escalate to the user — something systemic may be wrong.
