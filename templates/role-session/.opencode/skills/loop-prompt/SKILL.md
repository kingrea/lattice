---
name: loop-prompt
description: Generates a deeper follow-up prompt for later loops in a single-role session without duplicating prior findings.
---

# Generate Loop Prompt

You are generating a prompt for loop 2+ in a single-role audit session.

## Process

1. Read `.team` to get `current_loop`.

2. Read all beads already created or updated:
   ```bash
   bd list
   ```

3. Summarize what has already been found in prior loops.

4. Generate the prompt using this template.

## Prompt Template

---

**AUDIT PASS: Loop <N>**

## Your Role

Continue auditing as **<role title>**.

## What's Been Found So Far

These beads are already tracked. You MUST NOT duplicate them:

<list each bead ID, title, and one-line summary>

## Your Previous Findings

In your last pass you reported:
<summary of previous findings or that nothing new remained>

## Your Task Now

Go deeper and look for what may have been missed:

- Subtle edge cases and hidden assumptions
- Interaction effects across modules
- Fragility that appears correct today but is risky under change
- Missing validation, error paths, logging, or tests where they should exist

If the area is genuinely clean from this role perspective, say so. Do not invent findings.

## Target

Same as before: **<target area>**

## Focus Areas

<same focus areas as loop 1>

## Rules

- Do NOT duplicate any beads listed above.
- Do NOT repeat findings from previous loops.
- If you cannot find anything new, report that directly.
- Keep the same quality bar: only real issues with real impact.

---
