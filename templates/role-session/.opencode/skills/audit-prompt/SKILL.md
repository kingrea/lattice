---
name: audit-prompt
description: Generates the initial single-role audit prompt with role guidance, scope, and duplicate-avoidance context.
---

# Generate Audit Prompt

You are generating the prompt for loop 1 of a single-role audit session.

## Process

1. Read `context/TASK.md` to extract:
   - Role title and role guidance
   - Target area
   - Focus areas

2. List existing beads so duplication is avoided:
   ```bash
   bd list
   ```

3. Generate a prompt using this template.

## Prompt Template

---

**AUDIT PASS: Loop 1**

## Your Role

You are auditing as **<role title>**.

Role guidance:
<role guidance from task>

## Target

Examine: **<target area / file paths / section of the codebase>**

## Focus Areas

<bulleted list of focus areas from the task>

## Existing Beads

These issues are already tracked. Do NOT create duplicates. If your finding overlaps, update the existing bead instead.

<list of existing bead IDs and titles, or "None — this is a fresh audit.">

## Rules

- Only raise issues with real impact.
- Do not raise issues outside the focus areas listed above.
- If you find nothing, that is a valid result. Say so.
- Check `bd list` before creating any bead.
- Include specific file paths, line numbers, and code references in every finding.

---
