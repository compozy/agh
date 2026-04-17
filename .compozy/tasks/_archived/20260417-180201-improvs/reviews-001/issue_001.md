---
status: resolved
file: internal/api/core/sse.go
line: 77
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:a3d2cc1bf3a0
review_hash: a3d2cc1bf3a0
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 001: Wrap write failures with operation context.
## Review Comment

At Line 77 onward, raw `return err` drops which SSE write step failed (`id`, `event`, `data`, terminator), making debugging harder and violating error-wrapping policy.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

Also applies to: 88-96, 98-105

## Triage

- Decision: `VALID`
- Notes:
  `writeSSERaw` currently returns raw `io.WriteString` and `Write` errors, so
  callers lose which SSE step failed (`id`, `event`, `data`, or terminator).
  Plan: wrap each write failure with the specific operation context and add a
  regression test that asserts the error includes the failing write step.
