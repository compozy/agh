---
status: resolved
file: internal/testutil/e2e/runtime_harness_test.go
line: 35
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151161262,nitpick_hash:c37851268e75
review_hash: c37851268e75
source_review_id: "4151161262"
source_review_submitted_at: "2026-04-21T22:49:44Z"
---

# Issue 004: Split the three network scenarios into subtests.
## Review Comment

This packs three distinct behaviors into one test body, so the first failure hides the rest and the cases cannot be parallelized independently. A small table with `t.Run("Should ...")` would fit the suite better.

As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "Add `t.Parallel()` to independent subtests in Go tests".

## Triage

- Decision: `VALID`
- Notes:
  - `TestPrepareRuntimeLayoutUsesEnabledNetworkByDefaultAndAllowsExplicitDisable` currently checks three distinct behaviors in one body: default enablement, explicit disablement, and `EnableNetwork` overriding a disabled seed.
  - Root cause: the test was written as one linear scenario instead of the repo-default subtest structure, which hides later failures and blocks independent parallelization.
  - Fix approach: rewrite the test as a small table-driven suite with `Should...` subtests and keep the current network expectations per scenario.
  - Implemented: the three scenarios now run as parallel `Should...` subtests with the same enabled/disabled assertions.
  - Verified: focused `go test ./internal/api/httpapi ./internal/api/udsapi ./internal/api/core ./internal/config ./internal/daemon ./internal/testutil/e2e` passed, then `make verify` passed.
