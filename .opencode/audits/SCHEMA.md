# Audit Run State Schemas

Each run lives at `.opencode/audits/<run-id>/`.

## `state/plan.json`

Required shape:

```json
{
  "run_id": "run-20260217-001",
  "target": "internal/tui",
  "audit_type": "security",
  "intensity": 3,
  "assignments": [
    {
      "investigator": "investigator-alpha",
      "role": "security specialist",
      "active": true
    }
  ],
  "focus_areas": ["input validation", "authz"],
  "existing_beads": ["sec-1"]
}
```

## `state/loop-status.json`

Required shape:

```json
{
  "current_loop": 0,
  "max_loops": 3,
  "early_exit": false,
  "investigators": {
    "investigator-alpha": {
      "status": "ACTIVE",
      "last_summary": ""
    },
    "investigator-bravo": {
      "status": "IDLE",
      "last_summary": ""
    },
    "investigator-charlie": {
      "status": "IDLE",
      "last_summary": ""
    }
  }
}
```

Allowed investigator statuses: `ACTIVE`, `FINDINGS`, `NOTHING_MORE`, `IDLE`.

## `state/findings-map.json`

Required shape:

```json
{
  "findings": [
    {
      "bead_id": "sec-42",
      "role": "security specialist",
      "investigator": "investigator-alpha",
      "loop": 1,
      "refs": ["internal/tui/launch.go:77"],
      "disposition": "open"
    }
  ]
}
```

Allowed disposition values: `open`, `duplicate_closed`, `out_of_scope_closed`, `updated_existing`.

## Validation and Recovery

If any required key is missing or type-mismatched:

1. Stop loop execution.
2. Print the exact missing/invalid key path.
3. Print a minimal valid JSON snippet for repair.
4. Resume only after validation succeeds.
