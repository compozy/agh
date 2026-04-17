---
status: resolved
file: internal/bundles/service.go
line: 296
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:dc344a4b23a9
review_hash: dc344a4b23a9
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 004: Wrap new error returns in ListActivations with operation context.
## Review Comment

The new branch returns raw errors, which weakens debuggability at call sites.

As per coding guidelines, `**/*.go`: "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

## Triage

- Decision: `VALID`
- Notes:
  `ListActivations` returns raw errors from bundle lookup and activation
  inventory loading, which loses the failing operation when callers surface the
  error. Plan: wrap each failing branch with `ListActivations`-specific context
  and cover the new error paths in unit tests.
