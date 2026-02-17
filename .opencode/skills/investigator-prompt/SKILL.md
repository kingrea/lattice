---
name: investigator-prompt
description: Generate loop-1 prompt for an investigator with role, scope, and dedupe context.
---

# Investigator Prompt (Loop 1)

Build a prompt containing:

- investigator id and assigned role
- target and focus areas
- known existing beads (`bd list` summary)
- strict dedupe/scope rules

Required rules in prompt:

- Run `bd list` before creating a bead.
- Update existing bead when duplicate.
- Provide file references and impact.
- Return status as `FINDINGS` or `NOTHING_MORE`.
