---
status: resolved
file: internal/extension/host_api.go
line: 428
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:8108c8f8d412
review_hash: 8108c8f8d412
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 015: Use the shared Host API method constants here.
## Review Comment

These new dispatch entries add more raw method strings to a surface that already has a protocol/capability contract. Reusing the shared constants would make routing drift much harder and avoid silent `method not found` mismatches when one side is renamed.

## Triage

- Decision: `valid`
- Notes: Confirmed. `hostAPIMethodHandlers` uses raw string literals for the new task methods even though the canonical Host API method constants already exist. That invites routing drift between the handler registry and the protocol contract. I’ll switch these entries to the shared constants.
