---
status: resolved
file: internal/procutil/process_group_unix_test.go
line: 16
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167458928,nitpick_hash:f365cd44b7c5
review_hash: f365cd44b7c5
source_review_id: "4167458928"
source_review_submitted_at: "2026-04-24T02:26:11Z"
---

# Issue 007: Add one subtest for the dual-error join path.
## Review Comment

Please cover the branch where both `signalErr` (non-`EPERM`) and `waitErr` are non-nil, and assert that `errors.Is` matches both targets.

As per coding guidelines, "Must Check: Focus on critical paths: workflow execution, state management, error handling" and "MUST have specific error assertions (ErrorContains, ErrorAs)".

## Triage

- Decision: `valid`
- Notes:
- `joinProcessGroupKillResult` has an untested branch where both a non-`EPERM` signal error and a wait error are present and combined.
- Adding one subtest that asserts `errors.Is` matches both wrapped causes is the right regression coverage for the multi-error path.
- Resolved by adding the dual-error join case in `internal/procutil/process_group_unix_test.go` and verifying it in focused tests plus `make verify`.
