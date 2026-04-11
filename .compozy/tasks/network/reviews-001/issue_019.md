---
status: resolved
file: internal/network/helpers_test.go
line: 201
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:adda2b8cf9f7
review_hash: adda2b8cf9f7
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 019: Use t.Run subtests for the trace transition matrix.
## Review Comment

The table-driven test for `canApplyTrace` should use `t.Run` for each case to improve test isolation and failure reporting. This aligns with testing guidelines.

As per coding guidelines: "Use table-driven tests with subtests (t.Run) as default" and "MUST use t.Run(\"Should...\") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Root cause: the trace transition matrix is table-driven but does not execute each case as a named subtest, reducing failure localization.
- Fix approach: run each matrix entry through `t.Run("Should...")` while keeping the same transition coverage.
