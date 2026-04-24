---
status: resolved
file: internal/api/httpapi/helpers_test.go
line: 421
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151161262,nitpick_hash:50d2a42d9b89
review_hash: 50d2a42d9b89
source_review_id: "4151161262"
source_review_submitted_at: "2026-04-21T22:49:44Z"
---

# Issue 001: Centralize testConfigWithDisabledNetwork.
## Review Comment

This helper is now duplicated in `internal/api/httpapi/helpers_test.go`, `internal/api/udsapi/helpers_test.go`, and `internal/api/core/test_helpers_test.go`. Moving it into shared testutil would reduce drift the next time config defaults change.

As per coding guidelines, `**/*_test.go`: "Check for shared test utilities usage to avoid duplication".

## Triage

- Decision: `VALID`
- Notes:
  - `testConfigWithDisabledNetwork` is currently duplicated in all three cited test helper files, and each copy does the same `aghconfig.DefaultWithHome(homePaths)` plus `cfg.Network.Enabled = false` mutation.
  - Root cause: the network-default test additions reused a local helper pattern instead of extracting the shared API-test fixture into `internal/api/testutil`.
  - Fix approach: add one shared helper in `internal/api/testutil` and update the three test helper files to call it. This requires minimal out-of-scope edits to `internal/api/udsapi/helpers_test.go`, `internal/api/core/test_helpers_test.go`, and `internal/api/testutil/apitest.go` so the duplication is actually removed rather than moved.
  - Implemented: `internal/api/testutil.ConfigWithDisabledNetwork(...)` now owns the config construction and the three package-local helpers delegate to it.
  - Verified: focused `go test ./internal/api/httpapi ./internal/api/udsapi ./internal/api/core ./internal/config ./internal/daemon ./internal/testutil/e2e` passed, then `make verify` passed.
