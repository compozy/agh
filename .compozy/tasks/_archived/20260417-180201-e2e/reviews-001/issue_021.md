---
status: resolved
file: internal/testutil/acpmock/driver_test.go
line: 17
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:518b5d43e491
review_hash: 518b5d43e491
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 021: Add t.Parallel() to enable parallel test execution.
## Review Comment

The test functions are missing `t.Parallel()` calls. Per coding guidelines, independent tests should run in parallel.

## Triage

- Decision: `valid`
- Notes:
  The file has two independent tests that can safely run in parallel. The
  network-origin test must remain serial because it uses `t.Setenv`, but
  `TestDriverStreamsStablePermissionAndToolSequence` and
  `TestDriverAdvertisesAndSupportsLoadSession` do not mutate process-global
  state and should opt into `t.Parallel()`.

## Resolution

- Added `t.Parallel()` to the two safe top-level tests and kept the
  `t.Setenv`-based case serial.
