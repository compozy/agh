---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/api/core/tasks.go
line: 337
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:04fe79bf2745
review_hash: 04fe79bf2745
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 002: Strip server-managed fields from execution-profile writes.
## Review Comment

This endpoint binds directly into `taskpkg.ExecutionProfile`, and the helper only normalizes `TaskID`. The manager later preserves a non-zero `CreatedAt`, so callers can backdate or otherwise forge immutable profile metadata through the public API. Please switch to a dedicated request DTO, or at minimum clear the server-owned fields here before handing the object off.

Also applies to: 1537-1555

## Triage

- Decision: `valid`
- Notes:
  - `SetTaskExecutionProfileRequest` is still an alias of `task.ExecutionProfile`, so the request shape includes `created_at` and `updated_at`.
  - `taskExecutionProfileFromRequest` only normalizes `task_id` and returns the request object directly, leaving server-managed timestamps writable from the public API surface.
  - Planned fix: scrub server-managed profile fields at the handler boundary before calling task service authority, and cover the behavior with an API regression test.
  - Resolved: the write path now copies request data into a fresh profile, overwrites `TaskID` from the URL path, clears `CreatedAt`/`UpdatedAt`, and API tests assert the sanitized payload.
