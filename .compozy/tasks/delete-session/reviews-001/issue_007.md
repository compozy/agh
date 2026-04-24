---
status: resolved
file: internal/daemon/daemon_test.go
line: 4134
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151198531,nitpick_hash:00f8e2d887e2
review_hash: 00f8e2d887e2
source_review_id: "4151198531"
source_review_submitted_at: "2026-04-21T23:03:23Z"
---

# Issue 007: Avoid aliasing delete behavior to stop in the session test double.
## Review Comment

`Delete()` currently calls `Stop()`, which can hide regressions now that delete and stop are distinct operations. Consider tracking `Delete` calls independently (and optionally separate delete-specific errors/state mutation) so tests can assert the right path was used.

## Triage

- Decision: `valid`
- Notes:
  In `internal/daemon/daemon_test.go`, the fake session manager currently implements `Delete()` by delegating to `Stop()`, which means delete-path assertions would silently exercise stop bookkeeping instead of delete-specific behavior. I will split delete tracking from stop tracking in the fake manager so future tests can distinguish the two operations.
