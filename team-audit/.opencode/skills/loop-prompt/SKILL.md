---
name: loop-prompt
description: Generates a follow-up audit prompt for subsequent loops, instructing the investigator to go deeper without duplicating previous findings.
---

# Generate Loop Prompt

You are generating a prompt for an investigator's Nth audit pass (loop 2+). This prompt pushes them to look deeper while avoiding duplication.

## Process

1. Note the current loop number from `.team` (`current_loop`).

2. List all beads created so far (by all investigators, not just this one):
   ```bash
   bd list
   ```

3. Summarize what this specific investigator found in their previous pass(es).

4. Generate the prompt using the template below.

## Prompt Template

---

**AUDIT PASS: Loop <N>**

**INVESTIGATOR: `<investigator name>`**

## Your Role

You are still auditing as a **<role>**.

## What's Been Found So Far

These beads have already been created by the audit team (including by you and other investigators). You MUST NOT duplicate any of these:

<list each bead ID, title, and a one-line summary>

## Your Previous Findings

In your last pass you reported:
<summary of their previous response — findings or NOTHING_MORE>

## Your Task Now

Go deeper. Look for things you missed on the first pass:

- **Subtler issues**: Race conditions, implicit assumptions, edge cases that only surface under load or unusual input.
- **Interaction effects**: How does this code interact with other parts of the system? Are there coupling issues?
- **Things that look fine but aren't**: Code that works today but is fragile — magic numbers, hardcoded values, missing error paths.
- **What's NOT there**: Missing validation, missing logging, missing tests, missing error handling that should exist.

But — if the area is genuinely clean from your role's perspective, say so. Do not invent findings.

## Target

Same as before: **<target area>**

## Focus Areas

<same focus areas as loop 1>

## Rules

- Do NOT duplicate any beads listed above.
- Do NOT repeat findings from your previous pass.
- If you genuinely cannot find anything new, report `Status: NOTHING_MORE`. This is fine.
- Same quality bar: only real issues with real impact.

---

## Guidelines

- Each successive loop should look at a deeper layer. Loop 1 catches the obvious. Loop 2 catches the subtle. Loop 3 catches the systemic.
- If the investigator reported `NOTHING_MORE` on a previous loop, do NOT generate a loop prompt for them. They are done.
- If the investigator's previous findings were borderline (very minor issues), consider whether another loop is worth it. Sometimes stopping early is the right call.
