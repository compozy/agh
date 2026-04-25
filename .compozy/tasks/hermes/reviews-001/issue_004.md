---
status: resolved
file: internal/bridgesdk/errors.go
line: 344
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:d710364229bd
review_hash: d710364229bd
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 004: Wrap retry wait failures with stage context.
## Review Comment

This return path drops call-site context. Please wrap the error so downstream logs/diagnostics show that cancellation/failure happened during retry backoff waiting.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".

## Triage

- Decision: `VALID`
- Notes: `RetryDo` returns the raw error from `retrypkg.Wait`, which loses the fact that the failure happened while waiting for retry backoff. Wrap this return with retry-backoff context while preserving the original cancellation/deadline error for `errors.Is`.
