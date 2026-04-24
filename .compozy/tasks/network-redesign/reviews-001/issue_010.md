---
status: resolved
file: internal/store/globaldb/global_db.go
line: 143
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151559901,nitpick_hash:b9f10ce38600
review_hash: b9f10ce38600
source_review_id: "4151559901"
source_review_submitted_at: "2026-04-22T01:22:21Z"
---

# Issue 010: Use a composite index for workspace channel listings.
## Review Comment

Two single-column indexes do not help much for the common `WHERE workspace_id = ? ORDER BY updated_at DESC` path. A composite index on workspace and activity time will serve the new room-list query much better.

## Triage

- Decision: `valid`
- Reasoning: the common channel-list query filters by `workspace_id` and sorts by `updated_at DESC, channel ASC`, but the schema only provides single-column indexes. A composite index better matches that access path and avoids extra sorting work for workspace-scoped listings.
- Fix plan: add a composite workspace/activity index to the schema and assert its presence in the channel schema test.
- Resolution: added the composite `network_channels(workspace_id, updated_at DESC, channel ASC)` index and asserted its presence in the schema test.
- Verification: `go test ./internal/store/globaldb` and `make verify`
