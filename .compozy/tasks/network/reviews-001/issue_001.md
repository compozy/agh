---
status: resolved
file: internal/api/core/errors.go
line: 88
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:ab0202742709
review_hash: ab0202742709
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 001: Consider preserving the full error chain with %w.
## Review Comment

Using `%v` for the inner error loses its chain, so `errors.Is(err, someInnerSentinel)` will fail on the wrapped result. If callers need to unwrap both sentinels, use `errors.Join` or a multi-error pattern.

## Triage

- Decision: `valid`
- Root cause: `core.NewMemoryValidationError` and `core.NewNetworkValidationError` use `%v` for the nested error, so callers can match the shared sentinel but cannot unwrap the original validation cause.
- Fix approach: preserve both sentinels in the returned error by multi-wrapping the nested error instead of string-formatting it.
