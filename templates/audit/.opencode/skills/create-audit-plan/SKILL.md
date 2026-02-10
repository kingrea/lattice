---
name: create-audit-plan
description: Reads the audit task, assigns roles to investigators, and sets up the audit scope before the first loop begins.
---

# Create Audit Plan

You are setting up the audit before any investigation begins.

## Process

1. Read `context/TASK.md` to extract:
   - **Target**: What part of the codebase to audit
   - **Roles**: The 1–3 role perspectives (e.g., senior engineer, security specialist)
   - **Focus areas**: What to look for

2. Read `.team` to get the `intensity` value (max loops).

3. Check existing beads to understand what's already tracked:
   ```bash
   bd list
   ```

4. Assign roles to investigators:

   | Roles provided | Assignment |
   |---|---|
   | 1 role | `investigator-alpha` gets the role. Bravo and charlie are idle. |
   | 2 roles | Alpha gets role 1, bravo gets role 2. Charlie is idle. |
   | 3 roles | Alpha gets role 1, bravo gets role 2, charlie gets role 3. |

5. For each active investigator, note:
   - Their assigned role
   - The target area (same for all)
   - The focus areas (same for all)
   - The existing beads they should be aware of

## Output

Produce a plan summary:

```
## Audit Plan

### Target
<target area>

### Intensity
<N loops max>

### Existing Beads
<count and summary of relevant existing beads, or "None">

### Assignments

| Investigator | Role | Status |
|---|---|---|
| investigator-alpha | <role> | Active |
| investigator-bravo | <role or "Idle"> | Active/Idle |
| investigator-charlie | <role or "Idle"> | Active/Idle |

### Focus Areas
<bulleted list from task>
```

Then proceed to spawn investigators using the `investigator-prompt` skill for each active investigator.
