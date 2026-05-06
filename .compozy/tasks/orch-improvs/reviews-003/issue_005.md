---
provider: coderabbit
pr: "106"
round: 3
round_created_at: 2026-05-06T06:28:14.497092Z
status: resolved
file: internal/session/query_test.go
line: 557
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4233729248,nitpick_hash:3cd4282f4f0e
review_hash: 3cd4282f4f0e
source_review_id: "4233729248"
source_review_submitted_at: "2026-05-06T06:27:47Z"
---

# Issue 005: Add t.Parallel() to the new subtest.
## Review Comment

This new case does not opt into parallel execution, which diverges from the repository’s default test convention.

As per coding guidelines, "Default to `t.Parallel` in Go tests unless there is a specific reason to disable it (opt-out with `t.Setenv`)".

## Triage

- Decision: `valid`
- Root cause analysis: the new subtest `Should wait for finalization before reading a closed recorder handle` does not call `t.Parallel()` even though it does not use `t.Setenv` and operates on isolated harness state.
- Why this is valid: AGH test conventions require parallel execution by default for independent subtests, and this one is the new exception in the touched block.
- Fix approach: add `t.Parallel()` at the subtest start while preserving the existing synchronization logic.

## Resolution

- Added `t.Parallel()` to the affected `internal/session/query_test.go` subtest.

## Verification

- Focused regression: `go test ./internal/session -run 'TestOpenQueryRecorder|TestReadMetaAndQueryHelpers' -count=1 -race`
- Fresh full gate: `make verify` exited `0` in this session.
