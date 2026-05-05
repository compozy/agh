---
status: resolved
file: internal/retry/retry.go
line: 77
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:694a24fb220c
review_hash: 694a24fb220c
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 024: Wrap retry-loop propagated errors with attempt/stage context.
## Review Comment

These returns currently pass upstream errors through raw, which makes failure provenance harder to trace during operations.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".

## Triage

- Decision: `valid`
- Root cause: `retry.DoValue` returns raw context, operation, and sleep errors from several branches. `errors.Is` still works, but the returned error lacks retry attempt/stage context, which makes production failures harder to diagnose.
- Fix approach: wrap retry-loop exits with stage and attempt information while preserving the original error with `%w`; keep the existing tests asserting `errors.Is` behavior and extend the retry test suite under the required `Should...` subtest pattern.
