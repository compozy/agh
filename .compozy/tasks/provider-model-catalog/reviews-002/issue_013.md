---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/extension/capability_models_test.go
line: 86
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:93a33deb8346
review_hash: 93a33deb8346
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 013: Split per-method denials into subtests for clearer failures.
## Review Comment

Line 86-Line 95 currently checks three methods in one loop. Converting each iteration into a `t.Run("Should ...")` gives isolated failures and aligns with your test-case convention.

As per coding guidelines: "`**/*_test.go`: Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures."

## Triage

- Decision: `valid`
- Notes:
  - `internal/extension/capability_models_test.go` still checks three denied methods inside one loop within a single subtest.
  - That obscures which method failed and falls short of the required per-case `t.Run("Should ...")` structure.
  - Fix plan: keep the same assertions but split the denied-method loop into dedicated `Should ...` subtests.
  - Fixed in `internal/extension/capability_models_test.go` and verified with focused package tests plus `make verify`.
