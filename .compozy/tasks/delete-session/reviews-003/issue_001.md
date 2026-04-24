---
status: resolved
file: internal/session/manager_delete_test.go
line: 13
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167388414,nitpick_hash:1bb5ecf448a0
review_hash: 1bb5ecf448a0
source_review_id: "4167388414"
source_review_submitted_at: "2026-04-24T02:02:29Z"
---

# Issue 001: LGTM! Consider adding t.Parallel() for faster test execution.
## Review Comment

The table-driven test structure follows the `t.Run("Should...")` pattern and covers the key scenarios: stopped session removal, active session stop-before-delete, concurrent stop race handling, and error wrapping verification.

Each subtest creates an isolated harness, so they can run in parallel:

As per coding guidelines, "Add `t.Parallel()` to independent subtests in Go tests".

## Triage

- Decision: `valid`
- Root cause: `TestManagerDelete` uses a table-driven `t.Run(...)` loop but does not mark the independent subtests as parallelizable, so the suite misses the concurrency the workspace Go testing guidelines expect.
- Evidence: each case either creates its own `newHarness(t)` with `t.TempDir()`-backed state or exercises the pure `stopSessionBeforeDelete(...)` helper without shared mutable fixtures, so there is no inter-test coupling that would block `t.Parallel()`.
- Fix approach: add `t.Parallel()` to the parent test and each subtest closure so the suite can execute concurrently while preserving isolated setup and assertions.

## Resolution

- Added `t.Parallel()` to `TestManagerDelete` and to each table-driven subtest in `internal/session/manager_delete_test.go`.
- Kept the existing harness-per-case structure intact, so the change improves execution concurrency without changing session-delete behavior.
- Verified with `go test ./internal/session ./internal/task` and `make verify` (both exit `0`).
