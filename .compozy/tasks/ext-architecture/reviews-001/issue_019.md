---
status: resolved
file: internal/extension/describe.go
line: 71
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:663ee6f30b23
review_hash: 663ee6f30b23
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 019: Add explicit parentheses to clarify operator precedence.
## Review Comment

The boolean expression relies on `&&` binding tighter than `||`, but the intent isn't immediately obvious to readers. Explicit grouping improves maintainability.

## Triage

- Decision: `valid`
- Notes:
  The current health expression is correct, but it relies on `&&` precedence over `||` in a way that slows down review. Adding explicit grouping improves readability without changing behavior.
  Fix approach: parenthesize the resource-extension branch so the intended grouping is explicit.
