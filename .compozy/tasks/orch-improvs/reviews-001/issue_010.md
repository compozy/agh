---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/daemon/native_profile_tools.go
line: 66
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:b5d79ae0935c
review_hash: b5d79ae0935c
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 010: Reject conflicting task_id values before persisting the profile.
## Review Comment

`taskExecutionProfileSet()` keys the write by the top-level `task_id`, but `profile()` will prefer `profile.task_id` when it is set. A caller can submit two different IDs here and produce a stored payload that no longer matches the record being updated.

Also applies to: 111-123

## Triage

- Decision: `valid`
- Notes:
  - `taskExecutionProfileSet()` keys writes by top-level `task_id`, but `taskExecutionProfileInput.profile()` still prefers an embedded `profile.task_id` when present.
  - That allows a mismatched payload to persist a profile whose body no longer matches the addressed task record.
  - Planned fix: reject conflicting `task_id` values before materializing the profile payload and add a native-tool regression test.
  - Resolved: the native profile tool now rejects conflicting top-level and nested task IDs before persistence, returns a schema-invalid tool error, and tests confirm no write occurs.
