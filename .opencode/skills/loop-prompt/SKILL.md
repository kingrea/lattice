---
name: loop-prompt
description: Generate deeper-loop prompt that avoids duplicate or repeated findings.
---

# Loop Prompt (Loop 2+)

Build a follow-up prompt with:

- current loop number
- all currently tracked beads
- that investigator's prior findings/status
- remaining focus guidance

Required constraints:

- no duplicate beads
- no repeated prior findings
- return `NOTHING_MORE` when appropriate
- maintain the same target and scope boundaries
