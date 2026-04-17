---
status: resolved
file: internal/workspace/resolver_test.go
line: 761
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:9500eebe7ab4
review_hash: 9500eebe7ab4
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 023: Refactor new cancellation rollback cases into t.Run("Should...") subtests.
## Review Comment

These new cases are clear, but they should follow the repo’s mandatory subtest naming pattern and table-driven default for test cases.

As per coding guidelines, `MUST use t.Run("Should...") pattern for ALL test cases` and `Use table-driven tests with subtests (t.Run) as default in Go tests`.

Also applies to: 801-812

## Triage

- Decision: `valid`
- Root cause: the newly added cancellation rollback coverage was added as separate top-level tests, while the repo convention for new Go cases is table-driven coverage with `t.Run("Should...")` subtests.
- Why this is a bug: it creates style drift inside the new coverage and makes the two nearly identical cases harder to extend than a shared table-driven test.
- Fix approach: collapse the cancellation rollback scenarios into a single table-driven test with `t.Run("Should ...")` subtests, keeping the existing rollback assertions in the shared helper.
- Resolution: the cancellation rollback coverage now lives in one table-driven test with `t.Run("Should ...")` subtests, and the shared helper also asserts that rollback delete contexts are both detached and deadline-bounded.
- Verification: `go test ./internal/workspace` and `make verify` both passed after the refactor.
