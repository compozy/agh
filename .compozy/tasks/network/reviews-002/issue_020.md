---
status: resolved
file: internal/network/router.go
line: 515
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857291,nitpick_hash:125c6ad03827
review_hash: 125c6ad03827
source_review_id: "4093857291"
source_review_submitted_at: "2026-04-11T14:15:44Z"
---

# Issue 020: The seen map is bounded by the maxReplayAge window, not truly unbounded.
## Review Comment

The cleanup loop executes on every `markSeen` call (once per inbound message), not periodically. Map size is naturally bounded to the number of unique message IDs that can arrive within the `maxReplayAge` window (default 5 minutes per RFC). Each entry is deleted as soon as it expires, making this a standard bounded-cache pattern.

If you want to be explicit about the upper bound or add monitoring, consider tracking the map size in metrics or documenting the expected bounds in a comment based on your typical message rate.

## Triage

- Decision: `invalid`
- Notes: The review comment itself describes why this is not a defect: `markSeen` evicts expired entries on every insert, so the map is bounded by the replay window rather than growing without bound. There is no behavior regression to correct in `router.go`.
