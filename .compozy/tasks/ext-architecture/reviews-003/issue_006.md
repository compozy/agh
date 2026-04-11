---
status: resolved
file: internal/api/httpapi/handlers_test.go
line: 295
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093012644,nitpick_hash:17818a4108bc
review_hash: 17818a4108bc
source_review_id: "4093012644"
source_review_submitted_at: "2026-04-11T00:50:56Z"
---

# Issue 006: Wrap this test in t.Run() and add t.Parallel().
## Review Comment

This test is independent and should follow the repository's Go test conventions: all test cases must use the `t.Run("Should...")` pattern with `t.Parallel()` for independent subtests, as shown in other tests in this file (e.g., lines 787–839).

## Triage

- Decision: `valid`
- Notes:
  - This is a consistency-only test cleanup, but it is low risk and aligns the daemon-status handler test with the repo’s `Should...` subtest style used broadly in Go tests.
  - Root cause: the test was written as a standalone top-level test even though it is independent and can safely run as a parallel subtest.
  - Fix plan: wrap the test body in `t.Run("Should...")` and add `t.Parallel()` inside that subtest while preserving the existing assertions.
  - Implemented: wrapped the daemon-status handler test in `t.Run("ShouldReturnResolvedUserHomeDir", ...)` and added `t.Parallel()` inside the subtest without changing the assertions.
  - Verification: `go test ./cmd/agh-codegen ./internal/acp ./internal/api/core ./internal/api/httpapi ./internal/api/spec -count=1`; `make verify`.
