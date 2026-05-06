---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/store/globaldb/global_db_task_profile.go
line: 72
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:39f0cba1110f
review_hash: 39f0cba1110f
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 028: Always bump updated_at on a successful upsert.
## Review Comment

A normal read-modify-write flow reuses the previously loaded timestamp here, so profile changes can be persisted without changing `updated_at`, which breaks change detection and ordering for consumers.

## Triage

- Decision: `valid`
- Notes: `UpsertExecutionProfile` preserves `CreatedAt`, but it only sets `UpdatedAt` when the incoming payload leaves it zero. A normal load-modify-save flow can therefore write a changed profile back with its stale `UpdatedAt`, breaking change detection and ordering. Fix by always stamping `UpdatedAt` with `g.now()` on successful upsert while preserving the original `CreatedAt`.
- Resolution: `UpsertExecutionProfile` now always stamps `UpdatedAt` with the current store time, and the profile-store test covers both fresh updates and stale reload/save flows.
