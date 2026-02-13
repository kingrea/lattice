---
name: grunt-prompt
description: Generates a self-contained, structured prompt for spawning a grunt subagent to complete a single work bead.
---

# Generate Grunt Prompt

You are generating a prompt to hand to a grunt subagent. Grunts follow instructions precisely but do not think creatively or make architectural decisions. The prompt must be completely self-contained — the grunt has zero context beyond what you provide.

## Process

1. Read the target bead:
   ```bash
   bd show <bead-id>
   ```

2. If this is a sub-task, also read the parent bead for context:
   ```bash
   bd show <parent-bead-id>
   ```

3. Read all source files mentioned in the bead's file list.

4. Identify existing code patterns, naming conventions, and style from those files.

5. Generate the prompt using the template below.

## Prompt Template

Output the following prompt exactly, filling in each section:

---

**BEAD: `<bead-id>`**

**ASSIGNED TO: `<grunt-name>`**

## Your Task

<Step-by-step instructions. Number each step. Be specific about what to create, modify, or delete. Include exact function signatures, component names, CSS classes — whatever applies.>

## Context

<Background information the grunt needs to understand WHY this work exists. Include relevant code snippets from existing files showing patterns they must follow. Keep this focused — only include what's necessary to do the work.>

## Files

| Action | Path | What to do |
|--------|------|------------|
| create/modify | `<path>` | `<specific changes>` |

## Acceptance Criteria

<The specific criteria from the bead. List each one with a checkbox.>

- [ ] Criteria 1
- [ ] Criteria 2

## Rules

- Do ONLY what is described above. Nothing more, nothing less.
- Do NOT refactor, clean up, or "improve" surrounding code.
- Do NOT add features, utilities, or abstractions not specified.
- Do NOT modify files not listed in the Files table.
- Follow existing code patterns and conventions exactly as shown in Context.
- If something is unclear or you encounter an unexpected blocker, STOP and report back. Do not guess.

## When Done

1. Verify your changes compile/lint without errors.
2. Run any tests specified in the acceptance criteria.
3. Report: what you did, what files changed, any issues encountered.

---

## Guidelines for Prompt Quality

- Include actual code snippets from the codebase, not pseudocode.
- If the bead has child beads from previous rejections, include that feedback so the grunt does not repeat the same mistakes.
- Specify exact naming conventions (camelCase, kebab-case, etc.) from the existing codebase.
- If the bead depends on completed beads, summarize what those beads produced so the grunt knows the current state of the code.
