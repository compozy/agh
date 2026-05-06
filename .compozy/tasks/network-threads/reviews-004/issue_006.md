---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/api/httpapi/network_test.go
line: 18
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:96e9f744245d
review_hash: 96e9f744245d
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 006: Mark the subtest parallel as well.
## Review Comment

The parent test is parallelized, but this new `t.Run("Should ...")` block still runs serially. Adding `t.Parallel()` here keeps the new coverage aligned with the rest of the suite's default test behavior.

As per coding guidelines, "Use `t.Run("Should ...")` subtests with `t.Parallel` as default (opt-out with `t.Setenv`)."

## Triage

- Decision: `valid`
- Notes: The new `t.Run("Should create deterministic direct room", ...)` case in `internal/api/httpapi/network_test.go` is independent and does not use `t.Setenv`, shared globals, or cross-test mutable state. It should call `t.Parallel()` to match the repo's test-shape rules.
