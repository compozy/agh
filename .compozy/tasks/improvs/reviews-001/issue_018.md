---
status: resolved
file: internal/tools/tool.go
line: 54
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:2d708460e02d
review_hash: 2d708460e02d
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 018: Wrap returned validation errors with call-site context.
## Review Comment

Line 54 currently returns the error directly; please wrap it so callers can identify the failing operation.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

## Triage

- Decision: `VALID`
- Notes:
  `ToolSource.MarshalText` returns the raw `Validate` error, which loses the
  `MarshalText` call-site context. Plan: wrap the validation failure with a
  marshal-specific message and extend the invalid-source test to assert the
  context.
