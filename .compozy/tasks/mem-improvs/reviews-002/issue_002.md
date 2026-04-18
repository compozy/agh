---
status: resolved
file: internal/memory/store_test.go
line: 667
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133125556,nitpick_hash:1962e90d47fc
review_hash: 1962e90d47fc
source_review_id: "4133125556"
source_review_submitted_at: "2026-04-18T01:12:18Z"
---

# Issue 002: Prefer t.Run("Should...") blocks for these new single-case tests.
## Review Comment

These additions are all standalone top-level cases. Wrapping them in named subtests would keep the file aligned with the repo’s test shape and make future expansion easier.

As per coding guidelines, `MUST use t.Run("Should...") pattern for ALL test cases` and `Use table-driven tests with subtests (t.Run) as default in Go tests`.

Also applies to: 791-847, 849-880, 882-928, 956-963

---

## Triage

- Decision: `valid`
- Root cause: the cited tests are single-case top-level functions without the repo-mandated `t.Run("Should...")` case wrapper.
- Impact: inconsistent test structure versus project policy and harder incremental expansion for nearby cases.
- Fix plan: wrap each cited standalone case in a named `t.Run("Should ...")` block and move `t.Parallel()` into the case body without changing assertions.
- Resolution: wrapped each cited standalone store test in a `t.Run("Should ...")` case wrapper and kept the execution/assertion bodies unchanged inside the subtests.
- Verification: `go test ./internal/memory`; `make verify`
