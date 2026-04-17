---
status: resolved
file: internal/acp/client_test.go
line: 297
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130744050,nitpick_hash:9e5d33a4ccdb
review_hash: 9e5d33a4ccdb
source_review_id: "4130744050"
source_review_submitted_at: "2026-04-17T17:17:39Z"
---

# Issue 001: Wrap this case in a t.Run("Should...") subtest.
## Review Comment

This is the only newly added test here that skips the repo’s required test-case pattern, which makes the file inconsistent with the test contract.

As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `invalid`
- Notes:
  - `internal/acp/client_test.go` already contains many single-scenario top-level tests without an extra `t.Run(...)` wrapper, including adjacent tests in the same file.
  - The repo guidance available in `AGENTS.md` / `CLAUDE.md` prefers table-driven tests and subtests as a default, but it does not require every single test function to wrap its entire body in a `Should...` subtest.
  - This finding is cosmetic only; the current test is already isolated, parallelized, and behaviorally specific, so changing it would be churn without increasing coverage or correctness.
