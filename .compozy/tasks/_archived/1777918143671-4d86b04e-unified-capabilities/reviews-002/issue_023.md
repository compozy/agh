---
status: resolved
file: internal/session/liveness.go
line: 124
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:475a01ed67b5
review_hash: 475a01ed67b5
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 023: Consider removing unnecessary cloning in equality check.
## Review Comment

`CloneSessionLivenessMeta` creates copies but only field reads follow—no mutation occurs. The cloning adds allocation overhead without benefit.

## Triage

- Decision: `valid`
- Notes:
  `sessionLivenessEqual` clones both liveness payloads only to normalize fields before read-only comparison. The comparison can normalize strings and timestamps inline, so the current approach adds avoidable allocations on a hot metadata path.
  I will remove the cloning and keep the same semantic comparison by normalizing values during field checks.
  Fixed and verified with targeted package tests plus `make verify`.
