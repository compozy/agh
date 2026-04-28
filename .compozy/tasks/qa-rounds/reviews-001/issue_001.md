---
status: resolved
file: internal/api/core/coverage_helpers_test.go
line: 194
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4188693296,nitpick_hash:0cc472a0a94d
review_hash: 0cc472a0a94d
source_review_id: "4188693296"
source_review_submitted_at: "2026-04-28T12:24:35Z"
---

# Issue 001: Add t.Parallel() to the new top-level sort tests.
## Review Comment

Both new top-level tests are independent and read-only, but unlike the rest of this file they run serially. That drifts from the package’s normal test shape for no clear benefit.

As per coding guidelines, "`**/*_test.go`: Use `t.Run('Should ...')` subtests with `t.Parallel` as default (opt-out only with `t.Setenv`) in Go test files".

Also applies to: 290-354

## Triage

- Decision: `VALID`
- Notes: `TestSortedNetworkChannelPayloads` and `TestSortedNetworkPeerPayloads` already use independent `Should ...` subtests with `t.Parallel()`, but the top-level test functions do not call `t.Parallel()`. This is inconsistent with the file's surrounding shape and can be fixed safely by marking both top-level tests parallel before their subtests are registered.

## Resolution

- Added `t.Parallel()` to both top-level sort tests in `internal/api/core/coverage_helpers_test.go`.
- Verified through targeted Go tests and `make verify`.
