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
| **Role** | <role title> |
| **Focus Areas** | <focus areas, comma-separated> |
| **Intensity** | <max loops configured> |
| **Loops Completed** | <actual loops before stopping> |
| **Total Findings** | <count of beads created> |

## Summary

<3-5 sentences. What was the overall outcome? Was the code clean, problematic, or mixed?>

## Findings by Severity

### Critical (P0)
<list each finding with bead ID, title, location, and one-line impact — or "None">

### High (P1)
<same format — or "None">

### Medium (P2)
<same format — or "None">

### Low (P3)
<same format — or "None">

## Findings by Area

<group findings into practical code or subsystem areas>

## Existing Beads Updated

<list any pre-existing beads that were updated with new findings, or "None">

## Audit Process Notes

- <any notable observations about the audit itself>
- <if duplicates were caught and consolidated, note that>
- <if the session exited early because nothing new remained, note that>

## Recommendations

<if appropriate, 2-3 high-level recommendations based on the pattern of findings>
```

## Rules

- Every finding must reference its bead ID so it is traceable.
- If no findings were made, the report should say so clearly. "No actionable issues found" is a useful result.
- Group and organize for readability. The audience is humans who will prioritize work from this report.
- Do not add findings that are not backed by a bead. The report summarizes beads; it does not introduce new issues.
- Keep language neutral and factual. No dramatizing, no minimizing.
