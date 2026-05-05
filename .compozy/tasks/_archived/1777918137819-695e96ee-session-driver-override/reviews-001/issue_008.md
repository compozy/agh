---
status: resolved
file: internal/observe/reconcile_test.go
line: 117
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:3119286ecb3f
review_hash: 3119286ecb3f
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 008: Use table-driven subtests here, and assert the error via a helper instead of strings.Contains.
## Review Comment

The success and failure repair cases exercise the same setup matrix, so a shared `t.Run` table would cut the duplication. In the failure branch, `strings.Contains(err.Error(), ...)` is also looser than the repo's required error assertions and can pass on unrelated wrapped messages.

As per coding guidelines, `**/*_test.go`: Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests and `**/*_test.go`: MUST have specific error assertions (ErrorContains, ErrorAs).

## Triage

- Decision: `valid`
- Root cause: the reconcile tests duplicate similar repair setup across separate top-level tests, and the current failure-path assertion relies on loose string matching.
- Fix plan: restructure the legacy-provider repair coverage into `t.Run("Should ...")` subtests aligned with the new reconcile behavior and assert outcomes directly through indexed/registry state instead of substring-only error checks.
