---
name: compile-report
description: Compiles all audit findings into a structured final report at context/REPORT.md.
---

# Compile Audit Report

You are producing the final audit report.

## Process

1. Read all beads and their comments:
   ```bash
   bd list
   ```
   Then for each bead:
   ```bash
   bd show <bead-id>
   ```

2. Read `.team` for loop count and intensity.

3. Read `context/TASK.md` for the original audit scope.

4. Write the report to `context/REPORT.md`.

## Report Structure

```markdown
# Audit Report: <target area>

## Overview

| | |
|---|---|
| **Target** | <what was audited> |
| **Roles** | <roles used, comma-separated> |
| **Focus Areas** | <focus areas, comma-separated> |
| **Intensity** | <max loops configured> |
| **Loops Completed** | <actual loops before stopping> |
| **Total Findings** | <count of beads created> |

## Summary

<3-5 sentences. What was the overall outcome? Was the code clean, problematic, or mixed? Which role perspective surfaced the most issues?>

## Findings by Severity

### Critical (P0)
<list each finding with bead ID, title, location, and one-line impact — or "None">

### High (P1)
<same format — or "None">

### Medium (P2)
<same format — or "None">

### Low (P3)
<same format — or "None">

## Findings by Role

### <Role 1>
- Investigator: <alpha|bravo|charlie>
- Loops completed: <N before stopping or reaching intensity>
- Findings: <count>
- <one-line summary of what this perspective surfaced>

### <Role 2> (if applicable)
<same format>

### <Role 3> (if applicable)
<same format>

## Existing Beads Updated

<list any pre-existing beads that were updated with new findings, or "None">

## Audit Process Notes

- <any notable observations about the audit itself — e.g., "investigator-bravo stopped after loop 1 with no findings from a staff engineer perspective">
- <if duplicates were caught and closed, note that>
- <if the commissar closed any out-of-scope findings, note that>

## Recommendations

<if appropriate, 2-3 high-level recommendations based on the pattern of findings — e.g., "Multiple P1 findings in auth validation suggest a systematic review of all auth endpoints is warranted">
```

## Rules

- Every finding must reference its bead ID so it's traceable.
- If no findings were made, the report should say so clearly. "No actionable issues found" is a useful result.
- Group and organize for readability. The audience is humans who will prioritize work from this report.
- Do not add findings that aren't backed by a bead. The report summarizes beads, it doesn't introduce new issues.
- Keep language neutral and factual. No dramatizing, no minimizing.
