---
name: investigator-prompt
description: Generates the initial audit prompt for an investigator, including their assigned role, target area, focus areas, and existing beads to avoid duplicating.
---

# Generate Investigator Prompt

You are generating a prompt for an investigator's first audit pass (loop 1).

## Process

1. Gather the investigator's assignment from the audit plan:
   - Which investigator (alpha, bravo, charlie)
   - Their assigned role
   - The target area
   - The focus areas

2. List existing beads so the investigator knows what's already tracked:
   ```bash
   bd list
   ```

3. Generate the prompt using the template below.

## Prompt Template

---

**AUDIT PASS: Loop 1**

**INVESTIGATOR: `<investigator name>`**

## Your Role

You are auditing as a **<role>** (e.g., senior engineer, security specialist).

Think from this perspective. What would a <role> flag when reviewing this code? What concerns, risks, or issues would they raise?

<Include 2-3 sentences of role-specific guidance:>
- For "senior engineer": Focus on architecture, maintainability, abstractions, tech debt, scalability, code clarity, error handling patterns.
- For "staff engineer": Focus on systemic issues, cross-cutting concerns, observability gaps, operational risks, inconsistent patterns, missing documentation for critical paths.
- For "security specialist": Focus on injection vectors, auth/authz gaps, data exposure, input validation, insecure defaults, dependency vulnerabilities, OWASP top 10.
- For other roles: Derive appropriate focus from the role title.

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
